import { FormEvent, useEffect, useMemo, useRef, useState } from "react";
import { invoke } from "@tauri-apps/api/core";
import { open, save } from "@tauri-apps/plugin-dialog";
import { readText, writeText } from "@tauri-apps/plugin-clipboard-manager";
import {
  ArrowRight, Check, ChevronDown, Clipboard, Copy, Download, Eye, EyeOff,
  FileKey, KeyRound, LockKeyhole, Minus, Moon, Plus, RefreshCcw,
  ShieldCheck, Sun, Upload, X,
} from "lucide-react";
import "./App.css";

type Theme = "light" | "dark";
type View = "generate" | "vault";
type OnboardingMode = "create" | "import";
type SymbolMode = "standard" | "none" | "custom";
type VaultStatus = { exists: boolean; createdAt: string | null; location: string };

const supportedSymbols = "!@#$%^&*()-_=+[]{}?";
const isNative = "__TAURI_INTERNALS__" in window;

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
      const ready = new URLSearchParams(window.location.search).get("preview") === "ready";
      setStatus({ exists: ready, createdAt: ready ? "2026-07-15T00:00:00Z" : null, location: "~/Library/Application Support/acctpass/vault.json" });
      return;
    }
    invoke<VaultStatus>("vault_status").then(setStatus).catch((error) => setLoadError(toMessage(error)));
  }, []);

  const toggleTheme = () => setTheme((current) => current === "light" ? "dark" : "light");
  if (loadError) {
    return <Shell theme={theme} toggleTheme={toggleTheme}><div className="fatal-state"><span className="eyebrow">Vault unavailable</span><h1>acctpass could not open its local vault.</h1><p>{loadError}</p><button className="button secondary" onClick={() => window.location.reload()}><RefreshCcw size={17} /> Try again</button></div></Shell>;
  }
  if (!status) {
    return <Shell theme={theme} toggleTheme={toggleTheme}><div className="loading-state" aria-label="Opening local vault"><div className="loading-mark" /><p>Opening your local vault…</p></div></Shell>;
  }
  return <Shell theme={theme} toggleTheme={toggleTheme}>{status.exists ? <Workbench status={status} onStatusChange={setStatus} /> : <Onboarding onCreated={setStatus} />}</Shell>;
}

function Shell({ children, theme, toggleTheme }: { children: React.ReactNode; theme: Theme; toggleTheme: () => void }) {
  return <div className="app-shell">
    <header className="topbar">
      <a className="brand" href="/" aria-label="acctpass home"><BrandMark /><span>acctpass</span></a>
      <div className="topbar-actions"><div className="offline-status"><span /> Offline by design</div><button className="icon-button" onClick={toggleTheme} aria-label={`Use ${theme === "light" ? "dark" : "light"} mode`}>{theme === "light" ? <Moon size={18} /> : <Sun size={18} />}</button></div>
    </header>{children}
  </div>;
}

function BrandMark() {
  return <span className="brand-mark" aria-hidden="true"><span className="brand-dot" /><span className="brand-keyhole" /></span>;
}

function Onboarding({ onCreated }: { onCreated: (status: VaultStatus) => void }) {
  const [mode, setMode] = useState<OnboardingMode>("create");
  return <main className="onboarding">
    <section className="onboarding-intro">
      <span className="edition">Private utility · v0.1</span>
      <h1>Your passwords,<br /><em>without a password list.</em></h1>
      <p className="intro-copy">acctpass regenerates account-specific passwords from one encrypted local seed. Nothing is uploaded. Nothing is synced behind your back.</p>
      <div className="trust-line"><span><ShieldCheck size={17} /> Standard cryptography</span><span><LockKeyhole size={17} /> Local vault</span><span><RefreshCcw size={17} /> Same inputs, same password</span></div>
    </section>
    <section className="setup-panel" aria-labelledby="setup-title">
      <div className="setup-tabs" role="tablist" aria-label="Vault setup method"><button className={mode === "create" ? "active" : ""} onClick={() => setMode("create")} role="tab" aria-selected={mode === "create"}>New vault</button><button className={mode === "import" ? "active" : ""} onClick={() => setMode("import")} role="tab" aria-selected={mode === "import"}>Import vault</button></div>
      {mode === "create" ? <CreateVaultForm onCreated={onCreated} /> : <ImportVaultForm onImported={onCreated} />}
    </section>
  </main>;
}

