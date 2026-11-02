# Swing speed second pass: game.exe's attack rate run against d2d (SKIAS2)

> status: research, current (card skiasgame2, 2026-11-02). Supersedes the "game.exe side unread" part of `sk-ias-trace.md` (that doc's d2d reading stays valid). Offline: no game, no live d2d, nothing pushed or posted. Evidence tags: V read (disassembly of game.exe 1.14d, the read-only copy on the worker host), V run (game.exe's own code under unicorn, `tools/emu/skiasgame2_rate.py`, run through `tools/emu/run-on-worker.sh`), D (MPQ table), S (upstream doc), I inferred, U unverified.

## Verdict

1. **The plain attack rate is the same number on both sides.** game.exe's attack branch of `00623f50` (modes A1, A2, TH, KK for a player) computes `rate = AnimData speed x clamp(effective IAS + stat 68, 15, 175) / 100` (integer, capped at 0x7fff). d2d's `attack_speed` is `rate x (100 + clamp(eff(ias) - wsm + sias, -85, 75)) / 100`: the same function once stat 68 = 100 - WSM + skill speed (see Q3). V run: **198,988 cases, 0 mismatches** (IAS 0..200 x WSM -60..+60 x four AnimData speeds x modes A1 / A2, plus 4,420 skill-speed cases -100..+120).
2. **The tick count agrees**: 81 cases (frames 8..24, rates 128..448) of game.exe's per-tick step `00623e00` and end test `006217c0`: the ending tick is `ceil(256 x frames / rate)`, d2d's `attack_ticks` (0 mismatches; condition in Q5).
3. **One real deviation, not yet proven by a differential against d2d's code**: game.exe subtracts **30 percentage points** from the attack rate of a player in the Sequence mode (18) when the sequence skill goes through the attack branch; d2d's sequence path (`Fight::start_sequence`, fight.cpp:702-713) applies no such term and times the sequence as `seq.size()` frames at the A1 rate (D1).
4. For the Sorceress none of this changes the bot's play today: plain Attack (A1) is the equal case; Lightning and Chain Lightning are sequence skills whose record points to the cast path (FCR, not IAS; Q4).

## Questions

| Q | Question | Answer | Tag |
|---|---|---|---|
| Q1 | Where is the attack rate stored? | `00623f50` attack branch 0x6241d7..0x62430a: stores the rate as a word at unit +0x4c and as a signed dword at +0x3c. Entered when `00621580(mode, type)` is true (player table `006e8a00`, 20-byte rows by mode, flag at row +0xc: modes 7 A1, 8 A2, 11 TH, 12 KK; rows with +8 set (modes 13..16 S1..S4, 18 SQ) are true only when the unit has an active skill whose record byte +6 has the bit in the global at `006ce274`) | V read; table dump V run (`modes`) |
| Q2 | What is the formula? | 0x6241f9..0x624211: `ebx = 00621740(unit, row 0)` = stat 93 through the effective-stat table (`006e8e24` row 0: stat 93, factor 120): `120 x s / (120 + s)`, signed idiv, 0 stays 0; `esi = ebx + stat 0x44 (68)` (0x624205..0x62420f); 0x62429e..0x6242b2 clamp esi to 15..175; 0x6242b7..0x6242d5 base speed = AnimData record +0xc (or `00646170(unit)` when state 0x26 is on); 0x6242d5..0x6242f5 `speed x esi` unsigned `/100` (the 0x51eb851f multiply), floored at 0, capped 0x7fff | V read; V run (grid) |
| Q3 | Where do WSM and skill speed enter? | Not in this function: stat 68 is read as one number. Base 100 is carried by the stat (a chill is -50 on stat 68, combat.md in upstream docs/research/re), so d2d's `eias = eff(ias) - wsm + sias` equals game's `eff(ias) + (stat68 - 100)` when stat 68 = 100 - WSM + skill speed. How the weapon writes `-WSM` into stat 68 (the item side) was not read | S (chill -50), I (WSM sign and the item path), U |
| Q4 | Which modes use IAS, which do not? | Run with a unit record: A1, A2, TH (mode 11), KK (12) take the IAS rate (same numbers as d2d); SC (10) and DD (17) with stats at 0 give 256 (cast path `6216e0`, row 2 = stat 105 FCR: `0x623fcb..0x624019`, clamp 0..175 on the percent); S1..S4 and SQ without an active skill: stat 69 (other animrate, 15..175) branch, rate 38 = 15 % with stat 69 at 0 (U what stat 69 holds for a player); Sequence with a skill record whose byte +0x12 is 0xa (SC): cast rate, not IAS (`006216e0` 0x6216fa..0x62172c) | V read, V run (`seq`, `skill`; the skill record was a hand-made stand-in, so the record bits are I) |
| Q5 | Ticks: what ends a plain mode? | `00623e00` adds the rate to the position (+0x44) each tick, `006217c0` is true when `position + rate >= end (+0x48)`. Driving both with end = frames x 256: the ending tick is `ceil(256 frames / rate)` (V run, 81 cases, 0 mismatches) = d2d's `attack_ticks`. That end = frames x 256 is how the mode is set up, which was NOT read here (I) | V run (arithmetic), I (setup) |

