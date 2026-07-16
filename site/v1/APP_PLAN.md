# acctpass Desktop Application Plan

## Product

Build **acctpass Desktop** as an open-source graphical client around the existing Go derivation core. It should make the CLI usable without a terminal while preserving the defining rule: generated account passwords are never written to the vault.

The desktop app may store encrypted recipe metadata—service, account, counter, length, and symbol preference—because exact recipes are necessary for reliable regeneration. It must not store generated passwords or per-account secret seeds.

## Architecture

- **UI:** Fyne v2, pinned to a stable release. It provides one Go codebase for macOS, Windows, and Linux without sending the master password through a JavaScript runtime.
- **Core:** extract the existing vault and generator code into a reusable internal Go package shared by the CLI and desktop app.
- **Repository:** retain one MIT-licensed monorepo so a single compatibility suite protects both interfaces.
- **Distribution:** build on native GitHub Actions runners, then sign and notarize platform packages.

```text
cmd/acctpass/              existing CLI
cmd/acctpass-desktop/      desktop entry point
internal/core/             vault and deterministic derivation
internal/recipes/          encrypted non-password metadata
internal/platform/         secure file and clipboard integration
internal/app/              shared use cases
```

## Compatibility contract

1. Existing `vault.json` files import without changing their seed.
2. Current v1 normalization and derivation remain supported permanently.
3. Golden vectors prove identical outputs across CLI, desktop, macOS, Windows, and Linux.
4. A future context encoding becomes an explicit derivation version; it never silently changes an existing password.
5. The GUI rejects ambiguous delimiter characters for v1 recipes and explains why.

## Version-one experience

- Create a new encrypted seed or import an existing CLI vault.
- Warn about weak master passwords, matching current CLI behavior.
- Search encrypted recipes by service or account.
- Generate and copy without revealing the password by default.
- Clear the clipboard after 15, 30, or 60 seconds only if it still contains the value acctpass placed there.
- Lock after inactivity or immediately through a shortcut.
- Rotate one account by deliberately incrementing its counter.
- Re-encrypt the same seed when changing the master password.
- Back up and restore the encrypted vault with schema and checksum validation.

## Security gates

- Preserve Argon2id, XChaCha20-Poly1305, and HMAC-SHA256 compatibility.
- Keep decrypted seed material only in memory while unlocked and perform best-effort zeroing without claiming guaranteed memory erasure.
- Never log master passwords, seeds, generated passwords, clipboard contents, or full recipe data.
- Apply `0700`/`0600` permissions on Unix and a current-user DACL on Windows.
- Harden vault writes against symlink and replacement races.
- Fuzz vault parsing, KDF parameter bounds, and derivation inputs.
- Publish a threat model before beta.
- Make no automatic network requests in v1.
- Do not market the project as audited or “military-grade.” Complete an independent review before recommending it for critical accounts.

## Releases

| Platform | Initial artifact | Release gate |
| --- | --- | --- |
| macOS Intel + Apple silicon | Universal signed `.app` in notarized `.dmg` | CI, UI tests, signing, notarization |
| Windows 10/11 x64 | Signed installer and portable build | CI, UI tests, Authenticode signing |
| Linux x64 + arm64 | AppImage and `.deb` | CI on X11 and Wayland, checksums |

Every release includes SHA-256 checksums, an SBOM, build provenance, release notes, security policy, and MIT license.

## Roadmap

### Phase 0 — Core extraction (1–2 weeks)

- Extract the crypto, vault, path, and generator logic from `package main`.
- Keep CLI output byte-for-byte compatible.
- Add golden vectors, migration fixtures, an encrypted recipe schema, and threat model.

### Phase 1 — Desktop MVP (3–5 weeks)

- Onboarding, vault import, unlock, recipe CRUD, generate/copy, rotation, clipboard clearing, lock, backup, and restore.
- Keyboard navigation, screen-reader labels, high contrast, and reduced motion.
- Native builds on all three platform families.

### Phase 2 — Hardening (2–3 weeks)

- Fuzzing, filesystem hardening, crash-safe migrations, UI automation, signed installers, notarization, and release provenance.
- Closed beta with test vaults and resolution of security-review findings.

### Phase 3 — Public v1.0

- Publish installers and the compatibility specification.
- Keep the CLI first-class.
- Collect issues without telemetry; diagnostics remain explicit and redacted.

Biometric unlock, browser integration, and encrypted sync remain later work because each changes the threat model materially.

## Definition of done

- Existing CLI vaults generate identical passwords in every supported desktop build.
- No network request occurs during onboarding, unlock, recipe management, or generation.
- Generated passwords never appear in vaults, logs, preferences, crash reports, or temporary files.
- Interrupted writes leave either the old valid vault or the new valid vault.
- Install, backup, generation, rotation, lock, and restore flows work by keyboard.
- Signed artifacts pass checksum and provenance verification.
