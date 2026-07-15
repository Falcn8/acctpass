# acctpass

[![CI](https://github.com/Falcn8/acctpass/actions/workflows/ci.yml/badge.svg)](https://github.com/Falcn8/acctpass/actions/workflows/ci.yml)
![Go version](https://img.shields.io/github/go-mod/go-version/Falcn8/acctpass?logo=go)
![License](https://img.shields.io/github/license/Falcn8/acctpass)
![Security: unaudited](https://img.shields.io/badge/security-unaudited-orange)

A tiny offline CLI that generates strong account-specific passwords without storing the passwords themselves.

- No server
- No account
- No cloud sync
- No stored generated passwords
- Cross-platform: macOS, Linux, Windows
- Uses Argon2id + XChaCha20-Poly1305 + HMAC-SHA256

```text
master password
      +
encrypted local seed
      +
platform + email + counter
      ↓
same strong password every time
```

![acctpass terminal demo](assets/demo.gif)

`acctpass` stores one encrypted local vault seed, then regenerates account
passwords from your master password plus the platform, email, counter, length,
and symbol setting you provide. The generated account passwords are not saved.

## Install

```sh
go install github.com/Falcn8/acctpass@latest
```

Or build locally:

```sh
git clone https://github.com/Falcn8/acctpass.git
cd acctpass
make build
```

## Usage

```sh
acctpass init
acctpass gen --platform github --email alice@example.com
acctpass gen --platform github --email alice@example.com --print
acctpass gen --platform github --email alice@example.com --counter 2
acctpass gen --platform legacy --email alice@example.com --allowed-symbols '!@#'
acctpass gen --platform forum --email alice@example.com --exclude-symbols '[]{}'
acctpass info
```

By default, generated passwords can use `!@#$%^&*()-_=+[]{}?`. Use
`--allowed-symbols` to restrict generation to a non-empty subset of those
symbols, or `--exclude-symbols` to prohibit a subset while still requiring a
symbol. The two flags are mutually exclusive and cannot be combined with
`--no-symbols`. Quote symbol values so your shell does not interpret them.

Symbol restrictions are part of the deterministic generation inputs. Use the
same restriction whenever you regenerate a password; changing it produces a
different password.

## How is this different from a password manager?

`acctpass` is not a full password manager like Bitwarden, 1Password, or KeePass.
It does not store a vault of generated account passwords, autofill forms, sync
across devices, share credentials with a team, or help recover lost data.

Instead, it is a small deterministic password generator. Given the same master
password, encrypted local seed, platform, email, counter, length, and symbol
settings, it regenerates the same password again.

Use a password manager if you want browser/mobile apps, sync, sharing, secure
notes, passkeys, recovery workflows, or audited production-grade credential
storage. Use `acctpass` if you want a tiny offline CLI with no account, no
server, and no stored generated passwords.

## Builds and releases

CI runs tests, `go vet`, and builds on Ubuntu, macOS, and Windows. It also runs
`govulncheck` on Go 1.26.x.

Pushing a `v*` tag builds GitHub Release archives for:

- macOS arm64 and amd64
- Linux arm64 and amd64
- Windows amd64

Each release includes SHA-256 checksums so downloads can be verified.
Release archives include the README, SECURITY policy, and MIT license.

## Security disclaimer

`acctpass` is designed so that the source code and algorithm can be public.
Security depends on the secrecy of your master password and encrypted vault
seed, not on hiding the code.

The program does not store generated account passwords. It stores only an
encrypted random seed in the local vault file.

New vaults use Argon2id with 64 MiB memory, time cost 3, and 1 thread for
predictable cross-platform resource use. Existing vaults keep using the KDF
parameters stored in their vault file.

Do not commit your real vault file, master password, generated passwords, or
account list to GitHub.

This project has not been professionally audited. Use at your own risk. For
critical accounts, consider using an audited password manager.

## Limitations

- If someone gets both your vault and master password, they can regenerate your
  passwords.
- If you lose the vault, you cannot regenerate existing passwords.
- Malware on your computer can steal passwords when generated.
- Clipboard contents may be visible to other local apps.
- This project has not been professionally audited.

## Development

```sh
make test
make vet
make vuln
make build-all
```

## License

MIT. See [LICENSE](LICENSE).
