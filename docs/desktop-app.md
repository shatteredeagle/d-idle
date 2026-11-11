# Desktop app: d2idle-watch

> status: the Go launcher is the FALLBACK; the main desktop app is now `apps/desktop-electron` (docs/desktop-electron.md, user decision 2026-11-10).

A small launcher so nobody has to remember the watch URL. Code: `apps/desktop` (a package of the main module, standard library only; a nested go.mod made the merge gate's `go test -race ./apps/desktop` fail). Task: TASK-227. Workflow (draft, never run): `.github/workflows/desktop-app.yml`.

## Decision: Go launcher around a Chromium-family `--app` window

Options weighed (build size, cgo, cross-compile, runtime dependency, CI):

| Option | Size | Build pain | Verdict |
|---|---|---|---|
| Go + webview binding (webview_go) | about 10 MB plus runtime | cgo on both OS. V: the binding source pins `webkit2gtk-4.0`; S: the worker host has webkit2gtk 4.1 (pkg-config answered for 4.1; 4.0 not checked), so it would need shims. Windows needs MinGW and the WebView2 loader. | No |
| Same, with a native menu and cookie jar | | S: from its source it sets no persistent cookie store on WebKitGTK, so the login would be lost on every start unless we add more cgo | No |
| Tauri (Rust) | 5 to 10 MB | Rust toolchain, WebKitGTK 4.1 dev packages on the Linux runner, a second language in the repo | No |
| Go launcher + `chrome --app=URL` | V: 7 MB static binary, `CGO_ENABLED=0` | V: `GOOS=windows go build` works from Linux | Chosen |

Reasons: no cgo, trivial cross-compile, a CI that is one `go build` per OS, and the browser keeps its own persistent profile, so the session cookie survives restarts without any code of ours. Windows 10/11 ships Edge (Chromium), so the app window is guaranteed there. On Linux any of Chrome, Chromium, Edge or Brave is used.

Cost, stated plainly:
- A Chromium-family browser is required for the app window. Without one the launcher opens the default browser in a normal tab (everything else still works). U: the worker host has Firefox/Waterfox only, so there it takes the tab fallback until Chromium is installed.
- No native menu bar inside the window (app mode has none). Substitutes: `--page inventory|policy|return|watch`, `--setup` (change address), the Linux desktop entry (`d2idle-watch --install`) with right-click actions Inventory, Policy, Return screen, Change address, and on Windows a shortcut with `--page ...` arguments. The watch pages link to each other. If an in-window menu matters later, the upgrade path is a webview module in `apps/desktop` (own go.mod) with the same config, address and probe code.
- The window title is the page title; the icon on Linux comes from the desktop entry (`StartupWMClass=d2idle-watch`, SVG installed by `--install`). The Windows exe has no embedded icon yet.

## Behaviour

- Config: `<user config dir>/d2idle-watch/config.json` holds `{"address": "http://host:port"}` only (0600). Nothing is compiled in, no default; no passphrase is ever stored. The login passphrase is typed on the watch page; the session cookie lives in the browser profile `<config dir>/d2idle-watch/profile`.
- First run, or `--setup`: the app window shows a local page ("Where is the game?"), validates and stores the address. `--address host:port` sets it from the command line.
- Start: probe `GET <address>/watch` (no redirects, no cookies). Up (any answer below 500, a login redirect counts): open `<address>/watch` (or the `--page` page) in the app window directly and exit. Otherwise open the local helper page, which shows "not reachable" or "warming up" (5xx), says the container takes one to three minutes, retries every 3 s, has Retry now and Change address, and navigates the window to the game when it answers. The helper exits once the window stops polling (3 minutes idle).
- The helper listens on 127.0.0.1 on a random port behind a random path token and checks the Host and Origin headers (DNS rebinding, other pages on the machine).

## Same-origin rule (login)

The watch login POST is accepted only when the request's Origin/Host matches `D2BOT_WATCH_ORIGIN` (`internal/access/sameorigin.go`, `docs/watch-origin.md`). So:
- The game page is always loaded by top-level navigation to the exact configured origin (scheme, host, port as typed; the host is only lowercased). It is never proxied, framed or rewritten by the app; a loopback proxy would present the wrong origin and break login.
- Enter the address exactly as `D2BOT_WATCH_ORIGIN` is set on the server. Address typed with the IP while the server expects the host name fails login with 403 "forbidden origin"; change it with Change address.
- The address is reduced to an origin: path, query and fragment are dropped; user names and passphrases in the address are refused.

## Build and test

- Test: `cd apps/desktop && go vet ./... && go test -count=1 ./...`.
- Build: `cd apps/desktop && CGO_ENABLED=0 go build -ldflags "-s -w" -o d2idle-watch .`; Windows: `GOOS=windows CGO_ENABLED=0 go build -ldflags "-s -w -H=windowsgui" -o d2idle-watch.exe .` (`windowsgui` hides the console window).
- Smoke (fake browser, Xvfb, no window on any desktop): `cd apps/desktop && sh testdata/smoke.sh`.
- Windows shortcut (manual): target `d2idle-watch.exe`, add jump arguments `--page inventory` in a copy of the shortcut.

## Status of the claims

- V: address normalisation, config round trip, probe states (up, login redirect, 5xx warming, closed), browser lookup order, helper page guards and idle exit: unit tests (`go test`). V: first run, warming then up, second run opens the real page directly with the profile flags, under Xvfb on the worker host with a fake browser (`testdata/smoke.sh`).
- U: a real Chromium/Edge `--app` window (no such browser on the worker host), that the app window keeps navigating from the local helper page to the game origin, the actual login with persisted cookie, the Windows build running. Checklist for a machine with Edge or Chromium: run `d2idle-watch --address <host:port>` with the server stopped (expect the warming screen), start the server (expect the watch page in the app window without a tab bar), log in, close and reopen (expect no login), try `--page inventory`, `--setup`.
- Not done: CI never run; no embedded Windows icon; no macOS.
