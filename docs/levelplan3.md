# Level plan v3: where the hero farms between quests (TASK-414)

> status: current (2026-11-14, agent levelplan3). Data `internal/plandata/levelplan-normal.json` (copy in docs/plan/data), code `internal/levelplan`, bot wiring `internal/bot/plan.go`. Replaces the band table of [levelplan.md](levelplan.md). Nothing here ran live: the predicted hour (section 5) is the check.

Tags: **V** verified (how named), **S** secondary (an earlier doc or log, named), **I** inferred (a model), **U** unverified. Inputs: kolbot-SoloPlay as paraphrased in [research/awesome-diablo-mining-2-2026-11-12.md](research/awesome-diablo-mining-2-2026-11-12.md) sections 3.1 to 3.4 (structure only, nothing copied); the AREAMON table (docs/research/areamon-trace.md, branch agent/decomp18); the survival limits of docs/research/boss1-trace.md (agent/decomp19) and docs/research/hitrec-trace.md (agent/decomp20); [research/pal-skill-matrix.md](research/pal-skill-matrix.md); [research/xp-gold-trace.md](research/xp-gold-trace.md); hour 4 ([probes/hour4-analysis-2026-11-14.md](probes/hour4-analysis-2026-11-14.md)).

## 1. Boundary: journey vs plan

- The **guided journey** (data/campaign/normal.json, internal/campaign, `task.Guide`) decides quest order and when a quest goes (floor, recommended, readiness checks, intents). Unchanged in kind.
- The **level plan** decides where the hero farms between quests. A quest slice in the plan only names a chapter id; it carries no levels (the loader rejects levels on a quest slice), so nothing is duplicated. The simulation reads the chapter's recommended level from the spine.
- The bot's level step (`task.StepLevel` -> `LevelTo` -> `Plan.Pick`) is where the plan acts: `Plan.Pick` walks the class's v3 steps when the class has an active plan (`PickOptions.Class`), else the progression rows. The campaign's own `farm` list (`Spine.FarmFor`) still only names the zone in the Why line; it can disagree with Pick (U, existed before v3; a follow-up could feed `Opts.Zone` from the plan).
- No new wall: a farm slice's gates choose a place to farm; chapters keep "warn, never wall".

## 2. Schema (levelplan/3)

Per class: `active` (the bot uses it; false = review-only, the class keeps progression-normal.json), `build` (the build table it assumes), `slices` in plan order. A farm slice: `areas` (route order; the next opens when the one before is cleared), `entry`/`exit` (inclusive band, as progression `enter_clvl`/`leave_clvl`), `no_merc_add` (extra levels without a living merc), `kit_clvl` (below it only with a full kit), `way_in` walk or waypoint, `via_wp`, `why`. Validation (`levelplan.Parse`, tests in internal/levelplan): every area exists and is no town, `via_wp` has a waypoint, entry < exit, bands monotone in plan order, no clvl gap from the first entry to the last exit with and without a merc, quest chapters exist in the spine, unknown class is an error naming it.

