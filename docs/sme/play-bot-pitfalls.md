# Play patterns 4: what automation gets wrong in Diablo II, and where our bot differs from a human

Target: classic Diablo II LoD 1.14d, offline single player. Not D2R. Written 2026-10-29 by SMEKB-C.
Tags: **V** = fetched or read this session (URL or repo path), **S** = secondary or snippet, **U** = memory or inference.
`[D2R]` = D2R-only or unknown in 1.14d.

Sources:
- KI = Kolbot issue tracker, read with `gh issue view N -R blizzhackers/kolbot` (V, issue numbers given): https://github.com/blizzhackers/kolbot/issues
- KB = Kolbot default sorceress config: https://raw.githubusercontent.com/blizzhackers/kolbot/master/d2bs/kolbot/libs/config/Sorceress.js (V)
- SP = Kolbot SoloPlay FAQ: https://bhdocs.github.io/SoloPlay/faq (V)
- BOTTY = D2R pixel bot forks (aeon0/botty): search snippet only (S)
- PA = our prior-art notes, from reading Kolbot, SoloPlay, Kolbot-Go, Botty and TY source: docs/archive/prior-art/lessons.md, 05-town.md, 06-loot.md, 07-safety.md (V local; the underlying source reads are theirs)
- SA = docs/soak-analysis.md (Soaky 3.5 h no-cheat soak, V counted from logs)
- G1 = docs/g1-log.md (V), DS = docs/deathspiral.md (V), SF = docs/stallfatal.md (V), OF = docs/open-findings.md (V), KN = docs/knobs.md (V)
- HUM = the sibling files: play-act-route.md, play-economy.md, play-safety.md in this directory.

## 1. Lessons from other bots (external, V unless marked)

