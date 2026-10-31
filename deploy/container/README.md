# Container spike (WP-CONTAINER)

Goal: answer the headless-container questions of PLAN.md "End goal" without any Blizzard file.
Run everything on a lab worker: `labrun worker -- tools/container-spike.sh` (docker or podman; it labels
and removes what it creates; set `KEEP=1` inside the remote command to keep the image). Nothing here is
deployed anywhere. Tags: V = verified (run on the worker host 2026-10-28, Docker 29.9.2, kernel 7.2.x, Yama ptrace_scope=1,
debian:trixie-slim, Wine from Debian trixie), U = unverified.

## Files
- `Dockerfile`: debian trixie-slim, `wine` + `wine32:i386`, Xvfb, x11-apps (xwd), x11-utils, tini; user uid 1000; `WINEPREFIX=/data/prefix`. Binaries come from `./bin` (cross-compiled by the script, so no Go in the image).
- `entrypoint/main.go` (`d2container`): child subreaper, orphan reaper, SIGTERM handler, managed Xvfb. `run -- CMD` is the entrypoint shape for the real app; `spike`, `hold`, `probe`, `fdprobe` are test modes.
- `d2start.sh`: default command. Creates `/data/token` once (mode 600), then `d2fleet serve -setup -addr 0.0.0.0:8431 -data /data -token-file /data/token`.
- `compose.example.yml`: the wizard service (published `127.0.0.1:8431`, `stop_grace_period: 90s`, `/data` volume, read-only `/src` and `/patch`, no capabilities); the proven direct `d2bot --managed` run is in a comment.

## Wizard run (image from `tools/container-build.sh`; U: not run yet, no docker on the head box)
The image also carries `d2fleet`, `d2doctor` and the web assets (embedded in d2fleet). Default command = the setup wizard; the old
`d2container run -- d2bot ...` / `hold` / `mkchar` commands still work by overriding the command.
```
labrun worker -- tools/container-build.sh
docker volume create d2data
docker run -d --name d2idle --stop-timeout 90 -p 127.0.0.1:8431:8431 \
  --cap-drop ALL --security-opt no-new-privileges:true --shm-size 256m --cpus 2 --memory 3g \
  -v d2data:/data -v "/path/to/Diablo II":/src:ro -v /path/to/patch:/patch:ro d2idle:dev
docker exec d2idle cat /data/token           # then open http://127.0.0.1:8431/setup.html?token=TOKEN
```
`docker compose -f deploy/container/compose.example.yml up -d` is the same. 9-step user checklist: docs/container.md "User test checklist";
offline rehearsal: `tools/container-checklist.sh`; static image check: `go test ./deploy/container/...`.

