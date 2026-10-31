# Early game: a fresh character, clvl 1 to 12 (agent EARLYGAME, 2026-10-30)

Offline audit of what the bot does with a brand new character of each of the seven classes: points, skills, hotkeys, first purchases, area order,
first-hour failure modes. Nothing here was run live. Tags: **V** verified (how is said), **S** secondary (a doc or table), **U** unverified.
Tests: `go test ./internal/leveler -run Early -v` (replay and tables), `./internal/resource -run Fresh`, `./internal/task -run FreshCharacter`,
`./internal/bot -run GearPass`, `./internal/combat`.

## 1. Per-class table (replay of the default build; Den of Evil point taken at clvl 5)

Source of the rows: `TestEarlyPrintTable`, `TestEarlyPrintKeys`, `TestEarlyEveryClassHasAnAttackAtEveryLevel` (V offline: the real `leveler.Build.Plan`
with the skills.txt prerequisites from `the d2tables cache gear.json`, the real combat profile and the weapon rules of `internal/gear`).
Starting kits (V, charstats.txt via gear data): all classes 4 minor healing potions, 1 TP scroll, 1 ID scroll; Sorceress Short Staff; Necromancer Wand;
Paladin Short Sword + Buckler; Amazon **Javelin** + Buckler (no bow); Barbarian Hand Axe + Buckler; Druid Club + Buckler; Assassin Katar + Buckler.
New-character gold: U (not read anywhere).

| class (default build) | stats clvl 12 (str/dex/vit/ene) | skills owned at clvl 6 | at clvl 12 | attack order clvl 1 / 3 / 6 / 9 | keys (F) |
|---|---|---|---|---|---|
| Sorceress (`fallback`, Lightning) | 20/25/37/53 | Charged Bolt 5, Static Field 1 | CB 10, SF 1, Telekinesis 1 | Fire Bolt / CB, FB / CB, FB / CB, FB | SF f1, Frozen Armor f3 (learned clvl 22), CB f4, **Fire Bolt f7 (new)**, Frost Nova f8 (never learned) |
| Necromancer (`necromancer_summoner_normal30`) | 20/25/55/35 | Raise Skeleton 2, Skel. Mastery 2, Clay Golem 1, Teeth 1 | + Amplify 1, Skeletal Mage 1 | Attack / Attack / Teeth, Attack / Teeth, Attack | Teeth f1, Raise Skeleton f5, Mage f6, Amplify f4, Clay f8 |
| Paladin (`paladin-normal30`) | 46/21/58/15 | Might 1, Sacrifice 1, Resist Fire 2, Holy Fire 1, Smite 1 | Holy Fire 6, Zeal 1 | Attack / Sacrifice / Sacrifice, Smite / same | Might f1, Holy Fire f2, Resist Fire f7; Sacrifice/Smite/Zeal left hand |
| Amazon (`amazon_bow_normal30`) | 22/28/70/15 | Magic Arrow 1, Fire Arrow 5 | FA 8, Exploding Arrow 2, Multiple Shot 1 | Attack / Attack / Attack / Attack (javelin, no bow yet); with a bought bow Magic Arrow, Fire Arrow, Attack; **Exploding Arrow first from clvl 12 (new)** | Magic Arrow f1, Fire Arrow f2, Multiple Shot f3, **Exploding Arrow f6 (new)** |
| Barbarian (`barbarian_melee_normal30`) | 40/35/55/10 | Bash 3, Axe Mastery 2, Double Swing 1 | DS 3, Axe M. 3, Howl, Shout, Double Throw | Attack / Bash / Double Swing, Bash / same | Shout f2 (Bash and Double Swing: left hand) |
| Druid (`druid_fire_normal30`) | 20/20/75/20 | Firestorm 5, Molten Boulder 1 | Firestorm 9, MB 2, Eruption 1 | Attack / Firestorm / Firestorm, MB / same | Firestorm f1, MB f2, Eruption f3 |
| Assassin (`assassin_fireblast_normal30`) | 25/20/60/35 | Fire Trauma (= Fire Blast) 4, Claw Mastery 1, Quickness (= Burst of Speed) 1 | FT 9, Wake of Fire Sentry 1 | Attack / Fire Trauma / FT / FT | FT f1, Wake f2, Quickness f3 |

