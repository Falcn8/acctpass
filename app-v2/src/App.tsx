import { FormEvent, ReactNode, useEffect, useRef, useState } from "react";
import { invoke } from "@tauri-apps/api/core";
import { open, save } from "@tauri-apps/plugin-dialog";
import { readText, writeText } from "@tauri-apps/plugin-clipboard-manager";
import {
  Check,
  Copy,
  Download,
  Eye,
  EyeOff,
  KeyRound,
  Moon,
  RefreshCcw,
  Sun,
  Upload,
  X,
} from "lucide-react";
import "./App.css";

type Theme = "light" | "dark";
type VaultStatus = { exists: boolean; createdAt: string | null; location: string };

const supportedSymbols = "!@#$%^&*()-_=+[]{}?";
const isNative = "__TAURI_INTERNALS__" in window;
const previewParams = new URLSearchParams(window.location.search);

function App() {
  const [theme, setTheme] = useState<Theme>(() => {
    const saved = localStorage.getItem("acctpass-theme");
    if (saved === "light" || saved === "dark") return saved;
    return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
  });
  const [status, setStatus] = useState<VaultStatus | null>(null);
  const [loadError, setLoadError] = useState("");

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
    localStorage.setItem("acctpass-theme", theme);
  }, [theme]);

  useEffect(() => {
    if (!isNative) {
      const ready = previewParams.get("preview") === "ready";
      setStatus({
        exists: ready,
        createdAt: ready ? "2026-07-15T00:00:00Z" : null,
        location: "~/Library/Application Support/acctpass/vault.json",
      });
      return;
    }
    invoke<VaultStatus>("vault_status")
      .then(setStatus)
      .catch((error) => setLoadError(toMessage(error)));
  }, []);

  const toggleTheme = () => setTheme((current) => (current === "light" ? "dark" : "light"));

  if (loadError) {
    return (
      <Shell theme={theme} toggleTheme={toggleTheme}>
        <main className="center-state">
          <p className="state-kicker">Vault unavailable</p>
          <h1>The local vault did not open.</h1>
          <p className="state-message" role="alert">{loadError}</p>
          <button className="button secondary" onClick={() => window.location.reload()}>
            <RefreshCcw size={17} aria-hidden="true" /> Try again
          </button>
        </main>
      </Shell>
    );
  }

  if (!status) {
    return (
      <Shell theme={theme} toggleTheme={toggleTheme}>
        <main className="center-state" aria-label="Opening local vault">
          <span className="spinner" />
          <p>Opening vault…</p>
        </main>
      </Shell>
    );
  }

  return status.exists ? (
    <ReadyWorkspace theme={theme} toggleTheme={toggleTheme} onStatusChange={setStatus} />
  ) : (
    <Shell theme={theme} toggleTheme={toggleTheme}>
      <Setup onCreated={setStatus} />
    </Shell>
  );
}

function Shell({
  children,
  theme,
  toggleTheme,
  actions,
}: {
  children: ReactNode;
  theme: Theme;
  toggleTheme: () => void;
  actions?: ReactNode;
}) {
  return (
    <div className="app-shell">
      <header className="app-header">
        <div className="brand" aria-label="acctpass">
          <span className="brand-mark" aria-hidden="true"><KeyRound size={17} /></span>
          <span>acctpass</span>
        </div>
        <div className="header-actions">
          {actions}
          <button
            type="button"
            className="icon-button"
            onClick={toggleTheme}
            aria-label={`Use ${theme === "light" ? "dark" : "light"} mode`}
          >
            {theme === "light" ? <Moon size={18} aria-hidden="true" /> : <Sun size={18} aria-hidden="true" />}
          </button>
        </div>
      </header>
      <div className="app-body">{children}</div>
    </div>
  );
}

