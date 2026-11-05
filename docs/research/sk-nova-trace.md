# Frost Nova's ring: game.exe 1.14d vs d2d vs the bot (decomp row 18)

> status: card sknova, 2026-11-01, offline (no game, no d2d slot, no Go change). Decomp row 18 (SKNOVA, sorc-skill-matrix rank 5): how game.exe builds the Frost Nova ring (do 22), what d2d does, where they differ, and how the bot uses Frost Nova; Static Field's radius unit alongside. Follows docs/research/sk-missile-trace.md (SKMISS: spawn, flight, roll; its D3 is resolved here), docs/research/sk-mana-trace.md (SKMANA) and the SKLIGHT trace (docs/research/sk-lightning-trace.md on branch agent/sklight, not merged yet: the NextHit rule, its D1 handed over for the novas). The chill and freeze rules are NOT redone: docs/research/freeze-rules-trace.md, static-field-chill-trace.md, PRs #155 / #156.

Tags: **V read** disassembled by this agent with upstream's `tools/emu/disasm.py` on the 1.14d Game.exe (addresses given); **V run** run natively under the emulator by this agent (`tools/emu/sk_nova_ring.py`, on the worker host, upstream main 72132b99's tools/emu, seeds 1 and 7); **D** patched 1.14d tables (Skills.txt, Missiles.txt read with `mpq-cat` from the patch stack); **S** d2d source (upstream main 72132b99, file:line), an upstream doc, or an earlier trace of ours (named); **I** inference, including the hit MODEL in `sk_nova_ring.py` part C; **U** not verified.

## Summary