Stat rule, all classes: 5 points per level, every point spent by the replay (V offline, `TestEarlyReplaySpendsEveryPoint`: 5*(clvl-1) at every level, no unspent
skill point at clvl 12). Sorceress Str floor 12 (V: `TestEarlyStrengthFloorSorceress`): the first 2 points of clvl 2 go to Strength, so her Energy rows start at clvl 3.
Every class has an attack with its starting kit at every level 1..12 (`TestEarlyEveryClassHasAnAttackAtEveryLevel`); the weakest are the clvl 1 casters-by-skill-point
classes (Necromancer, Paladin, Barbarian, Druid, Assassin, Amazon), who use plain Attack until the first skill point at clvl 2 (Necromancer: the skeleton army, U that
Raise Skeleton exists at clvl 1: charstats start skill is "raise skeleton", whether it is level 1 or only pre-selected was not read).

### Comparison with the SME no-respec tables (docs/sme/class-*.md section 4, builds-leveling.md section 3)
The bot's default tables come from the Maxroll-based `internal/gear/builds` files and the live-proven Sorceress/Paladin tables. They differ from the SME lines; NOT changed
(each is a build decision for the user, none is a bug):
- Sorceress: SME wants the cold line (Ice Bolt, Ice Blast, Blizzard at 24). The bot buys Lightning (Charged Bolt to 10, Static Field at 6, Teleport at 18). Frost Nova (the
  combat code casts it on crowds of 3, `castfirst.go`) is never bought, so that code is dormant; Frozen Armor arrives at clvl 22 (before that no armor spell is cast). The
  `primary` table (Blizzard, respec at 26) stays off. Decision needed: fallback as is, or move Static Field at clvl 6 to Frost Nova (data file `internal/plandata/build-fallback-lightning.json`
  and the copy in docs/plan/data, kept equal by a test; not touched here).
- Paladin: bot Might, Sacrifice (clvl 3), Resist Fire, Holy Fire, Smite, Zeal at 12. SME: Might, Sacrifice, Prayer, Holy Bolt, Smite, Zeal. Sacrifice costs life per hit
  and the combat order is Sacrifice before Smite (Smite arrives only at clvl 10 in the table). U whether a fresh Paladin survives Sacrifice at clvl 3-9 with potions; flagged for the live run.
- Barbarian: Axe Mastery (kit is a Hand Axe) instead of the SME's Sword Mastery; Double Swing needs only a melee weapon in skills.txt (V data: itypea `mele`); with one weapon it is U.
- Necromancer: Teeth gets 1 point at clvl 5 and no more to 12, the army is the damage (Maxroll puts the points in Teeth). Matches the SME summoner line; Raise Skeleton and Mastery alternate.
- Amazon: Magic Arrow/Fire Arrow points need a bow (below). The javelin build `amazon_javelin_normal30` (Poison Javelin at 6, plain Throw before) fits a no-gold start better; not the default.
- Druid, Assassin: same as the SME (Firestorm 1-5, Molten Boulder 6-7; Fire Blast, Burst of Speed, Wake of Fire at 12).

## 2. Area and quest order for a fresh character (V offline: `bot.Plan.Pick`, `task` Normal Act 1 campaign in the fake world)
1. **Den of Evil first, always**: campaign entry `den_of_evil` has floor clvl 1 (`NoKitMinLevel` 5 only for a character without scroll and potions; a fresh one has both).
   `TestFreshCharacterStartsWithTheDenAndNeverStallsInAct1` (V offline, 4 variants: caster or not, gold 0 or 300) starts with talk-akara, wp cold plains, to-den, clear, home, reward.
