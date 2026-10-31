# Necromancer upkeep (internal/combat/necro.go)

Tags: V verified live (docs/classes/smoke-necromancer.md, run by classnec), S secondary, U unverified. First live run 2026-10-29 by agent necrolive on Necki (i2): results in the section 'Live results' at the end. Elsewhere tests used a hand-built snapshot (necro_test.go).

## What it does
- **Census** `CountMinions(snap, cat)`: living friendly monsters (Friendlies/IsFriendly, docs/friendly.md) minus townspeople and hirelings, by class: skeleton 363, mage 364, golem 289-292 (V monstats rows; V live for 363 and 289), others with child state 105 count as Other (revives, U).
- **Corpses** `Corpses`: monster units in mode 12 (V: 0.3 s after death, stay for minutes), not friendly, not NPC, killable, not a monstats boss (U: bosses cannot be raised), minus corpses already aimed at (V: a used corpse vanishes, a second cast spent no mana).
- **Summons** `KeepSummons(ctx)` (also run at the end of every `Clear` for a Necromancer, and `engage` casts from a corpse within 4 subtiles between attacks): golem first when none stands (no corpse needed, V Clay Golem), then Raise Skeleton, then Raise Skeletal Mage, from the nearest corpse. Walks to the corpse when farther than 4 subtiles (V: 4 worked, 30 did not; exact limit U) and only when no hostile is within 10. Never in town, never when dead. Counts come from the snapshot each time, so area changes, death and game restart need no dirty flag: the census is simply low again.
- **Cap by evidence, not formula**: wanted count is the skill level (U guess, skeletons capped at 12, mages at 4). A cast that spent mana and raised nothing records the cap at the current count (`capSeen`); no further cast of that kind until the count falls below it. A cast that spent no mana is a bad corpse, not a cap.
- **Curses** `Curse` (hooked in `engage` before the attack pick): target = boss or elite in 20 subtiles (V: landed at 11, not at 28), else the hostile with most others within 8 subtiles (needs at least one other), above 25% life. Curse: Lower Resist when physical resist < 50 and magic resist >= 50, else Amplify Damage, Weaken only on boss or elite when Amplify is not learned. Cast at the monster, not the caster (V finding 4: at the feet nothing happens). Recast when the state is gone: Amplify state 9 (V), Weaken 19 (S). A state number is trusted only after it was seen present once after our own cast (`learned`); until then, and for Lower Resist (state number unknown), a 30 s cast memory is used. After 3 casts on one unit with no state, give up on that unit (V: a cast at 2 subtiles registered nothing).
- **Bone Armor by state** `buffDue` (used by `Buff`): state 14 (V as a child-list state; whether `Player.States`, the bit array, shows it is U) is read from `Player.States`. Once seen present it is learned and the armor is recast as soon as it is missing (4 s grace after a cast), because it vanishes when the absorb is used up (V). Until learned, the old `BuffRecastSec` timer. Curses are no longer cast by `Buff` (they went to the caster's feet): profile `necromancer` now lists only Bone Armor as a buff.
- Layout (profiles.json): F1 Teeth, F2 Bone Spear, F3 Bone Armor, F4 Amplify, F5 Raise Skeleton, F6 Raise Skeletal Mage (was Weaken: the mage needs a key or popup and was never cast), F7 Lower Resist, F8 Clay Golem. Fire Golem was added to skills.json. Iron Golem (needs an item) and Blood Golem (costs life) are not used (U).

## Needs a live check (Necromancer instance)
1. `Player.States` has bit 14 after Bone Armor (else the timer stays; harmless).
2. Raise Skeleton from `Clear`: corpse chosen, walk, count rises; real skeleton cap at the skill level (does the first probe cast cost one extra 9 mana?).
3. Curse state 9 on the pack target through `Hostiles` (child-state read on monsters); Weaken id 19 and Lower Resist state number.
4. Raised minions are not hostile (V live earlier) and stay in `snap.Monsters` as friendly.
5. Skeletal Mage class 364 and Fire Golem rows; boss corpses really cannot be raised.
6. The extra casts do not starve the attack loop of mana (mana reserve is 10% of max).

## Interface
`Fighter.KeepSummons(ctx) (int, error)`, `Fighter.Curse(ctx, snap, hostiles) (bool, error)`, `CountMinions`, `Corpses`. Callers need nothing new: `Clear`, `Kill` and `Buff` use them. Cheat: none.

## Live results (necrolive, Necki, i2, 2026-10-29)
Cheats (ledger actor necrolive, all on Necki): `level 14` (memory), then a save edit statpoints 65 / skillpoints 13 (memory cannot create the stat, the game was left through `oog leave` and re-entered with `oog "start Necki"`). A level-up by exp cheat does not trigger (exp set above the next threshold, level stayed 14).
- V default build: `class-aware start ... levelerBuild necromancer_summoner_normal30` (fresh clvl 1 and clvl 14).
- V skill spending follows the table: `levelerlive spend` at clvl 14 with 13 points gave Raise Skeleton 4, Skeleton Mastery 4, Teeth 1, Clay Golem 1, Amplify 1, Skeletal Mage 2, Bone Armor 1 (build rows 2-14); the stat plan was energy 5 then vitality.
- V hotkeys bind for learned skills: Teeth f1, Bone Armor f3, Amplify f4, Raise Skeleton f5, Clay Golem f8 (and Skeletal Mage f6 after the layout fix). Build-file hotkeys are not used; the combat layout is.
- V Raise Skeleton from Clear with the caster beside a corpse: `combat summon ... grew=true paid=true`, count 1 to 4. V real cap: the fifth cast at skill level 4 paid and raised nothing (`minions=4 grew=false`), so the cap equals the skill level there; capSeen recorded it.
- V Clay Golem cast first in a fresh area (grew, paid), golem counted by IsFriendly.
- V Amplify Damage cast at pack members (`combat curse`, dist 9-19). `stateReadable=false` after every cast (the read 1.5 s later did not show state 9; the projectile may land later). `curseUp` now learns the state when it is seen on a cursed unit; U whether it ever shows. Weak spot: a cast costs 4 mana, so unread state only means the 30 s memory is used.
- V `Player.States` (the bit array) shows state 14 after Bone Armor (`before [97 105]`, `after [14 97 105]`), a second `Buff` right after did not recast. So the state path is live; 105 (minion owner) and 97 (Skeleton Mastery) are also in the array.
- V mana starvation: golem + 4 skeletons took mana from 71 to 14-17; the rest-before-pack waited 40 s and timed out twice (13.3 kills/min). Fix: `armyStands` (3 or more minions) skips the mana rest; the same farm gave 17.8 kills/min (not the same pack, so only indicative).
- Fix: every pack member got its own Amplify Damage cast (a cast every 1.2 s). Curses are area effects, so a unit within 5 subtiles (U radius) of a fresh cast now counts as covered (`curseCover`).
- Not seen live: Weaken and Lower Resist (not learned), boss corpses, Mage cast log line, Bone Armor recast after its absorb ran out.
- V census after the layout fix: `{Skeletons:3 Mages:2 Golems:1}` (class 364 counted; mage cap equals its skill level 2, like the skeletons). Lone minions persist across `combatclass`/`necrolive` runs while the game stays up.
- Driver: `tools/rigs/necrolive` (`buff`, `census`, `farm SECS AREA`; a copy of combatclass plus those subs). The bot's own goal queue persists between runs in the data dir (`d2bot --attach` re-ran old goals), so use `necrolive farm` for short probes.

# Druid and Assassin (internal/combat/druid.go, assassin.go; agent druidassa, offline only)

Tags: V = data in this repo (tables, skills.txt via the d2tables cache gear.json, unit tests with fake snapshots), S secondary (Maxroll via docs/classes), U not seen live. No game was touched; nothing here has run live.

## Builds (V tables, U live)
- `druid_fire_normal30` (internal/gear/builds, copies in internal/plandata and docs/plan/data): Maxroll Druid leveling (docs/classes/druid.md, S): Firestorm 1-5 (Den point in row 5), Molten Boulder 6-7, Firestorm 8-11, Fissure (skills.txt `Eruption`) 12+, Firestorm from the Radament (row 15) and Izual (row 21) points. U additions: Oak Sage 16, Arctic Blast 17, Cyclone Armor 18. End at clvl 30: Firestorm 11, Molten Boulder 2, Eruption 16, Oak Sage 1, Arctic Blast 1, Cyclone Armor 1 (32 points). No weapon requirement (all spells).
- `assassin_fireblast_normal30`: Maxroll Assassin leveling (S): Fire Blast (`Fire Trauma`) 2-5, Claw Mastery from the Den point, Burst of Speed (`Quickness`) at 6, Fire Blast to 11, Wake of Fire (`Wake of Fire Sentry`) 12-29 (20 points with the Radament and Izual points), last point Fire Blast. Martial arts was rejected (two-step timing, docs/classes/assassin.md). Claws are not required (V skills.txt: Fire Blast, traps, Burst of Speed have no item type); `Fists of Fire` and the other martial arts need `h2h` (V, TestDruidAssassinSkillsNeedNoWeapon).
- `gear.DefaultBuild` now returns these for dru and ass; `leveler.buildFiles` got the two entries (one-line edit outside the owned dirs, needed or the default fails to load).

## Combat profiles (V profiles.json, U live)
- Druid: F1 Firestorm, F2 Molten Boulder, F3 Eruption, F4 Cyclone Armor, F5 Oak Sage, F6 Raven, F7 Spirit Wolf, F8 Fenris. Attack from clvl 12 with Eruption: Eruption, Molten Boulder, Firestorm, Attack; below: Firestorm, Molten Boulder, Attack. Buff: Cyclone Armor by the `BuffRecastSec` timer (its state number is not known, U). Fire immune targets fall to the next skill, then melee Attack. Heart of Wolverine, Spirit of Barbs, Grizzly have no key (only 8 F-keys) and are never cast unless right-selected.
- Assassin: F1 Fire Blast, F2 Wake of Fire, F3 Burst of Speed, F4 Fade, F5 Shadow Warrior, F6 Lightning Sentry, F7 Charged Bolt Sentry. Attack: Fire Blast, then Attack. Traps are NOT in the attack list. Buff ladder Quickness then Fade (they exclude each other in game, S docs/classes/assassin.md, so one group: Burst of Speed wins when learned).

## Upkeep
- `classTick` (fight.go calls it instead of `necroTick`) and `classUpkeep` (after `Clear`) dispatch by class.
- Druid `CountDruidPets`: friendly units of class 419 raven, 420 spirit wolf, 421 dire wolf, 428 grizzly, 422-424 spirits, 425-427 vines (V which monstats rows are friendly AI, friendly.go; U which skill makes which class). `planPet`: spirits (n-th spirit skill while fewer than n spirits), grizzly, Fenris (max 3), Spirit Wolf (max 5), Raven (max 5), count capped at the skill level (U), learned and bound and affordable only, mana reserve 10%. A paid cast that raises nothing records the cap (`capSeen`, same rule as the Necromancer). `druidTick` casts one pet per fight tick above 40% life; `KeepPets` fills up after a Clear. Never in town. No corpse or walking.
- Assassin `CountAssassinPets`: traps = classes 410-415, shadows = 417 or 418. `assassinTick` lays one trap per tick (ladder Wake of Fire, Lightning Sentry, Charged Bolt Sentry) at the best target: boss or elite, else most neighbours within 8, 3 to skill range subtiles away, above 25% life, not immune to the trap element, and with no standing trap within 5 subtiles (U radius). A cast is remembered 3 s so a slow spawn is not double-cast. The sixth trap replaces the oldest in game (S Kolbot, cap 5), so no cap logic. `KeepShadow` (after Clear) casts Shadow Master else Shadow Warrior when no shadow stands.
- Friendly handling is unchanged: `combat.IsFriendly` already lists the AI rows (AssassinSentry, DruidWolf, ...); no change.

## Live check list (d2-live, one instance per class; U everywhere)
Druid (cheat via cmd/d2cheat on the throwaway only, reason logged): level 14 then 20, `levelerBuild` empty.
1. Start log says `levelerBuild druid_fire_normal30`; `levelerlive spend` at clvl 14 gives the table's rows 2-14 (Firestorm 9 including the folded Den point, Molten Boulder 2, Eruption 3).
2. Hotkeys f1..f3 bind for learned skills; Eruption is chosen from clvl 12 (log `attack` skill); ground cast lands on the monster, not the caster.
3. Cast Raven, Spirit Wolf, Fenris, Oak Sage by hand once each and dump `snap.Monsters` friendly rows: record class ids against the table above (U 420/421, 422-424 mapping) and their `Player.States`/owner words.
4. `Fighter.KeepPets` in the field: count rises per cast, `pet ... grew=true paid=true` log; find the real caps (ravens 5? wolves at skill level?); confirm a paid no-growth cast stops further casts.
5. Cyclone Armor state number (`Player.States` before and after), whether it vanishes when absorbed; then switch it to state-based recast like Bone Armor.
6. Mana: summons plus Fissure at clvl 14-20, does the 10% reserve starve the attack (compare kills/min with pets off).
Assassin: same, `assassin_fireblast_normal30`.
1. Start log and spend table: clvl 6 gives Quickness (needs Claw Mastery from the Den point in row 5: check the planner pulls the quest point forward).
2. Fire Blast from f1 at range <= 8; Burst of Speed cast by `Buff` (state number U).
3. From clvl 12: `trap` log lines, trap units appear (classes 410-415?) with `IsFriendly`, no trap re-laid on a burning pack, sixth trap behaviour at 5 standing, trap lifetime (do they expire? sets how often to relay), does a trap placed at 3-4 subtiles hurt us.
4. Wake of Fire needs the target in sight: corridor behaviour (Maggot Lair is a known weak spot).
5. Shadow Warrior (needs clvl 18 and skills outside this build; spend by hand with a cheat): `shadow` log, class 417 counted, one at a time.
6. Wear claws vs a staff: nothing changes for these skills (V offline); martial arts is not wired.

## Assassin live results (agent assalive, i3, Assi, 2026-10-29, no cheats)
One `d2bot --attach --goals reach:6` run of 20 minutes (config `ignoreTownAdvice`), Blood Moor, character level 1 to 4 (exp 4020), 279 kills, 0 deaths, about 10,000 XP/h. Level 6 (Burst of Speed) and level 12 (traps) were NOT reached, so checks 2 (Burst of Speed), 3, 4, 5 and the trap class ids 410-418 are still U.
- V: start log `class-aware start ... combatProfile assassin levelerBuild assassin_fireblast_normal30`.
- V (check 1, first rows): spend at clvl 2 and 3 = +5 energy +1 Fire Trauma each, clvl 4 = +5 vitality +1 Fire Trauma, `statLeft 0 skillLeft 0`, done in town after the level up (the bot walks home to spend). Skill name `Fire Trauma` is Fire Blast in skills.txt.
- V: hotkey `Fire Trauma` bound to f1 right after the first spend (`combat hotkey ok`).
- V: Fire Blast attacks work; at 25 mana the cast budget makes the fighter rest ("rest to cast budget", mana 8 of 21 target, waits 12-38 s) for the first levels. Energy points per the table help here.
- Observation (not changed): the 20 min run included 3 save-and-exit cycles (town trips with no portal: walk back, level changes) and one failed ring equip (`item did not reach slot rHand`, leveler).
- First town visit of a fresh character: Akara's greeting dialog stays open and the chores stall until it is closed; the second run got past it (the bot closed it), the first run sat 3 minutes. Not investigated further.
## Druid live results (druidlive, Druki, i2, 2026-10-29, no cheats)
Played honestly with `d2bot --attach --goals reach:8 --autostart` (two bg runs of 22 and 26 minutes, driver `tools/rigs/druidlive`: state, bind, buff, census, pets, farm). Reached clvl 4 (exp 6360) from clvl 1; one chicken exit and a failsafe save-and-exit/re-enter cycle in each run (the bot recovered by itself). The worker host load was about 20 on 16 cores, so progress was slow. Everything below the clvl 4 line is NOT reached: pets (first pet in the build is Oak Sage at clvl 16), Eruption (clvl 12), Cyclone Armor (18), caps, mana comparison.
- V start log: `class-aware start ... combatProfile druid levelerBuild druid_fire_normal30` at clvl 1.
- V skill spending: clvl 2, 3, 4 each `+1 vitality x5, +1 Firestorm` (build rows 2-4, Firestorm 3 at clvl 4).
- V hotkey: `combat hotkey ok skill Firestorm key f1`; `bind` gives `map[225:f1]`; right skill reads 225 (Firestorm), left 0.
- V profile at clvl 4: attack list `[Firestorm Molten Boulder Attack]` (Molten Boulder not learned yet, skipped by the learned filter).
- V `IsFriendly` in town lists only town NPCs (classes 148, 152, 179); `CountDruidPets` is zero there, no false pets.
- Observation (U, not fixed, outside this package): the stat plan puts every point into vitality, so mana stays 20-26 at clvl 1-4 and Firestorm drains it (a `combat potion kind mana` at mana 4, town trips for potions with 0 gold). An early Energy share would help a caster build; leveler stat plans belong to the leveler owner.
- Still open from the check list: items 2 (Eruption choice, ground cast), 3 (pet class ids), 4 (caps), 5 (Cyclone Armor state), 6 (mana reserve). They need about clvl 16-18 or a hand-spent point on a pet skill; neither is honest within the time spent. `druidlive pets` and `druidlive buff` are ready for it.

# W5 and W6: corpse spells and trap planner (agent classcombat, offline only)

Tags: V = unit tests with fake snapshots, U = not seen live. No game was touched.

## Necromancer (internal/combat/necro.go, necro_corpse.go)
- `Corpses` hardened: duplicate ids and units at position 0,0 (unread, U) are skipped.
- `planSummon` now ends with **Revive**: after golem, skeletons and mages, when `Other` (state-105 minions that are not skeleton/mage/golem) is below `wantMinions("revive")` (skill level, max 8, U). Corpse = `bestRevivable`: highest monstats level in reach (U heuristic), nearest on ties. The cap is learned by evidence like the skeletons (`capSeen["revive"]`, a paid cast that raises nothing).
- **Corpse Explosion** `corpseExplode` runs from `necroTick` only when no summon is planned (never eats an army corpse). Corpse = most hostiles within `ceRadius` 6 (U: real radius is a skills.txt calc), a boss or elite counts 3, at least 2; one use per corpse (id blacklisted).
- Skill ids come from skills.txt through `KnownSkills` by name ("Revive", "Corpse Explosion"); not learned or not bound = never planned. Both skills exist in classdata/skills.json (summon and ground).

## Assassin (internal/combat/trap.go, assassin.go)
- `assassinTick` -> `planTrap` (pure). Skills tried by skill level, ladder order on ties: Death Sentry, Wake of Fire Sentry, Lightning Sentry, Charged Bolt Sentry (learned and bound only).
- **Cap 5 per skill** (V Kolbot S, U in 1.14d): a skill at five standing plus pending traps is skipped and the next skill is tried. A trap older than `trapLifeSec - 4` (30 s, U) no longer counts or covers.
- **Lay point**: centre of the pack (hostiles within 8 of the target) when it is in range, else the target. Coverage is per skill (a Wake trap does not stop a Lightning trap); a trap of unknown class covers every skill.
- **Death Sentry** is laid on a corpse that has a hostile within `ceRadius`, in skill range and at least 3 away, with no own Death Sentry within 5; never without a corpse.
- **Class ids stay U**: `trapClassGuess` (trap.go) is empty. The class of a skill is learned at run time: a new trap unit (410-415 census, friendly.go) within 3 subtiles of exactly one cast of ours teaches `class -> skill`. Unlearned classes count only for coverage. If no cast is ever matched by a unit (ids wrong or sentries not listed), casts are remembered for the full lifetime so the cap still holds. Shadows (417, 418) are not traps (`CountAssassinPets`).
- Other classes: `assassinTick`/`necroTick` return at once for them (TestOtherClassesNoTrapTick); existing necro, druid, assassin tests unchanged.

## Live checks (needs a Necromancer and an Assassin instance, ask the main agent)
1. Necro with Revive: does the corpse cast raise a unit counted as state-105 `Other`? Real count cap per level; revive on a boss/elite corpse?
2. Corpse Explosion: radius and whether the explosion on a pack corpse kills; mana vs army upkeep.
3. Assassin: do sentries appear in `snap.Monsters`? Read the `trap` log lines (class of each skill; log `trap learned` is not printed yet, read `f.trap.classSkill`), fill `trapClassGuess`.
4. Real sentry lifetime and cap per skill (lay six Lightning Sentries: does the first vanish?).
5. Death Sentry on a corpse: fires on enemies near it? Range 15 is Kolbot (S).
## classlive3 (2026-10-29): not run
The worker host load was 25.7 (limit 24 for starting i3), so i3 was not started and nothing was driven live. Character map (docs/multi.md, one character per instance): Assi (assassin) lives in i3, Druki (druid) in i2, which belongs to another agent; a Druid cannot be tested on i3 without creating a character (not allowed). The open Assassin checks (Burst of Speed at clvl 6, trap class ids 410-418 from clvl 12, Shadow Warrior at 18) and all Druid checks stay U. Plan when load allows: `tools/game.sh start i3` + OOG login of Assi, d2cheat level to 12 then 18 with a reason, `d2bot --attach --goals reach:13`, dump `snap.Monsters` friendly rows.