function ReadyWorkspace({
  theme,
  toggleTheme,
  onStatusChange,
}: {
  theme: Theme;
  toggleTheme: () => void;
  onStatusChange: (status: VaultStatus) => void;
}) {
  const importDialogRef = useRef<HTMLDialogElement>(null);
  const [exportError, setExportError] = useState("");

  const exportVault = async () => {
    setExportError("");
    if (!isNative) {
      setExportError("Export is available in the desktop app.");
      return;
    }
    try {
      const path = await save({
        defaultPath: "acctpass-vault.json",
        filters: [{ name: "acctpass vault", extensions: ["json"] }],
      });
      if (path) await invoke("export_vault", { request: { path } });
    } catch (error) {
      setExportError(toMessage(error));
    }
  };

  return (
    <Shell
      theme={theme}
      toggleTheme={toggleTheme}
      actions={
        <>
          <button type="button" className="header-action" onClick={exportVault} aria-label="Export vault">
            <Download size={16} aria-hidden="true" /><span>Export</span>
          </button>
          <button type="button" className="header-action" onClick={() => importDialogRef.current?.showModal()} aria-label="Import vault">
            <Upload size={16} aria-hidden="true" /><span>Import</span>
          </button>
        </>
      }
    >
      <Generator externalError={exportError} />
      <ImportVaultDialog ref={importDialogRef} replacement onImported={onStatusChange} />
    </Shell>
  );
}

function Generator({ externalError }: { externalError: string }) {
  const [service, setService] = useState("");
  const [identity, setIdentity] = useState("");
  const [counter, setCounter] = useState(1);
  const [length, setLength] = useState(20);
  const [useSymbols, setUseSymbols] = useState(true);
  const [customSymbols, setCustomSymbols] = useState(supportedSymbols);
  const [master, setMaster] = useState("");
  const [result, setResult] = useState(() =>
    previewParams.get("result") === "ready" ? "xQ7!vN2#pL9@cR4$mK8%" : "",
  );
  const [revealed, setRevealed] = useState(false);
  const [copied, setCopied] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const feedback = error || externalError;

  const generate = async (event: FormEvent) => {
    event.preventDefault();
    setError("");
    setCopied(false);

    if (!service.trim()) return setError("Enter a service name.");
    if (!identity.trim()) return setError("Enter an email or username.");
    if (counter < 1) return setError("Counter must be at least 1.");
    if (length < 1 || length > 256) return setError("Length must be between 1 and 256.");
    if (useSymbols && !customSymbols) return setError("Choose at least one symbol.");
    if (!master) return setError("Enter your master password.");
    if (!isNative) return setError("Generation is available in the desktop app.");

    setBusy(true);
    try {
      const password = await invoke<string>("generate_password", {
        request: {
          masterPassword: master,
          platform: service,
          email: identity,
          counter,
          length,
          symbols: useSymbols,
          allowedSymbols: useSymbols ? customSymbols : "",
        },
      });
      setResult(password);
      setRevealed(false);
      setMaster("");
      await copyPassword(password);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 2500);
      clearClipboardIfUnchanged(password);
    } catch (generationError) {
      setError(toMessage(generationError));
    } finally {
      setBusy(false);
    }
  };

  const copyResult = async () => {
    if (!result) return;
    try {
      await copyPassword(result);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 2500);
      clearClipboardIfUnchanged(result);
    } catch (copyError) {
      setError(toMessage(copyError));
    }
  };

  return (
    <main className="workbench">
      <form className="generator-panel" onSubmit={generate}>
        <div className="generator-intro">
          <h1>acctpass</h1>
          <p className={feedback ? "is-error" : ""} role={feedback ? "alert" : undefined} aria-live="polite">
            {feedback || "Generate account passwords locally."}
          </p>
        </div>

        <div className="form-grid">
          <Field label="Service" className="service-field">
            <input
              value={service}
              onChange={(event) => setService(event.target.value)}
              placeholder="github.com"
              autoComplete="off"
              spellCheck={false}
              aria-required="true"
            />
          </Field>

          <Field label="Username or email" className="identity-field">
            <input
              value={identity}
              onChange={(event) => setIdentity(event.target.value)}
              placeholder="you@example.com"
              autoComplete="off"
              spellCheck={false}
              aria-required="true"
            />
          </Field>

          <Field label="Version" className="counter-field">
            <input
              type="number"
              min="1"
              max="4294967295"
              value={counter}
              onChange={(event) => setCounter(Number(event.target.value))}
              inputMode="numeric"
              aria-required="true"
            />
          </Field>

          <Field label="Length" className="length-field">
            <input
              type="number"
              min="1"
              max="256"
              value={length}
              onChange={(event) => setLength(Number(event.target.value))}
              inputMode="numeric"
              aria-required="true"
            />
          </Field>

          <label className="symbols-toggle">
            <input
              type="checkbox"
              checked={useSymbols}
              onChange={(event) => setUseSymbols(event.target.checked)}
            />
            <span>Symbols</span>
          </label>

          <Field
            label="Allowed symbols"
            className="symbols-input-field"
          >
            <input
              value={customSymbols}
              onChange={(event) => setCustomSymbols(filterSymbols(event.target.value))}
              disabled={!useSymbols}
              autoComplete="off"
              spellCheck={false}
            />
          </Field>

          <Field label="Master password" className="master-field">
            <input
              type="password"
              value={master}
              onChange={(event) => setMaster(event.target.value)}
              autoComplete="current-password"
              aria-required="true"
            />
          </Field>

          <button className="button primary generate-button" type="submit" disabled={busy} aria-busy={busy}>
            {busy ? <><span className="spinner small" /> Generating…</> : <><KeyRound size={17} aria-hidden="true" /> Generate &amp; copy</>}
          </button>
        </div>

        <div className={`result-row ${result ? "has-result" : ""}`} aria-live="polite">
          <div className="result-value">
            <span>Password</span>
            <output>{result ? (revealed ? result : "•".repeat(Math.min(result.length, 24))) : "—"}</output>
          </div>
          <div className="result-actions">
            <button
              type="button"
              className="result-button"
              onClick={() => setRevealed((value) => !value)}
              disabled={!result}
              aria-label={revealed ? "Hide password" : "Reveal password"}
            >
              {revealed ? <EyeOff size={17} aria-hidden="true" /> : <Eye size={17} aria-hidden="true" />}
            </button>
            <button type="button" className="result-button copy-button" onClick={copyResult} disabled={!result}>
              {copied ? <Check size={17} aria-hidden="true" /> : <Copy size={17} aria-hidden="true" />}
              <span>{copied ? "Copied" : "Copy"}</span>
            </button>
            <button
              type="button"
              className="result-button"
              onClick={() => { setResult(""); setRevealed(false); setCopied(false); }}
              disabled={!result}
              aria-label="Clear password"
            >
              <X size={17} aria-hidden="true" />
            </button>
          </div>
        </div>
      </form>
    </main>
  );
}