2. Then `level.8`: `Plan.Pick` windows (progression-normal.json): Blood Moor 1-6, Den 2-6, Cold Plains from 5 (4 with scroll and potions), Burial Grounds and Cave 1 from 6,
   Stony Field from 8, Dark Wood 9, Tamoe Highland 10, Outer Cloister 12. With Blood Moor and the Den already cleared in this game, clvl 1-3 gets NO area (by design `Floor`:
   a fresh game starts instead), clvl 4 Cold Plains, 6-7 Cave 1, 8 Stony Field. Measured XP/h (V from Soaky, docs/leveler.md): Stony Field 4.7k, Cold Plains 2.9k, Cave 1 2.4k: the
   progression table enters Stony Field only at clvl 8; moving it earlier is a data decision (not made).
3. Blood Raven gate (owned by BRKILL, not touched): `bossGates[QBurial]` clvl 8, 4 potions, scroll, 2 mana potions for a caster; SME 10.1 wants clvl 9 for Sorceress, Druid, Amazon.
4. Shopping order clvl 1-8: scroll/tome, healing potions (kit `HPFirst`), mana potions for casters, then a belt at Fara (Act 2). Armor is never bought below clvl 15
   (`resource.earlyGearMax`); the Amazon buys Short Bow + Arrows through `gear.StartPurchases` in the town chore when she has the gold (U how early that fires).

## 3. What the audit found and fixed
| # | finding | where | fix |
|---|---|---|---|
| 1 | The leveler gear pass (`gearStep`) bought armor and weapons with 70 % of (gold-100) from 150 gold on, at any level, against the early-gear policy: it competes with potions and the scroll. V offline (test failed before). | `internal/bot/leveling.go` | below `earlyGearMax` the pass only equips what dropped; `TestGearPassDoesNotShopBelowEarlyGearMax`. The old test now runs with the policy off (the rig default). |
| 2 | A quest blocked by "not ready: 0 mana potions" or no scroll stayed blocked until the next game; with every quest blocked and clvl >= the highest floor the plan was `Stuck` and the bot self-paused (the Soaky "blocked: no town portal, 0 gold" pattern). V offline: a poor Sorceress ended the Act 1 sim stuck at clvl 15. | `internal/task/executor.go`, `campaign.go` | supply blocks are released after `kitRetryGold` (60) more gold or a level; when every blocked quest only waits for supplies the campaign returns a level step (farm gold) instead of Stuck. A clvl block or a death/strike block is untouched. Tests: `TestFreshCharacterStartsWithTheDenAndNeverStallsInAct1`. |
| 3 | The Sorceress lightning layout had no key for Fire Bolt (her starting skill and the cheap fallback); it was lost once Charged Bolt was bound. | `internal/combat/classdata/profiles.json` | Fire Bolt f7 (Energy Shield f7 removed: the table never buys it). `TestEarlySorceressBoltsHaveKeys`. |
| 4 | The Amazon table puts clvl 12-30 points into Exploding Arrow but the combat profile never cast it (and gave it no key). | same | band from clvl 12 (and in the clvl 18 band), key f6; `TestEarlyAmazonCastsExplodingArrowFromClvl12`. |
| 5 | The task sim's Chores never bought mana or antidote potions and LevelTo dropped no gold, so a fresh poor caster could not be simulated. | `internal/task/sim/sim.go` | Chores buy mana potions (casters, 100 gold) and antidotes (150 gold); a level slice drops 30 gold (all U); the default world (500 gold, 2 antidotes) behaves as before. |
Verified clean: no clvl-1 character is gated (`TestFreshCharacterIsNeverGatedEarly`: 7 classes x clvl 1-14, starter weapon, 4 potions, 0 gold); a weapon-needing class
without a weapon is gated at once, a caster is not; clvl 5+ with no potions and no gold is gated to the safe farm area (the starvation path).

