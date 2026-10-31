> status: demo ran 2026-10-30 16:18-16:33 EDT and was stopped; rerun `tools/d2dview-demo.sh --minutes 15 [--watch]` (prints the URL, http://worker.example:8891/).

# d2dview: watching the sped-up d2d bot (agent D2DVIEW, 2026-10-30)

Tags: V = verified (offline test named, or live log), S = secondary (read in docs/code, not run), U = unverified.

`cmd/d2dview` serves one web page with the real game frame (left) and a top-down map (right): player, trail, monsters
with HP bars, exits, life and mana bars, area name, virtual time, measured speed (virtual s per wall s), xp, gold, dead
monsters seen, and the bot's goal text. Stdlib only, read-only.

## Run it (on the worker host, next to the d2d)

```
# 1. start d2d yourself on UPSTREAM (lockstep, `flock -n /tmp/d2d-lab.lock`, own socket), e.g. as ~/d2d-lab/view/run.sh does
# 2. start the viewer; it creates the bot's socket and its own filtered socket (UPSTREAM.view)
d2dview -addr :8891 -open -upstream $RUN/up.sock -bot-sock $RUN/bot.sock [-bot-api http://127.0.0.1:8471]
# 3. point the bot at the BOT socket instead of the d2d:  d2bot --backend d2d --attach --d2d-attach --d2d-sock $RUN/bot.sock ...
```
Open `http://worker.example:8891/` (port 8891: next to the board 8897 and d2watch 8890, free when checked U; any
port is approved for d2d work). Auth like d2watch: token in `D2DVIEW_TOKEN` or `~/.config/d2dview/token` (sent as
`?t=...`, kept in a cookie, never logged or printed); no token is allowed only on loopback or with `-open` (LAN, a
throwaway offline character; the page shows a game picture). Without `-bot-sock` the viewer just attaches to the d2d
(only possible while no bot holds it).

## Why a tap (the sharing choice)

d2d's control socket takes ONE client; a second gets `err busy` (docs/d2dctl.md, V in d2d-spike). So a viewer cannot dial
next to the bot, and the d2dctl package has no secondary socket or snapshot file. Options weighed: (a) a state file
written by the bot (needs a bot change, not our directory), (b) a tee inside d2dctl.Client (same), (c) a proxy that
owns the only upstream connection and hands out two sockets. (c) needs no change to the bot or the fork.
`Mux` (`cmd/d2dview/mux.go`): every command is one locked round trip upstream, replies never interleave. The bot socket
forwards everything; the viewer socket forwards only a whitelist (`readOnly`: state, monsters, npcs, ground, info,
clock, dump [items|player|ui], debug players/level/warps/portals/presets/objects/corpses/layout/nearby/grid N, and
`screenshot` to a path under the private shots dir). V offline `TestViewerSharesTheSocketReadOnly`: `step`, `cmd move`,
`click`, `quit`, `debug goto`, `key` and a screenshot outside the dir are refused with `err view: read-only ...` and
never reach d2d, while the bot is connected and stepping.
- The bot's `Prefetch` (8 lines in one write) is split into one round trip per line inside the tap (a unix-socket hop,
  about tens of microseconds each, S: lab5 verbs cost 10 to 60 us, docs/d2d-pace.md).
- The viewer waits on the lock, never the other way round beyond one read verb. A `step` reply held by d2d holds the lock;
  the viewer's reads queue behind it with a 3 to 4 s timeout.
- A d2d restart (failsafe ladder, chicken) closes the upstream connection; the next command redials. In attach mode the
  bot does not restart d2d itself (managed mode does, and its socket path is the d2d's own, so it cannot go through
  the tap): U, the viewer is for a run on a d2d you started. Fixing this needs a bot flag `--d2d-dial PATH` (dial a
  different path than d2d listens on), not done (not our directory).
- Pace: the viewer polls only while a page is open (`Touch`, 5 s), 15 Hz by default, one pipelined batch
  (`state`, `monsters`, `debug level`, `clock`) plus the grid when `d2dworld` says it is due (new level, or 60 subtiles
  moved and 3 s passed) and the bot API once a second (`GET /api/v1/status`, no token sent). Idle: zero cost.