## Results
| # | Question | Result | Tag |
|---|---|---|---|
| 1 | Plain Wine runs a windows/386 exe in the image | yes: `wine winprobe.exe` incl. first prefix creation 5.8 s, no Wine-mono/gecko needed (`WINEDLLOVERRIDES=mscoree,mshtml=`) | V |
| 2 | Xvfb starts | up in 0.1 s (`-nolisten tcp`, 800x600x24) | V |
| 3 | Subreaper reads a Wine process after the launcher exits, no caps | `open /proc/<sleeper>/mem` OK, "MZ" at 0x400000; user uid 1000, CapEff=0, default seccomp, PID 1 or under tini | V |
| 4 | Sibling process (child of ours, not ancestor of the target), no caps | `permission denied` | V |
| 5 | Same sibling with `--cap-add SYS_PTRACE` | still denied for the uid-1000 image user: Docker adds the cap to the bounding/permitted set only, CapEff stays 0 for non-root. OK only as root (`--user 0 --cap-add SYS_PTRACE`, CapEff has bit 19). Root without the cap: denied | V |
| 6 | Sibling via fd passing (`internal/memfd`, SCM_RIGHTS over a unix socket) | OK with no caps, uid 1000, `--network none` | V |
| 7 | Go entrypoint as PID 1 vs tini | both work: subreaper set (`prctl`), `docker --init` makes d2container pid 7 under `docker-init`, reads/reaping identical. As PID 1 `docker stop` takes 0.2 s (SIGTERM handled); a plain `sleep` PID 1 takes 10.3 s (SIGKILL after timeout) | V |
| 8 | Zombie reaping | SIGKILL of the orphaned sleeper leaves 1 zombie, 0 after the 2 s `launch.Reap` tick | V |
| 9 | Window capture | `wine notepad` appears as an X window ("Untitled - Notepad" in `xwininfo -root -tree`), `xwd -root` = 1.9 MB with 308 distinct pixel values (blank would be 1). `import` (ImageMagick) not installed, `xwd` is enough | V |
| 10 | Image size | 1.16 GB (275 packages; Debian's `wine32` hard-depends on ffmpeg/gstreamer/llvm-mesa libs). Build 115 s from cold on a loaded worker host, 245 MB download. | V |
| 11 | Idle cost (Xvfb + wineserver + services/plugplay/rpcss/winedevice + one sleeping probe) | 125 MB PSS in total (winedevice 38, Xvfb 13, services 13, plugplay 13, entrypoint 12, probe 11, rpcss 9, svchost 7, wineserver 5), 0.2 % of one core over 20 s. Xvfb alone: 10 MiB, 0 % CPU (`docker stats`). The game itself is not included (U) | V |
| 13 | Real D2 1.14d on plain Wine, Xvfb, read-only game mount, `--cap-drop ALL`, `--network none`, `d2bot --managed` | works: char created in the container (25 s), in game 14 s after `docker run`, 20 min run 63 kills / 0 deaths, CapEff=0 with Yama 1, container 102 % CPU (one core) and 212 to 289 MiB, `docker stop` 4.1 s; details and the Wine-vs-Proton difference list in docs/container.md | V |
| 14 | Image with d2bot, d2input.exe and the Proton shim (`d2idle:dev`) | 1.18 GB (+20 MB over the spike) | V |
| 12 | No network at runtime | everything above ran with `--network none`; only the build needs the network | V |

## Decisions that follow
- The app must be the subreaper and the reader (docs/launch.md 1); no `SYS_PTRACE` is needed. Other processes in the container (sentinel, watcher) get memory through `internal/memfd` fd passing (#6), which also works without caps. Do NOT rely on `cap_add: SYS_PTRACE` for a non-root image user (#5).
- Run the Go app as PID 1 with a SIGTERM handler (`d2container run` shows the shape: reaper tick, SIGTERM -> app, `wineserver -k`), or under `--init`. Both verified.
- Plain Wine, not Proton, in the image. Reason: Proton needs the Steam runtime container for its 32-bit libs and is not packaged; I did not try to run GE-Proton in the image (U). Plain Wine from Debian ran the 386 exe fine and, as verified later (docs/container.md), the real D2 1.14d too; no Proton needed. The Wine prefix is created on first start (6 s) and lives on the `/data` volume.
- Disk: 1.16 GB image is dominated by Debian's `wine32` dependencies; a smaller image needs a hand-picked WineHQ/Kron4ek wine build fetched at setup (U) or `--no-install-recommends` is already used.
- `/data` must be owned by the container user; Wine refuses a prefix dir owned by someone else.

## Not verified
Proton/GE inside the image (not needed), podman, a NAS/LXC host (the head box is an unprivileged LXC; Yama is host-wide), Yama scope 2/3 hosts, `/dev/shm` sizing for the real game, idle cost with the game running, multi-instance per container, noVNC/Xpra for the setup wizard.

## Decision (main agent, after the spike): one container = one bot HOST with N instance slots
The spike recommended one container per bot. That breaks party play: adding or removing a bot would mean creating or
destroying containers (Docker socket access for the app) and party members need to reach each other (the host bot's game
listens on TCP 4000), which conflicts with `network_mode: none`. Default deployment instead:
- One container per machine/pool. The app (d2container/d2bot) is the subreaper and starts and stops instances at runtime
  through `launch.Life` (own prefix, own Xvfb display, own helper port per instance), so adding or removing a bot is an app
  call, not a container operation. No Docker socket, no extra capabilities.
- Party traffic stays on loopback inside the container (bots join 127.0.0.1); no network is needed, `network_mode: none` holds.
- Several containers are for scaling across machines only; a cross-container party then needs a shared network (later).
- Trade-offs: CPU and memory limits are per container, not per bot (about one core per instance, see docs/multi.md); a crash
  can touch siblings, which the per-instance sentinel restart already covers. Work package: a fleet supervisor (admission
  control, add/remove API, docs/archive/review/design-gaps.md D8).