function CreateVaultForm({ onCreated }: { onCreated: (status: VaultStatus) => void }) {
  const [master, setMaster] = useState("");
  const [confirm, setConfirm] = useState("");
  const [visible, setVisible] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const warnings = masterWarnings(master);
  const submit = async (event: FormEvent) => {
    event.preventDefault(); setError("");
    if (!master) return setError("Enter a master password.");
    if (master !== confirm) return setError("The two master passwords do not match.");
    if (!isNative) return setError("Vault creation is available in the native app.");
    setBusy(true);
    try { const created = await invoke<VaultStatus>("create_vault", { request: { masterPassword: master } }); setMaster(""); setConfirm(""); onCreated(created); }
    catch (caught) { setError(toMessage(caught)); } finally { setBusy(false); }
  };
  return <form onSubmit={submit} className="setup-form">
    <div className="section-heading"><span className="step-number">01</span><div><h2 id="setup-title">Create your local vault</h2><p>This password unlocks the encrypted seed. It is never saved.</p></div></div>
    <PasswordField label="Master password" value={master} onChange={setMaster} visible={visible} onToggle={() => setVisible(!visible)} autoFocus />
    <PasswordField label="Confirm master password" value={confirm} onChange={setConfirm} visible={visible} onToggle={() => setVisible(!visible)} />
    {warnings.length > 0 && master.length > 0 && <div className="inline-note warning" role="note"><strong>A stronger master password is safer.</strong><span>{warnings.join(" ")} You can still continue for testing.</span></div>}
    {error && <InlineError message={error} />}
    <button className="button primary full" disabled={busy}>{busy ? "Encrypting local seed…" : "Create vault"} <ArrowRight size={18} /></button>
    <p className="form-footnote">Losing this password or your vault means losing access to regenerated passwords.</p>
  </form>;
}

function ImportVaultForm({ onImported }: { onImported: (status: VaultStatus) => void }) {
  const [path, setPath] = useState(""); const [master, setMaster] = useState(""); const [visible, setVisible] = useState(false); const [busy, setBusy] = useState(false); const [error, setError] = useState("");
  const chooseFile = async () => {
    setError(""); if (!isNative) return setError("File import is available in the native app.");
    const selected = await open({ multiple: false, filters: [{ name: "acctpass vault", extensions: ["json"] }] }); if (typeof selected === "string") setPath(selected);
  };
  const submit = async (event: FormEvent) => {
    event.preventDefault(); if (!path) return setError("Choose an acctpass vault file."); if (!master) return setError("Enter the vault's master password."); setBusy(true); setError("");
    try { const imported = await invoke<VaultStatus>("import_vault", { request: { path, masterPassword: master } }); setMaster(""); onImported(imported); }
    catch (caught) { setError(toMessage(caught)); } finally { setBusy(false); }
  };
  return <form onSubmit={submit} className="setup-form">
    <div className="section-heading"><span className="step-number">01</span><div><h2 id="setup-title">Bring this device into the loop</h2><p>Import the encrypted vault used by your CLI or another device.</p></div></div>
    <button type="button" className={`file-picker ${path ? "selected" : ""}`} onClick={chooseFile}>{path ? <FileKey size={22} /> : <Upload size={22} />}<span><strong>{path ? fileName(path) : "Choose vault.json"}</strong><small>{path ? "Ready to verify" : "Encrypted JSON only"}</small></span>{path && <Check size={18} />}</button>
    <PasswordField label="Master password for this vault" value={master} onChange={setMaster} visible={visible} onToggle={() => setVisible(!visible)} />
    {error && <InlineError message={error} />}
    <button className="button primary full" disabled={busy}>{busy ? "Verifying vault…" : "Verify and import"} <ArrowRight size={18} /></button>
    <p className="form-footnote">The vault is verified before it replaces local app data.</p>
  </form>;
}

function Workbench({ status, onStatusChange }: { status: VaultStatus; onStatusChange: (status: VaultStatus) => void }) {
  const [view, setView] = useState<View>("generate");
  return <main className="workbench"><aside className="side-rail"><div><span className="eyebrow">Local workspace</span><h1>{view === "generate" ? "Make one." : "Move safely."}</h1><p>{view === "generate" ? "Each field becomes part of the password recipe. Match them later to regenerate it." : "Your encrypted seed is the one file every device must share."}</p></div><nav aria-label="App sections"><button className={view === "generate" ? "active" : ""} onClick={() => setView("generate")}><KeyRound size={18} /> Generate <ArrowRight size={16} /></button><button className={view === "vault" ? "active" : ""} onClick={() => setView("vault")}><FileKey size={18} /> Vault transfer <ArrowRight size={16} /></button></nav><div className="rail-fact"><span className="fact-index">01</span><p>Generated passwords are never written to the vault.</p></div></aside><section className="work-area">{view === "generate" ? <Generator /> : <VaultTransfer status={status} onStatusChange={onStatusChange} />}</section></main>;
}

