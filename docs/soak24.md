# Milestone alpha: the 24 h unattended soak (agent SOAK24)

> status: Game.exe soak runbook; d2d is the primary backend (ADR 0009, 2026-10-30; docs/plan/STATUS.md). G1/G2/Fresha ran with it (docs/g1-log.md); Game.exe is frozen and only the retirement soak (card gameexe-retire-soak) still uses it. The d2d soak is docs/d2d-soak.md.

Tags: V = verified offline (tests, `bash -n`), U = unverified (no live run was made for this document).
Tools: `tools/soak/soak24.sh` (driver), `tools/soak/soak.sh` (harness wrapper), `cmd/d2soak` (profile `m24`),
background: docs/soak.md (judging, recycler, alerts), docs/stallfatal.md (stall/ladder/fatal history).

## 1. Preconditions (all must hold; `soak24.sh preflight iN` checks the machine ones, V bash -n only)
1. Master is green: `labrun worker -- go test -short -count=1 ./...` (known flaky: tools/ci/flaky.txt). Not run by the script (the head box has 4 GB).
   Soak the master checkout: `git status` clean, `git log -1` noted in the hand-off.
2. Helper deployed from this checkout: `tools/helper.sh deploy`, then `tools/helper.sh restart iN` (or let the managed bot start it
   after `tools/game.sh stop iN`). Preflight compares the sha256 of `<i0 root>/helper/d2input.exe` on the worker host with a fresh build of this
   checkout (same flags as helper.sh). A different checkout path changes the hash (no -trimpath): deploy from the checkout you soak.
3. Instance iN is free: no `d2bot --instance iN`, no `d2soak-iN` unit, both game locks free, no `/tmp/d2lod-live.notice`. The game must
   be stopped first (`NOLOCK=1 tools/game.sh stop iN`): the managed bot starts it itself (docs/launch.md 9). i4 is the usual bot instance.
4. Load guard: 1-min load on the worker host at most `D2_SOAK_MAXLOAD` (default 12 of 16 cores; each running instance is about one core; being replaced by the floating cap `tools/capacity.sh --add 1.2`, docs/process/worker-capacity.md, no fixed instance count); at least
   `D2_SOAK_MINDISK_GB` (10) free on /home. Run on a quiet evening: other agents' live jobs distort stall and RSS numbers.
5. The character stands in town with panels closed; character data backed up (the soak will level it; a death loop is a failure, not a reason to cheat).
6. Operator is reachable for the first 2 h (gate) and checks twice a day after that.

## 2. Sequence
Gate first, then the long run. Never start the 24 h run if the gate failed.
```
tools/soak/soak24.sh preflight i4
tools/soak/soak24.sh start i4 --gate           # 2 h, profile m1, XP floor off: bot + harness
tools/soak/soak24.sh watch --every 10          # in a second pane; log: ~/soak24-watch.log
# after 2 h: the harness writes report.txt/report.json in ~/d2lod-live/soak/i4-<ts>/ ; read it
tools/soak/soak.sh report                      # exit 0 pass, 1 fail, 3 no data
tools/soak/soak24.sh start i4                  # 24 h, profile m24 (default)
```
`start` does: preflight (unless `--skip-preflight`), then
`D2_INSTANCE=i4 tools/live.sh --bg --nocap --max <secs+1800> soak24-i4 d2bot --managed --instance i4 --char Booti --addr 127.0.0.1:8420 --stop-game --goals campaign --autostart --run-for <hours+5 min>`
(a managed bot, own subreaper, campaign goal, saves and stops the game on SIGTERM; it runs 5 min longer than the harness), waits 20 s, then
`tools/soak/soak.sh start i4 --for 24h --profile m24` (systemd user unit `d2soak-i4`, Nice 10, MemoryMax 256M, waits up to 2 min for the bot).
Output: `~/d2lod-live/soak/i4-<ts>/{samples.jsonl,status.txt,alerts.json,limits.json,report.txt,report.json}`; bot binary and log `~/d2lod-live/soak24-i4/{d2bot,runs/*.log}`
(`tools/jobs.sh tail soak24-i4/<ts>-d2bot`). Another character: `--char NAME`; other instance: `start i3 --char Towni`. Port: `i4` 8420, others 8440+N, `D2_BOT_PORT` overrides.
Stop early, gracefully: `tools/soak/soak24.sh stop i4` (SIGTERM to the bot, then the harness; the final report is still written).
Do NOT use `tools/jobs.sh kill` alone (it does not reach d2bot, docs/launch.md 9).