## 4. Not done / open (all U)
- Start gold and whether the Amazon can afford Short Bow (about 100) + Arrows before she needs them; until then she fights with the javelin (Attack = thrust or throw: U) and her
  first four skill points sit in bow skills she cannot use.
- Sorceress build choice (above), Paladin Sacrifice, Barbarian Double Swing with one weapon.
- Frozen Armor for the Sorceress before clvl 22 (not in the table, so nothing is recast); Cold Plains at clvl 5 for a 60-mana Sorceress.
- Hotkey binding itself (`EnsureHotkeys` through the skill popup) is V for the Sorceress and Paladin only; Necromancer/Druid/Assassin/Amazon/Barbarian bindings of a fresh character are U.

## 5. LIVE CHECKLIST: first fresh-character run (nothing below has been run)
Create with `tools/soak/fresh.sh <iN> <class> <Name>` (docs/fresh-character.md), then `tools/soak/soak24.sh start iN --gate --char <Name>`. No cheats. Start with the Sorceress, then one class
of each kind (Paladin, Necromancer, Amazon) as smoke. Expected milestones (docs/sme/pace-tables.md 5.2, bot-pace, U; plan-v2 section 5 loosens the first run to 2 h = clvl 6, 4 h = 10):

| wall time | expect (log) | red flag |
|---|---|---|
| start | `class-aware start` with the class, level 1, the right `combatProfile` and `levelerBuild`; no `gear shop` line below clvl 15 | wrong profile; `gear shop` with a budget |
| 0:05 | quest `den_of_evil`: talk-akara, waypoint, Den; `level plan` lines; kills with XP; potions drunk below 70 % life | no cast with hostiles (Amazon: `weapon problem`), idle in town |
| 0:20-0:45 | Den clear, Akara reward (+1 skill point, `Spending stat and skill points`), clvl 3-5 | `points not spent` repeated, claim loop |
| 1:00 | Den done (floor: by hour 1), clvl 5-6, Cold Plains or fresh-game Blood Moor cycles, gold > 0 | `poverty mode: on` for > 15 min, 3 deaths in 20 min |
| 2:00 | clvl 7 (floor) to 9 (good); belt has 4 potions and a scroll; `quest.sisters_burial` held or ran with the kit; Rogue not hired | Burial attempts below clvl 8, `every quest waits for supplies` for more than one gold-farm slice, chicken loops |
| 4:00 | clvl 10-12; `boss: held for andariel (clvl < floor)` then `gate pass`; Cave/Stony Field/Dark Wood XP | no XP gain in 45 min |
Per class, check once: the first skill bought at clvl 2 (table above), the key bound (`combat bind`), the first purchases (TP/potions first; Amazon bow + arrows), what the Amazon,
Paladin and Barbarian cast at clvl 1-3 (log `combat decision`), the Sorceress mana at clvl 6 (Static Field bought, Energy rows) and whether she rests more than 30 s per pack.
Record deaths with potions in the belt as bugs and deaths with 0 potions as supply bugs; save the log to the board hub.

