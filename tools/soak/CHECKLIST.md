# Soak checklist (agent OBS)

Tool: `cmd/d2soak` (analysis in `analyze.go`), registry in `internal/metrics`, wrapper `tools/soak/soak.sh`.
Tags: V = verified offline (tests in this branch), U = unverified live.

## What a run measures
- Samples every 30 s into `samples.jsonl`: health (`/health`, 503 and unreachable are both "unhealthy"), `/stats` totals, `/status` level, new events by kind, sentinel `acted` lines, `/proc` stats per watched process (RSS, PSS, fds, threads, CPU ticks, zombie state, pid+start for restart detection) and the bot's own `/metrics` gauges (goroutines, fds, heap). V (sim bot behind `internal/server`).
- Report: XP/h, kills/h, deaths/h, ladder rungs/h, failsafe events, sentinel actions/h, longest no-progress window, longest unhealthy window, process restarts, least-squares slopes of RSS, fds, threads, goroutines, plus the `warm-up` and `fd plateau` lines (value at end of warm-up -> end, area steps ignored, residual slope). Exit code 0 pass, 1 fail, 2 usage.
- Fails on: deaths/h over limit, no progress longer than the limit (idle and paused time is exempt), health down longer than the limit, a fatal bot error, XP/h under the floor, ladder or sentinel rate over limit, a slope over its limit (judged after the warm-up, default 10 min, with at least 30 min and 10 samples of post-warm-up data; fd steps at `area_change` events are ignored, see docs/soak.md), a zombie for more than 3 samples.
- No-progress signal: the bot's `progressCounter` from health if present, else XP + kills + drops from `/stats`. Deaths: `/stats` totals (sum of positive deltas, so a bot restart does not subtract), else death events.

## Before a live run (main agent)
1. The bot serves `GET /metrics` (mount `metrics.Registry.Handler()`; call `reg.RegisterRuntime()` and `reg.RegisterProcs(...)`). Not wired by OBS (cmd/d2bot is not ours). Without it, goroutine and heap slopes are missing and the report says so only through absent rows; threads and fds come from `/proc`.
2. Check the match strings once: `pgrep -af 'Game.exe|wineserver|d2input|Xvfb|d2bot'` on the worker host (U).
3. Sentinel running, bot started through its normal path, character in town.

## Live commands (exact)
```
tools/soak/soak.sh start i4 --for 2h --profile m1        # M1: 2 h run, twice, two characters
tools/soak/soak.sh start i4 --for 24h --profile m3       # M3 exit
tools/soak/soak.sh start i4 --for 72h --profile v1       # v1 definition of done
tools/soak/soak.sh status | alerts | log | report        # status = one line; alerts also in <run>/alerts.json (docs/soak.md)
tools/soak/soak.sh stop                                  # SIGTERM: final report is written
```
Bare binary on the worker host: `d2soak run --target http://127.0.0.1:8420 --for 24h --out ~/d2lod-live/soak/r1 --sentinel-log ~/.local/state/d2lod/sentinel.jsonl --proc game=match:Game.exe --proc wineserver=match:wineserver --proc helper=match:d2input --proc xvfb=match:Xvfb --proc bot=match:d2bot`.
Warm-up and plateau knobs: `--warmup 10m` (0 = judge from the start), `--area-step-fds 8` (0 = count every fd step); `soak.sh plateau` prints the warm-up and fd plateau lines of the newest run. Re-analysis with other limits: `d2soak report --in samples.jsonl --max-no-progress 8m --json`.
Smoke test with no game: `go run ./cmd/d2dash -addr 127.0.0.1:8420` then `go run ./cmd/d2soak run --for 2m --every 5s --profile m3 --max-no-progress 1m` (d2dash serves the flat routes; there is no `/bot/health`, so health shows 404 and the run reports unhealthy: use it to see the plumbing only).

## Exit criteria (docs/archive/review/roadmap.md) and how the harness checks them
| criterion | profile / flag | note |
|---|---|---|
| M1: deaths <= 1/h, sentinel <= 2/h, longest no-progress < 5 min, >= 15,000 XP/h | `--profile m1` | XP floor is the average over the whole run, including levels 1-5 (roadmap says after clvl 5): lower with `--min-xp-per-hour` for the first run |
| M1: bot ladder rung >= 3 <= 1/h | `--max-ladder-per-hour 1` | the harness counts all rungs, stricter than the roadmap |
| M1: RSS growth < 5 %/h after 15 min, fds and goroutines flat | `--warmup 15m`, `--max-rss-mb-per-hour`, `--max-fds-per-hour`, `--max-goroutines-per-hour` | profiles set absolute slopes (table in docs/soak.md); for the roadmap's 15 min pass `--warmup 15m`; set the RSS limit to 5 % of the game's settled RSS (first report) |
| M1: 0 human interventions | not measurable here | the operator logs it |
| M3: 24 h, no fatal error, no manual restart, report produced | `--profile m3` | fatal and unhealthy windows fail the run; "manual restart" shows as a `processRestarts` count the operator must reconcile with sentinel actions |
| v1: 72 h, 0 manual restarts, report | `--profile v1` | same |
| gotchas 54, 55, 57, 62 | zombie, fd/thread/goroutine slopes, game RSS slope, no-progress | |

## Not covered (needs another package)
Gotcha 56 (log rotation) and 58 (disk guard) are not measured: add `df` and log-size gauges to the bot's `/metrics` (`GaugeFunc`) and they appear in `bot:*` slopes only if added to `leakGauges` in `cmd/d2soak/sample.go`.

## Recycle
The game recycles (Recycler, docs/soak.md) show as `processRestarts` for `game`; each restarted process gets its own warm-up. Reconcile them with the bot's `recycle` events, not with manual restarts. fd growth is a plateau (docs/leakhunt.md), so `leak` on `proc:game:fds` after this change means growth beyond area first-visits.
