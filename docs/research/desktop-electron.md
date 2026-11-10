# Desktop launcher on Electron: design and decision

> status: research and proposal (TASK-265, 2026-11-10). Nothing here is built. Today's launcher is `apps/desktop` (docs/desktop-app.md). No decision is recorded; proposed ADR text is at the end for the user.

Tags: V = fetched or run by me (source given), S = snippet or secondary, U = unverified or from memory. Target: the Electron shell is a desktop app, so no game-version question; the game server it points at is the d.idle watch server (classic 1.14d bot). Nothing here is D2R.

## 1. Why, and what must carry over

The Go launcher needs a Chromium-family browser for its app window and otherwise opens a browser tab (docs/desktop-app.md, V read). The ask: an always-own window, no Chrome install, Windows, Linux and macOS. Carried over unchanged:

| Requirement | Source today | Electron plan |
|---|---|---|
| Config is `{"address"}`, 0600, no passphrase | apps/desktop/config.go | same JSON, same path (section 4) |
| Opens the exact configured origin | same-origin rule, `internal/access/sameorigin.go` | top-level load of the typed origin, never proxied (section 3) |
| Offline / warming helper page with retry | desktop.go, shell.go | local page, probe done in the main process |
| `--setup`, `--address`, `--page watch\|inventory\|policy\|return` | main.go | same flags (section 4) |
| Linux .desktop entry with actions | `--install` | electron-builder `desktop` entry (section 5) |
| Windows installer plus portable exe | user rule | NSIS plus portable targets (section 5) |
| Tray | optional | optional, phase 3 |

## 2. Electron version

- V: latest stable lines on 2026-11-10 (releases.electronjs.org/releases.json): 45.2.3 (Chromium 156.0.8078.80, Node 24.22.2, 2026-11-04), 44.10.2, 43.8.7, 42.11.18, 41.10.7. npm package `electron` 45.2.3, licence MIT, `engines.node >= 22.12.0`.
- V: support policy is "the latest 3 stable releases"; majors every 8 weeks, alpha 4 weeks, beta 4 weeks (docs/tutorial/electron-timelines.md, raw.githubusercontent.com/electron/electron/main). With 8-week majors a pinned major is unsupported about 24 weeks after it goes stable.
- Recommendation: pin the newest stable major (45.x) in `package.json` with an exact version and the lockfile. Bump the major at least every 3 majors (about 6 months), minor/patch monthly. A Renovate/Dependabot job is allowed to open PRs only (nothing auto-merges). The in-app update check (section 8) tells users when a release carrying a new Electron exists, which is the real security lever for a pinned Chromium.
- Node: needs Node >= 22.12 to install. CI uses Node 24 LTS (U: 24 is LTS in Nov 2026; Electron itself bundles 24.22).

## 3. Security posture

The shell loads a remote page over plain HTTP on a LAN (`http://host:port`, same as today). V: the Electron checklist item 1 says load only HTTPS (docs/tutorial/security.md). We cannot meet that for a LAN bot without TLS, so the controls below shrink what the remote page can reach instead. Say this plainly in the doc and the release notes.

