# Necromancer gap list against TASK-942 (card necroepic, 2026-11-27)

> status: current offline audit. Tags V (verified: a test or a code read this run), S (secondary), U (unverified). Nothing here ran in the lab; every scenario below is validated only (`go run ./cmd/d2scenario validate`: ok, 251 scenarios). Lab setups are CHEATED stand-ins: mechanics only, never evidence about balance.

## Found and fixed as data (this branch, V by tests)

| Gap | Fix | Test |
|---|---|---|
| `necromancer_leveling_normal30` had no `combat` key, so it fought with the class default layout, which has no Corpse Explosion key (same defect necroaudit fixed for normal30) | new profile `necromancer/bonespear` in `internal/combat/classdata/profiles.json` (Teeth, Bone Spear, Bone Armor, Amplify Damage, Corpse Explosion, Clay Golem on F1 to F6); the table names it | `cmd/d2bot` `TestMainAttackSkillsHaveAKey` (fails without the fix, V: reverted and rerun) |
| `necromancer_summoner_endgame90` (used above clvl 30 by the default path) had no `combat` key: Corpse Explosion main attack with no key, Teeth-only bands | `"combat": "summoner"` | same test |
| Table hotkeys drifted from the profile layout (ratchet 5 and 7) | both tables' hotkeys follow the layout, ratchet entries removed | `TestBuildHotkeysDoNotDriftFromTheProfileLayout` ratchet |
| `internal/decide` catalogue knew only sorceress and paladin: a Necromancer hero had no policy-sheet rows, mana band or propose lead (errors "unknown class") | class rows in `catalogue.json`, `policy_chores.json` (stance knobs, the Sorceress caster values, U), `proposezone.json` (lead 1, U) | `internal/decide/necro_test.go`; the five tests that used "necromancer" as the unknown class now use "barbarian" |
| No Necromancer row in `levelplan-normal.json` or `act2plan.json` | caster rows copied from the Sorceress (review-only, `active:false`), build `necromancer_summoner_normal30` | `internal/levelplan/necro_test.go` (Necromancer and Sorceress), the existing act2plan shape tests |

## Acceptance criteria

| AC | State | What is left |
|---|---|---|
| 1 traits, plan table, combat profile data-only, two-build test | Done offline (V): traits row, 6 tables, 4 profiles (default, summoner, poisonnova, bonespear), every table resolves to a profile with a key for its main attack; the check runs on Necromancer and Sorceress tables | Other classes fail the same check today (Amazon Throw/Poison Javelin, Barbarian War Cry, Druid Tornado/Twister/Arctic Blast, Paladin Smite in the hammerdin endgame tables): their epics, not this one |
| 2 leveling clvl 1 to 30 | Needs live drill | `necro.level-1-30` (new, nightly, fresh hero, no_writes, level 30, deaths max 6 U); plus `necro.den-l14`, `necro.den-l16`, `necro.act1-earned` (existing) |
| 3 stat and skill allocation locked, spends every point | Offline done for summoner (existing `points.necro-summoner-lands-at-30`, autoalloc rows, `leveler` tests); new per-table rows `points.necro-{leveling-normal30,summoner-normal30,summoner-sme,poison-nova-sme,summoner-endgame90}-lands-at-30` need a lab run. Skill ids of Bone Spear (84), Poison Dagger (73), Poison Nova (92) are U (memory of skills.txt) | run the five rows; fix ids if a skill_points assertion reads 0 |
| 4 kit and economy, no 3 h death spiral | Offline: caster belt and loop rows exist (GREEN, matrix note 70); `classtraits` caster, mana need 2, `mercEarlyRogue` | Needs live: `necro.soak-3h-kit` (new, nightly). Merc choice: act2plan merc aura is archetype-keyed (caster: Defiance); docs/classes/necromancer.md says a Necromancer wants Might from the desert merc (S): a per-class merc aura row is not data today (needs a field in `act2plan.json` merc, then Go; not done) |
| 5 quest drills GREEN per act | Needs live drill | Act 1: `necro.act1-andariel`, `step.act1.*.necro` (existing GREEN). Acts 2 to 5, new file `necro-acts.json`: `necro.drill.act2.{radament,cube,shaft,amulet,assemble,summoner,duriel,radament.handin}`, `necro.drill.act3.{lam_esen,blade,golden_bird,council}`, `necro.drill.act4.{izual,hellforge}`, `necro.drill.act5.{siege,prison,betrayal,ancients,baal}`; setup is a caster stand-in (yew wand, 700 life, two-mana belt) |
| 6 fresh uncheated live Normal run | Needs live drill | `necro.act1-earned` (Act 1), `necro.earned-act2` (new; goals `campaign:normal_act2`, V exists); Acts 3 to 5 are the same scenario with `campaign:normal_act3..5`. Record N and the real clvl |
| 7 known blockers closed | Needs fork | Tools of the Trade reward word 0x204e (`necro.act1-tools` RED: the fork's Imbue line, backend limit); `necro.tools-paladin-control` BLOCKED by its seeding (scenario fix, not Necromancer); Smith/plan_overlap are GREEN with `-class Necromancer` (matrix rows already list it). Still open (U): minion caps and curse durations are guesses in the profile `source` fields; Weaken, Decrepify, Lower Resist, Fire Golem and Iron Golem have no F-key in the summoner layout (F8 is the last) |

## Exact commands (lab, night window, from a labrun-synced tree; none run here)

```
tools/provingground/run.sh -id necro.level-1-30,necro.earned-act2 -class Necromancer -seed 11 -parallel 1
tools/provingground/run.sh -id 'necro.drill.act2.,necro.drill.act3.,necro.drill.act4.,necro.drill.act5.' -class Necromancer -seed 11 -parallel 1
tools/provingground/run.sh -id 'points.necro-' -class Necromancer -seed 11 -parallel 1
tools/provingground/run.sh -id necro.soak-3h-kit -class Necromancer -seed 11 -parallel 1
```
(the `-id` prefix syntax is as in the soak note of `faultsoak.json`; U for a comma list of prefixes: use one run per prefix if the runner refuses it.)

## Proposed, not done
- A per-class merc aura row (Might for the summoner) once the SME confirms; Fire Golem and Revive keys need the layout past F8 (key budget, card necrokeys).
- ADR/decision text: none proposed.