- **The ring (V read + V run).** Frost Nova's do 22 (FUN_005c9b50) calls FUN_0056d400, which makes **64 missiles** of srvmissilea `frostnova` (119) from the caster's subtile, missile k aimed at the caster + (table 0x6e1288[k], table 0x6e1388[k]): 30 x (cos, sin) of k x 5.625 degrees **truncated to integers** ((30, 0), (29, 2), (29, 5), (28, 8), ...). The aim is FUN_0064fc60's 128-row table on those integer offsets, so the 64 directions are not evenly spaced (k = 1 is 3.6 degrees, not 5.6). Run natively: FUN_0056d400 makes exactly 64 calls to FUN_0059fa30 with flags 7, the tables as offsets, the caster's subtile as start.
- **Range and reach (V run + D).** 14 frames (Range 14, no LevRange), velocity 4608 (Vel 24 << 8 x 75 / 100), Accel -1000 every fifth frame: every one of 384 ring flights (seed 1) and 512 (seed 7) ran exactly 14 frames and ended 12.00 to 12.73 subtiles from the caster in open ground. Units are tested on frames 1 to 13 only (the frame the range ends tests none, S SKMISS / SKLIGHT), so in the model a size-1 monster is hit up to 11 subtiles away, a size-2 or size-3 one up to 12. **The hit radius is about 11 to 12 subtiles, not the bot's 4.**
- **calc1 goes to the velocity, not the range (V read).** FUN_005c9b50 passes Vel + VelLev x lvl / 8 (FUN_00663270) + calc1 as a velocity override (params +0x28, flag 4); FUN_0059fa30 uses it as the velocity. d2d adds calc1 to the range instead. Frost Nova and Nova have no calc1 (D), so nothing changes for them; only Howl differs (D3).
- **D1, the big one (V read both sides, magnitude MODEL I): d2d hits a monster with many ring missiles per cast; game.exe hits it once.** In game.exe the first ring missile to hit puts unit state 0x56 `justhit` on the monster for NextDelay 4 frames, and every NextHit missile (all 64 ring missiles are NextHit rows) passes it by while the state lasts (FUN_005adb00, FUN_005a8730: SKLIGHT D1, V read). In d2d, NextHit rows keep a per-missile `hit_at` and skip the ring's shared `struck` list (missiles.cpp:587-596), so each of the 64 missiles that touches the monster strikes it with a full roll. On the real game.exe flights: **6.2 hits per monster per cast in d2d vs 1.0 in game.exe** (seed 1: 6.16 vs 0.99; seed 7: 6.25 vs 0.99), point blank 8 / 24 / 34 hits for monster sizes 1 / 2 / 3, at 4 subtiles 3 / 7 / 9, at 8 subtiles 1.3 / 3.4 / 4.8. d2d overrates Frost Nova (and Nova, Poison Nova) damage several times over, most at close range, where the bot casts it.
- **D2 (V run game side vs a line copy of d2d's aim, S): d2d aims the ring at 64 even float angles.** 40 of the 64 directions get another row of the aim table than game.exe's integer offsets; endpoints move up to 0.38 subtile. Small; it shifts which subtiles the ring enters.
- **Damage, chill, mana (D + S, cross-references).** One hit at level 1: cold 2 to 4 points (EMin 4, EMax 8, HitShift 7; top exclusive, SKMISS D1), chill length 200 frames (8 s) at level 1 plus 25 per level, resisted, chill only (EType cold, no SrvHitFunc 13: Frost Nova never freezes); chill rules: freeze-rules-trace.md and static-field-chill-trace.md. Mana 9 + (lvl - 1) points, paid at the action frame (SKMANA, SKMISS D2). Cast SC, 13 ticks at 0 FCR (K1).
- **Bot (described, not changed).** B1: the crowd rule counts hostiles within 4 subtiles while the ring hits everything within about 11; B2: skills.json says 5, unused for it; B3: any Frost Nova number measured on d2d is inflated by D1; B4: mana gate and cast point fine; B5: novaGap 2.5 s vs an 8 s chill; B6: Static Field's range is 2/3 of the true radius (radius and `Target.Dist` both in subtiles).
- **PR candidate: none.** D1 is read on both sides and quantified by a model on V-run flights; the existing differential (`tools/emu/sk_missile_hit.py` + `sk_missile_hit_test.cpp`) covers the damage roll only, so it cannot show mismatches before > 0 and after 0 for a hit-count rule. How to build one is under D1.

## 1. game.exe: the nova do and the ring

| Q | Address | Rule | Tag |
|---|---|---|---|
| Q1 the do | `FUN_005c9b50` (do 22; table 0x7322b0 entry 22 = 0x5c9b50, read from the image) | unit +0xc4 \|= 0x40; skill record (stride 0x23c); row = FUN_005d3cf0(unit, skill) (srvmissilea); **vel = FUN_00663270(row, lvl) + FUN_00646ca0(unit, calc1 at Skills +0x138, skill, lvl)**; FUN_0056d400(game, unit; unit, row, skill, lvl, vel); returns 1 (so the do pays the mana, SKMANA) | V read (0x5c9b50..0x5c9bfe) |
| Q1 empty calc | `FUN_00646ca0` | the calc offset is compared unsigned against the calc table length: a blank calc (-1) returns 0 | V read (0x646ca0..0x646cbf) |
| Q1 FUN_00663270 | Missiles record (stride 0x1a4) | byte +0x9a (Vel) + byte +0x9b (VelLev) x lvl / 8 (signed division by 8) | V read (0x663270..0x6632b6) |
| Q2 the ring | `FUN_0056d400` (ecx game, edx unit; stack unit, row, skill, lvl, vel; ret 0x14) | params block 0x5c bytes zeroed; +0 flags 3; +4 owner; +0x10 row; +0x14 / +0x18 start = the unit's subtile (players: FUN_006488c0 / FUN_00648900 on the path; types 2, 4, 5: the static path's +0xc / +0x10); vel != 0: flags \|= 4 and +0x28 = vel; +0x2c skill; +0x30 lvl. **Loop i = 0..63**: +0x1c = dword 0x6e1288[i], +0x20 = dword 0x6e1388[i], FUN_0059fa30(game, params) | V read (0x56d400..0x56d4dd) |
| Q2 run | the same, natively | FUN_0056d400 for Frost Nova level 1 made **64** FUN_0059fa30 calls (hooked to record): flags 7 (3 \| 4), vel 24, offsets equal to the two tables, start equal to the caster's subtile, skill 44, row 119 | V run (part A, seeds 1 and 7) |
| Q2 tables | 0x6e1288 (x), 0x6e1388 (y), 64 signed dwords each | x = 30, 29, 29, 28, 27, 26, 24, 23, 21, 19, 16, 14, 11, 8, 5, 2, 0, -2, ... (k = 0..63); y the same a quarter turn on: **truncated 30 x (cos, sin) of k x 2 pi / 64** (checked: 30 cos 5.625 = 29.85 -> 29, 30 sin 5.625 = 2.94 -> 2) | V read (bytes of the image) |
| Q3 params in the maker | `FUN_0059fa30` | flags bit 1: start = +0x14 / +0x18; **bit 2: target = start + (+0x1c, +0x20)** (bit 0x20: target absolute); **bit 4: velocity = +0x28 << 8** (bit 0x10: +0x28 as is) instead of (Vel + VelLev x lvl / 8) << 8; state 0x57 scaling when the row's flag is set; then x 75 / 100. Range = Range (+0x96) + LevRange (+0x98) x lvl (no 0x8000 / 8 flag here). Target set by FUN_00648ad0 (a point; aim by FUN_0064fc60) | V read (0x59fa79..0x59fbab, 0x59fc93..0x59fcaa, 0x59fd43..0x59fd4c) |
| Q3 aim | `FUN_0064fc60` | the unit vector from the 128-row table at 0x6eb7e0 by minor x 127 / major (integer); on the ring's offsets row = trunc(small x 127 / big): k = 1 (29, 2) row 8 -> (4087, 257), 3.6 degrees | S (upstream rules `missile_aim`, proven by `missiles.py`); V run (the flights below) |
| Q4 flight | srvdofunc 1 (`FUN_005ae1f0`) each frame | velocity 4608 frames 1-4, 3608 frames 5-9, 2608 frames 10-14 (Accel -1000 every fifth frame, MaxVel 24 << 8), Range 14 | D + S (upstream flight rules); V run |
| Q4 run | 64 x 3 spots x levels 1 and 20 (seed 1: 384 flights; seed 7: 4 spots, 512) | made by the real FUN_0059fa30 from params built as FUN_0056d400 builds them, stepped by the real srvdofunc in a Blood Moor game, no foe: **every flight 14 frames, end 12.00 to 12.73 subtiles from the caster**; missiles.py's port fed the table offset matches every frame of every flight (0 of 384 / 0 of 512 differ) | V run (part B) |
| Q5 who it hits | FUN_005ae1f0 / FUN_005adf10 / FUN_005a8730 / FUN_005adb00 | one unit a missile a frame, the missile's last unit skipped (LastCollide 1), NextHit 1: a unit in state 0x56 `justhit` is passed by; a hit sets `justhit` for NextDelay 4 frames; CollideKill 0 (the missile flies on); no ToHit (never misses) | S (SKLIGHT Q3 / Q4, V read there; rows D) |
| Q5 the last frame | FUN_005ae1f0 | range - 1 < 1: the flight ends with no unit test that frame: units are tested on frames 1-13 | S (SKMISS section 2, SKLIGHT Q3) |

