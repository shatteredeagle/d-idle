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
