# Desktop app on Electron: d2idle-watch

> status: built, not yet run on real desktops (TASK-289 skeleton, TASK-290 tray, autostart, hide-to-tray, update check; 2026-11-10). Design and reasons: `docs/research/desktop-electron.md` (TASK-265). The Go launcher `apps/desktop` (docs/desktop-app.md) stays as the fallback.

Tags: V = run or read by the agent (how), U = unverified.

User decision 2026-11-10: build the desktop app on Electron (portable) with the tray icon from the start. Defaults taken for the open questions of the research doc (the user was not asked individually; change them here if wrong):
- Keep `apps/desktop` (Go) as a fallback for now. Windows and macOS builds are unsigned for the beta: Windows SmartScreen and macOS Gatekeeper will warn; the friend guide must say how to continue (More info, Run anyway; right-click Open). A ~100 MB download is accepted.
- `enableCookieEncryption` fuse stays off (profile not tied to the OS keyring).
- Update check is CHECK-ONLY against our own signed feed, never auto-install, no github.com contact (see Tray and update check).
- `config.json` may carry an optional `window` {width,height} key besides `address`; unknown keys are kept.

## Layout

`apps/desktop-electron/`: `src/` (app.js wiring, security.js pure rules, address.js, config.js, args.js, probe.js, state.js, menu.js, helper/ with the only preload), `test/` (node:test, smoke/), `build/` (icon.svg, generated icon.png/icon.ico/tray.png, fuses, afterPack), `electron-builder.yml`, `scripts/` (make-icons.js, scratch.sh). A stub `go.mod` makes `go list ./...` of the main module ignore the folder (V: research doc experiment). Root `go.mod` untouched.

## Run, test, build

- Local: `cd apps/desktop-electron && npm ci --ignore-scripts && node node_modules/electron/install.js && npx electron . --address host:8891` (needs Node 22.12+; V on the worker host with Node 26 and Electron 45.2.3).
- Flags (same as the Go launcher plus `--hidden` and `--debug`): `--setup`, `--address host:port`, `--page watch|inventory|policy|return`, `--print` (prints the URL, opens nothing), `--hidden` (start in the tray). Unknown flags exit 2; a short list of Chromium switches is passed through (macOS section).
- Unit tests: `npm test` (V: 32 tests, no display). Smoke (real Electron under Xvfb against fake local servers): `npm run smoke` (V on the worker host).
- Packages: `npm run dist:linux` (AppImage + deb, V built on the worker host), `npm run dist:win` (NSIS + portable; U: runs on the Windows CI runner only, never run). Fuses read back from the packaged binary (V: `npx @electron/fuses read`): RunAsNode, NodeCliInspect, NODE_OPTIONS off, OnlyLoadAppFromAsar and embedded asar integrity on.
- On the worker host use `sh apps/desktop-electron/scripts/scratch.sh <cmd>` so `node_modules` survives labrun's re-sync.
- Icons: original artwork `build/icon.svg` (an eye; no game art). Regenerate with `xvfb-run -a -s '-screen 0 1280x1024x24' npx electron scripts/make-icons.js` (V).

## Security posture (V by tests unless noted)

Two views: helper (local `d2idle-app://helper/`, the only preload, 4 invoke channels and one push) and game (the exact configured origin, partition `persist:d2idle`, NO preload). Both `contextIsolation`, `sandbox`, no node, no webview, devtools off. Navigation of the game view is locked to the exact origin (`will-navigate`, `will-redirect`, `will-frame-navigate`); `window.open` is denied everywhere; permissions are default-deny; the helper has a CSP (meta and header) and its files are served by path-normalised custom protocol; every IPC handler checks the sender frame URL and web contents. Smoke V: the game page sees no `d2idle`, `require`, `process`; `window.open`, a script `location.href` and a 302 to another origin never reach the other server. Residual risk (U to improve): the page is http on a LAN, see the research doc section 3.

## Tray and update check (TASK-290)

