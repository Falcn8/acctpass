# acctpass

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

## Demo

```sh
acctpass gen --platform google --email you@example.com --print
acctpass gen --platform google --email you@example.com --print --no-symbols
acctpass gen --platform google --email you@example.com --print --no-symbols --counter 2
acctpass gen --platform google --email you@example.com --print --no-symbols --counter 2 --length 12
```

By default, generated passwords are copied to the clipboard. Use `--print` to
print one instead.

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
