# LAUNCH: prefix, install, launch, helper, OOG (agent `launch`)

> status: Game.exe launch, prefix and OOG; d2d is the primary backend (ADR 0009, 2026-10-30; docs/plan/STATUS.md). Still true for the frozen dev path on the worker host (tools/game.sh); the d2d process is started by internal/d2denv (docs/d2d-bot.md).

Tags: VERIFIED (observed, how stated) / UNVERIFIED (not observed here). Code: `internal/launch`, `tools/subreaperproof`, `tools/stubmeta`.

## 1. Subreaper proof (the key result)

VERIFIED 2026-10-28 on the worker host (kernel 7.2.9, Yama ptrace_scope=1, GE-Proton11-11), scratch prefix `~/Games/d2lod-scratch` (deleted afterwards), harmless Go windows/386 exe `tools/subreaperproof/winprobe` (launcher starts a "sleeper" child, then exits after 2 s). Run as an unprivileged user, CapEff=0, no setcap:

| Run | Result |
|---|---|
| `subreaperproof run` (calls `prctl(PR_SET_CHILD_SUBREAPER)`) | After the `proton run` chain exited, the sleeper had ppid == our pid. `open /proc/<pid>/mem` OK, read "MZ" at the image base. |
| same, from a non-descendant (`systemd-run --user`) | `permission denied`. |
| control: `-subreaper=false` | sleeper ppid=1 after the launcher exited; open `/proc/<pid>/mem` = `permission denied` even for the app that started it. |

Rules that follow:
- The process that must be the subreaper is the APP (the Go process that reads memory), set once at startup with `launch.BecomeSubreaper()` BEFORE the first Wine process starts. Reader and subreaper must be the same process (or the reader a descendant of it).
- Yama checks the real-parent chain. Wine's launchers (`proton` python, `steam.exe`) exit; wineserver, services.exe, d2input.exe and the game get re-parented. Without the prctl they go to PID 1 (or the systemd user manager) and stop being readable. The live game today has PPid 1184 (systemd user), which is why it needs CAP_SYS_PTRACE.
- Consequence: a game started outside the app (or by a previous app instance that died) is NOT readable. `Game.Attach` says so explicitly. After an app crash, stop the orphaned game (or restart with the capability) before relaunching.
- Finding the process: every Windows process is `/proc/<pid>/exe -> .../wine-preloader`; `comm` is a 15-byte thread name (for Game.exe it happens to be "Game.exe", unreliable). Match on cmdline: Wine rewrites argv so `argv[0]` is the Windows command line (`C:\Program Files (x86)\Diablo II\Game.exe -w`). `launch.FindImage(procs, "game.exe")`. The image base comes from `/proc/pid/maps` (Game.exe is file-backed at 0x400000, VERIFIED on the live game; the test Go exe was at 0x7bcf0000 because it is relocatable).
- `PR_SET_PDEATHSIG`: in Go it fires when the forking THREAD dies, so `Spawner` forks from a locked OS thread that lives for the whole process. It only covers the direct child (the python launcher), not Wine's processes, so it is a minor net. Real cleanup is `Config.WineserverKill()` (wineserver -k for the prefix) on shutdown.
- Subreaper side effect: dead orphans stay zombies until we wait on them. `launch.Reap()` (call every few seconds) reaps only non-`exec.Cmd` children.
- Other scratch findings (VERIFIED): Proton does not create `STEAM_COMPAT_DATA_PATH` (dies on missing `pfx.lock`; `ProtonCmd` mkdirs it). First `proton run` in a new dir builds the prefix itself (~25 s). SIGTERM to a Wine process ends it within 100 ms (`Managed.Stop`). `proton run` of an exe whose launcher exits returns while children live on.

## 2. Config and discovery (`config.go`)
`Config` documents every field. `Resolve()` fills defaults: Root `~/Games/d2lod` (env `D2_ROOT`), prefix `Root/proton`, game dir `.../pfx/drive_c/Program Files (x86)/Diablo II`, helper `Root/helper/d2input.exe`, `127.0.0.1:47012`, args `-w`. Runners: `FindRunners` scans `compatibilitytools.d` dirs (Steam, ~/.steam/root, /usr/share/steam), picks `GE-Proton11-11` (VERIFIED) else newest GE-Proton; env `PROTON` overrides. The worker host has many runners; only 11-11 is verified for D2.