function Setup({ onCreated }: { onCreated: (status: VaultStatus) => void }) {
  const importDialogRef = useRef<HTMLDialogElement>(null);
  const [master, setMaster] = useState("");
  const [confirmation, setConfirmation] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const create = async (event: FormEvent) => {
    event.preventDefault();
    setError("");
    if (!master) return setError("Enter a master password.");
    if (master !== confirmation) return setError("Master passwords do not match.");
    if (!isNative) return setError("Vault setup is available in the desktop app.");

    setBusy(true);
    try {
      const created = await invoke<VaultStatus>("create_vault", { request: { masterPassword: master } });
      setMaster("");
      setConfirmation("");
      onCreated(created);
    } catch (creationError) {
      setError(toMessage(creationError));
    } finally {
      setBusy(false);
    }
  };

  return (
    <main className="workbench setup-workbench">
      <form className="setup-panel" onSubmit={create}>
        <div className="panel-heading">
          <h1>Set up your vault.</h1>
        </div>
        <Field label="Master password">
          <input
            type="password"
            value={master}
            onChange={(event) => setMaster(event.target.value)}
            autoComplete="new-password"
            aria-required="true"
            autoFocus
          />
        </Field>
        <Field label="Confirm master password">
          <input
            type="password"
            value={confirmation}
            onChange={(event) => setConfirmation(event.target.value)}
            autoComplete="new-password"
            aria-required="true"
          />
        </Field>
        {error ? <p className="feedback" role="alert" aria-live="polite">{error}</p> : null}
        <button className="button primary" type="submit" disabled={busy} aria-busy={busy}>
          {busy ? <><span className="spinner small" /> Creating…</> : <><KeyRound size={17} aria-hidden="true" /> Create vault</>}
        </button>
        <button className="button secondary" type="button" onClick={() => importDialogRef.current?.showModal()}>
          <Upload size={17} aria-hidden="true" /> Import existing
        </button>
      </form>
      <ImportVaultDialog ref={importDialogRef} onImported={onCreated} />
    </main>
  );
}

