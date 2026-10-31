# 01 Leveling per class, quest/area order, difficulty change, merc

Scope: what Kolbot-SoloPlay (SP, GPL: facts only) and Koolo's leveling chars (KO, MIT) do from level 1 to the end of Hell.
Our character today is a Sorceress, so Sorceress is detailed; other classes are summarised from the SP build files.

## 1. Architecture of a leveling bot (SP)
1. A build is a data object: `stats` list (attribute, target-value pairs, applied in order; `"all"` = dump remainder), `skills` list
   (skill, target-level[, "allow-spending-now" flag]), `wantedskills`, `usefulskills`, `wantedMerc`, `active()` predicate, and an
   `AutoBuildTemplate` keyed by character level that rewrites the combat Config (attack skills, belt layout, potion buffers) at a level.
   `SP:BuildFiles/<class>/<class>.<Name>Build.js`.
2. The active build is chosen from character level, not stored: `CharInfo.getActiveBuild` (`SP:BuildFiles/sorceress/sorceress.js:1-51`):
   - level < respecOne (26)            -> "Start"
   - level < respecTwo (63 expansion / 60 classic) -> "Stepping" (or stay "Start" if Nova already hard-skilled)
   - later -> final build if its `respec()` condition holds, else "Leveling" (generic mid build)