Windows and views (V sources are the same security.md items 2, 3, 4, 13, 14, 17, 19, 20 unless noted):
- Two `WebContentsView`s in one `BaseWindow` (or one `BrowserWindow`, helper swapped in): the HELPER view (local page, has a preload exposing 3 functions) and the GAME view (remote origin, NO preload, nothing exposed). Reason: a preload is per web contents, so reusing one view for both would hand the remote page `contextBridge` functions after navigation.
- Both: `contextIsolation: true`, `nodeIntegration: false`, `sandbox: true`, `webSecurity` on, no `allowRunningInsecureContent`, no `enableBlinkFeatures`, no `webviewTag`, no `@electron/remote` (the remote module is not used at all; it is a separate package since Electron 14, U).
- GAME view: persistent session partition `persist:d2idle` (holds the login cookie). V: the server sets a persistent cookie, `MaxAge` = `sessionAge`, HttpOnly, SameSite=Strict (`internal/access/access.go` Cookie()), so Chromium persists it across restarts in the partition without our code. SameSite=Strict is fine because the app only does top-level loads of that origin.
- Navigation lock: parse the configured address to an origin once. `will-navigate`, `will-redirect` and `setWindowOpenHandler` compare `new URL(url).origin` to it with strict equality; mismatch is cancelled (window open: `{action: "deny"}`). The only exception is the local helper, which is a different view. `shell.openExternal` is never called with page-supplied URLs (item 15); a link to another origin is simply dropped, or after a deliberate user choice opened in the default browser through the main process with an `http:`/`https:` scheme check. Chosen: drop, plus a one-line toast in the helper bar (phase 2). The watch pages link only to each other (docs/desktop-app.md), so nothing legitimate breaks.
- Never frame, proxy or rewrite the game page (the login Origin/Host check, `docs/watch-origin.md`). The shell must not inject headers; loading by `loadURL(origin + path)` sends the real Origin on the login POST exactly like a browser. U: Electron adds an `Electron/45.x` token to the User-Agent; the server does not look at it (not verified in code, worth one grep and a live login test).
- Session permissions: `session.setPermissionRequestHandler` and `setPermissionCheckHandler` default-deny everything (item 5); the watch pages need none (U: confirm they use no notifications or clipboard; if they do, whitelist by name).
- CSP: V: the server already sends CSP on its pages (`internal/api/api.go` has `default-src 'self'; ...; frame-ancestors 'none'`; watch pages set `policyCSP`). The HELPER page is ours and gets a strict CSP via a `<meta>` tag: `default-src 'none'; style-src 'self'; script-src 'self'; connect-src 'none'; img-src 'self'`. It talks to the main process only through the preload (probe, save address); it never fetches the game itself, so `connect-src 'none'` holds.
- Helper page source: load from a registered privileged custom protocol (`d2idle-app://helper/...`, `protocol.handle`) rather than `file://` (item 18), serving only files from the packaged `app/helper/` folder, path-normalised so `..` cannot escape.
- IPC (item 17): `ipcMain.handle` channels only for `probe`, `saveAddress`, `openPage`, `retry`; every handler checks `event.senderFrame.url` starts with `d2idle-app://helper/` and validates arguments (the address goes through the same normaliser as `address.go`: origin only, no userinfo, no path). Remote GAME view has no preload, so it cannot reach any channel.
- Probe: main process does `GET <origin>/watch`, `redirect: manual`, no cookies, 3 s timeout, via `net.fetch` on a throwaway session (same rule as probe.go: below 500 is up, 5xx is warming).
- Fuses (item 19; V `@electron/fuses` 2.1.3, MIT): flip `runAsNode` off, `enableNodeCliInspectArguments` off, `onlyLoadAppFromAsar` on, `enableEmbeddedAsarIntegrityValidation` on (Windows and macOS only, V per fuses.md), `enableCookieEncryption` ON is worth trying but see open question 3 (it ties cookies to an OS keyring/DPAPI key; portable use across machines is not a goal). Done in an electron-builder `afterPack` hook.
- Dev tools: off in packaged builds. `--remote-debugging-port` ignored (the app builds its own argv handling and never forwards `--inspect`).
- No auto-update from untrusted sources: see section 8. Updates never download or execute code in phases 1 and 2.
- No telemetry, no crash reporter upload (do not call `crashReporter.start`), no network except the configured address (plus the opt-in update check).
- Linux sandbox: U (from memory, not verified here): AppImage/unpacked Electron needs the setuid `chrome-sandbox` or unprivileged user namespaces; some distros restrict userns (Ubuntu 24.04 AppArmor policy) and an AppImage then fails with a sandbox error. We never pass `--no-sandbox` by default; the Linux notes carry the workaround (the deb sets up `chrome-sandbox`; for AppImage a documented one-time AppArmor profile or an explicit `--no-sandbox` the user adds knowingly). Verify on the worker host (KDE desktop) and one Ubuntu 24.04 machine before claiming Linux support.

