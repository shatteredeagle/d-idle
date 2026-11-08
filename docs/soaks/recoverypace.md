# recoverypace: the early-game gold loop (TASK-183)

> status: measured 2026-11-08 (EST), lockstep d2d, `d2bot play` (ladder on = player profile), lab verbs off, writes after start: 0 (no `d2cheat` run, no memory write tool ran; pace measurement, not a gate run). Tags: V verified live (how stated), S read in code, U unverified.

## Diagnosis (V from `docs/probes/classpacing/minutes/*.jsonl` and the worker logs of the 8 classpacing runs, S from code)
- The claim "68 of 90 minutes on the recovery gold farm" is only half true. After TASK-138 the step label "Recovery: farming gold" covers 19 to 30 of the first 90 virtual minutes per class (Barbarian 29, Necromancer 26, Sorceress 25, Paladin 19, Amazon 20, Druid 30, Assassin 21); the rest of Blood Moor time is the campaign step "Level up to N" in the same field (V: `area` 2 for 45 to 63 minutes in every run).
- A fresh hero has 0 gold (V: `recovery: start (gold 0 < kit 500)` at second 0 in every log) and the kit costs 190 (portal 100 plus potions), so recovery starts at once. The bar of 500 never matters: the start test is `shortfall > gold` and 500 only stops a start when gold is already above it. What costs time is the END: a running recovery ignored that the kit had become affordable and went on until the 20 minute stall check (income at clvl 1 to 4 is about 100 gold per hour, V `recovery: gold +21 in 11 min` and similar), then the 60 minute hold, then a second recovery at minute 80 to 114 (V: bar1, nec1, sor1, pal2 all restart late).
- Pooled over the 7 classpacing runs, first 90 minutes (S: script in the report, V data): recovery 120 XP per virtual minute against 173 for the campaign "farming" step and 75 in town chores. Recovery is 26 percent of the time and town chores 17 percent. The old A/B (`docs/d2d-xp-gap.md`) found the label itself as productive as the quest step; the lower recovery rate here is the clvl 1 minutes.
- Gold use (V logs): sell trips at 4 items (`povertySellMin` 0 means 4 on foot) every 6 to 10 minutes, each 3 to 4 virtual minutes of "Returning to the field"; mana potions at 60 gold and a portal scroll at 100 are bought with the first 150 gold (Sorceress), the Den reward and Akara give no gold. Rebuy loops of the Amazon were the equip failure fixed by TASK-182 (#427); none are left in the Paladin or Sorceress logs.
- Cold Plains: the plan enters it at clvl 5 (kit level 4); the recovery XP pick may enter earlier only with `resource.recoveryXPEarly`. Not changed: the xpgap A/B (neutral) is still the evidence.

## Fix (class free, all data in knobs)
- `resource.recoveryEndOnKit` (0/1, code default 0): a running recovery ends with reason `kit affordable` when `shortfall <= gold`, the same test that keeps one from starting (`internal/resource/recovery.go`).
- Player and tester preset, applied only with the ladder on (`cmd/d2bot/ladder.go` `playerTuning`, used by `d2bot play` and the tester container, which set `D2BOT_LADDER=1`): `resource.recoveryEndOnKit` 1, `resource.recoveryStallMin` 10 (default 20), `resource.recoveryStallGold` 50 (default 100), `resource.povertySellMin` 8 (default 0, built-in 4). A key set in the config file wins; a saved `tuning.json` replaces the override map as before (the drill uses that to switch the preset off).
- Gates and soaks do not set the ladder, so their code defaults are unchanged (V `TestPlayerTuningPreset`: without the ladder `recoveryEndOnKit` is 0).
- Tests: `TestRecoveryEndsWhenKitAffordable` (caster kit with mana potions and melee kit, knob on and off), `TestPlayerTuningPreset` (profiles optimal and relaxed, config key wins, caller map unchanged), knob doc sync.

## Before and after (V, same binary, seed 7, one run per arm, `docs/probes/recoverypace/run-all.sh`, minute lines in `docs/probes/recoverypace/minutes/`, table by `an.py`)
Old arm = `tuning.json` with the pre-fix values; new arm = the preset. Runs stop at 75 to 87 virtual minutes (5 wall minutes); values are clvl/XP.

| run | v20 | v45 | v60 | v75 | recovery / town / farming / quest min | deaths |
|---|---|---|---|---|---|---|
| Paladin old | 3 / 3309 | 5 / 8068 | 5 / 12405 | 6 / 18037 | 19 / 15 / 40 / 3 (n=77) | 0 |
| Paladin new | 4 / 4736 | 5 / 8855 | 5 / 10644 | 5 / 13373 | 11 / 8 / 53 / 8 (n=80) | 0 |
| Sorceress old | 3 / 2562 | 4 / 5316 | 4 / 6518 | 5 / 9366 | 23 / 15 / 45 / 3 (n=87) | 0 |
| Sorceress new | 3 / 2655 | 4 / 5424 | 5 / 9318 | 5 / 13932 | 10 / 5 / 57 / 2 (n=75) | 0 |

Reading: time in recovery fell by 42 and 57 percent and town chores by 47 and 67 percent (V, both classes), the Paladin got clvl 4 at minute 20 instead of 3 and the Den quest at minute 11 instead of 21. XP at minute 75 is -26 percent (Paladin) and +49 percent (Sorceress): the worlds diverge after the first 20 minutes (the Paladin new run sat 15 minutes in an emptied Blood Moor at minutes 38 to 53), so one run per arm CANNOT show a pace gain beyond the recovery and town time saved; U for the XP effect (the xpgap A/B needs about 35 runs per arm for 5 percent). The old arm reproduces the classpacing numbers (Paladin v20 3309, v45 8068 against 7993; V). The Barbarian clvl 6 target (AC) was not re-run (U): the new Paladin and Sorceress arms show the recovery farm ends before 15 minutes (reason `stalled` at 10 minutes or `kit affordable`).
Seen: a `kit affordable` end can be followed by a new start seconds later when buying potions drops the gold below the kit again (pal-n cycles 2 and 3, no harm: the cooldown is bypassed by a low kit on purpose).

## Not done
Barbarian re-run, 90 virtual minutes (the runs stop at 75 to 87), several seeds, Cold Plains entry earlier (`recoveryXPEarly`), shorter trips (waypoint out of town, BACKLOG LEVELWP), TASK-138 AC2 (realtime hour).
