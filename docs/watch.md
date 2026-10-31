# d2watch: view-only web viewer of the game instances (agent WATCH)

`cmd/d2watch` (Go, stdlib, embedded `web/`), `tools/watch.sh`. Runs ON the worker host as a transient user service (`d2watch.service`).
`tools/watch.sh start|stop|status|url`. `url` prints `http://worker.example:8890/?token=...` (token in `~/.config/d2watch/token`
on the worker host, mode 600, random, never logged; the first valid `?token=` sets a SameSite=Strict cookie so images and streams need no token).
Flags: `-addr :8890` (token required unless loopback), `-fps 4`, `-capture-conc 3`, `-capturer import|ffmpeg` (default import), `-conf`, `-token-file`.
Capture and streaming are `internal/viewport` (docs/viewport.md); d2watch only discovers windows and samples metadata.

## What it does
- Discovery every 3 s: `mem.FindGamesFor("")` + `mem.ProcInstance` (same markers as tools/game.sh; unmarked = i0), agent names from
  `tools/instances.conf`, display `:100+N` (i0 = `:1` with the xauth file of the desktop), window id by `xdotool search --name "^Diablo II$"`,
  helper = TCP connect to 127.0.0.1:47012+N. Tiles appear/disappear by themselves.
- Capture ONLY the game window (`import -window <id> -quality 65 jpg:-`), never the root window (i0 is on the user's real desktop).
  Lazy: only while a stream is open or `/frame` was requested in the last 3 s; one capture in flight per instance, at most 4 fps each,
  frames cached and shared by all viewers, global limit of 3 concurrent captures.
- Endpoints (GET/HEAD only, anything else 405; no input path exists): `/` page, `/api/instances`, `/frame/<id>.jpg`, `/stream/<id>.mjpg`.
- Metadata (read-only `/proc/<pid>/mem` via internal/mem + internal/d2: State, Player, Mouse) is sampled (mouse 10 Hz, player 2 Hz) only while
  the API was polled in the last 5 s. Without ptrace the instance degrades to capture-only (`mem: "denied"`). No game locks are taken.
- Page: auto-fit grid, cursor dot + 4 s fading trail from the game's own mouse position, amber flash on a mouse jump > 220 px, red flash on a
  life drop (>= 5 % max life), click to enlarge (other streams are disconnected), Esc back, Pause (drops streams and polling), theme toggle.

## Verified live (the worker host, 2026-10-28)
- ptrace via `worker-sudo run d2lod-bot -- setcap cap_sys_ptrace=ep` works under the user service (mem "ok" for i1..i5).
- Frames are valid JPEGs (about 35 KB at q65, 800x587), MJPEG stream delivers about 4 fps, API shows character/level/area/life/mana/mouse.
- CPU cost of the whole service cgroup (server + import children), the worker host load average about 30: no viewer 0.3 % of a core;
  1 viewer with 5 tiles about 37 %; 5 viewers (25 streams) about 46 %; 1 enlarged tile about 10 %. About 22 ms CPU per capture.
- Page rendered in headless Firefox (5 tiles, dots, labels, bars).

## Gotchas
- The worker host's firewalld blocks 8890 from the LAN until `! pkexec firewall-cmd --permanent --add-port=8890/tcp && pkexec firewall-cmd --reload`
  is run on the worker host (not done by the tools; no worker-sudo use beyond the setcap). Until then use `ssh -L 8890:127.0.0.1:8890 worker`.
- Dead end: the first capture scheduler queued a new capture per tick when one was slow and ran captures back to back; now one in flight per instance.
- The black rectangle in the top-left of each frame is in the game's own Xvfb output (also in `tools/shot.sh`), not an artifact of d2watch.
- Area names are from memory (levels.txt order), UNVERIFIED beyond area 1.

## Video hang with a browser left open (VIEWHANG)

Capture was already one shared loop per instance (`viewport.Stream`), idle-stopped, so open tabs do not multiply
capture cost (V: code read; the worker host d2watch at 18 h uptime: 20 MB RSS, 3 fds, no leaked ffmpeg/import).
Likely cause (U, browser behaviour not reproduced live): a hidden/throttled tab stops draining the MJPEG
response, the 5 s write deadline fires, the server returns cleanly, and the `<img>` fires `load` (not `error`)
and freezes on its last frame; the old client only retried on `error` and never paused on `document.hidden`.
Changes: `cmd/d2watch/stream.go` per-client writer (3 s write deadline, latest-frame-wins via `Stream.Next`,
max 24 viewers -> 503 + Retry-After, one stats log line per client: frames/skipped/KiB/reason); `app.js`
reconnects on `load` and `error` with 1..30 s backoff, drops streams when the tab is hidden and resumes on
`visibilitychange`. Test: `go test -race -count=1 -p 1 ./cmd/d2watch` (fake never-reading client is dropped,
capture stops, cap enforced). Deploy: rebuild d2watch and restart the user unit on the worker host (not done here).

## Idle CPU burn (D2WATCHCPU)

Evidence (V, the worker host read-only, 2026-10-30): the live `d2watch -addr :8890` (transient user unit `d2watch`, Nice=10,
CPUQuota=100%) showed ps pcpu 70.0 over 26 h; per-thread `top -H`: all 20 threads busy, 4 running; 6.1M utime ticks
(self, not children), 17 fds, 15 MB RSS, one Waterfox tab connected (2 sockets: API poll + stream). No import/ffmpeg
children running, so the cost was in-process.
Measured on a scratch copy (`/tmp/d2watch-test`, port 18890, pprof 18891, API polled at 4 Hz plus two MJPEG streams):
0.6-0.8% of a core. So the sampler, API and normal streaming are cheap; the burn needs a state the fresh copy lacks.
Root cause (V by regression test, not by live profile): `serveStream` looped on `Stream.Next`. When an instance
disappears, `registry.discover` calls `Stream.Close`; `Next` then returns "closed" instantly, the loop saw an error,
found a cached frame, and `continue`d: a hot loop for as long as the browser kept that `<img>` connection open.
A second bug hid it: the idle-timeout test used `ctx.Err()` after `cancel()`, which is always non-nil.
Fix: `Stream.Closed()` (viewport), `Close` now wakes waiters, `serveStream` returns on closed and tells a real
timeout via `errors.Is(err, context.DeadlineExceeded)`. Test `TestClosedStreamEndsResponse`.
CPU estimate: ~70% of a core with a stale stream -> ~0.7% (the measured scratch figure). A pprof endpoint is now
available opt-in: `-pprof 127.0.0.1:PORT` (default off).

Redeploy (main agent; not done here): `tools/watch.sh start` does it all (builds, stops the unit, token check,
scp to `~/d2lod-live/watch/d2watch`, `worker-sudo` setcap cap_sys_ptrace, `systemd-run --user --unit=d2watch ...`).
The old binary is kept by hand as `d2watch.prev`. Verify: `tools/watch.sh status`, then after a few minutes with the
board open `ps -o pcpu,cputime -p $(pgrep -x d2watch)` should show low single digits.
