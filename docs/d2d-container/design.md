# d2d + bot container: design (agent D2DCONTAINER, 2026-10-30)

Status: DESIGN AND DRAFTS ONLY. Nothing is built, deployed, pushed or changed on the NAS. Drafts live in this directory with a
`.draft` suffix so nothing builds by accident (README.md here says how to test). Tags: V = read or measured by me this session
(source or command named), S = secondary (a doc of ours, not re-checked), U = unverified or from memory.
Target: classic 1.14d data through the d2d re-implementation (GPL-3.0), not D2R.

Context: docs/decisions.md 2026-10-29 (container is the product, native worker host is the dev path), 2026-10-30 (likely move to d2d; Wine container
work PAUSED, the Wine `d2idle` container on the NAS keeps running as is; fork lives in private forge repo `<owner>/d2d-fork`).
docs/d2d-backend.md section 5 phase 4 is the plan item this file expands. User update (relayed): ports may be added as needed; each is listed in section 7.

## 1. What goes into the image

Three stages. No Wine, no Xvfb, no X11: d2d `--headless` runs on SDL's dummy video driver (V `apps/d2d/main.cpp` ~1879
`SDL_SetHint(SDL_HINT_VIDEO_DRIVER, "dummy")`), audio is OpenAL's null device (S docs/d2d-fork.md section 2), and the control channel is an AF_UNIX
socket (V `docs/control_channel.md` line 3), i.e. no network port by itself.

