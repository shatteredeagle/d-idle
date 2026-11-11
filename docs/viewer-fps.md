# Viewer game video at 12.5 to 25 fps (card viewerfps1, TASK-293, 2026-11-10)

Tags: V = verified (how is said), S = secondary (read in code or docs, not run), U = unverified.

User requirement (2026-11-10): the d.idle viewer's live game video runs at AT LEAST 50 percent of the game's frame rate
(d2d draws 25 fps in real time, so 12.5 fps minimum) and as close to 25 as we can get, on the sandbox hardware, with the
bot's speed and CPU unharmed (bot pace comes first). Compatibility: `/frame` and `/stream.mjpg` keep working, changes are
additive only (docs/ui-design-system.md).

## 1. Measurements (sandbox container on the worker host, 2026-11-10 21:20 EST, read-only)

The sandbox container runs d2d in REAL TIME (no `--lockstep`), `d2dview -jpeg` with the default `-fps 6`, and is limited
to ONE CPU (`NanoCpus` 1e9, `nproc` 1; V, docker inspect and docker exec). Somebody was watching during the measurement.
Method: `docker top` CPU, `/proc/<pid>/stat` utime+stime before and after a 30 s window, and the mtime of d2dview's
`shot.png` polled every 10 ms (each new mtime is one sampled frame). `/stream.mjpg` and `/api/state` need the token
(401 without), so the stream itself was not read.

| what | value | tag |
|---|---|---|
| delivered sample rate | 4.3 fps (130 frames in 29.8 s; gap median 227 ms, p90 289 ms, max 390 ms) | V |
| screenshot verb wall time | about 57 ms (median gap 227 ms = 4 x verb time, since 4 x verb > the 167 ms of `-fps 6`) | V (derived) |
| PNG per frame | 450 KB average (800x600 RGBA, zlib level 1, filter None) | V |
| d2d CPU | 30.5 percent of a core (window), 30 percent average over 7 min | V |
| d2dview CPU | 12.1 percent (window), 7.1 percent over 28 min | V |
| d2bot CPU | 4.2 percent (window), 14 percent over 28 min | V |
| zlib level 1 of one 800x600 frame | 13.4 ms on an idle worker-host core (python zlib, the same library d2d links) | V |
| d2dview old `-jpeg` path (PNG decode + generic JPEG q72) | 25.9 ms CPU per frame, 78 KB | V (BenchmarkFramePath, real frame) |
| PNG decode + JPEG via the RGBA fast path (this card) | 18.6 ms, 78 KB | V |
| raw RGBA to JPEG q72, full / half size | 9.3 ms 78 KB / 2.8 ms 18 KB | V |
| raw RGBA to JPEG q50, full / half size | 8.4 ms 52 KB / 2.5 ms 12 KB | V |
| memcpy of one frame (1.9 MB) | 0.09 ms | V (python on the worker host) |

Benchmarks: `D2DVIEW_BENCH_PNG=<a frame> go test -run X -bench FramePath -benchtime 40x -cpu 1 ./cmd/d2dview` on the
worker host (load average about 7 on 16 cores). The frame used stays in a scratch dir there and is never committed.

Where the time goes (S, fork source of the pinned container ref `lab3-labverbs` f75816c6, `apps/d2d/main.cpp`): in real
time the `screenshot` verb runs `save_png` directly in the control channel handler, which is pumped on d2d's GAME
THREAD. So every sample stalls the game loop for the PNG encode plus the file write (about 57 ms here, more than one
40 ms game frame), and the tap holds its upstream lock for that time, so the bot's own commands queue behind it. At
4.3 fps that is about 25 percent of the game thread's wall time. Raising `-fps` or softening the back-off on this path
would buy frames with bot pace, which the requirement forbids. The PNG path cannot reach 12.5 fps safely: 12.5 x 57 ms
is 71 percent of the game thread.

## 2. Design: a frame feed in the fork (no PNG, no per-frame request)

