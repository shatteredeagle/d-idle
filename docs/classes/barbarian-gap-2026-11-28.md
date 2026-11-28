# Barbarian gap list (card cls1, TASK-1111, 2026-11-28)

> status: current offline audit, same shape as `necromancer-gap-2026-11-27.md`. Tags V (verified: a test or a code read this run), S (secondary: an SME file), U (unverified). Nothing here ran in the lab; the scenarios below are validated only (`go run ./cmd/d2scenario validate`: ok, 272 scenarios). Lab setups with a level or item write are CHEATED stand-ins: mechanics only, never evidence about balance.

## Found and fixed as data (this branch, V by tests)

| Gap | Fix | Test |
|---|---|---|
| `internal/decide` had no Barbarian rows: every call returned "unknown class" | `catalogue.json` mana band 10/30 (S: pool about 10 at the start, Double Swing free from skill level 9, Frenzy 1.5 mana, `docs/sme/class-barbarian.md` sections 1 and 2); `policy_chores.json` stance knobs: the melee sentence values (radius, drink, leave, town potions) and its own crowd numbers (kites at 75 percent cautious and 60 aggressive, gives up a crowd at 12 and 20; S section 0: the biggest life pool of the seven); `proposezone.json` lead 0 (S sections 0 and 3) | `internal/decide` `TestShippedTablesKnowDruidAndBarbarian` |
| `act2plan.json` row was a Paladin copy (tag U) | judged: entries are the melee legacyFloor rows; life gates stay at the melee admission road (160 to 220), because a plan gate may not undercut its road (`TestAct2PlanShape`), so the SME's lower floors (change 7: 140 at the Halls, 200 at the Tombs) are not applied; they would not bind (about 305 life at clvl 20, S section 3). Tag rewritten | `internal/levelplan` `TestDruidAndBarbarianPlansAreTheirOwn` |
| `levelplan-normal.json` tag U | judged against the ready table (S section 10: Den and Blood Raven at 5 to 7, Andariel at 15 to 17): the melee rows hold; still `active: false` | same test |
| No merc aura row | `classtraits.json`: Defiance, Might, Defiance by difficulty (S section 5); no early Rogue (the free one after Blood Raven is enough for the class with the most life) | same test, `internal/d2` |
| War Cry profile carried the Zealot's surround numbers and backed out exactly when the nova works | `profiles.json` barbarian/warcry `melee`: surround 6, surroundLifePct 50 (S SME review 2026-11-27 change 3, confidence low to medium) | `internal/combat` `TestBuildsOfOneClassCarryTheirOwnStance` (warcry against frenzy) |
| `barbarian_warcry_endgame90` put the Radament point into War Cry above the per-level cap (5 ratchet rows, clvl 44 to 48) | the quest point at 44 buys Battle Cry, row 49 buys War Cry; totals equal from clvl 49 on | `internal/plandata` `TestBuildTablesConsistent`, ratchet lowered by 5 |

Already in place before this branch (V by reading): `combat` keys on all three tables, passive Iron Skin out of the buff ladder, War Cry as an attack kind.

## Build-specific data

Potion columns: class row health 1, mana 0 (`chores-defaults.json`), right for the melee and Frenzy tables; the War Cry table is `mana_heavy`, which adds the mana columns. Stance: per profile `melee` block (default and `frenzy` 4 and 65, `warcry` 6 and 50). Merc aura and decide rows are keyed by class.

## Acceptance criteria (the seven of the class epics)

| AC | State | What is left |
|---|---|---|
| 1 traits, plan table, combat profile data-only, two-build test | Done offline (V) | nothing offline |
| 2 leveling clvl 1 to 30 | Needs live drill | `barbarian.level-1-30` (new, nightly, fresh hero, no_writes) |
| 3 stat and skill allocation, spends every point | Offline: `internal/leveler` frenzy tests (V); cap rows left: `barbarian_frenzy_guide` (Double Swing 8 to 10, Frenzy 26 to 28), generated from `internal/buildfile/testdata/barbarian-frenzy-guide.json`, needs a goal-file change, not done | the `skillsspent` assert of `barbarian.level-1-30` |
| 4 kit and economy, no 3 h death spiral | Offline: four healing columns, repair every trip | Needs live: `barbarian.soak-3h-kit` |
| 5 quest drills per act | Needs live drill | existing `combat.whirlwind-clears` and its control; no per-act Barbarian drill file yet |
| 6 fresh uncheated Normal run | Needs live drill | `barbarian.earned-act2` |
| 7 known blockers | Needs live, one needs Go | Double Swing and Frenzy need two one-hand weapons and the bot starts with an axe and a shield: the gear check drops the skill, so the default profile fights with Bash (buying a second weapon is a policy gap, `docs/process/barbarian-support.md` B2; not data). `cast War Cry` was never seen live. Stamina is read by nothing (B7). Battle Command needs two casts (B3) |

## Exact commands (lab, night window; none run here)

```
tools/provingground/run.sh -id barbarian.level-1-30,barbarian.earned-act2 -class Barbarian -seed 11 -parallel 1
tools/provingground/run.sh -id barbarian.soak-3h-kit -class Barbarian -seed 11 -parallel 1
```

## Proposed, not done
- SME changes 4 and 5 for the War Cry table (Leap into Taunt, Battle Command at 49, Bash 1 then Double Swing): not applied; Double Swing needs the second weapon first.
- SME change 6 (`barbarian_melee_normal30` says Double Swing, the profile leads with Bash): left, Bash is the attack that works with a shield.
- The `frenzy` profile for the default table once the bot can hold two weapons. ADR or decision text: none proposed.
