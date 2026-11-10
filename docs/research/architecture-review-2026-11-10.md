# Architecture review, 2026-11-10: containers plus a web UI, or an all-in-one app?

> status: REVIEW for the owner (Fable, TASK-270, branch `agent/archreview`, 2026-11-10). Docs only: nothing built, deployed or changed in the sandbox, the lab, the fork or the gates. The user's question, verbatim: "Should we rethink the containerized path with a web UI and move to an all-in-one app? I don't care how much time we have spent on the current iteration, I want the best overall approach. We went containerized since it makes it easier to distribute the app, but I am starting to think that was a huge mistake in light of it being an idle/incremental game first." Sunk cost is ignored below; only forward cost, risk and player experience count. Section 12 holds proposed decision text; the guard files are not edited.
>
> Tags: **V** read in this tree or produced by a command run in this review (named); **S** another doc's claim or a web source, not re-checked; **U** unverified estimate; **I** my judgement. Target: classic 1.14d data through d2d; nothing here is D2R. Host names are roles ("the worker host").

## 0. Answer in twelve lines

1. **Not a huge mistake.** The container-specific surface is about 1,300 lines (Dockerfile, entrypoint, compose, context, smoke, image tests, publish workflow, dev-update scripts; V `wc -l`) against about 194,000 lines of portable Go (V). Everything a native app needs already exists and compiles for Linux with CGO off: the bot with the watch screen, the wizard, the video tap, the signed updater with rollback, the boot shim, the d2d build recipe (V section 4).
2. **But Docker as the only door is the wrong primary for an idle game aimed at non-technical friends.** Docker Desktop wants 8 GB RAM, virtualization in the BIOS, WSL 2.1.5+, Windows 10 22H2 or 11 23H2 (V docs.docker.com); the image is amd64 only, so Apple silicon emulates (S); the friend edits a compose variable, reads a token from a log and types the page origin by hand (V entrypoint and compose).
3. **A single static all-in-one binary is impossible.** d2d is a C++26 program linked against SDL3, OpenAL, StormLib, libcurl and libstdc++ (V `ldd` in the sandbox), GPL-3.0, and it must stay a separate process beside our Go code (ADR 0003). "All-in-one" can only mean one folder or installer: a Go supervisor plus a d2d sidecar plus the web UI it already serves.
4. **Recommendation: C, the hybrid.** One release layout (`bin/`, `d2d/`, `data/`, which is already the signed-manifest format, V `internal/selfupdate/manifest.go`), one Go supervisor, packaged natively per OS AND as the existing container for always-on hosts (NAS, server, streamer) and for the dev sandbox. The container becomes one packaging of the product, not the product.
5. **The fastest route to a second pair of eyes is D1, a hosted beta**: the user runs one tester container per friend on the worker host; the friend opens a URL with a passcode on any device. Nothing is installed, nothing is conveyed (watching `/watch` is not distribution, S legal note), fixes reach friends in the 5-second dev-update loop (V docs/dev-update.md), and the hero plays around the clock, which no laptop gives: ADR 0016 rejects an abstract simulator, so progress exists only while the engine runs (V).
6. **Order:** D1 (about 6 to 10 agent-hours) now; C phase 1, Linux native (25 to 40 h); C phase 2, Windows (30 to 45 h, needs a fork patch because d2d's control channel is a stub on Windows, V); macOS last (15 to 25 h, needs a Mac build and a signing decision). Section 8.
7. **What no packaging changes:** d2d costs 0.4 to 0.7 of a core and about 630 MB in realtime (V sandbox `docker stats`), the hero progresses only while the host runs, and the return screen, question queue and notifications are server-side features that work the same in every option.
8. **Option A as planned** (friend-build 26 to 36 h plus the Electron launcher 10 to 15 h, S) buys friends with Docker on Linux x86-64 and "best effort" WSL2. The same hours spent on C phase 1 buy Linux friends without Docker and the base for Windows.
9. **B (native only, drop the container) is dominated by C**: it saves nothing (the container exists and costs nothing to keep) and loses the always-on case, the streamer case and the proven sandbox loop.
10. **The web UI is not the problem and stays in every option**: the phone, the stream scene and the hosted beta all need it; a native window is the existing 7 MB launcher in browser app mode (V build) or the Electron shell later.
11. **Legal:** GPL duties are per copy conveyed and identical for an image and a native package (the fork binary ships either way); the hosted beta conveys nothing; a friend's game files on the user's box is an open question (U). Section 10.
12. **What flips the recommendation** is in section 1; the matrix is in section 6, costs in 7, the migration plan in 8.

## 1. Questions for the user first (and what each answer flips)

| # | Question | If the answer is... | ...the recommendation becomes |
|---|---|---|---|
| Q1 | What do the first friends run: Windows, macOS or Linux, and do they own an always-on box? | all Linux with Docker or a NAS | keep A for them (friend-build PRs), still do D1; native waits |
| | | mostly Windows laptops | D1 first; C phase 2 (Windows) moves ahead of phase 1 (Linux) if the fork patch is acceptable (Q3) |
| | | a Mac | D1 only until phase 3; the amd64 image on Apple silicon is not an offer (S) |
| Q2 | Must the hero progress while the friend's PC is off or asleep (true idle), or is "the hero sleeps when you sleep" acceptable with an honest return screen? | must progress 24/7 | hosted or NAS containers stay primary for the serious player; native is the "try it" door only |
| | | acceptable | C as written; the return screen says "paused while your machine slept" |
| Q3 | May the d.idle fork carry a Windows transport for the control channel (AF_UNIX via `afunix.h` on Windows 10 1803+, or loopback TCP), given upstream marks the channel "never enable in shipped builds" (V fork docs)? | no fork change | Windows native is off the table; Windows friends use WSL2 Docker (A) or D1 |
| | | yes, as an upstreamable patch | phase 2 as planned |
| Q4 | Hosted beta: are you comfortable running friends' heroes on your hardware, with each friend's own game files uploaded to your box, behind your tunnel? | no | skip D1; C phase 1 is the first friend route, which delays feedback by 3 to 5 weeks (U) |
| Q5 | Is a browser tab (plus the existing app-mode launcher) enough as the UI, or is a tray icon with a native menu required for v1? | tray required | add the Electron plan (10 to 15 h, S) or a tray library, which breaks the standard-library-only rule (Q6) |
| Q6 | May the Go side take ONE third-party dependency for a tray or webview (`gogpu/systray` claims pure Go, S; `fyne.io/systray` needs cgo, S)? | no | tray through Electron or not at all; autostart and OS notifications still work by `exec` (section 5) |
| Q7 | Signing budget: a Windows code-signing certificate and an Apple Developer ID (99 USD a year, S)? | none | native packages warn on first run (SmartScreen, Gatekeeper right-click open, S); fine for friends, not for strangers |
| Q8 | Do you still want to run your own hero on the NAS or the worker host as a container? | yes | the container packaging is a first-class target of C (it is anyway); if no, it stays as the dev sandbox only |

Robustness: the recommendation (C with D1 first) holds for every answer except Q4 = no AND Q1 = Linux-Docker friends, where A as planned is enough and native is a later product step, not a beta step.

## 2. What runs today (V unless marked)

### 2.1 The image

Image `d2dbot:devp1`, 422 MB (V `docker images` on the worker host). Layers (V `docker history`):

| Layer | Size | Content |
|---|---|---|
| Ubuntu 26.04 base plus apt runtime | 228 MB | `libsdl3-0 libopenal1 libcurl4t64 libstorm9 zlib1g libbz2-1.0 tini ca-certificates`; user `d2` uid 1000 |
| `/opt/d2d/` | 15 MB | the fork's `d2d` binary (14.96 MB), built in stage 1 from the public fork tag `didle-gate2-lab3` (commit `f75816c6`) and sha256-pinned vendored CLI11, sentry-native, FFmpeg 9.0.2 (V `context.sh`) |
| `/opt/d2lod/` | 66 MB | `d2bot` 28.6 MB, `d2dsetup` 13.1, `d2dview` 13.5, `d2dsoak` 6.6, `d2dping` 4.5 (not stripped), `entrypoint.sh` |
| `/opt/didle/boot` plus `pubkeys/` | 11 MB | the boot shim `didleboot` and the baked release public keys |
| licences | 129 B | `/usr/share/licenses/d2d/` (GPL text, upstream LEGAL.md, `SOURCE.txt`) |

Game data is never in a layer: the user's `d2data.mpq`, `d2exp.mpq`, `d2char.mpq`, `Patch_D2.mpq` are mounted read-only at `/game` (V entrypoint check, V docs/tester-live.md minimal set).

### 2.2 Processes and resources (the sandbox)

The sandbox is the tester compose file with an override (container name, `cpus: 1`, update check off) and the dev-loop override, running on the worker host (V the compose directory there). `docker top` and `docker stats` (V, one sample, 2026-11-10):

```
tini -> /opt/didle/boot --launcher /opt/d2lod/entrypoint.sh tester
  -> /bin/sh /data/didle/releases/0.0.261110201746/bin/entrypoint.sh tester     (the APP LAYER: a signed dev release in the volume)
     -> d2dview  -addr 127.0.0.1:8893 -upstream /run/d2d/d2d.sock -bot-sock ... -jpeg   RSS 24 MB,  2 % CPU
     -> d2bot --backend d2d --managed --addr 127.0.0.1:8420 --char ... --d2d-dial ...   RSS 78 MB, 15 % CPU
        -> /opt/d2d/d2d --data /game --headless --devctl /run/d2d/d2d.sock --no-video  RSS 626 MB, 38 % CPU
container: 731.7 MiB of 3 GiB, 68.7 % of one core (cpus 1), 56 PIDs
```

Earlier measurements agree: a one-hour realtime session with video open averaged 39.6 percent of one core and 607 MiB (V docs/tester-live.md); d2d grows about 100 MB per first-visited level (S docs/d2d-fork.md).

### 2.3 How the bot talks to d2d

An AF_UNIX line protocol ("devctl"), one client at a time, driven from d2d's main loop (V fork `docs/control_channel.md`). d2d takes one client, so `d2dview` holds the upstream connection and serves two sockets, `d2d.sock.ctl` (full) and `d2d.sock.view` (read-only, the healthcheck) (V entrypoint). The bot launches d2d itself (`--managed`, `internal/d2dinput`), saves and quits it on SIGTERM; the shipped d2d runs without `--lab-verbs` (V docs/d2d-container/README.md). The bot's play path needs `dump`, the read-only `debug` verbs and `screenshot` (S docs/lab-verbs.md via the tester plan).

### 2.4 The update engine

Tier 2 of the self-update design is built and drilled: a signed `MANIFEST.json` (Ed25519, `cmd/d2sign`) lists files with `dest` in `bin`, `d2d` or `data` (V `internal/selfupdate/manifest.go` line 47); the client verifies, unpacks side by side under `releases/<ver>`, swaps `current`, and the boot shim confirms or rolls back (V docs/self-update-client.md, docs/dev-update.md: update, tamper, downgrade and crash rollback all proven in 5 to 6 s on the worker host). What cannot self-update is the base image, the shim and the baked keys (V design 3.9). The Tier 1 registry workflow is written and inert; nothing is published (V docs/tester-registry.md).

### 2.5 The UI

The watch screen is static HTML, CSS and JS served by `d2bot` (18 files in `internal/watch/web`, V) behind a token file and an optional passcode (`access.json`, V entrypoint); the answer route checks the page origin against `D2BOT_WATCH_ORIGIN`, which the tester sets by hand (V compose: `?set D2_WATCH_ORIGIN to the URL you open`). There is no web manifest or service worker (V grep), so the page is not installable as a PWA. Video is MJPEG from the tap's screenshots (V tester-live: about 3.4 to 3.8 frames a second).

### 2.6 The launcher

`apps/desktop` (`d2idle-watch`) is a 7 MB CGO-free Go program that opens the configured origin in a Chromium-family `--app` window, with a warming page and a `--setup` flow (V docs/desktop-app.md; V it builds for linux, windows and darwin, section 4). The Electron rebuild is designed, not built: a 100 MB-class download for an always-own window and a tray (S docs/research/desktop-electron.md).

### 2.7 Platform reach today

Linux x86-64 with Docker is the only proven host (V tester-live, first-session). Windows through WSL2 is "works if the friend is Docker-literate, must be tried before promising" and macOS is "not v1" (S docs/research/friend-build.md section 9). Docker Desktop requirements (V docs.docker.com, fetched 2026-11-10): WSL 2.1.5 or later; Windows 10 64-bit 22H2 or Windows 11 23H2 or later; "Windows Home or Education editions only allow you to run Linux containers" (which is all we need); a 64-bit CPU with SLAT; 8 GB RAM; hardware virtualization enabled in the BIOS; a per-user install needs no administrator; free for personal use and companies under 250 employees and 10 M USD revenue, paid above that and for government entities.

### 2.8 The decisions that shaped this

ADR 0001 made the container the product and the worker host the dev path (2026-10-29); ADR 0009 made d2d the backend and "v1 is the d2d container; users never run Game.exe"; ADR 0015 deferred the container until Gate A; ADR 0017 D3 un-deferred it at high priority because "remote friends are the only testers" (V all four). The user's decisions of 2026-11-06 fixed "d2d first, fork shipped with source, tarball with source, game-dir only" (V decisions.md). None of these records weighed a native build: ADR 0001 says "A Windows-native port is NOT planned; revisit only if the container is a bad experience on Windows" (V), which is the question now asked.

## 3. What an idle-first product needs

From ADR 0016 to 0019 and the incremental-games research (V the ADRs; V docs/research/incremental-arpg.md):

| Need | Source | Depends on the packaging? |
|---|---|---|
| Return screen first; away for hours, then a 5 to 20 minute return | ADR 0018 item 5 | no: server-side page, built (S first-session: never read after a real absence) |
| Questions queue while away; the hero acts on its default | ADR 0017 D2 | no |
| Progress while away | genre norm: Melvor simulates up to 24 h offline, Godville runs server-side (V incremental-arpg 8.2, 8.1) | **yes**: ADR 0016 rejects an abstract simulator, so the engine must run; a laptop asleep means no progress; an always-on host (container on a NAS, hosted) gives it |
| Runs quietly in the background, resumes on boot | user's framing | yes: tray and autostart are OS features; the container needs the Docker daemon up |
| Notifications (death, poverty, a question) | ADR 0019 alerts; senders OFF today (S beta review) | partly: OS toasts need a native process; phone push works from anywhere |
| Low resource use | genre norm (a tab) | no: d2d is 0.4 to 0.7 core and 630 MB wherever it runs (V 2.2); only hosting hides it |
| Phone access on the LAN | watch screen design | no: same server; both routes need a LAN bind and the origin |
| Install in minutes by a non-technical friend | ADR 0017 D3 (friends are the only testers) | **yes**: this is the whole question |
| Honest first run: own game files, licence note | decisions 2026-11-06, docs/tester-licence.md | partly: the wizard is portable; a native app can also find the install folder itself (U: the Windows registry `InstallPath` key, S from docs/container.md Wine-era notes) |
| Watch on a stream | docs/streamctl-d2d.md | no: the kiosk pulls the tap's MJPEG over HTTP (V) |

## 4. Feasibility facts established in this review (V unless marked)

### 4.1 Our Go side cross-builds, except one package

Run on the worker host from this worktree, Go 1.27.3, `CGO_ENABLED=0 go build -trimpath -ldflags "-s -w"` (V, 2026-11-10):

| Target | d2bot | d2dsetup | d2dview | didleboot | d2dping |
|---|---|---|---|---|---|
| linux/amd64 | OK 20.7 MB | OK 9.0 MB | OK 9.2 MB | OK 7.3 MB | OK 3.0 MB |
| linux/arm64 | OK 18.9 MB | OK 8.3 MB | OK 8.6 MB | OK 6.7 MB | (not tried) |
| windows/amd64 | FAIL `internal/launch`: `syscall.Flock` | FAIL (same) | OK 9.5 MB | FAIL `syscall.Kill` (dev.go, real.go) | OK 3.1 MB |
| darwin/arm64 | FAIL `internal/launch`: `Pdeathsig`, `SYS_PRCTL` | FAIL (same) | OK 8.8 MB | OK 6.9 MB | OK 2.9 MB |
| darwin/amd64 | FAIL (same) | FAIL (same) | OK 9.5 MB | OK 7.4 MB | (not tried) |
| `apps/desktop` launcher | OK on linux 7.3 MB, windows 7.5 MB, darwin 6.9 MB | | | | |

The whole d2bot dependency graph has exactly ONE OS-specific package: `internal/launch` (the Wine and Xvfb game lifecycle: `Flock`, `Pdeathsig`, `prctl` subreaper), imported by `internal/bot` and `cmd/d2bot` (V `go list -deps`). The d2d path does not use it at run time (the d2d lifecycle is `internal/d2dinput`, V entrypoint command line), so build tags with stubs make d2bot and d2dsetup portable; `didleboot` needs `os.Process` signalling instead of `syscall.Kill`. A whole-tree `GOOS=windows go build ./...` also fails in `cmd/d2dsoak`, `cmd/npcprobe`, `cmd/inputcal` and `internal/memfd` (lab tools, not shipped; V). Nothing in the tree uses cgo (V `go.mod`: standard library only).

### 4.2 d2d builds on three platforms upstream, ships as a Flatpak, and has no control channel on Windows

- Upstream CI matrix: `ubuntu-26.04` (g++), `macos-latest` (clang, `brew install cmake ninja sdl3 openal-soft qtbase stormlib`), `windows-latest` (MSVC 2022, SDL3 VC zip, OpenAL binaries, BtbN's LGPL shared FFmpeg DLLs, vcpkg zlib and StormLib, Qt 6.8); `cmake --build build --config RelWithDebInfo` builds all targets (V `.github/workflows/ci.yml` in the lab fork checkout; the README says "CI builds Linux, macOS and Windows and runs the tests", V WebFetch of the upstream README).
- Prebuilt releases: only `d2decomped-x86_64.flatpak` on GitHub Releases; no Windows or macOS binaries are offered (V README). The config dir is `~/.config/d2d` on Linux, `~/Library/Preferences/d2d` on macOS, `Documents\My Games\d2d` on Windows (V README).
- The fork's batch-2 branches built with MSVC 14.51 and passed 71 to 73 tests with the game-data tests skipped (V docs/research/b2-winworker-rerun.md).
- `--headless` runs on SDL's dummy video driver, which exists on every SDL platform (V `apps/d2d/main.cpp` header comment).
- **The control channel is POSIX only.** `components/devctl/devctl.cpp` has `#ifdef _WIN32` ... "Windows: no AF_UNIX; keep call sites portable with a warn-once stub": `listen()` prints "[devctl] not supported on Windows" and `active()` is false (V, read at the lab checkout head and at the shipped pin `f75816c6`; the same stub is in upstream `main`, V `git show`). So a Windows d2d cannot be driven by the bot today; a fork patch is required for any Windows native option (Q3). On macOS the POSIX branch applies.
- Windows FFmpeg: the Bink-only static build is "off for now, its configure wants MSYS2; CI ships BtbN's DLLs there" (V `apps/d2d/CMakeLists.txt`), so a Windows package carries full LGPL FFmpeg DLLs (tens of MB, U) unless the Bink-only build is made under MSYS2.
- Dynamic linking (V `ldd` in the sandbox): SDL3, OpenAL, StormLib, zlib, libcurl, libstdc++, and through them X11, Wayland, PulseAudio, OpenSSL, LDAP, libssh2. A native Linux package must bundle or static-link the first group (AppImage, Flatpak, or a static build; U which succeeds).

### 4.3 Sizes to expect (U, from the V binary sizes above)

Linux native: our four binaries 46 MB plus d2d 15 MB plus SDL3, OpenAL, StormLib, libcurl (about 8 MB, U) is roughly 70 MB unpacked, 25 to 30 MB compressed. Windows: our binaries about 40 MB plus d2d.exe plus DLLs; with BtbN's full FFmpeg the DLL set alone is likely 60 to 100 MB (U). Compare: the image is 422 MB (V) and the Electron shell alone 124 to 158 MB (S desktop-electron.md).

## 5. The options, concretely

**A. Status quo: container primary, thin launcher, signed in-app updates.** `compose.tester.yml` plus the game folder; `./install.sh`, a `didle` wrapper and a tarball with source (friend-build plan); the Go or Electron launcher opens the page; updates through the signed app layer; a rare image rebuild for base changes. Hosts: Linux x86-64 Docker; WSL2 best effort; no macOS.

**B. Native all-in-one per OS, container dropped as a product.** One download per OS (zip or installer, AppImage, `.app`) containing: the Go supervisor (the boot shim's release selection and rollback, the entrypoint's mode logic moved from shell to Go as the design already asks, V self-update-design 3.9), `d2dsetup`, `d2bot` with the embedded web UI, `d2dview`, the d2d build for that OS with its libraries, licences and `SOURCE.txt`. Autostart (systemd user unit, Windows Startup entry or Run key, macOS LaunchAgent), OS notifications by `exec` (`notify-send`, PowerShell toast, `osascript`; standard library only), the window through the existing app-mode launcher or the system browser, a tray later. Updates: the same signed manifest with per-OS asset names (`didle-bin-vX-windows-amd64.zip`...), `current` as a junction or a pointer file where symlinks are awkward (Windows, U). Data dir: the OS user data dir (same shape as `/data`).

**C. Hybrid: one layout, several packagings.** Everything in B, plus the container kept as the Linux packaging of the same release layout (it already runs the app layer from `/data/didle/releases/<ver>`, V 2.2) for NAS, servers, streamers and the dev sandbox. One manifest lists per-OS assets; the shim picks its own. The container gets no features the native build lacks and vice versa (the ADR 0001 rule, kept).

**D1. Hosted beta (not a product shape; a beta route).** The user runs one tester container per friend on the worker host with distinct ports, volumes and origins, behind the existing TLS tunnel; each friend gets a URL and a passcode (the `access.json` gate exists, V entrypoint); the friend uploads their own four MPQ files once (about 760 MB, V tester-live). Updates are the user's `tools/dev-update.sh` (5 to 6 s, V). Cost per friend: about one core and 0.75 GB (V 2.2) on a host that is also the lab (capacity rule, S AGENTS.private).

**D2. PWA and notification add-ons (orthogonal).** A web manifest and a service worker make the watch page installable on a phone home screen and give an offline "last known state" (2 to 4 h, U); Web Push needs HTTPS and a push service, so phone alerts stay a sender (ntfy-style) question. Applies to every option; not scored separately.

**Rejected on sight:** Tauri or Wails shells (a second language or cgo, already rejected in docs/desktop-app.md; Tauri's sidecar pattern would only wrap what the Go supervisor does); a Flatpak of our own as the only Linux route (its sandbox and portals complicate reading the game folder and serving on the LAN, U); shipping game.exe under Wine (ADR 0009 froze it; a bot on game.exe is the EULA "bots" clause in the letter, S legal note).

## 6. Decision matrix

Scores 1 (worst) to 5 (best), my judgement (I) on the evidence in the notes. "B/C native" is the native build of B and C; C's container column is A's where it differs. D1 is scored as a beta route for a few friends, not as how a stranger would get the game.

| Criterion (weight) | A container + launcher | B native only | C hybrid | D1 hosted |
|---|---|---|---|---|
| Install friction, non-technical friend (3) | 2 | 4 | 4 | 5 |
| First run: game files, licence note (2) | 3 | 4 | 4 | 2 |
| Updates and rollback (2) | 4 | 3 | 3 native, 4 container | 5 |
| Idle: quiet in the background (1) | 3 | 4 | 4 | 5 |
| Idle: resumes on boot (1) | 3 | 4 | 4 | 5 |
| Idle: notifications (1) | 2 | 4 | 4 | 3 |
| Idle: low resource use on the friend's machine (1) | 2 | 3 | 3 | 5 |
| Idle: phone access (1) | 3 | 3 | 3 | 5 |
| Idle: progress while away (3) | 3 | 2 | 3 | 5 |
| OS reach (2) | 2 | 3 now, 5 after phases 2 and 3 | 3 now, 5 later | 5 |
| Security surface (1) | 3 | 3 | 3 | 4 friend, 2 user |
| d2d dependency model (1) | 5 | 2 | 3 | 5 |
| GPL and EULA (1) | 3 | 3 | 3 | 4 |
| Supportability when it breaks (2) | 3 | 3 | 3 | 5 |
| Dev and test loop (1) | 5 | 3 | 5 | 5 |
| Stream and overlay (1) | 4 | 3 | 5 | 4 |
| Migration, reuse of today (1) | 5 | 4 | 5 | 5 |
| **Unweighted total** (max 85) | **55** | **55** | **62** | **76** |
| **Weighted total** (weights above, max 125) | **77** | **80** | **89** | **113** |

Evidence per row:

- **Install friction.** A: Docker Desktop's requirements (V 2.7), the compose edit, the token in the log and the hand-typed origin (V), amd64 only. Native: download, unpack, pick a folder; unsigned binaries warn (S). D1: a link and a passcode.
- **First run.** The wizard (`d2dsetup`) is the same everywhere (V). A native build can also look for the install folder itself (U), and the licence note is one page in both. D1 scores 2 because the friend must get 760 MB of their own files onto the user's box and the legal reading of that is open (U).
- **Updates.** A: built, drilled, nothing to add (V dev-update). Native: the engine and shim are Go and portable in design, but `didleboot` uses `syscall.Kill` (V 4.1), the `current` symlink swap and replacing a running `.exe` need Windows handling (U), and a release has per-OS assets (three sets to build and sign). D1: the user updates everyone in seconds.
- **Quiet, boot, notifications.** A: Docker Desktop must be running (its own tray icon, its own updates); the container restarts with `unless-stopped` (V compose) but cannot show a desktop notification. Native: a user service or login item and OS toasts by `exec` (standard library). D1: nothing on the friend's machine; alerts are in-page or phone push.
- **Low resource.** d2d is the cost everywhere (V 2.2); on Windows and macOS Docker adds a VM on top (S); hosted hides it.
- **Phone access.** Identical server; both A and native need the LAN bind, the Windows firewall prompt (native) and the origin value. D1 is reachable from anywhere through the tunnel with HTTPS.
- **Progress while away.** The product's idle promise. The engine must run (ADR 0016, V). A laptop asleep stops both a container and a native app; a NAS or hosted container never sleeps. B scores lowest because it drops the always-on packaging.
- **OS reach.** A: Linux Docker proven; WSL2 untested; macOS none (V, S). Native: Linux today with the container's d2d; Windows after a fork patch (V 4.2) and a DLL bundle; macOS after a Homebrew-dependency build and bundling (S CI recipe).
- **Security.** A: `cap_drop ALL`, `no-new-privileges`, read-only game mount, cgroup limits (V compose) but Docker itself is root-equivalent on Linux or a VM elsewhere; HTTP on the LAN. Native: runs as the friend's user, loopback by default, no sandbox on Windows or macOS, `systemd` hardening possible on Linux; the control socket dir is 0700 (V Dockerfile). D1: the friend runs nothing; the user exposes a service to the internet (token plus passcode, TLS at the tunnel).
- **d2d dependency.** A: apt libraries, the same toolchain as upstream CI (V). Native: per-OS builds that upstream itself does not ship (V Flatpak only), the Windows channel stub (V), DLL and dylib bundling, behaviour on other platforms unmeasured (all fidelity evidence is Linux, V STATUS and probes).
- **GPL and EULA.** Section 10.
- **Supportability.** A: `docker logs`, the planned report bundle; failure classes are Docker's (volume uid, WSL mounts, ports; S container docs). Native: logs in a known folder, OS-specific failures, no remote hands. D1: the user owns the containers.
- **Dev loop.** A and C keep the sandbox plus `dev-update.sh` (V) and the lab untouched; B would keep the container for dev anyway or lose the proven loop; native adds cross-build jobs to CI.
- **Stream.** The kiosk on the worker host pulls the tap's `/stream` over HTTP (V streamctl-d2d); a server-side container is the natural streamer case; a native streamer feeds OBS the same MJPEG. C serves both.
- **Migration.** All Go is reused in every option; A reuses the most as-is; B drops the deploy files (1,300 lines, V).

Reading the totals: C wins among product shapes on both counts; D1 wins as a beta route because it removes the friend's machine from the problem entirely, which is also why it cannot be the product (the user's hardware, legal comfort and attention do not scale). A and B tie: A is cheaper and proven, B fits the idle framing better but loses the always-on and streamer cases.

## 7. Costs and risks

Estimates are agent-hours for Sonnet-class agents with a human merge, in the style of the other plans (U unless marked). "Live check" rows are user or live-agent time on real machines.

### 7.1 Option A (as planned in the repo)

| Work | Source | Hours |
|---|---|---|
| Friends beta pack, install.sh, feed, wrapper, drills (6 PRs) | S friend-build section 10 | 26 to 36 |
| Electron launcher (3 PRs) | S desktop-electron section 10 | 10 to 15 |
| Registry publish (user steps) | S tester-registry | about 1 user hour |
| **Total** | | **36 to 51**, Linux Docker friends only; WSL2 untested |

### 7.2 D1 hosted beta

| PR | Content | Hours |
|---|---|---|
| H1 | ops script: N tester containers from `compose.tester.yml` with per-friend project name, ports 8891+2n and 8892+2n, volume, `D2_WATCH_ORIGIN=https://<friend-name>.<tunnel>`; a reverse-proxy or tunnel map; a one-page friend note (URL, passcode, "upload your four MPQ files here") | 4 to 6 |
| H2 | origin from the request: accept the `Host`-derived origin for the answer route when it matches a configured host list, so a tunnel hostname needs no hand edit (bot, `internal/watch`; the beta review's 403 trap) | 2 to 4 |
| live | user: tunnel entries, the first friend's files, one evening | 1 to 2 user hours |
| **Total** | | **6 to 10**, plus about 1 core and 0.75 GB per friend on the worker host |

### 7.3 Option C (hybrid) by phase

| Phase | PR | Content | Hours |
|---|---|---|---|
| 1 Linux native | C1 portability chore | build tags in `internal/launch` (`_linux.go` plus stubs for other OSes), `didleboot` signalling through `os.Process`, `GOOS=windows,darwin go vet` in CI | 4 to 6 |
| | C2 `didle` supervisor | the boot shim runs natively: data dir from `os.UserConfigDir`/`UserCacheDir`, release layout `releases/<ver>` and `current` under it, the entrypoint's mode logic moved from `entrypoint.sh` into Go (`bin/entry-modes` of the design, V 3.9), d2d spawned without the `nice` prefix where it does not exist, loopback origin by default, `--open` through the `apps/desktop` code | 8 to 12 |
| | C3 Linux package | tarball or AppImage with d2d and its four libraries (`linuxdeploy` or a static d2d attempt, 2 to 4 h experiment), `install.sh` writing a `systemd --user` unit (autostart, hardening) and a `.desktop` entry, uninstall, the licence directory | 6 to 10 |
| | C4 release assets | per-OS asset names in `app-assets.sh` and the manifest; the updater picks `runtime.GOOS/GOARCH`; the Linux d2d comes from the existing Dockerfile builder stage | 2 to 4 |
| | live | a clean Linux machine or VM: install, wizard, one hour of play, update, rollback | 2 to 4 |
| | **Phase 1 total** | | **25 to 40** |
| 2 Windows | C5 fork patch | a Windows transport for devctl: AF_UNIX through `afunix.h` (Windows 10 1803+) keeping the path model, or loopback TCP with a token; tests; the lab verbs stay gated; offered upstream as a draft (user go per the fork rules) | 8 to 16 |
| | C6 d2d.exe build | a workflow job mirroring upstream's Windows CI (MSVC, SDL3 VC, OpenAL, vcpkg zlib and StormLib, FFmpeg DLLs), `d2d` target only; licence notices for the DLLs; optional MSYS2 Bink-only FFmpeg to shrink the set (plus 4 to 8) | 6 to 10 |
| | C7 Go side on Windows | `-H=windowsgui` launcher, Startup-folder shortcut or `reg add` Run key, `current` as a junction or pointer file, replace-on-restart for a running exe, firewall note for the LAN bind, path handling for the game folder | 4 to 8 |
| | live | the Windows build worker or a spare PC: install, wizard, play, update; SmartScreen behaviour | 4 |
| | **Phase 2 total** | | **30 to 45** |
| 3 macOS | C8 | arm64 d2d from Homebrew dependencies, dylib bundling, `.app` with the supervisor, LaunchAgent, Gatekeeper note (unsigned) or notarisation (Q7); needs a Mac with the command-line tools | 15 to 25 |
| 4 polish | C9 | tray and native menu (the Electron plan, 10 to 15 h, S) or a tray library (Q6); PWA manifest (D2, 2 to 4 h) | 12 to 19 |
| **C total** | | phases 1 to 3 | **70 to 110**, of which 25 to 40 gives the first native friend |

Option B is C without the container packaging: the same hours, minus nothing (the deploy files already exist), plus the loss of the sandbox loop unless the container is kept for development anyway.

### 7.4 Risks

| # | Risk | Hits | Mitigation |
|---|---|---|---|
| R1 | The Windows control-channel patch is a fork divergence on a component upstream calls dev-only (V); a TCP transport widens exposure to every local process | C phase 2 | AF_UNIX via `afunix.h` keeps the file model; token on the socket path dir; offer upstream; if refused, Windows friends stay on WSL2 or D1 |
| R2 | d2d behaviour on Windows and macOS builds is unmeasured: every fidelity and gate record is Linux (V) | C phases 2 and 3 | the drill suite on the new build before any friend copy; one 45-minute smoke per OS |
| R3 | Unsigned native binaries: SmartScreen, Gatekeeper, antivirus false positives on a Go exe (S) | C | friends are told; signing is Q7 |
| R4 | Three packagings for one person to support | C | one layout, one manifest, one supervisor; the container stays the reference; docs name one supported OS at a time |
| R5 | "The hero sleeps when you sleep" disappoints an idle player | B, C native | the return screen states it; the NAS or hosted container is the always-on offer (Q2) |
| R6 | Linux library spread: a tarball that works on Ubuntu 26.04 may not on an older glibc or without X11 libraries on a headless box (U) | C phase 1 | AppImage or a static d2d; test on two distributions |
| R7 | Resource: d2d on a laptop (0.4 to 0.7 core, 630 MB, growing per level, V/S) | every friend-side option | state the requirement; a restart of d2d on a memory ceiling (the lifecycle has `Restart`, S design) |
| R8 | Hosted: friends' game files on the user's box; a service exposed to the internet | D1 | own files only, passcode plus token, TLS at the tunnel, no lab verbs, per-friend container, user's call (Q4) |
| R9 | Scope creep: the native build grows features the container lacks (the ADR 0001 rule in reverse) | C | one release layout, parity tests in CI (`tester_image_test.go` style) |
| R10 | The friend-build PRs already queued (TASK-29, 27, 48, 120...) are partly wasted if C starts now | A to C | the pack, report bundle and guide carry over; only `install.sh`'s Docker checks and the image feed change |

## 8. Recommendation and migration plan

**Recommendation.** Adopt C: one release layout, one Go supervisor, native packages per OS and the container as one packaging. Start with D1 to get feedback this month, then phase 1 (Linux native) because it reuses the container's d2d binary and proves the supervisor without any d2d porting, then Windows when Q3 is answered, macOS last. Keep the web UI; keep the Go launcher; defer the tray.

**Why not keep A as the primary.** The idle framing makes the friend's machine the host, and Docker on a friend's machine is the heaviest dependency we could pick: a daemon or a VM, 8 GB RAM, virtualization, WSL2, and a terminal for every step (V 2.7, V compose). The container's real strengths (immutable base, sandbox, one-command rebuild, always-on on a NAS) are strengths of a server, and that is where it should stay the primary.

**Why not B.** It throws away the one packaging that gives true idle progress and the proven dev loop, for no saving.

### 8.1 Phases

0. **Now (this week): D1 hosted beta** (section 7.2) and the two measurement steps the project review already asked for (fresh first hour, 4-hour away run) on the sandbox. Friends see the loop; nothing is shipped.
1. **Weeks 1 to 3: C phase 1, Linux native** (C1 to C4, live check). Deliverable: `didle-linux-amd64.tar.gz` (or AppImage) with `install.sh`; the same signed manifest serves it and the container.
2. **Weeks 3 to 6: C phase 2, Windows** (C5 to C7), gated on Q3 and a fork patch draft the user approves. Deliverable: `didle-windows-amd64.zip` and an optional installer.
3. **Later: macOS** (C8), on a Mac with the command-line tools, gated on Q7.
4. **Polish**: tray or Electron (Q5, Q6), PWA manifest, OS notifications.

### 8.2 What is reused, adapted or dropped

| Today | In C | Change |
|---|---|---|
| `d2bot` with ladder, watch, updater client | reused as is | build tags for `internal/launch` (C1) |
| `d2dsetup` wizard | reused | optional install-folder detection |
| `d2dview` tap | reused (already portable, V) | none |
| `didleboot` shim | adapted | signals through `os.Process`; native data dir; `current` on Windows |
| `entrypoint.sh` | replaced | mode logic moves into Go (the design wanted this anyway, V 3.9); the container entrypoint becomes a two-line call |
| `Dockerfile.d2d`, `context.sh`, compose files | kept | the container packaging; d2d builder stage also feeds the Linux native asset |
| signed release format, `d2sign`, `app-assets.sh` | reused | per-OS asset names |
| `tools/dev-update.sh` sandbox loop | reused | also targets a native install dir |
| `apps/desktop` launcher | reused | becomes `didle --open` or stays a sibling |
| friend-build plan PRs 1, 4, 5 (pack, wrapper, guide, report) | adapted | pack carries native archives too; `install.sh` loses the Docker checks on the native path |
| friend-build PR 2 (non-DEV feed URL) | reused | unchanged |
| Electron plan | deferred | only if Q5 says tray |
| lab, gates, lockstep, hidden instances, the fork's lab verbs | untouched | nothing in this review changes them |

### 8.3 Effect on the dev and test loop, the lab, CI, the stream

- **Sandbox:** unchanged; `dev-update.sh` keeps updating the container in 5 to 6 s (V). A native install on the worker host could be a second sandbox target later.
- **Lab:** hidden instances, lockstep, the lab binary and the gate ladder are not touched; the shipped d2d stays the public tag built without `--lab-verbs` (V).
- **CI:** add `GOOS` matrix vet and build jobs (cheap); a Windows d2d job mirrors upstream's; the tester-image workflow stays. The whole-repo gate on the worker host is unaffected.
- **Stream and overlay:** the kiosk pulls the tap's MJPEG (V); a container on the worker host remains the streamer's shape; a native user can point OBS at the same URL. The art-off default on streamed views (legal note) is unchanged.
- **Measured runs:** none are affected; `D2BOT_RUN` semantics stay.

## 9. Effect on the friend's experience, side by side

| Moment | A (Docker) | C native | D1 hosted |
|---|---|---|---|
| Before play | install Docker Desktop (8 GB RAM, BIOS virtualization, WSL2), get the pack, edit two variables, `./install.sh`, read the token (V, S) | download one archive, run `install.sh` or the installer, pick the game folder in the wizard | open a link, type a passcode, upload four files once |
| First hour | the same wizard and watch page (V) | same | same |
| Away | hero runs while the Docker host runs; a laptop asleep pauses it | hero runs while the PC is awake; the return screen says so | hero runs around the clock |
| Update | banner click or `./didle update` (planned, S) | banner click; the shim restarts the supervisor | nothing; the user deploys |
| Something breaks | `./didle report`, Docker-class failures | a logs folder and the same report bundle | the user looks at the container |
| Notifications | in-page; phone push if a sender is turned on | OS toast plus in-page | in-page; phone push |

## 10. GPL and EULA by option (NOT LEGAL ADVICE; builds on docs/research/legal-note.md and docs/d2d-container/gpl.md)

- **Conveying the fork binary** triggers the same duties whether it sits in an image layer or in a zip: the GPL text, the written offer, the exact source at the shipped commit, the vendored inputs and build recipe (V legal note section 1). The repo already does this for the image (`SOURCE.txt`, the public fork tag `didle-gate2-lab3`, V `context.sh`); a native package needs the same files in its licence directory plus notices for the bundled runtime libraries (SDL3 zlib licence, OpenAL Soft LGPL, StormLib MIT, FFmpeg LGPL on Windows: S gpl.md). LGPL libraries shipped as DLLs or shared objects keep the user's right to replace them, which static linking would complicate (U; prefer dynamic for OpenAL and FFmpeg).
- **Our own code** is GPL-3.0-or-later (ADR 0010) and talks to d2d over a socket as a separate program (ADR 0003); nothing changes by packaging the two in one folder instead of one image ("aggregate", S gpl.md).
- **Blizzard posture** is unchanged: own game files, read in place, offline, no Battle.net, no art in anything we ship, "not affiliated" notice, no Blizzard marks in package or domain names (V legal note). A native Windows app that reads the friend's own install folder is the same posture as the container reading a mounted folder.
- **Hosted (D1)** conveys no software: a friend who only uses the page receives no copy (S legal note section 1, "a friend who only watches /watch gets no copy"). The new question is the friend's game files on the user's machine for the friend's own play; the legal note does not cover it (U). Ask it in the one-page risk note the project review already requests.
- **Package count multiplies the paperwork, not the duties**: three archives need three `SHA256SUMS`, one `SOURCE.txt` each, and the same reviewer question (ADR 0003: none named).

## 11. Dead ends and unverified

- I did not build d2d for Windows or macOS, bundle it, or run any native supervisor: the Windows control-channel finding is from the source (V); the DLL sizes, AppImage or static-link success, Gatekeeper behaviour and the `current` swap on Windows are U.
- The cross-build matrix is `go build` only; no native binary was run outside Linux (V builds, U behaviour).
- The status board was unreachable from this run (HTTP 403); the agent status fell back to local files (V tool output).
- Melvor Idle's Steam wrapper (Electron or NW.js) could not be confirmed; the search only confirmed browser, Steam, iOS and Android with a cloud account (S). Kolbot SoloPlay ships as a GitHub download dropped into the D2BS tree, "no config files", legacy D2 only (S search results). Progress Quest and Godville are a download and a web game (S incremental-arpg). Pattern: idle games ship as a tab first and a Steam shell second; D2 bots ship as a folder next to the game. Neither ships a container.
- Docker Desktop requirement lines are V from docs.docker.com on 2026-11-10; the subscription FAQ confirms the 250-employee and 10 M USD thresholds (V) but does not spell out "personal use" beyond the free Personal plan.
- `gogpu/systray` ("pure Go, zero CGO") and `fyne.io/systray` ("requires cgo") are search-result claims (S), not built.
- The sandbox hero is a long-lived Paladin on a dev image; its CPU and memory are one sample (V) consistent with the one-hour tester session (V).

## 12. Proposed decision text (for the coordinator or the user's grill session; not written to docs/decisions.md or docs/adr)

**Current truth bullet (replace the "Tester build" and "Deferred or parked: d2d container" wording):**

> **Distribution (user, 2026-11-xx):** d.idle ships as ONE release layout (`bin/`, `d2d/`, `data/`, signed manifest) in several packagings: native per OS (Linux first, Windows after the fork gains a Windows control channel, macOS last) and the container for always-on hosts (NAS, server, streamer) and the dev sandbox. The container is a packaging, not the product (amends ADR 0001 and ADR 0009 item 1). The first friends play a HOSTED beta on the owner's host (one container per friend, own game files, passcode) before any copy leaves the machine. The web UI stays the UI; a native window is the app-mode launcher; a tray is a later decision.

**ADR 0020 (proposed): One release layout, many packagings; hosted beta first.**

> Status: proposed. Context: ADR 0001 made the container the product and ruled out a native port; ADR 0017 D3 made the tester container the friends route because remote friends are the only testers. The architecture review of 2026-11-10 found the container-specific surface small (about 1,300 lines), the Go side portable except one package, d2d buildable on three platforms but without a control channel on Windows, and Docker Desktop a heavy prerequisite for a non-technical friend; it also found that an idle game needs an always-on host for progress while away, which a laptop cannot give and a container on a NAS can. Decision: (1) one signed release layout and one Go supervisor; (2) packagings: native Linux, native Windows (after a fork patch for the control channel, offered upstream), native macOS, and the container; no packaging gets features another lacks; (3) the first friends test through a hosted beta run by the owner; (4) the web UI remains the UI on every packaging; the app-mode launcher is the native window; a tray or Electron shell is decided separately; (5) GPL duties are met per package exactly as for the image. Consequences: the friend-build plan's pack, wrapper, guide and report carry over with native archives added; the Electron plan is deferred; CI gains cross-build jobs; the lab, gates and fork rules are unchanged. Amends ADR 0001 (no native port) and ADR 0009 item 1 (v1 is the container); ADR 0017 D3 stands with the hosted beta as its first form.

## Sources

Repo (V read in this worktree at `a2b2bcf9`): AGENTS.md, docs/START-HERE.md, docs/decisions.md, ADR 0001, 0003, 0009, 0015, 0016, 0017, 0018, 0019, docs/research/{friend-build,desktop-electron,legal-note,tester-build-plan,incremental-arpg,beta-readiness-review,project-review-2026-11-10,direction-review,self-update-design,b2-winworker-rerun}.md, docs/{desktop-app,container,d2d-container-bot,dev-update,self-update-release,self-update-client,tester-live,tester-registry,streamctl-d2d,overlay}.md, docs/d2d-container/{design,README,gpl}.md, deploy/d2d-container/{Dockerfile.d2d,entrypoint.sh,compose.tester.yml,context.sh}, internal/selfupdate/manifest.go, internal/watch/web, go.mod, backlog/tasks/task-264 and 265.
Commands (V, 2026-11-10): on the worker host `docker ps`, `docker images`, `docker stats --no-stream`, `docker top`, `docker history`, `docker exec ... ls; ldd /opt/d2d/d2d`, the sandbox compose files; the lab fork checkout: `git log -1`, `.github/workflows/ci.yml`, `CMakeLists.txt`, `apps/d2d/CMakeLists.txt`, `apps/d2d/main.cpp`, `components/devctl/devctl.cpp` at HEAD, at `f75816c6` and at upstream `main`, `docs/control_channel.md`; from this worktree `CGO_ENABLED=0 GOOS=... go build -trimpath -ldflags "-s -w"` for five binaries and the launcher on linux/amd64, linux/arm64, windows/amd64, darwin/arm64, darwin/amd64; `go build ./...` for windows and darwin; `go list -deps ./cmd/d2bot`; `wc -l` on the deploy files and the Go tree.
Web (fetched 2026-11-10): https://github.com/psi29a/d2decomped (README: platforms, Flatpak, build steps); https://docs.docker.com/desktop/setup/install/windows-install.md (requirements); https://www.docker.com/pricing/faq/ (subscription terms). Search results (S): Tauri and Electron size comparisons (dev.to, betterstack), Wails v3 status (toolradar, youngju.dev), systray libraries (github.com/fyne-io/systray, dev.to on gogpu/systray), Melvor Idle platforms (Steam store, gigazine), Kolbot SoloPlay install (bhdocs.github.io, blizzhackers.discourse.group).
