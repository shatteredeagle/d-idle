# Barbarian skills: Bash and its family, Whirlwind, Frenzy, Howl, the war cries, Berserk, the passives; game.exe 1.14d vs d2d vs the bot (card skbarb, coverage gap skills.bar)

> status: card skbarb, 2026-11-02, offline (no game, no d2d slot, no Go change). First pass over the Barbarian rows (docs/coverage/rule-coverage.md: skills.bar 0 of 30 before this card). Method of docs/research/nec-skill-trace.md (calc cells through game.exe's own calc machine against the real d2d rules) plus runs of the unit-search and ring code of the war cries. d2d = the upstream main checkout the earlier traces name (72132b99); the d2d-side calc test is the hand build of nec-skill-trace.md section 1 (same scratch build), the d2d code quoted below was read in the main checkout (not re-verified to be the same commit: U).

Tags: **V run** game.exe's own code run under the emulator by this card (scripts named, counts given); **V read** read in the disassembly (`tools/emu/skbarb_dis.sh`, upstream's `disasm.py`) or in d2d's source at file:line, never offered as proof of a d2d deviation by itself; **D** from the patched 1.14d tables (`Skills.txt`, `Missiles.txt`); **S** an upstream doc or an earlier trace of ours; **I** inference; **U** not verified. Dead ends are in section 8.

## Summary

- **The calc cells of all 30 Barbarian rows equal d2d (V run, 0 differ of 27,800 cases)**: calc1..4, petmax, sumsk, delay, aura range / length / period, aura and passive stat calcs, plus the skill's own physical min / max (FUN_00647bc0 / FUN_00647d00, weapon share off) against d2d `eval_calc` and `skill_phys`. Levels 1..40 alone and with every neighbour at 20 (clvl 30 and 99, masteries 0 and 50), plus 20,000 random cases (seed 7) and 3,000 (seed 1). So the numbers the plan relies on (Bash / Concentrate / Frenzy / Berserk damage percent, Battle Orders / Shout / Battle Cry lengths and shares, the masteries, Iron Skin, Natural Resistance, War Cry's 20-30 .. 240-250 physical) are the same on both sides.
- **Six d2d deviations found (D1 to D7), all around the Barbarian's area attacks and Howl**, none with a differential against the real d2d build yet (the game side is V run, the d2d side is V read or a model), so **no PR candidate**:
  - **D1 Whirlwind's target search radius**: game.exe hands the whirl hit the next unit within **5 subtiles** (one cell, V run: 108 cases, 0 differ from the integer disc); d2d takes everything within **5 cells** (25 subtiles, V read fight.cpp:2356, 2360).
  - **D2 Whirlwind's hit timer** is anchored differently (V read both sides).
  - **D3 / D4 Howl**: the fear lasts `Param5 + (L - 1) x Param6` = **75 + 25 (L - 1) frames** in game.exe (V run of the readers, V read of the call), d2d uses `Param3 + (L - 1) x Param4` = 24 + 5 (L - 1) (at least 25); the level gate is `monster level < Param2 + clvl + L` (Param2 = 1) against d2d's `clvl + 2 (L - 1) + L`.
  - **D5 Howl's ring**: velocity override (reach 3.6-4.5 subtiles at L1, **15.0-15.7 at L20**, V run); d2d adds calc1 to the range, so its reach stays at the L1 value (I, arithmetic).
  - **D6 War Cry / Battle Cry** are NextHit rows (NextDelay 6 / 4): the existing nova finding (sk-nova-trace D1) applies; the model on game.exe's real flights gives **0.5 hits per monster per cast on game.exe's rule against 4.3 on d2d's** (8x; at one subtile 1.0 against 8 to 34).
  - **D7** a curse-resistance cut and immunity on Howl's duration (game, V read) that d2d does not show (U).
- **Frenzy's stacking (counter to the skill level, duration auralen, aura stats at the counter) is the same on both sides (V read, FUN_005d8c70 against fight.cpp:4362).**
- **Bot (B1 to B5):** the War Cry build's main attack from clvl 30 is typed `buff, range 0` and no code path casts it as an attack; the class option table lists Berserk as a fire skill; Whirlwind and Berserk are not in the profile's attack bands.
- **Ledger:** `skills.bar` 30 rows TESTED-0-DIFF on the calc and physical aspect (27,800 cases). The deviations stay in this doc (no row earns TESTED-DEVIATION without a run against the real d2d rules).