Fork verb `framefeed <abs path> <every 1..250> <lease ms 100..60000>` / `framefeed off` (branch `d2d.lab3-framefeed`
on the private fork remote, on top of the pinned container ref). While a lease is live, d2d copies every `every`-th
DRAWN frame into a two-slot file and bumps a sequence number; it never draws a frame for the feed, never changes game
state, and stops by itself when the viewer stops renewing (a dead viewer costs nothing after 3 s). File layout (the
contract, also in `cmd/d2dview/ring.go` and the fork's `components/screenshot/framefeed.hpp`), little-endian:

```
0 "D2FF"  4 u32 version 1  8 u32 width  12 u32 height  16 u32 stride  20 u32 format 1 (RGBA8)
24 u64 seq (stored last, release)  32 u64 frame counter  40 u64 game ms  48..63 reserved
slot 0 at 64, slot 1 at 64 + stride*height; frame seq lives in slot seq % 2
```

d2dview (`-feed /dev/shm/d2dview/frames.ring`) renews the lease once a second while someone watches, reads the newest
slot with `pread` (keeps it only if seq did not move during the read), encodes JPEG itself (RGBA fast path, optional
`-frame-half`), and serves the same `/frame` and `/stream.mjpg`. The tap allows the verb on the viewer socket only for
exactly that path (`feedVerb`). Fallbacks: a d2d without the verb answers `err unknown verb` and the sampler uses
screenshots for good; a ring that has not moved for 1 s (lockstep without `step n render`, or a restarted d2d) gets one
screenshot (which draws in lockstep) and the ring is reopened.

The file must be on tmpfs (`/dev/shm`): a MAP_SHARED file on a disk would be written back at up to 48 MB/s.

Cost per published frame: d2d one 1.9 MB memcpy on the game thread (about 0.1 ms, V for memcpy speed, U inside d2d);
d2dview one pread (about 0.3 ms, S) plus the JPEG encode (9.3 ms full, 2.8 ms half, V). The back-off now applies to
d2dview's own read+encode time (it competes for the same single CPU), default 4, so the video never takes more than
about 20 percent of a core.

Expected (U until the coordinator's A/B on the sandbox): full size q72 at `-fps 25`: encode 9.3 ms x back-off 4 = 37 ms
gap, about 25 fps on an idle core, about 20 fps when the encode slows to 12 ms under load; d2dview about 25 percent of a
core for video (today 12 percent for 4.3 fps). Half size: 25 fps at about 7 percent. d2d loses the 25 percent game-thread
stall and about 6 to 9 points of CPU (4.3 PNG encodes a second gone). Bandwidth per viewer: full q72 2 MB/s (16 Mbit/s),
half q72 450 KB/s, half q50 290 KB/s.

Alternatives weighed: `screenshot jpg <path> [q]` (still one request per frame on the game thread, and d2d would need
libjpeg: rejected); encoding in d2d on a worker thread (adds threads and a JPEG dependency to the fork; the encode cost
moves, it does not shrink: rejected for now); WebSocket or fMP4 (H.264 would cut bandwidth 5 to 10x, but needs an
encoder library or ffmpeg process in the image: stage 4 only if MJPEG bandwidth proves the limit).

## 3. Risks

- Bot pace: the feed removes the game-thread PNG encode and the tap-lock hold, so the bot should get FASTER while
  watched. The remaining cost is d2dview's encode on the shared CPU: bounded by `-frame-backoff` (default 4) and zero
  when nobody watches. Check in the A/B: d2bot kills/min and nav click gaps with and without a viewer.
- Memory: the ring is 3.8 MB on `/dev/shm` (docker's default shm is 64 MB) plus one 1.9 MB read buffer in d2dview.
- Determinism and fidelity: the feed copies the framebuffer after drawing and reads nothing that the simulation
  writes; in lockstep it does not force draws (`skip_draw` frames are not offered), so stepping and RNG are untouched
  (S, by code; det.py not run). Not a cheat or a memory write: it changes no game state (ADR 0014 does not apply).
- A torn header read is caught by the seq re-check; a torn pixel copy without a seq change is impossible (the writer
  only touches the slot of seq+1).
- Security: the tap refuses `framefeed` for any other path; the file is mode 0600 in a 0700 dir.

## 4. Stages (PRs)

1. **Stage 1, this branch (d2dview only, works with the current d2d):** flags `-jpeg-quality` (72), `-frame-half`,
   `-frame-backoff` (4, unchanged default), `-feed PATH`; the JPEG path uses the RGBA fast path (25.9 to 18.6 ms CPU per
   frame, V); counters on `GET /api/frames` (last 10 s window: source, fps, parts/s to stream clients, grab ms avg and
   max, encode ms, bytes, size, d2dview CPU percent, knobs) and a log line every 30 s while watched
   (`d2dview: frames screenshot: 4.3 fps over 10s ...`). With the stock d2d this does NOT raise fps (see section 1);
   it lowers d2dview CPU and gives the numbers for the A/B.
2. **Stage 2, fork:** `framefeed` verb on branch `d2d.lab3-framefeed` (commit in section 5), container build from that
   ref (`deploy/d2d-container/context.sh CTX <commit>`), entrypoint flags `-fps 25 -feed /dev/shm/d2dview/frames.ring`
   (keep `-jpeg`, it still covers the screenshot fallback). Target 20 to 25 fps full size.
3. **Stage 3, client and bandwidth:** per-client `/stream.mjpg?half=1&q=50` (encode once per variant, only while a
   client wants it) and the page picking half on narrow viewports or slow links; the `<img>` MJPEG path already drops
   to the newest frame when a client is slow (it always sends `Current`), so no requestAnimationFrame canvas is needed
   unless stage 4 lands. The Electron app and the stream overlay keep using `/stream.mjpg` unchanged.
4. **Stage 4 (only if bandwidth is the limit):** H.264 over WebSocket/fMP4.

## 5. Status and the deploy checklist (coordinator)

- Stage 1: code and offline tests in `cmd/d2dview` (V: `go test -p 1 -count=1 ./cmd/d2dview`, tests `TestRingRoundTrip`,
  `TestRingRejectsBadHeaders`, `TestFeedVerbOnlyForTheRing`, `TestFeedVerbThroughTheViewerSocket` (unknown-verb
  fallback against the fake d2d), `TestGrabRingEncodesNewFramesOnly`, `TestHalveAveragesBlocks`,
  `TestRGBAOfKeepsOpaqueNRGBA`, `TestFrameStatsWindow`). NOT deployed, NOT run against a live d2d.
- Stage 2 fork prototype: branch `d2d.lab3-framefeed`, commit `80cc7adc` on the private fork remote (pushed
  2026-11-10) = the container pin `f75816c6` + one commit (`components/screenshot/framefeed.{hpp,cpp}`, the verb and the
  offer call in `apps/d2d/main.cpp`, `tests/test_framefeed.cpp`). V: d2d, `test_framefeed` and `test_screenshot` build
  on the worker host and both tests pass. The verb was NOT run inside a live d2d by this card (U): no scratch d2d was
  started. First live check: start that d2d headless, `framefeed /dev/shm/x/frames.ring 1 3000`, read the header twice
  1 s apart (seq up by about 25), `framefeed off`.

Deploy and compare (sandbox, coordinator only):
1. Before: `curl -s -H "Authorization: Bearer $T" http://127.0.0.1:8893/api/frames` inside the container (with a viewer
   open) and `docker top` CPU of d2d, d2dview, d2bot for 5 min; d2bot `minute` log lines (kills, clvl) for the same time.
2. Stage 1 image: same, expect the same fps and lower d2dview CPU.
3. Stage 2 image (d2d from `d2d.lab3-framefeed`, `-fps 25 -feed /dev/shm/d2dview/frames.ring`): expect `source
   framefeed`, fps 20 to 25, d2d CPU down, d2dview up to about 25 percent; d2bot kills/min not lower than step 1.
4. If d2dview's share is too high on the single CPU: `-frame-half` or `-jpeg-quality 50`, or `-fps 15`.

## 6. Deploy and A/B on the sandbox (card viewerfps2, TASK-299, 2026-11-11 EST)

Tags: V = measured on the sandbox container. KEPT: the frame feed stays deployed.

What was deployed (V): image `d2dbot:devp2` = `context.sh` with fork commit `80cc7adc` (`d2d.lab3-framefeed`) and
`--build-arg DEV=1`, built on the worker host at low CPU share; the old image `d2dbot:devp1` is kept for rollback.
`entrypoint.sh` now passes `$D2DVIEW_FLAGS` to the tap in tester mode (this card). The sandbox compose got a fourth file
`feed.yml` (`shm_size: 128m`, `D2DVIEW_FLAGS=-fps 25 -feed /dev/shm/d2dview/frames.ring`). The container was recreated
through compose (volume `d2play-data`, game mount, ports and hero untouched; the bot saved and quit normally).
The running dev release in the volume brings its own `bin/entrypoint.sh`, so the new flags only took effect after one
`tools/dev-update.sh --to d2play-20261108` (UPDATED 0.0.261111065511 -> 0.0.261111142718 in 13 s): after any image swap run a
dev-update, otherwise the old release's entrypoint ignores `D2DVIEW_FLAGS`. The dev updater itself still works (V).

| what (full size 800x600, JPEG q72) | BEFORE (stock verb path, `-fps 6`) | AFTER (`framefeed`, `-fps 25`) |
|---|---|---|
| delivered to a stream client | n/a (sampled 5.9 fps, `/api/frames`; 4.3 fps in section 1) | 21.6 fps (3896 parts in 180 s, one client), 1.9 MB/s |
| sampler fps, 10 s windows | 5.9 | mean 21.2 (18 windows; 18.1 to 24.1, one start-up window 10.5) |
| grab / encode per frame | 30.9 ms (max 49) / 17.8 ms | 0.18 ms / 10.2 ms |
| JPEG size | 82 KB | 90 KB |
| d2d CPU (core share) | 24.4 percent | 20.5 percent |
| d2dview CPU | 11.3 percent (while watched, own log) | 21.4 percent |
| d2bot CPU | 10.0 percent | 29.3 percent (fighting, see below) |
| kills per minute (`combat kill` lines) | 5.4 (70 min) | 10.0 (3 min) |
| nav legs per minute | 9.6 (70 min) | 10.3 (3 min) |
| nav stalls per minute (per kill) | 4.4 (0.8) | 10.0 (1.0) |

Method: before = read-only 60 s on the old image (CPU from `/proc/<pid>/stat`, `/api/frames`, d2dview log line) plus the
bot log of the previous 70 min; after = a 180 s run of `ab.sh` (a perl client on `/stream.mjpg` counting parts, `/api/frames`
every 10 s, CPU deltas, bot log of the same window). The bot pace is NOT a controlled comparison: the old window was mostly
town chores, the new one a fight stretch, so kills, d2bot CPU and stalls are higher for that reason (stalls per kill are
close). What it shows: the feed did not slow the bot (kills and legs per minute equal or higher). Container total about
70 percent of its one-core limit. Decision rule (>= 12.5 fps and pace within 10 percent): met, KEPT; `-frame-half` not needed.
Two things to know: (1) the container exited cleanly and restarted once at 09:32 EST during the first after run
(bot stop in town, save and quit, docker restart policy, no kill or stop event; cause not found, not this card's commands;
the hero came back, the second run is after it); (2) the page's own viewer was not looked at by the card (U), only the
stream, the stats and the watch port (303 to login).

Rollback (the old tap and old d2d, nothing else changes): `sh ~/play-build-20261108/up.sh old` on the worker host (compose
with `D2_IMAGE=d2dbot:devp1`, without `feed.yml`), then one dev-update to settle the release. Files: ops repo
`docs/play-sandbox.md`.

## 7. Frame pacing and the 1-2 fps report (card viewerfps3, TASK-302, 2026-11-11)

User reports: "smoother, but still not smooth", then "looks sub 1-2 FPS". Measured read-only at 15:00Z with
`cmd/d2dview/jitterprobe` (a static probe copied into the container's /tmp, run with the tap token from d2dview's
environment, never printed, removed after; one /stream.mjpg client plus the ring header polled every 1 ms, 60 s).

| what (sandbox, image devp2, `-fps 25 -feed`) | value | tag |
|---|---|---|
| d2d draw/publish rate in real time | 60.0 Hz (not 25): gap p50 16.9 ms, p99 17.5, max 21.2, sd 0.5; game ms advance 1.000 x wall; no lumps | V |
| tap stream to a loopback client | 24.2 fps, 77 KB/part, 1.9 MB/s | V |
| stream arrival gaps | p50 41.0, p90 41.8, p99 62.1, max 83.3 ms; 68 of 1453 over 60 ms, 7 over 80, 0 over 120 | V |
| what the user's page shows | NOT this stream: the tester watch screen (8891) gets its video from d2bot's `watch.TapCapture`, which polled the tap's `/frame` at `DefaultTapFPS` 4 (max 6) and skipped unchanged frames | V by code and the running flags (`D2BOT_WATCH_VIDEO_SOURCE=tap`, no fps set) |

Findings:
1. The "1-2 fps" is the watch-screen path: at most 4 polls a second of `/frame`, each a fresh HTTP request, over the
   user's link; whatever the tap delivers, the page cannot exceed 4 fps (S for the exact 1-2: the 8891 stream needs the
   watch token, not read).
2. The judder on the tap stream: the sampler ran on its own timer (`max(40 ms, 4 x 10.4 ms encode)` = about 41.6 ms)
   against a 16.7 ms source, so consecutive frames were 33 or 50 ms of game time apart (2 or 3 draws), in an aliasing
   pattern; plus up to 20 ms extra from the stream loop's sleep-poll (S, arithmetic from the measured rates).
3. Not causes (V): the ring (no missed publishes, no stalls), d2d's draw cadence, the 14:32 restart (a tuning agent)
   and the 14:53 event (a normal chicken "Save and Exit": the managed bot restarts d2d, the socket is recreated; the
   feed lease is renewed every second, so the ring came back by itself, as the 24 fps at 15:00 shows).

Fixes (this card; d2bot and d2dview only, a dev-update deploys them, no image rebuild):
- `watch.TapCapture` relays the tap's `/stream.mjpg` (one long request, every part as it arrives, latest wins when the
  page lags; `--watch-video-fps` is now an optional cap of 1 to 30; 15 s without a part ends the stream and the page
  reconnects). A tap without `/stream.mjpg` (404) is still polled on `/frame` at 4 fps. Tests: `TestTapCaptureRelays
  TheStreamAtItsOwnRate`, `TestTapCaptureStreamCapAndIdleEnd` and the old polling tests (fallback).
- d2dview picks d2d's publish divisor so frames are evenly spaced in game time: the smallest whole number of 60 Hz draws
  that fits both `-fps` and the back-off budget of its own encode (25 fps with a 9-10 ms encode and back-off 4: every 3rd
  draw = 20 fps, each 50 game ms). The source period is measured from the ring's game ms; a change needs two lease
  renewals in a row. The sampler checks the ring header every 3 ms (a 64-byte read) and encodes a frame as soon as it
  appears, so the send cadence is the draw cadence.
- `/stream.mjpg` sends each frame the moment it is published (a notify channel instead of the 20 ms sleep), flushes
  its headers at once, sets a 10 s write deadline per part, and adds `X-Frame-Game-Ms` to each part (additive).
- `/api/frames` gains `every`, `sourceHz`, `feedWanted` and `degraded` (ring asked for but the window ran on screenshots
  or under 12.5 fps); the 30 s log line starts with `DEGRADED` then.
- Bug fixed on the way: the stream-parts counter of card viewerfps1 sat in the `/events` handler (it counted SSE events,
  and would have crashed `/events` on a tap started with `-fps 0`); it is in the stream handler now.

After (offline, V on the worker host, `go test -run PacingEndToEnd -v ./cmd/d2dview`: a 60 Hz ring writer, the sandbox
flags, 800x600 frames, one stream client, first quarter skipped): every 3rd draw, 20.0 fps, arrival gaps p50 49.8,
p90 50.8, p99 53.2, max 53.2 ms; game-time gaps exactly 50 ms (p99 50). Live after numbers: U until the coordinator
deploys (checklist below).

For 30 fps (every 2nd draw, 33 ms) the encode budget must reach about 33 ms: `-frame-backoff 3` (about 30 percent of a
core for video instead of 25) or `-frame-half` (2.8 ms encode). Bot pace first: keep 4 unless the A/B shows room.

Deploy check (coordinator): dev-update the sandbox (d2bot and d2dview change, the fork does not), then
1. `jitterprobe` as above (ring plus stream): expect `every 3`, 20 fps, stream gaps p99 under 60 ms, part game-ms gaps
   all 50; `/api/frames` `degraded` false.
2. The watch screen's game video: look at it (or count parts on the 8891 `/stream` with the watch token): 20 fps
   instead of 4. A remote viewer on a slow link gets 1.5 MB/s; if that lags, set `D2BOT_WATCH_VIDEO_FPS=10` (a cap).
3. d2bot CPU: the relay passes JPEG through (no re-encode); expect no more than before.

Next (not done): a canvas player with a 2-3 frame jitter buffer on the watch page (only if the browser's `<img>`
multipart display still looks uneven after this), and per-client half size for phones (stage 3 in section 4).

## 8. No game pointer in the picture (card nocursor, 2026-11-11)

The user saw a white blade-shaped sprite next to the hero in the watch video: the game's own mouse pointer. d2d draws it into the frame itself (the OS pointer is hidden in the game window), at the mouse position, right before the frame is presented; the frame feed and the `screenshot` verb copy that same frame, so the pointer is in every picture the viewer gets, resting where the bot last put it (V: read in the fork's `apps/d2d/main.cpp`, the block "D2's own cursor, drawn into the frame", and by the test below; S that the second, smaller shape in the user's picture is the same sprite a frame later, not looked into).

- **Fork:** verb `cursor off` / `cursor on` (default on). Off skips that one blit: the pointer, the item in hand and the identify pointer are not drawn. The mouse position, the held item, clicks and the game state are untouched, so determinism is not affected. Branch `d2d.lab3-nocursor`, commit `c418e235` on the private fork remote, on top of the frame feed commit `80cc7adc`. V: built on the worker host; the fork test `tests/cursor_verb_d2d.py` (a headless d2d on a throwaway profile with `--no-save`, no character) shows that a frame differs with the pointer off, that a moved pointer does not show in an off frame, and that `cursor on` gives the earlier frame back byte for byte.
- **Viewer:** `d2dview -hide-cursor` (default true) sends `cursor off` when it starts sampling and again every 10 s (the switch lives in the d2d process, so a restarted d2d draws its pointer for at most that long). The verb passes the read-only viewer socket (`cursorVerb` in `mux.go`: exactly `cursor on` or `cursor off`). A d2d without the verb answers "unknown verb": the viewer notes that once, says nothing and never asks again, so older binaries keep working with the pointer drawn. V: `go test ./cmd/d2dview -run TestHideCursorVerb`. U: the two halves together on the sandbox (needs the image below).
- **To ship** (not done by this card; the sandbox was not touched): rebuild the sandbox image from the fork commit, as section 6 did for the frame feed: `FORK_URL=<fork checkout> sh deploy/d2d-container/context.sh <absolute ctx dir> c418e235`, `docker build --cpu-shares 2 --build-arg DEV=1 -t d2dbot:devp3 <ctx>`, point the compose at `d2dbot:devp3` and recreate the container, then one dev update so the container runs a `d2dview` that sends the verb. Roll back by going back to the previous image tag: nothing else depends on the verb.
- The pointer is also gone from the bot's own screenshots while the viewer runs. Nothing reads the pointer from a picture (the bot reads state through the dump verb), so that is harmless; `d2dview -hide-cursor=false` keeps it drawn.

## 8. The picture freezes after a while (card videoresume, TASK-336, 2026-11-11)

User: "hitches now and then, not bad; the video freezes if it sits too long, maybe the tab sleeping".

Likely causes, in order (no live reproduction by this card):
1. NOT a server timeout (V by code): neither watch server (`internal/watch/serve.go`, view and answer listeners) nor
   d2dview sets `ReadTimeout`/`WriteTimeout`/`IdleTimeout`, so nothing cuts a long MJPEG stream after N minutes.
2. The page could not see a dead stream (S by code, the likely freeze): the old panel was a plain `<img src=/stream>`
   that reconnected only on `error`. A multipart stream that ends or stops sending (the relay ends after 15 s without
   a frame, e.g. during a d2d restart; a capture restart; a proxy or Wi-Fi drop that leaves the TCP connection open)
   leaves the last frame on screen with no event, so it froze until a reload.
3. A viewer that vanished without closing (a sleeping phone or laptop, a dropped link) held its slot forever (V:
   `TestVideoDropsAViewerThatStoppedReading` fails without the fix: the handler blocked in Write). Three such and the
   next viewer got 429 and waited in the backoff loop.
4. Hidden tabs: browsers throttle them; on return the old `<img>` sometimes kept a stale frame (U, browser-dependent).

Fixes:
- The panel reads the stream with `fetch` and splits the parts itself (`partParser`), so each frame is an event. A
  watchdog tears the stream down after 3 s without a frame while the tab is visible and reconnects after 1, 2, 4, 8,
  10 s; a badge says "reconnecting..." over the last picture, and after five failures in a row "stream down" with a
  Retry button (it still retries every 10 s). A tab hidden for 10 s closes the stream ("paused"); visibilitychange to
  visible reopens it at once; focus and online revive a stream that has gone quiet; pageshow from the back-forward
  cache reconnects. `window.d2video.reconnect()` (also `state()`, `fps()`) is the hook for the desktop app. With
  `?debug` in the page address the card head shows `stream: N fps`. Browsers without streaming fetch keep the plain
  `<img>` (backoff now 1, 2, 4, 8, 10 s). The `/stream` and `/stream.mjpg` endpoints are unchanged.
- The watch video writes each part under a per-part deadline (`VideoOptions.WriteTimeout`, 10 s): a viewer that stops
  reading is dropped and frees its slot, and the capture stops after the linger; a live stream renews the deadline at
  every part, so it lives as long as it is read (`TestVideoStreamOutlivesTheWriteDeadline`).
- Tests: node `testdata/video_panel.js` (fake clock: stall, backoff, down and Retry, hidden then closed, visible then
  reopened, wake on a stale stream, the part parser on 7-byte chunks), Go `videoresume_test.go`.

Desktop app (not changed here, apps/desktop-electron is another card's): call the hook on resume, e.g. in main.js
`powerMonitor.on('resume', () => gameView.webContents.executeJavaScript('window.d2video && window.d2video.reconnect()').catch(() => {}))`
and the same on the window's `show` and `restore`. The page's own visibility handling already covers most of it.

Deploy check: dev-update (d2bot only). Open the watch page, leave it 20 min, hide the tab 1 min and come back, sleep
the laptop: the picture resumes by itself within a few seconds, the badge shows while it does.