Concurrency rule (V offline tests, live U): two soaks on two different instances run side by side; the same instance twice is refused (preflight FAILs on a running bot, a `d2soak-iN` unit or a held lock).
Everything a soak owns is per instance (`tools/soak/layout.sh`): agent `soak24-iN`, so the bot binary is `~/d2lod-live/soak24-iN/d2bot`, its pid/log `~/d2lod-live/soak24-iN/runs/`, the live.sh build dir `/tmp/d2live-soak24-iN`;
the harness unit `d2soak-iN` and run dirs `soak/iN-<ts>`; bot port 8420 for i4, 8440+N otherwise; local temp builds `/tmp/soak24-d2bot-iN`. The d2soak binary `~/d2lod-live/soak/d2soak` stays shared but is replaced by copy + rename, never in place.
`stop iN` and `status iN` act on that instance only (`soak.sh stop iN` stops only `d2soak-iN`); `watch` follows every instance with a running harness (`--inst iN` for one).
Old layout: a soak started before the split (i4 G2: bot `~/d2lod-live/soak24/d2bot`, logs `~/d2lod-live/soak24/runs/`) keeps working: `stop i4` falls back to that binary (pinned to `--instance i4`), `soak.sh start` falls back to the old log dir while the instance has no new-layout log, and nothing ever overwrites the shared `soak24/d2bot`.
Example while G2 runs on i4: `tools/soak/soak24.sh start i5 --for 4h --profile m24 --char Fresha`, then `tools/soak/soak24.sh watch --every 10` and `tools/soak/soak24.sh stop i5`.

## 3. Pass criteria (profile m24, `cmd/d2soak/main.go`; the numbers are U until the first 24 h run calibrates them)
| metric | limit | flag | note |
|---|---|---|---|
| deaths (death rule, plan-v2 10.2) | no `death_streak` / `death_recovery_stuck` / `death_rate` FAIL (docs/soak-analysis.md section 12) | `--death-rule` (on in m1 and m24), `--death-max-back`, `--death-min-kills`, `--deaths FILE` | bot log + `deaths.jsonl`; replaces the old deaths/h gate (m24 sets it to 0) |
| deaths/h (optional extra) | off in m1 and m24 | `--max-deaths-per-hour N` | `/stats` death total; still works when given |
| restarts/h (any watched process pid/start change) | alert > 8 in trailing hour; judged by hand against sentinel/recycle events | `--alert-restarts-per-hour` | recycles show here; report counts `processRestarts` |
| XP progress | >= 1000 XP/h average over the run; no XP change > 10 min (idle/paused exempt) | `--min-xp-per-hour`, `--max-no-progress` (also `--alert-stall`) | floor is low on purpose: it only rejects a bot that idles. Raise after run 1 |
| no gain (minute lines) | no clvl/quest-step/gold gain < 90 min (m1: 45 min); loops < 20; no Waiting > 10 min | `--no-gain`, `--loop-repeats`, `--idle-in-town` (report only; docs/soak-analysis.md section 10) | clvl floor per hour is a warning unless `--fail-below-floor` |
| max stall (no progress counter) | < 10 min | `--max-no-progress` | longest window printed in the report |
| bot health down | <= 3 min at a time | `--max-unhealthy` | a fatal bot error fails the run |
| ladder rungs/h, sentinel actions/h | <= 2, <= 2 | `--max-ladder-per-hour`, `--max-sentinel-per-hour` | |
| helper, game, bot fds | <= 5 fds/h slope after a 10 min warm-up, area-first-visit steps (>= 8 fds at area_change) ignored | `--max-fds-per-hour`, `--warmup`, `--area-step-fds` | "helper fds strict": a real leak of 5/h is 120 in a day |
| RSS, threads, goroutines | <= 40 MB/h, 5/h, 5/h | `--max-rss-mb-per-hour`, `--max-threads-per-hour`, `--max-goroutines-per-hour` | judged after warm-up, >= 30 min and 10 samples |
| wineserver fds | plateau: judged only with >= 2 h post-warm-up; <= 40 fds/h and slope x 24 h <= 2000 | `--wine-min-span`, `--wine-max-fds-per-hour`, `--wine-max-growth-24h` | docs/soak.md (WINEFIX); steps, not continuous growth |
| zombies | none for > 3 samples | `--max-zombie-samples` | |
| verdict | `report.txt` PASS, exit 0; NO-DATA (3) is not a pass | | |
Explicit flags beat the profile: `tools/soak/soak24.sh start i4 --max-fds-per-hour 8`. Re-judge a finished run with other limits:
`d2soak report --in <run>/samples.jsonl --profile m24 --max-no-progress 8m`. Process recycles (Recycler) restart game/wineserver segments, each with its own warm-up.
The roadmap's "no manual restart" is the operator's: any `soak24.sh stop`/`game.sh start` during the run voids it, note the time.

