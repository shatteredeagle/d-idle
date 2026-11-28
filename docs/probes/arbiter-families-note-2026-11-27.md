# Arbiter families and admission: data note (2026-11-27)

> status: data note for the coordinator (TASK-1096, card fix7). No knob was changed on any sandbox by this card; `bot.arbiterFamilies` is the coordinator's to flip. Source: `docs/design/fable-review-2026-11-28/07-arbiter-and-deciders.md`; plan: `docs/design/arbiter-target-and-migration.md`. Adds to `docs/probes/dark-queue-final-2026-11-27.md` (row `bot.arbiterControl`, `bot.arbiterFamilies` 255: "stay on").

## What the data says

| claim | tag | where |
|---|---|---|
| Families 255 is six executing hooks and one veto; area choice stays in the chain | V read | review part 7, section 1 |
| The evidence the ledger row asks for (per-family verdicts on recorded facts, lockstep incident rows, a 24 h watch, a scored `-session` night) does not exist | V read | review part 7, section 3 |
| Under control the tick line's `chain` is the arbiter's family, so a night under 255 cannot feed the session verdict | V read | `docs/arbiter.md` |
| The admission counters of the two sandbox windows counted both-open as agreement; no row could fire for a melee hero above the quest-walk level in Act 1 | V read | review part 7, section 4 |
| One death row ("killed while Recovery: farming gold") falls after families 255 went on; whether the low-life gate was skipped was not recorded | U cause | `internal/plandata/admission.json`, `poverty.recoveryEvidence` |

## Recommended values (not applied)

- Now: `bot.arbiterFamilies` 14 (player, chores, claim) on the sandbox that carries 255; ledger rows to `trial`.
- 64 + 128 after the A3 change of card fix7 is deployed and one control-0 night is scored.
- 16 + 32 after the A1 change is deployed and a second scored night.
- 1 last, after the TASK-1034 lab rerun with the A2 change.

## Reading the admission line after fix7

`ops status --why` admission: `agree`, `differ`, and new `agreeRefused`, `oldOnly`, `newOnly`. A shadow window counts as evidence only when `agreeRefused` > 0 (a row fired and both sides refused) and `differ` is 0. `oldOnly` or `newOnly` above 0 fails the window; the log line `admission shadow differ=<cell>` names the area, the purpose and both answers.
