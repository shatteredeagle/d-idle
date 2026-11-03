# Missiles coverage, first pass: table values, flight formulas and the hit roll of the Sorceress and Paladin rows (card covmissiles)

> status: card covmissiles, 2026-11-02, offline (no game, no d2d slot). Coverage gap "missiles" (684 rows, 9 percent run-tested before this card). Two differentials run on the worker host under upstream's emulator: (A) the element hit of eight more Sorceress skills, (B) every one of the 684 rows made by game.exe (`FUN_0059fa30`) against the table and against the real d2d loader and rules. Nothing is changed in d2d or in the bot.

Tags: **V run** = game.exe's real code under the emulator, counted cases, compared with the real d2d build (or a port, said where); **V read** = read in the disassembly or in d2d's source; **D** = from the patched 1.14d tables (`mpq-table.sh`); **S** = an earlier doc of ours or upstream's `docs/research/re/missiles.md`; **I** = inferred; **U** = unverified.

## Summary

- **(B) Table and flight, all 684 rows.** game.exe's loaded Missiles.txt record agrees with the patched table on 17 columns of all 684 rows (0 disagreements), and `FUN_0059fa30` makes every row at 20 levels with `range = Range + LevRange x lvl`, `velocity = ((Vel + VelLev x lvl / 8) << 8) x 75 / 100`, `MaxVel << 8` and `Accel` as the rule says: 13,680 cases, 0 mismatches (V run). d2d's loader and `rules::missile_velocity` / `missile_range` give the same values on the 302 rows d2d loads (6,040 cases, 0 differ). d2d does NOT load the other 382 rows (V read + V run): no value to compare, so those rows stay untested.
- **(A) Hit roll, eight more skills.** D1 of SKMISS (the roll must not reach the top of the pair) is real for Inferno, Blaze, Fire Ball, Fire Wall, Glacial Spike, Chilling Armor, Thunder Storm and Fist of the Heavens too: 138,000 cases, d2d before the fix differs on 40,850, after the fix on 0; the element min / max (synergies, mastery) agree on all 138,000 (V run). No new deviation id: it is the same rule, same function, same fix.
- **Not a new PR candidate.** D1 already has its fix on a local branch (`23e2baec`, the SKMISS fix; its draft is docs/research/pr-playerroll-draft.md, verified by an independent verifier); this run only adds evidence for it. No new deviation was proven.
- **Ledger.** Six hit rows (60 Inferno flame, 62 Fire Ball, 96 Glacial Spike, 166 Thunder Storm, 233 Fist of the Heavens, 264 Chilling Armor: 17,242 + 17,261 + 17,169 + 17,285 + 17,174 + 17,308 = 103,439 cases) and 222 table / flight rows not claimed before (222 rows x 20 levels = 4,440 cases, TESTED-0-DIFF, loader and flight values only) earn a ledger line (docs/coverage/ledger.tsv); rows whose rule I did not run stay untested (Blaze 67 and Fire Wall 69 hit cases are not counted, see Dead ends).

## What ran