## 4. Alerts (`alerts.json`, rewritten every ~2 min; `soak.sh alerts`)
`{updated, level ok|warn|crit, alerts:[{level,code,msg}], status}`. `ok` = nothing. `warn` = trend to watch, no action needed now (`fd-slope`:
an fd series over its limit in the trailing 3 h; `no-samples`). `crit` = act now (`stall-xp`, `restarts`, `unhealthy`, `sampler-stale`).
Bot-log and Wine alerts (SOAKTOOLS2; V offline tests, live behaviour U): `tick-age` crit = no bot `minute` line for 3 min (`--alert-tick-age`; the bot is stalled
or dead even if /health answers); `game-restarts` warn = more than 2 game restarts in the trailing hour from the minute lines' `restarts` total
(`--alert-game-restarts-per-hour`; chicken re-entries count, so judge by hand); `wine-orphans` warn = more than 2 orphaned `winedevice.exe` (parent not in the
instance prefix, ~35 MB each, docs/leakhunt.md; `--alert-wine-orphans`, sampled per prefix via `--wine-prefix`); `panics` warn (SUPFIX counter, already there).
`soak.sh start` passes `--bot-log` (newest `*.log` of `$D2_SOAK_BOTLOGDIR`, default `~/d2lod-live/soak24/runs`) and both prefix layouts; `soak.sh status` adds the bot
log's last minute line, game restarts/h and the winedevice count; the report gets a `winedevice` line and a `wine-orphans` warning (m24: `--max-wine-orphans 2`, never a failure).
Memory alert (NAVMEM, V offline test, live U): `heap_runaway` crit = the bot's RSS above 8192 MB (`-alert-heap-runaway-mb`) or up by more than 1024 MB within 5 min (`-alert-heap-growth-mb`, `-alert-heap-growth-window`). Normal runs sit at 40 to 120 MB. The bot's own heap guard (`bot.heapLimitMB`) usually acts first; see docs/stallfatal.md 7b.

