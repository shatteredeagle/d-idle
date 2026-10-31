# Goal manager, Runner contract and HTTP API

Owner: GOALS agent. Code: `internal/goals` (model, manager, persistence, stats), `internal/goals/sim` (simulated runner),
`internal/api` (HTTP + SSE), `web/` (embedded dashboard), `cmd/d2dash` (demo: `go run ./cmd/d2dash`, prints the URL; `-instances N`, default 2, serves simulated instances i1..iN on the instance routes).

Wiring in `cmd/d2bot` (wave 2):

```go
m, _ := goals.New(goals.Options{Runner: bot, Path: stateFile})
srv, _ := api.New(api.Config{Manager: m, Addr: addr, Token: token, Assets: web.FS})
ln, _ := srv.Listen()
go m.Run(ctx)          // pumps bot.Events() and ticks the scheduler every second
srv.Serve(ctx, ln)
```

## The Runner contract (`goals.Runner`)

```go
type Runner interface {
    Start(g Goal) error   // work on g.Type / g.Params, replacing any running goal
    Pause() error         // freeze at the next safe moment
    Resume() error
    Stop() error          // abandon the goal, leave the character safe (town); idempotent
    Status() Status       // character card + activity, consistent snapshot
    Events() <-chan Event // one long-lived channel
}
```

Rules for the bot:

- All methods return quickly (well under 100 ms) and never call the Manager (the Manager holds its lock while calling them).
  Slow work happens in the runner's own goroutine.
- The runner only does the work. **Progress, completion and scheduling belong to the manager**, which counts events:
  `reach_level` completes when `Status().Character.Level` or a `level_up` event reaches the target; `farm_area` counts
  `kill` events and `drop` events matching `params.item` (any drop if empty); `kill_boss` counts `kill` events whose
  `monster` equals `params.boss` (case-insensitive); `collect_item` sums `item.qty` of `drop` events matching the filter;
  `idle` finishes after `params.minutes` of active time. Time limits use time the runner reported `state: running`.
- Emit `kill` (monster, boss, xp), `drop` (item), `death`, `level_up` (level), `area_change` (area, areaId), `potion`
  (potion), `error` (message; `fatal: true` if the bot cannot continue, which stops the whole bot). Leave `time`, `seq`
  and `goalId` empty: the manager stamps them. Never block on the channel; drop the oldest if it is full.
- `Status.State` must be `running` while it actively plays (that is what accrues play time), `paused`, `idle`, `stopped`
  or `error`. `Activity` is short human text ("Fighting in Cold Plains"). `Character.XP` is XP within the current level and
  `XPNext` the XP needed for the next (0 if unknown).
- `farm_area.params.area` is a free name or a run key from `GET /catalog` (`run:mephisto`, ...); `areaId` may be set.
  If the bot does not know the area, return an error from `Start` (the goal then shows as failed with that message).
- An `idle` goal means: rest safely in town. With a schedule and no `minutes` it is a recurring rest window.
- Optional: implement `goals.Configurer` (`Configure(Settings)`) to receive the user's settings (chicken and potion life %,
  difficulty preference). Called at start and on every change.

## Goals

| type | params | completes when |
|---|---|---|
| `reach_level` | `level` 1..99 | character level >= level |
| `farm_area` | `area` (name or `run:*`), optional `areaId`, `untilKills`, `untilMinutes`, `untilDrops`, `item` filter | any limit reached; no limit = endless |
| `kill_boss` | `boss`, `count` (default 1) | `count` boss kills |
| `collect_item` | `item` {name substring, quality, type}, `count` | `count` matching items |
| `idle` | `minutes`, or only a `schedule` window | minutes elapsed; windowed idle never completes, it recurs |

Common fields: `title` (auto-generated if empty, and it keeps following the parameters while it is still the auto title),
`priority` (-10..10, higher first), `schedule` (list of windows `{days:[0..6], start:"HH:MM", end:"HH:MM"}`, local clock; a
window with start > end runs overnight; a goal with windows runs only inside one), `params.difficulty`
(`normal|nightmare|hell`). States: `queued`, `active`, `paused` (held by the user or after too many deaths), `done`, `failed`,
`skipped`. `progress`: `current`, `target`, `percent`, `kills`, `bossKills`, `drops`, `deaths`, `activeSeconds`, and optional `steps` (`{id,title,state,current,target}`), `phase`, `etaSeconds`, `source` (`memory|events|mixed`), `verified`, all omitted when empty (docs/goals/tracking.md).