Rows (D, patched tables): Frost Nova (44) srvdofunc 22, srvmissilea `frostnova`, **calc1 blank**, EType cold, EMin 4, EMax 8, EMinLev 4 / 6 / 8 / 10 / 12, EMaxLev 5 / 7 / 9 / 11 / 13, HitShift 7, ELen 200, ELevLen 25 / 25 / 25, EDmgSymPerCalc (Blizzard + Frozen Orb blvl) x 10, mana 9, lvlmana 1, manashift 8, minmana 1, reqlevel 6, anim SC, seqtrans SC. `frostnova` (119): Vel 24, MaxVel 24, Accel -1000, Range 14, no VelLev / LevRange, Size 1, CollideType 3, CollideKill blank (0), LastCollide 1, NextHit 1, NextDelay 4, no SrvHitFunc, Skill Frost Nova, HitClass 48. `nova` (90) the same shape with Range 13 (lightning, EMin 1 EMax 20, HitShift 8).

Other do-22 skills (D): Nova (48), Poison Nova (92), Howl (130, **calc1 = par1 x (lvl - 1) = 2 (lvl - 1)**, `howl` Vel 12 Range 12, not NextHit), and monster novas PrimeIceNova (190), DiabFire (195), Baal Nova (316), MephFrostNova (346), Trap Nova (353), all calc1 blank and NextHit 1 / 4.

### Reach and hit radius

| frame | velocity | subtiles / frame | from the caster (I, straight along an axis) |
|---|---|---|---|
| 1-4 | 4608 | 1.125 | 4.5 |
| 5-9 | 3608 | 0.881 | 8.9 |
| 10-13 | 2608 | 0.637 | 11.4 (last frame with a unit test) |
| 14 | 2608 | 0.637 | 12.0 (no unit test) |

