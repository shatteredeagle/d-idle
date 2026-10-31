# Boss readiness gate (BOSSGATE, WP1/WP3/WP5 of docs/soak-analysis.md)

- WP1 `internal/task/gate.go`: `bossGates` table (Burial, Tools, Tower, Andariel): min clvl, healing potions, TP, mana potions for casters.
  The campaign floor of Burial Grounds is now 8 (U: judgment from the Soaky log 11:05 to 11:12, clvl 6, belt 0, 11 potions in 358 s, 0 kills;
  runs.json has no Blood Raven row; `internal/boss/act1.go` MinLevel is still 6, not owned here). Level below the floor: planner returns the
  level step. Kit short at the floor: the travel step's Ready inserts a chores trip, and when chores cannot fix it the executor blocks the quest
  ("not ready") and the planner moves on. Mana potions are only checked for casters (Sorceress, Necromancer, Druid).
- WP3 `internal/bot/plan.go`: progression field `kit_clvl` (Cold Plains: 4). Below it the step is skipped unless `PickOptions.FullKit`
  (4 healing potions and a TP, `Runner.fullKit`). Cause (V by reading the code): with a belt under 2 potions the Den is excluded, and the
  "first eligible step" fallback then picked Cold Plains at clvl 1-3.
- WP5 `internal/bot/loop.go replanCause`, `task.go KillBoss`: the stale `targetArea` of an earlier leveling slice made the guard cancel every
  boss run (state clearing, area differs) after one guard tick (U: matches the 1 s runs; not seen live). A boss run now owns the step
  (guard re-plan checks off while `bossRun`), clears `targetArea` at start, logs "action cancelled: re-plan" with the reason, and starts at most
  once per 30 s per boss.

## Blood Raven gate (BRKILL)
`bossGates[QBurial]` is `BR: true`: the floor is class based (Sorceress, Druid, Amazon 9; others 8; knobs `boss.brCasterLevel`, `boss.brMeleeLevel`) and at least the wait level of a failed attempt (`World.BRWaitLevel`), the kit is `brShort` (potions, town portal, charges when known, mana for casters) plus `brGoldShort` (gold 250). Normal only; Nightmare and Hell keep their static floors. Details, attempt rules and the claim step: docs/recovery.md, section "Blood Raven for the free Rogue".