| Stage | Base | Output |
|---|---|---|
| `d2d-build` | `ubuntu:26.04` (upstream CI uses ubuntu-26.04 + `build-essential`, V `.github/workflows/ci.yml` lines 44, 81; apt there offers g++ 15.2, cmake 4.2, SDL3 3.4.2, openal 1.25.1, libstorm9 9.30, V `apt-cache policy` in a container on the worker host) | `/out/d2d` (about 13.5 MB, V `ls` of the worker host build) |
| `go-build` | a `golang` image matching AGENTS.md (Go 1.27.3; the image tag is U and may not exist; alternative: today's pattern of cross-compiling on the host into `bin/`, V deploy/container/Dockerfile) | `d2bot`, `d2fleet`, `d2doctor`, `d2soak`, `d2container` |
| runtime | `ubuntu:26.04` (same libc and libstdc++ as the build stage, no static link needed) | the final image |

Why Ubuntu 26.04, not Debian trixie: trixie offers only g++-14 (V apt on the worker host) and the engine is C++26 (`CMAKE_CXX_STANDARD 26`, V top CMakeLists.txt).
Whether g++-14 builds it is untested (U). The worker host built it with GCC 16.2.1 (V `c++ --version`). "g++ 15 on 26.04" matches upstream CI but nobody here has built it (U): the first test build proves it.

### 1.1 Everything the build pulls from outside (V: read from the CMake files of fork commit c36b77b)

| # | Dependency | How CMake gets it | Pin | Vendor / offline |
|---|---|---|---|---|
| 1 | Fork source | build context | `c36b77b13388548deb8e2651ab4e199a59d2dbe3` (<owner>/d2d-fork main, 2 commits on upstream `e36922e`, V `git log` on the worker host) | `git archive` at the commit into the context plus a `VERSION` file (top CMake reads `VERSION` when there is no `.git`, V). Never clone with credentials inside the Dockerfile. |
| 2 | CLI11 v2.6.2 | `FetchContent`, github.com/CLIUtils/CLI11 (V CMakeLists.txt 105-110) | commit `37bb6edc5317e99af72ef48405e65d9ca5218861` (V the worker host `_deps`) | tarball into `vendor/cli11`; `-DFETCHCONTENT_SOURCE_DIR_CLI11=/vendor/cli11` |
| 3 | sentry-native 0.17.3 | `FetchContent`, shallow, submodules `external/crashpad` and `external/third_party/lss`, recursive (V apps/d2d/CMakeLists.txt 27-35) | commit `fb2a04d4a9ca78d83c9cc5d10e33cfac79eed77f`; crashpad `493151b224a6536bcedc3467d042863a063addc4`; lss `ed31caa6...` (v2024.02.01); crashpad's own submodules: mini_chromium `2f5c168fa462cf3836c4a7801b4f4955496d975d`, zlib `fef58692c1d7bec94c4ed3d030a45a1832a9615d`, lss `9719c1e1e676814c456b55f5f070eabad6709d31` (V `git submodule status`) | clone once online with submodules (crashpad needs a second-level recurse), check out those hashes, tar to `vendor/sentry-native`, `-DFETCHCONTENT_SOURCE_DIR_SENTRY_NATIVE=...`. With a SOURCE_DIR CMake does no git work, so submodules must be populated beforehand (U, not tried). Cheaper: a small fork patch making Sentry optional (U, about 30 lines; Sentry is about 300 of 430 translation units per upstream's coverity.yml comment, V), which also drops crash-report code from a product image. |
| 4 | FFmpeg 9.0.2 (Bink-only, static) | `ExternalProject` downloads `https://ffmpeg.org/releases/ffmpeg-9.0.2.tar.xz` (V apps/d2d/CMakeLists.txt 140) | SHA-256 `8c3850283eb25fa026482078a04051e0be17347b09ef81a0849bec15a96e002e` (V) | download once, verify hash, extract into `vendor/`, `-DD2D_FFMPEG_SOURCE_DIR=/vendor/ffmpeg-9.0.2` (documented switch, V). Alternative `-DD2D_BUNDLED_FFMPEG=OFF` needs system libav* dev and runtime packages (not chosen, U). |
| 5 | StormLib | system `libstorm-dev` if found, else `FetchContent` v9.30 (V components/mpq/CMakeLists.txt) | Ubuntu 26.04 ships 9.30 (V apt) | apt `libstorm-dev` / `libstorm9`; no download. Fallback: vendor v9.30 (commit `86f9b99ffe4d3417dad16d00541cf6f2e3d7bf79`, V), `-DD2D_BUNDLED_STORMLIB=ON -DFETCHCONTENT_SOURCE_DIR_STORMLIB=...` |
| 6 | Qt 6.5+ | `find_package(Qt6 ... REQUIRED)` in `apps/launcher/CMakeLists.txt`, reached unconditionally from `apps/CMakeLists.txt` (V) | n/a | The launcher is unwanted but configure fails without Qt. Draft: install `qt6-base-dev libgl1-mesa-dev` in the BUILD stage only (as CI does, V) and build only `--target d2d`. Alternative: a fork patch skipping the launcher (U). |
| 7 | apt | Ubuntu archive | floats | build: `build-essential pkg-config cmake ninja-build git ca-certificates libcurl4-openssl-dev libsdl3-dev libopenal-dev zlib1g-dev libbz2-dev libstorm-dev qt6-base-dev libgl1-mesa-dev` (V = CI list minus clang-tidy). Runtime: `libsdl3-0 libopenal1 libcurl4t64 libstorm9 zlib1g libbz2-1.0 tini curl ca-certificates` (names V from apt-cache; set derived from `ldd` of the worker host binary: SDL3, openal, z, bz2, curl, stdc++, V). For repeatability use an Ubuntu snapshot date (snapshot.ubuntu.com, U) or pre-build and tag a `d2d-build-base` image once. |
| 8 | Go modules | none (standard library only, AGENTS.md) | | nothing |

Nothing else is fetched: only two `FetchContent_Declare` plus one `ExternalProject_Add` URL exist outside the Flatpak/Windows branches (V grep of all CMakeLists).

Build options in the draft: `-DCMAKE_BUILD_TYPE=Release -DD2D_WERROR=OFF -DD2D_BUILD_TESTS=OFF -DD2D_BUILD_TOOLS=OFF -DD2D_CHECK_HEADERS=OFF -DD2D_SENTRY_DSN=`
(Release and WERROR=OFF are the spike's flags, V d2d-fork.md section 1; the rest are my additions, U until the first build; the DSN cache option exists, V, and blanking it keeps upstream's Sentry endpoint out of our binary).
Build time: 8 min at -j4 for the engine on a loaded worker host (V d2d-fork.md); expect 15 to 25 min from scratch with Qt and FFmpeg (U).

## 2. Run stage and processes

Non-root uid 1000, no capabilities, tini as PID 1:

