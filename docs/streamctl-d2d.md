# Stream readiness: showing d2d on the Kick stream

> status: readiness note (agent streamview, 2026-10-30). Nothing was started, stopped or configured; no stream key was read.
> Tags: V verified (code read or run offline), S secondary, U unverified.

## Verdict
The stream can be launched again and CAN show the d2d combined page, with no code change in streamctl or stream.sh. The kiosk is just a Chromium on a hidden Xvfb, so any URL works. One caveat: the d2dview that serves the page must be the new build (the /stream page is in this branch; the demo binary running on the worker host at the time of writing predates it, V: `/stream` answered 404 there while `/api/state` answered 200).

## How the stream works (V, tools/stream/scene.sh, stream.sh)
- The "scene" is a web page in a kiosk Chromium on a private Xvfb (`:120` 1280x720 full, `:121` 640x360 tile), systemd user units `d2-scene` / `d2-scene-tile`, CPUQuota 35 %, MemoryMax 700M. stream.sh captures it with x11grab like a game tile and encodes with VAAPI h264 (GPU), 15 fps, 2500 kbit/s, to RTMP; ffmpeg unit CPUQuota 60 %.
- Scene-only (`instances:["scene"]`) uses full mode, 1280x720, no padding, no overlay. With games the scene is a 640x360 tile (the page gets `?tile=1`, which /stream understands).
- Default URL: `https://shatteredeagle.github.io/d-idle/scene.html`. Configured by `--url` of `scene.sh start`, or env `D2S_SCENE_URL` (stream.sh passes it on). If the scene unit is already active, `scene.sh start` keeps its running URL (V: FRESH=0 skips the launch).

## Pointing the kiosk at d2d (not run)
The kiosk runs on the worker host, so `http://127.0.0.1:8891/stream` is local; port 8891 need not be exposed. d2dview must be started with the bot or a d2d run behind it (docs/d2d-view.md); `-addr 127.0.0.1:8891` is enough for the stream, and needs no token on loopback.
1. Start the d2d run and d2dview (as in docs/d2d-view.md).
2. `~/d2lod-live/scene.sh start --mode full --url http://127.0.0.1:8891/stream` (scene-only stream), or for a scene tile next to games `--mode tile` (same URL, `?tile=1` is added).
3. Go live with the usual streamctl call (`POST /api/stream/start {"instances":["scene"]}`), which reuses the running scene unit. Requires approval, see below.
To go back to the public scene: `scene.sh stop all`, then start again without `--url` (or restart the stream).

## When no d2d run is active
- The page itself says "waiting for a d2d run" (state API answers, d2d not attached): safe but dull.
- If d2dview is down the kiosk shows Chromium's error page ON AIR (U: not tried). So switch the scene back to the public page as soon as the run ends: the d2d run script should call `scene.sh stop all` in its cleanup, or only use the d2d scene while the run is watched. A watchdog (restart the scene with the default URL when `http://127.0.0.1:8891/api/pub` fails twice) would close this gap; not built (outside this card's paths).

## Cost
- Capture and encode: encoder is on the GPU (V from the ffmpeg line); Chromium kiosk is capped at 35 % of a core and ffmpeg at 60 % (V, unit limits). Rough total for a scene-only stream: about 0.5 to 1 core (S, estimate).
- d2dview adds: state polls at 15 Hz while the page is open (about 50 us each, docs/d2d-pace.md) and sampled frames at 6 fps (PNG pass-through, no re-encode unless `-jpeg`); the kiosk page keeps both busy all the time because it is always watching. Prefer `-jpeg` for the stream to cut bandwidth (about 25 ms CPU per frame, about 15 % of a core at 6 fps, S).
- I/O pressure: The worker host iowait was about 40 % lately (user report). The stream writes nothing to disk, but the Chromium profile (`~/.cache/d2-scene/prof-*`), d2d run logs and screenshots in the shots dir do; a loaded disk can stall the x11grab pipeline (U). Check `tools/devhealth.sh` and `tools/capacity.sh --add 1` before going live.

## Privacy checklist (the stream is PUBLIC)
The page reads only `/api/pub` (cmd/d2dview/pub.go), never `/api/state`, `/events` or `/api/mode`. It shows: backend badge, speed factor, area, character name and class, level, XP, kills, deaths, goal title and activity, map cells and dots. It must never show: hostnames, IPs, paths, socket names, tokens or keys, raw bot status, monster names, error strings (they can carry socket paths), account data. Free text passes `safeText` (drops anything with a path, URL, address, `@`, token-like word). Unit tests assert the page source holds none of these and that the feed drops them. Before going live also check by eye: the Chromium window shows no address bar (kiosk), the scene shows no Chromium error page, and the character name is a throwaway one.

## Exact command for approval (not run)
```
ssh worker.example '~/d2lod-live/scene.sh start --mode full --url http://127.0.0.1:8891/stream'
curl -s -H "Authorization: Bearer $(ssh worker.example cat ~/.config/streamctl/token)" -X POST \
  -d '{"instances":["scene"]}' http://worker.example:8433/api/stream/start
```
streamctl is installed in dry mode by default (docs/streamctl.md): going live needs `install.sh --live` or `STREAMCTL_DRY=0`, which is the user's call. The user must approve: (1) deploying the new d2dview build, (2) starting the d2d run and d2dview, (3) the two commands above, (4) the live switch (public broadcast), and a decision on the d2d character name shown.
