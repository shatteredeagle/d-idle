# Combat: internal/combat (agent COMBAT)

## API summary (package `d2lod-bot/internal/combat`)

```go
f := combat.New(e *env.Env, mover combat.Mover)        // Mover: GoTo, GoToUnit, PathTo (consumer-side subset of nav)
f.Clear(ctx, ClearOpts{Radius, Origin, Timeout}) (ClearResult, error)
f.Kill(ctx, unitID) error
f.Buff(ctx) error
f.Safety(snap) SafetyAction      // None DrinkLife DrinkMana DrinkRejuv TownPortal Chicken (pure given snapshot + own drink times)
f.Act(ctx, SafetyAction) error
f.Died(prev, cur *d2.Snapshot) (DeathInfo, bool)      // the one death transition; the BOT emits goals.Death from it
f.EnsureHotkeys(ctx) / f.BindSkill(ctx, name, key)    // bind F-keys through the right-skill popup (needs SetListRows)
f.SetListRows(m) / combat.LoadListRows(installDir, db) // skill id -> skilldesc ListRow (popup rows), read from the MPQ
f.SetHotkeys / f.Hotkeys / f.SetDeathWeights / f.SetCatalog / f.SetSeed
f.Build (BuildBlizzard | BuildLightning), f.Leave, f.Portal  // hooks: leave game, open town portal (defaults UNVERIFIED)
```
Pure pieces, all unit-tested: `Hostiles(snap, HostileCtx)`, `Rank(targets, quality, roll)`, `PickAttack(order, target, mana, ready)`,
`Decide(SafetyInput)`, `BeltOf(inv, cat)`, `Died(prev, cur, cat)`, `ResistsFromStats`, `awayPoint/approachPoint`, `Layout(build)`.

- `Clear`/`Kill` return the partial `ClearResult` plus `ErrTownPortal` (a portal was opened, go to town), `ErrChicken` (the game was left),
  `ErrDead`, `ErrNotInGame`, `ErrTargetGone` / `ErrTargetGaveUp` (Kill). A Mover error during a fight is not fatal: the approach counter
  drops the target after `maxApproaches` failed walks and `ClearResult.Skipped` counts it.