```
/opt/d2d/d2d                      fork binary
/opt/d2lod/{d2fleet,d2bot,d2doctor,d2soak,d2container}   our Go binaries (separate layer and directory)
/usr/share/licenses/d2d/          GPL text, NOTICE, SOURCE.txt (gpl.md)
/game   (volume, ro)              the user's MPQs
/data   (volume, rw)              token, fleet.json, slots, saves, logs, soak runs
/run/d2d (tmpfs, 0700)            AF_UNIX control sockets, one per slot
```

Who starts d2d: the bot. `internal/d2dinput.Process` already launches it (`D2D_BIN`, `D2D_DATA`, `Sock`, config/data dirs become XDG dirs; args `--data X --headless --devctl SOCK --no-video [--seed N]`,
V internal/d2dinput/lifecycle.go) and keeps the socket inside its own tree. The container sets `D2D_BIN=/opt/d2d/d2d`, `D2D_DATA=/game` and puts each slot's config dir under `/data/slots/<id>/d2d`
(saves in `$XDG_CONFIG_HOME/d2d/save/`, log `d2d.log` beside them, V listing of `~/d2d-lab/home/config/d2d`). Also `SDL_VIDEODRIVER=dummy`, `ALSOFT_DRIVERS=null` (U name), `SENTRY_DSN=` empty, and never
`--send-crash-reports` (default off, V main.cpp 1643).

Not wired yet (S docs/d2d-backend.md phases; V `ls cmd`): there is no `d2bot` entry that drives d2d end to end (only `internal/d2dinput`, `d2dsrc`, `d2dworld`, probes `cmd/d2dwalk`, `cmd/d2dsnap`), and
`d2fleet`'s wizard has Wine-only steps (prefix, patch, import-game, S docs/nas-container.md). The container is blocked on those; see risks.md.

## 3. Assets (the game data)

d2d reads the user's MPQs through `--data`. The spike copy: 9 files, 1.4 GB, no video MPQs: d2data 257 MB, d2exp 239, d2char 254, Patch_D2 5.5, d2sfx 47, d2speech 156, d2music 333, d2xmusic 53, d2xtalk 58
(V `du` on the worker host; runs with `--no-video`). The install check requires d2data and d2exp and looks for d2char/d2sfx/d2speech/d2music/d2video (V components/install/install.cpp 433-463). Which are strictly needed headless
is untested (U: try d2data, d2exp, d2char, Patch_D2 only, about 760 MB).

Rules, no exceptions:
- MPQs are user-supplied, mounted read-only at `/game`, NEVER in an image layer, the build context (`.dockerignore` excludes `*.mpq`) or the repo.
- `Game.exe` is not needed for single player (only for `--join`, V d2d README), so it is not mounted.
- No copy into `/data` (the Wine wizard copied 1.9 GB); the mount is used in place. d2d writes only to its XDG dirs (V: the spike ran from a read-only copy; U that nothing writes beside the MPQs).
- Source: the user's install. The NAS already holds one staged for the Wine container (`/mnt/user/appdata/d2idle/game`, S nas-container.md); putting the MPQs into `/mnt/user/appdata/d2dbot/game` is a user action, not done here.

| Mount | Mode | Content |
|---|---|---|
| `/game` | ro | MPQs |
| `/data` | rw, uid 1000 | `token`, `fleet.json`, `slots/<id>/{d2d/save, d2d/d2d.log, bot state, logs}`, `soak/` |
| `/run/d2d` | tmpfs | sockets, not a host mount |

Saves are `.d2s` plus `.ma0`/`.key` per character (V the worker host listing); `d2savecheck` reads `.d2s` (S). Back up `/data` as a whole.

## 4. Resources and limits

Measured on the worker host (V docs/d2d-probes.md, d2d-spike.md, d2d-fork.md): d2d RSS 471 MB flat for 20 min idle in town, 564 MB in a level-cycling run, 515 to 754 MB after many first visits (about 100 MB per
first-visited level, a revisit is free); CPU 0.9 to 1.0 core per d2d in real time and in lockstep (it runs flat out when stepping). The Go bot is 40 to 120 MB (V soak24.md line 68).
One slot is about one core plus 0.7 to 1 GB; each extra slot adds about the same (U linear).

