# Wine mount sandbox (agent SANDBOX)

> status: STALE: Wine mount sandbox; Wine container work is PAUSED (docs/decisions.md, ADR 0002); d2d is the likely primary backend, see docs/d2d-container/ and docs/d2d-backend.md.

User requirement: "The prefix D2 runs in should only mount explicitly what it needs."
Tags: V = verified live on the worker host (i4, 2026-10-29, bubblewrap 0.13.1, GE-Proton11-11, kernel with unprivileged userns,
Yama ptrace_scope=1), U = unverified.

## Why
The worker host has two CIFS mounts to Tower (`/mnt/appdata`, `/mnt/media`). With the NAS down they hang and Wine/Game.exe stalls
enumerating mounts: Game.exe starts and never creates a window (V: G1 attempts 2 and 3). Fixing the host is not enough, the
prefix should not see anything it does not need.

## What it does
Every Wine process of an instance (Game.exe via proton, the d2input helper, the prefix wineserver/explorer/services, prefix
creation and the installers) is started as `bwrap --unshare-user <allowlist> -- proton run ...` (`internal/launch/sandbox.go`).
The kernel gives the children a private mount namespace in which ONLY the allowlist exists. Entry points that are wrapped:
`Config.ProtonCmd` (d2bot managed launch via `Managed.Start`, installer, setup wizard) and `Life.Start` / `Life.EnsureHelper`
(`cmd/d2game`, `tools/game.sh start|helper`). `wineserver -k/-w`, `Xvfb`, rsync and the reaper run on the host, unwrapped.
Not wrapped: `tools/d2-prefix.sh` and `tools/helper.sh` (hand debugging and the i0 helper, plain shell).

Namespaces: user + mount ONLY. No pid, net or ipc namespace (V: pids stay visible so `Managed.Find`, `mem.FindGamesFor`,
`game.sh stop` and the reaper work; 127.0.0.1 helper ports and `/tmp/.wine-UID` sockets are shared with the host). Not used:
`--die-with-parent`, `--new-session`, `--unshare-ipc` (V: it works too but buys nothing for mounts).

## On / off
- Default: ON for i1..i9 when `bwrap` is found and a probe run works (the probe launches `/usr/bin/true` with the same allowlist).
  i0 (Testy, the real desktop game on :1 with the user's X authority) stays OFF by default: UNVERIFIED there. `D2_SANDBOX=1` opts in.
- `D2_SANDBOX=0` (also off/false/no) disables it everywhere. `D2_SANDBOX=1` (or `Config.Sandbox.Mode = "on"`) makes an
  unusable bwrap an error instead of a fallback. Env beats config. `D2_BWRAP=/path` or `Config.Sandbox.Bwrap` picks the binary.
- Auto mode without a working bwrap (missing, userns disabled, AppArmor) falls back to the old unsandboxed launch with ONE
  logged warning, and the hung-mount guard (`mounts.go`: refuse to launch while a network mount hangs) stays in force.
  With the sandbox active that guard only logs (`network mount hung, launching anyway`).
- Doctor: check `host.sandbox` (warn, never fail) in `internal/setup/checks.go`; `Config.SandboxStatus()` is the one-line reason.
- `tools/soak/soak24.sh` preflight: a hung network mount is a WARNING when the sandbox is on (instance != i0, bwrap present on
  the worker host, `D2_SANDBOX` not 0), still a FAIL otherwise.

## The allowlist (single place: the vars at the top of the table in `sandbox.go`, expanded by `BuildSandbox`)
| path | mode | why |
|---|---|---|
| `/usr`, `/etc` | read-only | system libraries, wine/proton python, fonts, ld.so |
| `/bin /sbin /lib /lib32 /lib64` | same symlink as the host (usr-merge), else read-only | loader and tools |
| `/dev` | dev-bind | graphics, `/dev/shm`, futex/eventfd; no file system mounts of interest |
| `/proc` | bind of the host `/proc` | a fresh procfs needs a pid namespace; must stay visible for the bot anyway |
| `/tmp`, `/run` | empty tmpfs | private scratch; `/run/user/UID` exists empty (mode 0700) |
| `/tmp/.X11-unix` | bind | the X socket of the private Xvfb `:10N` |
| `/tmp/.wine-UID` | bind (created on the host first) | wineserver sockets: `wineserver -k` and the reaper reach them from outside |
| `$HOME` | empty tmpfs | other homes, Downloads, `.ssh` ... absent; the entries below are bound into it |
| instance root (`Config.Root`, = `~/Games/d2lod-inst/iN`; also PrefixDir, LogDir, GameDir when elsewhere) | read-write | the prefix (saves, registry), logs |
| dir of `HelperExe` (`~/Games/d2lod/helper`) | read-only | the shared d2input.exe |
| the runner dir (`.../compatibilitytools.d/GE-Proton11-11`) | read-only | proton, wine, DXVK (NOT the whole compatibilitytools.d) |
| `SteamRoot` (`~/.local/share/Steam`) | empty dir | `STEAM_COMPAT_CLIENT_INSTALL_PATH` must exist, nothing in it is needed |
| absolute paths among the command's arguments and its cwd (installer exe, patch) | read-only, skipped when already under a bound dir | so `proton run /x/setup.exe` can open it |
| `$XAUTHORITY` | read-only, i0 only | the real desktop game |
| `Config.Sandbox.ExtraRO/ExtraRW`, env `D2_SANDBOX_RO` / `D2_SANDBOX_RW` (colon lists) | as named | per-instance override |

Absent on purpose: `/mnt/*`, `/media`, `/run/media`, `/run/user/UID/{gvfs,doc}` (FUSE), other homes, `/opt`, `/srv`, `/var`,
`/sys`, the user's other Games dirs. Paths in the list that do not exist on the host are skipped, never created (except
`/tmp/.wine-UID`). The golden command line is `TestSandboxGoldenArgv`.

