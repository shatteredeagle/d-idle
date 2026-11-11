# Desktop app on Electron: d2idle-watch

> status: PR1 built (TASK-289, 2026-11-10): app skeleton, security posture tests, packaging, CI job. Tray, autostart, hide-to-tray and the update check are PR2 (TASK-290). Design and reasons: `docs/research/desktop-electron.md` (TASK-265). The Go launcher `apps/desktop` (docs/desktop-app.md) stays as the fallback.

Tags: V = run or read by the agent (how), U = unverified.

User decision 2026-11-10: build the desktop app on Electron (portable) with the tray icon from the start. Defaults taken for the open questions of the research doc (the user was not asked individually; change them here if wrong):
- Keep `apps/desktop` (Go) as a fallback for now. Windows and macOS builds are unsigned for the beta: Windows SmartScreen and macOS Gatekeeper will warn; the friend guide must say how to continue (More info, Run anyway; right-click Open). A ~100 MB download is accepted.
- `enableCookieEncryption` fuse stays off (profile not tied to the OS keyring).
- Update check is CHECK-ONLY against our own signed feed, never auto-install, no github.com contact (PR2).
- `config.json` may carry an optional `window` {width,height} key besides `address`; unknown keys are kept.

## Layout

`apps/desktop-electron/`: `src/` (app.js wiring, security.js pure rules, address.js, config.js, args.js, probe.js, state.js, menu.js, helper/ with the only preload), `test/` (node:test, smoke/), `build/` (icon.svg, generated icon.png/icon.ico/tray.png, fuses, afterPack), `electron-builder.yml`, `scripts/` (make-icons.js, scratch.sh). A stub `go.mod` makes `go list ./...` of the main module ignore the folder (V: research doc experiment). Root `go.mod` untouched.

## Run, test, build

- Local: `cd apps/desktop-electron && npm ci --ignore-scripts && node node_modules/electron/install.js && npx electron . --address host:8891` (needs Node 22.12+; V on the worker host with Node 26 and Electron 45.2.3).
- Flags (same as the Go launcher): `--setup`, `--address host:port`, `--page watch|inventory|policy|return`, `--print` (prints the URL, opens nothing). Unknown flags, including Chromium switches, exit 2.
- Unit tests: `npm test` (V: 20 tests, no display). Smoke (real Electron under Xvfb against fake local servers): `npm run smoke` (V on the worker host).
- Packages: `npm run dist:linux` (AppImage + deb, V built on the worker host), `npm run dist:win` (NSIS + portable; U: runs on the Windows CI runner only, never run). Fuses read back from the packaged binary (V: `npx @electron/fuses read`): RunAsNode, NodeCliInspect, NODE_OPTIONS off, OnlyLoadAppFromAsar and embedded asar integrity on.
- On the worker host use `sh apps/desktop-electron/scripts/scratch.sh <cmd>` so `node_modules` survives labrun's re-sync.
- Icons: original artwork `build/icon.svg` (an eye; no game art). Regenerate with `xvfb-run -a -s '-screen 0 1280x1024x24' npx electron scripts/make-icons.js` (V).

## Security posture (V by tests unless noted)

Two views: helper (local `d2idle-app://helper/`, the only preload, 4 invoke channels and one push) and game (the exact configured origin, partition `persist:d2idle`, NO preload). Both `contextIsolation`, `sandbox`, no node, no webview, devtools off. Navigation of the game view is locked to the exact origin (`will-navigate`, `will-redirect`, `will-frame-navigate`); `window.open` is denied everywhere; permissions are default-deny; the helper has a CSP (meta and header) and its files are served by path-normalised custom protocol; every IPC handler checks the sender frame URL and web contents. Smoke V: the game page sees no `d2idle`, `require`, `process`; `window.open`, a script `location.href` and a 302 to another origin never reach the other server. Residual risk (U to improve): the page is http on a LAN, see the research doc section 3.

## Findings

- V (smoke): Electron 45 `ses.fetch(..., {redirect: 'manual'})` REJECTS on a 3xx ("Redirect was cancelled") instead of returning an opaque redirect; the probe treats that as up (a login redirect counts as up).
- U: the real login through the app, the Linux sandbox on Ubuntu 24.04 (AppArmor userns), Windows install and portable run, SmartScreen text, macOS. Checklist for a desktop machine: start with the server stopped (expect the "not reachable" helper), start the server (expect the watch page), log in, quit and reopen (expect no login), `--page inventory` with the app running (expect the same window navigates), `--setup`.

## Not done in PR1

Tray, autostart, hide-to-tray, update check (PR2); macOS build; code signing; THIRD_PARTY.md entry for the Electron bundle (proposed line in the PR report); `docs/OWNERS` line `apps/desktop-electron/` (coordinator).
