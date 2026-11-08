# Tester build: realtime check (TASK-33, gap G6)

> status: first audit and one realtime session, 2026-11-07. Tags: V verified live (how stated), S read in code only, U unverified.

The shipped tester d2d runs in realtime (`D2D_MODE=realtime`, the tester default of `deploy/d2d-container/entrypoint.sh`): no `--lockstep`, no `--lab-verbs`, no `step` driver. Everything the Gate 2 evidence measured ran in lockstep. This doc lists every place that assumes lockstep or virtual time and what it does when the clock is the wall.

How the bot picks its clock: `internal/d2denv/env.go` `LateClock.Detect` asks d2d `clock`; if `Lockstep` is false no `d2dinput.Pacer` is installed and `Sleep`/`Now` are the wall (S, read; V by the session below: `d2d backend up ... "lockstep":false`, no `pace` log line).

## Audit: places that assume lockstep or virtual time

| # | Place | Assumes | Realtime verdict | Fix needed |
|---|---|---|---|---|
| 1 | `cmd/d2bot/d2dwire.go` `d2dHealthWith`, `HangAfter: 90s` (`internal/launch/monitor.go` default 45 s) | lockstep: heartbeat is the wall clock, so a hang is never seen; realtime: the d2d counters (`beat.Next`) must advance | uses the real counters; a frozen d2d is seen after 90 s. V: no restart and no `heartbeat flat` line in the 1 h session (numbers below) | none; U: a real freeze was not provoked |
| 2 | same file, `pingGrace` 45 s and `lastStep` | a long `step` starves the ping | no steps in realtime: `lastStep` is zero, the ping alone decides (45 s silence = gone). V: the tap serves pings next to the bot (healthcheck healthy) | none |
| 3 | `internal/bot/stall.go` `campaignStall`, knobs `bot.stallWarnMin`, `bot.stallExitMin` ("virtual min") | `r.e.Clock.Now()` | wall in realtime, so the knob unit is real minutes; exit 5 ends the container and `restart: unless-stopped` brings it back (S). U: not provoked | none; decide whether the real-minute values are right for a person watching (card) |
| 4 | `internal/bot/drought.go` `bot.droughtMin` (8) | virtual minutes | same clock, real minutes; a drought is 8 real minutes without a kill | none (S) |
| 5 | `internal/bot/thoughts.go:34` ladder clock, thought fade (45 s) | `r.e.Clock.Now()-t0` | wall: V the thought card shows `remainingSec` counting down in real time and an unanswered question fades to its default | none |
| 6 | `internal/watch/hub.go` `Config.Speed` (default 1) | the page runs its countdown at `Speed` virtual s per wall s; `cmd/d2bot` never sets it | 1 is right for realtime. Under lockstep the page countdown would be wrong (U, not the tester case) | none for the tester |
| 7 | `cmd/d2bot/d2dbackend.go` `soakHours`, `stopAtVirtual`, `logPace` | lockstep only (guarded by `be.Clock.Lockstep()`) | not started; the tester has no run limit (`D2BOT_RUN_HOURS` unset); `--run-for` is wall | none |
| 8 | `cmd/d2bot/d2dbackend.go:183` `qs.Now = e.Clock.Now` (query cache) | virtual under lockstep | wall in realtime: the 200 ms cache is correct | none |
| 9 | `cmd/d2bot/mercwire.go` `envNow`, `cmd/d2bot/milestone.go` `virtual=` field | env clock | wall in realtime; the milestone line then says real minutes | none (naming only) |
| 10 | `internal/d2dinput/pacer.go` | a lockstep d2d | not used in realtime | none |
| 11 | tools and drills (`d2dsoak`, `d2drill`, the Gate harness) | `--lockstep --lab-verbs` | not part of the tester image path (`entrypoint.sh tester`); they still pass `--lockstep` | none |

Result of the audit: nothing in the tester path needs a code change to run in realtime; rows 1 to 5 are the ones the session below exercises.

## Realtime session (V live, container on the worker host)

See the section "Realtime session" in `docs/tester-live.md` for the numbers and the defects found by the run (the ones that needed an entrypoint fix before a session was possible are there too).