| run | command (worker host, repo tools/emu) | cases | seed | result |
|---|---|---:|---|---|
| etype and element rows | `run-on-worker.sh cm_run.sh rows` | 60 skill rows | none | etype 1 fire, 2 lightning, 3 magic, 4 cold in game.exe's Skills table (read by `skill_damage.Oracle.row`); the eight skills below match the table in `cm_missile_hit.py` (the `game` mode aborts otherwise) |
| (A) hit, run 1 | `cm_run.sh hit 20000 1` | 64,000 (44,000 grid + 20,000 random) | 1 | before: 18,331 lines differ; after: 0 |
| (A) hit, run 2 | `cm_run.sh hit 30000 7` | 74,000 (44,000 grid + 30,000 random) | 7 | before: 22,519 differ; after: 0 |
| (B) flight | `cm_run.sh flight` | 13,680 = 684 rows x 20 levels (1 2 3 4 5 7 8 9 12 16 17 20 22 23 28 29 30 35 40 60) | none (the shooter's subtile from `Random(1)`) | game vs table 0; d2d 6,040 compared, 0 differ |

Build: `cm_build.sh` compiles `sk_missile_hit_test.cpp` against the scratch builds `hit_before` (upstream main 72132b99 plus the Paladin skill checkout) and `hit_after` (with the SKMISS fix 23e2baec), and the new `cm_flight_test.cpp` (real `load_game_data`, real `components/rules/missiles.hpp`) against the first. Per-skill counts of (A): `cm_bycase.py DIR`. The flight comparison: `cm_flightcmp.py DIR`. Which ids got a ledger line: `cm_ledger_rows.py`.

(A) per-skill cases (runs 1 + 2) and lines differing before the fix:

| skill | Skills id | missile row(s) (D, Missiles.txt `Skill` column) | cases | differ before | differ after |
|---|---:|---|---:|---:|---:|
| Inferno | 41 | infernoflame1 60, infernoflame2 61, infernoflame3 575 | 17,242 | 8,230 | 0 |
| Blaze | 46 | blaze 67 | 17,314 | 9,405 | 0 |
| Fire Ball | 47 | fireball 62 | 17,261 | 2,776 | 0 |
| Fire Wall | 51 | firewall 69 (the maker, 68, has no Skill) | 17,247 | 7,698 | 0 |
| Glacial Spike | 55 | glacialspike 96 | 17,169 | 2,660 | 0 |
| Thunder Storm | 57 | thunderstorm1-4 166-169 | 17,285 | 3,188 | 0 |
| Chilling Armor | 60 | chillingarmorbolt 264 | 17,308 | 5,989 | 0 |
| Fist of the Heavens | 121 | fistoftheheavensdelay 233 | 17,174 | 904 | 0 |

## Questions

| Q | Question | Answer | Where | Tag |
|---|---|---|---|---|
| Q1 | How does game.exe lay out a Missiles.txt record, and does its loader agree with the table? | Record 0x1a4 bytes at `*(PTR_DAT_00744304 + 0xb64)`: Range +0x96, LevRange +0x98 (shorts), Vel +0x9a, VelLev +0x9b, MaxVel +0x9c (bytes), Accel +0x9e (short), Activate +0x136, CollideType +0x183, ClientCol +0x185, CollideKill +0x186, CollideFriend +0x187, NextHit +0x188, NextDelay +0x189, Size +0x18a, ToHit +0x18b, AlwaysExplode +0x18c, Trans +0x18d. All 17 columns read back from game.exe's memory equal the patched table on all 684 rows (0 disagreements; blank = 0) | `cm_flight.py` COLS; offsets from docs/research/re/missiles.md | V run (offsets S, confirmed by the 0 disagreements on 684 rows) |
| Q2 | What range, velocity, MaxVel and Accel does `FUN_0059fa30` give a missile? | range = Range + LevRange x lvl (a short, clamped to -32768..32767), read at `*(missile + 0x14) + 0x10`; path velocity (+0x7c) = ((Vel + VelLev x lvl / 8, C division) << 8) x 75 / 100; MaxVel << 8 at path +0x84, unscaled; Accel at path +0x88. 684 rows x 20 levels, 0 mismatches against the formula, all 13,680 made (none refused) | `FUN_0059fa30` (ECX game, EDX params flags 0x21, level at [0xc]) | V run |
| Q3 | Speed in subtiles per frame | velocity x 0x400 >> 6 = velocity x 16 in 16.16, so Vel x 192 x 16 / 65536 = Vel x 0.046875 subtiles a frame for a missile with no Accel and no VelLev (arithmetic on the Q2 values; the step rule is upstream's `missiles.py`, proven there) | `FUN_00650840` / `FUN_006502d0` | I on V run values; step rule S |
| Q4 | Which rows have Accel, VelLev, LevRange? | Accel nonzero on 21 rows (nova, frostnova, nova1, nova2, trapnova, howl, shout, battlecry, warcry, battlecommand, battleorders, blessedhammer, blessedhammerex, blade shield missile, baal inferno, queenpoisoncloud, lightning charge up nova, clawsofthundernova, lightningtowernova, coldunique, ...): novas lose 1000 a step, Blessed Hammer gains 250 up to MaxVel 30; VelLev nonzero on 4 (spike1-4); LevRange nonzero on 32 (blaze, firewallmaker, firesmall, firemedium, throwingstar, ...) | Missiles.txt | D |
| Q5 | The hit of an element missile: pair, roll, apply | min / max built at creation (`FUN_0064b860` -> `FUN_00644d50` / `FUN_00644e40` with synergies and mastery), the roll `FUN_005a89a0` -> `FUN_005a8910` (nothing unless both ends over 0; `min + rand(max - min)`, the top is never rolled), then `FUN_0057c6c0` (resistance less pierce, MulDiv, absorb, life in 256ths). Same chain as SKMISS, run for 8 more skills | docs/research/sk-missile-trace.md | V run (138,000 cases) |
| Q6 | Which d2d code reads which missile column? | `gamedata_load.cpp:337-362` reads Size, VelLev, MaxVel, Accel, Activate, Vel, Range, SrcDamage, Min/MaxDamage, AnimSpeed, AnimLen, Trans, Light, Skill, LevRange, pSrvHitFunc, sHitPar1, ToHit, CollideKill, Pierce, ReturnFire, pCltDoFunc, pSrvDoFunc, Param1-3, sHitPar2, NextHit, NextDelay, the three sub-missile names, ExplosionMissile, AlwaysExplode, EType, EMin, Emax, HitShift, HitClass, sounds, ELen, EDmgSymPerCalc, SHitCalc1, the level brackets and CelFile. It never reads CollideFriend, CollideType, ClientCol, ClientSend, KnockBack, SoftHit, CanSlow, Holy, Town, NoMultiShot, Qty, SrcTown, CanDestroy, LastCollide, Collision (0 matches in `components/`, `cm_sweep.sh`) | upstream main + SKMISS checkout | V read |
| Q7 | Which rows does d2d load? | Only a row that a skill, a monster's MissA1 / MissA2, a trap, a boss mod or an explosion of those names uses (`gamedata_load.cpp:329-336`, two passes): 302 of 684 (V run: the other 382 print "not loaded" in `flight_test`, 7,640 of 13,680 cases) | `gamedata_load.cpp` | V read + V run |
| Q8 | What does d2d do with CollideKill and Pierce? | `missiles.cpp:632`: the missile is spent when `collide_kill` and NOT (`pierce` and `rng(100) < pierce stat`). game.exe's own pierce rule (stat 328 percent, where it is read) is not read by this card | `fight/missiles.cpp:632` | V read (d2d); game side U |

## Deviations d2d vs game.exe

| D | Rule in game.exe | Rule in d2d | Tag | Differential |
|---|---|---|---|---|
| D1 (extends SKMISS D1) | the element roll is `min + rand(max - min)`: the top of the pair is never rolled | before the fix `min + rand(max - min + 1)`: the top can be rolled | V run: 138,000 cases, 40,850 differ before, 0 after (8 skills; table above) | yes, built and run: `cm_run.sh hit` |
| D2 | game.exe loads and makes all 684 rows | d2d loads 302 rows; every other row has no velocity, range, flags or element in d2d | V read + V run (382 rows, 7,640 cases "not loaded"). Whether a skill that needs one of them (sanctuarybolt, sanctuarycenter, redemption, redemptionfail, blessedhammerex, blizzardexplode1-3, thunderstorm2-4, infernoflame2-3, frozenorbexplode, meteor, meteorexplode, meteortail) is carried by another d2d rule: U | differential exists for the values (the table values cannot be compared where there is no d2d value); for the behaviour it needs the rows to be loaded first |
| D3 | `CollideFriend` (+0x187) lets a missile strike friends: Holy Bolt (55), Fist of the Heavens bolt (234), healingbolt (357), radamentholybolt (349), impfireballexplode (405), catapultchargedballbolt (408) carry it; `CollideType` (+0x183) picks the collision mask `DAT_0073c724[type]` and unit filter `DAT_0073c720[type]` (type 3: `0x184` and `FUN_005a8850` -> `FUN_005a8730`: targetable, not the last unit hit, hostile unless CollideFriend) | neither column is read (Q6) | game: S (docs/research/re/missiles.md) and D (which rows); d2d: V read. No V run of the filter | could be built: call `FUN_005a8730` (unit filter) in the emulator for (row, shooter, target) cases (a player, a merc, a pet, a hostile, a neutral) and compare with d2d's friend test of the same missile; oracle = the boolean per case. U until run |
| D4 | game.exe's pierce chance and `Pierce` column (which rows: 12, 22-26, 28, 31, 92, 205, 206, 214, 434, 641, 683, ... D) | `pierce && rng(100) < pierce stat` (Q8) | d2d V read; game side not read: U | could be built: find the reader of the record's Pierce byte (offset not in the S doc; fit it by column against the 684 rows), run `FUN_005adf10` (the hit) on a pierce row with a stat 328 value, count survivals over seeds |

## Bot findings (described, not changed)

| B | Where | What | Tag |
|---|---|---|---|
| B1 | `internal/data/extra.go:361-366` `RangeAt` | `Range + LevRange x (lvl - 1)`; game.exe makes `Range + LevRange x lvl` (Q2). Wrong by one LevRange for the 32 LevRange rows (firewallmaker at level 1: game 9 frames, bot 7). No non-test caller today (so no effect now); `data_test.go:371` pins the `lvl - 1` form | V run (game), V read (bot) |
| B2 | `internal/combat/threat.go:15-36` | Speed = Vel x 0.046875 x 25 subtiles a second is right for a missile with no Accel and no VelLev (Q3, 25 fps is S). It ignores Accel / MaxVel (21 rows: a nova slows by 1000 path units every fifth frame and stops; Blessed Hammer, Blade Shield and Baal's inferno speed up) and VelLev (spike1-4: Vel + VelLev x lvl / 8). `Classify` therefore uses the launch speed: too fast for a decelerating nova, too slow for a hammer. Whether the monster rows that matter (novas) reach `Classify` as dodgeable: U | V run (rule), V read (bot), I (effect) |
| B3 | `internal/combat/classdata.go:183-193` | The derived default says Bolt vs Ground from `srvmissile` only when `range` is `rng` or `both`; in Skills.txt every Sorceress and Paladin missile skill has `range` `none` except Inferno (`rng`, `srvmissile` blank, `srvmissilea` infernoflame1), and 11 of them keep their missile in `srvmissilea`. So the derived default gives KindBuff for all of them and KindGround for Inferno. Only matters when a profile lacks an explicit kind (profiles carry their own): U whether reached | D (columns); bot V read; reach U |

## PR candidates

None new. D1 is proven by the existing differential on 8 more skills (before 40,850 differing, after 0), but its fix is the already existing local branch of the SKMISS card (the PLAYERROLL draft); adding these skills changes the evidence pack, not the patch. D2 to D4 have no run. The independent Opus verifier is not needed for this card.

## Dead ends and limits

- The flight walk (aim table, subtile stepping, room handling, Charged Bolt's wiggle) is upstream's `missiles.py` proof (S) and was not redone; this card runs the table-driven inputs of it for all rows.
- Rows 67 (Blaze) and 69 (Fire Wall) have `pSrvDmgFunc` 3; I did not read what that function does to the roll, so the 17,314 and 17,247 hit cases of those two skills are NOT counted for the ledger rows 67 and 69 (they stay untested / read-only). Fist of the Heavens: row 233 carries the lightning pair; the holy bolt 234 has its own magic row element (40-50) and its hit function 22 / 7 chain was not read; the d2d test only switches fire, lightning and cold, so magic rows cannot run on this harness.
- Rows d2d loads but never flies (monster-only rows) are compared on loader and formula values only; the ledger line says so.
- `labrun` rejects `|` in script arguments: my first grep for several column names returned nothing because of that, not because d2d lacks them; the sweep of Q6 is a script (`cm_sweep.sh`) with one pattern per loop step.
- The `(Vel + VelLev x lvl / 8)` division is C division (truncation toward zero); no VelLev is negative in the table (D), so the case where it matters was not exercised.

## Drill recipe for the fork verbs (lab only, one hidden slot, throwaway Sorceress, nothing here was run)

Verbs: docs/forkdbg.md (`debug cast`, `debug missiles`, `debug points`, `debug spawn`, `debug mon set`, `dump stats`); binary and lockstep: docs/d2d-fork.md, docs/forkcheats.md.

- C1 (D1 live, any of the 8 skills): `debug points 47 1`, `debug spawn` a monster, `debug mon set` its life large and its fire resistance 0, `debug cast 47 unit <id>` 50 times with `step 1`, read the life drop per hit from `monsters`. Fire Ball's pair is EMin 12 / EMax 28 with HitShift 7 (D, Skills.txt row 47), so about half of that in points at level 1 before brackets and synergies (I): the 256ths drop must never equal the top of the pair after the fix (compare with `dump stats` life raw). Same for 55 (Glacial Spike, cold), 60, 57.
- C2 (Q2 live): `debug cast 36 <point 60 subtiles away>` then `debug missiles` each frame: the frame count equals `Range` (50) and the speed 4.688 cells a second (docs/forkdbg.md M3 already lists it); repeat with `blessedhammer` (`debug points 112 20`): speed must rise from the launch value (Accel 250 every fifth frame up to MaxVel 30) - the check for B2.
- C3 (D2): `debug cast 119` (Sanctuary) and `debug cast 121` (Fist of the Heavens) on a spawned monster, `debug missiles` per frame: a missile row of a name in the D2 list appearing or not tells whether another d2d rule carries it.
- C4 (D3): Holy Bolt (`debug cast 101 unit <merc or pet id>`): a friendly target must be struck (healed) by game.exe; compare in d2d.

## Matrix-cell updates (proposed text; docs/research/sorc-skill-matrix.md is not edited here)

Append to the "END / LEG note" column of rows 41, 46, 47, 51, 55, 57, 60: "hit roll D1 proven (covmissiles, 17k cases per skill; the roll never reaches the top of the pair); min / max (synergies, mastery) agree with game.exe". For rows 46 and 51 add "pSrvDmgFunc 3 not read (U)". Rows 36, 38, 39, 44, 45, 48, 49, 53 keep the SKMISS evidence. Add to the Paladin rows 112 (Blessed Hammer): "Accel 250, MaxVel 30: speeds up in flight (D); d2d reads Accel, bot threat table does not (B2)".

## Shared-file lines for the merger

See `openspec/changes/covmissiles/merge-lines.txt` (BACKLOG, INDEX, CHANGELOG lines). Ledger and generated table: `docs/coverage/ledger.tsv`, `docs/coverage/rule-coverage.md` (regenerated here).
