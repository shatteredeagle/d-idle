# Mercenaries in classic LoD 1.14d: SME reference for the hire/keep-alive bot (MERCRESEARCH, 2026-10-30)

Target: classic LoD 1.14d, offline single player. Tags: **V** = verified (our extracted real 1.14d table, our own live observation, or a page I fetched this session, URL given), **S** = snippet/secondary, **U** = unverified or from memory. **D2R** marks facts that are D2R-only.
Table = `hireling.tsv` from our extracted MPQ data (`/tmp/claude-1000/gx/cache/hireling.tsv`; loaded by `internal/data`, see `docs/data.md`). It has 120 rows in two blocks of the same hirelings (rows 1-12/25-42/61-78/97-108 and a second block 13-24/43-60/79-96/109-120). The 1.10 hire-level ranges in Arreat/PureDiablo (below) match the **second block**, so I treat that block as the expansion (LoD) one (V by that match, the file has no version column in our extract).
Complements `docs/merc-farm.md` section 1 (hire/revive/memory reading) and `docs/sme/blood-raven-caster.md`; this file adds the hire-level and "which merc when" analysis. Sources:
- ARR = Arreat Summit hirelings page (Wayback of classic.battle.net, fetched): https://web.archive.org/web/2010/http://classic.battle.net/diablo2exp/basics/hirelings.shtml (mirror: https://classic.hiveworkshop.com/diablo2exp/basics/hirelings.shtml)
- AB = Amazon Basin wiki Mercenary page (fetched): https://www.theamazonbasin.com/wiki/index.php/Hireling and https://www.theamazonbasin.com/wiki/index.php/Rogue_Scout
- PD = PureDiablo mercenary guide by Grux (fetched, written for 1.10, republished D2R era): https://www.purediablo.com/strategy/diablo-2-guide-the-mercenary-guide
- MX = Maxroll mercenary mechanics (fetched via the worker host browser, **D2R**): https://maxroll.gg/d2/resources/mercenary
- LIVE = `docs/town.md` section "Mercenary" and `docs/friendly.md` (our live observations).

## 1. The four merc types (V table, ARR)

| Act | NPC (monstats class) | Merc (class) | Skills (V table) | Equipment slots (V ARR/AB) |
|---|---|---|---|---|
| 1 | Kashya (150), Rogue Encampment | Rogue Scout (271). Fire variant: Inner Sight + Fire Arrow; Cold variant: Inner Sight + Cold Arrow | Fire Arrow chance 25% (+5 per level); Inner Sight 10% (it is a cast, not an aura; it lowers monster defense for the party) | helm/circlet, body armor, bow only (no crossbow; no Amazon-only bows in LoD) |
| 2 | Greiz (198), Lut Gholein | Desert Mercenary (338): melee Jab (2 hits, 70% chance +4/level in Normal) plus ONE passive aura by type | Normal: Combat = Prayer, Defensive = Defiance, Offensive = Blessed Aim. **Nightmare: Combat = Thorns, Defensive = Holy Freeze, Offensive = Might. Hell: back to Prayer / Defiance / Blessed Aim** | helm, body armor, spear/polearm/javelin (never throws) |
| 3 | Asheara (252), Kurast Docks | Eastern Sorceror / Iron Wolf (359), 3 variants | Fire: Inferno (60% of casts) + Fire Ball (30%); Cold: Glacial Spike 60%, Ice Blast 24%, Frozen Armor (self buff); Lightning: Charged Bolt 60%, Lightning 30% | helm, body armor, sword + shield (the shield never blocks) |
| 5 | Qual-Kehk (515), Harrogath | Barbarian (561), 1-handed sword | Bash and Stun, each 15% at merc level 28 rising to 50% at 42/58 and 75% at 75/80 (V table rows 97-120); no aura | helm (primal helms OK), body armor, ONE sword |
| 4 | Tyrael (Act 4) | none to hire; revives only (V ARR: "no Hirelings available in Act IV ... Tyrael will resurrect your Hireling but he will charge you") | | |

