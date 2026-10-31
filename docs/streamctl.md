# streamctl: stream control service

Runs ON the worker host as user unit `d2-streamctl`, port 8433, wraps `tools/stream/stream.sh` (installed at `~/d2lod-live/stream.sh`).
Code: `tools/streamctl/` (Python stdlib only). Tests: `cd tools/streamctl && python3 -m unittest`.

## Security
- The service never sees the Kick URL or key: `stream.sh` reads them from the keyring. A missing entry makes `start` answer 503.
- API token: file `~/.config/streamctl/token` on the worker host (mode 600, created on first run). Send `Authorization: Bearer TOKEN` or `X-Token: TOKEN`. Only `GET /healthz` is open.
- Optional header `X-Client: label` is recorded with the source IP in the event log. Open port 8433 for the LAN only.
- Going live is public. Dry mode (`STREAMCTL_DRY=1`, the installed default) only logs and flips state; it never runs stream.sh.

## API (JSON in and out)
- `GET /api/stream/status` -> `{state: live|off|starting|error, layout, instances:[ids], since, uptimeSeconds, lastError, overlay, scheduler:{enabled,rule,nextCheck}}` plus `maxLiveSeconds`, `stopsAt`, `dry`.
- `GET /api/stream/instances` -> `[{id, character, running}]` (i1..i5; running = Xvfb :101..:105 exists; character from docs/multi.md (Navi, Combi, Towni, Booti, Multi), via instances.conf if present).
- `POST /api/stream/start` `{instances:["i4"], layout:"single|grid2|grid4", overlay:true}` -> status. 409 already live, 400 bad input or instance not running or layout the installed stream.sh cannot do, 503 keyring entries missing.
  An EMPTY body (`{}` or none) starts with the stored selection (400 when none is stored). The start event records `storedSelection`.
- `GET|PUT /api/stream/selection`: the ONE shared selection for StreamDesk, Home Assistant and the API. Stored in `~/.local/state/streamctl/selection.json`.
  GET -> `{instances:[ids], items:[{id,character,running}], layout, layoutAuto, overlay, updatedBy, updatedAt}`. Unknown or stopped ids are kept but have `running:false` in `items`; `layout` is the stored one or, when not set (`layoutAuto:true`), derived from the count (1 single, 2 grid2, 3-4 grid4).
  PUT takes any of `instances` (max 4, ids i1..i5), `layout` (`single|grid2|grid4|null`), `overlay`; merged. Sending `instances` without `layout` resets the layout to automatic. 400 when the layout cannot hold the instances. Changes are logged as `selection` events with the X-Client label; unchanged PUTs are not logged. Explicit starts and the scheduler do not write the selection.
- `POST /api/stream/stop` -> status.
- `GET|PUT /api/stream/settings` `{keepAlive:bool}` -> `{keepAlive, maxLiveSeconds}`; see Keep alive and scene below. Status gains `keepAlive` and `scene` (the scene tile is on air; `instances` lists games only).
- `POST /api/stream/extend` `{seconds}` -> status (pushes the max-live deadline out; extra route).
- `GET|PUT /api/stream/scheduler`. PUT takes any of `enabled`, `maxLiveSeconds` (60..86400, default 14400), `rule` (merged):
  `{instance:"i4", layout:"single", overlay:true, maxDurationSeconds:14400, quietHours:{start:"01:00",end:"08:00"}|null, startAfterHealthySeconds:60, stopAfterUnhealthySeconds:90}`.
  `instance` may be `"auto"` (see below). Stored in `~/.config/streamctl/settings.json`. Default: disabled.
  `PUT {enabled:true, rule:{instance:"auto"}}` is the one call that turns auto streaming on; switching the rule to auto also sets both debounce times to 30 s (override in the same call).
- `GET /healthz` -> `{ok:true}`.

## Safety and scheduler behaviour
- Hard limit: every stream stops itself after `maxLiveSeconds` (default 4 h) unless extended. Logged as `limit_stop`.
- Scheduler (checks every 15 s): when the bot of the rule's instance is healthy (`GET 127.0.0.1:8420/api/v1/instances/<id>/health`: status `running`, no stuckReason, not paused) for `startAfterHealthySeconds`, it starts the stream. It stops it when the bot is unhealthy for `stopAfterUnhealthySeconds`, at quiet hours (local time, start inclusive), or after the rule's max duration (then it waits for the bot to go unhealthy once before restarting). It only stops streams it started itself; manual streams are never touched. A failed auto start is not retried until the bot goes unhealthy.
- Event log `~/.local/state/streamctl/events.jsonl`: one line per start, stop, extend, limit_stop, scheduler_config, auto_start_failed with `ts, event, who` (X-Client or `scheduler`), `ip`, instances and reason.

## Auto rule (`instance: "auto"`)
Every 10 s the scheduler checks each running game (Xvfb up) against the bot API. Health of `iN`: first the d2bot process started for that instance (found in the process list: `d2bot ... --instance iN --addr HOST:PORT`, one port per instance), then the shared `STREAMCTL_BOT_URL` (default 127.0.0.1:8420, `GET /api/v1/instances/iN/health`). A bot counts when status is `running`, no stuckReason, not paused.
The stream carries every bot that has been healthy for `startAfterHealthySeconds` (default 30 in auto), max 4, layout from the count. A bot leaves the set after `stopAfterUnhealthySeconds` (30) unhealthy or when its game is gone. The stream is restarted only when the set actually changes; with no healthy bot it stops. Quiet hours and the rule's max duration apply as before; the scheduler only stops streams it started itself (manual streams are never touched) and never writes the selection.