- Events: `goals.EvKill` (Monster, Boss, XP delta attributed from the player's exp change) and `goals.EvPotion` (Potion = healing|mana|rejuvenation)
  through `e.Emit`. Combat does NOT emit `EvDeath`: the runner calls `f.Died(prev, cur)` on its snapshot stream and emits it (avoids double counts).
- `DeathInfo`: Area, Pos (corpse), killer unit id/class/name/level/affixes/distance (nearest living hostile in the last live snapshot, elites and
  bosses within 8 subtiles preferred), LastLife/LastMax, Curve (last 40 life samples the fight loop recorded), Difficulty, CharLevel, Mode.
- Death history: `SetDeathWeights(DeathWeights{Monster: {"fallen1": 1}, Affix: {"mana-burn": 3}})`. Weight >= 1 raises kill priority and triggers kiting,
  >= 3 makes the target "avoid" (not fought, counted in `Avoided`) unless it is the Kill target or a boss. Keys: lower-case monstats id, `Affix.String()` names.
- Hotkey layout (`Layout`): the game has only F1-F8. Blizzard build: F1 Static Field, F2 Teleport (NAV casts it; combat binds it but never needs it),
  F3 Frozen Armor, F4 Charged Bolt, F5 Blizzard, F6 Glacial Spike, F7 Ice Blast, F8 Fire Bolt. Lightning build: F5 Chain Lightning, F6 Lightning, F7 Energy Shield.
  Deviation from docs/plan/data/hotkeys.json: Telekinesis dropped, Charged Bolt and Fire Bolt added. All casting is right-click (`CastAt`).
- Tuning: `env.CombatTuning` in internal/env/tuning_combat.go, 38 knobs with ranges (thresholds, potion gaps, reaction, poll, target quality, kite, static,
  timeouts, buff recast). Distances are subtiles (5 per tile): the plan's "8 tiles" kite rule is 40 subtiles, which does not fit a 25-tile viewport; default kite distance 10 subtiles.

## Behaviour
- Hostile = alive (mode not 0/12), not a town NPC (monstats npc/!killable), not owned by the player or by a unit the player owns (merc, summons), nothing in town.
- Rank: kill-first (shaman/oblivion/unraveler/bloodraven names, Cursed/Mana Burn/Multi Shot/Aura affixes) +3000, death weight x400, elites within 25 +600,
  anything within 6 subtiles +1500, then nearest. Danger table "avoid" (Hell, clvl < 75: Cursed, Mana Burn, Lightning Enchanted, Spectral Hit) drops the target.
  `targetQuality` < 100: with probability (100-q)% the nearest target replaces the best, and (100-q)/2 % of ticks are lost to hesitation.
- Skills: Blizzard build: Charged Bolt, Fire Bolt, Ice Bolt before clvl 26 (or without Blizzard); then Blizzard, Glacial Spike, Ice Blast, Ice Bolt, Charged Bolt, Fire Bolt.
  First skill that is bound, not immune-blocked for the target, affordable and off cooldown wins (Blizzard's 45-frame delay is respected); too expensive falls to a cheaper one,
  blocked falls to the next element, everything blocked tries Static Field, else the target is skipped. Immunity: stat list resists (36/37/39/41/43/45) when present, else monstats; >= 100 is immune.
- Static Field opens elite/boss fights (`staticCasts`, floor by difficulty) when the target is within its range.
- Positioning: approach to range minus margin and into the viewport (`onScreen`); kite melee monsters closer than `kiteDistance` when life <= kiteLifePct or an elite/dangerous one is near
  (budget `kiteMaxSteps` per 10 quiet seconds); dodge hostile missiles within `dodgeRadius`. All walking via Mover (`PathTo` validates candidate points).
- Survival (`Decide`): life <= chicken%: rejuv if ready else Chicken (out of a fight a healing potion first); < rejuv%: rejuv; < drink%: healing potion, no potion at all -> TownPortal;
  hurt, quiet and <= `townPotionMin` potions -> TownPortal; mana < drinkMana% -> mana potion. Chicken % from tuning (Hell has its own). Reaction delay applies to routine
  potions only; at chicken+10% and below the bot acts at once. Cast gaps never block survival checks (the loop ticks every `pollMs`).
- Per-target watchdogs: engaged time `maxTargetSec`, no life drop for `noProgressSec` (counted only while casting at it).
- Buff: one armor (Frozen/Shiver/Chilling) and Energy Shield for the lightning build, recast every `buffRecastSec` (buff state is not readable: UNVERIFIED offset).

## Tests (offline)
`go test ./internal/combat ./internal/env` (about 0.5 s). Pure tests plus a small world simulator (`sim_test.go`): monsters walk and hit, casts damage, potions heal, time moves only when the
fighter sleeps. Covers clearing, XP attribution, radius, potions/events, chicken and portal, rejuv at the chicken line, immune skip, no-progress give-up, boss Kill, out-of-mana bound, cancel,
kiting, Static Field opener, binder (simulated popup, broken binding detected), Died, death weights.
Handicap tests: every knob slowed through its range keeps the clear alive (12 seeds); targetQuality, pollMs, castGapMs make a clear >= 5% slower; the survival knobs
(drink %, potion gap, reaction) only have the "not suicidal" half. **Each knob at its worst alone, and all knobs half way, never die against a 5-monster pack; all knobs at their
worst at the same time DO die** (reaction 700 + poll 400 + cast gap 1200 + chicken 15). Handicap profiles must not stack every knob at its limit. NOTE (2026-10-29, U): the `rookie` profile (level 1) sets every knob to its worst end at once, including the combat ones (see docs/knobs.md section 3); the offline sim above says that combination dies, so rookie is not promised to survive and has never been run live. Ranges here are the sim windows, not a survival guarantee for the full stack; use `sloppy` or per-group `Per` strengths for a handicap that must live.

## Death: VERIFIED LIVE (the worker host i2, "Combi", softcore level 1 Sorceress, Normal, Blood Moor, 3 real deaths 2026-10-28)
Recorded by `combatlive deathrec` (100 ms samples of player, unit tables, UI flags, inventory), fixtures in `internal/combat/testdata/death/` (README there).
Code: `internal/combat/deathfacts.go` (`ModeDying`, `ModeDead`, `IsDead`, `CanRespawn`, `ActTown`, `FindCorpse`, `Corpse.Touched`, `GoldPile`, `Fighter.Respawn`, `Fighter.Corpse`);
`Died()` and the fight loop use `IsDead`; `DeathInfo.Gold` is the gold carried at the last living snapshot.

| fact | value |
|---|---|
| player mode while living | 1 stand, 2 walk, 4 hit recovery, 5 stand in town, 10 cast |
| mode at the killing blow | **0** (dying). Life reads 0 and carried gold reads 0 in the SAME sample; equipment still reads as worn |
| mode after the fall | **17** (dead), 0.91 s after the first mode 0 sample (death 1; death 2 still read 0 at 0.87 s). Equipment count drops to 0 in this sample, the unit-table player list grows by one |
| UI flags | NO death flag. Flags before, during and after were identical ([0 21 35] in the field). The "You have died" text is not in the UI array: detect death by mode/life only |
| Esc | ignored while mode is 0 (Esc at +0.17 s and +0.67 s did nothing), works as soon as mode is 17: Esc at +6.0 s respawned within 0.1 s (death 1), Esc every 400 ms respawned at +1.11 s (death 2). `Fighter.Respawn` measured 290 ms end to end on a death that was already mode 17. So: about 1 s after death, then 0.1 to 0.3 s |
| respawn | act 1 town (area 1), at the start position of a new game (Combi (6343,5413)), mode 5, FULL life and mana (40/40), no experience lost at level 1 (exp stayed 69/87 across deaths), belt potions and inventory items kept |
| corpse | a second player-type (type 0) unit in the SERVER unit table (`Reader.Units(d2.TypePlayer)`), mode 17, a new id (2; own unit stays id 1), at the exact death position, flags 0x1000b (the living player reads 0x3000000f). The own unit also reads mode 17 until the respawn. It is not in `Snapshot.Monsters/Objects/Ground` |
| equipment | everything worn goes onto the corpse (equip 1 -> 0). Inventory (2 items) and belt (4 potions) stay on the character |
| gold | all carried gold (500) left the character at the first dead sample and lies as a gold pile: a ground item (class 523, mode 3) 2 to 3 subtiles from the corpse. Not on the corpse |
| pickup | walking within about 2 subtiles of the corpse takes it back at once (touched at 1.4 and 2 subtiles; the worn staff returned in the same second, no click needed; clicking via `Interact` never hovered). Walking onto the gold pile gave back all 500 gold. Retrieval is therefore `GoTo(corpse.Pos)` then `GoTo(pile)`; `Corpse.Touched`, `CorpseTouch` = 2.5 |
| side effect | with the staff off the corpse the Fire Bolt skill (+1 from the staff) disappears from the skill list; it is back with the item |
| second death | three deaths happened; a second death while a corpse was still lying (death 3, no pickup in between) was not examined for what happens to the older corpse (UNVERIFIED) |
| gold lost | nothing at level 1 / 500 gold (full pile recovered). UNVERIFIED for higher levels or larger sums (the rule may scale) |

How the bot should use it: detect with `combat.IsDead`; wait for `CanRespawn` (mode 17) and then press Esc every `EscRetryEvery` (400 ms) (`Fighter.Respawn` does exactly that and never presses Esc while alive); after the respawn read `Fighter.Corpse()` is only possible near the body, so keep `DeathInfo.Pos`;
retrieval: nav to the area, `GoTo(Pos)` (the corpse unit can be refined with `FindCorpse` once close), then `GoldPile` and `GoTo` it. The nav `CorpseApproach` refuses while monsters stand within the caution radius (they killed us there).

## Unreachable and avoided hostiles (reach.go)
Every hostile is assessed once per 2 s while it is out of range, off screen or without line of sight:
1. a path exists (`Mover.PathTo`), or the map is unknown there, or it is farther than 35 subtiles: approach as before;
2. no path: `FiringPosition` looks on rings (100/75/50/30% of the skill range) around the target for a point that is reachable and has a clear line of sight (`GridLOS` over the collision grid: BlockLOS bit; rivers block walking but not bolts), walks there and shoots from range;
3. nothing qualifies: the target is ignored with a reason.
Reasons (`IgnoreReason`): unreachable, no-shot (never in range with a clear line within `ReachSec` 15), approaches (`MaxApproaches`), avoid (danger table / death weights), immune, no-progress, time. They are returned in `ClearResult.Ignored` (with id, name, position, reason, detail), remembered across Clear calls for `IgnoreMemorySec` (90, doubling per repeat up to 4x) so a second `Clear` returns at once and the sweep cannot loop; an ignored monster that comes within 7 subtiles is fought again. `Fighter.IgnoredTargets`, `IsIgnored`, `ForgetIgnored` (call on area change). Knobs `combat.reachSec`, `combat.ignoreMemorySec`.
Live (i2, Blood Moor): 2 Zombies behind water/cliff at 21 to 34 subtiles were reported `unreachable` in 60 ms while the other 6 hostiles were killed (6 kills in 29 s); before the change the same kind of unit made the walk loop.

## Survival, rotation and farming: VERIFIED LIVE (i2, Combi)
- **Chicken** (`combatlive survive chicken`, chicken line set to 100% to force it): the fighter drank a healing potion (out of a fight a potion comes first), then `Clear` returned `ErrChicken` 1.8 s after the decision, with `launch.Flow.LeaveGame` as `Fighter.Leave` (Save and Exit to the main menu), and `Flow.Start("Combi")` was back in the game 2 s later. A re-entered game always starts in the act 1 town at the start position (the character had been in the Blood Moor), full state, 0 monsters near. Whole exit + re-entry about 4 s.
- **Potions mid-fight**: 1 and 2 healing potions were drunk during the Cold Plains farms by the default thresholds (event `potion`), kiting fired repeatedly (Fallen packs). A level 1 character took no damage from Zombies (slow), so the 70% line run did not trigger; the threshold logic itself is unit-tested.
- **Town portal in a fight**: NOT verified (Combi has no scroll or tome; the portal object clicks belong to TOWN2).
- **Mid-level rotation** (cheated: save edit to level 18, vit 50, Fire Bolt 3, Charged Bolt 4, Static Field 3, Frozen Armor 1, Ice Bolt 5, Ice Blast 1, Glacial Spike 1, ledger actor `combat`; these sessions are cheated and flagged in `cheats.jsonl`). EnsureHotkeys bound the new skills. Blizzard (level 24) was not reachable.

| session (cheated, clvl 18, 57 life, 69 mana, no mana potions) | time | kills | kills/min | XP | XP/h | notes |
|---|---|---|---|---|---|---|
| Blood Moor (Fallen, Zombie, Quill Rat) | 4m11s | 58 | 13.9 | +15 | 215 | mana ended at 1/69, low-mana waits; 0 potions, 0 deaths, 1 target ignored |
| Cold Plains run A | 5 min | about 50 | about 10 | tiny | | helper (port 47014) died after 3 min, fight went on blind; killed by hand |
| Cold Plains run B | 3m36s | 44 | 12.2 | +31 | 517 | 2 potions, kiting from Fallen packs, 1 ignored, 0 deaths |
XP per hour at level 18 in these areas is near zero (monster level 1 to 6): the numbers measure speed, not progression. At level 1 (uncheated, bot runs) docs/bot.md has about 2,700 XP/h.
- **Affixes**: no champion, unique or minion was met in 12 minutes of Blood Moor and Cold Plains (the census printed nothing), except the ordinary Fallen Shaman. So the danger table's behavior on affixes (`Avoid` for Cursed, Mana Burn, Lightning Enchanted, Spectral Hit in Hell below clvl 75; death weights) stays covered by unit tests only. `combatlive farm` prints every elite/affixed/avoided unit once as `AFFIX ...`.
- **Infra**: the d2input helper of i2 died four times in about 40 minutes (every 3 to 8 minutes of heavy input); `tools/game.sh helper i2` revives it. A fight that continues after the helper dies only logs `move failed`; `Clear` should treat repeated "helper unreachable" as fatal (not done: the Controller is not mine).
- **NAV observations** (for NAV2): `ExitTo(1)` from the Blood Moor picked the gate (6279,5347) in the north (contact strip) and failed with "did not change the level" while (6279,5432) works; fixed by NAV2's presets since? Not re-tested after the merge. `ExitTo` from Cold Plains to town is not a neighbor: use `ToArea`.

## VERIFIED LIVE (the worker host, 2026-10-28, "Testy" level 1 Sorceress, Normal)
- Belt reading: 4 columns, GX = column; `db.Items[class]` is weapons+armor+misc in file order: class 587 is hp1 (VERIFIED against the real tables and the live belt: 4 minor healing potions, one per column).
- `Act(DrinkLife)`: belt key "1" drinks (healing count 4 -> 3), Potion event emitted.
- Right-skill popup binding: click the right skill button (660,578), hover the icon, press F8: Fire Bolt bound; checked by picking another skill from the popup and pressing F8 (right skill 219 -> 36).
  `BindSkill("Fire Bolt","f7")` ran the whole find-icon + bind + verify path from scratch. Popup geometry (800x600): rightmost column x=698, pitch 49; bottom row y=487, pitch 47; rows are the
  skilldesc ListRow values the character has skills in (generic skills 0, fire 1, lightning 2, cold 3, tomes 4; empty rows are skipped, row 0 at the bottom); the grid grows left and up. Icon order inside a row is
  found by picking each cell and reading the right skill back (a pick selects that skill: VERIFIED).
- Skill panel facts: Fire Bolt mana cost tooltip 2.5 (matches the data formula); skill list of the new character: 0,2,1,217-220,3 and 36 (level 0 + 1 from the staff).
- Fight: from the Blood Moor entrance (gap x=5960, y=4945..4959) the fighter, with a throwaway BFS mover, killed 4 Quill Rats / a Zombie (Kill events with names and XP; life never dropped below 40/40, mana 35 -> 19),
  hostile filtering (town NPCs excluded, merc none), `Clear` radius and approach, Mover errors tolerated. Monster names resolve from monstats ("Quill Rat", "Zombie").
- Walking back: area 2 -> 1 at x about 5963.
- Not yet verified: potions mid-fight, chicken, town portal, kiting, Static Field, Blizzard/Charged Bolt, affix ids, stat-list resists of a monster (the read ran, values were not inspected), `MonsterInfo.LifePct` on a damaged monster
  (the no-progress watchdog did not fire wrongly, which suggests it moves), dying/respawn/corpse states (since verified, see "Death: VERIFIED LIVE").

## Class profiles and the Paladin loop (agent COMBATCLASS, 2026-10-28)

Tags: **V** verified live on Palli (clvl 13 Paladin, i1, Normal Blood Moor), **U** not verified.

- **Data, not code.** `internal/combat/classdata/skills.json` (per skill: kind bolt/ground/self/buff/melee/aura/summon/charged, hand, reach in subtiles, hold) and `classdata/profiles.json` (per class, Sorceress per build: level bands with a `requires` skill, attack ladder, aura ladder with the element each aura adds, buffs, F-key layout, melee knobs). Mana cost, level and prerequisites come from skills.txt through the Catalog (`SkillSpec.Cost/ReqLevel/Prereq`); auras cost 0 (V). A skill a profile names without a table row is classified from skills.txt (`deriveMeta`: aura, summon, h2h = melee, rng + missile = bolt). Ranges are Kolbot's (U); the skills.txt range column is only none/h2h/rng/both.
- **Sorceress unchanged.** `attackOrder`, `buffOrder`, `Layout` now read the Sorceress profile; every old combat test passes untouched. Skills outside the table and outside every profile (Teleport) stay unknown to combat, as before.
- **Profile selection** (`SelectProfile`): the class the learned skills belong to (skills.txt charclass) wins over `Player.Class`; a profile that uses none of the character's skills gives way to the best matching one. The five non-Sorceress, non-Paladin profiles are U (names checked against skills.txt by `TestRealSkillNames` with `D2LOD_DATA_CACHE=<data cache dir>`; Fissure is "Eruption", Fire Blast "Fire Trauma", Burst of Speed "Quickness", Wake of Fire "Wake of Fire Sentry" there).
- **Hands.** Hotkeys only select the RIGHT skill. `Fighter.SetLeft(ctx, id)` uses the left popup (`LeftPopup`: button (140,578), grid from x=104 growing right, pitch 49, bottom row y=480 = Attack, rows 47 px apart, cells in tree order Row/Column within a skilldesc ListRow, only leftskill=1 skills listed) and verifies by reading `LeftSkill`; the cell that worked is remembered. V: Sacrifice/Smite/Holy Bolt/Zeal/Attack each selected on the first predicted cell, about 1.1 s per switch. `SetRight` uses the verified hotkey (V: about 200 ms, no mana) or the right popup. `EnsureHotkeys` binds the profile's layout (Paladin: Might f1, Holy Fire f2, Thorns f6, Resist Fire f7 bound V).
- **Aura state** (`aura.go`, `Fighter.ActiveAura`): player StatList +0x3c child chain, owner +0x34 = player list, stat 350 = aura skill id (V, CLASSPAL). `KeepAura` re-applies the profile's aura after a death, respawn or area change (`auraDirty`, area compare) and once per `AuraCheckMs` during a fight; it picks the first ladder aura whose element the target is not immune to (Holy Fire vs fire immune gives Might). `Buff` calls it too. Without a readable state it trusts the selected right skill until a death or area change.
- **Melee fight** (profile `melee`): the attack ladder ends in the free basic attack (Paladin: Zeal, Smite, Sacrifice, Attack). A left skill that is not selected costs `manaReserve` extra so the pick does not flip at the cost line; the left skill is selected by the binder only when the pick changes (failures are put aside for `LeftFailSec`). Hold skills (Zeal) go through `input.Game.HoldClick` (button down 600 ms on the monster; V from CLASSPAL: 19 kills/55 s held vs 7/30 s single clicks); others are one left click. No lowMana wait and no mana rest before a pack for melee profiles. `meleeGuard`: back out when `surround` monsters stand within 3 subtiles and life is under `surroundLifePct`, or at `retreatLifePct`, using the kite budget. Approach reach is the profile's `reach` (4). A target with no usable attack skill is ignored with the new reason `no-skill` (was filed as immune).
- **Live results (V, Palli clvl 13, Zeal/Sacrifice/Smite ladder, Holy Fire right, 0 potions, 0 deaths in all runs)**: Blood Moor 6 min first run 88 kills = 14.7/min (4 zombies dropped by the approach walk, fixed afterwards by `meleeGo`; 15-20 s mana rests per pack, removed); fresh Blood Moor after the fixes 63 kills in 2m46s = 22.8/min (explored, 0 ignored); Cold Plains 91 kills in 3m53s = 23.4/min (0 ignored). Hand reference (CLASSPAL, Zeal + Holy Fire, held 700 ms): 19 kills in 55 s = 20.7/min. The bot flips between Zeal and Sacrifice about every 25 s as mana drains and refills (about 1.1 s per left switch). Aura switch by hotkey Might <-> Holy Fire x6: 161-229 ms per key call, aura visible in memory with the key call, mana 33 -> 33. Palli lost life only to 108/154 over 4 min; the first start showed life 0/154 from an earlier test (four potions healed it).
- **Prayer (EARLYFIX2)**: an aura-ladder entry with `restLifePct` (Paladin: Prayer 70, first in every ladder) is a resting aura. `wantAura(p, pr, tg, resting)` returns it only when resting with life under that percent; in a fight it is skipped (Might or Holy Fire stay up) and used only as the last resort when no other aura is known (clvl 4, before Might). `preEngage` calls `restAura` in the RestHere/RetreatRest branch: no hostile within 4 subtiles, then `keepAura(..., resting=true)`; the first fight tick (`KeepAura` with a target) puts Might back. Aura switches are a key press, no cast time (V, above). Hotkey: Prayer = f8 (replaced Cleansing, which nothing used; F1-F8 are all 1.14d has), bound by `EnsureHotkeys` from the profile layout and shown in the `combat bind` line. Attack ladder unchanged (Zeal, Smite, Attack; no Sacrifice). Tests: `TestWantAuraPrayerWhileResting`, `TestRestAuraSwitchesUnlessAdjacent`. U live: whether the melee life rest is ever entered at 70 % (PreEngage thresholds) is not measured.
- **Tuning**: `combat.auraCheckMs`, `combat.leftFailSec`.
- **Live tool**: `tools/rigs/combatclass` (`state`, `bind`, `setleft NAME`, `auras A B N`, `farm SECS [AREA]`, `key K...`), run with `tools/live.sh combatclass combatclass ...` (instance i1 in tools/instances.conf).

## Cast-first loop (agent COMBATLOOP, 2026-10-29)

Tags: **V** from the G1 attempt 13 log and `docs/deathautopsy.md`, **U** not run live (offline fakes only).

Problem (V, user report on Soaky clvl 11, Cold Plains: full mana, packs, circles, no attack): every movement branch of
`engage` (dodge, missile sidestep, curse retreat, keep range, kite, approach, pre-engage rest, low-mana back-off) took the tick
instead of a cast. Frost Nova was never cast (U whether the plan even buys it: `build-fallback-lightning.json` puts the clvl 6
point in Static Field, so Nova is dormant until a point is spent; the code is ready and the lightning layout binds it to f8).

Per-tick order now (fight loop, `castfirst.go`): safety (potions, portal, chicken) -> hard-mode rules -> abort-fight -> scope ->
`castFirst` (cast gap respected) -> pre-engage -> `engage`.
- `castFirst`: if the nearest hostile is within the attack range (longest non-self attack skill, Charged Bolt 10) then
  (1) Frost Nova when 3+ hostiles are within 4 subtiles (bound, affordable, 2.5 s apart, not immune; no step back, the bolts follow),
  (2) the WATCHDOG: no cast for 6 s while a hostile has been in range that long and a skill is affordable, ready and not blocked
  -> cast it at the nearest on-screen hostile and log `combat: forced cast (no cast for Ns, <last non-cast reason>)`.
- `engage`: evasive moves (missile dodge, sidestep, curse retreat, keep range) only within 2 s of the last cast (`evadeOK`), so
  they can never replace casts for long. The step back from a melee monster is the SME rule only: ONE melee monster within 3
  subtiles, life below 60% and falling, at most 2 steps per burst (resets after 10 s), then a cast or a drink; elites and
  "danger" no longer trigger it, packs never do. A target inside the skill's range is shot at (the range margin only applies
  to walking: a Charged Bolt target at 8 of 10 subtiles used to trigger an approach walk). `KindMelee` profiles do not kite
  (the `meleeGuard` of the Paladin stays, gated by `evadeOK`; SME says melee classes never kite, left for the owner to decide).
- Pre-engage never rests while a hostile is in attack range and a cast is affordable.
- No bound attack skill (U: a likely cause of "never attacked" if Charged Bolt was learned but its F4 bind failed): with the basic
  attack on the left button the bot swings it instead of ignoring the target; otherwise the log line `combat no attack skill bound`
  lists the known skills and the key map.
- Nav yield loop: `ClearOpts.Yield` (set by `packYield` in cmd/d2bot and cmd/bossrun) ignores the earlier ignore memory, so the
  nearest blocking hostile is fought at once. When a fight finds nothing in scope with hostiles within 18 subtiles it logs
  `combat no target in scope near= avoid= blacklisted= ignoredEarlier= yield=`.
- Telemetry: `combat fight ...` gains `novas= forced= castsPerSec=`; `combat health killsPerMin= castsPerMin= kitesPerMin= novas=
  forced= damagePerKill= windowSec=` is logged by the first fight that ends after a minute (rolling window, not a timer).
  Debug `combat decision act=... why=...` is written for every non-cast tick (dodge, sidestep, kite, approach, wait, class upkeep).
- Kit gate (`internal/task/gate.go`): the Barracks walk (quest Tools of the Trade) needs 4 healing potions (belt + pack), a town
  portal scroll or tome and, for casters, 2 mana potions. A short kit gives the existing KindResource fault with a chores insert; if the
  chores cannot fix it (no gold) the quest is blocked and the planner levels. No level floor was added.
- Tests: `castfirst_test.go` (pack of 15 at 60% casts and does not walk, Nova then bolts, single melee step back bounded then casts,
  yield engages an ignored monster, watchdog forced cast and no cast out of range, evade gate), `TestBarracksKitGate`.
- Live checklist (not run): Soaky in Cold Plains with the lightning build; grep the run log for `combat fight ... casts=` (>0 every
  fight with kills), `forced cast` (should be rare; each one names the branch that stole the ticks), `combat health`, and
  `no target in scope` around `nav stalled by monsters, yielding`.

## UNVERIFIED
- `Fighter.openPortal`: inventory cell geometry from data.InventoryLayout("Sorceress2") plus hover check; never run. `Fighter.leave` default (Esc + click (400,315)) is a placeholder: install `launch.Flow.LeaveGame` in `Fighter.Leave`.
- Affix ids (`monumod.txt` row ids read from the real file; that the client's Enchants bytes hold them is from D2BS), skill ranges (Kolbot's), Static Field range formula, buff state.
- `HasPortal` assumes any tsc/tbk in the inventory has charges (quantity stat not read).

## Dead ends
- Skill tree (T) route: hovering Fire Bolt in the tree and pressing F8/F7/F6/F5 bound nothing on this character (F8 checked: key did not select the skill). The tooltip says "not learned yet / granted by an item".
  Memory diff of Game.exe data (RVA 0x2E0000-0x5BA000) and of the player's Info + skill structs around a bind found no hotkey table (it is probably on the heap), so binding is verified behaviourally, not by reading memory.
- A key press cannot be verified when the right skill already is that skill (the default right skill is Fire Bolt): `checkKey` first moves the right skill elsewhere through the popup.
- Pressing Esc with nothing open opens the game menu: the binder closes the popup by clicking the chosen icon instead.
- The first naive walk (MoveTo straight at far points) opened an NPC dialog / got stuck in town; the live tool uses BFS over the d2map grid and side steps. That mover is test scaffolding.
- Kiting every target separately burned the budget (and starved casting): the budget is per fight burst now. A cast-gap sleep inside the loop starved survival checks: replaced by a non-blocking gate.

## Live tool
`tools/live.sh combat combatlive <state|probe|bind|bindkey SKILL KEY|popup X Y [KEY]|vtest KEY|drink|fight SECS|home>` (source internal/combat/cmd/combatlive, symlinked at cmd/combatlive like inputcal).
`fight` needs the character in the Rogue Encampment; `home` walks back east along y=4952. The character was left in the town, no panels.

## Crowd rule (agent ROUTEGUARD, 2026-10-30; SME decision EXP, U live)
Trigger: Soaky died at one spot three times (G1 attempts 11, 12, 15; 22 hostiles within the fight radius, life 50 percent to 0 in about 3 s, monster level 8 against clvl 12). The rule is ADDITIVE: it counts hostiles within 8 subtiles (`Crowd`, internal/combat/crowd.go); below `combat.crowdN` (8) the old 55 percent drink / 25 percent chicken rules decide alone, so ordinary packs are not fought more timidly (the G1 attempt 13 lesson: 15 Fallen at 85 percent life died in 2 s).
- crowd >= `crowdN`, life <= `crowdPortalPct` (70), a portal scroll or tome exists and fewer than 3 hostiles are within 3 subtiles: `TownPortal`.
- 3 or more adjacent (a portal cast takes about 2 s and fails when surrounded, U): drink at `crowdDrinkPct` (65); between 65 and 70 the new action `CrowdRetreat` steps away from the crowd (only offered when `awayPoint` finds a step, so the fight loop cannot spin).
- crowd >= `crowdAbandonN` (15) on a travel or quest walk (`Fighter.SetWalking`, set by the bot around StateTraveling), or at life <= 70 anywhere: `CrowdAbandon` = retreat up to 8 steps until 20+ subtiles from the start, then open the portal; no scroll degrades to `CrowdRetreat`. The bot then plans no walk back into that area for 4 x `bot.navSkipSec` and runs a chores trip.
- Chicken (25) still wins; at <= 25 with a rejuvenation potion it drinks first. Tests: `crowd_test.go` (pure table, snapshot, abandon walk). The retreat goes straight away from the crowd (U: no bias toward the entry yet, the Fighter does not know it).
- Autopsy facts and the potion counter fix: docs/deathautopsy.md "Route guard fields".

## Lockstep spin guard (change navstall, 2026-10-30)
Under a lockstep d2d the clock (`d2dinput.Pacer`) only moves on sleeps and steps. Soakfull seed 11 (virtual minute 109) and runab seed 42 (walk-only, about 90 s) froze there: Safety chose `CrowdRetreat`, `retreatStep` walked, the walk failed at once (`nav: no path: empty path`), `Fighter.move` swallowed the error without a sleep, and the pin rule (`noEvadeUntil`, a clock deadline) could never expire because time was frozen. Hundreds of thousands of iterations per second, no tick. Fixes (V offline, see `navstall_test.go`; live result in the change card):
- `move`: a failed walk costs one poll tick (covers every caller: step-away, flee, out-of-mana back-off, kite).
- `canRetreat` is false while pinned: no step-away is offered when it cannot work; the crowd is fought or the abort rules decide.
- Fight-loop watchdog: `noProgressMax` (40) iterations without the clock moving force one poll sleep and log `combat no progress, holding` once per fight.
- Audit: the other `for {` loops in nav, town, loot and bot sleep, are bounded by tries/hops, or end on a deadline; only the combat loop could iterate without sleeping. Shared combat code: needs a G1/G2 (Game.exe) check before it counts as verified there.