Aura and skill facts for 1.14d: the Hell Might row does NOT exist (V table, rows 12-14 Hell = Prayer, Defiance, Blessed Aim; ARR and AB agree; MX says Nightmare+Hell have all six, but that is **D2R**). Might/Holy Freeze/Thorns exist only in Nightmare hires (V). Desert aura skill level scales with merc level, not with the player's skills (U details; table: Prayer level 3+, Thorns 5+, Holy Freeze 14, Defiance 3+).
Desert Mercenaries share their aura with the whole party including minions (V ARR note: "most party friendly"). Rogue physical damage is doubled in LoD (S AB Rogue page, known bug fixed in D2R).
Iron Wolf Inferno is bugged to do half damage (S AB). Iron Wolves never swing; they need +skills/FCR gear.

## 2. Hire requirements

| Question | Answer | Tag |
|---|---|---|
| Act 1 Rogue: when can I hire? | "once you have completed the second quest in Act I or have reached level 8" | V ARR |
| 8 or 9? | ARR says level 8; AB Rogue page says "clvl >7" (=8); Maxroll (D2R) says character level 8; the Diablo Wiki Sisters' Burial Grounds page says 9 (cited in `blood-raven-caster.md`). Three sources say 8, treat **8** as the rule; the bot can simply re-check at 8 and try again every level up to 10 | V ARR, V AB, V MX; the 9 claim S |
| Free Rogue? | Completing Blood Raven (quest 2) makes Kashya offer a free Rogue if you have no merc ("do not receive reward if you already employ a mercenary", AB Rogue page). So hiring one for gold before killing her wastes the reward | V AB |
| Desert Mercenary (Act 2) | Hireable on arrival in Lut Gholein, no quest. Our live Normal hire at clvl 13 worked | V LIVE |
| Iron Wolf (Act 3) | On arrival in Kurast Docks; quest 2 reward is a free Iron Wolf if none (MX, **D2R**; AB does not say) | S, D2R |
| Barbarian (Act 5) | Only "once you have finished his quest" (ARR). The quest is Rescue on Mount Arreat (Act 5 quest 2); reward is runes plus Barbarian hires (see `builds-open-questions.md`) | V ARR |
| Difficulty gating | None beyond level: hire prices/levels exist for all three difficulties in each act | V table |

**Hire level.** The merc's level is not the player's: each hire row shows a level in the window. Rules by source:
- 1.10 (ARR/AB/PD): you cannot buy a merc whose level is above your own. Ranges "possible levels at hiring": Rogue 3-7 / 36-40 / 67-71 (Normal/NM/Hell), Desert 9-13 / 43-47 / 75-79, Iron Wolf 15-19 / 49-53 / 79-83, Barbarian 28-32 / 58-62 / 80-84 (V AB). PD: if you visit an act before you reach the top of its range, only mercs equal to your level are offered (S PD). These ranges equal the first row level of the second table block (Rogue 3/36/67, Desert 9/43/75, Iron Wolf 15/49/79, Barbarian 28/58/80), so the table's first row per Id is the minimum hire level (V table, inference).
- 1.11 and later (AB, which is the era of 1.14d): "When you hire a mercenary now, they will be within 5 levels of your character. They also keep up with your level much more easily." Also, hire-screen stats may look wrong until hired (S AB "1.11" section).
- Consequence: **the ranges in the table are the 1.10 picture; in 1.14d the offered level follows the player.** The exact rule (does a clvl-35 player see a Nightmare Desert at level 43+ or at about 30-40?) is U. Block 1 of the table says Nightmare Desert starts at 31, block 2 at 43; I could not settle which applies. **Verify live** by recording the "Lvl" column of the hire window per difficulty/clvl. Do not gate a Nightmare Might hire on clvl 43 until that is read.
- Cap: merc level never exceeds the player level; once they match, the merc gets no XP until you level (V ARR). Max merc level 98 (V ARR). Merc XP: they share kills, get more from their own kills, none from party members' kills but yes from your summons (V ARR).

**Hire price.** Table `Gold` is a factor: Rogue Normal 100 (Fire) / 150 (Cold), NM 6000/7500, Hell 12500/14000; Desert 350 / 7900 / 15000; Iron Wolf Normal 1000 Fire & Lightning, 1500 Cold, NM 9500/12000/10000 (F/C/L), Hell 21000/27000/21000; Barbarian 9000 / 18000 / 32000 (V table). The actual price is shown in the window and was **360 for the Normal Desert factor 350** (V LIVE, clvl 13, gold 20000 -> 19650). The exact formula from factor and level is U (I found no source; searches only return the revive formula). Plan with the factor, then read the number. Hire is one click on the row and one confirm-less purchase (V LIVE). Hiring while a merc lives replaces it: the old merc and his gear are lost (V ARR).

