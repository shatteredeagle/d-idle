# Mercenaries and post-campaign farming (research, MERCRESEARCH, 2026-10-29)

> status: SUPERSEDED by docs/merc.md, docs/merc-wire.md, docs/sme/mercenary.md and openspec/specs/merc-policy/spec.md; kept for the research.

Target: classic LoD 1.14d (expansion character), offline single-player. Tags: **V** = verified (our code or live, or a source I fetched, URL given),
**S** = snippet or secondary source, **U** = unverified or from memory. **D2R** marks facts from D2R-era pages that may not hold in 1.14d.
Data tables below come from our own extracted real 1.14d tables in the shared cache (`hireling.tsv`, `monstats.tsv`, `monlvl.tsv`, `experience.tsv`,
`treasureclassex.tsv`, `superuniques.tsv`; V as extracts of the real MPQ data, the same files `internal/data` loads).

## 1. Mercenaries

### 1.1 Who exists (V, hireling.tsv)
| Act | NPC (class) | Merc (unit class) | Variants per difficulty | Hire cost factor `Gold` (N / NM / H) |
|---|---|---|---|---|
| 1 | Kashya (150) | Rogue Scout (271) | Fire Arrow or Cold Arrow, both with Inner Sight | 100-150 / 6000-7500 / 12500-14000 |
| 2 | Greiz (198) | Desert Mercenary (338), attack Jab | Normal: Prayer (Combat), Defiance (Defensive), Blessed Aim (Offensive). Nightmare: Thorns, Holy Freeze, Might. Hell: Prayer, Defiance, Blessed Aim again | 350 / 7900 / 15000 |
| 3 | Asheara (252) | Eastern Sorceror / Iron Wolf (359): Fire (Inferno, Fire Ball), Cold (Glacial Spike, Frozen Armor, Ice Blast), Lightning (Charged Bolt, Lightning) | | 1000-1500 / 9500-12000 / 21000-27000 |
| 4 | Tyrael | none to hire | | |
| 5 | Qual-Kehk (515) | Barbarian (561), Bash + Stun | 1hs rows | 9000 / 18000 / 32000 |

`Gold` is a factor, not the price: the live Normal Act 2 row (factor 350) read "360" and gold fell 20000 -> 19650 (V docs/town.md, Palli, merc level about 9).
The price formula by merc level is U; budget Nightmare Act 2 at roughly 8-12k and Hell at 15k+ (U, scale from the factors) and read the quoted price from the window.