- Memory: `--memory=3g` for one slot (3x headroom, U), roughly 1 GB plus 1 GB per slot. The NAS's kernel ignores swap limits (S nas-container.md), so no `--memory-swap`. d2d grows with each new level, so a long soak must restart the d2d child periodically
  (V d2d-fork.md section 7); the lifecycle already has `Restart` (V); the period is U.
- The 4 GB heap guard: `bot.heapLimitMB` defaults to 4096 and sets a GC soft limit 25 percent above (V stallfatal.md line 128, knobs.md line 199). Inside a 3 GB cgroup it would never fire before the kernel OOM-kills the biggest process
  (likely d2d). So in the container set `bot.heapLimitMB` to 512 to 1024 (range 0..8192, V) and let the cgroup be the last net. Same layering as docs/soak24.md section 8 (V: guard first, cgroup net for what it cannot see, `memory.oom.group` 0 so the
  supervisor survives; there `MemoryHigh=10G/MemoryMax=12G` because Wine adds 2 to 3 GB). A cgroup OOM kills one child and the fleet supervisor restarts it (S nas-container.md; exit 137 is restarted, V soak24.md line 143).
- CPU: `--cpus=2` per slot (1.5 minimum). `D2FLEET_MAX_LOAD_FACTOR=-1` stays (containers see host load, V container.md).
- PIDs: NAS adds `--pids-limit 2048` itself (S); the Wine image ran about 190 PIDs, d2d needs far fewer (U).
- No `--shm-size`, no Xvfb. `--cap-drop=ALL --security-opt=no-new-privileges:true` as today. No `SYS_PTRACE` (no memory reads; that is the point of the backend).
- `--read-only` root fs plus tmpfs for `/tmp` and `/run/d2d` is plausible but untested (U); not in the draft.

## 5. Healthchecks and restart policy

- `HEALTHCHECK` (draft): the dashboard answers on 127.0.0.1:8431; HTTP 200 or 401 both mean up (401 without token is V in nas-container.md).
- Deeper, optional (U): with the token, `GET /api/v1/fleet-admin/slots` (the soak sidecar polls it every 10 s, V nas-container.md) and require a slot with `bot.addr` and advancing game time (`clock`). Needs a small `d2doctor` subcommand, not written.
  d2d's own heartbeat thread aborts a stalled main loop ("MAIN STUCK", V d2d-fork.md), which the bot sees as a dead socket.
- Restart: inside, `d2fleet` restarts a dead bot with backoff 2 s to 2 min, at most 10 times, `CleanGame` first (S nas-container.md). Outside: no Docker `--restart`; the template keeps autostart off until the container soak passes
  (same as the Wine container, V my-d2idle.xml), then the user may set autostart in the NAS UI.
- Stop: SIGTERM, bot saves and quits d2d (V `Lifecycle.SaveExit`). Wine needed 90 s; d2d should need seconds (U). Draft `--stop-timeout 45`, measure then lower.

## 6. Lockstep versus real time

Facts (V docs/d2d-fork.md): `--lockstep` (clock moves only on `step n`; about 120x real time in town, 90x to 370x outdoors; deterministic for one seed and input sequence), `--time-scale x` (faithful only to about x11),
stock real time (25 ticks/s). Bot side: `d2dinput.Pacer` turns `Sleep(d)` into `step(d/40ms)` (V pacer.go; a non-lockstep game does not move on `step`).

Recommendation (my judgement, U until the user agrees):
- Live product (idle bot for days): real time. Speed buys nothing for a bot that plays continuously, the dashboard and viewer assume wall-clock progress, and real time has the only run history (10 to 20 minute runs; no day-long d2d run exists, U).
- Soak and statistics (G1/G2-style verdicts, drops, XP/h): lockstep (or time-scale), because 24 h shrinks to minutes to an hour and repeats exactly. The bot is then the only client and clock owner.
- Selection: `D2D_MODE=realtime|lockstep|timescale:N` (default `realtime`), turned into `--lockstep`/`--time-scale N` through `d2dinput.Process.ExtraArgs` and into `Pacer` versus wall clock in the bot (U: not wired). One mode per container; a separate soak
  container for lockstep. Lockstep still burns one core per slot: "fast" means more game per CPU-hour, not less CPU.