## 3. Mechanics (what the bot must know)

| Topic | Fact | Tag |
|---|---|---|
| Follows | Hirelings follow through town portals, waypoints, acts and difficulty changes; if stuck, take a waypoint/TP or change area. Saved with the character between games | V ARR ("Hirelings follow you through Town Portals, Games, Waypoints, Acts, Difficulty Levels") |
| Dismiss | No dismissal (ARR joke: get killed). Replacing = hire a new one (old retires, its worn items are gone: remove them first via the hireling inventory) | V ARR |
| Death | When you die, the merc dies too. When the merc dies: go to a hireling NPC and pay resurrect (keeps level, XP, equipment) or hire new (lose equipment). Tyrael revives in Act 4. Per MX any special NPC revives any merc type | V ARR; V MX (D2R) for any-NPC |
| Revive cost | Rises with level, never above 50,000 (V ARR). Formula `min(int(hlvl*hlvl/2)*15, 50000)` (S MX, **D2R page**, assumed same in 1.14d): level 20 = 3000, 30 = 6750, 40 = 12000, 57 = 24360, 75 = 42180, 82+ = 50000. PD says the peak is reached around level 80 (S, consistent) | V cap, S formula |
| XP on death | In 1.09 and 1.10 mercs do not lose XP when killed in NM/Hell | S PD |
| Heal | Town NPC healers, Paladin Prayer/Holy Bolt, wells, Life regen items, potions. Potion: hold Shift and press the belt key or Shift+right-click a belt potion; or drag a Healing/Rejuvenation potion onto the merc's portrait (top-left, hold the cursor 3 s). Mana potions do nothing (mercs have infinite mana). Poison can kill the merc even in town if unhealed | V ARR |
| Regeneration | Slow natural regen; Act 5 mercs regenerate in about 16 s, others about 41 s (S AB quoting Zath) | S |
| Items | Armor, helm, weapon (slots by type in section 1). Must meet level/Str/Dex requirements; Mana/Energy/Vitality useless; ethereal gear is good (no durability loss); +skills only if the merc already has that skill (S AB); elemental/class +skills do not apply in LoD | V ARR, S AB |
| Equip flow | Press O (or the hireling button on the mini-panel) to open the hireling inventory; you must be near the merc; drop items into the slots like your own inventory. A stripped merc keeps a basic weapon and armor (V ARR) | V ARR |
| Damage limits | Hirelings do 50% damage to Act bosses in Normal, 35% NM, 25% Hell; take 10x damage from end-act bosses (V ARR). Mercs deal 17% vs players (S AB) | V/S |
| HP growth | Level up heals fully; HP/level in the table: Rogue +8, Desert +10 to +40 by band, Iron Wolf +8 to +27, Barbarian +14 to +45 (V table) | V |
| Merc level lag | Keep the merc near the player level: hire cost is constant, but a merc left behind in higher-level areas dies; "Always quickly resurrect them" (V ARR) | V |