## 3. Helper and game lifecycle
- `Helper` (`helper.go`): `Ping` (dial + `info`; "ok ..." = window up, "err ..." = alive, no window), `Ensure` (adopt a listening helper, else start via Proton and wait for readiness), `Stop` (`quit`, then signals), `Supervise` (restart with 1..30 s backoff). Only `info` and `quit` of the d2input protocol are relied on.
- `Game` (`game.go`, `managed.go`): `Launch` (cwd = game dir, `-w`), `Attach` (pid, base, `MemReader` which satisfies `mem.Source` shape), `Shutdown` (SIGTERM then SIGKILL; for a saved exit use `Flow.LeaveGame` first).
- `Monitor` (`monitor.go`): pure state machine, events Up/Exited/Crashed/Hung. Hang = alive but `CPUTicks` (utime+stime from /proc/pid/stat) flat for 45 s. VERIFIED: the idle live game gains ~11 ticks/s. UNVERIFIED: behavior in menus/loading screens, and that `BlizzardError.exe` (present in the install) is what 1.14d spawns on a crash (used as the crash signal). Better hang signal for the INPUT agent to add to d2input: `IsHungAppWindow(hwnd)` via user32.
- Offline-tested with fakes; lifecycle also run live on the scratch prefix (start, find, mem open after launcher exit, Monitor hung/exited, Stop).

## 4. Install flow (`install.go`, `download.go`, `pe.go`, `stubmeta.go`, `settings.go`)
Steps with independent verifiers (`Installer.Steps/Run`): prefix, install-d2, install-lod, patch-download, patch-apply. Done steps are skipped.
- Stub (VERIFIED by reading both stubs and the results on the worker host): it is the Blizzard Downloader (SC2/WoW-era code) with an embedded bencoded torrent. D2: "D2-1.14b-Installer-enUS", 1,551,865,181-byte `Installer Tome.mpq` + `Installer.exe` (2,664,936) + DirectX redist; LoD: "D2LOD-1.14b-Installer-enUS", 490,399,607-byte Tome + Installer.exe (2,671,080). It downloads into `C:\users\steamuser\Desktop\<name>\` of the prefix and autolaunches `Installer.exe` (the torrent says autolaunch=1). The torrent also names a CDN `direct download` base (`http://rogue.blizzard.com.edgesuite.net/applications/Diablo2/1.14B/{D2,LOD}/enUS`); probing it returns 403 from Akamai (UNVERIFIED why; likely retired or needs the stub's request shape). So the stub must be run; do not rely on direct download.
- `ReadStubMeta` parses the torrent from the stub (nothing Blizzard-owned is shipped) and `VerifyDir(dir, pieces=true)` checks every SHA-1 piece. VERIFIED: both the worker host install folders verify OK (`tools/stubmeta STUB DIR -pieces`, 0.8 s).
- Installer (`Installer.exe`) is NOT InstallShield: it is Blizzard's BAM installer (XML scripts inside the Tome, popt command line). Strings show options `--silent`, `--noui`, `--destination`, `--gamepath`, `--path=`, `--productcode`, `--locale`, `--skippostlaunch`, `--noverify`, `--hiddenProcessing` (VERIFIED strings, no help text). It asks for the CD key in its own UI (fields cdkey1/cdkey2; `{EnterGameCDKey}`), requests admin in its manifest (harmless in Wine). Running `Installer.exe --help` and `--silent --noui --destination ...` in the scratch prefix did not exit within 40-60 s (no Tome present), so silent mode is UNVERIFIED and a response-file route does not exist. Plan: the key is typed by the user in the installer window (the app never stores keys); step `Interactive=true`; completion is judged by files, not process exit.
- Verifiers VERIFIED on the real install: `Diablo II Install Log.html` / `Diablo II LOD Install Log.html` contain "Installation completed successfully"; MPQs d2data/d2char/d2sfx (base), d2exp/d2xmusic/d2xtalk/d2xvideo (LoD).
- Patch: `Fetch` with resume (Range), sha256 check before atomic rename, deletes and returns `ErrChecksum` on mismatch, refuses a download without a hash. URL/hash/size constants in `download.go` (HTTP only; HTTPS gives no answer; VERIFIED `Accept-Ranges: bytes`, size and sha256 match the file on the worker host). The patch exe is a Blizzard GUI updater that reads `InstallPath` from the registry written by the installer; it has no known silent switch (UNVERIFIED), so `patch-apply` is interactive and verified by `FileVersion`.
- `FileVersion` parses RT_VERSION in pure Go. VERIFIED: real worker host Game.exe = 1.14.3.71 (`D2_GAME_EXE=/path go test ./internal/launch -run Real -v`; the exe copy lives outside the repo). Unit-tested on a synthetic PE.

