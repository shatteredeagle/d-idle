# live1: fixes from the sandbox live review of 2026-11-28

> status: current. Offline only, nothing here was run against a live game. Source of the findings: [the review](design/fable-review-2026-11-28/13-sandbox-live-review.md). Task: TASK-1119. Tags: **V** verified (how), **S** secondary, **U** unverified.

## What changed

| | Rule now | Where | Test |
|---|---|---|---|
| L1 | A travel error names the leg that failed (`legFault`: from, to). The first failure of a leg skips the target as before; the second failure of the same leg inside 5 minutes skips nothing and leaves by town portal. | `internal/bot/goal.go`, `task.go` (LevelTo) | `TestCutOffExitIsChargedOncePerLeg` (Sorceress, Paladin) |
| L2 | `survival.Decision.Hold`: the hole's best zone has no margin (under 0.05, the log prints 0.0). The bot refuses a hole walk to a zone other than the act's first field when Hold is set or while any plan zone is skipped for a reason other than a death streak; it takes the first field if the plan offers it and the kit is full, else picks nothing (the caller waits in town; fault skips are short). | `internal/survival/controller.go`, `internal/bot/zonemodel.go` | `TestHoleWithoutMarginHolds`, `TestZoneHoleNotWalkedAfterFaultBansOrWithoutMargin` |
| L3 | The board counts an identical navigation refusal (same text: exit, cause, cell) once per minute. A quest travel that fails twice on the same exit leaves by town portal before the next retry. | `internal/questboard/exec.go`, `internal/bot/task.go` (Travel) | `TestNoPathParkNamesTheExitAndCause`, `TestQuestTravelStepsOffAfterTheSameExitFailsTwice` |
| L4 | A non-emergency trip is held for a pending reward only when somebody will walk it home: the board runs or awaits a chunk, a campaign goal runs, or the legacy claim list holds it. One log line per hold. | `internal/bot/goal.go` | `TestRewardNobodyClaimsHoldsNoTripAndTheHoldLogsOnce` |
| L5 | `Report(true)` for an upkeep intent that needs town forgets the last area change, so the way back is not "undoing" the trip. | `internal/arbiter/arbiter.go` | `TestTravelBudgetAcceptsTheFieldAfterAFinishedTownTrip` |
| L6 | No `arbiter-starved` row for the leveling filler while the winning quest step is the board's training slice. | `internal/bot/starvedcause.go` | `TestTrainingStepDoesNotStarveTheLevelingFiller` |

All V by the named unit tests (fakes); U live for every row. Numbers that are judgment (U): 5 minutes leg memory, 0.05 margin, 1 minute refusal window.

## Not changed, and why

- **L3, the nav gate.** A second pocket cell per `ExitTo` was tried and reverted: on the synthetic Tower Cellar 3 maps (`internal/nav/cellar3gate_test.go`) it changed no outcome, because `ExitTo` already picks a new nearest cell as the hero moves. The live cell ({45 21}, {42 21}, cause `cut_off`) needs the recorded grid; the capture commands are in `docs/probes/cellar3-gate-capture.md`. Dead end until then.
- **L2 (c), a parked boss chunk removes its area from the leveling candidates.** Not done: the bot has no table from a board chunk to its areas. The fault-ban rule above covers the live case (Tower Cellar Level 4 was picked only after ten skips).
- **L4, the reward taken in passing.** S: the game pays a pending reward on the first interaction with the reward NPC (the Den reward comes with the plain talk, V live per `internal/town/imbue.go`), so "never in passing" means the bot does not use that NPC while an owner-only reward waits there. In act 1 that NPC is the potion seller. This is an owner decision (no potions until the owner takes the reward, or accept the passing claim); nothing was changed.
- **L7, vault full.** Working as decided: docs/vault.md "Decision 2026-11-21: a full Vault never removes anything". A drop the vault cannot record stays on the ground on purpose (so the pack does not clog), is counted as a missed drop, and one Needs-you line is raised per episode. Options that exist: a capacity step, `loot.vaultAutoTrim` (1), `loot.vaultPoorPickup`.
- **L8, gear at 0 percent.** Working as configured: with the repair gate the town repairs by the hero's repair MODE (docs/repair.md, `internal/repair/modes.json`), not by `town.repairBelowPct` or `town.tripRepairPct`. The trip reason "equipment at 0% durability" and "repair (not needed)" at 5 of 15 are the mode "When an item breaks" (S: read from the code path `repairTripAllowed` and `Permit`; the hero's mode itself was not read).
