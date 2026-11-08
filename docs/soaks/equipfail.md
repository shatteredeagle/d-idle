# equipfail: why the equip clicks failed (TASK-182)

> status: fixed and measured 2026-11-08 (EST), lockstep d2d, 5 wall minutes per run (about 90 virtual). Tags: V verified live (how stated), S read in code. Follows `docs/soaks/classpacing.md`.

## Method (V)
`docs/probes/equipfail/run-class.sh TAG CLASSIDX 7 RUNFOR [BUILD]` (copy of the classpacing script; `BIN=... BIN_PREBUILT=1` runs a prebuilt bot, used for the BEFORE binary). Fresh characters through the normal path (`d2bot play --backend d2d --managed`), lab verbs off, seed 7. writes after start: 0 for every run (V: no lab verbs, no cheat tool). Characters in `docs/probes/test-characters.md`. Logs stay on the worker host (`~/d2d-lab/equipfail/logs`).

## Two root causes, neither a click problem
1. Unidentified magic items (V: classpacing `bot-bar1`: the town chore logs `equip: skip Quilted Armor reason=unidentified` for item 46 at 04:56:44, the leveler clicks the same item into the torso slot two seconds later, `slot click refused ... stillOnCursor true`; same pattern for Sorceress ids 14 to 17, 28; the Amazon's Cap). The game refuses to wear an unidentified magic or better item (S: d2d `rules::meets_requirements` comment, FUN_0062eaf0). `leveler.Candidate` had a level rule for magic items (ilvl) but no identification rule; `town.unitReject` had both. Retries then failed with `cursor never held item N` because the failed try had put the item into another pack cell and the retry clicked the old cell (also fixed).
2. The Amazon's Short Bow (V: BEFORE binary rerun `eqa2base`, 28 times `town: item 15 did not reach slot rArm`; classpacing `ama2` 46). d2d takes off at most ONE worn item per equip click (S: `components/rules/rules.hpp` `equip`: "as long as only one item comes off in all"; a bow is two-handed, so the left hand must be free). The Amazon wears a Javelin (right) and a Buckler (left): two items would come off, the click is refused and the bow stays on the cursor. The bot then sold the bow and bought it again.

Not the cause (V): click coordinates and the previous-frame pointer: the same slot centres wore boots, gloves, helm, armor and belt in every run, with the 150 ms hover.

## Fix
- `internal/leveler/score.go` `Candidate`: a magic or better item that is not identified is not usable (Why "unidentified") until the identify chore has run.
- `internal/town/ammo.go` `shelveOffhand`: before a two-handed weapon goes into an occupied right hand, the occupied left-hand item is picked up and put into the pack (data driven: `data.ItemBase.TwoHanded`, no class check; only the ammo chore reaches it).
- `internal/leveler/equip.go` `doSwap`: a retry takes the item from its current pack cell.
- game.exe behaviour unchanged (all three are bot-side planning/ordering).

## Numbers (same seed 7, about 90 virtual minutes)
| Run | slot click refused | did not reach slot | cursor never held |
|---|---|---|---|
| Amazon BEFORE (`eqa2base`, old binary) | 1 | 28 | 3 |
| Amazon BEFORE (classpacing `ama2`) | 1 | 46 | 3 |
| Sorceress BEFORE (classpacing `sor1`) | 6 | 9 | 5 |
| Amazon AFTER (`eqa3fix`) | 0 | 0 | 0 (worn: Cap, Short Bow, Arrows) |
| Sorceress AFTER (`eqs1`) | 0 | 0 | 0 |
| Barbarian BEFORE (classpacing `bar1`) | 1 | 1 | 3 |
| Barbarian AFTER (`eqb2`, 94 virtual minutes, 5 equips) | 0 | 0 | 0 |

Each AFTER run is one seed-7 run of about 90 virtual minutes; the class runs differ after the first loot so this is a rate comparison, not a same-item comparison.

## Tests
`TestEquipSkipsUnidentifiedMagicItem` (Sorceress and Paladin; fails without the fix), `TestAmmoWearsTwoHandedBowOverJavelinAndBuckler` (sim flag `OneOffPerClick`; fails without the fix with the live message `did not reach slot rArm`). Both need the real tables (`/tmp/d2mpq`, the worker host).
