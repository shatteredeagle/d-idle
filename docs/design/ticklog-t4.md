# Recorded sessions: the tick line (T4)

> status: current (card ticklog, TASK-981, 2026-11-26). Roadmap row T4 (docs/roadmap-fable.md); why: docs/research/fable-logic-review-2026-11-25.md 5.3 and 7.3.

## What it is
With `bot.tickLogSec` (default 0, `[dark-launch]`, ledger row in `data/darklaunch.json`) above 0 the bot writes one JSON log line `msg=tick` at most every that many seconds of the bot clock (V: `internal/bot/ticklog.go`, test `TestTickLogRateLimitedAndReplays`). It carries:

| field | meaning |
|---|---|
| `area` | area name, `town` in town (the arbiter's own view) |
| `kills`, `xp` | the minute line's lifetime kill counter and the character's XP; the replay's progress is their sum |
| `chain` | the hook family the goal chain acted for this tick (`chainFamily` of the whyline code), empty when it did not act |
| `objectiveArea` | area of the quest step the planner picked last (U, same caveat as the shadow) |
| `facts` | the arbiter's world facts (`arbiter.Fact*` names), the same map the shadow and the control path decide on |
| `vtMs` | the bot clock in Unix milliseconds (the game clock under lockstep) |

It is log only and independent of `bot.arbiterShadow`: with the shadow off the facts are still collected, the shadow is not created. The line passes the dedup handler unfolded because `vtMs` always differs.

## Replay
`internal/arbiter.Replay` (and so `cmd/d2arbiter`) reads tick lines. When a log has any tick line, its minute lines are skipped (their area is a number and their counters lag a minute), and the tick's `vtMs` is the time base instead of the wall-clock `time` (V: `TestReplayPrefersTickLines`; a minute-only log replays as before, `TestReplayMinuteOnly`). The night log of a sandbox is kept in memory only as the parsed minute and tick lines (about 30,000 small records for 8 h at 1 s).

## Procedure for one sandbox night (AC 2; not run yet, U)
1. Flip the knob per docs/process/dark-launch.md: `python3 tools/flip/flip.py <instance> bot.tickLogSec=1 --why "ticklog night" --window 24h --dry-run`, then without `--dry-run`. Do the same on the second sandbox.
2. After the night copy the bot's JSON log of each sandbox (the container's bot stdout log for that night) to the worker host scratch dir. Never commit it.
3. `go run ./cmd/d2arbiter LOG` (logged hero, divergence lines and per-family counts), then `go run ./cmd/d2arbiter -follow LOG` (the simulated hero obeys the arbiter).
4. Pass: both runs exit 0 with no panic and no `bad time` error, `ticks` close to the night's seconds divided by `bot.tickLogSec`. Record per sandbox: ticks, logged area changes, arbiter area changes (follow), travel refusals, the family counts line, and the first ten divergence lines in this doc's section below.
5. A short slice of a night (a few hundred lines, character names removed) may replace the assumed facts of the Dark Wood fixture (docs/arbiter.md, `internal/arbiter/testdata`).

## Results
None yet: no sandbox night recorded (offline build only).

## Limits
- U: `chain` can lag one activity (the whyline caveat in `internal/bot/shadow.go`).
- The arbiter's `Progress` in the live shadow is the observer's real-progress counter; the replay uses kills plus XP. Both only ask "did progress happen", but they are not the same number.
- Volume: 3,600 lines an hour at 1 s; `bot.tickLogSec` up to 60 thins it.
