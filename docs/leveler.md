# leveler: points, gear and the leveling loop (agent LEVELER)

Code: `internal/leveler` (point spending, gear scoring, equip, shop), `internal/leveler/townvendor` (adapter to the Townsman's vendor hooks),
`internal/bot/leveling.go` (loop integration: points, gear pass, starvation, runs, difficulty), `tools/rigs/levelerlive` (live probes),
`internal/env/tuning_bot.go` (knobs). Tags: VERIFIED LIVE = observed on instance i4 (Booti, Sorceress, level 2, 2026-10-28); UNVERIFIED = not.

## Stat and skill points (`leveler.Do`, `bot.Leveler.Spend`)

- **Detect**: `Player.StatPoints` / `SkillPoints` (stat ids 4 and 5). The HUD buttons are UI flags 6 (stat) and 7 (skill), both set while points are
  unspent and both clear after the last point (VERIFIED LIVE, screenshots before and after).
- **Plan** (`Build.Plan`, pure, `plan_test.go`): the build tables `docs/plan/data/build-*.json` (embedded copy `internal/plandata`, a test keeps them equal)
  become an ordered list of "wants" (per level row: str, dex, vit, ene points, then skills). What the character already owns is subtracted (base attributes from the
  base stat vector, so item bonuses are not counted as spent; skill base levels from the skill list), the next `unspent` wants are the plan. A rerun after a partial
  spend, a hand-spent point or a quest point therefore continues where the table says; quest points pull future rows forward (quest-points.json policy).
  A skill is skipped (with its later points) when `skills.txt` says the game would refuse it (required level, `reqskill`s, max level) or the game refused it earlier this level.
- **Build**: default `fallback` (Lightning/Chain Lightning, no respec). `primary` (Blizzard) has rows after clvl 26 that assume Akara's respec, which the bot does not
  do (UNVERIFIED trigger quest); set `"levelerBuild": "primary"` in the bot config only if you do the respec by hand. Rows up to 25 are identical.
- **Click path** (`run`): `c` opens the character panel (flag 0x02), then one left click per point on the red + button beside the attribute; `t` opens the skill tree
  (flag 0x04), a click on the tab of the skill's page, a click on its icon. Every point is proved by re-reading memory (unspent count fell); 3 tries per stat point,
  2 per skill; a refused skill is remembered for the level and the plan is recomputed (up to 4 rounds). Panels are closed afterwards (`CloseAll`, checked).
- **Geometry** (800x600, measured on a game-window screenshot; identity between screenshot and client coordinates VERIFIED by the clicks landing):
  stat + buttons at x 220, y 145 (str), 210 (dex), 294 (vit), 354 (ene). Skill tabs at x 680; `skilldesc` SkillPage 1 = Fire is the BOTTOM tab (y 420), page 2 =
  Lightning (320), page 3 = Cold (218). Icons: column x = 437 + 70 (col-1), row y = 95 + 67 (row-1), from skills.txt `SkillRow`/`SkillColumn`.
  VERIFIED LIVE: 5 vitality and Charged Bolt (page 2, row 1, col 2) in one run. Other window sizes UNVERIFIED.
- **Never during a fight**: `Do` refuses (`ErrBusy`) when a living monster is within 25 subtiles outside town; the bot additionally only calls it with `hostileNear == 0`
  and the guard interrupts a long action for it (`errPoints`) at a calm moment. Retry rule: after an attempt that spent nothing the bot waits `bot.pointsRetrySec`, and
  gives up for the level after 3 such attempts (a visible `points not spent` warning each time).
- After a spend that learned a skill the fighter's `EnsureHotkeys` binds it (hook `Options.Rebind`; Charged Bolt went to F4, VERIFIED LIVE in the d2bot run).

### Hover rule (VERIFIED LIVE, matters for every UI click)
The stock `Click` (move, short settle, press) does not switch the NPC vendor tabs, and a click that places an item into an equipment slot behaves the same way:
the game needs the pointer to have been over the button for a few frames. `hoverClick` = move, 450 ms, click, 350 ms works (Akara's Weapons/Misc tabs: grid content
hash changed after one hover click, never after the stock click in 6 tries). Stat + buttons, skill tabs inside the skill tree and right-click buying in a vendor grid
work with the stock click, but the leveler uses hoverClick for every tab anyway.

## Gear (`Candidate`, `ChooseGear`, `PlanBuys`, `Equip`, `Shop`)

Scoring (score.go, constants at the top): armor = average base defence (+15 % superior, +25 % ethereal); weapons = 0.4 x average damage, +8 for staves, wands and orbs;
identified items add their stat list (all skills 40, class skills 30, tab skills 25, resists 1.2 each, life 0.6, mana 0.3, fcr 1.5, frw 0.4, str/dex 0.4, vit 1.0, energy 0.5,
magic find 0.15; listed defence replaces the base); unidentified items get a flat expectation by quality (magic 8, rare 20, set 25, unique 30, crafted 18).
Everything is "defence equivalents"; the stat ids on items are the standard ItemStatCost ids, UNVERIFIED except 0-3 and 7-9.

Usability: class restriction (itemtypes `Class`), strength, dexterity and base level requirements, low quality weapons, and **magic or better items count as unusable
until the character level reaches their item level**: affixes raise the required level and memory does not expose it (VERIFIED LIVE: a magic Cap and Wand, ilvl 7,
sat red in the pack of a level 2 character; the equip click did nothing; a magic Wand cost 1669 gold before this rule existed). A swap the game refuses is remembered
per character level and not retried.

Choice (`ChooseGear`, pure, tested): per slot the best usable candidate that beats the worn item by 10 % + 1 point; two-handers are never planned over a worn shield
and shields never over a two-hander (no swap needs a second unequip); rings fill empty slots, then replace the weaker worn ring.

Equip (`Equip`, VERIFIED LIVE): open the pack, take the item (cursor holds it, checked), hover-click the slot rectangle from inventory.txt (all 10 slot centres matched
the screenshot), prove the item is worn at that body location, put a displaced item back into the pack. A Wand replaced a Spear this way.

Shop (`Shop`, `PlanBuys`; VERIFIED LIVE on Charsi and Akara with cheat gold): `Vendor` (adapter over the Townsman's `OpenForTest`/`PickEntryForTest`/`Stock`) opens the
trade window, the stock is rated like pack items, `PlanBuys` takes the best gain per gold within budget (items in the pack count as worn first, each pick counts as worn
afterwards), the vendor tab is hover-clicked, the item right-clicked, and the purchase proved by "gold fell and the pack grew". Prices are not readable: `estimatePrice`
(base cost x quality) is only a budget guess (real: normal items about the base cost, a superior Wand 235 vs 205, a magic Wand 1669 vs 205; the magic rule above keeps the
bot out of the expensive class until it can wear it).

Bot side (`gearStep`): in town, at most every `bot.gearCheckSec`: equip when `PendingSwaps > 0`; shop when gold >= `bot.gearMinGold` once per character level (or when gold
doubled) with budget (gold - `bot.gearReserve`) x `bot.gearBudgetPct`/100. Errors are logged, never escalated. Loot pickup stays with the Looter: armor and weapons it picks
up land in the pack and are rated at the next town visit.

## Progression, runs, poverty, difficulty (`bot/leveling.go`)

- **Planner**: `reach_level` already picked an area per level from `progression-<difficulty>.json` (docs/bot.md). New: `LevelRun` (Kind clear | quest | boss) built per picked step by `runFor`:
  Normal Den of Evil = quest run `den_of_evil`, Burial Grounds = boss run `blood_raven` (the sweep clears it; the boss is not scripted), everything else a clear run.
  A finished quest run queues the reward claim; in town (or after a trip home) `Quester.Claim` runs once per clear (adapter: talk to Akara, success = unspent skill point rose; UNVERIFIED).
  The Den is skipped while the belt holds fewer than 2 healing potions (a dungeon quest needs a buffer).
- **Starvation / poverty**: a character with no healing potions and gold below the poverty threshold (`resource.poorGold` / `resource.kit*` / `resource.povertyMargin`; the old `bot.starveGold` and `bot.starveRecoverGold` knobs were removed, they had no reader) is starving: the planner sends it to the poverty farm area (`starveArea`) until the kit is affordable (or potions), then back to its plan. Potion trips are not made with less than 30 gold (cheapest potion, VERIFIED LIVE);
  a poor character with `bot.sellMinItems` sellable pack items makes a sell trip. Death/poverty rebuy is the existing death policy (docs/bot.md "Death").
- **Difficulty**: `wantedDifficulty` (auto, reach_level): the highest difficulty whose first plan step the level qualifies for (+ `bot.levelMargin`), never down, not within
  `bot.difficultyBanMin` of a failed switch, not with `PauseDeathsPerHour/2` deaths in the last hour. The switch is `Cycler.NewGame(difficulty)` through the OOG difficulty
  screen (UNVERIFIED: it never appeared for a Normal-only character); afterwards the real difficulty is read and a switch that did not happen bans the wish. The game only
  offers Nightmare/Hell to characters that finished the previous difficulty, which the bot cannot read, hence attempt-and-verify.

## Knobs (internal/env/tuning_bot.go)
`bot.pointsRetrySec 30`, `bot.gearCheckSec 60`, `bot.gearMinGold 150`, `bot.gearBudgetPct 70`, `bot.gearReserve 100`,
`bot.sellMinItems 4`, `bot.difficultyBanMin 60`. Config: `levelerBuild`, `noLeveler`.

## Live tool
`tools/live.sh bot levelerlive <state|tree|items|spend|plan|equip|shop BUDGET [X Y]|opentrade CLASS SECS|goto X Y|exitto N|slots|click|key|tabprobe|trybuy>`; `state` also prints skill pages.
`shop` takes an optional point to walk to first (see below).

## Dead ends and cautions
- `d2cheat gold N` fails on a character without a gold stat; `setstat unit=both vec=both stat=14 value=N insert=1` inserted it (client and server) and `gold N` works afterwards.
- This game's Rogue Encampment sits at another world offset than the table in `internal/town` (Charsi at 5118,4220 instead of 5984,4922: offset -868,-709), so the Townsman walked
  out of town toward wrong coordinates. `cmd/d2bot/adapters.go` `calibrateTown` shifts the table by the waypoint object's offset (workaround, TOWN owns the table).
- `townlive`, `combatlive` find the game with `input.FindGamePID`, which ignores the instance: with i0 stopped their runs can act on another agent's character. Gear bought and
  equipped on Booti while only the leveler was running (Cap, Boots, Heavy Gloves, Light Belt, Spear) points at that.
- The d2input helper of i4 died repeatedly (connection refused on 47016); `tools/game.sh helper i4` revives it but waits for the instance lock, so a leftover d2bot process
  (killing the job wrapper does not always kill the binary on the worker host: `pkill -f d2lod-live/bot/d2bot`) makes it hang.
- Not done: respec, equipment of the mercenary, runes/gems/sockets, charms, belt choice by potion slots, repair-aware scoring, reading real item requirements, picking up gear
  from the ground by score (the Looter's pickit decides), quest rewards other than Den of Evil, Act changes.

## Full-pack purchases (TOWNFIX, V live on i4, 2026-10-28)
Cause of "every gear purchase fails": the game places a bought item at the first free spot, so a 2x3 armor needs a real 2x3 hole; a pack 80-85% full of scattered 1x1 items has none and the click does nothing (gold and pack unchanged, 2 s wait per try). Now `buyWithRoom` checks the footprint first (`Town.PackFits`, `panel.Occupancy.FindSpot`; skipped when the item goes straight into an empty slot), calls `Town.MakeRoom` (sell pack gear the leveler does not need plus whatever the pickit sells, then stash and the unsold junk) and reopens the trade if stashing closed it; a failed purchase gets one make-room-and-retry. `ErrNoRoom` when nothing helps. The proof now counts any inventory item (right-click into an empty slot equips at once). Tests: `internal/leveler/room_test.go`, `internal/town/room_test.go` (82%-full fragmented pack fixture, need /tmp/d2mpq tables; run on the worker host).

## Class builds (LEVELERCLASS, 2026-10-28; fixes CLASSPAL findings 3, 9, 12 and the build/stat part of 5)
Tags: V live (Palli, Paladin clvl 13 on i2), S secondary, U unverified.
- **Build per class**: `Build` carries `Class` (from the file's `"class"`, default sorceress so the old files stay valid), `Hotkeys` (plan for the combat agent, never pressed here) and `RespecPool` (documentation). The default build comes from the character's class (`DefaultBuildName`: sorceress -> `fallback`, paladin -> `paladin-normal30`); `Options.Build` still selects one explicitly. A table for another class, or a class without default, is an error (`ErrWrongClass`, `ErrNoBuild`) and calls `Options.Pause` instead of guessing. Sorceress behaviour is unchanged.
- **`build-paladin-normal30.json`** (clvl 2-30, sources and tags inside the file): Maxroll order Might, Sacrifice, Resist Fire x2, Holy Fire x4, Smite, Zeal + Charge, Holy Fire (S, Maxroll Paladin leveling), Holy Fire to 19, then Holy Bolt, Blessed Hammer, Blessed Aim, Concentration, Blessed Hammer (U, needs no respec; prerequisites V from skills.txt). Stats (U, not Maxroll's all-vitality): 2 Strength + 3 Vitality per level to 13, 1 Dexterity at clvl 6, 1 Str + 4 Vit to 19, then 1 Str + 2 Dex + 2 Vit. The respec at clvl 19 is recorded but not implemented.
- **Skills by id, never by icon**: the plan rule refuses skills whose skills.txt class is not the character's; after every skill click `spendSkill` requires the unspent count to fall AND exactly the planned skill id to gain a level in the player's skill list. Anything else is `ErrSkillMismatch` (run stops, Pause called). V live: the whole clvl 13 plan (60 stat points, 12 skill points) was spent through the panel and the skill list read Might 1, Sacrifice 1, Resist Fire 2, Holy Fire 5, Smite 1, Zeal 1, Charge 1; str 48 dex 21 vit 61. Mismatch path: offline fake panel only (U live, there is no way to provoke it).
- **Gear score**: `prefs.go` replaces the Sorceress +8: per class damage weight, type bonus, two-hand multiplier, `WantsShield`. Sorceress identical (test). Paladin: weight 0.8, two-handers x0.5, no two-hander planned while a shield is worn or in the pack (ChooseGear) and none rebought (town/rebuy.go). Weights U. Strength and dexterity requirements were already enforced (`Usable`).
- **Weapon swap** (`internal/town/hand/weaponset.go`, V by CLASSPAL: `w` flips bodies 4/5 and 11/12): `SetsOf`, `FilledEither`, `EnsureMainSet` (presses `w` and proves it). Rebuy and `Leveler.Equip` swap back first, treat a slot as filled when the other set holds its twin and never buy into the inactive set. Offline tests only; the `w` press itself was not re-run (U).

## Pause wired into the bot (INTEGRATE2, 2026-10-28)
`leveler.Options.Pause` is `Runner.LevelerPause` (cmd/d2bot `runnerRef`, the runner is built after the behaviors): the bot pauses itself with "Leveler: <reason>" in `Status().Message`, a non-fatal error
event and the log, and resumes by itself as soon as no stat or skill points are pending (the plan is satisfied, for example the operator spent them); `Resume` by hand works as before.
`tools/rigs/levelerlive tree` prints the logged-in character's class (or `tree ama|sor|nec|pal|bar|dru|ass`), no longer always `sor`. V offline (`TestLevelerPauseShowsReasonAndResumesWhenSatisfied`).

## Why Booti stayed at clvl 12 (LEVELRATE, offline analysis of the i4 soak log, 2026-10-29)

Source: `livebot/20261029-053830-d2bot.log` (5482 lines, 05:38 to 07:34, Booti, Sorceress, Normal, task campaign mode). Everything tagged V below
was counted from that log; formulas not read from game data are U.

Where the two hours went (V, from status lines, about 7000 s):
Rogue Encampment (chores, heals, re-entering after chickens) 2286 s, Stony Field 1469 s, Cold Plains 1419 s, Den of Evil 720 s,
Cave 1 666 s, Blood Moor 210 s, Burial Grounds 180 s. Real farming was about 55 minutes.
XP per hour by area (V, sum of `combat kill` xp over time in area): Stony Field about 4.7k, Cold Plains 2.9k, Cave 1 2.4k,
Blood Moor 70, Den of Evil 40 (120 kills, 8 XP total). Total about 4.0k XP in the run.

1. Area choice. The campaign has no quest left it may do (Blood Raven and Charsi were `blocked`: "town: no town portal available",
   0 gold), so it runs `level.14` = `taskActs.LevelTo`: `Plan.Pick` by clvl, then `unreachable()`. At clvl 12 Pick prefers Outer Cloister
   (enter 12), Forgotten Tower, Black Marsh, Dark Wood: all refused or stalled (V): Cloister/the NAS/Catacombs "N hops and no activated
   waypoint" (Booti has no waypoint but town), Black Marsh and Dark Wood were walkable by the 3-hop rule but the character stood still in
   Stony Field for 65 s each time ("no progress", failsafe skip 15 min; `nav` never logged an exit to Dark Wood: nav owns this, U why).
   After those, Pick fell back to "the last eligible step", which with Stony Field/Cold Plains/Cave/Burial Grounds cleared was
   Den of Evil and Blood Moor (leave level 6): 12 minutes for 8 XP. `cleared` is forgotten after every re-entered game (16 re-entries),
   so the cycle Stony Field, Cave 1, Burial Grounds, Cold Plains repeated; it was in the right areas, there simply was nothing better it could reach.
   Poverty mode (`starveArea`, 108 then 15 gold, no TP) deliberately farms Stony Field (plan area for clvl-4). EARLYFIX holdTarget
   plays no role here (target0 for ReachLevel is not held, V from code).
2. XP penalty. Level 13 needs 22.5k XP and 14 another 28k (expTable in telemetry/exp.go, U: recalled, matches the classic table);
   at about 3.5k XP/h of real farming that is 6 to 8 hours per level, so "stuck at 12" is mostly a rate problem. Kills in areas with
   monster level far below clvl give 0 to 2 XP (V, Den/Blood Moor). The exact penalty formula was not found in docs or internal/data (U);
   the bot uses the empirical rule below.
3. Mana. Sorceress "fallback" is the Lightning build (Charged Bolt F4, Static Field F1; no Fire Bolt) with 0 energy points in every row of
   `build-fallback-lightning.json` (V): max mana 60 at clvl 12, regen about 0.5/s, so each pack costs 30-40 s of rest ("rest to cast budget",
   target 36). `PreEngage` already skips the rest for 1-2 monsters. Not changed: giving the build energy (or a mana-potion budget) is a build
   decision outside this agent's files; U whether it pays off versus vitality.
4. Points. No "Spending stat and skill points" activity and no "points not spent" warning in the whole log (V), and the level never
   changed, so there were no unspent points; the spend path is not the cause.
5. Poverty loop. 12 "heal below the resume line" and 10 "0 healing potions" town trips, 3 chore timeouts, one failed shop dialog (npc 148),
   7 gold pickups in two hours. Without gold there are no potions and no TP, so every fight ends in a chicken. Not solved here (shop/town agents).

### Changes
- `PickOptions.Floor` (bot `worthFloor` = 4): when the "last eligible" fallback step's leave level is more than 4 below the character and a
  better step is only excluded (cleared or avoided), `Pick` reports nothing instead of Den of Evil or Blood Moor. The classic goal loop then
  uses `noTarget` (fresh game); the task slice `LevelTo` now does the same (`freshGameToLevel`: town, then `Cycler.NewGame`).
  V offline: `TestPickFloorRefusesWorthlessFallback`, `TestLevelToStartsFreshGameWhenWorthyAreasAreCleared`.
- NAVSTALL (offline): the Stony Field stall was the guard cancelling the level slice's walk, not nav; waypoints are now activated in passing. See docs/nav.md "NAVSTALL". Not verified live.
- Not done, needs the game: activating the Cold Plains and Stony Field waypoints while leveling (would let the bot chain to Black Marsh and
  later areas and walk home without a scroll), and finding why nav does not leave Stony Field for Dark Wood.

## Mana and poverty fixes (MANAPOVERTY, offline, 2026-10-29)

All offline (`go test -p 1 ./internal/resource ./internal/leveler ./internal/gear`); nothing here is verified live.

- Build rows: `LoadBuild("fallback")` moves 2 of every 5 vitality points to energy (`withEnergy`, `sorcEnergyPerRow`), applied at load because
  the data file is in `internal/plandata` (not owned here). Strength rows and the 5 points per level are unchanged. Characters that already
  spent points keep them (the plan only places the remainder). U: 2 is not an optimised split; check max mana at clvl 12 (expect about 60 + 2*2*11 = 104 with items aside).
- PreEngage: when the rest to the cast budget would take over two thirds of RestMaxSec (about 30 s, U), the pack is harmless (danger 0), at least one cast of mana
  is left and a fallback exists (melee/staff, free skill, or a cheaper spell), it proceeds instead of resting; AttackChoice then fights cheap.
- AttackChoice: when mana is below hostiles x CastsPerMonster x cost of the preferred spell it uses the cheapest spell (economy) before the pool runs dry.
- Poverty: `resource.PovertyAdvice(in, PovertyCheck(...), sellMin)` returns SellTrip (>= sellMin sellable items, fewer than 2 idle trips in a row)
  or PlayLean (no potions, MaxPackDanger 0). New `Input.Sellable` and `Input.IdleTrips`; `TownTripTrigger` suppresses a heal-only trip after 2 idle trips
  while life is above MinEngageLifePct (breaks the flee<->town loop).
- Not wired (bot/shop are not owned here): `bot` must fill `Sellable`/`IdleTrips` in `resInput`, call `PovertyAdvice` from `updateStarvation`, and honour `MaxPackDanger`.
- Live checks needed: Booti respec-free level-up shows energy +2 per level; mana-rest time per pack at clvl 8-12 in the log; one poverty spell with
  junk in the pack ends in a sell trip, and no more than 2 empty town trips occur.