## 4. Stat tables (V `hireling.tsv`, expansion block; interpolate by merc level)
Format: level: HP / Defense / AR / damage min-max.
| Merc | Normal | Nightmare | Hell |
|---|---|---|---|
| Rogue Fire | 3: 45 / 15 / 10 / 1-3; 36: 342 / 279 / 406 / 9-11; 67: 900 / 744 / 1150 / 25-27 | 36: 308 / 251 / 365 / 8-10; 67: 866 / 716 / 1109 / 24-26 | 67: 779 / 644 / 998 / 23-25 |
| Desert (Def/Comb/Off) | 9: 120 / 45 / 20 / 7-14; 43: 630 / 419 / 428 / 24-31; 75: 1430 / 1027 / 1196 / 48-55 | 43: 567 / 377 / 385 / 22-29; 75: 1367 / 985 / 1153 / 46-53 | 75: 1230 / 887 / 1038 / 44-51 |
| Iron Wolf | 15: 160 / 80 / 15 / 1-7; 49: 466 / 250 / 423 / 18-24; 79: 1006 / 640 / 1143 / 33-39 | 49: 419 / 225 / 381 / 16-22; 79: 959 / 615 / 1101 | 79: 863 / 554 / 991 |
| Barbarian | 28: 288 / 180 / 150 / 16-20; 58: 828 / 480 / 750 / 39-43; 80: 1422 / 1250 / 1520 / 61-65 | 58: 745 / 432 / 675 / 37-41; 80: 1339 / 1202 / 1445 | 80: 1205 / 1082 / 1301 |
The first block (rows 1-12/25-42/61-78/97-108) gives smaller HP growth for the same ids (for example Desert 31: 340 HP, 55: 820 HP) and is probably the classic (non-expansion) data (U). Because the Rogue is the only merc low enough in level for a clvl 8 character, the early-game values are Rogue 3-7, Desert 9-13. The window shows level, life, defense, cost for each row.
Observed early damage: a level 3-7 Rogue does only 1-3 base plus bow damage (bow doubled in LoD); a level 9 Desert does 7-14 Jab plus weapon. HP at level 9: 120 (Desert), at level 3: 45 (Rogue). That is why a hired merc under about clvl 12 is a decoy, not a killer (U judgment, from the numbers).

## 5. What a human does (guide consensus)

