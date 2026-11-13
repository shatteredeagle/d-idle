# levelplan: SME-backed leveling changes (2026-11-12)

Source: d2-sme answers of 2026-11-12 (EXP, medium confidence; docs/research/awesome-diablo-mining-2-2026-11-12.md), hour analysis docs/probes/hour1-analysis-2026-11-12.md, docs/hourfix.md. Tags: V read from code or a test in this branch, S secondary (SME), U unverified live. Nothing here ran against the game.

## Changes
1. Den after the claim (V already done, docs/hourfix.md section 3). `questClaimed` (internal/bot/leveling.go) reads the live quest word (Stage Done) as well as the runner's map; `target0` excludes the Den in Normal once claimed (goal.go), `runFinished` does not queue a claim. The word is in the save, so a fresh game of the same hero and a restart both see it (V by `TestClaimedDenIsNotARunAgain` and the word test in earlyfix_test.go; U live in a second game). No change in this branch.
2. Blood Raven / Burial Grounds for melee (V data + test). New optional plan key `melee_min_clvl` (Step.MeleeMin, a HIGHER entry for Paladin and Barbarian only): Burial Grounds row gets `melee_min_clvl` 8 (S, SME). Casters keep `enter_clvl` 6. The campaign chapter `blood_raven` already has floor 8 / recommended 9 for non-casters and 9 / 10 for sorceress and necromancer (data/campaign/normal.json): unchanged. Test `TestBurialGroundsMeleeEntryIsHigher`.
3. Tristram stays a quest stop (V, no change): chapter `find_cain` is `optional`, floor 9 / recommended 11, and Tristram has no plan row, so it is not a leveling area. (SME: quest stop at clvl >= 11.)
4. Potion buffer (V data + test). `loot.potionBuffer` 2 to 4 (healing and every non-mana kind), new `loot.potionBufferMana` 2 (the SME wants 1 to 2 mana for the Paladin; one knob for both would carry 4 mana potions in 40 cells). The belt-fill path is untouched (`WillBeltFor` returns before the buffer test); only the pack-buffer comparison uses the new `potionBuffer(b)` helper. Tests `TestDefaultPackBuffer`, `TestPackBufferHealingAndManaApart` (Paladin, Barbarian, Sorceress; these skip without the game tables, run on the worker host).
5. Shrines: PROPOSAL ONLY, see below.
6. Bands: see the table. Changed only the clear cases.

## Area bands: SME vs plan table (Normal, clvl)
| Area | SME | plan before | plan after | note |
|---|---|---|---|---|
| Blood Moor | 1-4 | 1-6 | 1-4 | changed; caster at 4 stays, at 5 Cold Plains opens |
| Den of Evil | 3-6 | 2-6 | 2-6 | same exit; entry 2 vs 3 (U, quest floors 3 / 5 in campaign data cover it) |
| Cold Plains | 4-8 | 5 (melee 4)-10 | 5 (melee 4)-8 | changed exit |
| Cave 1 | 6-9 | 6-10 | 6-9 | changed exit |
| Burial Grounds | 5-10 | 6-10 | 6 (melee 8)-10 | melee entry raised; casters 6 vs SME 5 (U) |
| Stony Field | 7-11 | 8 (melee 7)-12 | same | exit 12 vs 11 (U, review) |
| Underground Passage | 8-12 | no row | no row | U: not in the plan; proposal for the area-data owner |
| Dark Wood | 8-12 | 9-13 | same | one level later at both ends (U) |
| Black Marsh | 10-14 | 10-14 | same | equal |
| Forgotten Tower | 12-18 | 10-14 | same | SME later and longer (U) |
| Cave 2 | 10-14 | 10-15 | same | exit 15 vs 14 (U) |
| Crypt | 13-17 | 10-17 | same | entry 10 vs 13 (U; row is fallback-kit gated) |
| Jail 1 | 14-20 | 13-16 | same | exit 16 vs 20 (U) |
The decomp14 XP/area table (docs/research/xp-gold-trace.md, in progress) will validate the U rows; do not retune them before it lands. Risk of the three exits: a melee clvl 8 hero finds Stony Field (melee 7) and Cave 1 open, a caster at 8 finds Stony Field (8) open; no gap in Normal (V by reading the rows, plan tests pass). U live.

## Shrines: finding and proposal
Finding (V by grep): the bot has no shrine logic. `internal/data/items.go` parses `ShrineFunction` of objects.txt, `internal/d2dctl` parses shrine lines of the d2d `debug objects` verb, `internal/drill/dfield.go` lists one drill row "operated by the bot's Interact", and `internal/d2/skilllevel.go` knows state 134 (shrine skill). Nothing in internal/bot, nav or town walks to a shrine or reads which one it is.
Proposal (not built). 1) Source: the field objects the d2d source (`d2dsrc`) already lists, kind `shrine`, with the Shrines.txt row name (V: `d2dctl.Object.Row`). 2) Table (data, not Go): row name to take / skip per class and clvl: take Experience, Armor, Combat, Mana Recharge, Skill, Resist x4 on melee; skip Fire, Light, Poison below clvl 10 (SME); the "Gem", "Stamina" and "Refilling" rows optional. 3) Behaviour: when a shrine is in the area and within N subtiles of the route, with no pack alive nearby, walk to OperateRange and click (same Interact path as chests); read the state flag afterwards to log what was granted. 4) Needs: the object id from objects.txt (ShrineFunction column) and the state ids per shrine, both readable from the data tables the bot already loads; live check on a hidden instance that Interact works on shrines (the dfield drill row). Card needed; owner of the interaction code: bot/nav.