XP alerts (XPLOSS/XPWIRE; V offline tests, live U): `xp_at_risk` warn = unsaved XP (memory exp minus the `.d2s` exp) above one level's worth; it needs `--save-file`, which `soak.sh` passes automatically from `D2_SOAK_SAVEFILE`, else `D2_SOAK_CHAR` (set by `soak24.sh start --char`), else the instance's newest `.d2s`. The report's `xpSource` is `save` (exp stored in the .d2s, what survives a crash), `memory` (live exp, includes unsaved) or `counter` (gross kill-XP counter, counts XP later lost); `xp_lost` is the XP lost to restarts. Details: docs/soak.md. The m24 XP floor judges `xp`, so with a save file it measures saved progress.
`--for` accepts `24h`, `90m`, `1h30m`, `45s` (`tools/soak/duration.sh`, test `bash tools/soak/soak24_test.sh`).
Progress and death alerts (SIDECAR; V offline tests, live U): the report's `no_gain`, `loop_no_gain`, `idle_in_town`, `state_dominant`, `below_floor_clvl` and the death rule (`death_streak`, `death_recovery_stuck`, `death_rate`, `death_recovery_share`, `death_pair`, `death_repeat_site_no_action`) run in the live refresh too, through the same judge as `d2soak report`; a report fail is `crit`, a warn is `warn`, and an alert disappears when its condition no longer holds (table: docs/soak-analysis.md section 13). Every alert has a `since` time. `alerts.json` and the last line of `status.txt` carry the summary `live 60m: clvl, xp/h, deaths, recovery share, top state group`. `no_gain` counts clvl, quest-step, net gold above its mark and XP (>= max(`Progress.NoGainMinXP`=500, 1% of the level's span) since the last progress point); the knob is `-no-gain-min-xp` / `limits.json` `Progress.NoGainMinXP` (0 = 500, -1 = ignore xp). Optional push: set `D2SOAK_ALERT_CMD` (a shell line, default unset: nothing is run or contacted) in the sidecar's environment; it runs once per new crit alert with `D2SOAK_ALERT_CODE/_MSG/_SINCE/_RUN/_LEVEL/_STATUS`. The sidecar needs no `limits.json` block: missing progress/death limits default to profile m24 and the effective limits are written to the run's `limits.json`.
The `watch` log prints the one-line status, the live summary line, the unit state, the level and every alert with its since time (`tools/soak/watchfmt.py`); grep it with `grep -E 'level: crit|ALERT crit' ~/soak24-watch.log`.

## 5. What to do per failure class
| symptom | meaning | action |
|---|---|---|
| `stall-xp` crit, bot health ok | failsafe ladder (reset, rung 4 restart) should handle it; stalls in a town loop or an exit-less area | wait one ladder cycle (about 10 min). Still crit: read the bot log, docs/stallfatal.md sections 1 and 4 (task-layer loops, `nav no exit known`), record area and activity; do not restart unless the bot is paused |
| `unhealthy` / bot gone, game gone | fatal (docs/stallfatal.md 2 and 5: exit after chicken Save and Exit, game gone) | `tools/jobs.sh tail soak24/<job>` for the `fatal` line; the run FAILS; restart only for diagnosis (counts as a manual restart) |
| `restarts` crit | recycle or crash loop; also see the Recycler in docs/soak.md | which process in the message: game/wineserver recycling too often = login loop (docs/oog.md); d2input = helper crash log `tools/helper.sh log iN` |
| `fd-slope` warn | helper/game/bot fd growth | docs/leakhunt.md (steps at new areas are expected and ignored); wineserver: docs/soak.md WINEFIX; confirm over 3 h before calling it a leak |
| deaths/h over limit, chicken loops | docs/stallfatal.md 6 (low-life chicken/re-entry loop); docs/difficulty.md level gap rule | note the area; a death spiral may need the planner's avoid list; the run fails if > 1/h |
| `sampler-stale` crit | the harness died | `systemctl --user status d2soak-iN`, `tools/soak/soak.sh log`; restart the harness only: `tools/soak/soak.sh start iN --for <remaining> --profile m24` (new run dir; merge by hand) |
| load > guard during the run | other agents | the soak is `Nice 10`; ask other agents to pause; do not kill their jobs |
| game frozen frame | `tools/devhealth.sh` shows STATIC | `/tmp/d2lod-live.notice` and the sentinel (docs/sentinel.md) handle it; count sentinel actions |