function Generator() {
  const [platform, setPlatform] = useState(""); const [email, setEmail] = useState(""); const [counter, setCounter] = useState(1); const [length, setLength] = useState(24); const [symbolMode, setSymbolMode] = useState<SymbolMode>("standard"); const [customSymbols, setCustomSymbols] = useState(supportedSymbols); const [advanced, setAdvanced] = useState(false); const [master, setMaster] = useState(""); const [masterVisible, setMasterVisible] = useState(false); const [password, setPassword] = useState(""); const [passwordVisible, setPasswordVisible] = useState(false); const [busy, setBusy] = useState(false); const [error, setError] = useState(""); const [clipboardSeconds, setClipboardSeconds] = useState(0); const clipboardTimer = useRef<number | null>(null);
  const warnings = useMemo(() => { const messages: string[] = []; if (length < 12) messages.push("Short passwords are easier to guess."); if (symbolMode === "none") messages.push("Removing symbols reduces the available character set."); return messages; }, [length, symbolMode]);
  useEffect(() => () => { if (clipboardTimer.current) window.clearInterval(clipboardTimer.current); }, []);
  const copyWithExpiry = async (value: string) => {
    if (!value) return; if (isNative) await writeText(value); else await navigator.clipboard.writeText(value); if (clipboardTimer.current) window.clearInterval(clipboardTimer.current); setClipboardSeconds(45);
    clipboardTimer.current = window.setInterval(() => setClipboardSeconds((current) => { if (current > 1) return current - 1; if (clipboardTimer.current) window.clearInterval(clipboardTimer.current); clipboardTimer.current = null; void clearClipboardIfUnchanged(value); return 0; }), 1000);
  };
  const submit = async (event: FormEvent) => {
    event.preventDefault(); setError(""); if (!platform.trim()) return setError("Enter the service this password is for."); if (!email.trim()) return setError("Enter the account email or username."); if (!master) return setError("Enter your master password."); if (symbolMode === "custom" && !customSymbols) return setError("Choose at least one allowed symbol."); if (!isNative) return setError("Password generation is available in the native app."); setBusy(true);
    try { const generated = await invoke<string>("generate_password", { request: { masterPassword: master, platform, email, counter, length, symbols: symbolMode !== "none", allowedSymbols: symbolMode === "custom" ? customSymbols : "" } }); setPassword(generated); setPasswordVisible(false); await copyWithExpiry(generated); }
    catch (caught) { setError(toMessage(caught)); } finally { setMaster(""); setBusy(false); }
  };
  const toggleSymbol = (symbol: string) => setCustomSymbols((current) => supportedSymbols.split("").filter((item) => item === symbol ? !current.includes(item) : current.includes(item)).join(""));
  return <div className="generator-page">
    <div className="page-heading"><div><span className="eyebrow">Password recipe</span><h2>Generate locally</h2></div><span className="privacy-label"><span /> No network request</span></div>
    <form className="generator-form" onSubmit={submit}>
      <div className="identity-grid"><Field label="Service" hint="Use the same name every time"><input value={platform} onChange={(event) => setPlatform(event.target.value)} placeholder="github" autoCapitalize="none" autoCorrect="off" spellCheck={false} autoFocus /></Field><Field label="Email or username" hint="This separates accounts on one service"><input value={email} onChange={(event) => setEmail(event.target.value)} placeholder="alice@example.com" autoCapitalize="none" autoCorrect="off" spellCheck={false} /></Field></div>
      <div className="counter-row"><div><label htmlFor="counter">Password counter</label><p>Increase only when you rotate this account’s password.</p></div><div className="stepper"><button type="button" onClick={() => setCounter(Math.max(1, counter - 1))} aria-label="Decrease counter"><Minus size={16} /></button><input id="counter" type="number" min="1" max="1000000" value={counter} onChange={(event) => setCounter(Math.max(1, Number(event.target.value) || 1))} /><button type="button" onClick={() => setCounter(counter + 1)} aria-label="Increase counter"><Plus size={16} /></button></div></div>
      <div className={`advanced-section ${advanced ? "open" : ""}`}><button type="button" className="advanced-toggle" onClick={() => setAdvanced(!advanced)} aria-expanded={advanced}><span>Rules <small>{length} characters · {symbolMode === "none" ? "no symbols" : symbolMode === "custom" ? "custom symbols" : "standard symbols"}</small></span><ChevronDown size={18} /></button><div className="advanced-content"><div>
        <Field label="Length" hint={length < 12 ? "Allowed for testing, but weak" : "24 is a strong general default"}><div className="length-control"><input type="range" min="1" max="64" value={length} onChange={(event) => setLength(Number(event.target.value))} aria-label="Password length" /><input type="number" min="1" max="256" value={length} onChange={(event) => setLength(Math.min(256, Math.max(1, Number(event.target.value) || 1)))} aria-label="Exact password length" /></div></Field>
        <fieldset className="symbol-options"><legend>Symbols</legend><div className="segmented-control">{(["standard", "none", "custom"] as SymbolMode[]).map((mode) => <button type="button" key={mode} className={symbolMode === mode ? "active" : ""} onClick={() => setSymbolMode(mode)}>{mode[0].toUpperCase() + mode.slice(1)}</button>)}</div>{symbolMode === "custom" && <div className="symbol-grid" aria-label="Allowed symbols">{supportedSymbols.split("").map((symbol) => { const selected = customSymbols.includes(symbol); return <button type="button" key={symbol} className={selected ? "selected" : ""} aria-pressed={selected} onClick={() => toggleSymbol(symbol)}>{symbol}</button>; })}</div>}</fieldset>
        {warnings.length > 0 && <div className="rule-warning">{warnings.map((warning) => <span key={warning}>{warning}</span>)}</div>}
      </div></div></div>
      <PasswordField label="Master password" value={master} onChange={setMaster} visible={masterVisible} onToggle={() => setMasterVisible(!masterVisible)} />
      {error && <InlineError message={error} />}
      <button className="button primary generate-button" disabled={busy}><span>{busy ? "Deriving…" : "Generate & copy"}</span>{busy ? <span className="button-spinner" /> : <Clipboard size={18} />}</button>
    </form>
    {password && <section className="password-result" aria-live="polite"><div className="result-meta"><span><Check size={16} /> Copied locally</span><button onClick={() => { setPassword(""); setPasswordVisible(false); }} aria-label="Clear generated password"><X size={17} /></button></div><div className="password-line"><code>{passwordVisible ? password : "•".repeat(Math.min(password.length, 28))}</code><button onClick={() => setPasswordVisible(!passwordVisible)} aria-label={passwordVisible ? "Hide password" : "Reveal password"}>{passwordVisible ? <EyeOff size={19} /> : <Eye size={19} />}</button><button onClick={() => void copyWithExpiry(password)} aria-label="Copy password again"><Copy size={19} /></button></div><p>{clipboardSeconds > 0 ? `Clipboard clears in ${clipboardSeconds} seconds if unchanged.` : "Clipboard was cleared if it still contained this password."}</p></section>}
  </div>;
}