- Tray: icon `build/tray.png` (from icon.svg), menu = Open, Pages (Watch, Inventory, Policy, Return screen), a disabled status line, Check for updates / Update available, Start with the system, "Closing the window hides it to the tray", Quit. The status line is reachability only (Online, Warming up, Offline plus the address): the watch pages need the login and the watch server has no token-less endpoint with the hero name or level (V: grep of `internal/watch` routes; `/healthz` is the d2d control server's), so nothing more is shown. Left click opens the window. The status is polled every 3 s while the helper shows and every 15 s while the game page is up.
- Hide to tray: closing the window hides it while a tray exists (`hideToTray` in config.json, default true, toggled from the tray menu); Quit really quits; a second start focuses the window. U: on Linux there is no way to know whether a tray host exists (GNOME needs the AppIndicator extension; KDE works): with no host a hidden window is reachable only by starting the app again, which restores it. Turn the toggle off there.
- Single instance: second start with `--page`/`--setup` navigates the running window (V by code path, not smoked).
- Autostart: Linux writes `~/.config/autostart/d2idle-watch.desktop` (Exec = the AppImage file when run from one, else the binary, plus `--hidden`); Windows and macOS use `setLoginItemSettings` with `--hidden` (Electron has no login item on Linux). `--hidden` starts in the tray when a tray exists.
- Notifications on alerts: skipped (needs a server event contract; research doc section 8).
- Update check (check-only): at start (15 s later) and daily the main process GETs `<feed>/repos/<repo>/releases?per_page=10`, takes the highest non-draft tag `desktop-v<x.y.z>`, fetches `MANIFEST.json` and `MANIFEST.json.sig` from the SAME host, verifies the Ed25519 signature over the exact manifest bytes against the two pinned keys (the bot image's `deploy/d2d-container/pubkeys/*.pub`, a test keeps them equal), checks schema 1 and that tag and manifest version agree, and compares with the app version. A newer version shows `Update available: x.y.z` in the tray; clicking shows the notes and says to get the new build where this one came from. Nothing is downloaded or installed, no redirect is followed, no github.com contact (V: unit tests with a fake feed: wrong key, tampered manifest, other host, mismatched version, http feed, userinfo; smoke with a signed fake feed).
- The feed address is a deployment value, not in the repo: `D2IDLE_UPDATE_FEED` at package time (`scripts/write-feed.js` writes `build/feed.json`; CI reads the repository variable of the same name) or `"updateFeed"` in config.json. Without either the check is off. The intended value is the friends' signed feed (docs/friend-feed.md); publishing desktop releases there needs a `desktop-v<x.y.z>` entry in the feed list with the two manifest assets (U: `cmd/d2feed` currently writes `app-v` releases; a desktop release needs the publisher to accept the other prefix, not done). `"updateCheck": false` in config.json turns the check off.
- Config keys now: `address`, optional `window` {width,height}, `hideToTray`, `updateCheck`, `updateFeed`; unknown keys are kept.
- V (smoke under Xvfb, no tray host there): `Tray` constructs, the menu template has the expected items and states, the update flow works against a signed fake feed, the autostart toggle and hide-to-tray behave. U: an actual tray icon on KDE or Windows.

## macOS (TASK-291)

Personal build for a MacBook, unsigned, LAN only, never published. UNIVERSAL (arm64 + x64), so the CPU of the Mac does not matter. Tags: V = run (how), U = unverified.

Build: `npm run dist:mac` on a Mac (Node only, no Xcode; V on the Mac mini via the worker lab tool), or the `package-mac` job of `.github/workflows/desktop-electron.yml` (`macos-latest`, `--publish never`, no signing env, artifact `d2idle-watch-macos-universal`; U: never run on GitHub). Output: `dist/d2idle-watch-<version>-mac-universal.dmg` and `.zip`. Config: `mac:` in `electron-builder.yml` (`identity: "-"` = ad-hoc signature only, because Apple Silicon kills a binary with no signature (V: the first build with `identity: null` died with Killed: 9 on the Mac mini); `hardenedRuntime: false`, `notarize: false`, icon `build/icon.icns`).

Icons come from `build/icon.svg` (the eye, no game art) through `scripts/make-icons.js`: `icon.icns` (128 to 1024 px, PNG entries) and the menu bar template image `build/trayTemplate.png` (22 px) plus its double-size sibling (44 px), black on transparent from `build/tray-template.svg`; the `Template` suffix makes macOS recolour it for light, dark and active menu bars (U: not seen on a real menu bar).

Behaviour on macOS (code in `src/platform.js`, `src/menu.js`; V by unit tests, U on a real Mac):
- Closing the window hides it (even when "Closing the window hides it to the tray" is off, the window must survive) and hides the Dock icon; opening it from the menu bar icon or clicking the Dock icon brings both back. `window-all-closed` does not quit while the tray exists. Cmd+Q (app menu Quit) really quits.
- Menu bar: standard app menu (About, Hide, Quit), Pages (Watch, Inventory, Policy, Return screen, Change address, Reload with Cmd+R), Edit (copy and paste work) and Window roles.
- Start with the system: `setLoginItemSettings({openAtLogin, args: ['--hidden']})`, so it starts in the menu bar. On a first start macOS may ask to allow the login item in System Settings.

### Install on the MacBook (steps for the user)

1. Download the dmg from the LAN link in the build report (it works only on the home network). Check the sha256 with `shasum -a 256 <file>` against the report.
2. Open the dmg and drag `d2idle-watch` onto Applications.
3. First launch: Gatekeeper blocks unsigned apps. Either right-click the app in Applications, choose Open, then Open again; or run `xattr -dr com.apple.quarantine /Applications/d2idle-watch.app` in Terminal and start it normally.
4. Local Network: on the first start macOS asks whether d.idle watch may find and connect to devices on your local network (the app declares `NSLocalNetworkUsageDescription`). Click Allow. Until you do, the page cannot load: the window shows the helper page with "Waiting for the server ... allow Local Network access", and it retries by itself with a growing delay (3 s up to 30 s), so no relaunch is needed after you allow it. If you clicked Don't Allow: System Settings > Privacy & Security > Local Network > switch d2idle-watch on.
5. Set the address: the app asks on first start; type `<watch-host>:8891` (the LAN host name of the watch server, given in the build report). Or start with the flag: `/Applications/d2idle-watch.app/Contents/MacOS/d2idle-watch --address <watch-host>:8891`. To change it later use Pages > Change address, or `--setup`.
6. It only works on the home network: there is no remote-access plan yet, so away from home the helper shows "not reachable".

V on the Mac mini (Apple M4, headless over ssh): the universal app has x86_64 and arm64 slices, `codesign --verify --deep --strict` passes (ad-hoc), `hdiutil verify` of the dmg passes, the packaged app starts, stays up 12 s and probes a fake local server (`GET /watch`), `npm test` passes (macOS runs the menu test with the Linux layout). Troubleshooting: run `/Applications/d2idle-watch.app/Contents/MacOS/d2idle-watch --enable-logging=stderr` in Terminal (Chromium switches `--enable-logging`, `--log-file`, `--log-level`, `--v`, `--vmodule`, `--disable-gpu`, `--disable-software-rasterizer`, `--remote-debugging-port` are passed through; other unknown flags still exit 2). `--debug` writes the verbose log (page loads, console warnings and errors, window sizes); load failures, stalls and renderer crashes are always logged. Log file: macOS `~/Library/Logs/d2idle-watch/main.log`, Linux `~/.local/state/d2idle-watch/main.log`, Windows `%APPDATA%\d2idle-watch\logs\main.log`; the tray item "Open log folder" opens it. A failed, stalled (no answer for 20 s) or empty page load shows the helper with the reason and retries; it never leaves a blank window (V: smoke on the worker host with a server that drops the first loads, one that never answers, and pixel checks that neither view is one uniform colour).

Not verified: the x64 slice running (no Intel Mac or Rosetta check), the menu bar icon rendering, the Gatekeeper dialogs, the login item prompt, behaviour on the MacBook itself (the Mac mini smoke is headless, see the build report).

## Findings

- V (smoke): Electron 45 `ses.fetch(..., {redirect: 'manual'})` REJECTS on a 3xx ("Redirect was cancelled") instead of returning an opaque redirect; the probe treats that as up (a login redirect counts as up).
- U: the real login through the app, the Linux sandbox on Ubuntu 24.04 (AppArmor userns), Windows install and portable run, SmartScreen text, macOS. Checklist for a desktop machine: start with the server stopped (expect the "not reachable" helper), start the server (expect the watch page), log in, quit and reopen (expect no login), `--page inventory` with the app running (expect the same window navigates), `--setup`.

## Not done

macOS signing and notarisation; real-desktop checks of the tray, autostart on Windows and the installers; desktop releases in the feed publisher; code signing; THIRD_PARTY.md entry for the Electron bundle (proposed line in the PR report); `docs/OWNERS` line `apps/desktop-electron/` (coordinator).
