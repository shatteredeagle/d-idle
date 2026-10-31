# 04 Combat

Sources: KB:core/Attack.js (kill 295-427, clear 813-1082, sortMonsters 1689-1773, validSpot, findSafeSpot 1906-1956, deploy, skipCheck 2039-2265, checkResist 2332-2410, getIntoPosition 2470-2590), KB:core/Attacks/Sorceress.js, KB:core/Skill.js, KB:core/GameData/SkillData.js, KB:core/Precast.js, SP:Core/*Overrides, KO:internal/action/step/attack.go + character/*.go, TY:docs/features/sorceress-blizzard.md and route-threat-combat.md.

## 1. Two loops
1. `kill(boss)`: single target. Find unit (poll up to 2 s) else if `already in killed-set` return true else `clear(10)`. Loop while attackCount < MaxAttackCount (300; SP 1000), target attackable, not skipped:
   - if the unit went invalid (reference lost, e.g. raised by Necro): refetch by gid; else walk the path to the last known location checking each node; fail if still lost.
   - optional dodge: if `Dodge` and hp% <= `DodgeHP`: `deploy` to a safer spot (see 5).
   - every 15 attacks and range < 4 skill: flash (resync).
   - result = classAttack.doAttack(target, preattack every 15th). FAILED: up to 3 retries with flash; CANTATTACK (no usable skill): stop; NEEDMANA: continue without counting; else counts and resets retry.
   - afterwards `afterAttack` (re-precast) and pick up items.
2. `clear(range, spectype, bossId, sortfunc)`: area clear around origin (player, or boss position).
   - Build list of attackable monsters not skipped (spectype mask filter: 0 = all; 0xF skip normal; 0x7 champions/bosses; flags: 0x1 minion? 0x2 champion... values per monster spec flags). Start only when at least one is within `range` of origin and reachable (teleporters ignore walls, walkers need no WallOrRanged collision).
   - Loop: sort list (sortfunc), take first, re-fetch by gid; attack if within range of origin (or "scary" > 7 and within `range` of me) and attackable.
   - Per-monster attack counter; `FAILED_POSITION` result re-queues the monster (up to 15 attacks); a non-special monster still alive after 15 attacks is dropped (except in Throne of Destruction); if class attack fails: retry 3 times then drop (Storm Casters behind walls in Chaos dropped immediately); every 5 attacks (15 for specials) of a melee skill flash to unstick.
   - Barbarian uses secondary skill index 2/4 if target is immune to primary; Paladin Blessed Hammer position check.
   - After loop: afterAttack, open chests within range, pickit; if no attacks this call just re-precast.
   - Returns SUCCESS (attacked), NOOP (nothing), FAILED (boss still alive).
   Constants: default range 25; Config.ClearType 0xF; `PickRange` 40.

## 2. Target selection (`sortMonsters`) - ordering rules
Comparator (negative = attack first):
1. In Catacombs 4: Andariel if < 5 away and clear line -> first. In Durance 3: same for Mephisto. In Worldstone Chamber: Baal first.
2. Barbarian only: monsters immune to the skill we would use go last.
3. Monsters under "Attract" curse (helping us) go last.
4. A list of "dangerous support" classids come before everything else: Oblivion Knights, all shamans (Fallen, Carver, Devilkin, Dark, Warped, Rat Man, Fetish, Flayer, Soul Killer, Stygian Doll shamans), Hollow One, Guardians, Unravelers, Ancients, Baal Subject Mummy, Blood Raven, Flesh Spawners, Stygian Hag, Grotesques. Among two of them: uniques first, else nearest. (Reason: shamans resurrect packs, Oblivion Knights cast Amplify/Iron Maiden/Life Tap, Unravelers raise, etc.)
5. Optional `BossPriority`: super-uniques first.
6. Otherwise nearest first.
"Scariness" score (for override of range): minion +1, champion +2, unique +4, summoner type +8.
Special Diablo sort (`Common.Diablo.sort`): by y descending when far above/below the star, x descending near west, etc.: it prefers monsters farther along the direction of travel (cleaning the corridor ahead). Not needed for MVP.

## 3. Skill selection and immunity handling
Config model (Kolbot): `AttackSkill = [preattack, bossTimed, bossUntimed, otherTimed, otherUntimed, immuneFallbackTimed, immuneFallbackUntimed(, 7,8 extra)]`, `LowManaSkill=[timed, untimed]`. "Timed" = skill with a cast delay (Blizzard, Meteor, Hydra, Frozen Orb: state `SkillDelay`), "untimed" = fill skill used while the timed one cools down. Index = special (unique/champion/player) ? 1 : 3.
`decideSkill(unit)` (Sorceress):
```
timed   = custom skill for this monster else AttackSkill[idx]
if immune?(unit, timed) false OR no valid spot: timed = AttackSkill[5] if not immune, else none
untimed = AttackSkill[idx+1] same logic with [6]
if mana < cost(timed) and LowManaSkill[0] set and not immune: timed = LowManaSkill[0]; same for untimed
```
`doCast(unit, timed, untimed)`:
```
if neither: CANTATTACK
if timed usable (not on delay or not timed) and cost < mana:
    if melee-range skill and no valid spot: FAILED
    if too far or blocked: getIntoPosition(range)   (walk allowed for melee <10 away); fail -> FAILED
    cast
elif untimed usable and cost < mana: same
else: if no mana -> NEEDMANA else wait up to 1 s for skill delay to end
```
Immunity (`checkResist(unit, skillOrElement, maxres=100)`): element from skills.txt `etype` index [physical, fire, lightning, magic, cold, poison, none]; special elements: Holy Fire/Freeze/Shock = fire/cold/lightning, Corpse Explosion/Stun/etc = physical, Holy Bolt only works on undead (monstats lUndead/hUndead).
- Resist stat ids per unit: physical (DamageResist), magic, fire, lightning, cold, poison (memory doc: monster stat list; for monsters these are in the unit's stat list; immune if >= 100).
- Static Field special: castable only while target life% > `CastStatic` (SP: classic 15, Normal 25, NM 33, Hell 50) and monster not lightning/phys-immune... (it ignores resist, hits physical? In 1.14d Static is lightning; Kolbot treats via `checkResist` + life%).
- Immunity-break aware: with Conviction/Infinity or Lower Resist active the effective resist is reduced: Infinity merc/aura: fire/light/cold treated as immune only if >= 117 (conviction -> maxres); Lower Resist curse reduces by floor((lowerResPct + 85)/5)... (details KB `checkResist`). For our MVP: immune if resist >= 100; add `-X` for known debuffs later.
- `SkipImmune = ["fire and cold", "cold", ...]`: monsters immune to ALL listed elements (for "and") are skipped entirely (not attacked; merc may handle). SP uses `["fire and cold"]` for Blizzard/Fire Ball builds, `["cold"]` for cold-only, `["lightning"]` for lightning builds, etc. `SkipEnchant` (e.g. "lightning enchanted" when light res < 75) and `SkipAura` (fanaticism, conviction, holy fire/freeze/shock, might, blessed aim) skip dangerous monsters; `SkipId` e.g. Fire the NAS. `AdvancedSkipCheck` allows custom rules. Boss fights ignore skip rules (Throne of Destruction too).
- Teleport-stomp: if the sorc cannot hurt a boss (all skills immune) but physical works via merc: stand by merc; (needs merc present, else CANTATTACK).

## 4. Ranges, positions
Skill range table (KB SkillData, yards): Fire Bolt/Ice Bolt/Ice Blast/Glacial Spike/Frozen Orb 20; Charged Bolt 10; Lightning/Chain Lightning 25; Fire Ball 20 (PvP 40); Fire Wall/Meteor/Blizzard/Hydra 30 (PvP 40); Nova 7; Frost Nova 5; Teleport 40; Telekinesis 40; Static Field floor((lvl+4)*2/3); Inferno (17+3*lvl)/4*2/3; Enchant 40; melee 3 (bow 20). Mana cost: `max((mana + lvlmana*(lvl-1)) * shift, minmana)` with `shift = 2^manashift / 256` (skills.txt `mana`,`lvlmana`,`manashift`,`minmana` -> `internal/data`). Durations: Frozen Armor (12*lvl+108) s plus synergies; Energy Shield 84+60*lvl; Thunder Storm 24+8*lvl; Enchant 120+24*lvl; Shiver 12*lvl+108; Chilling 6*lvl+138 (seconds; `Precast.needSoon` refreshes at < 25 percent remaining).
`getIntoPosition(unit, distance, coll, walk)`:
```
if distance < 4: melee: walk/move onto the unit (walk if >8 away or blocked); success when no collision between
angle = direction from unit to me; for 3 rounds (distance shrinks by floor(full/3 - 1) each round):
   candidates = points at that distance on angles [0, +-15, +-30, +-45, +-60, +-75, +-90, 135, -135, 180] from angle, valid (no wall at spot, not within `distance` of me when forced)
   sort by distance to me; for each: require clear line (no `coll` collision between spot and unit); for walkers skip if walking path longer than direct distance; move there; success on arrival
fail -> "no valid positions"
```
Position rules: casters keep line of sight (Ranged flag 0x4 absent between caster and target). For skills that hit areas/missiles (`Skill.missileSkills`), `validSpot` returns true (don't care about floor), except Blizzard/Meteor need floor in non-floor areas (Arcane, River of Flame, Chaos, Abaddon, Pit, Infernal) and must be aimed at flying-less monsters.
Melee/Hammerdin: stand at fixed offsets around target ((6,11),(0,8),(8,-1),(-9,2),(0,-11),(8,-8) in KB Diablo hammer pre-attack).

## 5. Kiting / dodging (`Config.Dodge`)
- Trigger: `Dodge` on and hp% <= `DodgeHP` (SP: 90 hardcore, 75 softcore; DodgeRange 15; in early builds DodgeRange 7).
- `findSafeSpot(unit, distance, spread, range)`: if no monsters within 15 of me -> stay. Else build a grid around the unit (spread 5 spacing, +-distance), sort by farthest from unit, accept cells without BlockWall and with clear ranged line to unit, score = number of monsters within `range` of that cell (+100 per fire object within 4), pick lowest, early exit on 0. Then `moveToUnit(spot)`.
- Boss-specific: Mephisto sidestep radius 18 (see 02), Duriel/Andariel fixed spots, Diablo none, Baal corner waits for hydras.
- KO: moves to attack distance range min..max per attack step (e.g. Blizzard at 25..30, Static Field 1..5, Fire Ball 1..25, Ancients 8..10), re-evaluated every cast, and `stand still` key (shift) held while clicking so the character casts without walking: **1.14d: hold Shift while left clicking attack to attack in place**: UNVERIFIED but classic D2 behavior; Kolbot uses shift for `Left` hand skills (LeftDown + Shift) and no-shift for the right hand.

## 6. Precast (buffs) (`Precast.doPrecast(force)`)
- Called after arriving in each area via WP/portal, after every `clear`/`kill` (`afterAttack`), and after revives.
- Per class (Sorceress): Thunder Storm, Energy Shield, Cold/Frozen Armor, Enchant (party).
  Necro: Bone Armor (refresh at <75 percent remaining of absorb), golem, raise army. Paladin: Holy Shield if shield equipped. Barbarian: Battle Command, Battle Orders, Shout (from weapon swap slot with best +skills). Druid: Cyclone Armor, Raven, summons, Hurricane. Assassin: Fade, Venom, Blade Shield, Burst of Speed. Amazon: Valkyrie.
- CTA (Call to Arms: Battle Command + Battle Orders) cast from the other weapon slot: switch, cast BC twice (it needs two casts to apply?), BO, switch back; recast when `!BattleOrders state` or remaining < 25 percent.
- Buff validity by state flags: `me.getState(stateId)` after cast; track last-cast time and duration; recast when state missing or `remaining < 25 percent` of computed duration.
- Skill hotkey handling: right-hand skill set via `setSkill`; casting = select skill (hotkey), right-click on target; wait for `attacking` mode to start within 200 ms (3 tries) then wait until finished. Timed skills: afterwards poll up to 100 ms for `SkillDelay` state.
- Mapping: states list = `stats`/states bitmap in the player unit (memory doc: "state" array on UnitAny? UNVERIFIED offset). Skill durations/costs from `internal/data`.

## 7. Merc cooperation
- Merc kills help: `TeleStomp` waits for merc to wear down immune bosses (merc within 7 of target else moveToUnit + safe spot), merc-revive only once.
- Merc is a unit with owner = me; HP% read via `getMercHP` (0 when dead). Chicken/ potion thresholds in 07.
- Merc position: no active control; it follows. Avoid leaving it behind: teleporters wait? (not handled by Kolbot; SP adds `MercWatch` to revive).
- TY: casters use merc to kill cold-immune monsters; no idle penalty for lack of damage.

## 8. Dangerous monster avoidance
- Skip lists above. Plus: `Attack.canAttack(unit)` checks that at least one configured skill element is not immune for that monster class (special vs normal).
- "Charged-skill" use: slow-missiles, weaken, lower-resist charged items (`ChargeCast`): cast on monsters with spectype mask or specific classes before main attack if state missing.
- Ice/poison: Andariel (poison), Duriel (cold slow), Mephisto lightning/cold and frozen; thawing/antidote potions (SP buys 2 thawing when frozen at healer; antidote for Andariel and Act5).
- Elite modifiers: Lightning Enchanted + low lightning res = skip; Mana Burn / Cursed = avoid for casters; Fire Enchanted explode; Extra Fast (ES/EF) dangerous for melee.
- Avoid corpses exploders, dolls (Soul Killer: `AvoidDolls` clears them only by list), Burning Souls near Baal.
- Environmental: fire objects (Hephasto/Diablo "fire"), flames of River of Flame lava (non-floor), hydras (treated as monsters of other owner - move away).

## 9. External-design notes (our constraints)
- `doAttack` in Kolbot knows `unit.hpPercent`, resist stats and states directly. Our `d2` must provide monster stat list (resist ids per monster; stats in the unit stat list: hp stat 6 in 1/256 units and maxhp 7) and state flags. TY's D2R world model had no monster HP; ours should (memory doc 2.4).
- Casting = (1) ensure right skill selected: read selected skill from memory (mem 2.8), press its F-key hotkey if different, (2) mouse move to target screen position, hover check on monster unit (mem 2.12; TY uses spiral search around body anchor 2 tiles above the tile center) before pressing, (3) right-click or shift+left-click, (4) wait for mode "casting" then back to neutral.
- Cast cadence: Blizzard `localdelay` = 45 frames at 25 fps = 1.8 s (TY) so never click faster; use `SkillDelay` state if readable.
- Out-of-range monsters outside the visible 800x600 viewport cannot be clicked: project then clamp or move closer (TY: range gate of 22 -> approach to 15 for bosses).

## 10. Mapping
| need | package | data |
|---|---|---|
| monster list, sort, skip rules | `bot/combat` | monster unit: classid, mode, spec flags (minion/champion/unique/super-unique/ghostly), enchants list, aura states, stats (hp, resists), position, owner |
| skills table | `internal/data` | skills.txt: id, etype, range hints, mana params, delay (`delay`/`localdelay`), state |
| position finder | `nav` | collision grid from d2map, line-of-sight test on that grid (Bresenham with flags) |
| precast | `bot/combat` | player states, skill levels, equipped swap slot (for CTA) |