## Pixels (phase 2)

Source: the fork's `screenshot <abs path>` verb, which lab5 already has (docs/d2d-fork.md lab5: "frames never render
except during `step n render`, or when a `screenshot` is requested, the reply is deferred until the next frame has been
drawn"). The viewer calls it through its own socket into a private dir, passes d2d's PNG through untouched (no CPU here; `-jpeg` re-encodes to q72, about 25 ms per frame) and serves it as
`/frame` and as `/stream.mjpg` (multipart, shows in an `<img>`; parts are PNG unless `-jpeg`). Only while somebody watches. No fork change is needed for
this. The d2d runs `--headless --no-video`; whether lab5 draws a real picture in that mode is checked live (see Live).

- **Fast mode (default)**: sampled frames at up to `-fps 6` per wall second. The sampler waits at least 4x the duration
  of the last screenshot, so the d2d spends at most about 20 percent of its time on pictures. The label under the
  frame reads `SAMPLED 1 frame per N virtual ticks`, N = bot ticks stepped through the tap between two frames (measured
  at the tap, V by `TestFramesGrab`). At 50x and 6 fps one frame covers about 8 virtual seconds (200 ticks): it looks
  like a slideshow, the map trail is the smooth view.
- **Watch mode** (button, `POST /api/mode?m=watch[&speed=8]`, `-watch-speed`): the tap rewrites the bot's `step n`
  into `step n render` (d2d draws every stepped frame, lab5 V in fork docs) and sleeps after each step so virtual time
  runs at most `speed` x real time (default 8; 4 to 10 is the useful range). The screenshots then show a frame at most
  one step old. V offline `TestWatchModeRendersAndPaces` (rewrite and pacing). Cost: software rendering is 10 to 15 ms per
  frame (docs/d2d-pace.md profile, S); with ~3 ticks per bot step that is about 30 ms of draw per 120 virtual ms,
  so watch mode cannot exceed about 4x on its own, and the pacing sleep only slows it further. Smoothness is bounded by
  the sample rate: 6 fps at 8x = one frame per 0.17 virtual s (about 4 ticks), motion looks continuous.
  The bot's own pacing knobs (`Pacer.Chunk`, 50 ticks per step call, and the Sleep loop in `internal/d2dinput/pacer.go`)
  are not touched: the tap only edits the `step` line and delays the reply, so the bot sees a slower d2d (its virtual
  clock still follows `game_ms`). The bot's `D2DSPEED` setting is a past work item, not a runtime knob (U: no such env
  var in the code, grep).
- Switching modes mid-run is safe (only the next `step` line changes). `step n render` replies are assumed to carry the
  same clock line as `step n` (U; the tap passes them through untouched, d2dctl parses the `ok` rest).

### Fork requests (none required; two would make it better)
1. (nice) `screenshot` verb variant `screenshot jpg <path> [quality]` or an in-memory `frame` verb (reply = size, bytes
   in a shared memory file): the PNG encode inside d2d is the dominant per-frame cost (U, not measured).
2. (nice) a `render <0|1>` runtime switch ("draw every frame while stepping", like `step n render` but sticky), so watch
   mode does not need to rewrite the bot's lines. Not needed now.

## d2watch tile (done by card d2dview2; code V by node --check and d2watch tests, NOT deployed, NOT seen in a browser)
`cmd/d2watch/web/app.js` (about 30 lines at the end plus two edits): a tile `d2d bot` whose `<img>` is `http://<host>:8891/stream.mjpg`
straight from d2dview, so d2watch spends no CPU or bandwidth on it. Shown only while d2dview answers (opaque no-cors GET of
`/api/state` every 5 s); the stream opens only while the tile is wanted (not paused, tab visible). No token support: use `-open`.
d2watch itself idles at about 0.8 percent CPU (V, 4.5 h average of the running service); the tile adds nothing server-side
(S). Deploy with `tools/watch.sh start`. Original request text: Cheapest option today: link the d2dview page from the d2watch
header. d2watch's own capture path (ImageMagick `import` per frame) is the CPU cost seen before; d2dview needs no X
capture, d2d writes the frame itself.