## 6. BUILDV2 (2026-10-30): fresh-character decisions of the SME, implemented offline
Selection: `leveler.SelectBuild(class, level, ownedSkills)` (used by the leveler, `cmd/d2bot` class-aware start and the bot's log name). A config `levelerBuild` still wins. Level 1 gets the v2 table; a higher-level character gets v2 only while every skill the OLD table buys up to clvl 15 that it owns is also in the v2 table (Soaky, with Static Field, keeps `fallback`; a Paladin with Holy Fire or Resist Fire keeps `paladin-normal30`). The planner never refunds or "fixes" owned skills: it only counts them against the table (V, `Build.Plan`), so a switch could only change the order of future points. The `class-aware start` log line shows `levelerBuild` (`fallback-lightning-v2`, `paladin-normal30-v2`).
| class | change | where | tag |
|---|---|---|---|
| Sorceress | `fallback-lightning-v2`: Charged Bolt 4 and Frozen Armor 1 (Den point) by clvl 5, Frost Nova 1 at 6 (Static Field dropped), Telekinesis 11, Charged Bolt 8 at 10, 12 at 15, 5 Vit per level after the Str 12 floor | plandata + docs/plan/data, `TestBuildV2SorceressSequence` | V offline |
| Sorceress | `combat.novaManaFloor` 25: no Frost Nova on a crowd below this mana (the crowd rule is 3 within 4 subtiles); Frozen Armor recast by the existing `combat.buffRecastSec` 150 s timer (profile buff group 1) | `castfirst.go`, `tuning_combat.go`, docs/knobs.md (38 combat knobs) | V offline; floor value and armor duration U |
| Amazon | bow table: Multiple Shot at 7 (6 with the Den point); javelin variant kept; trip at 150 gold for a bowless bow-path Amazon, ammo chore before the potion shop | plandata + gear/builds copies, `town/ammo.go`, `chores.go` | trip test needs /tmp/d2mpq (skipped on the head box); price U |
| Paladin | `paladin-normal30-v2` (above); Sacrifice removed from every combat band | plandata, `classdata/profiles.json`, `TestPaladinBands` | V offline |
| Barbarian | opener Bash, then Double Swing | profiles.json | V offline; live U |
| all | `combat bind` summary line (class, level, wanted, bound list, failed) at the end of `EnsureHotkeys`, also for "popup rows not loaded" | `combat/bind.go` | V offline |
| areas | Stony Field `melee_enter_clvl` 7 (Barbarian, Paladin), 8 for the rest | progression-normal.json, `bot.Step.MeleeEnter`, `PickOptions.Melee` | V offline |

EARLYFIX2 (done, V offline, U live): Den of Evil class floor. `campaign.denMinLevelCaster` 5 (Sorceress, Necromancer, Druid, an Amazon without a worn bow or crossbow) and `campaign.denMinLevelMelee` 3 (Barbarian, Paladin, Assassin, bowed Amazon). `World.DenMinLevel` is filled by `bot.denFloor`; `Entry.Den` raises the Den floor to it (Normal only, max with the no-kit floor 5). Below the floor the plan is `level.N` with Why `den: waiting for clvl N in Blood Moor`, and `guardedAreas` keeps area 8 out of the level picker, so Blood Moor is farmed; the Den stays the first quest (no entry precedes it, tests `internal/task/den_gate_test.go`, `TestDenFloorPerClassAndPickerExclusion`). Consistency with section 6: a Sorceress reaches the Den at clvl 5, so the Den point (Frozen Armor, 5th point) lands at clvl 5-6 as BUILDV2 expects. The floor 5 is the safe end of the SME range 4-5 (U).

### LIVE CHECKLIST (BUILDV2, nothing run)
1. Fresh Sorceress: log `class-aware start ... levelerBuild=fallback-lightning-v2`; at clvl 5-6 `Spending stat and skill points` shows Charged Bolt then Frozen Armor (after the Den reward), Frost Nova at clvl 6, no Static Field; `combat bind` line lists Frozen Armor=f3 and Frost Nova=f8; `buff skill=Frozen Armor` before fights and about every 150 s (check the real duration; the state is not read). Mana at clvl 6 (about 50-60): Frost Nova casts (`combat decision act=nova why=crowd n=3+`) only with mana >= 25 and the health line `novas=` stays small; rests per pack under 30 s.
2. Fresh Amazon: `class-aware start levelerBuild=amazon_bow_normal30`; before gold 150 plain Attack with the Javelin (`combat decision`), no `weapon problem` line; at 150 gold a town trip with reason `bow-path Amazon without a bow and gold for one`, `Short Bow` and Arrows bought at Charsi BEFORE potions, bow and quiver worn; a `weapon problem, not attacking` line after the bow appears means arrows or the quiver slot failed; Magic Arrow then Fire Arrow cast; Multiple Shot bought at clvl 6-7.
3. Fresh Paladin: clvl 1-9 `combat decision` / cast lines show plain Attack with Might on, no Sacrifice cast at any level; clvl 3 Sacrifice 1 only; Prayer at clvl 4; Smite at 10 and used (left skill set); Zeal 1 at 12 and used first from then on.
4. Fresh Barbarian: first attacks at clvl 2-4 are Bash (left skill), not Double Swing.
5. Every class: a `combat bind class=... wanted=N bound=... failed=0` line after the first `EnsureHotkeys`; `failed>0` or "popup rows not loaded" is the bug to chase.
6. Stony Field entered at clvl 7 by a Paladin/Barbarian and 8 by the others (`level plan` line).
7. Soaky (or any character that owns Static Field) must log `levelerBuild=fallback` after a restart.

## MANABACKOFF: mana-starved caster (Fresha soak 2026-10-30, V from the log)

Fresha (Sorceress clvl 4-6, i5, 10:52-11:44): 207 `out of mana: backing off` lines in 21 bursts, about 400 s of 53 min (13 %), at clvl 4-6 in Cold Plains/Blood Moor, 9-17 hostiles near, `danger=2`. Recovery state was 34 of 51 minute lines, and in recovery the shop plan said `wantMP 0` every visit (the recovery potion floor zeroed mana for casters); the first mana potions (3) were bought at 11:27 with 542 gold. Charged Bolt costs 3 mana (S, docs/sme/class-sorceress.md), pool about 45 at clvl 6 (U), so a pool is about 15 casts. Back-off rule: `resource.AttackChoice` returns Retreat when no skill is affordable, melee needs pack <= 3 or a weak pack and danger < 2, and waiting needs the same; in a big danger-2 pack nothing ever ends the retreat (the pack follows, mana barely regenerates).
Fix: `resource.backoffMaxSteps` 8 (melee after 8 consecutive back-off steps, danger < 3, life >= 50 %, reset by any cast); `resource.recoveryManaFloor` 2 (casters keep a mana reserve while recovering, after the healing floor). Watch live: `out of mana: back-off cap, melee`, `step` on `backing off`, `shop: potion plan ... wantMP`, `beltMP`/`mana` in the minute line. Side finding (not fixed, other layer): 159 `town portal attempt failed` and about 12 TP scrolls bought on the way, 1200 gold.

## Portal failure loop (PORTALFAIL, 2026-10-30)

Evidence (V, Fresha i5 log 20261030-105216, clvl 4-6): 159 `town portal attempt failed` + 53 `safety action failed` (action 4 = TownPortal), all `no portal appeared after the click`, in 11:08-11:20. The pack held an EMPTY Tome of Town Portal (`tbk`, shop line `tp:0 tome:true`), no scrolls. `internal/combat` counted any `tbk` as a portal, so every low-life tick (about 7 s apart) clicked it three times. No scrolls were lost (none existed), no deaths (deaths=0 all run); the cost was about 12 minutes of repeated failed casts while low on life, 3 `town trip abandoned`, and the walk-back-on-foot fallback. The G2 log (old build) has 3 such lines only.

The 12 scrolls (~1200 gold) bought 11:26-11:36 were mostly spent: 6 `town portal cast` lines (one per emergency), 1 found. Not a loop, but the target of 5 (tome present) ate potion money at 300-460 gold.

Fix: an empty tome (no quantity stat, same rule as `town.qty`) is not a portal (Safety.HasPortal, openPortal); after a failed cast the TownPortal action is silent for 30 s (potions, retreat or chicken instead); failed attempts log `portal: attempt N failed because <reason> scrolls=<before>-><after>`; the shop holds at most 2 spare scrolls under twice `TripGoldFloor` (1 under the floor, as before). Watch live: `portal: attempt`, `shop: visit ... tp=`.