## How to add a path
1. Needed for one instance or one machine: set `D2_SANDBOX_RO=/a:/b` / `D2_SANDBOX_RW=/c` in the environment of the launcher,
   or fill `Config.Sandbox.ExtraRO/ExtraRW`.
2. Needed always: add it to `sandboxRO` (or `sandboxTmpBinds`) in `internal/launch/sandbox.go`, update the golden test and this table.
3. Never add a network mount. A new bind of something under `$HOME` needs its parents to exist on the host (bwrap creates the
   mount points inside the tmpfs itself).

## Live results (instance i4, main menu only, no character loaded)
Rig: `tools/rigs/sandboxlive` (`D2_INSTANCE=i4 tools/live.sh --nocap sandbox sandboxlive`, managed launch as d2bot does it).
The worker host had the two dead CIFS mounts in /proc/mounts during all runs (`stat /mnt/media` fails; the NAS itself answered ping by then).
- (a) V memory attach from OUTSIDE the sandbox: `bwrap --unshare-user` keeps the same uid mapped to itself, so `/proc/<pid>/mem`
  reads across the user-namespace boundary work, both as the descendant subreaper (managed d2bot, no capability: read "MZ" at
  0x400000) and as a non-descendant with the setcap'ed live.sh tools (d2snap RVA reads). bwrap itself stays as an extra parent
  between the app and the python launcher (V: `bwrap -> sleep`), so Game.exe is still a descendant. `--unshare-ipc` made no difference.
- (b) V helper: d2input started in the sandbox answers `info` on 127.0.0.1:47016 (`ok hwnd=0x40070 client=800x587 visible=1` once
  the game is up). Input injection itself (posted mouse/keys) was not exercised: it needs a loaded character and goes through the
  same wineserver, so no change is expected (U).
- (c) V the window appears with the dead `/mnt/media` present: title screen screenshot via `tools/shot.sh x.png i4`, window
  reported 7 s after the launch (helper 5 s before). The game's `/proc/<pid>/mountinfo` has 19 lines, none cifs/fuse//mnt//media.
- (d) V `game.sh stop i4` (SIGTERM by marker, `wineserver -k`, reaper) and `Game.Shutdown` + `WineserverKill` from the host
  find and end every process: the pids are visible and the sockets are shared. No Game.exe/helper/Xvfb of i4 left over.
- (e) V (write path): a program inside the sandbox writes into the real `Saved Games/Diablo II` of the prefix and the file is
  on the host; Wine's `user.reg`/`system.reg` in the real prefix were rewritten by the sandboxed wineserver at shutdown. U: an
  actual in-game Save and Exit (needs a character; i4's Soaky was off limits for this task), no reason to differ.
- Earlier recipe (`unshare -Urm` + `mount -t tmpfs none /mnt/media`) is superseded; it hid only what it named.

## Failure modes
- bwrap missing or userns off: auto mode logs `running Wine WITHOUT the mount sandbox` and launches as before (hung-mount
  guard active); forced mode fails the launch with `D2_SANDBOX=1 but sandbox off: ...`.
- Game starts, no window, `game.out` shows `bwrap: ...`: a path of the allowlist cannot be bound (the message names it);
  most likely a changed Proton location (`ProtonScript` dir is bound automatically) or a read-only `/tmp`.
- A program needs a path not listed (installer on a share, a different Proton): it fails with a plain file-not-found inside
  Wine; add the path as in "How to add a path".
- The first launch after a Proton upgrade works unchanged (the runner dir is derived from the resolved `ProtonScript`).
- i0 not sandboxed by default: a hung mount can still stall it. U for D2_SANDBOX=1 on i0 (X authority and the user's desktop
  session variables were designed but not tested; do not test it on the user's game without asking).
- Caches under `$HOME` (`~/.cache/mesa_shader_cache`, DXVK) are lost per launch (home is a tmpfs); no measurable effect (V: 7 s start).