- docs/soak24.md criteria assume wall-clock XP/h and 24 h; lockstep needs a game-time variant of those profiles (open item).

## 7. Ports (user update: ports allowed; each listed with purpose and exposure)

Today's Wine container publishes one: 8431 (V my-d2idle.xml). For d2d the draft proposes:

| Port (container) | Proto | Purpose | Exposure | In template |
|---|---|---|---|---|
| 8431 | tcp | dashboard and setup wizard (existing, token required, V nas-container.md) | LAN only, no WAN forward, no tunnel unless the user adds one later | Port field, host 8431 (see question 1) |
| 8432 | tcp | game viewer: MJPEG/frame stream of the headless game, view only. Today it rides on 8431 (`/api/v1/instances/iN/viewport/stream`, S); a separate port lets a plain `<img>`/OBS/Kick source reach it without the dashboard token flow and keeps video bandwidth off the API port. Source for d2d is the fork's `screenshot` verb (V) or a new frame verb (U) | LAN only; token or one-time URL token required, never public | Port field, optional (default off: leave the host port empty) |
| 8433 | tcp | installer/setup UI if it must stay separate from the dashboard (decisions.md "viewer for the installer needs a second port"): MPQ check, character create, slot add | LAN only, enabled only while setup is not done (U: d2fleet serves it on 8431 today; merge is the cheaper design) | Port field, optional; likely not needed |
| d2d control | AF_UNIX | `/run/d2d/<slot>.sock` between bot and d2d (V) | container-internal, never published, 0700 tmpfs | not a template item |
| 8420+N | tcp | per-slot bot API | loopback inside the container only; the dashboard proxies it (S) | not published |
| stream/RTMP out | tcp out | the Kick stream (memory d2-idle-bot) | outbound only from a separate streamer, not from this container | none |

Recommendation: ship with 8431 only (dashboard plus viewer proxied on it, as today), add 8432 only when a real consumer exists (OBS source), skip 8433 (merge the installer into the dashboard). Every published port is a visible Port field in the NAS template,
so nothing is hidden; unused ones are left empty. Bind to the LAN only: The NAS's bridge publishes on all host addresses, so "LAN only" means no router forward and no Cloudflare tunnel entry, the token as the gate (U: a host-IP restriction in the Port field is possible in the NAS, e.g. `nas.example:8432`,
from memory). On the NAS the host 8431 is held by the Wine `d2idle`; `d2dbot` can use 8431 only after that is stopped, or a different host port (template field): question 1.

## 8. NAS template (UI-visible rule)

Rule (user rule: NAS changes must be UI-visible): every add/change/delete is visible in the NAS UI, no hidden containers or ports. The container is created only from a template `my-d2dbot.xml` in
`/boot/config/plugins/dockerMan/templates-user/` via `tpl-apply.sh` (V deploy/nas/README.md), shows in the Docker tab and the Games Folder View folder, and every setting is a template field. `my-d2dbot.xml.draft` follows my-d2idle.xml (V read):

| Field | Type | Value |
|---|---|---|
| Name / Repository | | `d2dbot` / `d2dbot:dev` (locally built, `docker save | docker load`; update check shows "not available", like d2idle) |
| WebUI | | `http://[IP]:[PORT:8431]/` |
| Web UI | Port tcp | 8431 -> host 8431 (question 1) |
| Viewer | Port tcp, optional | 8432 -> host empty by default |
| Data | Path rw | `/mnt/user/appdata/d2dbot/data` -> `/data` |
| Game data | Path ro | `/mnt/user/appdata/d2dbot/game` -> `/game` (user's MPQs) |
| D2D_MODE | Variable | `realtime` (soak container `lockstep`) |
| D2FLEET_MAX_LOAD_FACTOR | Variable | `-1` |
| D2FLEET_SOAK | Variable | `0` (1 only when the d2d soak driver exists) |
| ExtraParams | | `--memory=3g --cpus=2 --stop-timeout 45 --cap-drop=ALL --security-opt=no-new-privileges:true --init` |
| Autostart | UI toggle | off |

Dropped versus d2idle: `--shm-size`, the `/src` install mount, `/patch`. The compose draft mirrors this for the worker host tests only (NAS uses templates; V deploy/container/compose.example.yml comment).