Run (V): ends 12.00 (axis) to 12.73 (diagonal: the aim table's diagonal row is longer per step) subtiles. Model (I, part C): monsters of size 1 are hit up to about 11 subtiles from the caster, size 2 and 3 up to 12; at 12 a size-1 monster is never hit, at 13 almost nothing is. Walls end single missiles (S: the flight rules); in rooms the ring is cut where walls stand.

## 2. Damage, chill, mana, cast (cross-references; not re-traced)

| Q | Rule | Tag |
|---|---|---|
| Q6 damage | built at missile creation (FUN_0059fa30 -> FUN_0059f900 -> FUN_0064b860) from Missiles.txt `Skill` = Frost Nova: min (EMin + brackets) << 7, max (EMax + brackets) << 7 in 256ths, synergy (Blizzard + Frozen Orb) x 10 %: **level 1: 2 to 4 points; levels 2-8 add 2 to the min and 2.5 to the max per level** (level 4: 8 to 11.5, level 8: 16 to 21.5) | S (K3, PR #154); D values; arithmetic I |
| Q6 roll | FUN_005a8910: min + rand(max - min), the max never rolled | V (SKMISS D1, real builds; candidate fix/missile-roll-top) |
| Q6 resistance | resistance less pierce, MulDiv, >= 100: 0 (a cold immune takes no damage and no chill) | S (PR #135) |
| Q7 chill | cold length ELen 200 frames + 25 per level (brackets 2-8, 9-16, 17+): 8 s at level 1; resisted like the damage; FUN_0057b230: chill (FUN_0057af80) for players, bosses, champions, uniques, mercs; a plain monster is FROZEN only by a freeze length (EType frze, hit 13), which Frost Nova does not have: **Frost Nova only chills** | S (docs/research/freeze-rules-trace.md, static-field-chill-trace.md; PRs #155 / #156 and the local chill / freeze candidates) |
| Q8 mana | 9 + (lvl - 1) points (manashift 8), checked at the start, paid in the do at the action frame | S (SKMANA, SKMISS D2) |
| Q8 cast | SC with seqtrans SC (no sequence): 13 ticks at 0 FCR | S (K1, cast-trace-gameexe.md) |

## 3. d2d (upstream main 72132b99)

- `Fight::fire` (components/game/fight/missiles.cpp:278-285): do 22 (and War Cry 68): 64 missiles, `angle = float(k) * 2 * 3.14159265f / 64`, `send(cos(angle), sin(angle), range)`, range `missile_info.range + calc1` (War Cry: the row's Range), `shot.struck = shared` (one list for the ring), `shot.to.reset()` ("a way, not a point"). The header says so itself: "ponytail: the nova's 64 directions are even angles, not the tables at 0x6e1288 / 0x6e1388" (components/game/fight.hpp:890-891).
- Flight: `missile_fly` (components/game/ai.hpp:579-604): with no `to`, the aim is `rules::missile_aim_along(velocity)` (float ratio, components/rules/missiles.hpp:70) and the missile counts as sent outside its room (`follow`). Velocity from `missile_speed` = `rules::missile_velocity(Vel, VelLev, lvl)`: equal to game.exe's for calc1 0.
- Hits: the per-frame loop (components/game/fight/tick.cpp:394-411) tests every targetable monster against the subtiles the missile entered this frame (`missile_struck`, ai.hpp:625), skips monsters in the shared `struck`, calls `skill_missile_hits` (missiles.cpp:540) and moves on to the NEXT monster when it returns false (CollideKill 0). In `skill_missile_hits`, NextHit rows check `missile.hit_at` (a per-missile list, ai.hpp:544; missiles.cpp:587-591), strike, and **push to the shared `struck` only for non-NextHit rows** (:596). So for `frostnova` / `nova` / `poisonnova` the shared list stays empty and every ring missile that touches a monster strikes it once (and again after 4 frames if still touching). The comment on `struck` (ai.hpp:531, "a nova's missiles share theirs (one hit a monster)") states the intent the NextHit branch undoes.
- No unit state `justhit` anywhere in components/ (SKLIGHT section 4, grep).
- Monsters' novas on the player (ai.hpp:789-792) key the ring's shared list per foe ("a ring's missiles strike each foe once"), with no NextHit window (not traced on the game.exe side here, D6).

## 4. Deviations (d2d vs game.exe)

| id | what | game.exe | d2d | tag | magnitude | differential |
|---|---|---|---|---|---|---|
| D1 | hits per monster per nova cast (NextHit rows: Frost Nova, Nova, Poison Nova) | the first ring missile to hit sets unit state 0x56 `justhit` for 4 frames; the other ring missiles pass the unit by while it lasts; each missile also skips its last unit and hits one unit a frame | each of the 64 missiles keeps its own `hit_at`; the shared `struck` is not used for NextHit rows; every touching monster is struck in the same frame | V read both sides (game: SKLIGHT Q3 / Q4; d2d: missiles.cpp:587-596, tick.cpp:394-411) | **model on the V-run flights** (part C; window 4 frames; 3 / 4 frame windows give the same picture): all monsters within 13 subtiles: game.exe rule 0.99 hits / cast, d2d rule 6.16 (seed 1, 8,364 monsters) and 6.25 (seed 7, 16,275). By distance and size (game / d2d): 1 subtile size 1 1.0 / 8.2, size 2 1.0 / 23.7, size 3 1.0 / 33-35; 4 subtiles 1.0 / 3.0, 1.0 / 7.2-7.4, 1.0 / 9.0; 8 subtiles 1.0 / 1.3, 1.0 / 3.4, 1.4 / 4.8; 11 subtiles 1.0 / 1.0, 1.0 / 2.6, 1.0 / 3.0. **d2d deals several times game.exe's Frost Nova damage per cast at the ranges the bot casts it.** The model also gives game.exe a SECOND hit on big monsters (size 3 at 6-10 subtiles: 1.4 to 2.0 hits; size 2 at 7-10: up to 1.5) once `justhit` ends while the slowed ring still covers them: U, drill N3 | **none exists.** Buildable: (a) game.exe side natively under the emulator: part B's 64 real missiles plus monsters made by FUN_00555230 (missiles.py's `Game.unit`), FUN_00554200 hooked hostile to them, and the REAL FUN_005adf10 / FUN_005adb00 (not hooked) with the roll FUN_005a89a0 hooked to count hits; needs the game frame (+0xa8) advanced and the timed-state list run each frame so `justhit` ends (the part this card did not build). Oracle: hits per monster per cast. (b) d2d side: a test mode in tests/game that builds a `Fight` with N monsters at given subtiles (sizes 1-3, distances 1-13), casts Frost Nova once through `Fight::fire`, steps 15 frames and prints each strike (frame, monster). Inputs: 1-6 monsters, random subtiles, levels 1 / 20. Fix shape (for a later PR): the unit-level NextHit window (SKLIGHT D1) or, as the minimum for the ring, push NextHit strikes to the shared `struck` with a frame stamp |
| D2 | ring directions | integer offsets (truncated 30 x cos / sin), aim row trunc(small x 127 / big) | float even angles, `missile_aim_along` | V run (game flights) vs a line copy of d2d's aim (S) | 40 of 64 directions take another aim row (k = 1: game (4087, 257), d2d (4077, 385)); 240 of 384 flights (seed 1) and 320 of 512 (seed 7) differ in some frame; endpoints up to 0.38 subtile apart; 112 subtiles a ring entered by one side only. Effect on hits: small (which monster at the ring's edge is touched) | **partly exists**: `tools/emu/sk_nova_ring.py` part B (game.exe natively vs the line copy). For the real d2d build: the D1(b) test mode printing each ring missile's 16.16 position per frame, diffed against part B. Fix shape: a 64-entry offset table in d2d (the two tables) aimed at caster + offset |
| D3 | calc1 of do 22 | added to the VELOCITY (FUN_005c9b50 -> params +0x28, flag 4) | added to the RANGE (`missile_info.range + calc1`); LevRange not added | V read (game); S (d2d) | 0 for Frost Nova, Nova, Poison Nova and the monster novas (calc1 blank, no LevRange, D). Howl only (Barbarian): calc1 = 2 (lvl - 1): game.exe flies faster for 12 frames, d2d slower for more frames; the reach is the same product (no Accel on `howl`), the timing differs | none needed for the Sorceress; small pure-function check if Howl matters |
| D4 | units per ring missile per frame | one (the first that passes the filter) | every touching monster | V read both sides (SKLIGHT D2) | folded into D1's model | as D1 |
| D5 | the last frame | no unit test on the frame the range ends | not re-run (SKMISS section 2, U) | S / U | at most one frame of reach (about 0.64 subtile at the ring's edge) | as D1(b) |
| D6 | monster novas on the player (Izual, Mephisto's frost nova, Cold Enchanted death nova, Trap Nova) | NextHit rows like the player's ring (D); whether `justhit` blocks re-hits on a player as on a monster: not read | each foe once per ring (ai.hpp:790-792) | U | if game.exe re-hits a player after 4 frames, d2d UNDER-rates those novas; not traced | game.exe side as D1(a) with a player unit |

Equal (no deviation found): 64 missiles; start at the caster's subtile; the target point is ignored (the ring is always round the caster); row `frostnova`; Range 14 and velocity 4608 for Frost Nova (calc1 0); the flight rules (S, missiles.py); damage built from the caster's skill (S, #154); no to-hit roll.

## 5. The bot (internal/combat; described, not changed)

| id | finding | where | tag | impact |
|---|---|---|---|---|
| B1 | the crowd rule counts only hostiles within `novaRadius` 4 subtiles (`h.Dist > novaRadius`: stop), at least `novaCrowd` 3 non-immune; the ring hits every monster within about 11 subtiles (size 1) to 12 (sizes 2-3) once each, on open ground | internal/combat/castfirst.go:20-21, :105-116 | V run (reach) + I (model radius) + S (our code) | the bot holds Frost Nova back from packs that stand 4-11 subtiles away, where one cast would chill all of them; with 2 monsters within 4 and more behind it does not cast. A radius near 10 (a margin under 11 for walls and the last-frame rule) after drill N3 |
| B2 | `skills.json` gives Frost Nova range 5 (Kolbot) but it is never read for it: `castRange` skips KindSelf skills and `novaTick` uses `novaRadius` 4 | internal/combat/classdata/skills.json:1; castfirst.go:51 | S | two numbers for one radius, both well under the real one; whichever is kept should come from Missiles.txt (Range, Vel, Accel) |
| B3 | anything about Frost Nova measured on d2d (kills and damage per minute, `novas` in the health line, casts per kill around a nova, soak tuning of `novaCrowd` / `NovaManaFloor`) is inflated by D1: at 1-4 subtiles d2d's nova deals 3 to 35 times game.exe's damage | castfirst.go:117-121, :204; internal/env/tuning_combat.go:49, :79 | I (from D1's model) | do not tune the nova rule from d2d numbers until D1 is fixed upstream; on game.exe a level-1 Frost Nova (2-4 cold per monster per cast) is a chill tool, not a killer |
| B4 | mana gate `nova.Cost > p.Mana` plus `NovaManaFloor` 25; cast at the player's own point | castfirst.go:102, :121 | S + V run | equal to the rules: cost 9 + (lvl - 1) is a whole number (manashift 8: no rounding, SKMANA); game.exe ignores the target point for do 22 (part A: start = the caster's subtile) |
| B5 | `novaGap` 2.5 s against a chill of 200 frames (8 s) at level 1, more per level | castfirst.go:22 | D + I | casting again inside the chill adds only the 2-4 damage; the gap could follow the chill length (resisted) if mana matters. Tuning, not fidelity |
| B6 | Static Field: `staticRange(level) = (level + 4) x 2 / 3`, compared with `Target.Dist` (subtiles, `math.Hypot` of subtile coordinates); game.exe's radius is aurarange `ln12` = 5 + (lvl - 1) = lvl + 4 subtiles, tested as dx^2 + dy^2 <= radius^2 in subtiles (FUN_0056b7e0 / FUN_006492a0) | internal/combat/skills.go:48, :112; fight.go:741; target.go:86, :126 | S (static-field-chill-trace Q1; SKLIGHT Q2 distance; our code) | the bot casts Static Field only within 2/3 of its radius (5 -> 3 subtiles at level 1): conservative, settles the unit question of static-field-chill-trace B1. Inert on fresh characters (docs/fixstatic.md) |

## 6. PR candidates

No PR candidate. D1 is a hit-count rule: it is read on both sides and quantified by a model applied to game.exe flights that were run natively, but the existing differential harness (`tools/emu/sk_missile_hit.py` + `sk_missile_hit_test.cpp`) only compares the damage roll of one hit, so it cannot show mismatches before > 0 and after 0 for this. D2 has a game.exe-side run against a line copy of d2d's aim, not against the real d2d build. No fork branch was made. D1 is the strongest candidate of the Sorceress set once the stateful harness of D1(a)/(b) exists (it is the same harness SKLIGHT's D1 / D2 and SKMANA's D2 need).

## 7. The oracle run (`tools/emu/sk_nova_ring.py`)

`python sk_nova_ring.py 3 1 3000` and `python sk_nova_ring.py 4 7 6000` from upstream's tools/emu (upstream main 72132b99, the worker host; about 16 s each).

| Part | What runs | Result | Tag |
|---|---|---|---|
| A | FUN_0056d400 natively, FUN_0059fa30 hooked to record | 64 calls, flags 7, vel 24 (= FUN_00663270), offsets = the tables, start = the caster | V run |
| B | 64 ring missiles by the real FUN_0059fa30 + srvdofunc, 3 / 4 free spots x levels 1, 20 | 14 frames each, ends 12.00-12.73 subtiles; the reading's port: 0 of 384 / 0 of 512 flights differ; d2d's aim (line copy): 240 of 384 / 320 of 512 differ, 40 of 64 directions | V run (game); S (d2d copy) |
| C | hit model on part B's flights, monsters of size 1-3 within 13 subtiles, lone or packs of 3 / 6 | game.exe rule 0.99 hits / monster / cast, d2d rule 6.16 / 6.25; by distance in D1 | I |

Limits of part C: open ground, monsters do not move, missiles in creation order within a frame, the game.exe `justhit` window taken as 4 frames from the hit frame (3 tried too: same picture), the last-frame rule applied to both sides (to isolate the NextHit rule).

## 8. Drill recipe (live, later; writing it is allowed, running it is not)

Fork verbs on main (docs/forkdbg.md; lab binary docs/d2d-fork.md, docs/forkcheats.md), a throwaway Sorceress, lockstep: `debug stat 12 6`, `debug points 44 1` (Frost Nova), `debug mana full`, `debug spawn <monster> <n> near`, `debug mon set <id> life 1000 maxlife 1000 res cold 0`, `debug cast 44 near` (or `debug cast 44 <x> <y>`: the point is ignored), `step N`, `monsters`, `debug missiles`. On d2d these show d2d's behaviour; the game.exe side needs a hidden game.exe instance and memory reads only (monster life, missile units), no debug verbs.

- N1 (count, start, D2): cast once; on the action-frame step `debug missiles` lists 64 `frostnova` from the caster's subtile. Record each missile's position for 14 steps; compare with part B's table (game.exe) and expect d2d's to differ in 40 of the 64 directions by up to 0.38 subtile.
- N2 (D1, the main check): one monster of size 1 at 1-2 subtiles, life 1000, cold resistance 0; one Frost Nova at level 1. game.exe: life drops by one roll, 2-4 points (512-1023 256ths); d2d today: several rolls (model: about 8 at 1 subtile; a size-3 monster about 30). Repeat at 4 and 8 subtiles (model d2d: 3 / 1.3 hits).
- N3 (reach and second hits): monsters at 10, 11, 12, 13 subtiles, sizes 1 and 3: game.exe hits size 1 up to 11, size 3 up to 12, nothing at 13; a size-3 monster at 7-10 subtiles may take a SECOND hit after 4 frames (model, U): count life drops in 256ths.
- N4 (chill): `monsters` shows the chill state on each monster hit for 200 frames less its cold resistance (normal monster, Normal); a boss / champion chilled, never frozen (freeze-rules-trace drill 3).
- N5 (Static Field radius, legacy characters only): `debug points 42 1`, monsters at 4 and 6 subtiles: hit at 4 (< 5), not at 6; the bot's `staticRange` at level 1 is 3.
- N6 (bot B1, game.exe side): a pack of 3+ at 5-10 subtiles: the bot does not cast Frost Nova (expected with novaRadius 4).

Expected on d2d today: N1 differs slightly (D2), N2 differs a lot (D1), N3 differs (many hits), N4 per the chill candidates, N5 equal.

## 9. Matrix cell updates (docs/research/sorc-skill-matrix.md, proposed text; the merger or the matrix owner applies)

| Where | Cell | New text |
|---|---|---|
| 3.4 Frost Nova (a) behaviour | ring | `srvdofunc 22` (FUN_005c9b50 -> FUN_0056d400): **64 `frostnova` missiles** from the caster's subtile, missile k aimed at caster + (0x6e1288[k], 0x6e1388[k]) = truncated 30 x (cos, sin) of k x 5.625 degrees; calc1 (blank) would add to the velocity; Range 14 frames, velocity 4608 slowing every 5 frames: ends 12.0-12.7 subtiles out; hit radius about 11 (size 1) to 12 (sizes 2-3) [V read + V run `sk-nova-trace.md`]; each monster once per cast by `justhit` (state 0x56, 4 frames) [V read SKLIGHT; second hits on big monsters U]; cold hit: chill only [V rules `freeze-rules-trace.md`] |
| 3.4 Frost Nova (b) d2d status | ring | 64 even float angles (D2, small); **NextHit per missile, so a monster is struck by every ring missile touching it: about 6x game.exe's hits per cast, 8-35x point blank (D1, model I on V-run flights)**; no differential yet (needs the stateful Fight harness) |
| 3.4 Frost Nova (c) bot use | crowd rule | `novaTick` counts hostiles within 4 subtiles (castfirst.go:21) while the ring reaches about 11 (B1); skills.json range 5 unused (B2); d2d-measured nova numbers inflated (B3) [S + V run] |
| 3.5 Static Field (c) bot use | range unit | settled: radius `lvl + 4` subtiles (squared subtile distance) and `Target.Dist` in subtiles: the bot's `(lvl+4) x 2/3` is 2/3 of the radius (conservative) [S, sk-nova-trace B6] |
| section 4 drills, Frost Nova row | missile count | 64 at the action frame, from the caster's subtile; one hit per monster per cast in game.exe (drills N1-N3 of sk-nova-trace.md) |
| section 5 rank 5 (SKNOVA) | status | done 2026-11-01: ring V read + V run; D1 hit count (model), D2 directions; no PR candidate |
| closing open questions | Frost Nova's ring size | resolved: 64 missiles, truncated-table directions, reach 12-12.7 subtiles (sk-nova-trace.md) |
| summary "Coverage is thin" | Frost Nova's ring | move from UNTRACED to traced (V read + V run); the hit count rule is read, its d2d magnitude modelled |

SKMISS (docs/research/sk-missile-trace.md) section 1 row "Frost Nova / Nova (srvdofunc 22)" and its D3 / section 9 / section 10 line "The nova tables" are superseded by this trace: a one-line pointer there is for the merger (below).

## 10. Not traced (U)

- Whether game.exe's `justhit` ends exactly 4 frames after the hit or one frame later (inclusive / exclusive; SKLIGHT Q4: I); the model tried 3 and 4.
- Second hits on big monsters by another ring missile after `justhit` ends (model only).
- The order in which game.exe updates missiles within a frame (decides which ring missile hits; not the count).
- D6: monster novas on a player.
- d2d's last-frame unit test (SKMISS, D5 here).

## Shared-file lines for the merger

docs/decomp-queue.md, row 18, replace the status cell with:

`done 2026-11-01 (SKNOVA, Opus): do 22 FUN_005c9b50 -> FUN_0056d400 read and run: 64 frostnova from the caster's subtile at caster + tables 0x6e1288 / 0x6e1388 (truncated 30 x cos / sin), calc1 to the velocity, Range 14: ends 12.0-12.7 subtiles (384 + 512 real flights, reading 0 differ); D1 d2d strikes a monster with every touching ring missile (NextHit per missile) vs game.exe once (justhit 0x56): model 6.2 vs 1.0 hits per cast, 8-35 point blank; D2 even float angles (40 of 64 directions differ, <= 0.38 subtile); D3 calc1 to range (Howl only); no PR candidate (no stateful harness); bot B1 crowd radius 4 vs reach 11, B3 d2d nova numbers inflated, B6 Static Field range is 2/3 of the radius; docs/research/sk-nova-trace.md; tools/emu/sk_nova_ring.py`

docs/decomp-queue.md, results list, add:

`- 18 Frost Nova ring (SKNOVA, 2026-11-01): FUN_0056d400 makes 64 missiles from the caster aimed at caster + (0x6e1288[k], 0x6e1388[k]), truncated 30 x (cos, sin), velocity override (calc1 added to the velocity, flag 4), Range 14: reach 12.0-12.7 subtiles, hit radius about 11-12 (V read + V run, sk_nova_ring.py); d2d: 64 even angles (D2, small) and NextHit per missile so every touching ring missile strikes (D1: model 6.2x hits per cast, 8-35x point blank); D3 calc1 into range (Howl only); Static Field radius lvl + 4 subtiles, the bot uses 2/3; no PR candidate; bot B1-B6`

docs/BACKLOG.md, add:

`- sknova follow-ups (card sknova, docs/research/sk-nova-trace.md): D1 nova hit count needs the stateful harness (game.exe FUN_005adf10 / FUN_005adb00 with the timed-state list under the emulator + a d2d Fight test mode printing strikes), then an upstream candidate (shared justhit window or ring-wide NextHit stamp) together with SKLIGHT D1 / D2; bot B1 (Frost Nova crowd radius 4 vs reach about 11, after drill N3) and B3 (do not tune the nova rule from d2d numbers until D1 is fixed); drills N1-N6`

docs/INDEX.md, decomp lane line, add after the sk-mana-trace entry:

`[research/sk-nova-trace.md](sk-nova-trace.md) (Frost Nova ring: 64 missiles, tables, reach; d2d multi-hit, row 18)`

docs/research/README.md, add:

`- sk-nova-trace.md (SKNOVA, 2026-11-01): Frost Nova's 64-missile ring from the truncated tables at 0x6e1288 / 0x6e1388, reach 12-12.7 subtiles (V run); d2d hits a monster with every touching ring missile (D1, model ~6x); no PR candidate`

docs/research/sk-missile-trace.md (owned by skmiss, archived): under section 1's "Frost Nova / Nova" row or in section 10, a pointer line:

`> Frost Nova / Nova ring: resolved by docs/research/sk-nova-trace.md (64 missiles, tables 0x6e1288 / 0x6e1388, reach 12-12.7 subtiles; D3 superseded).`

docs/OWNERS, add (card sknova): `docs/research/sk-nova-trace.md  sknova  sknova`, `tools/emu/sk_nova_ring.py  sknova  sknova`, `openspec/changes/sknova/**  sknova  sknova`; then `changecheck status -write`.

CHANGELOG.md: none (docs and an emulator script only).

Glossary: `justhit` (state 0x56, the NextHit unit state; also proposed by SKLIGHT) and "ring" (the 64 missiles of a nova do 22) are not in GLOSSARY.md; proposed for the coordinator, not added.