## 6. Hand-off (when the run ends, pass or fail)
1. Final report (under 300 words) to the main agent: instance, character, start/end time and commit hash, profile and any flag overrides, the
   `report.txt` verdict and its numbers (deaths/h, XP/h, restarts, longest stall, fd slopes per series incl. wineserver, RSS), alert
   timeline (first crit and what it was), manual interventions (time, why), what failed and which section above handled it.
2. Copy `report.txt`, `report.json`, `alerts.json` and the watch log to `/mnt/cold/d2lod/soak/<run>/` (cold data; no samples to git).
3. Memory note: update `~/.claude/memory/d2-idle-bot.md` with one line: date, hash, verdict, the calibrated thresholds that should replace the U numbers in
   `cmd/d2soak/main.go` (m24) and in this document.
4. Leave the instance in town: `tools/soak/soak24.sh stop iN` (bot saves, stops the game); say how to bring it back (`tools/game.sh start iN` plus OOG login by the main agent).

## 7. Not done / unverified here
Nothing was run against the worker host: preflight's remote part, `start`, `watch` against a real run and the `m24` numbers are U. Master-green is a manual step.
V offline: `bash -n tools/soak/soak24.sh`, the usage/argument paths, the alert parse of `watch` against a sample JSON, `TestProfileM24`.

## Before the 24 h run: crash drill (M6)
Proves on a live game that `--supervise` recovers from a real crash (tags: V = offline tests pass, U = never run live; the drill is U until run).
Offline (V): `go test -race -count=1 -p 1 ./cmd/d2bot` (`supervise_proc_test.go`: the supervisor against a fake child process that crashes twice then exits 0:
backoff 5 s then 10 s, `--goals`/`D2BOT_GOALS` dropped and `--autostart --clean-start` added on every restart, the crash hook dropped so it fires once, cap gives up,
exit 3 not restarted, cancel (SIGTERM) forwarded and no restart). The game cleanup itself is child-side (`--clean-start`, `cmd/d2bot/game.go`): the supervisor requests it once per restart.
Test hook: `d2bot --test-crash-after N` (or `D2BOT_TEST=1 D2BOT_TEST_CRASH_AFTER=N`) panics in an unprotected goroutine after N s, exit 2.
Live (U), on a quiet evening, instance i3 (Towni) or i5 (Multi), never i4 (the soak instance) and not i0/i1/i2; the character must be an allow-listed throwaway:
```
tools/soak/soak24.sh preflight i3            # instance free, game stopped first: NOLOCK=1 tools/game.sh stop i3
tools/soak/crashdrill.sh i3 --crash-after 90 # about 5 to 8 min; add --char / --wait SECS if needed
```
It starts the supervised managed bot (`--supervise-max 3 --goals idle --autostart --run-for 30m`), waits for `supervise: d2bot exited, restarting`, then for
`/healthz` `ready:true` (fresh game attached, character in game), checks the `clean start` and `managed game attached` log lines, that `tools/game.sh reap i3 --dry-run`
lists no orphan winedevice.exe and `game.sh status` shows one game, then `soak24.sh stop i3` and expects `child ended, not restarting`. Exit 0 = PASS.
If it fails: `tools/jobs.sh tail crashdrill/<ts>-d2bot 200`; clean up with `tools/soak/soak24.sh stop i3` and `tools/game.sh stop i3` (reap runs with stop).
Do not start the 24 h run until this passes once. Open question (U): the death-screen stale state after a crash and the OOG re-login of the relaunched game are only exercised by this drill.

## Crash survival (panicsafe)