Scheduling: live goals are ordered by priority, then queue order. The first one that is queued and inside its schedule
window runs; when an earlier goal becomes runnable (window opens, goal resumed, reorder) the running goal is switched
away and returns to `queued` with its progress kept. Moving a goal across a different priority adopts the neighbour's
priority, so the visible order is always the run order. At most 100 finished goals are kept.

Bot modes: `stopped` (default after a restart unless `autoResume`), `running`, `paused`. Stop returns the active goal to
the queue. A gap between ticks over 5 minutes (suspend, clock jump) is not credited as play time.

## Persistence

One JSON file (`Options.Path`), versioned (`"version": 1`). Writes are atomic: write+fsync `file.tmp`, move the old file to
`file.bak`, rename `.tmp` over the file, fsync the directory. Load order: file, then `.bak`. A corrupt file is moved to
`file.corrupt-<unix>` (newest 3 kept); the manager recovers from `.bak` or starts empty and reports `loadWarning` in
`/status` and the event log. A file with a newer version is refused (`ErrFutureVersion`) and never touched. Invalid goals in
the file are dropped with a warning. The active goal comes back as `queued`. Saved: goals, settings, statistics (totals,
24 h of per-minute and 30 days of per-hour buckets, last 100 drops and area visits). Not saved: the event log.

Memory is bounded: event log 1000 entries (ring), minute buckets 1440, hour buckets 720, drops 100, areas 100.

## HTTP API

Base `/api/v1`. JSON in and out (`Content-Type: application/json` required on POST/PUT/PATCH). Errors:
`{"error": "...", "code": "invalid|not_found|conflict|unauthorized|forbidden|unsupported|internal"}` with status
400/404/409/401/403/415/500. Unknown JSON fields are rejected. Bodies are limited to 64 KiB.

| method path | body | result |
|---|---|---|
| `GET /status` | | `{mode, activeGoalId, activeGoal, runner:{state,activity,goalId,message,character{...}}, xpPerHour, totals, rev, lastSeq, lastError, loadWarning, now, started, version}` |
| `POST /control/start` | | start, or resume from pause; returns status |
| `POST /control/pause` | | 409 unless running |
| `POST /control/stop` | | returns status |
| `GET /goals` | | all goals: live in run order, then finished newest first |
| `POST /goals` | `{type, title?, params, priority?, schedule?, position?}` | 201 + goal; `position` = index among live goals, default after goals of equal or higher priority |
| `GET /goals/{id}` | | goal |
| `PATCH /goals/{id}` | any of `{title, params (replaces), priority, schedule}` | goal; editing the running goal restarts it (progress kept); type cannot change |
| `DELETE /goals/{id}` | | 204 |
| `POST /goals/{id}/move` | `{position}` | goal; index among live goals |
| `POST /goals/reorder` | `{ids:[...]}` all live goal ids | goals |
| `POST /goals/{id}/pause` `resume` `skip` `reset` | | goal (`reset` = back to the queue with progress cleared) |
| `GET /events?since=&limit=` | | JSON array of log events, oldest first |
| `GET /events/stream` | | Server-Sent Events, see below |
| `GET /stats` | | `{since, totals, xpPerHour, qualities, drops[newest first], areas[newest first]}` |
| `GET /stats/series?window=6h&step=` | | `{window, step, points:[{t, xp, xpPerHour, kills, drops, deaths, potions}]}`; durations `30m`, `6h`, `7d`; step defaults by window; max 1000 points; steps under 1 h only cover the last 24 h |
| `GET /settings`, `PUT /settings` | `{autoResume, maxDeaths, chickenLifePct, potionLifePct, difficulty}` | settings |
| `GET /catalog` | | areas, bosses, named runs, item types, qualities, difficulties (static, for forms) |