Reproduce (worker host, each prints its own counts): `tools/emu/run-on-worker.sh skiasgame2_rate.py grid|seq|skill|form|dual|ticks|modes|table`.

## Differential result (V run)

| Run | Cases | Mismatches vs d2d `attack_speed` |
|---|---|---|
| `grid`: modes A1 and A2, AnimData speed 256 / 152 / 128 / 100, IAS 0..200, WSM -60..+60 (stat 68 = 100 - WSM) | 194,568 | 0 |
| `grid` skill speed: IAS 0/20/60/120/200, WSM -20/0/10/30, sias -100..+120 (stat 68 = 100 + sias - WSM; chill is sias -50) | 4,420 | 0 |
| `ticks`: frames 8..24, rate 128..448 | 81 | 0 |
| `form`: player mode 18 under a hand-simulated shapeshift vs d2d with sias -30 | 24,321 | 0 (game.exe equals "d2d + sias -30", not d2d as it is) |

Oracle limits (honest): hooked as stubs, not game.exe's code: the stat getters `00625500` / `00625480` (return the grid's value for stat 93 / 68), `00645270` (shapeshift remap), `00620f00`, `00623c80`, `006235a0` (dual-wield test, 0 unless `dual`), `0063a7b0` (state 0x26 test) and `00646170`. Run as game.exe code: `00623f50` itself, `00621740`, `00621580`, `00621630`, `006216e0`, `006214a0`, `00620250`, `00623e00`, `006217c0`. The AnimData record is a stand-in (+0xc = speed), so "AnimData speed" per class is D from `animdata.d2`, not read here.