- **Who to take.** AB "What Merc for Build X?" (S, forum opinion): casters want a tank, Act 2 or Act 5 melee; Holy Freeze is popular (slows even cold immunes, keeps monsters off you); Defiance if the aim is merc survival; Might is a contender because more damage means more leech; Barbarian is more aggressive and standard auras do not help casters. Melee physical builds: Blessed Aim if AR is short, Might for damage. Summoners: Thorns in NM or vs higher-level monsters, Might in Hell (V AB, S for opinion).
- **Why Act 2 Normal.** Cheap factor (350), arrives at clvl about 13+, aura benefits the party (V table/ARR). PD: "Act II mercs are the most popular because of the aura"; PD also says Desert mercs are "weak but cheap and soak hits at Duriel" (see `play-safety.md` section 7, V).
- **Early game (clvl 8-17).** Humans hire the Rogue at 8 (or take the Blood Raven reward) as a ranged decoy; she is fragile and Normal-only in value; they swap to the Act 2 Desert Mercenary on arrival (V ARR/AB; swap = hire Greiz's merc, the Rogue retires).
- **Nightmare swap.** Only Act 2 NM offers Might/Holy Freeze/Thorns. Because AB (S): "mercs hired in Normal have better stats than those in Nightmare, those better than Hell" (1.10, and PD gives a level 80 example: 1290 vs 1256 life). So only swap for the aura, not for stats. In Hell the Desert auras are Prayer/Defiance/Blessed Aim, so a Hell hire only matters for a fresh level (V table).
- **Act 5 Barbarian** (9000 Normal, 18000 NM, 32000 Hell factors): strong physical melee but costs 25-90x the Desert merc; only for a physical build that needs it. **Iron Wolf**: no tanking, humans use Cold/Fire with +skill gear; a bot should avoid it (S PD/MX).
- **Budget.** Normal Rogue cost factor 100-150 (U: price of low hundreds at clvl 8, read the window); Normal Desert 350 (360 live); NM Desert about 7900-8500 (U, from factor); revive at merc level 30 is 6750, 43 is 13860 (S formula). The real constraint is revive money: a level 43+ merc costs more than a NM Desert re-hire to revive.

## 6. Bot-oriented recommendation (decision rules)

1. **clvl < 8, Act 1**: no merc. At clvl 8 (or after Blood Raven): if Kashya shows a hire row and the free quest reward has not been taken, take the quest reward (kill BR first) rather than paying. Pay only when the bot would otherwise fight BR below clvl 10 as a caster (U; matches `blood-raven-caster.md`). Pick the **Cold** variant for a caster/summoner that kills nothing at range (slows; the Cold factor is 150 vs 100; V table) or **Fire** for more damage; consensus S, no hard data on which.
2. **Act 2 Normal (clvl 13+ or any on arrival)**: hire the Desert Mercenary. Pick: **Defiance** (Defensive) default for casters; **Prayer** if the class has no life regen and potions are scarce; **Blessed Aim** only for bow/javelin/attack-rating builds and Necro/Summoners with AR needs (S AB). The Rogue is replaced; strip her first.
3. **Nightmare, first Lut Gholein visit**: if gold allows (factor 7900, expect about 8-9k, U) and the build is physical or has cold-immune problems, hire **Might**; for casters with a slow-friendly kit hire **Holy Freeze**; skip Thorns. If gold does not cover hire plus revive reserve, keep the Normal merc (he levels with you).
4. **Hell**: keep the merc, revive him; do not hire anew unless dead and unrevivable. No Might in Hell (V table).
5. **Act 3/5 hires**: not recommended (see section 5), but the free quest reward Iron Wolf/Barbarian (D2R, S) should be ignored if a merc exists.
6. **Revive rule**: revive after every fight where the merc died, at any hireling NPC the bot is already near (Tyrael included); gold reserve before deep runs >= 1.5 x revive cost (formula section 3). If gold is under the cost, play without him until the next gold.
7. **Gear**: any helm and armor first (merc dies from missing defense, U), then weapon. Level/Str/Dex must be met (V ARR). Do not equip mana/vitality/energy. A cheap Normal bow/polearm drop with sockets and Amn/Thul is a bonus, not a goal.
8. **Potions**: Shift+belt-key a healing/rejuvenation potion when the merc is under about 50% in a fight (Kolbot uses `UseMercHP` 75, S `play-safety.md` section 8). Never use mana potions on him.
9. **Do not strand him**: after TP/WP/Act change, wait until the merc unit is next to the player (hirelings follow; teleporting casters can leave him behind) before engaging; if he stays missing >10 s, take a WP or re-enter the area (V ARR remedy).

## 7. Memory/state we can read (see `docs/merc-farm.md` 1.4 for the full table)
- Merc is an ordinary monster unit: class 271/338/359/561 in the monster table (V table class column; Rogue 271 verified live, AI `Hireable`, state 105, V `docs/friendly.md`). Unit owner words read 0/0, so find by class, not owner (V friendly.md).
- Alive/dead: `Unit.Alive()` (mode). Life: `MonsterInfo.LifePct` (unverified on a hurt merc). Level/aura/gear: no reader; the hire window is visual only.
- Gaps: hurt-merc life, dead-merc unit behaviour (stays or vanishes?), resurrect dialog line (never seen), merc inventory panel coordinates, hire with a living merc (three live attempts spent nothing, cause U: LIVE).

## 8. Verify live (ordered)
1. Hire window: read level, life, defense, price for every row at Normal Act 1 (clvl 8), Act 2 (clvl 13+), and at the first Nightmare Act 2 visit; this resolves the hire-level question (section 2) and gives the price formula (fit factor, level, price).
2. Free reward vs paid Rogue at clvl 8: does Kashya list hire rows before Blood Raven dies; does the BR reward appear if a merc is employed.
3. Dead merc: kill/die, record unit presence, then the resurrect dialog line at Kashya, Greiz, Tyrael and the price paid versus the formula.
4. After TP, WP and act transition: time until the merc unit is next to the player, from several positions.
5. Shift+key potion on the merc: does our `d2input` send Shift+key (U), and does LifePct change.
6. Merc inventory (key O) cell coordinates for helm/armor/weapon, then `hand.Transfer`.

## 9. Risks
- Paying for a Rogue then losing the free Blood Raven reward (V AB).
- Replacing a merc with gear: gear is lost; strip first (V ARR).
- Nightmare hire price and revive price drain the budget; death spirals (player dies -> merc dies -> 10-40k revive) can bankrupt the character. Gate deep runs on gold >= 1.5 x revive.
- Hell Might is not available: a plan that assumes it fails (V table). The documented source conflict is D2R (Maxroll) versus 1.14d (table).
- Merc blocking narrow doors or running off to wake packs (S PD intro: "mercs will run off and attack monsters your character cannot see").
- Poison kills an unhealed merc in town (V ARR): visit the healer after poison areas.
- D2R-only facts used here for hints: any-NPC revive, quest-gift Iron Wolf/Barbarian, revive formula; tagged accordingly.