## 1. How it was run

| Piece | What | Tag |
|---|---|---|
| `tools/emu/skbarb_calc.py` + `skbarb_test.cpp --calc` + `skbarb_run.sh N SEED` | `nec_calc.py` for Skills 126..155: game.exe's FUN_00646ca0 on every calc pointer (record +0x138..0x144, +0xc0, +0xd0.., +0x190, +0x64, +0x60, +0x128, +0x68.., +0xa4..) and FUN_00647bc0 / FUN_00647d00 (skill physical min / max, flag 0) vs d2d `eval_calc` and `skill_phys` on the loaded patched tables; stat getter hooked as in skill_damage.py (stat 12 = clvl, 0x149..0x14c masteries, else 0) | V run |
| `tools/emu/skbarb_ww.py` | FUN_0056bd10 (the whirl hit's target picker) called as FUN_005d9580 calls it, in upstream's Game (skg_blast.py's harness): a 19 x 19 block of one monster a subtile round the caster, radius 1..9, chain of calls with the last id; random fraction words | V run |
| `tools/emu/skbarb_howl.py` | FUN_004efc20 / FUN_004cc7c0 / FUN_004efcb0 (Howl's three record readers) for levels 1..40 against the record's parameter words; the parameter words of all 30 rows | V run |
| `tools/emu/skbarb_ring.py` + `sk_nova_ring.py` helpers | the 64 ring missiles of War Cry (row 235), Battle Cry (219), Battle Orders (237), Shout (149), Howl (148) made by the real FUN_0059fa30 from params as FUN_0056d400 builds them and stepped by their real srvdofunc; part C of sk_nova_ring.py (a model, I) for the NextHit rule | V run (flights), I (hits) |
| `tools/emu/skbarb_dis.sh` | disassembly by address | V read |

Runs (the worker host): `skbarb_run.sh 3000 1` = 5,400 cases, DIFF 0; `skbarb_run.sh 20000 7` = 22,400 cases, DIFF 0 (27,800 total; the first run, before the physical columns were added, was also 0 on the 27,800 calc cells). `skbarb_ww.py 1` and `7`: 54 + 54 = 108 (site, radius, fraction) cases, 0 differ from `dx^2 + dy^2 <= r^2`, 0 laps not ascending, nothing returned when nothing is in reach. `skbarb_howl.py`: 40 levels x 3 readers, 0 differ from the record words. `skbarb_ring.py 235 154 6 4 1` and `... 4 7`: 2 x 4 spots x 2 levels x 64 = 1,024 War Cry flights, all 64 of 64 made, all 16 frames, end distance 5.39 to 6.40 subtiles, level-independent; Battle Cry 256 flights (12 frames, 3.61-4.47), Battle Orders and Shout 256 each (15 frames, 18.38-19.31), Howl 256 (12 frames; L1 3.61-4.47, L20 15.00-15.65).

## 2. Questions and answers

**Q1. What do the calc cells say, by skill level? (V run, clvl 30, no synergy)**

| Skill | cell (Skills.txt expression, D) | L1 | L5 | L10 | L20 | L30 |
|---|---|---|---|---|---|---|
| Bash 126 | calc1 ED% = ln12 + Stun x par8 / calc2 flat | 50 / 1 | 70 / 5 | 95 / 10 | 145 / 20 | 195 / 30 |
| Leap Attack 143 | calc1 ED% = ln34 + Leap x par8 | 100 | 220 | 370 | 670 | 970 |
| Concentrate 144 | calc1 ED% (+ Bash, Battle Orders synergy); aura armor % (ln34) | 70 / 100 | 90 / 140 | 115 / 190 | 165 / 290 | 215 / 390 |
| Frenzy 147 | calc1 ED%; length par7 = 150 frames; at the full counter L: run speed % / attack rate % | 90; 47 / 7 | 110; 110 / 25 | 135; 142 / 34 | 185; 171 / 42 | 235; 183 / 45 |
| Whirlwind 151 | calc1 ED% (negative at low levels) | -50 | -18 | 22 | 102 | 182 |
| Berserk 152 | calc1 ED%; calc2 = state length in frames (shorter with level); calc4 = 100 (all to magic) | 150; 68 | 210; 50 | 285; 41 | 435; 33 | 585; 30 |
| Double Swing 133 | calc3 attack rate bonus (par5) | 50 | 50 | 50 | 50 | 50 |
| Howl 130 | calc1 = 2 (L - 1) | 0 | 8 | 18 | 38 | 58 |
| Taunt 137 | to-hit % and damage % on the target | -5 | -13 | -23 | -43 | -63 |
| Shout 138 | length (frames) / armor % | 500 / 100 | 1500 / 140 | 2750 / 190 | 5250 / 290 | 7750 / 390 |
| Battle Orders 149 | length / max mana, max life, stamina % | 750 / 35 | 1750 / 47 | 3000 / 62 | 5500 / 92 | 8000 / 122 |
| Battle Command 155 | length / +all skills | 125 / 1 | 1125 / 1 | 2375 / 1 | 4875 / 1 | 7375 / 1 |
| Battle Cry 146 | length / armor % / damage % | 300 / -50 / -25 | 540 / -58 / -29 | 840 / -68 / -34 | 1440 / -88 / -44 | 2040 / -108 / -54 |
| War Cry 154 | skill physical min - max | 20 - 30 | 44 - 54 | 76 - 86 | 150 - 160 | 240 - 250 |
| Grim Ward 150 | calc1 (duration) / radius subtiles | 1000 / 3 | 1000 / 7 | 1000 / 12 | 1000 / 22 | 1000 / 32 |
| Leap 132 | calc1 knockback / distance range | 4 / 7 | 8 / 17 | 13 / 21 | 23 / 25 | 33 / 27 |
| Find Potion 131 / Find Item 142 | calc1 chance % | 15 / 13 | 50 / 32 | 68 / 42 | 84 / 51 | 91 / 55 |
| Sword, Axe, Mace Mastery | attack rating % / damage % / critical % | 28 / 28 / 5 | 60 / 48 / 17 | 100 / 73 / 23 | 180 / 123 / 29 | 260 / 173 / 31 |
| Pole Arm, Throwing, Spear Mastery | the same, +2 attack rating | 30 / 28 / 5 | 62 / 48 / 17 | 102 / 73 / 23 | 182 / 123 / 29 | 262 / 173 / 31 |
| Iron Skin 145 / Increased Speed 148 / Increased Stamina 141 | armor % / speed % / stamina % | 30 / 13 / 30 | 70 / 28 / 90 | 120 / 36 / 165 | 220 / 43 / 315 | 320 / 46 / 465 |
| Natural Resistance 153 | the four resists % | 12 | 40 | 54 | 67 | 72 |

All equal d2d (V run). Stun (139), Double Throw (140) and War Cry's stun length have no calc cell: Stun and Double Throw are table rows of the hit build (Q2), War Cry's stun length is Param1 + Param2 (L - 1) of the row (25 + 5 per level, D), whose application was not run (dead end 5). The 6 passive weapon masteries read only `lvl`, so the stat-getter limit of nec-skill-trace.md section 8 does not bite (V: no Barbarian cell reads another stat).

**Q2. How is a Bash-family hit built, and which rows share it? (S upstream skills.md; V read of the d2d side)** srvstfunc 32 / srvdofunc 2 (Bash, Stun, Concentrate; Berserk is start 39): FUN_005d7ea0 builds the hit on each swing: to-hit bonus from the skill, ED = calc1, **calc2 added after the damage for Bash** (the table's flat 1 .. 30 above), the skill's element with calc4 as the conversion percent (Concentrate: Berserk level; Berserk: 100), Stun's EType 5 = a stun of the elemental length; d2d `Fight::skill_swing` (fight.cpp:820..839) has exactly these arms: `swing.ed_pct = calc1`, `swing.flat = calc2` (only start 32 / do 70), `conv_pct = calc4` for another element, `stun_ticks` for EType 5. No run of the game build function (U for equality of the result; the cells feeding it are V run equal).

**Q3. Whirlwind (start 38 FUN_005d8f50, do 76 FUN_005d9580): which units, how often? (V read; V run for the search)**
- Per event frame FUN_005d9580 calls FUN_005d9320 (the hit timer, below) for the number of hits (0, 1, or 2 with a weapon in each hand, FUN_005d92d0: both slots hold items passing the same type test), then per hit **FUN_0056bd10(x 0, y 0, radius 5, flags 3, last target id)** (0x5d968a..0x5d969f): x = y = 0 means the whirling unit's own subtile (FUN_0056b7e0 0x56b824..0x56b83c), the radius is squared (0x56b85a) and compared with FUN_006492a0, the squared distance between integer subtile words (unit path +6 / +4, FUN_00648900 reads `word [path + 6]`), callback FUN_0056bcd0 picks the unit with the next higher id than the last, else the lowest. **V run: 108 cases, the units handed out are exactly the integer disc `dx^2 + dy^2 <= 25` (80 of the 360 neighbours of the 19 x 19 block), the ring order ascending, fractions irrelevant.** Then to-hit, ResultFlags, ED = calc1, element (S upstream, FUN_0057ec10 / FUN_0057dbf0 / FUN_0057d4f0 as in the other do functions).
- Timer FUN_005d9320 (V read, player branch): if the skill entry's timer (entry +0x24, FUN_00644530 / FUN_00644620) is 0: set it to `frame + 4` and return one hit; if `frame < timer`: no hit; else the gap `g` by the attack frames of FUN_0062a710 (< 12: 4, < 15: 6, < 18: 8, < 20: 10, < 23: 12, < 26: 14, else 16; no weapon: 10), `timer = timer + g`, **or `frame + g` if that is already past**; returns 2 with two weapons (FUN_005d92d0). The gap table equals d2d's `whirlwind_gap` (skills.hpp:821, V read both). A branch on `[game + 0x70] == 0` returns one hit with no timer: what that field is (the expansion flag?) is not read (U).

**Q4. Frenzy (do 9, FUN_005d8e00 / FUN_005d8c70): the stacking? (V read both sides)** On a landed hit: a state is made if absent (counter stat 0xa9 set to 0), its length is `auralen` (calc, `par7` = 150 frames) from now on **every** hit, the counter becomes `min(counter + 1, skill level)`, and the aura stats (velocitypercent dm34, attackrate dm56) are applied with the counter as the level (FUN_005c6cc0, FUN_005d8c70 0x5d8db3..0x5d8ded). d2d `Fight::frenzy` (fight.cpp:4362..4370): `min(found ? level + 1 : 1, lvl)`, until = now + auralen: **same** (V read both). The odd frame retargets to the next id in reach (FUN_0056bd10, S upstream); d2d `other_target()` (fight.cpp:959): not compared (U).

**Q5. Howl (do 22 FUN_005c9b50, hit 17 FUN_005aafb0, run FUN_005ddd00): who flees and for how long? (V run readers, V read)** Skills row 130: Param1..Param6 = 2, 1, 24, 5, 75, 25 (V run, record +0x148..+0x15c). The ring is row `howl` (Vel 12, MaxVel 12, Accel -1000, Range 12, no NextHit, hit func 17). FUN_005aafb0 (0x5ab021..0x5ab08d): the target is a monster without the terror state, **and `Param2 + caster level + skill level > monster level`** (FUN_004efc20 = record +0x14c = Param2 = 1, FUN_00625480(.., 12) for both levels); then FUN_004efcb0 (**Param5 + (L - 1) x Param6**) and FUN_004cc7c0 (Param3 + (L - 1) x Param4, the "distance") are pushed and FUN_005ddd00 runs: gates (not special, the monster class passes FUN_00623470 / FUN_005b0da0, FUN_005a0180 flag 0xa), the Terror state 0x38 of skill 77 level 1, the **duration = the first pushed value (FUN_004efcb0's: struct +0x10, 0x5dddbf..0x5dddd2)**, cut by the monster's stat 0x6d percent (`MulDiv(len, stat, 100)` subtracted, refused at 100 and over: FUN_0056e970 0x56ea17..0x56ea3d), added to the game frame (0x56eaa9..0x56eab1). The "distance" value is not read by FUN_005ddd00. V run: FUN_004efc20 / FUN_004cc7c0 / FUN_004efcb0 return Param2 / Param3 + (L - 1) Param4 / Param5 + (L - 1) Param6 at all 40 levels (L1: 1, 24, 75; L10: 1, 69, 300; L20: 1, 119, 550).

**Q6. War Cry, Battle Cry, Shout, Battle Orders, Battle Command (do 68 FUN_005d83e0)? (V read, V run for the rings, S for the effects)** Do 68 sets flag 0x40 on the caster, takes the row of srvmissilea (FUN_005d3cf0), calls **FUN_0056d400(unit, row, skill, level, velocity 0)**: 64 missiles, flags 3, no velocity override (0x5d843d), then puts the skill's state on the caster (FUN_005d8290: length = auralen, aura stats FUN_005c6cc0). Rows (D): `warcry` 235 Vel 12, Accel -600, Range 16, **NextHit 1 / NextDelay 6**, SrvDmgFunc 7, ResultFlags 4; `battlecry` 219 Vel 12, Accel -1000, Range 12, NextHit 1 / 4, hit func 21 (the curse state on the monster, S upstream); `shout` 149 and `battleorders` 237 / `battlecommand` 236: Vel 30, Accel -500, Range 15, hit func 18 (the state on an ally the nova reaches; the last two with NextHit 1 / 4). **V run flights: War Cry reaches 5.4-6.4 subtiles, Battle Cry 3.6-4.5, Shout / Battle Orders / Battle Command 18.4-19.3**, all independent of the level. d2d builds the ring as the nova (fight.cpp:2633..2662): 64 missiles at even angles, range = the row's. Its ally reach for Shout and the like is `speed x Range x 0.04` (fight.cpp:2648), which ignores Accel: its magnitude against 18.4-19.3 subtiles was not resolved (U; the units of `missile_speed` were not traced).

**Q7. Leap, Leap Attack? (S upstream skills.md "Leap Attack", "Leap"; V run cells)** Leap (start 40 FUN_005d9d50, do 77): the distance range (aurarange calc 7 .. 27 subtiles) and the knockback radius calc1 (4 .. 33) are V run equal; the flight and landing rules are upstream's (S) and d2d's `Fight::leap` / `move_event` (fight.cpp:2382..2396): not compared (U). Leap Attack (start 41, do 78): ED = ln34 + Leap x par8 (V run), knockback flag 8 on the strike (d2d: `knockback = ... kLeapAttack`, fight.cpp:831). Not run.

**Q8. Berserk, Concentrate, Frenzy as magic damage? (D, V run cells)** Berserk, Concentrate and Frenzy have EType `mag`; calc4 is the conversion percent: **Berserk 100, Concentrate and Frenzy = Berserk's level** (V run). So Frenzy and Concentrate do part of their physical as magic at one percent a Berserk point; Berserk alone is fully magic (a physical immune takes it, the bot's class table agrees except for the fire entry, B2). d2d: `conv_type = etype`, `conv_pct = calc4` (fight.cpp:837..838). Berserk's state: length calc2 (68 frames at L1 falling to 30), aurastat armor_override_percent -100 (defence 0 while it lasts); d2d `self_state` (fight.cpp:1316..1319) uses calc2 as the ticks (V read).

**Q9. The other rows? (D only)** Taunt 137 (do 71), Find Potion 131 / Find Item 142 (do 69 / 72), Grim Ward 150 (do 75, start rows `grimward*start`, calc1 1000 frames), Double Throw 140 (do 74), Stun 139: the calc cells above are V run equal; the do functions are upstream-documented (S) and d2d has arms for them (fight.cpp:1199..1290, 2713); not compared (U).

## 3. d2d side (read, main checkout)

Barbarian skills in d2d are driven by Skills.txt's srvstfunc / srvdofunc: `Fight::skill_swing` (fight.cpp:775..841) for the Bash family, Whirlwind, Frenzy, Double Swing, Leap Attack; `move_event` (2344..2397) for the moving skills; `self_state` / `start_state` (1313..1335) for Berserk, Concentrate, the war cries' caster state; `Fight::fire` (2622..2725) for the rings; the missile hit arms (2911..2938) for War Cry's allies, Battle Cry (21), Howl (17). Whirlwind: `start_move` (725..739: `whirl_next = now`, run at the walk velocity), `move_event` (2345..2362: `now < whirl_next` returns; else `whirl_next = now + whirlwind_gap(ticks)`; one hand or two; nearby = every targetable monster within **5.f** of the player in cells; `upper_bound` on the current target then wrap; `hit(now, 5.f)`).

## 4. Deviations d2d vs game.exe

| id | rule in game.exe | rule in d2d | tag | differential |
|---|---|---|---|---|
| D1 | whirl hit target: next id within 5 **subtiles** (integer disc, FUN_0056bd10 r 5) | next id within **5 cells** (25 subtiles), `hit` reach 5 cells (fight.cpp:2356, 2360; `kMeleeReach` is 1.1 cell) | V run (game, 108 cases) + V read (d2d, one constant read in cells; unit.x is in cells: ai.hpp:475, strafe's `* 5`, fight.cpp:2672) | game side exists (`skbarb_ww.py`). Real d2d side buildable: a `Fight` harness (pets_merc_harness.cpp shows one) with a Barbarian, monsters at 1..6 cells, `move_event` called through the whirlwind sequence; oracle: which monsters lose life per event and how many events |
| D2 | the timer lives on the skill entry: first ever hit sets `frame + 4`; later `timer + g` (anchored) unless already past, then `frame + g`; two hits with two weapons | `whirl_next = now` at each cast, `now + g` after each hit (drifts by the event's lateness) | V read both sides; effect size I (depends on the event spacing of sequence 10, not tabulated) | buildable: FUN_005d9320 natively with a player unit (FUN_00535bc0 weapon, FUN_0062a710 frames; stat getter hooked as in skill_damage.py) against `move_event`'s schedule at the sequence's event frames; oracle: hit frames over a 4 s whirl for IAS 0..150 |
| D3 | Howl terror duration **Param5 + (L - 1) Param6** frames = 75 + 25 (L - 1) (550 at L20), less the monster's stat 0x6d share | `max(Param3 + (L - 1) Param4, 25)` frames = 24 + 5 (L - 1) (119 at L20) (fight.cpp:2932) | V run (the three readers, 120 values) + V read (which value reaches the state) ; the pushed-argument order is V read | buildable: the real d2d hit arm needs a Fight harness (same as D1); the value alone is a one-line comparison |
| D4 | Howl lands iff `Param2 (1) + clvl + L > monster level` | `monster level < clvl + calc1 (2 (L - 1)) + L` (fight.cpp:2931; its comment marks the sum a guess: "ponytail") | V run (FUN_004efc20 = 1 at 40 levels) + V read | as D3; oracle: the highest monster level frightened, clvl 1..99, L 1..40 |
| D5 | Howl ring velocity override `Vel + calc1`: MaxVel clamps at the fifth frame (flights: L1 3.61-4.47, L20 15.00-15.65 subtiles) | calc1 added to the range (fight.cpp:2638); velocity from the row at every level, which Accel -1000 takes to 0 inside 14 frames: reach about the L1 value | V run (game, 256 flights) ; d2d side I (arithmetic from `MissileFlight::step`; sknova D3 first named it) | buildable: the real `rules::MissileFlight` in a test with the row's Vel, MaxVel, Accel, 40 levels; oracle: the game flights' end distance |
| D6 | War Cry (NextDelay 6) and Battle Cry (4) are NextHit rings: `justhit` on the unit blocks the other ring missiles for the window; model on the V-run flights: **0.54 hits per monster per cast** (all monsters within 16 subtiles; War Cry; Battle Cry 0.47) | per-missile `hit_at`: every touching ring missile hits; **4.26** (8.0x; Battle Cry 4.26, 9.1x); at 1 subtile 1.0 against 8.3 / 23.9 / 33.9 for monster size 1 / 2 / 3, at 4 subtiles 1.0 against 3.0 / 8.1 / 11.8 | I (model rule on V-run flights, as sk-nova-trace D1, the same code path); flights V run | as sk-nova-trace D1: `test_verify_nexthit.cpp` / `verify_nexthit.py` with the war cry rows (235, window 6; 219, window 4); oracle FUN_005adcc0's justhit chain |
| D7 | the terror duration is cut by the monster's stat 0x6d and a monster at 100 or more is refused (FUN_0056e970 0x56ea17..0x56ea3d, V read) | no such cut found in the Howl arm (fight.cpp:2929..2937; U) | V read game, U d2d | as D3 with a monster carrying stat 109 |

Not deviations (checked): the Whirlwind gap table (FUN_005d9320 against `whirlwind_gap`, V read both); Frenzy's counter and duration (Q4); the War Cry flights' reach (5.39-6.40 on game.exe; d2d's flight rules `MissileFlight` are the port of the same code, S upstream missiles.md, so equal modulo the nova's D2 direction rows of sk-nova-trace); the calc cells (Q1).

## 5. Bot findings (described, not changed)

| id | finding | where |
|---|---|---|
| B1 | The War Cry build's main attack from clvl 30 to 90 is War Cry (`main_attack[1]`), but the combat catalogue types War Cry `buff`, range 0; every damage picker skips `KindBuff` (rules.go:202, castfirst.go:151, resource.go:174), the barbarian profile's attack bands list Frenzy, Bash, Double Swing, Attack only and its buff groups Battle Orders, Shout, Iron Skin, and the plan's `main_attack` is applied only for the Amazon (gearcheck.go:96). I (grep, no run): the build as shipped never casts War Cry as an attack. Its ring reaches only 5.4-6.4 subtiles (Q6): a cast must happen with the pack at one or two cells. | internal/combat/classdata/skills.json:1; internal/combat/classdata/profiles.json:266..285; internal/gear/builds/barbarian_warcry_endgame90.json `main_attack` |
| B2 | `classOptions` lists Berserk under Fire as well as Magic; Berserk, Concentrate and Frenzy are `mag` (D), so the Fire entry is wrong, and Frenzy / Concentrate convert only Berserk-level percent to magic (Q8), so they are not a physical-immune workaround unless Berserk has points. | internal/combat/rules.go:241 |
| B3 | Whirlwind is in the class option table and in `skills.json` (range 10, hold) but in no profile band; its real reach per hit is 5 subtiles (one cell, Q3), not 10, and a soak calibrated on d2d would over-credit its area damage (D1). | internal/combat/classdata/skills.json:1; rules.go:241 |
| B4 | Howl is on the plan's rows (clvl 9, one point; the rest of the build never raises it) but not in the profile's buff groups: it is never cast. At L1 its ring reaches 3.6-4.5 subtiles and the fear lasts 75 frames (D3, D5) on game.exe; a d2d soak gives 24 (floored to 25) frames at L1 and 69 at L10 where game.exe gives 300, and its reach stays about 4.7 subtiles where game.exe's velocity rule gives about 11.6 at L10 (arithmetic, I) and 15 at L20 (V run). | internal/gear/builds/barbarian_warcry_endgame90.json rows; profiles.json |
| B5 | `dualWield` marks Frenzy "only with a weapon in each hand" as the bot's rule (comment: "U: skills.txt itypea says only mele"); FUN_005d8c70 (read here) does not call FUN_005d92d0, the two-weapon test Whirlwind uses; FUN_005d8e00 (the do, only described by upstream's doc) was not read, so whether a single weapon works is U; if it does, the rule only costs Frenzy uses. | internal/combat/gearcheck.go:70 |

## 6. PR candidates

**No PR candidate.** D1 to D5 and D7 are proven only on the game side (V run) with the d2d side read; D6 is the known nova deviation (sk-nova-trace D1, PR #158 covered the nova and Frost Nova only) extended by a model. The existing differential harnesses that run the real d2d build (`nec_test`-style calc tests, `test_verify_nexthit.cpp`) do not reach the whirl, Howl or war-cry hit arms. The two cheapest next steps that could become candidates: (a) a `Fight` harness for D1 / D3 / D4 (the changes are single constants: the whirl's `5.f` cells to 5 subtiles = 1 cell, the Howl parameter indices), (b) the war-cry rows through the existing NextHit verifier for D6.

## 7. Matrix cell updates (proposed text; no Barbarian matrix doc exists, so a table is proposed here)

| Row | Cell | New value |
|---|---|---|
| all 30 Barbarian skills | calc cells and the skill's own physical min / max | V run equal to d2d, 27,800 cases (`skbarb_run.sh`) |
| Whirlwind | hit radius | D1: 5 subtiles on game.exe (V run), 5 cells on d2d (V read) |
| Whirlwind | hit timer | D2: V read both, anchored vs drifting |
| Frenzy | counter, duration, aura stats | same (V read both) |
| Howl | terror length, level gate, ring reach | D3 / D4 / D5 |
| War Cry, Battle Cry | hits per cast | D6 (I, model on V-run flights); flights V run |
| Shout, Battle Orders, Battle Command | ally reach | game 18.4-19.3 subtiles V run; d2d U |
| Bash, Concentrate, Berserk, Double Swing, Leap Attack | hit build | V read (S upstream); cells V run |
| Leap, Taunt, Find Potion / Item, Grim Ward, Stun, Double Throw | do function | cells V run; effect U |

Ledger line added: `skills.bar 126-155 TESTED-0-DIFF 27,800 cases (calc and physical aspect)`.

## 8. Dead ends and limits

1. **No run of the d2d hit arms.** The whirl, Howl and war-cry arms live inside `Fight` (fight.cpp) and need a harness with a world, a Barbarian and monsters; none exists in tools/emu for the Barbarian. All six deviations therefore stop at "game side V run, d2d side V read / I".
2. **FUN_005d9320's first branch** tests `[game + 0x70]`; its meaning was not found. The timer branch above is the one a LoD game takes under the usual reading (I).
3. **Hit build FUN_005d7ea0 (Bash, Stun, Concentrate, Berserk, Double Swing)** and the hit builders of Whirlwind / Leap Attack were not re-run: the calc cells that feed them are V run equal, the plain melee blow is SKMELEE2's (docs/research/sk-melee2-trace.md).
4. **War Cry's damage on the monster and its stun** (SrvDmgFunc 7, ResultFlags 4, Param1 / Param2 = stun frames 25 + 5 per level): the min / max is V run equal (Q1); how the stun is applied and whether d2d applies it was not read (U).
5. **Battle Orders / Shout / Battle Command effects** (FUN_005c6cc0 aura stats on the caster, FUN_005ab0b0 on allies): the cells are equal; the application and d2d's `bo_life` / `merc_cry` were not compared (the max-life percent op is playerdef-trace.md Q5).
6. **d2d's ally reach for the cries** (`speed x Range x 0.04`): the unit of `missile_speed` was not resolved, so no number is claimed (Q6).
7. **Leap's and Leap Attack's flight** only from upstream's doc.
8. The ring-hit model (D6) is a rule model on real flights, not the game's hit code; the 4,000-monster draw (3,000 per run) is random placement, not combat.

## 9. Drill recipe (lab, fork verbs; for later, not run here)

Fork verbs (docs/forkdbg.md): `debug skill`, `debug points`, `debug spawn`, `debug mon set <id> life .. maxlife .. res <type> ..`, `debug cast <skill> unit <id>|<x> <y>`, `debug missiles`, `debug states`, `step`, `dump stats`. A throwaway Barbarian on one slot, `--no-save`, lockstep (docs/forkcheats.md, docs/d2d-fork.md). Expected values are game.exe's (the reads above); d2d's in brackets.

| Drill | Confirms | How |
|---|---|---|
| BW1 Whirlwind radius (D1) | D1 | `debug points 151 1`, a monster with large life at 2, 3, 4, 6 cells beside the path, cast whirlwind past it; life lost per event: game.exe none beyond about 1 cell [d2d hits out to 5 cells] |
| BW2 Whirlwind cadence (D2) | D2 | one monster in reach, `step` frames, list the frames its life drops over 4 s at IAS 0 and 75: gap by the table, anchored |
| BH1 Howl length and gate (D3, D4) | D3, D4 | `debug points 130 1/10/20`, spawn monsters of level clvl + 1 .. clvl + 40 close by, `debug states`: terror (state 0x38) present iff level < 1 + clvl + L; its remaining frames 75 + 25 (L - 1) [24 + 5 (L - 1)] |
| BH2 Howl reach (D5) | D5 | monsters at 3, 5, 8, 12, 15 subtiles, Howl at L1 and L20, which get terror: L1 to about 4, L20 to about 15 [about 4.7 at both] |
| BC1 War Cry hits (D6) | D6 | one monster, large life, at 1 and 4 subtiles, War Cry L1 and L20, life lost per cast over 5 casts: about 1 hit of the physical min-max each (20-30 at L1) [3 to 34 times that] |
| BC2 War Cry reach | Q6 | monsters at 4, 5, 6, 7 subtiles: hit up to 6, not at 7 |
| BC3 ally reach | Q6 / U | a hireling at 10, 15, 18, 20 subtiles, Battle Orders: which gets the state |
| BF1 Frenzy stack | Q4 | `debug states` after 1, 3, 10 hits at Frenzy 10: counter, run speed %, attack rate % per stack |

## Shared-file lines for the merger

See `openspec/changes/skbarb/merge-lines.txt`.