Residual risk, honestly: the page is HTTP on the LAN; an on-path attacker could alter it, and it would run inside a sandboxed, isolated, preload-less renderer that is locked to one origin with no IPC. That is no worse than the same page in a browser tab and better than the Go launcher's browser profile with extensions.

## 4. Behaviour and config parity

- App name `d2idle-watch` so `app.getPath("userData")` is `<user config dir>/d2idle-watch` on all three OSes (U: Electron uses `~/.config`, `%APPDATA%`, `~/Library/Application Support`, the same bases as Go's `os.UserConfigDir`). The new app then reads the existing `config.json` with no migration. The Go launcher kept its Chromium profile in `profile/`; Electron keeps its Chromium data directly in userData, so the first Electron start asks for the passphrase once (the old profile is not read). Config write: `fs.writeFile(path, json, {mode: 0o600})` then `chmod` (Windows ignores the mode, as in Go).
- Flags: `--setup`, `--address host:port`, `--page watch|inventory|policy|return`, `--print` (dry run, prints URL, opens nothing), `--install` becomes electron-builder's job on Linux (section 5). Parse `process.argv` after `app.isPackaged`-aware slicing; an unknown `--page` exits 2 with the same message as the Go launcher. Chromium switches are not accepted from the user's command line other than a short allow-list.
- Single instance: `app.requestSingleInstanceLock()`. A second start with `--page inventory` focuses the running window and navigates it (replaces the current "second Go process" behaviour).
- First run / `--setup`: helper view shows "Where is the game?" and validates the address with the shared normaliser (port of `address.go`, same test vectors from `desktop_test.go`). Start: probe; up opens the page in the GAME view; otherwise the helper shows not reachable / warming up (5xx), polls every 3 s, Retry now, Change address, then swaps to the GAME view when the probe answers.
- Window: min 900x600, remembers size/position in the same config file under an optional `window` key (this adds a key to `{"address"}`: open question 4; default is NOT to, and keep `{"address"}` exactly).
- Menu: a native application menu is free in Electron (Go app mode had none): Watch, Inventory, Policy, Return screen, Change address, Reload, Quit. Menu items call `gameView.loadURL(origin + path)` from the main process. This replaces the Linux right-click actions and Windows shortcut arguments (both still exist as well).
- Tray (optional, phase 3): icon plus the same menu, close-to-tray off by default.

## 5. Packaging

| | electron-builder | Electron Forge |
|---|---|---|
| Version | V: 26.18.0 on npm, MIT, 10 direct deps | V: `@electron-forge/cli` 8.0.1, MIT |
| Targets | V (scheme.json read): NSIS (`oneClick`, `perMachine`, `allowToChangeInstallationDirectory`), `portable`, AppImage, deb (`desktop` entry, `executableArgs`, `mimeTypes`), and more; mac dmg/zip (U) | per-OS makers: Squirrel.Windows (no portable exe, U), deb, rpm, zip, dmg (U) |
| Portable Windows exe | yes, first-class `portable` target (V in schema) | no (U) |
| Config | one `electron-builder.yml` | `forge.config.js`, plugin set |
| Auto-update | `electron-updater` 6.8.11 MIT (V) | Electron's built-in `autoUpdater` (Squirrel) plus update.electronjs.org (V in updates.md) |

Choice: electron-builder. It is the only one that gives the user rule (NSIS installer AND portable exe) from one config, AppImage and deb are built-in, and it does not need extra plugins. Cost: its release cadence and bus factor are smaller than Electron's own Forge (U), its docs site is JS-rendered so options are read from `scheme.json` in the repo, and it adds a transitive dependency tree that must be audited (section 7).

Outputs per OS:
- Windows: `d2idle-watch-setup-<ver>-x64.exe` (NSIS, per-user by default, no admin), `d2idle-watch-<ver>-x64-portable.exe` (self-extracting; start-up slower, U: unpacks to a temp dir each run). Portable stores config in userData as well; a `--config-dir` flag is NOT offered (one way per option, user rule). Unsigned in phase 1: SmartScreen warns; EV/OV signing needs a certificate, user cost and decision (open question 5). Also add the embedded icon (the Go exe has none).
- Linux: AppImage (x64) and deb; desktop entry `d2idle-watch.desktop` with `Actions` Inventory, Policy, Return screen, Change address through electron-builder's `desktop` option (`Exec=... --page inventory`). `--install` is not needed for the deb; for a bare AppImage use AppImageLauncher or the app's one-time "integrate" menu item (U, phase 3).
- macOS: zip/dmg, arm64 plus x64 (or universal). Needs a macOS runner and, to open without a Gatekeeper warning, an Apple Developer ID (99 USD/year, U) and notarisation. Phase 3, after the user decides; unsigned builds run after a right-click open (S).
- Size: V (HEAD on the GitHub release assets): the Electron 45.2.3 zips are 124.1 MB (linux-x64), 158.2 MB (win32-x64), 121.2 MB (darwin-arm64) before our app, so expect installers of roughly 80 to 120 MB and installed size 250 to 350 MB (U). The Go launcher is 7 MB. Compression flag `compression: maximum`; `electronLanguages: [en-US]` strips locale packs (about 40 to 50 MB of the install, U).
- Reproducibility: pin Node, `npm ci`, `package-lock.json` committed, electron-builder with `--publish never`.

## 6. Repo layout and gates

```
apps/desktop-electron/
  go.mod                  placeholder: "module electronplaceholder"  (see below)
  package.json            exact versions, "private": true, no postinstall of our own
  package-lock.json
  electron-builder.yml
  src/main.js, config.js, address.js, probe.js, views.js, menu.js, ipc.js
  src/helper/ (index.html, helper.js, helper.css, preload.js)
  test/*.test.js          node:test (no extra test dep); address vectors shared with Go
  build/ icons/           svg/png/ico source, no files over 1 MB
.github/workflows/desktop-electron.yml
```

Why a stub `go.mod` (V, run locally with Go 1.27.3 in `/tmp`): `go list ./...` DOES descend into `node_modules` and would pick up any `.go` file an npm package ships (a package dir `m/node_modules/x` with a broken `b.go` was listed as package `m/node_modules/x`). A nested `go.mod` in `apps/desktop-electron` made `go list ./...` list only the root module, so `go vet ./...` and `go test ./...` on the worker host ignore the folder entirely. The root `go.mod` and `go.sum` are not touched. The old comment in docs/desktop-app.md (a nested go.mod broke `go test -race ./apps/desktop`) concerns a nested module around Go code that the gate tests by path; here there is no Go code in the folder, so the stub is safe. Alternative without the stub: never run `npm install` on the gate host (the whole-repo gate checks out a clean tree, so `node_modules` is absent); the stub is belt and braces for developers' local trees. U: `tools/ci/check.sh` uses `git ls-files -co --exclude-standard` (V read), so a gitignored `node_modules` and `dist/` are not scanned by the size and art checks; add both to `.gitignore`.

CI (new, separate): `.github/workflows/desktop-electron.yml`, `paths: ["apps/desktop-electron/**", ".github/workflows/desktop-electron.yml"]`, jobs: `lint+test` (ubuntu: `npm ci --ignore-scripts`, run electron's install explicitly, `node --test`, a headless smoke with `xvfb-run` that launches the app against a Go fake server and asserts the probe/navigation lock, U that Electron runs under Xvfb on the runner with `--disable-gpu`), `package` matrix (ubuntu AppImage+deb, windows NSIS+portable, macos later), `actions/upload-artifact` only. It publishes nothing: no release, no tag, no secret, same rule as `desktop-app.yml`. Actions pinned by commit SHA. It is NOT in `needs` of any job in `ci.yml` and `ci.yml` never runs Node, so the Go gates and the worker-host whole-repo gate (`go vet ./... && go test -short -count=1 -p 2 ./...`) are unaffected. A release of the Electron artifacts is a separate, user-approved step (AGENTS: no publishing overnight).

Owners: `docs/OWNERS` gets `apps/desktop-electron/` (coordinator adds; I only write my doc). Docs: `docs/desktop-app.md` gets a banner and an Electron section when phase 1 lands.

## 7. Supply chain and licences

- Electron: MIT (V: `electron/electron` LICENSE "Copyright (c) Electron contributors / Copyright (c) 2013-2020 GitHub Inc."). electron-builder MIT (V), `@electron/fuses` MIT (V), `electron-updater` MIT (V).
- The distributed bundle also contains Chromium and its dependencies under BSD, MIT, LGPL (FFmpeg), MPL, Apache and others. Electron releases ship `LICENSES.chromium.html` and `LICENSE` inside every build (S: from memory of the zip layout; the asset exists on the release page but the HEAD I did for it returned a 9-byte response, so it is a redirect or absent from that exact URL: U). We must keep both files in the package and cite them in `THIRD_PARTY.md`.
- GPL fit: the repo is GPL-3.0-or-later (THIRD_PARTY.md, V read). MIT and BSD parts are GPL-compatible; we are distributing an aggregate of our GPL code with MIT/BSD Electron, which is allowed. Obligations: provide the corresponding source of OUR code (the public repo does), keep every third-party notice. U: Electron's bundled FFmpeg (LGPL-2.1+) and codecs; the proprietary-codec patent question for H.264/AAC in the default Electron FFmpeg is not verified; the watch pages use none, but the doc must not claim "royalty free" until checked. Not legal advice; the same shape as every Electron GPL app (e.g. other GPL projects ship Electron, U).
- Our assets: icons are ours. No Diablo II art (the repo rule forbids it); the window must not display or bundle Blizzard art, and the shell does not (the server decodes art from the user's install at request time).
- Supply chain controls: exact pins plus lockfile, `npm ci --ignore-scripts` (then run only the one required `electron` install script by name), npm `audit signatures`/provenance (U available since npm 9), no `postinstall` of our own, dependency PRs reviewed by a human or the coordinator, Actions pinned by SHA, no caret ranges. @electron/get verifies the downloaded Electron zip against the release SHASUMS (S, from memory), so a mirror swap is detected; do not set `ELECTRON_MIRROR`. Artifacts upload only to the workflow; checksums (SHA-256) written to a `SHA256SUMS` file at release time (phase 4).
- Dependency surface to expect: about 10 direct deps for electron-builder, several hundred transitive (U). That is the biggest honest cost vs the Go launcher's zero dependencies; mitigations above, and `electron-builder` only lives in devDependencies (not shipped in the app).

## 8. Self-update, sandbox/dev controls, hosting later features

Self-update, in two steps:
1. Check only (phase 3, opt-in, default off): at most once a day the MAIN process fetches `https://api.github.com/repos/<owner>/<repo>/releases/latest` over HTTPS, compares tags, and shows a menu item "Update available x.y.z" that opens the release page in the default browser. No download, no install, no remote code path. Pinned URL, no redirects off github.com. The helper never sees it.
2. Real auto-update (not recommended yet): `electron-updater` with the GitHub provider would download an installer. On Windows it verifies the Authenticode signature publisher against the one embedded at build time, on macOS Squirrel.Mac requires a signed app (S: updates.md describes Squirrel and the signing requirement; electron-updater details U). Without code signing the trust root is only "whoever can publish a release in the repo", and AppImage/deb behave differently per OS (V: `autoUpdater` docs say Linux is unsupported, auto-updater.md). So defer until the user signs releases and agrees.

Hosting later features (all hang off the main process, the GAME view stays preload-less):
- Sandbox/dev controls: native menu entries that call the server's own HTTP API with the SAME session cookie, done by loading a server page (preferred, nothing new in the shell) or, if a native panel is wanted, by a second privileged HELPER-class view whose preload exposes `controls.start()/stop()` and whose main handler uses the cookie jar from the `persist:d2idle` session. Anything with side effects stays server-side behind the existing login; the shell never stores the passphrase and never auto-confirms an action.
- Notifications: main polls an events endpoint with the session cookie and shows `Notification`s; needs a server contract (not designed here).
- Deep links `d2idle://inventory` (protocol client) feed the same single-instance path.
- Each new IPC channel gets the same sender check, argument validation and a test; the rule: GAME view never gets a preload.

## 9. Go launcher: keep or retire

Recommendation: keep `apps/desktop` as the "lite" launcher for one release cycle after Electron has been live-verified on all three OSes, stop publishing it from the release workflow when the Electron build is accepted, and then move it to `apps/desktop` archive per the doc rules (stub, no deletion). Reasons to keep it meanwhile: 7 MB, zero dependencies, fully tested headless on the worker host (`testdata/smoke.sh`), and it is the only thing that works where Electron cannot run (ancient glibc, no userns). Reasons to retire later: two launchers means two config writers, two address normalisers (the test vectors are shared to keep them identical) and two docs. Decision is the user's (proposed ADR below).

## 10. Effort

Estimates assume Sonnet-class agents with a human merge; one agent-hour is a focused run including tests.

| PR | Content | Agent-hours | Verified how |
|---|---|---|---|
| 1 Scaffold + core | package.json, main/helper/ipc, address+config port with shared vectors, probe, navigation lock, fuses, tests (`node --test`), CI lint/test job, stub go.mod, docs banner | 4 to 6 | unit tests, Xvfb smoke against a Go fake server (offline); real login on a desktop: manual |
| 2 Packaging | electron-builder.yml, NSIS+portable, AppImage+deb, desktop entry actions, icons, package CI matrix, THIRD_PARTY entry | 3 to 4 | artifacts built in CI; installers run on real Windows and Linux: manual (user machine or spare Windows PC) |
| 3 Polish | menu, tray, window state decision, update CHECK, macOS build + notarisation hooks | 3 to 5 | manual on macOS (needs a macOS runner or machine; ask first) |
| 4 Later | sandbox/dev controls, notifications, release workflow with checksums | decide with the server side | |

Total about 10 to 15 agent-hours over 3 PRs before macOS signing; real-machine checks (Windows install, Ubuntu 24.04 AppArmor sandbox, macOS Gatekeeper) are user or lab tasks, not agent-verifiable here.

## 11. Open questions (user choice needed)

1. Keep the Go launcher as the lite fallback for a cycle, or retire it at once?
2. Sign Windows builds (certificate cost, SmartScreen) and enrol in the Apple Developer programme for macOS? Until then installers are unsigned.
3. Enable the `enableCookieEncryption` fuse (cookies tied to the OS user, safer on disk) or leave it off (profile copyable)?
4. Allow a `window` key in `config.json` beside `address`, or keep exactly `{"address"}` and forget window size?
5. Is a 100 MB-class download acceptable versus 7 MB?
6. Opt-in update check: allowed to contact github.com from the app?
7. Verify: UA token `Electron/x` does not affect the server; the watch pages need no browser permissions.

## 12. Dead ends and unverified items

- electron.build docs are JavaScript-rendered and the curl fetches returned 404/empty shells; electron-builder facts come from `packages/app-builder-lib/scheme.json` on GitHub (V) and npm metadata. Option names for `desktop` actions and the AppImage sandbox flag are U until packaged once.
- Nothing was built or run; the only experiment is the Go `go list` behaviour (V). All size estimates for installers are U.

## Proposed ADR text (for the user's grill session; I did not write docs/adr)

> ADR: The d.idle desktop launcher is an Electron app (`apps/desktop-electron`, electron-builder, pinned latest stable Electron). Status: proposed. Decision: portable across Windows, Linux and macOS with its own window; config stays `{"address"}` (0600); it loads the exact configured origin top-level in a sandboxed, context-isolated, preload-less view with navigation locked to that origin; the Go launcher `apps/desktop` stays as a lite fallback for one release cycle, then is archived. Consequences: a 100 MB-class download, an npm dependency tree audited by lockfile and pins, a separate CI workflow, a stub `go.mod` keeps the Go gates blind to the folder, unsigned builds until the user buys signing. Supersedes the stack decision in docs/desktop-app.md (amends, not replaces, its same-origin rule).
