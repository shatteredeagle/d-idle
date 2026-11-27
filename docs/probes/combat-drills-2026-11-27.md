# Combat drill scenarios, lab run commands (2026-11-27)

> status: current. Authored offline (card whirlscen); NOT run live. Tags: V verified, S secondary, U unverified.

Scenarios: `tools/provingground/scenarios/combatdrills.json` (tier T3 in `tiers.json`; `go run ./cmd/d2scenario validate` ok, V). Every setup is CHEATED (field warp, spawns, class skill points by `debug points`; new `pre.skills`, `internal/scenario/setup.go`, V test `TestSkillPointsAfterTheLevel`): mechanics evidence only, never balance. `no_writes` asserts nothing writes after the start.

| id | class, build | knob | oracle (log markers) |
|---|---|---|---|
| `combat.whirlwind-clears` | Barbarian clvl 30, `barbarian_melee_normal30`, Whirlwind 151 x1 (S) | `combat.deliveryUpkeep` 2 | `combat move use=whirl` min 1; none with life under 50 percent; no death |
| `combat.whirlwind-clears.control` | same | 0 | no `combat move use=whirl`; no death |
| `combat.hold-interruptible` | Paladin clvl 24, `paladin-zealot-lite` | 2 | Zeal attack, `combat potion` min 1 (U patterns) |
| `combat.teleport-reposition` | Sorceress clvl 24, `sorceress_fire_normal30`, Teleport 54 x1 | 2 | `combat move use=escape` min 1; no death |
| `combat.corpse-explodes` | Necromancer clvl 24, `necromancer_summoner_normal30`, Corpse Explosion 74 x3 | 2 | `combat corpse-explosion` min 1 |

Whirlwind needs clvl 30 (S skills.txt), so the brief's clvl 24 became 30. No build table learns Whirlwind, hence the SETUP points. Whether the played hero keeps Whirlwind usable (right-skill slot) is U.

Lab run (hidden instance, lab window, one at a time; shadow arm: add `D2SCEN_KNOBS="combat.deliveryUpkeep=1"`, expect `combat move shadow use=whirl`):

`tools/live.sh <agent> provingground -id combat.whirlwind-clears,combat.whirlwind-clears.control -class Barbarian -seed 11 -repeat 3 -parallel 1`
`tools/live.sh <agent> provingground -id combat.hold-interruptible -class Paladin -seed 11 -parallel 1`
`tools/live.sh <agent> provingground -id combat.teleport-reposition -class Sorceress -seed 11 -parallel 1`
`tools/live.sh <agent> provingground -id combat.corpse-explodes -class Necromancer -seed 11 -parallel 1`

Not asserted (the card oracles need counters or verbs that do not exist): potion within one poll of a scripted life drop, hits per cast for Corpse Explosion.

## Second batch (card combatscen2)

> Authored offline, NOT run live. Same CHEATED setup rules as above (field warp, spawns; `pre.skills` only where a row says so). Markers are the ones the code emits (V grep of internal/combat): `combat stance act=` (stance.go), `combat stance shadow` (mode 1), `combat trap skill=` (assassin.go), `combat minion skill=` (minions.go), `combat fight.tick ... attacks=` (castfirst.go).

| id | class, build | setup points | oracle | gap (no counter or marker, so not asserted) |
|---|---|---|---|---|
| `combat.hammer-in-pack` (+ `.control`, knob 0: no stance line) | Paladin clvl 24, `paladin_hammerdin_guide` | Blessed Hammer 112 x1 (CHEATED skill) | `combat stance act=` min 1; no death | casts in place, distance from the centroid |
| `combat.point-spell-hits` | Sorceress clvl 30, `sorceress_blizzard_endgame90` | Blizzard 59 x1 (CHEATED skill) | `fight.tick attacks=[1-9]`; no death | `AimCounts` (aim.go) is never logged and `aim shadow`/`aim refused` are Debug: needs an Info summary at fight end |
| `combat.point-spell-hits.druid` | Druid clvl 24, `druid_fire_normal30` | none | same | same |
| `combat.traps-stand` | Assassin clvl 30, `assassin_sentry_endgame90` | Charged Bolt Sentry 261 x4, Death Sentry 276 x1 (S ids, CHEATED skill) | `combat trap skill=` min 1; no death. No control arm: the planner is not behind the knob | traps standing or cap beyond the `standing=` field |
| `combat.army-holds` (+ `.control`, knob 0: no stance line) | Necromancer clvl 24, `necromancer_summoner_normal30` | none | `combat minion ` min 1; no death | hero-to-army distance |
| `combat.pets-hold` | Druid clvl 24, `druid_fire_normal30` | none | `combat minion ` min 1; no `combat stance ` line (a druid has no stance block); no death | pet distance and pet deaths |

Lab run (one at a time; shadow arm for the stance rows: `D2SCEN_KNOBS="combat.deliveryUpkeep=1"`, expect `combat stance shadow`):

`tools/live.sh <agent> provingground -id combat.hammer-in-pack,combat.hammer-in-pack.control -class Paladin -seed 11 -parallel 1`
`tools/live.sh <agent> provingground -id combat.point-spell-hits -class Sorceress -seed 11 -parallel 1`
`tools/live.sh <agent> provingground -id combat.point-spell-hits.druid,combat.pets-hold -class Druid -seed 11 -parallel 1`
`tools/live.sh <agent> provingground -id combat.traps-stand -class Assassin -seed 11 -parallel 1`
`tools/live.sh <agent> provingground -id combat.army-holds,combat.army-holds.control -class Necromancer -seed 11 -parallel 1`

U: skill ids 59 and 112 are used in repo docs and tests; 261 and 276 are from skills.txt memory (S); the build tables fix the profile (hammerdin, blizzard, fire, summoner) only if the plan id resolves to it (U until the first run). Pet and trap rows may need a first-run look at the exact log lines.