Choice rule (`ClassPlan.Choose`, used by the simulation; Pick's rule is the same in effect, I): the best XP per minute among the fresh areas of slices whose band holds the level; when none is fresh, the best of a fresh area of any enterable slice past its band and a re-clear of an in-band area at half rate. That is "advance before re-clearing": hour 4 lost its second half re-clearing Blood Moor and Cold Plains (S hour4 sections 1.3 and 5).

## 3. The Paladin slices (active)

| slice | areas | entry-exit | gates | XP/min (I) | kills/min (I) | why |
|---|---|---|---|---|---|---|
| blood_moor | Blood Moor | 1-3 | | 156 | 7.1 | the only field of clvl 1-2 |
| den (quest) | Den of Evil | journey 3/4 | | 235 | 6.3 | once for the skill point, never a farm (kolbot agrees) |
| cold_plains | Cold Plains | 3-6 | full kit at 3 | 435 | 14.6 | 2.8x Blood Moor; Bishibosh floor 5 (I) but cleared live at clvl 4-5, 0 deaths (S hour4) |
| stony_field | Stony Field | 5-8 | full kit at 5 | 625 | 11.2 | clvl 5 has no other fresh field; Rakanishu floor 6 (I): one level early, see section 6 |
| caves | Cave 1, Cave 2 | 6-9 | | 385 / 488 | 7.3 | filler; Coldcrow floor 6 (I) |
| blood_raven (quest) | Burial Grounds | journey 8/9 | | | | the Rogue; boss1 floor 8 = journey floor |
| black_marsh | Black Marsh | 7-11 | | 961-980 | 14.6 | mean mlvl 5.7, no superunique (I); fills clvl 7 |
| dark_wood | Dark Wood | 8-11 | | 1,022 | 17.0 | Treehead WoodFist floor 8 (I) |
| tamoe | Tamoe Highland | 9-13 | +2 without merc | 1,209 | 14.3 | best of clvl 7-11; no superunique; walk from Black Marsh |
| find_cain, countess (quests) | | journey | | | | optional; Tristram 953/min is under Dark Wood and Tamoe, Griswold floor 10 (I) |
| jail | Jail 1 | 11-14 | +2 without merc | 911-920 | 5.9 | filler on the way to the Catacombs |
| tools_of_the_trade (quest) | Barracks | journey 14/16 | | | | optional; the Smith is an Extra Strong superunique (boss1 B2) |
| catacombs | Catacombs 2, 3 | 12-18 | +2 without merc | 1,268 / 1,629 | 6.8 / 6.1 | best Act 1 fields to Andariel (v2 entered at 13 / 14) |
| andariel (quest) | | journey 17/20 (merc 15/18) | | | | boss1: 18 with a merc, 20 alone (I) |

Entry levels: survival (I): `max(the area's superunique floor of boss1-trace, mean mlvl <= clvl + 1 of areamon)`, with the one live exception (Cold Plains, S) and the one deliberate early entry (Stony Field, section 6). Exits: the level from which a later slice pays more per minute (I, areamon section 4). hitrec-trace: stun-lock is no Act 1 Normal risk for the Paladin, Zeal and Smite are not interruptible (V run); only packs that hit for life/8 and more cut the Attack cadence (I): this supports the earlier field entries, it does not prove them.

Burial Grounds, Crypt, Mausoleum, Cave 1 as a lone farm and the Forgotten Tower are no Paladin farm slices any more (294, 354, 304, 385 XP/min; the cellars are the Countess quest). Barracks, Jail 2, Pit and Hole pay well (areamon) but have no travel row in the bot yet (U) and the Barracks hold the Smith: not in v3.

Barbarian and Sorceress rows: same schema, `active` false, entries copied from today's melee / caster progression rows (U: no survival model for them), only the class-independent XP-table changes applied (Tamoe and the Catacombs farmed, Burial Grounds, Crypt and Mausoleum dropped). The other classes have no v3 plan and keep the progression rows (not the Paladin's, not the Sorceress's: `TestLevelPlanV3PickPaladin`).

## 4. Skill points (Paladin): no change to build-paladin-normal30-v2.json

| point | v2 (ours) | kolbot | decision and evidence |
|---|---|---|---|
| clvl 2 | Might | Might | agree |
| clvl 3 | Sacrifice | Sacrifice | agree; the profile casts it from clvl 2 with a 70 percent life floor (combat profile since palprofile; the v2 note "never cast before Zeal" is stale) |
| clvl 4 | Prayer | (none) | keep: the profile's rest aura below 70 percent life saves potions, the hour's supply limit |
| Den point | Might (row 5) | Resist Fire | keep Might: one aura at a time and the profile's aura ladder has no Resist Fire entry, so the point would idle; Might is +10 percent damage a point (pal-skill-matrix 3.3) |
| clvl 6-9 | Might | Holy Fire 6-8 | keep Might: Holy Fire needs about 3 monsters inside its range to beat Might (pal-skill-matrix 3.4, I); `combat.holyFireMinMonsters` (3) already gates it per fight |
| clvl 10 | Smite | Smite | agree |
| clvl 12-15 | Zeal 4 | Zeal 4 by 15, Charge 12 | agree on Zeal; Charge at 16 (the bot does not use it) |
| clvl 16-19 | Holy Fire | Holy Fire to 6 | keep |

Proposal, not done: Sacrifice +15 percent ED a point beats Might's +10 (pal-skill-matrix 5) and the life price is capped by the target's life (hitrec Q4, V read), so clvl 7-9 points into Sacrifice instead of Might are worth an A/B run of two tables; Might carries over to Zeal from clvl 15 and Sacrifice does not, so the gain is clvl 7-14 only (I).

## 5. Predicted trajectory (I) for the next measured hour

`levelplan.Simulate` (AREAMON time model: each area full rate for its clear time, half on a re-clear, 1 minute per area change; no deaths, no town trips, full kit, quests at the journey's recommended level, the Rogue from Blood Raven on). Calibration: plan v2's rows give 12,858 XP (clvl 5) at 60 minutes against the AREAMON model's 13,117 and the logged hour 4's 13,883 (S) (`TestSimulateOldPlanMatchesAreamon`).

| minute | v3 Paladin XP | clvl | plan v2, same model |
|---|---|---|---|
| 60 | 20,376 | 6 | 12,858 (clvl 5) |
| 120 | 63,772 | 10 | 27,680 (clvl 7) |
| 240 | 170,551 | 14 | |

Route of the model: Blood Moor to clvl 3 (minute 15), Cold Plains (29, clvl 5), Den (42), Stony Field (57, clvl 6), Cave 1 (75, clvl 7), Black Marsh (90, clvl 8), Dark Wood (104, clvl 9), Blood Raven (111), Tamoe (125, clvl 10), Jail 1 (150), Catacombs 2 and 3 (180-192, clvl 13). Against the first-hour tiers (docs/first-hour.md): 60 minutes at clvl 6 is above the floor (5), under the target (7 needs 22,680). The measured hour validates the model when clvl@60 is 6 and XP is 20,400 within 15 percent; Cold Plains kills per minute are the weakest assumption (the model 14.6, hour 4 9.7, V).

Barbarian and Sorceress on the same model (review rows, U): 60 minutes 9,174 / 5,690 XP, 240 minutes 101k (clvl 12) / 69k (clvl 10).

## 6. Deviations from kolbot and from the survival model

- Blood Raven at 8 (journey and boss1), not kolbot's 5 after the Cave: boss1 puts a third of a clvl 8 Paladin's life on her standing fight (I).
- Tristram is a quest stop, not kolbot's clvl 6-12 farm: 953 XP/min at clvl 8 against Dark Wood 1,022 and Tamoe 1,209 (V run XP, I time), Griswold floor 10 (I).
- The Smith at 14/16 (journey), not kolbot's 7: a superunique with Extra Strong (V run, boss1). Andariel recommended 20/18 (Paladin), floors 17/15 (ADR 0011), not kolbot's 11.
- Stony Field at clvl 5 with a full kit: one level under boss1's Rakanishu floor 6 (I). Reason: without it clvl 5 has no fresh field (hour 4's stall, S). boss1's floor is "at most half the life lost toe to toe with 4 potions", a soft criterion; drill X4 of areamon-trace (a clvl 3-5 Stony Field clear) decides.
- Black Marsh at 7 rests on the mean-mlvl proxy only (no superunique); the first walk crosses Dark Wood without clearing it (U).

## 7. Boss findings (boss1-trace B1 to B4)

- B1 done: `boss.View.Melee` (from the combat profile's melee block via `combat.Fighter.MeleeClass`, data, no class check) skips a script's Kite step; Andariel's script kited a melee Paladin out of reach every 1.5 s (`TestAndarielMeleeHeroDoesNotKite`). Hazards (poison clouds) still move the hero. U live.
- B2 data: journey chapter `tools_of_the_trade` floor 14, recommended 16 (was 10 / 13, merc 9 / 12); in Guided mode the chapter floor is the wall (`withWall` takes the max). Not changed: the boss script `MinLevel` 8 and the task gate and entry 8 (raising them breaks the Monastery route rules that share the Tools gate: route_test, task_test; owner of internal/task).
- B3 data: Paladin Andariel row recommended 20 / 18, floors 17 / 15 (the ADR 0011 interim floors; the old floors 16 / 14 were one under them).
- B4 not changed: the Countess run floor 14 (script, task gate, sim) is the cellars' number with a margin; boss1 puts the cellars at 11-13 (I). Splitting it touches the route guard knobs (`route.towerMinLevel`); listed for the task owner.

## 8. Unverified

Survival of every early entry (no live drill), the time model below Cave 1 (areamon section 2), the Pick vs Choose equivalence (I), travel to Tamoe, Jail 1 and the Catacombs by the bot (rows existed, never farmed), the Barbarian and Sorceress rows, merc survival after Blood Raven.