function VaultTransfer({ status, onStatusChange }: { status: VaultStatus; onStatusChange: (status: VaultStatus) => void }) {
  const [replacementPath, setReplacementPath] = useState(""); const [master, setMaster] = useState(""); const [visible, setVisible] = useState(false); const [confirmed, setConfirmed] = useState(false); const [message, setMessage] = useState(""); const [error, setError] = useState(""); const [busy, setBusy] = useState(false);
  const exportFile = async () => { setError(""); setMessage(""); if (!isNative) return setError("Vault export is available in the native app."); const path = await save({ defaultPath: "acctpass-vault.json", filters: [{ name: "acctpass vault", extensions: ["json"] }] }); if (!path) return; setBusy(true); try { await invoke("export_vault", { request: { path } }); setMessage(`Encrypted vault exported as ${fileName(path)}.`); } catch (caught) { setError(toMessage(caught)); } finally { setBusy(false); } };
  const chooseReplacement = async () => { setError(""); setMessage(""); if (!isNative) return setError("Vault import is available in the native app."); const selected = await open({ multiple: false, filters: [{ name: "acctpass vault", extensions: ["json"] }] }); if (typeof selected === "string") { setReplacementPath(selected); setConfirmed(false); } };
  const replaceVault = async (event: FormEvent) => { event.preventDefault(); if (!replacementPath) return setError("Choose the replacement vault."); if (!master) return setError("Enter the replacement vault's master password."); if (!confirmed) return setError("Confirm that you understand this replaces the local vault."); setBusy(true); setError(""); setMessage(""); try { const imported = await invoke<VaultStatus>("import_vault", { request: { path: replacementPath, masterPassword: master } }); setMaster(""); setReplacementPath(""); setConfirmed(false); onStatusChange(imported); setMessage("Replacement vault verified and stored locally."); } catch (caught) { setError(toMessage(caught)); } finally { setBusy(false); } };
  return <div className="vault-page"><div className="page-heading"><div><span className="eyebrow">Encrypted seed</span><h2>Vault transfer</h2></div><span className="privacy-label"><span /> Local file</span></div><section className="vault-summary"><div className="vault-seal"><FileKey size={28} /></div><div><span className="eyebrow">Active vault</span><h3>Ready on this device</h3><p>Created {formatDate(status.createdAt)} · Passwords are not stored here.</p></div><button className="button secondary" onClick={exportFile} disabled={busy}><Download size={17} /> Export copy</button></section><div className="transfer-explainer"><span className="explainer-number">Why one vault?</span><p>Every vault contains a different random seed. To regenerate the same passwords on another device, import an encrypted copy of this vault there.</p></div><form className="replace-form" onSubmit={replaceVault}><div className="section-heading"><span className="step-number">02</span><div><h3>Replace this device’s vault</h3><p>Useful when moving to a vault exported from another device.</p></div></div><button type="button" className={`file-picker ${replacementPath ? "selected" : ""}`} onClick={chooseReplacement}><Upload size={21} /><span><strong>{replacementPath ? fileName(replacementPath) : "Choose replacement vault"}</strong><small>It will be verified before saving</small></span>{replacementPath && <Check size={18} />}</button>{replacementPath && <PasswordField label="Master password for replacement vault" value={master} onChange={setMaster} visible={visible} onToggle={() => setVisible(!visible)} />}{replacementPath && <label className="confirmation"><input type="checkbox" checked={confirmed} onChange={(event) => setConfirmed(event.target.checked)} /><span><strong>Replace the local vault</strong><small>I have exported a backup if I still need the current seed.</small></span></label>}{error && <InlineError message={error} />}{message && <div className="inline-note success"><Check size={17} /><span>{message}</span></div>}{replacementPath && <button className="button danger" disabled={busy}>Verify and replace</button>}</form><p className="vault-location" title={status.location}>Stored at <code>{status.location}</code></p></div>;
}