## Tests
`go test -count=1 -p 1 ./cmd/d2dview`: tap read-only filter, watch rewrite and pacing, snapshot from the fake d2d, frame
sampler (fake plus a hand-made PNG). The screenshot page test needs a headless browser: none is installed on the worker host
(`which chromium google-chrome chrome` empty, V), nothing installed; the page was checked by `curl` of the endpoints
only (see Live).

## Live (V, the worker host, own d2d lab5 `--headless --no-video --lockstep`, throwaway Sorceress "Viewy", `~/d2d-lab/view/run2.sh`)
- Whole stack through the tap: `d2dwalk` (Rogue Encampment to Blood Moor and back, 5 legs) and the real `d2bot --backend d2d
  --attach --d2d-attach --goals campaign` (100 s wall, clvl 3, 118 kills, no error) both ran with the bot on the tap's bot
  socket. The viewer served `/events` (SSE), `/map.png` (cave grid with walls, 480x280 subtiles for Blood Moor), real
  800x600 game frames (headless lab5 does draw: Rogue Encampment frame looked right) and the monsters list.
- Speed with the viewer open (SSE at 15 Hz plus the MJPEG stream at 6 fps, JPEG re-encoding on, first build): pace line
  `ratio 24.4` at 60 s wall (virtual 1466 s). docs/d2d-pace.md has 27 to 33x without a viewer on seeds 5 and 11, load
  differs, so the cost is not isolated (U). The d2dview process used 27 s CPU in 104 s (26 percent of a core), mostly the
  JPEG decode/encode of each frame plus 38 MB of stream: therefore the default is now PNG passthrough (`-jpeg` off). The
  d2d process itself used 90 s CPU in 104 s (86 percent, with screenshots). The re-measure with PNG passthrough and the
  no-viewer baseline could not be run: the d2d lab lock was held by other agents' runs (queued, then cancelled). Run
  `CLIENTS= WALL=100s ./run2.sh` vs `CLIENTS=1 ...` (scripts in `~/d2d-lab/view`, not in the repo) when the lock is free.
- Tap overhead, synthetic (`BenchmarkStepThroughTap`, fake d2d answering at once): +12 us per command through the tap.
  At about 230 bot steps per second and 9 commands per step that is about 2.5 percent of a core and under 3 percent of
  step time (S, extrapolated).
- Not tested live: watch mode (offline only), the d2watch tile, a browser rendering of the page (no headless browser on the worker host).

## Demo run (V, card d2dview2, `tools/d2dview-demo.sh`, 2026-10-30)
- `--d2d-dial` (d2bot flag, `Config.Dial`): V live, a managed d2bot (starts d2d lab5 itself) went through the tap for 15 min, including 2 d2d restarts
  (save point and death), no error. First attempt used `--no-save`: every save point wiped the character (clvl 1 loop, d2d restarted every 2 s); the demo
  now saves into its own dir (wiped at each start), as docs/d2d-loop.md found.
- Fast mode, 15 min, fresh Sorceress seed 5: clvl 7, 1275 kills, 2 deaths, virtual 38000 s. Speed per wall minute (virtual s / 60, load average 3 to 5): 56, 44, 38, 38
  with no page; 22, 24 then 29, 29 with SSE (15 Hz) plus MJPEG (6 fps PNG) clients attached by curl for 3 min; 56, 42, 40, 34, 36, 53 after they left. So a
  page open costs about 35 to 40 percent of speed here (U: the content also changes per minute, one run, not the same seed A/B). d2dview CPU: 2:39 over 15 min
  (18 percent of a core, mostly the tap forwarding every bot command, with clients about 25 percent).
- Watch mode, 3 min, clients attached: ratio 2.1 (not the 8x cap: software rendering of every step bounds it, as predicted), d2dview CPU 5 s in 3 min.