Event: `{seq, time, type, message, goalId, monster, boss, xp, item{name,base,type,quality,qty}, level, area, areaId, potion, fatal}`.
Types: `kill drop death level_up area_change potion error` (from the runner) and `goal_added goal_started goal_done
goal_failed goal_paused goal_resumed goal_skipped goal_removed goal_edited goal_preempted control info` (from the manager).

### Server-Sent Events

`GET /events/stream` sends: `event: log` (one event; `id` = `seq`), `event: status` (the `/status` body, every second),
`event: goals` (the goal list, whenever anything changed, at most once a second). A new client gets the last 100 log events;
a reconnecting `EventSource` sends `Last-Event-ID` (or use `?since=<seq>`) and gets only newer ones. A heartbeat comment is
sent every 15 s.

## Security model

Binds to `127.0.0.1:8420` by default. Anything else requires a token (`api.New` refuses otherwise). With a token set,
every request needs `Authorization: Bearer <token>`, the `d2token` cookie, or `?token=`; opening the dashboard once with
`/?token=...` sets an HttpOnly SameSite=Strict cookie and strips the token from the URL. On a loopback listener the `Host`
header must be localhost, 127.0.0.1 or ::1 (DNS-rebinding defence; add names with `Config.AllowedHosts`, e.g. for an ssh
tunnel or proxy). Mutating requests must be `application/json` and, if they carry an `Origin`, it must match `Host`.
Responses carry a restrictive CSP (self only), nosniff, no-referrer. No TLS: put a proxy in front for remote use.

## Dashboard

`web/` (embedded via `web.FS`): vanilla HTML/CSS/JS, no build, no external requests. Character card and current goal, goal
queue with a guided add/edit form (five goal types, plain-language preview, optional priority/difficulty/time window),
reorder with Up/Down buttons, hold/skip/remove, live log, XP/hour line chart (keyboard readable, table view), drops by
quality, area history, settings, light/dark/auto. It uses only the endpoints above.

## One server, instance routes (internal/server)

`internal/server` replaces the d2bot API (8420), d2watch (8890) and d2sentinel (8895) servers: one listener, one token
(`api.Security`: Bearer, `X-Token`, `d2token` cookie, `?token=`), Host/Origin/Content-Type/CSP checks as above. Route table and
mount points are in the package doc. Summary:

- `GET /api/v1/instances`, `/api/v1/instances/{id}` summaries; `/api/v1/instances/{id}/...` = every route above (status, goals, settings, stats, catalog, control)
  plus `health`, `deaths`, `state`, `events` and `events/stream` as **event envelopes** (`internal/event`, `kind`/`data`, `instance` set; `id:` = bus seq, resume with `Last-Event-ID`/`?since=`), and `<mount>/` handlers (`viewport`).
- `GET /api/v1/fleet` (`{now, instances[], sentinel?, failsafe}`) and `/api/v1/fleet/events/stream` (`fleet` every second, `log` envelopes of all instances, no resume ids: seq is per instance).
- `GET /api/v1/sentinel`; `GET /instances/{id}/live/` per-instance display, `/live/` = host hub or the first instance.
- `/healthz` (process, open), `/readyz` (open, 503 until every instance's `Ready`).
- Compat: every other `/api/v1/*` (and `/api/v1/bot/{health,deaths,state}`) is the first instance with the old flat shapes (`/events` stays `goals.Event`).
- Instance ids are `[a-z0-9-]{1,16}`. An instance without a Bus streams envelopes derived from the manager's flat log (`event.FromGoals`).

## Tuning routes (dashboard Tuning tab)

Served by `cmd/d2bot` as instance mounts (`/api/v1/instances/{id}/knobs`, `/api/v1/instances/{id}/upgrades/{id}/unlock`), contract at the top of `web/panels.js`.
`GET knobs` returns `{knobs, values, profile, profiles, upgrades, telemetry?, readonly?}`; `PUT knobs` takes `{profile?, overrides?: {key: number|null}}` (null resets a knob to the profile value), is all or nothing (400 on an unknown knob or profile) and answers with the GET body; edits persist to `<data>/instances/<id>/state/tuning.json` and are re-applied at start. `POST upgrades/{id}/unlock` answers 501 until upgrade data exists (the list is empty). The flat `/api/v1/knobs` has no route yet (the server compat shim does not forward mounts).