3. Level caps that gate difficulty change (`levelCap`): softcore Normal 33, Nightmare 64 (classic 60), Hell 100; hardcore Normal 36 (classic 33), Nightmare 71 (classic 67).
4. A script list (see section 4) is evaluated each game: each script has `preReq`, `skipIf`, `shouldRun`; first run all that say yes, in list order.
5. Stat and skill points are spent by AutoStat/AutoSkill after every level-up (KB:core/Auto/*). Our equivalent: `bot/progression` reads
   `stat.StatPoints/SkillPoints` (d2 Stats) and clicks the stat window / skill tree. See memory doc 2.4 for stat ids.

Mapping: build tables -> `internal/data` (static build specs, one Go file per class, no MPQ needed); evaluator -> `internal/bot`
(`progression.go`); needs from `internal/d2`: char level, class, free stat/skill points, hard-points per skill (`getSkill(id, hardPoints)`
equivalent), difficulty, quest flags (section 5).

## 2. Sorceress builds compared
All numbers from `SP:BuildFiles/sorceress/*` (expansion unless noted) and `KO:internal/character/sorceress_leveling.go`.

### 2a. Phase table (SP)
| phase | levels | primary | stat order | skill order (hard points) |
|---|---|---|---|---|
| Start | 1..25 | Charged Bolt early, Ice Blast/Ice Bolt, Nova from ~17 | energy 40, vit 15, energy 45, vit 20, energy 50, str 15, vit 25, energy 60, vit 40, str 35, rest vitality | Charged Bolt 3 (lvl4), Ice Bolt 1, Frozen Armor 1, Telekinesis 1, Frost Nova 1, Static Field 1, Ice Blast 1, Static 4 (lvl10), Teleport 1 (lvl~18), Nova 7, Static 6, Ice Blast, Glacial Spike 1, Blizzard 6 and Cold Mastery listed but "never gets here" |
| respec 1 at 26 | | | | points reset, Stepping build applied |
| Stepping | 26..62 | Blizzard (boss+normal), Ice Blast filler, Static/Lightning third | expansion: energy 69, str 48, vit 165, str 61, vit 200, str 100, vit 252, dexterity "block" (to reach block cap), rest vit. classic: energy 60, vit 40, str 55, energy 80, vit 80, str 80, energy 100, rest vit | Warmth 1, Frozen Armor 1, Static 1, Teleport 4, Blizzard 1, Ice Blast 15; then Cold Mastery 1, Frozen Orb 1, Blizzard 20, Chain Lightning 1, Lightning Mastery 1, Lightning 5, Lightning Mastery 20 |
| respec 2 at 63 | | | | |
| Leveling (generic) | 63+ | Blizzard / Ice Blast, Meteor, Fire Ball as immune-switch | energy 50, str 48, vit 165, str 61, vit 200, str 127, vit 252, dex "block" | Warmth, Frozen Armor, Static 1, Teleport 1, Meteor 1, Fire Mastery 1, Cold Mastery 1, Lightning Mastery 1, Nova 1, Frozen Orb 1, Fire Ball 20, Blizzard 20, Nova 5, Ice Blast 15, Meteor 15/20, Cold Mastery 5, Fire Bolt 20 |
| Final builds (need gear) | 64+/71+ | see below | | |
Final builds present: Blizzballer (Blizzard 20 + Fire Ball 20 + Glacial Spike 20 + Meteor 20; active at 71 with full Tal Rasha set, classic: lvl 75 and Diablo dead), Meteorb (Meteor 20 + Frozen Orb 20 + Fire Ball 20 + Fire Mastery 20), Lightning (Lightning 20, Chain Lightning 20, Lightning Mastery 20, Nova 20, Charged Bolt 20; needs Infinity merc; lvl 64+; strength 156 / dex 35), Blova, Cold, Ensorb (not read in detail).

### 2b. KO (D2R) leveling sorc
- Stats: vitality only below level 9; then energy 45, strength 25 (to <15); then energy 60, strength 50, rest vitality.
- Skills below 25: Fire Bolt x4 (interleaved with Frozen Armor, Warmth, Static Field), Telekinesis, then Fire Ball to ~20, Teleport, Fire Mastery at the end. At 25 with Fire Ball > 10: respec to cold (Static x4, Telekinesis, Teleport, Frozen Armor, Ice Bolt, Ice Blast, Frost Nova, Glacial Spike, Blizzard, then Glacial Spike heavy, Cold Mastery x5).
- Combat loop (`KillMonsterSequence`): if mana < 15 percent and level < 15 just use primary on target; else with Blizzard: every second loop, if any enemy within 4 of player cast Blizzard on that one first, then Blizzard on target (range 25..30) then primary x3; without Blizzard: Fire Ball x4 at distance 1..25. Hard cap of attack loops per target (`sorceressMaxAttacksLoop`, value not read). Static Field casts before bosses: 8 on Normal, 6 otherwise (Andariel, Duriel, Mephisto, Diablo, Izual, Baal, Ancients).

### 2c. Takeaways for us
- Both bots are "fire early, cold later" or "bolt/nova early, cold later". The cheap early game for a hand-off idle bot is: put every point into one projectile (Fire Bolt or Charged Bolt), Frozen Armor, Static Field, Teleport at 18/24; vitality dump; respec once.
- Respec is a real cost (Akara in Normal quest "Den of Evil" gives one respec token in Expansion, 1.14d: Akara free respec after Den of Evil only in D2R/1.13+? UNVERIFIED). SP and KO both avoid doing it before 25-26. Our `Tuning` handicap can simply skip the respec and stay on the first skill (slow but works).
- Static Field vs bosses is a required step: it takes a percentage of life (value read in `Config.CastStatic` below: Normal 25, NN 33, Hell 50 percent target life for SP; classic 15). See 04-combat.

## 3. Other classes (SP build files, facts only)
| class | start build primary | leveling/final highlights | merc wanted |
|---|---|---|---|
| Amazon | Charged Strike -> Lightning Strike; str 34, vit 30, dex 47, vit 45, str 47, dex 65 | final builds Javazon (Lightning Fury), Faithbow, Wfzon, Witchyzon, Frostmaiden; Valkyrie as summon | Holy Freeze (A2 Nightmare) |
| Assassin | Fire Blast / Wake of Fire traps; vit 35, energy 35, str 33, dex 33 | Whirlsin, Trapsin; Shadow Master, Death/Lightning Sentry | Holy Freeze |
| Barbarian | Frenzy/Double Swing, Battle Orders; str 35, dex 27, vit 45, ... str 71, dex 49 | Whirlwind, Frenzy, Singer, Uberconc; Lawbringer for fastest leveling | Might (A2 NM) |
| Druid | Firestorm, Fissure; vit 70, str 35, energy 85 | Elemental, Wind (Tornado/Hurricane), Wolf, Stormbear | Blessed Aim |
| Necromancer | Teeth, Bone Spear; str 20, energy 45, vit 70, str 35, energy 85 | Bone, Summon, Poison, Griefmancer (Corpse Explosion) | Might |
| Paladin | Zeal, Holy Fire; vit 80 | Hammerdin (Blessed Hammer + Concentration + Vigor), Auradin, Smiter, Zealer | Holy Freeze |
Interesting: "Start" builds spend almost all early points in vitality, a safe default for an idle bot.

## 4. Quest and area order (SP `Tools/SoloIndex.js`)
Evaluation order of the script list (each has its own conditions; Normal unless noted):
corpsefire(hell only, den done) , mausoleum(disabled), den, bishibosh, bloodraven, tristram, treehead, countess, smith, pits, jail, boneash, andariel, a1chests, cows, cube, radament, creepingfeature, beetleburst, amulet, summoner, fireeye, maggotlair, tombs, ancienttunnels, staff, duriel, lamessen, templeruns, lowerkurast, eye, heart, brain, travincal, mephisto, izual, hellforge, river, hephasto, diablo, shenk, savebarby, anya, pindle, nith, ancients, baal, a5chests.

Key gates (level and flag):
| script | condition |
|---|---|
| den | skip if done, or level 9..(17 for sorc, 11 others); Den of Evil first |
| andariel | requires level >= 11 (kills at level >= 12; "too early, NG" below 12); needs poison res: if < 75 percent go to town and buy antidotes (`antidote:true`) |
| countess | normal: when rune needs (Tal+Eth for Stealth/Ancients Pledge) or broke; others when teleport or level < 60 |
| smith | level > 6 (Horadric Malus, Act1 Barracks) |
| cube | Act2 unlocked and level >= 18 for sorc (wait for teleport); `Halls of the Dead lvl3` chest |
| radament | Act2, quest not done |
| creepingfeature / beetleburst | level 12..20 (xp farm) |
| amulet | `Lost City -> Valley of Snakes -> Claw Viper Temple 1 -> 2`; amulet altar at (15045,14051); walkers use waypoint nodes (15065,14047),(15063,14066),(15051,14066),(15045,14051) |
| staff | Far Oasis -> Maggot Lair 1,2,3, staff chest preset `ShaftoftheHoradricStaffChest` |
| summoner | Act2, Tainted Sun quest completed, `Arcane Sanctuary` WP; gives the Canyon route to Tal Rasha tombs |
| fireeye | level 16..23, only if Summoner not done; Palace Cellar -> Arcane portal |
| maggotlair, tombs | Normal only, level <= 21 / 22, tombs requires summoner done and teleport |
| duriel | needs completed staff or amulet+shaft |
| eye/heart/brain | Act3, skip once Khalim's Will or Travincal done; levels via Lower Kurast/Spider cave/Flayer dungeon |
| travincal | not done; or level < 25; Normal broke; barb/pal special cases |
| mephisto | teleport or non-hardcore etc.; requires Travincal done |
| izual | Act4 reached, not done (or Normal and Diablo not killed) |
| river/hellforge/hephasto | Act4, difficulty-specific level floors: Normal 24, Nightmare 40, Hell 80 |
| diablo | Act4; level floors Normal 24, NM 40, Hell 80 (85 sorc). Normal: do it if level < 30 or classic; else only while difficulty incomplete |
| shenk/savebarby/anya/ pindle/nith/ancients/baal | Act5 expansion only; ancients before baal; Nihlathak only with teleport and not in Hell |
Gold scripts when broke: bishibosh, tristram, treehead, countess, lowerkurast (`goldScripts`).

### Difficulty rule (SP `Core/Globals.js` `Check.nextDifficulty`)
Go to next difficulty when ALL: (1) current diff completed (`diffCompleted`: Baal in expansion, Diablo in classic), (2) `charlvl >= levelCap` of the current diff, (3) not broke (`Check.broken` zero), and (4) resistances after the next-diff penalty are all > 0 for fire, cold, lightning. Or when over-levelled by 2 (Normal) / 5 (others) above the cap regardless of resistance. Resistance penalty (`getResPenalty(diff+1)`): Nightmare -40, Hell -100 in expansion (classic -20/-50) UNVERIFIED (not read, standard D2 values for expansion are 40/100).
Setting the difficulty: SP stores the next one in a data file and the OOG layer picks it on the single-player difficulty dialog (see 08-oog).

### "Broke" logic (`Check.brokeAf`)
Broke when: gold < min(floor(500 + level*100*sqrt(level-1)), 250000), except never below level 15, never in Normal before Act 2, not if gold > half of that and gold > repair cost. When broke: add `[name]==gold` pickit line, rerun gold scripts. In Hell casters that are broke may fall back to Nightmare (`brokeCheck`); melee chars with 0-durability weapon fall back to Normal.

## 5. Quest flags needed
`internal/d2` must expose quest states (memory doc: quest data in the game info / `0x6FDAB...`? UNVERIFIED location). SP uses: `me.den`, `andariel`, `radament`, `cube`, `horadricstaff`, `amulet`, `summoner`, `duriel`, `lamessen`, `eye/heart/brain`, `khalimswill`, `travincal`, `mephisto`, `izual`, `hellforge`, `diablo`, `shenk`, `anya`, `ancients`, `baal`, `cows`, `tristram`, `smith`, `diffCompleted`. These are quest bit arrays (16 quests per act x 3 difficulties); `internal/d2` quest reader is a wave-2 prerequisite. Fallback without quest reader: infer from waypoints and `accessToAct` (what acts have town waypoints active) and from inventory item ids.

## 6. Gear tiers (SP tier system)
Items get a numeric tier by a scoring expression (`tierscore(item)`), separate slots: `tier` (main), `secondarytier` (weapon swap, e.g. charged Weaken/Lower Resist wands = 25000/50000+), `charmtier`, `Merctier`. Rules: runewords and uniques get fixed high tiers (e.g. 100000 for final-gear lines, 110000 for perfect variants); generic leveling uses rarity filters ("magic or better, non-ethereal, one-hand weapon + shield" for a caster). An item is equipped when its tier exceeds the worn item in that slot and strength/dex requirements are met. Items that become unusable (str/dex requirement not met, broken ethereal, quest item equipped) are removed at start of each game (`SoloPlay.js` startup loop).
Our minimum viable: score = base defense + sum of resist values + FCR/skills weights, equip if score > current and requirements met; reuse NIP-like rules from 06-loot.

## 7. Mercenary handling
- Preference (SP `Mercenary.js`): Act2 Desert Mercenary with aura "Holy Freeze" (Nightmare tier) for casters; Normal: "Defiance" Act2 (Prayer/Blessed Aim/Defiance are the Normal auras; Holy Freeze/Might/Thorns are NN). Act1 Rogue (Fire/Cold Arrow) only before Act2 is reachable and only if Sisters' Burial Grounds quest done. Act3 Iron Wolf (Fire/Ice/Lightning) only for casters wanting spells. Barbarians want Might.
- Hire conditions: no hire if gold < ((level-1)^2/2)*7.5 (their affordability heuristic), if already have wanted skill, if `me.diff > wanted difficulty` and so on. Re-hire loop: go to merc NPC (Kashya A1, Greiz A2, Asheara A3, Tyrael? A4 no merc, Qual-Kehk A5), strip merc gear first, open the hire menu, read all offers via the merc list packet (UNVERIFIED for external reading; our version must scrape the NPC dialog or read the offered merc list from memory) and pick the highest-level offer with the wanted skill; skip if cost > gold and remember `minCost`.
- Revive: Kashya/Greiz/Asheara/Qual-Kehk "Resurrect" option, repeated up to 3 times, waiting `max(750ms, 2*ping)` after each click; abort on "not enough gold" text.
- Merc gear: separate tier list for helm/armor/weapon (Insight/Infinity/Treachery... rune words: files `BuildFiles/Runewords/Merc*.js`).
- Merc potions: health potion given by shift+potion key (KO) or `Packet.useBeltItemForMerc` (KB); thresholds: UseMercHP 75, MercRejuv 0 (off by default), MercChicken 0 (KB defaults); KO default mercHealingPotionAt 80, mercRejuvPotionAt 30, mercChickenAt 10. Intervals: heal 6s, rejuv 2s (KO).
- `MercWatch`: if merc dies mid-fight and gold allows, go to town and revive immediately (KB:Attacks/Sorceress.js doAttack).
- Classic LoD: merc does not exist in classic (no expansion). Our offline 1.14d with expansion has them (character is "expansion"? UNVERIFIED for our char).

## 8. Mapping to our packages
| need | package | data |
|---|---|---|
| build tables (stat order, skill order, respec levels) | `internal/data` (hand-written Go), consumed by `internal/bot/progression` | skills.txt ids/prereqs from MPQ (internal/data) to validate prerequisites |
| spend points | `internal/input` clicks on stat/skill panels | UI-open flags (mem 2.11), panel button coordinates at 800x600 (calibrate live) |
| next-difficulty decision | `internal/bot/goals glue` | level, resist values from stats (resist ids), difficulty, quest done flags |
| gear scoring | `internal/bot` | item stats list per item (mem 2.7), base item table from `internal/data` (armor.txt/weapons.txt: reqs, durability) |
| merc hire/revive | `internal/bot/town` | merc unit (monster unit with owner), merc HP percent, NPC dialog menu positions |