Notes on the Act 2 aura table (important, it contradicts a lot of web text):
- **V (our extracted 1.14d hireling.tsv)**: Hell rows 12-14 are Prayer, Defiance, Blessed Aim; Might, Holy Freeze, Thorns exist ONLY in Nightmare rows 9-11.
- V (Hive Workshop classic Battle.net mirror, https://classic.hiveworkshop.com/diablo2exp/basics/hirelings.shtml, fetched) shows the same table (Combat Prayer/Thorns/Prayer, Defense Defiance/Holy Freeze/Defiance, Offense Blessed Aim/Might/Blessed Aim).
- Maxroll (https://maxroll.gg/d2/resources/mercenary, fetched, **D2R**) says Nightmare and Hell both offer all six auras. That is D2R behaviour; do not rely on it for 1.14d.
- Consequence for 1.14d: a Might merc must be **hired in Nightmare** and kept alive (revived) through Hell. Existing `docs/progression/merc.md` row "Hell: Desert Mercenary Might" is wrong for 1.14d (V from tables). The merc keeps leveling with the player and revives with gear, so this is the standard plan (S Hive Workshop: revive keeps level, exp and equipment).
- hireling.tsv holds two blocks of the same hirelings: merc levels 9/31/55 with lower HP growth, and 9/43/75 with higher growth (e.g. Act 2 HP/Lvl 20 vs 25). Which block the game uses for a LoD character is U (probably the second, the expansion block). `town.MercCandidates` takes the first row of each Id, so any stat use of it must pick the right block first.

### 1.2 Which merc is best for a bot
| Situation | Pick | Why | Tag |
|---|---|---|---|
| Normal, before Lut Gholein (clvl 1-17) | Free Rogue from Kashya after Blood Raven (Act 1 quest 2, Sisters' Burial Grounds / Blood Raven); a hire is possible from clvl 8 | cost factor 100-150, cold arrows slow; dies fast, Normal-only value | S Maxroll (quest-gift, clvl 8 rule, D2R wording); gold factor V |
| Normal Act 2 on | Desert Merc **Defiance** (Defensive) for most builds, **Prayer** (Combat) if the build has little life regen; Blessed Aim only for bow/javelin/attack-rating builds | Defiance (+defense) keeps merc and player alive; Blessed Aim does nothing for casters; Prayer heals slowly | auras V; value judgement U |
| Nightmare, Hell | Desert Merc **Might** (Offensive, hired in NM) for physical/cold-immune support; **Holy Freeze** (Defensive) for casters that need slows and have no immune problem; Thorns not recommended | Might raises merc and party physical damage, answers cold/fire/lightning immune packs for caster builds; Hell cannot rehire Might (see 1.1) | auras V; picks S (Maxroll, prior-art 04/01), U for ranking |
| Alternative | Act 5 Barbarian (561), 9000/18000/32000 factor, Bash+Stun, 1hs; tanky physical | no aura; only for physical-damage builds or cold-immune breaking | table V, use U |
| Not recommended | Iron Wolf (359): dies to missiles, 1000-27000 factor | caster merc has no tanking | prior-art 01, `docs/plan/data/mercs.json` (R) |

Equipment slots by merc (S Hive Workshop): Rogue bow + helm + armor; Desert spear/polearm/javelin + helm + armor; Iron Wolf sword + shield; Barbarian sword + helm + armor.
A merc keeps a basic weapon when stripped. Hiring a new merc while one lives destroys the old merc's gear (S Maxroll, D2R text; V Hive says a new hire loses the old gear): strip first.
Merc level never exceeds the player's level (S Hive Workshop); stat rows (V tsv, expansion block, Act 2 Defiance): level 43 HP 630 dmg 24-31, level 75 HP 1430 dmg 48-55 in Normal rows; Nightmare/Hell rows start a little lower (-5% HP).
Gear priorities for a bot with little gold: any cheap helm + armor from the floor first (merc dies mostly to missing defense/HP, U); weapon second; runeword gear (Insight, Infinity, Fortitude) is a farm reward, not a goal (rune cost; Infinity needs Ber+Mal+... U).

### 1.3 Hiring cost and revive
- Hire: pay the price in the window (see 1.1). One click on the highlighted row hired at once, no confirmation (V docs/town.md). A living merc seems to block a second hire: three coded hire attempts with a living merc opened the window and spent nothing (V observation, cause U).
- Revive: talk to ANY act's hireling NPC (Kashya, Greiz, Asheara, Qual-Kehk, and Tyrael in Act 4) and pick the resurrect line (S: Hive Workshop says Tyrael revives in Act 4; Maxroll says every special NPC can revive any merc type). This corrects `docs/progression/merc.md` ("Act 4 has none: go to Act 3"): Act 4 CAN revive via Tyrael (S, not yet live-checked; U for the dialog line).
- Revive price (S, Maxroll, **D2R** page, formula plausibly unchanged): `min(int(hlvl*hlvl/2)*15, 50000)`. Merc level 30 = 6750, 43 = 13860 (V arithmetic: int(924.5)=924 x15), 57 = 24360, 75 = 42180 (int(2812.5)=2812 x15), 82 and up = 50000 cap. Plan gold: at merc level 75 every death costs about 42k.
- If the player dies, the merc dies too (S Hive Workshop); a revive trip is needed after every player death too.
- Merc heal: shift+use a belt potion, or drag a healing/rejuvenation potion onto the merc portrait; wells, town healers and Paladin Prayer help; mana potions are useless to mercs; they regenerate slowly (S Hive Workshop). Rejuvenation for emergencies. Whether our `input` helper can send shift+key is U (docs/progression/merc.md).

### 1.4 Reading merc state from memory (what our code has, what is missing)
| Need | What exists | Status |
|---|---|---|
| Unit identification | the merc is an ordinary monster unit in the server unit table. Classes: Rogue 271, Desert Mercenary 338, Iron Wolf 359, Barbarian 561 (V hireling.tsv Class column). `town.isMercClass` also lists 560, which no hireling row uses (U, harmless but suspect) | class table V; live: Rogue 271 hired from Kashya reads class 271, AI `Hireable`, state 105 (V docs/friendly.md); Desert Merc spawn seen live (V docs/town.md) but its class number was not recorded (U, expected 338) |
| Owner | `Unit.OwnerType/OwnerID` at unit +0x94/+0x98 read **0/0 for the hired Rogue** (V docs/friendly.md). So "owner == player" does NOT find the merc (this refutes the D2BS `getMercenary` assumption in docs/progression/merc.md) | V |
| Friendly/never attacked | `combat.IsFriendly` via monstats AI `Hireable` and the hireling class list | V for Rogue, U for 338/359/561 |
| Alive/dead | `Unit.Alive()` (mode not death/dead). A merc that died may vanish from the table after its death animation (U); Kolbot's rule is "no unit three reads 100 ms apart" (S docs/archive/prior-art/05) | U live |
| Life | `MonsterInfo.LifePct` from stat 6/7; V only for town NPCs (128/128), U for damaged monsters (comment in units.go) | needs a live check on a hurt merc |
| `town.HasMerc(snap)` | "a merc-class unit within 40 subtiles of the player is alive" | works as a proximity heuristic only; a merc left behind in an earlier area counts as dead |
| Level, aura, gear | none. Aura/level are visible only on the hire window (no memory reader; V docs/town.md); merc equipment panel unverified | gap |
| UI | `d2.UIMerc` = 0x24, a UI flag; whether the merc panel sets it is U | U |
| Dead flag on the player | not found. docs/progression/merc.md guesses a `pMercDead`; no verified location | gap |
| `resource.Input` | `MercKnown`, `MercLifePct` exist; `bot/resource.go` "leaves merc fields zero" (V code comment), so nothing feeds them yet | wiring gap |

Proposed live probe for a d2-live agent (no code here): hire the Rogue, then log class, mode, LifePct, states, owner words, and the position relative to the player while (a) standing, (b) hit by a monster, (c) after it dies (does the unit stay, which mode, when does it disappear), (d) after a town portal trip (does the unit respawn at once or follow).

### 1.5 Revive handling in the bot
Existing: `town.Revive` is a skeleton returning `ErrMercUnverified`; `resource` has MercDeadPack (leave a pack fight when merc dead and >= N hostiles). Needed (in `docs/progression/merc.md` section 2 too): chore after repair, gold check against the revive formula above, verify merc unit appears within about 2 s and gold fell. Treat the revive price as a farm-budget item: a goal like "farm until gold >= 2 x revive cost" is a valid precondition for any deep run.

## 2. Farming after the campaign

The planner already has `rune_farm_countess` and an endless-farm fallback (docs/goals/planner.md); the boss runs are in `internal/plandata/runs.json` and boss procedures in `docs/progression/bosses.md` (coordinates there are Kolbot, S/U). This section adds numbers and risks.

### 2.1 XP model (V formula, my arithmetic from our tables)
Source: Amazon Basin Experience page, https://www.theamazonbasin.com/wiki/index.php/Experience (V fetched): base XP = monster level XP value (monlvl.txt `L-XP` column for the monster's level in that difficulty) x monstats `Exp` / 100; unique/super-unique/minion x5 (+3 monster levels); champions x3 (+2); extra players raise XP; level-difference penalty: for 24 < clvl < 70 only higher-level monsters are reduced by clvl/mlvl; for clvl >= 70 XP is multiplied by the `ExpRatio`/1024 column of experience.txt (V: 736/1024 at 75, 256/1024 at 85, 61 at 90). Death penalty is only Nightmare 5%/Hell 10% of the next level's requirement (V same page; "recover 75% by fetching the corpse" is the same page). Ceiling 8,388,607 XP per kill.
Boss XP per kill (my computation, rough: ignores the clvl<25 table and any random boss modifiers; U on the second decimal):

| Boss (difficulty, mlvl) | clvl 30 | clvl 45 | clvl 55 | clvl 65 | clvl 70 | clvl 75 | clvl 80 | clvl 85 |
|---|---|---|---|---|---|---|---|---|
| Mephisto N (26) | 10.7k | | | | | | | |
| Mephisto NM (59) | | 183k | 224k | 241k | | | | |
| Mephisto H (87) | | | | | 1.10M | 826k | 556k | 287k |
| Baal N (60) | 108k | | | | | | | |
| Baal NM (75) | | 972k | 1.19M | 1.40M | | | | |
| Baal H (99) | | | | | 4.32M | 3.26M | 2.20M | 1.13M |
| Diablo H (94) | | | | | 2.09M | 1.58M | 1.06M | 549k |
| Andariel H (75) | | | | | 535k | 403k | 272k | 140k |
| Pindleskin H (88 incl +3, x5) | | | | | 277k | 209k | 141k | 73k |
| Eldritch H (82) | | | | | 180k | 135k | 91k | 47k |
| Shenk H (83) | | | | | 289k | 218k | 147k | 76k |

Next-level XP needs (V experience.tsv, difference of the Amazon column): clvl 30 -> 31: 904k; 45 -> 46: 3.29M; 55 -> 56: 7.80M; 65 -> 66: 18.5M; 70 -> 71: 28.4M; 75 -> 76: 43.7M; 80 -> 81: 67.3M; 85 -> 86: 103M.
Read: Hell Baal kills alone, at clvl 75 about 13 kills per level, at clvl 85 about 90 kills per level (the 256/1024 ratio bites). Pindleskin/Eldritch/Shenk are 6x to 15x smaller per kill than Baal and only worth it for drops, not XP. Packs, waves and minions add to every row (not included).
Run time per kill is the missing number (U): my estimates for a teleporting caster are Pindleskin 1-1.5 min, Eldritch+Shenk 2-3 min, Countess 3-5 min, Andariel 2-3 min, Mephisto 2-4 min, Baal waves + boss 8-12 min. Replace by measured telemetry (section 3, item 8).

### 2.2 Best XP run by level (Hell/NM/Normal)
| clvl band | Difficulty | Best XP source | Why | Tag |
|---|---|---|---|---|
| 1-30 | Normal | normal progression (campaign) | the planner's default | repo |
| 30-45 | Normal then NM | finish Normal; NM Act 1-2 areas, NM Mephisto at the end (183k at clvl 45 = 5.6% of a level per kill) | high mlvl NM bosses beat Normal ones | table V |
| 45-65 | Nightmare | NM Baal (1.0-1.4M per kill = 25-30% of a level at 45-55, 8% at 65) and NM Mephisto/Diablo; Cows in NM at clvl 45-60 (cows have level about 45-55 in NM; U) | Baal is the best XP per kill by a wide margin | V arithmetic; cows U |
| 65-85 | Hell | Hell Baal (4.3M at clvl 70, 3.3M at 75) is the XP boss; Hell Mephisto/Diablo second | | V arithmetic |
| 85+ | Hell | Baal remains best, but the 25% ratio makes each level a 90-kill grind; level 90+ needs about 500+ kills per level (U) | | partly V |

Entering Nightmare/Hell gates: clvl floors in `runs.json` (Baal 32 N / 58 NM / 82 H; Mephisto 28 / 55 / 75; those are Kolbot-style floors, U). The planner already blocks NM/H entry until a resist/gold gate passes (docs/goals/planner.md).

### 2.3 Best drop run by goal (V treasure classes from our tsv)
- **Runes: Countess** (area 25, Tower Cellar 5; WP Black Marsh area 6 then Forgotten Tower 20-25; V runs.json). TC V: Normal "Countess Rune" = 3 picks, NoDrop 5 vs 15, group "Runes 4" (top rune r07 Tal-class per the group chain); Nightmare group "Runes 8" (r15 Hel / r16 Io on the top slots); Hell group "Runes 12" (r23 Mal / r24 Ist top slots, then chains into Runes 11 with weight 1066 vs 5: so about 0.5% of the rolls reach r23/r24 per roll). Rune numbering: El=r01 ... Ral=r08, Hel=r15, Io=r16, Mal=r23, Ist=r24 (U from memory of the rune order; check against `runes.tsv`). Expected 2.25 rune rolls per kill, so in Hell Countess gives about 1% chance per kill of Mal+ and most kills give Shael-to-Lum range runes (U on that range). Needs 1 teleport-free walk, no portal; Countess kills are the standard rune farm for Insight (Ral Tir Tal Sol) and Spirit (Tal Thul Ort Amn).
- **Uniques/set: Mephisto (Durance 3), Baal, Pindleskin**: Mephisto and Baal TCs are the "Act 4/5 Equip + Good" chains (V tsv: picks 7, nodrop 15 each, group guaranteed drops `ceh`/`fed` chests in Hell). Pindleskin uses `Act 5 Super Cx` (V: picks -4, TC level 96 in Hell, 66 in NM, 38 in Normal; "Uitem C"/"Cpot C" chains). The bot should use Pindleskin for fast MF-style runs (no loop beyond one arena) and Mephisto/Baal when it wants runes and the larger unique pool (S, keengamer/diablobytes summaries via search; no hard rates found).
- **Andariel (NM/Hell)**: S Almar's list says best place on Nightmare to farm Stone of Jordan etc.; Hell resists V monstats: cold 50, lightning 50, poison 80, fire -50, so a fire/physical build kills fast, poison-based builds are slow. Poison cloud and melee poison are the failure risk (runs.json: carry antidotes; docs/progression/bosses.md).
- **Cows (cow level, area 39)**: needs Wirt's Leg (Tristram cairn stones, V runs.json note) and a Tome of Town Portal in the Cube in the Rogue Encampment; requirement of "kill Diablo in that difficulty first" is U (open question in docs/goals/README.md). Danger: a very dense pack, the Cow King's room is "bad" in Kolbot and killing him ends cow portals for that difficulty (V runs.json note). Best for gold/XP clvl 25-50 Normal/NM and for mass white/blue drops (runes are low-chance).
- **Eldritch and Shenk (Frigid Highlands 111 / Bloody Foothills 110, Act 5)**: V superuniques.tsv: Eldritch = `Megaflow Rectifier` (class minion1, id 49), Shenk = `Siege Boss` (overseer1, id 42), Pindleskin id 52 (strings.tsv names V). They use `Act 5 Super Cx` family TCs (same as Pindleskin). Frigid Highlands is one waypoint hop from town in the same trip as Pindleskin; Shenk is Normal-quest-relevant (Rite). Both are short but poor XP (above).

### 2.4 Requirements and risks
| Run | Needs before | Teleport | Portal to town | Failure risks |
|---|---|---|---|---|
| Pindleskin | Act 5, Nihlathak rescue (Prison of Ice) in that difficulty so Anya opens the red portal (S/U) | helps, not required (arrival beside him, V runs.json) | yes (back to Harrogath) | revived vipers, Cold/Fire enchant in Hell (U), do not enter Halls of Anguish |
| Eldritch+Shenk | Act 5 WPs Frigid Highlands/Bloody Foothills | helps | yes | Shenk's pack of siege units, Hell elite packs |
| Countess | Act 1 WP Black Marsh, clear Forgotten Tower to Tower Cellar 5 | helps a lot (5 levels, random layout) | yes | Hell: fire/poison on her pack (dark stalkers), Nightmare lightning-enchanted packs; random layout; low mana |
| Andariel | Act 1 quest chain done | helps | yes | poison, 80% poison resist in Hell for poison builds |
| Mephisto | Act 3 WP Durance 2 | strongly (Durance 2 -> 3 is a short corridor) | yes | Mana-burn minions in Hell, Attacking2 ranged cast, lightning/ice cast; the moat trick is optional (bosses.md) |
| Baal | Act 5 complete in that difficulty, Throne WP | strongly | yes (portal allowed in the Throne) | wave 3 hydras, cold-immune waves (runs.json: "Might merc"), Souls/Dolls, Hell clvl floor 82 |
| Cows | Wirt's Leg + Cube | optional | yes | pack density, Cow King |
Common: death penalty only NM/H (V above); the planner's chicken thresholds and merc-dead escape rules (`resource/abort.go`) apply. Always revive the merc before a Hell boss run (revive price 42k at merc level 75).

### 2.5 Suggested farm order after the campaign (my recommendation, U on timing)
1. Normal done at clvl ~32: go to NM (planner gate), hire NM Might at the first Lut Gholein visit.
2. NM: Countess for runes (Insight/Spirit/Stealth set), NM Mephisto and NM Baal when clvl >= floor; this is also the XP route to clvl 60+.
3. Hell: Baal first (XP + drops), Mephisto and Pindleskin rotations for drops once the XP ratio makes Baal cost more than Pindleskin in time per useful reward (clvl 85+).
4. Cows only as a gold/XP interlude for clvl 25-50 and when gold < 2 x revive price.

## 3. Prioritised work list for d2-worker agents (none done)
1. **Merc live probe (d2-live, not worker)**: log merc class, mode, LifePct, states, owner words after hire and after death, plus the post-portal behaviour (section 1.4). Output: a short fixture. Blocks items 2-4.
2. **`d2.Merc` reader** (`internal/d2` or `internal/town`): `Present/Alive/LifePct/Class` from the monster unit by class in {271, 338, 359, 561}, not owner; drop class 560 unless verified; test with `mem.Fake` using the probe fixture. Replaces the proximity heuristic in `town.HasMerc`.
3. **Feed `resource.Input.MercKnown/MercLifePct`** in `bot/resource.go` from item 2 (a one-line wiring); keep the "mercDeadPack" escape.
4. **`town.Revive`**: implement for Kashya (150), Greiz (198), Asheara (252), Qual-Kehk (515), Tyrael (class 245? U, check monstats) with `MercMenu` entries; cost from the formula in 1.3 and `PickLine`; verify by merc unit appearing and gold falling; table-driven offline test with the fake clock.
5. **Hire policy by difficulty** in the task/goal layer: add `task.World.HasMerc`; Normal Act 2 default Defiance (exists as `Act2Merc`), NM Act 2 `Might` (or Holy Freeze for casters), Hell: keep the NM merc and revive; never hire in Hell without a Might shortage; fix `docs/progression/merc.md` and `docs/plan/data/mercs.json` (Hell Might row and "Act 4 has no reviver").
6. **Rune/farm task data**: extend `internal/plandata/runs.json` with per-run `diff`, expected seconds (telemetry once measured), `teleport` flag, and XP/kill (section 2.1 table, recomputed from `monstats`/`monlvl`/`experience` at plan time, with `Exp`, `L-XP`, `ExpRatio` as inputs). A pure function `ExpectedXP(boss, diff, clvl)` with tests against the table rows here.
7. **Planner `farm.next`** (docs/goals/planner.md): choose by clvl band (section 2.2) and goal (runes -> Countess, uniques -> Pindleskin/Mephisto, XP -> Baal), with the gold gate (revive price) and Hell resist gate.
8. **Telemetry per run**: kills, seconds, drops, XP gained (from the existing event log), per run name, to replace my U run-time estimates; dashboard line "xp/hour by run".
9. **Cow level runner** (low priority): Wirt's Leg pickup, Cube recipe, portal use, pack-density flee; needs the Diablo-requirement check first (U).
10. **Merc gear** (low priority): equipment panel calibration and `hand.Transfer` to merc slots; buy a cheap helm/armor first; runeword gear only as a farm reward (section 1.2).

## Open questions
- Does a hire with a living merc fail, and what exactly does the resurrect dialog line look like (never seen live)?
- Which hireling.tsv block (9/31/55 or 9/43/75) applies to our expansion character, and the true price formula by merc level.
- Does the Act 4 Tyrael dialog expose "Resurrect", and does the dead merc's unit vanish or stay as a corpse?
- Cow level opening condition per difficulty; Anya portal per-difficulty requirement.
- Hell Countess usable rune ceiling in practice; need a measured sample (rune counts per 100 kills).