function Field({ label, hint, children }: { label: string; hint?: string; children: React.ReactNode }) { return <label className="field"><span><strong>{label}</strong>{hint && <small>{hint}</small>}</span>{children}</label>; }
function PasswordField({ label, value, onChange, visible, onToggle, autoFocus = false }: { label: string; value: string; onChange: (value: string) => void; visible: boolean; onToggle: () => void; autoFocus?: boolean }) { return <label className="field password-field"><span><strong>{label}</strong><small>Never saved by acctpass</small></span><div><input type={visible ? "text" : "password"} value={value} onChange={(event) => onChange(event.target.value)} autoComplete="off" spellCheck={false} autoFocus={autoFocus} /><button type="button" onClick={onToggle} aria-label={visible ? "Hide password" : "Show password"}>{visible ? <EyeOff size={18} /> : <Eye size={18} />}</button></div></label>; }
function InlineError({ message }: { message: string }) { return <div className="inline-error" role="alert"><X size={16} /><span>{message}</span></div>; }
async function clearClipboardIfUnchanged(expected: string) { try { const current = isNative ? await readText() : await navigator.clipboard.readText(); if (current === expected) { if (isNative) await writeText(""); else await navigator.clipboard.writeText(""); } } catch { /* clipboard permission can expire */ } }
function masterWarnings(value: string) { const warnings: string[] = []; if (value.length < 16) warnings.push("It is shorter than 16 characters."); if (["password", "master", "acctpass", "admin", "letmein", "qwerty"].some((word) => value.toLowerCase().includes(word))) warnings.push("It contains a common password word."); return warnings; }
function formatDate(value: string | null) { if (!value) return "on an unknown date"; const date = new Date(value); return Number.isNaN(date.getTime()) ? value : new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(date); }
function fileName(path: string) { return path.split(/[\\/]/).pop() || path; }
function toMessage(error: unknown) { return error instanceof Error ? error.message : String(error); }

export default App;