- In the runner (`internal/bot/panicsafe.go`): a panic in a step is recovered, logged at ERROR with the stack, emitted as a non-fatal error event ("Failsafe: internal error recovered"), and handled as a failed action (failsafe ladder). More than 5 panics within an hour self-pause the bot, which resumes by itself (5 min first, backoff up to 1 h). `Health().panics` counts them. U: tested with a panicking Mover only, not live.
- Helper goroutines of `cmd/d2bot/live.go` (runner, manager, log, bus bridge, display, probe) restart after a panic (`goSafe`, max 10/hour); an API server panic ends the process with an error.
- `d2bot --supervise` (used by `soak24.sh start`) restarts the process when it exits non-zero: backoff 5 s doubling to 2 min, at most `--supervise-max` (10) restarts per hour, `--run-for` shortened by the elapsed time. Restarts add `--clean-start`, which stops the game the dead process left behind before launching a fresh one. Exit 0 or SIGTERM ends supervision. U: the clean-start relaunch and re-login path is untested live.
- Failsafe owners (docs/stallfatal.md s7, tools/soak/failsafe.sh): preflight prints the effective config and FAILs when the sentinel (`/api/v1/sentinel` on the worker host) may act on the soak instance; `bash tools/soak/failsafe_test.sh` tests the printout. Watchdog stays at its 10 min default, supervise is on, d2soak observes.

## Build stamp (STRATEGY item 2)
Every soak must be traceable to a commit. `tools/live.sh` computes `tools/stamp.sh` (`$D2_VERSION`, else `git describe --always --dirty`, else `tree-<content hash>`) on the head box and builds with `-ldflags -X`; it prints `built <pkg> version=<stamp>`. Without ldflags, `internal/buildinfo.Current()` falls back to the go tool's VCS revision (V: a local `go build` printed `d2bot f79afac-dirty`). d2bot logs `d2bot start version=...` and reports it in `/healthz` and `/status` (`d2bot <stamp>`). d2soak writes `version.txt` in the run dir and a `bot version` line in the report (and `botVersion` in report.json). `soak24.sh preflight` (so `start`/`--gate`) FAILS unless the d2bot it would ship reports exactly `git rev-parse --short HEAD`; override with `D2_SOAK_ALLOW_DEV=1`. If you run from a tree without .git, set `D2_VERSION=<short commit>`.

## 8. Memory cgroup: recommended MemoryMax for the bot (NAVMEM; not applied, the live units were not touched)

Situation (V, read on the worker host 2026-10-30): the `d2soak-i4.service` sidecar already has `MemoryMax=256M` (tools/soak/soak.sh:71).
The bot is NOT in a unit: `soak24.sh` starts it through `tools/live.sh --bg` over ssh, so `d2bot --supervise` and its
child live in the login session scope (`/proc/<pid>/cgroup` = `user.slice/user-1000.slice/session-N.scope`), and
`user-1000.slice/memory.max` is `max`. The worker host has 62 GB RAM, 69 GB swap on NVMe, `overcommit_memory=0`. So one runaway bot
takes the whole machine's memory (the kernel killed d2bot at 48.8 GB RSS; a different process could just as well have been
picked, and the swap thrash before it stalls every other instance).

Recommendation: run the bot in its own transient scope, for example in `tools/live.sh` where it starts d2bot:

    systemd-run --user --scope --unit=d2bot-$inst -p MemoryHigh=10G -p MemoryMax=12G -p MemorySwapMax=0 -- <d2bot cmd>

Why these numbers (S): the Go heap is normally under 100 MB; the game under Wine plus the helper add about 2 to 3 GB to
the same cgroup (children of the bot); `MemoryMax=12G` leaves 4x headroom over that. The heap guard trips at 4 GB of Go heap, so the cgroup limit is the net for what the guard cannot see (a runaway in the game or in cgo) or when it cannot run. `MemoryHigh` throttles before the kill,
`MemorySwapMax=0` stops a runaway from paging the machine. A cgroup OOM kills the largest process of that cgroup (the
bot child, `memory.oom.group` is 0), the `--supervise` parent survives and restarts it (exit 137 is restarted, V
supervise.go). Per instance a scope, so i1..i5 cannot hurt each other. Verify before relying on it (U): start one bot
this way and read `cat /sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/d2bot-i4.scope/memory.max`.
