# acctpass app

The release interface for the native acctpass app. It shares the Rust/Tauri
backend in `../app/src-tauri` and keeps password generation fully offline.

```sh
npm ci
npm --prefix ../app ci
npm run tauri:dev
```

Build a desktop installer with `npm run tauri:build`. Generated Android and iOS
projects live under `../app/src-tauri/gen`; signed store builds require the
corresponding developer account credentials.

Browser preview with a simulated ready vault:

```sh
npm run dev -- --host 127.0.0.1 --port 1421
```

Open `http://127.0.0.1:1421/?preview=ready`.