function Field({ label, className = "", children }: { label: string; className?: string; children: ReactNode }) {
  return (
    <label className={`field ${className}`}>
      <span>{label}</span>
      {children}
    </label>
  );
}

function ImportVaultDialog({
  ref,
  replacement = false,
  onImported,
}: {
  ref: React.RefObject<HTMLDialogElement | null>;
  replacement?: boolean;
  onImported: (status: VaultStatus) => void;
}) {
  const [path, setPath] = useState("");
  const [master, setMaster] = useState("");
  const [confirmation, setConfirmation] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const close = () => {
    ref.current?.close();
    setPath("");
    setMaster("");
    setConfirmation("");
    setError("");
  };

  const chooseFile = async () => {
    setError("");
    if (!isNative) return setError("Import is available in the desktop app.");
    const selected = await open({
      multiple: false,
      directory: false,
      filters: [{ name: "acctpass vault", extensions: ["json"] }],
    });
    if (typeof selected === "string") setPath(selected);
  };

  const importVault = async (event: FormEvent) => {
    event.preventDefault();
    setError("");
    if (!path) return setError("Choose a vault file.");
    if (!master) return setError("Enter the vault master password.");
    if (replacement && confirmation !== "REPLACE") return setError("Type REPLACE to confirm.");

    setBusy(true);
    try {
      const imported = await invoke<VaultStatus>("import_vault", { request: { path, masterPassword: master } });
      onImported(imported);
      close();
    } catch (importError) {
      setError(toMessage(importError));
    } finally {
      setBusy(false);
    }
  };

  return (
    <dialog
      ref={ref}
      className="import-dialog"
      onClick={(event) => { if (event.target === event.currentTarget) close(); }}
    >
      <form onSubmit={importVault}>
        <div className="dialog-heading">
          <div>
            <h2>{replacement ? "Replace vault" : "Import vault"}</h2>
          </div>
          <button type="button" className="icon-button" onClick={close} aria-label="Close import dialog">
            <X size={18} aria-hidden="true" />
          </button>
        </div>

        <button type="button" className="file-picker" onClick={chooseFile}>
          <Upload size={17} aria-hidden="true" />
          <span>{path ? fileName(path) : "Choose vault file"}</span>
        </button>

        <Field label="Master password">
          <input
            type="password"
            value={master}
            onChange={(event) => setMaster(event.target.value)}
            autoComplete="current-password"
            aria-required="true"
          />
        </Field>

        {replacement ? (
          <Field label="Type REPLACE to confirm">
            <input
              value={confirmation}
              onChange={(event) => setConfirmation(event.target.value)}
              autoComplete="off"
              aria-required="true"
            />
          </Field>
        ) : null}

        {error ? <p className="feedback" role="alert" aria-live="polite">{error}</p> : null}
        <div className="dialog-actions">
          <button type="button" className="button secondary" onClick={close}>Cancel</button>
          <button type="submit" className="button primary" disabled={busy} aria-busy={busy}>
            {busy ? <><span className="spinner small" /> Importing…</> : "Import"}
          </button>
        </div>
      </form>
    </dialog>
  );
}

function filterSymbols(value: string) {
  return supportedSymbols.split("").filter((symbol) => value.includes(symbol)).join("");
}

async function copyPassword(password: string) {
  if (isNative) await writeText(password);
  else await navigator.clipboard.writeText(password);
}

function clearClipboardIfUnchanged(password: string) {
  window.setTimeout(async () => {
    try {
      const current = isNative ? await readText() : await navigator.clipboard.readText();
      if (current === password) {
        if (isNative) await writeText("");
        else await navigator.clipboard.writeText("");
      }
    } catch {
      // Clipboard access can disappear when the app loses focus.
    }
  }, 30_000);
}

function fileName(path: string) {
  return path.split(/[\\/]/).pop() || "vault.json";
}

function toMessage(error: unknown) {
  return error instanceof Error ? error.message : String(error);
}

export default App;
