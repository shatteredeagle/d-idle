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