## Include mode (`rule.include`)
`include: 'bots'` (default) is the behaviour above. `include: 'games'` counts an instance when its Xvfb display is up AND the sentinel (`STREAMCTL_SENTINEL_URL`, default http://127.0.0.1:8895, optional `STREAMCTL_SENTINEL_TOKEN`; `GET /api/v1/sentinel`, fetched once per 10 s pass) reports it in a game: `obs.state == "ingame"`, `obs.procAlive`, a character (`obs.char`), not `obs.hung`, and none of the rules game-frozen, game-hung, game-stopped, proc-gone, window-gone active. This also picks up games run with dev tools (navlive, townlive, oog) without a d2bot. The same debounce (`startAfterHealthySeconds`/`stopAfterUnhealthySeconds`), 4-tile max, scene tile and layout-by-count rules apply. When the sentinel is unreachable the check falls back to "Game.exe with the instance's DISPLAY is alive" (display already required) and logs `sentinel_unreachable` once (`sentinel_back` when it returns).
`PUT /api/stream/scheduler {"enabled":true,"rule":{"instance":"auto","include":"games"}}`; status `scheduler.include` shows the mode. StreamDesk has a Show dropdown, HA has `input_select.d2idle_stream_include`.

## Scene and keep alive
The scene is a status web page (default `https://shatteredeagle.github.io/d-idle/scene.html`, `?tile=1` for the compact 640x360 layout) shown by a kiosk Chromium on a hidden Xvfb and captured by stream.sh like a game tile (`tools/stream/scene.sh start|stop|status [--mode full|tile] [--url U]`; transient user units `d2-scene` :120 1280x720 and `d2-scene-tile` :121 640x360, Restart=always, Nice 10, CPUQuota 35%, MemoryMax 700M, no sound, no cursor in the capture). Only the mode in use runs; stream.sh starts it on `start` and stops it on `stop`.
- `scene` is a source id: `instances:["i1","scene"]` or `scene:true` in `POST /start`; also valid in the stored selection (max 4 tiles in total). Layouts: `single` with only the scene fills the frame (no padding, no overlay); 1 game + scene `grid2`; 2 games + scene `grid3` (two games on top, scene centred below); 3 games + scene `grid4`; 4 games: no scene. Name labels only on game tiles. Layout `grid3` is new. With the scene automatic layout counts every tile.
- `keepAlive` on: `PUT {keepAlive:true}` starts the scene-only stream when nothing is live (the only way the service goes live on its own; never at service start, and an explicit `stop` disarms it until the next enable). The auto scheduler adds the scene as a tile (not counted as a bot, dropped when 4 games are healthy) and falls back to the scene alone when no bot is healthy; a rule for one instance falls back the same way. Manual starts get the scene added when the layout has room and `scene` is not given. Quiet hours still stop the stream. Instead of stopping at `maxLiveSeconds` (or the rule's max duration) the stream restarts itself with the same content (`keepalive_renew` event) unless extended; switching keepAlive off restores the normal limit and stops a scene-only stream that keepAlive started.
- stream.sh layouts: `single|grid2|grid3|grid4`; the scene is always the last tile.

## stream.sh interplay
New stream.sh (has `--layout`): `start --layout L [--no-overlay] ids`, `stop`, `status --json` (state is taken from it). Old stream.sh: only layout single, `start iN` / `stop iN`, state synthesized from `systemctl --user` units `d2-stream-*`.

## Install (on the worker host)
`rsync -a tools/streamctl/ worker.example:/tmp/streamctl/ && ssh worker.example 'bash /tmp/streamctl/install.sh'`
Installs to `~/d2lod-live/streamctl/`, unit `~/.config/systemd/user/d2-streamctl.service` (dry mode), enables it, enables linger so it starts on boot. To go live-capable: `install.sh --live` (or edit the unit's `STREAMCTL_DRY` to 0, `systemctl --user daemon-reload && systemctl --user restart d2-streamctl`).

stream.sh, scene.sh and overlay.py are deployed by `rsync -a tools/stream/{stream.sh,scene.sh,overlay.py} worker.example:d2lod-live/` (keep the three together). `stream.sh test --layout L ... SECONDS` dry-runs any layout to `~/d2lod-status/stream-test-*.mp4` (no network; `D2S_SCENE_URL=http://127.0.0.1:PORT/x.html` points the scene at a local stub page).

## Troubleshooting
- `journalctl --user -u d2-streamctl -n 50`; `systemctl --user status d2-streamctl`.
- 503 on start: store the keyring entries (header of stream.sh), as the desktop session user.
- 400 "instance not running": the game's Xvfb display is not up (`tools/game.sh`, main agent only).
- Status stuck at error: stream unit died; `stream.sh log iN`.
- 401: wrong token; `cat ~/.config/streamctl/token` on the worker host.
- Port unreachable from LAN: The worker host firewall must allow 8433.
