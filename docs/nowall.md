# NOWALL: campaign stall escalation

> status: implemented on agent/nowall (card nowall). Verified offline only; no live run.

Finding (docs/probes/navsoak-compare.md): after the permanent town wedge both navsoak runs logged the same "no eligible reachable leveling area" lines for hours (52k, 173k) with 0 area entries and 0 purchases, 950 save-and-exit re-entries did not help, and the loop watchdog stayed quiet because the virtual clock advanced.

## Where
- Progress: `observer.campaignAt` (internal/bot/observe.go), set on kill or xp gain (bumpReal), a non-town area entered, a gold decrease (purchase) and a quest step. Virtual clock.
- Ladder: `Runner.campaignStall` (internal/bot/stall.go), called every step before recovery and the goal, only while a goal runs and is not paused.
- Exit: Run returns `ErrCampaignStalled`; `haltError` (cmd/d2bot/live.go) maps it to status 5 (`exitStalled`), which the supervisor restarts (only 3 is final).

## Ladder and defaults
1. `bot.stallWarnMin` 30 virtual min without progress: one WARN `campaign stalled: no progress for X virtual min` (last progress time, state, step, area) and a fresh game (newGameRun then forgetGame, same path as freshGameToLevel; skipped without a Cycler).
2. `bot.stallExitMin` 30 more: ERROR and exit status 5, `campaign stalled, exiting for a restart`.
3. Any progress returns to stage 0. `bot.stallWarnMin` 0 = off (old behaviour).

## Evidence for 30 min
Minute lines of the worker host's navsoak and navsoak2 logs before their wedges (read-only awk; one minute line is one virtual minute; progress = kills up, xp up, area change or gold drop between minute lines): navsoak 2276 min, longest no-progress run 19 min (13 runs of 5+ min); navsoak2 581 min, longest 6 min. 30 leaves 1.5x margin over 19. The wedge would have cost at most 60 virtual min (a few wall min) instead of 60 to 167 wall min.

## Scope
Shared code, motivated by d2d (the original backend has the same loop; its exit path is the same haltError). Not verified live. The wedge cause itself is card townwedge (internal/nav).