Edge cases (V run, `grid`): AnimData speed 1..8 gives the same rate on both sides (d2d's `max(.,1)` never bites because the percent is at least 15); speeds above about 20,000 hit game.exe's 0x7fff cap (d2d has none; no real record is that large, I); a negative IAS stat of -50 or lower: game 38 (15 %), d2d 128, and -121 or lower divides by a negative denominator in game.exe (448) while d2d clamps; no gear gives such a stat (I).

## Deviations d2d vs game.exe

| D | Rule on both sides | Tag | Differential |
|---|---|---|---|
| D1 | Sequence mode (player mode 18) through the attack branch: game.exe `0x624290..0x62429b` (`cmp [unit], 0; cmp [unit+0x10], 0x12; sub esi, 0x1e`): rate percent - 30. d2d: `start_sequence` (fight.cpp:702-713) times `seq.size()` frames at the A1 AnimData rate through `attack_ticks` with no -30, never sees mode 18 | V read + V run on game.exe (24,321 cases equal "sias -30", with the shapeshift and skill record stubs); d2d side V read; the Skills.txt rows that reach this path (anim SQ with UseAttackRate: Jab, Impale, Double Swing, Double Throw, Frenzy, D) and the real record bits are U | exists for game.exe's side (`skill`, `form`); the d2d side needs a probe build (copy `tools/emu/ms_build.sh`, call `Fight::start_sequence` timing for a Frenzy-like record) or the drill below. Not proven against d2d's code, so no PR candidate |
| D2 | Dual wield (Assassin): game.exe `0x62421a..0x62428e` averages the two weapons' stat 68 contributions (`esi += (a + b) / 2 - a`) when `006235a0` is true and both items pass `00629bb0(item, 0x2d)`; d2d: one `wsm` from the main weapon | V read; semantic of the 0x2d test and of stat 68 on items U | needs an item record stand-in for `00629bb0` and `0063bde0`; Assassin is not in the bot plan, low value |
| D3 | A unit under state 0x26 uses `00646170(unit)` as the base speed instead of AnimData +0xc (0x6242bf..0x6242d2) | V read; V run: with the stub returning 300 the rate is 300 at IAS 0 (`dual` mode, last line); what state 0x26 is: U | none in d2d (no such state use); only matters if that state is reachable |
| D4 | Shapeshift: `00645270` rewrites type, class and mode of a form before the rate is computed, so a player's form takes the monster table rows and the monster modes 4 / 5; d2d's druid forms were not read here | V read (0x645311..0x645378); d2d U | out of scope for the bot (no druid plan) |
| D5 | Monster attack: same function with type 1 (monster modes 4, 5 take the IAS branch, V run: rate 256 at stat 93 = 0, stat 68 = 100). d2d feeds MonStats `attackrate` as IAS (riskmap D3); which stat game.exe holds the monster's rate in (stat 93 or 68) is not known | S riskmap, U | build with a monster unit stand-in and the MonStats `aidel` / `attackrate`; part of monster-attack-trace work |

## Bot findings (described, not changed)

| B | Finding |
|---|---|
| B1 | The resource simulator models one melee swing per fixed second with no frame count, IAS or WSM: `MeleeSec` is 1.0 and 0.6 in the two scenarios (`internal/resource/sim.go:133`, `:150`) and the swing advances time by that constant (`sim.go:382`). With the rate now known equal to d2d's, a breakpoint-aware swing time (frames x 256 / rate / 25 s) could replace the constant; staff WSM and a Sorceress's A1 frame count (AnimData, D from `animdata.d2`) are needed. |
| B2 | The bot reads no IAS: `internal/d2dsrc/player.go:298` states there is no effective stat vector (FCR, FRW, IAS, FHR, MF), so a bot decision that depends on swing speed cannot be made from the d2d state. Not changed. |

## Matrix cells proposed

`docs/research/sorc-skill-matrix.md` line 217 (plain Attack, SKMELEE row): replace "Speed / IAS not traced [U]" with "Speed / IAS: game.exe attack rate (FUN_00623f50) equals d2d `attack_speed` over 198,988 cases and the tick count over 81 [V run, skiasgame2-trace.md]; WSM into stat 68 [I]". Lightning / Chain Lightning rows (SKLIGHT, line 250): add "sequence skills with seqtrans SC take the cast path (FCR), no -30 [V read, V run with a stand-in record]".

## PR candidates

No PR candidate. The only mismatch (D1) is game.exe vs d2d's source by reading and by a run that stubs the skill record; the existing harness does not run d2d's sequence code, so "mismatches before > 0, after the fix 0" cannot be shown.

## Dead ends

- Reading `-30` as a class check: `[unit+0x10]` is the unit's current mode (the same field the earlier checks compare with 6 / 9 / 4), 0x12 = 18 = Sequence in `plrmode.txt` (D, the 18th row), not a class.
- A first run with no skill record gave 38 for modes 13..16 and 18 (the stat 69 branch); this looked like "IAS ignored for sequence skills" until the record flag at `006ce274` (= 8, V run) and the `00620250` skill pointer were supplied.
- `00645270` first looked like a sequence remap; it is the shapeshift remap (states with byte +0x2d 1 or 2).

## Not covered (limits)

Class AnimData frames and speeds, the action frame (the frame at which the hit lands), the dual-wield averaging and its item test, Whirlwind / Zeal / Fury sequence frame counts (`seqnum`), throw and bow animation lengths, how WSM becomes a stat 68 contribution, monster stat sources, and everything live.

## Drill recipe (lab binary; docs/forkdbg.md, docs/d2d-fork.md, docs/forkcheats.md)

Not run. In the lab: with a Sorceress and `debug stat` / `debug points` per docs/forkcheats.md, swing a staff at a spawned dummy (`debug spawn zombie1 1 near`), reading `dump` for the player mode and the frame count between the A1 start and the end: repeat at IAS values straddling the table (`python3 -I tools/emu/skias_bp.py 16 256 0`: 8/9, 17/18, 29/30) to confirm Q2 on d2d; for D1 give a Barbarian-style sequence skill (Frenzy) and compare the sequence length at IAS 0 and 60 with the prediction `ceil(256 x frames / (speed x (70 + eff(ias)) / 100))` (game.exe) against d2d's `seq.size()` frames at the A1 rate.

## Shared-file lines for the merger

See openspec/changes/skiasgame2/merge-lines.txt. Missing glossary terms: none.