### Stuck and loop states
- **Difficulty selection loop**: D2BS "highest difficulty" in single player clicked the unavailable Hell button forever because the available one was not tested for (KI #102, open-ended loop). Lesson: attempt-and-verify the difficulty, never loop on a control that is not enabled. Our `ensureDifficulty` already verifies and bans after a failed switch (V KN `bot.difficultyBanMin`).
- **Town loop**: "goes back to town to a healer/merchant, back through the TP and then immediately starts over" (KI #245); "infinite tp to town and back" after picking a set item because the inventory lock grid left no free slot to stash it (KI #480, answered with the Config.Inventory comment: all cells locked by default, "if you don't change the default values, the bot won't stash items", V KB). Cause class: a chore whose need check stays true forever. Our parallel: the chores step re-picked every tick (SF section 4, fixed) and the `futileGoldStep` / `futileMinutes` knobs (KN).
- **Stash move failures**: "Ral Rune failed to move to 9,0", only the first two columns used, every trip fights to stash (KI #456); item left floating on the crosshair at the stash (KI #261, answered "reinstall", i.e. UI cursor state not recovered). Lesson: after every place/move check the cursor is empty; if not, put it back or drop.
- **Gem stacking until inventory full**: a new GemHunter script made the druid keep gems until the pack was full, then it could not pick anything (KI #363).
- **Panel/NPC refusing to open**: "Bot cant open trade/repair with certain NPCs" (KI #449); our equivalents `town.npcOpenTries` and "NPC greeting never gave way to the menu" (SA section 4).
- **Fresh-game loops**: "Failed to create game after a while", "Character Refreshing... no character" (KI #482, v. #447): out-of-game state machines need a bounded retry with escalation. Ours: failsafe ladder, `MaxFailsafeRestarts` (SF).
- **Chicken vs town-chicken at death**: with LifeChicken 0 and TownChicken 30, bots died trying to port out, "Failed to go to town"; the fix added a safety check so a bot that cannot TP does not try (KI #373).
- **Chicken thresholds meet temporary life boosts**: a Druid casting Oak Sage at under 50% chickened because max life jumped and the ratio moved (KI #326, chicken 50). Lesson: compute thresholds from current/max at one instant after the buff, and do not run chicken on a stale max.
- **Corpse pickup breaks precast**: after retrieving the body the bot never recast buffs (BO, HS) (KI #368, #344: chickens and deaths with expired BO). Lesson: after any corpse run or area change re-read the active auras/buffs and recast.
- **Boss-specific pathing**: seal packs killed light sorceresses "jumping into the middle of them" (KI #269); quester left to die by a rusher that does not clear the quest spot (KI #435). Lesson: a bot that moves fast (teleport) must stop and check the pack before the next hop; ours walks, which is safer but slower.

### Immunities and attack choice
- **False immunity**: a rare Griswold with poison resist 120 made the sorceress refuse to attack at all, because the bot's "immune" test returned false for fire/cold though both were 0 (KI #441; the maintainer blamed corrupted game data). Lesson: decide immunity per element from the specific resist, not from "any resist >= X"; keep a fallback attack and a no-progress timer.
- **Skill choice with short range**: bots that attack from too far (poison Necro) or teleport before every melee (Enigma Barb/Paladin) waste time or get blocked by walls (KI #446). Lesson: per-skill ideal range and a "walk, do not teleport" rule when the target is near.
- **CPU spike from a hot skill loop**: high CPU with Blizzard (KI #462). Lesson: bound per-tick work.
- **Short-range classes**: `Config.Dodge` doc says "Don't use with short-ranged attacks like Poison Dagger" (V KB). Our kiting is controlled by `combat.kiteEnabled` and ranges (KN).

### Pickit and loot
- **Pickit file not loaded**: Countess runs sold low runes instead of keeping them because the default pickit was no longer auto-loaded (KI #376); a follower picked nothing but gold and potions with the pickit lines commented out (KI #406). Lesson: log the number of loaded rules at start; refuse to start with zero rules.
- **LowGold mode with no pickit**: "if you have less than 500k gold and have not loaded a pickit file, items will be picked to prevent the bot from breaking due to gold" and followers spent most of their time selling in town, ending Hell Act 5 with ~50k (KI #374). Our `loot.poorGoldBelow` (300) and `poorMinSell` (6) are the same mechanism; the same trade-off applies (more selling trips, shorter play time).
- **Crash in pickit**: `unit.code` undefined (KI #413). Lesson: items may lack fields while loading; tolerate nil.
- Kolbot default pickit shapes matter more than the engine; `Config.Inventory` lock grid, `Config.PickitFiles`, `Config.MinColumn` [3,3,3,0], `Config.HPBuffer` are all user data, not code (V KB).

### Game-process and UI
- **Waypoint menu**: with `Config.WaypointMenu = true` a user reported the connection dropping at the waypoint (KI #198); packets or menus can fail. Ours uses panel clicks and waits for the area id (PA 03-pathing).
- **D2BS not responding / not starting after updates**: reinstall everything (KI #490, #412). Process lifecycle is part of the product; ours is in SF and OF.
- **Shopbot delay after waypoint** (KI #397): stock must be re-read after a short wait (stock refresh on town entry, V play-economy section 5).
- **SoloPlay difficulty rule (V SP)**: "the bot will not start the next difficulty with negative resistances; if the bot is more than 5 levels above the minimum and still lacks resistances, it moves on automatically." In Classic the bot advances "after beating Diablo and reaching a level requirement". Lesson from a bot that plays the whole game: resists gate the difficulty, with a level-based escape hatch so it never deadlocks. Compare ours: resist gate is net >= 0 after the penalty (V OUR docs/difficulty.md); `bot.levelMargin` is the slack.
- **D2R Botty (S)**: resolution and language restrictions, a game-length timeout and a "restart D2R when stuck" option (S snippet). Same ladder idea as ours.

### Cross-cutting lessons already recorded in PA (not repeated here)
docs/archive/prior-art/lessons.md section B lists the ten most common failure causes (UI swallowing clicks, lost references, clicks that do not move, portal races, WP timing, unloaded collision, boss ranged attacks, out of resources, inventory full, no-progress loops). Section A lists 15 designs worth copying. Read them with this file.

## 2. What goes wrong most often, by area (consolidated, with our status)

| # | Pitfall | Typical symptom | Human behaviour that avoids it | Our status (V unless marked) |
|---|---|---|---|---|
| 1 | Never buying the kit (TP, potions) because another task owns town time | belt empty, deaths on pure bad supply, 3779 gold unspent over 2 h | A human spends the first gold on TP and potions every town visit (play-economy section 2) | G1 attempt 6: potions used 0, 3779 gold unspent, "no town portal: walking back" x15 (G1). Open: SHOPTP |
| 2 | Fighting a boss under-levelled and without supplies | 181 aborted Blood Raven runs, 0 kills, 11 potions in 358 s | Skip Blood Raven until a merc or clvl ~8+; fight with 4+ potions and a TP | SA R1; BOSSGATE passed in G1 (0 attempts) (G1 criterion 5) |
| 3 | Town loop and futile trips | 12 heal trips after chicken, 11 empty-belt trips with no gold | A human farms gold first when poor | `futile*` knobs, poverty mode (SA, KN); not proven live |
| 4 | Chicken re-entry loop at low life with no potion/TP | re-enter at 25 life every 20 s | Humans heal at Akara (free) then re-enter | SF section 6 fixed (`lowLifeGate`, `noteReentry`); live check pending |
| 5 | No TP: walking home | 10 of 37 trips, 30-83 s each | Always carry a TP | SA; waypoint home added for Act 1 only (SF section 6) |
| 6 | Death spiral at the same spot | 4 deaths in 11 minutes in Underground Passage 1 | Humans do not walk a naked character into a tougher area; they rebuy | DS: gear gate, ban, corpse by value (V logic); attempts 5-6 had 0 deaths |
| 7 | Gate and planner contradict | no eligible area for a gated character, 4 restarts in 5 min | Humans simply go to the weakest area | GATEPLAN merged (G1 attempt 4/5) |
| 8 | Inventory full or items that do not fit | pack full stops pickup; stash move errors | Leave with 20+ free cells; sell/stash first | `loot.freeCellsKeep` 2, `town.tripPackUsedPct` 80, `bot.invFullPct` 85 (KN); stash move verified by memory `town.moveTries` |
| 9 | Pickit too tight or too loose | gold stuck at 0-3 (too tight); sell trips dominate (too loose) | Pick everything worth gold per cell when poor; only keepers when rich | LOOTGOLD (poor mode), G1 attempt 5; loose-mode trade-off U |
| 10 | Equip loop on items the character cannot wear | `Amulet 8.0 -> neck: item N did not reach slot neck` x8 | Humans read requirements before buying | Per-clvl refusal memory added (leveler.md); SA section 4 |
| 11 | Price estimates that are wrong by 50x | Javelin estimate 5, actual 240 | Humans see the price | SA R2: `estimatePrice` wrong; vendor price calibration open (items.md P9) |
| 12 | Belt column semantics | belt full of the wrong type, rejuv in the wrong column | Humans keep hp hp mp rv | `BeltPlan` columns, `resource.retreatBelt` knob; not proven live (V KN) |
| 13 | Quest turn-in never completes | Charsi quest loops every ~3 min; Den "ran but is still needed" | A human reads the dialog | G1 attempt 6 note, SA section 4 |
| 14 | Game or helper process gone | helper dead after 7.5 min, Game.exe killed | n/a for humans | SF sections 2-7, FAILOWN table; kill test PASS in attempt 6 |
| 15 | Panels left open or cursor holding an item | clicks swallowed | Press Esc, `CloseAll` | Esc-menu has no detector (SF table row 2) |
| 16 | Wrong difficulty after Baal | stuck on the new-game dialog | n/a | `ensureDifficulty` verify-and-ban; U: the dialog never appeared for a Normal-only character (V leveler.md) |
| 17 | Level-appropriate area choice errors | farming a trivial area, or an area whose monster level >> clvl | See play-act-route section 6: leave when monster level is 6+ below clvl; XP penalty table | `bot.leaveSlack` 0, `levelMargin` 0; Blood Moor at clvl 8 had no eligible area under the xp floor (G1 attempt 4) |
| 18 | Immunity misread | refuses to attack a monster with one weird resist | Switch skill or skip | not applicable in Normal; `ImmuneWorkaround` logged but skills not cast in Hell (V difficulty.md) |
| 19 | Buff state after corpse/area change | no armor/shield recast | Humans recast Frozen Armor by habit | `combat.buffRecastSec` 150, `auraCheckMs` (V KN); corpse run recast U |
| 20 | Level-up point spending in a fight | wrong order, panel open in combat | Spend when calm | `Do` refuses within 25 subtiles of a hostile (V leveler.md) |

## 3. Where our bot's behaviour differs from a human's (the cross-check)

Evidence: knobs (KN), soak analysis (SA), G1 log, deathspiral, stallfatal, and the three sibling files. "Human" refers to community routines recorded in HUM.

### A. Supplies and shopping
1. **Kit never bought when the quest step owns town time.** V G1 attempt 6. Human: gold goes to TP and potions first, every time. **Largest behavioural gap.**
2. **Gear shop before kit.** V SA section 4: 1175 gold on gear in one visit, then the belt empty 5 minutes later; SA R2 recommends the kit first. A human fills the belt before upgrading gear.
3. **Merc**: our bot can run mercless; merc revive is behind `bot.mercReviveUnverified`=0 (V KN). A human hires Kashya's Rogue at 8-9 and keeps it alive (V HUM safety).
4. **No Thawing or antidote policy beyond `AntidoteNeeded`**: humans prepare per boss (HUM safety).
5. **Gold deposit**: our `stashGoldPerLvl` 1000 is more conservative than the death exemption (HUM economy).

### B. Fighting and safety
6. **Drink thresholds**: ours 55 (life), 35 (rejuv), chicken 25; human/Kolbot 75/40/30 (V KN, V KB). Ours waits later to drink.
7. **Leaves later in melee**: `resource.minEngageLifePct` 40, `fleeLifePct` 50 exist; `retreatBelt` off by default (V KN, G1 turned it on by hand).
8. **Boss attempts under-supplied**: fixed by BOSSGATE; Blood Raven should be optional (V SA R1, HUM route).
9. **Elite mods unread**: the bot catalog "has neither" mlvl nor affixes (V difficulty.md section 4 note); a human avoids Extra Strong/Cursed/Mana Burn/LEB packs in Normal when weak.
10. **No generic "run to the waypoint" retreat**: only town portal or walk home (V SF section 6; waypoint-to-town is Act 1 only).
11. **Corpses**: our bot retrieves by value and falls back to older bodies; the Arreat page says an open character keeps only its oldest corpse (V HUM safety), which may make older-body targets non-existent (U).

### C. Route and pacing
12. **Entry levels later than humans**: NM at 36, Hell at 64, Andariel at 22 (V HUM route table). Safer; ~10 levels more grinding.
13. **Walks Act 3 where casters are told to skip** (V HUM route). The bot needs the quest points though.
14. **Cain, Tristram runs, Countess runes**: the bot skips Cain by default (V campaign.go); humans do Cain for the ring and Tristram XP. Countess runes are in the plan (the NAS at 14).
15. **Levelling by planner table vs by XP rate**: a human moves to the next area when XP per hour drops. Our `leave_clvl` is a fixed table; SA shows Den of Evil gave the best XP at clvl 4-6 (13-18k XP/h) while Burial Grounds gave 2k XP/h; the table put the bot into Burial Grounds anyway (V SA R1).
16. **Waypoints**: humans activate every waypoint on the way; our bot treats waypoint activation as a quest side effect (U; check the WP memory bitmask for coverage).
17. **No area-based XP-penalty awareness**: leaving when monster level is 6+ below clvl is a pure rule (V play-act-route section 6); we use `leave_clvl` instead (U).

### D. Process
18. **Restarts are costly**: 15 restarts (planned save-points) in 2 hours in G1 attempt 6 (V G1); humans never restart.
19. **Fresh game after a chicken** is a feature (fresh Den) for both a human and our bot (V SA).
20. **Quest talk dialogs**: human reads text; our `Menus` rows are mostly U (V quest.md live checks).

## 4. Suggested guard rails (derived from the evidence, U for effect)

1. At every town visit: TP first, belt fill second, sell third, gear last; a purchase whose gold would leave the kit unaffordable is refused (SA R2, HUM economy).
2. Refuse boss runs unless: potions >= 4, TP present, merc alive (or clvl >= 8 and a staff/merc plan), mana ok (SA R1).
3. At game start log: pickit rule count, belt plan, TP count, free cells, merc state (KI #376, #406 style failures show as silent wrong behaviour).
4. Never loop on a chore whose need stays true: after N failures, block it for M minutes and plan something else (already in SF section 4).
5. A watcher separate from the action loop (human reflexes): potions, flee, chicken at 20-100 ms (PA 07-safety; our `bot.watchMs` 50).
6. Treat "no eligible area" and "no target" as planning waits, not faults (DS "Area ban is a planning block").
7. Leave an area when monster level <= clvl - 6 or XP/h falls below half of the best known (HUM route section 6).
8. Add a per-boss prep list (antidote for Andariel, Thawing for Duriel, full belt, merc alive) before the portal step.

## 5. Open questions

1. The Arreat page claims open characters keep only the oldest corpse; the bot assumes the newest/valuable one is retrievable (DS): verify with two deliberate deaths on a throwaway.
2. Whether the human "Cain first" order materially helps XP with a bot (Tristram runs 4-15 per Almar, S) given our fighting speed.
3. Whether a Rogue/Desert merc changes the Blood Raven/Andariel risk enough to move the gate levels down.
4. Kolbot GitHub issue sampling here covered ~25 of 400+; the search term "belt", "merc dead" and "inventory full" found few matches, so belt/merc bugs may be recorded elsewhere (Discord, the discourse forum https://blizzhackers.discourse.group/, S).
5. Median XL / MXL botting lessons: not researched (the brief named them, no reliable primary source found in this session; U).

## Cast-first rules (COMBATLOOP, 2026-10-29, d2-sme veteran review, EXP)

Implemented in `internal/combat/castfirst.go` (see docs/combat.md "Cast first"). Charged Bolt is the clvl 1-12 workhorse; Frost Nova
(clvl 6) once when 3+ hostiles are within 4 subtiles, then bolts; Static Field only for bosses/champions in Normal; stand and cast;
priority shamans/archers/casters, then nearest, then lowest HP in a cluster (existing `Rank`); step back ONLY from a single melee
monster within 3 subtiles at life < 60% and falling, max 2 steps; never kite a pack; melee classes do not kite. Invariant: in any
10 s window with hostiles in range and mana for a cast, casts > 0 (watchdog at 6 s). Not implemented: Frozen Armor recast timing
(existing `Buff`), mana potion at "< 2 casts" with the 2 s gap (existing potion policy decides, U whether it matches), back to a wall.
