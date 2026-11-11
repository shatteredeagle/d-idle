# fid-zeal: the spacing of Zeal strikes, game.exe vs d2d (batch 5 left it unresolved)

> status: finished 2026-11-11 by agent decomp10 (Sonnet), offline. Result: **game.exe restarts the swing after every Zeal strike (Skills Param2 = 100 through FUN_0056e210 / FUN_00553b10), so the next strike comes `action frame` ticks later, about 7 ticks for a one-hand sword at 0 IAS, not a full 15-tick swing. d2d waits the full swing between strikes (V read). A five-strike Zeal takes about 43 ticks in game.exe and 75 in d2d (D arithmetic on the V run table).** The game side ran natively; the d2d side was read, not run; the unit-struct field meaning of the harness is I (checked on 63 cases against the independent swing-length rule). Consumer: TASK-330 (zealdiff). No PR candidate (no d2d differential yet).

Tags: V run (FUN_00553b10 natively in the emulator with a hand-built unit and the event scheduler hooked), V read, D, I, U. Script: `tools/emu/d10_roll.py` (`d10_roll.py` one case; `d10_roll.py sweep` the table).

## Approach (what was different from batch 5)

Batch 5 looked for the rollback by position fields and failed to give them a meaning. Here the function was run with the events hooked: FUN_00540e60 (clear the unit's events of type 0 and 1) and FUN_005417d0 (add an event: tick and flag). The inputs: a game struct whose `+0xa8` is the current tick, a unit with `+0x44` the swing's start tick (shifted 8), `+0x48` the swing length (frames x 256), `+0x4c` the per-tick rate (word), `+0x50` an AnimData record whose bytes from `+0x10` are the per-frame flags (1 = action frame, the hit; 2 = the last frame). That layout is I from the code at 0x553b3c..0x553c3b; the proof of the reading is below (Q2).

## Questions

**Q1. What does srvdofunc 13 do after a strike? (V read FUN_005dbc60)** It resolves the hit (FUN_0057ec10, FUN_0057dbf0, FUN_0057d4f0), then decrements the sequence's remaining-hit counter (FUN_0006444a0 get, FUN_00644560 set); if hits remain it finds the next target with the 0x20003 filter (FUN_0056bd10), sets it, and calls **`FUN_0056e210(unit, FUN_004efc20(skill))`**: FUN_004efc20 returns Skills record `+0x14c` = `Param2` (V read), and FUN_0056e210 calls **FUN_00553b10 with that percent**. Zeal's Param2 is **100** (D, Skills.txt; Fury, which shares the handlers, 100). On the last hit nothing is rolled back and the swing runs out.

**Q2. What does FUN_00553b10 do? (V run)** It clears the unit's pending events, sets the swing's start tick to `now - (now - start) x (100 - pct) / 100` (100 means: start = now), then re-adds the events of the swing for that new start: one per flagged frame as the frames pass at the unit's rate, and a type-1 event at the end. Run for 7 records x 9 rates (63 cases, `d10_roll.py sweep`): the end event always lands at exactly `ceil(256 x frames / rate)` ticks after the strike, the swing-length rule of the attack-rate proof (S, skiasgame2: 198,988 cases), **63 of 63 equal**; that is the check that the fields are read right. With pct 0 the events are those of the original swing (action at start + 8 for frame 7), so the function does move the swing, not just rewrite it.

**Q3. The spacing (V run, harness fields I).** Ticks from a strike to the next action event, and to the swing end, after a rollback of 100:

| Record (Paladin, AnimData) | frames / action frame | rate 256 (IAS 0, WSM 0) | rate 307 (IAS about 20) | rate 358 | full swing (d2d's wait) at 256 / 307 / 358 |
|---|---|---|---|---|---|
| PAA11HS one-hand sword class | 15 / 7 | 7 | 6 | 6 | 15 / 13 / 11 |
| PAA11HT one-hand thrust | 17 / 8 | 8 | 7 | 6 | 17 / 15 / 13 |
| PAA12HS two-hand sword | 18 / 8 | 8 | 7 | 6 | 18 / 16 / 13 |
| PAA12HT two-hand thrust | 20 / 8 | 8 | 7 | 6 | 20 / 17 / 15 |
| PAA1STF staff | 18 / 9 | 9 | 8 | 7 | 18 / 16 / 13 |
| PAA1HTH hands | 14 / 7 | 7 | 6 | 6 | 14 / 12 / 11 |

(rate = AnimData speed 256 x clamp(100 - WSM) / 100 plus the IAS term, fid-pal-attack.md; the two other columns are the sweep rows for rates 307 and 358.) So **a strike lands about `action frame / rate x 256` ticks after the last one, close to half a swing**; the first strike of a cast is at `ceil(256 x (action + 1) / rate)` (8 ticks for the sword class).

**Q4. Whole-cast time (D arithmetic on Q3).** First strike + (n - 1) x spacing + the rest of the swing after the last strike. One-hand sword class, IAS 0: n = 2: 22 ticks, 3: 29, 4: 36, **5: 43**; d2d: n x 15 = 30, 45, 60, **75**. A five-strike Zeal is **1.7 times slower in d2d** with this class; with a Scimitar (rate 320, WSM -20) the same ratio holds within a tick (the shape is action frame against full swing). Mana is paid once per cast in both (fid-pal-attack.md).

**Q5. UseAttackRate. (D)** Zeal's Skills.txt `UseAttackRate` is blank, unlike Attack, Smite, Sacrifice and Vengeance (1): the column is the to-hit flag of the skill, not the speed (not tested here: U).

## d2d (V read)

`components/game/fight/strike.cpp:244-249` sets `kicks_left = calc1 - 1` at cast time; `Fight::player_modes` (strike.cpp:893-910) lets the swing run to `pmode_until`, and only then, with `kicks_left > 0` and a target, calls `set_pmode(swing_mode(), now_ms)` again. No use of Param2 (grep: Param2 appears only in missile code). Each Zeal strike therefore starts a full new swing (strike at the action frame again, 15 ticks apart for the sword class).

## Deviations

| id | game.exe | d2d | Tag | Differential |
|---|---|---|---|---|
| D1 | strike spacing = action frame ticks after the rollback of Param2 (100) | full swing (`pmode_until`) | game V run (fields I), d2d V read | to build (TASK-330): `Fight` with a Zeal cast on a stationary dummy, count ticks between `debug missiles`-less strike events (monster life steps); expected 7 vs 15 |
| D2 | Dragon Talon kicks and Fend use the same counter path (kicks_left) | same full-swing wait | U (their rollback Param2 not looked up) | D2 table row of Dragon Talon, Fend: Param2 from Skills.txt |

## Bot findings (not changed)

- B1 kill-time models for a Paladin on Zeal (`docs/classes`, `internal/combat/classdata` Zeal band, clvl 12) must not assume d2d's pace is the game's: on d2d a five-strike cast takes 75 ticks (3 s) where the game takes 43 (1.7 s). The bot is tuned on d2d, so its Zeal results are conservative by that factor (I). Consumer TASK-330.

## Drill recipe (fork verbs, not run)

R1: lab d2d, throwaway Paladin, `debug points 106 4`, `debug spawn fallen1 1` with `debug mon set <id> life 100000 maxlife 100000`, `debug cast 106 unit <id>` then `step 1` and read the monster life every tick: the strike ticks come out 15 apart (d2d). The game number (7) needs the lab to run game.exe or the harness above. R2: `debug points 106 1` gives 2 strikes at level 1; confirm spacing is the same.

## Matrix proposal

Paladin Zeal row: "mana once per cast V read; strike spacing game 7 ticks (V run harness, fields I) vs d2d 15 (V read): DEVIATES, TASK-330".
