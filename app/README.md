# acctpass app

Native acctpass interface for macOS, Windows, Linux, iOS, and Android, built with Tauri 2, React, TypeScript, and Rust.

The app remains offline and compatible with the Go CLI:

- creates or imports the same encrypted `vault.json`
- regenerates passwords without saving them
- supports counters, lengths, and standard or custom symbol sets
- clears the clipboard after 45 seconds if it still contains the generated password
- exports an encrypted vault copy for another device

## Development

```sh
npm install
npm run tauri dev
```

Rust compatibility and vault tests:

```sh
cd src-tauri
cargo test
```

Desktop release build:

```sh
npm run tauri build
```

Mobile project setup requires the normal Tauri Android/iOS prerequisites:

```sh
npm run tauri android init
npm run tauri ios init
```

The shared compatibility fixture lives at `../compatibility/vectors.json` and is exercised by both the Go and Rust test suites.