## 5. Settings (VERIFIED from the live prefix)
No `d2.ini`/`.ini` in the game dir. Settings are registry: `HKCU\Software\Blizzard Entertainment\Diablo II` (AutoMapFade, Contrast, Music Volume, NPC Speech, CmdLine, UseCmdLine, GamePath, InstallPath, `NewSavePath=C:\users\steamuser\Saved Games\Diablo II\`), `...\VideoConfig` (Render, DeviceDDraw, dwFlags), `HKLM\Software\Wow6432Node\Blizzard Entertainment\Diablo II` (InstallPath). Characters: `Saved Games\Diablo II\<name>.d2s/.key/.map/.ma0`. Per-character key bindings live in `<name>.key`. Windowed comes from the `-w` argument, not the registry (`UseCmdLine=0` here, the installer left `CmdLine=-w -skiptobnet`, UNVERIFIED whether it is honored). `Config.ReadUserReg` (read-only parse of user.reg; Wine flushes lazily) and `Config.SaveDir()`; `RegAddArgs` builds a `reg.exe add` for writes through Wine (UNVERIFIED).

## 6. D2DX (optional, `wrapper.go`)
Open source (GPL-3.0, github.com/bolrog/d2dx), README lists 1.14d as fully supported, latest 0.99.529, development paused 2021. Install = copy `glide3x.dll` into the game folder and launch `Game.exe -3dfx`; D3D11 renderer (needs DX10.1 GPU), expected to go through DXVK. Nothing about Wine in its README; works-under-Proton is UNVERIFIED. Not worth requiring: the stock DirectDraw path already renders windowed 800x600, and the bot depends on that fixed client layout (clicks, iso transform). Hook provided: `InstallGlideWrapper(cfg, dll)` returns the extra arg `-3dfx`; the user supplies the dll.

## 7. Other blockers for one-click setup
- CD keys: must be typed by the user twice (base, LoD). Not automatable by design.
- Admin manifest, DirectX redist step inside the installer (it ran fine under Proton).
- Installer and patch GUIs need a visible desktop session; a headless box needs a virtual display.
- First launch may show a "System Survey" (registry `System Survey` key exists) and `BnetLog.txt` noise; harmless.
- The Tome is 1.5 GB (+0.5 GB): disk and time. Stubs may break if Blizzard retires the CDN.

## 8. OOG flow (`oog_table.go`, `oog.go`)
Design: canaries (VERIFIED live in wave 1): player pointer `Game.exe+0x3A6A70`, FirstControl `+0x3D55BC` -> `ReadPhase` (menu / in-game / loading). Screens are recognised by matching the D2Win control list (struct layout from D2BS, UNVERIFIED live: +0 type, +8 disabled, +0xC x, +0x10 y, +0x14 w, +0x18 h, +0x3C next, +0x48 text) against signatures in the table, not by pixel guesses. Click point of a control = (x+w/2, y-h/2) (D2BS Control.cpp, bottom-anchored). `Flow.EnterGame` handles splash -> main menu -> Single Player -> char select (select existing by label text, else create) -> class -> name/hardcore/classic -> OK -> difficulty -> in game; `Flow.LeaveGame` = Esc + Save and Exit. Stuck screens return `ErrStuck`; no endless clicking. Tested offline against a simulated menu state machine over `mem.Fake`.

Coordinates are now VERIFIED LIVE: see docs/oog.md (supersedes this section 8 table).

### OOG verification checklist for wave 2 (next game restart)
1. At each screen dump `ReadControls`; compare with the table (type,x,y,w,h); flip `Verified`.
2. Does FirstControl go non-zero on the splash screen, and does the Control list exist for the in-game Esc menu (it may not)?
3. Calibrate Save-and-Exit by screenshot; confirm Esc opens the menu via posted key (key press works, VERIFIED, in menus UNVERIFIED).
4. Does a posted click (PostMessage) work on menus at all, or do menus need the real cursor/mouse-move first (the helper's `click` sends a move first)?
5. Does name typing via posted keys fill the edit box (WM_KEYDOWN without WM_CHAR may not)? Likely needs a `char` command in d2input. Request for the INPUT agent.
6. Character label text location for slot selection; behavior when more than 8 characters.

## 9. `d2bot --managed` on a dev instance (agent LIVEBOT, VERIFIED LIVE on the worker host i4, 2026-10-28)
Run: `tools/live.sh --bg --max 6600 --nocap livebot d2bot --managed --instance i4 --char Booti --addr 127.0.0.1:8420 --stop-game --run-for 100m` (`--nocap` = no setcap, no worker-sudo; the flock is still taken).
With `--instance iN` (N >= 1) `managedConfig` (cmd/d2bot/game.go) uses that instance's own prefix `~/Games/d2lod-inst/iN`, helper port 47012+N and private Xvfb `:10N` (started if missing); any other id keeps the old `--data` root. Stop the instance with `game.sh stop` BEFORE the first managed run (the app must start the game itself).
- CapEff/CapPrm of the d2bot process = 0, `getcap` empty, ptrace_scope=1. The proton python, wineserver, services.exe, d2input.exe, Xvfb and Game.exe all have PPID = d2bot (subreaper works).
- Attach, memory reads (state menu / in game, snapshots, map tracker), input through the helper (clicks, keys, OOG menus: login, Save and Exit, re-entry), and the whole leveling loop (fighting, town chores, shopping, stat/skill points, hotkeys) ran for 90+ minutes without any capability.
- A second process cannot read the game (not a descendant): `d2cheat` (memory write) still needs the setcap route, so run it with `tools/live.sh --nolock livebot d2cheat ...` next to a running managed bot (the bot holds the instance flock).
- Bugs found and fixed: (1) the managed config ignored the instance (wrong prefix, no Xvfb, helper port 47012); (2) Wine children inherited the flock fd of `flock ... d2bot` and held the instance lock after the bot exited (`launch.CloseOnExecInherited`, as d2game already did); (3) `Manager` starts the saved goals only after `POST /control/start` (or `--goals`), idle otherwise.
- A managed game outlives a bot that is killed with SIGKILL or without `--stop-game`: it is then an orphan, unreadable for the next `--managed` run ("not our descendant"); stop it with `NOLOCK=1 tools/game.sh stop iN` and let the bot start it again. SIGTERM with `--stop-game` stops the game cleanly in about 2 s. `tools/jobs.sh kill` does not signal the d2bot itself: `pkill -TERM -f '^~/d2lod-live/<agent>/d2bot'`.

Follow-ups (INTEGRATE, 2026-10-28): `--autostart` (env `D2BOT_AUTOSTART`, config key `autostart`) starts the saved goals without a `/control/start`; `--goals campaign[:normal_act1]` enqueues the Normal Act 1 campaign goal. Hotkeys are bound by the runner once it stands in a game (`Deps.Prepare`), so a login after the launch works. Shutdown waits for `Runner.Settled` (the walk home), then Save and Exit through the Cycler's flow before `--stop-game` stops the game (the .d2s only carries quests and waypoints after that). If the character cannot get to town (no portal, dungeon exit failing) the stop gives up after the timeout and the game is saved where it stands; the next start begins in town anyway.
