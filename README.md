# acctpass

`acctpass` is an offline deterministic account password generator.

It stores one encrypted local vault seed, then regenerates account passwords
from your master password plus the platform, email, counter, length, and symbol
setting you provide.

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
acctpass info
```

By default, generated passwords are copied to the clipboard. Use `--print` to
print one instead.

## Security disclaimer

`acctpass` is designed so that the source code and algorithm can be public.
Security depends on the secrecy of your master password and encrypted vault
seed, not on hiding the code.

The program does not store generated account passwords. It stores only an
encrypted random seed in the local vault file.

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

CI runs tests, `go vet`, builds on Linux/macOS/Windows, and scans with
`govulncheck` on Go 1.26.x.

Pushing a `v*` tag builds GitHub Release archives with SHA-256 checksums.
