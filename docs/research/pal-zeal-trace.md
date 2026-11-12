# PALZEAL: the Zeal strike spacing and cast time, game.exe 1.14d against d2d's real Fight (decomp13, TASK-330 d2d side)

> status: finished 2026-11-12 by agent decomp13 (Sonnet), offline (emulator and a d2d test binary on the worker host; no game, no live d2d, no Go change, nothing pushed or posted). Result: **the d2d side of fid-zeal.md is now V run: d2d's real `Fight` strikes Zeal 15 ticks apart (the full swing) at 0 IAS, 13 at IAS 20, 12 at IAS 40, 19 at WSM +20; game.exe's rollback restarts the swing at each strike so the next strike lands 7 / 6 / 6 / 9 ticks after (the action frame). All 6 rate rows differ, the whole 5-strike cast is 75 ticks in d2d against 43 in game.exe at 0 IAS (1.74 times).** The game side is the same V run as decomp10 (field meaning I); no PR candidate (the harness is new, not the existing differential; no fix built). Consumers: TASK-330 (zealdiff), docs/research/pal-skill-matrix.md.

Tags: **V run** (the game.exe function or the d2d class executed: `tools/emu/zd_game.py` runs FUN_00553b10 natively; `tools/emu/zd_zeal_test.cpp` drives d2d's real `Fight::engage` / `Fight::player_modes` / `start_swing`), **V read**, **D** (1.14d tables), **S** (earlier trace), **I** inferred, **U** unverified. Follows [fid-zeal.md](../probes/fid/fid-zeal.md), [fid-pal-attack.md](../probes/fid/fid-pal-attack.md), [pal-smite-zeal-sacrifice-trace.md](pal-smite-zeal-sacrifice-trace.md) (D9) and [palskill-trace.md](palskill-trace.md). No game code is pasted.

## Questions answered

**Q1. What does d2d do between two Zeal strikes? (V run)** A real `Fight` with a Paladin (class 3, the starting Short Sword `ssd` and Buckler), skill 106 at level 4, a passive dummy (a Zombie row with a million life) one cell away, ticking `Fight::world`, `Fight::engage` and `Fight::player_modes(held=true)` once per 40 ms tick. Result (rate 256, WSM 0, IAS 0): strikes (the tick `pstruck` turns true) at 7, 22, 37, 52, 67; the swing mode ends at 75; the next cast starts at 76. **Gap between strikes inside a cast: 15 ticks every time (the swing length of PAA11HS: 15 frames); between casts 16 (one idle tick).** Same at skill level 1 (2 strikes: 7 and 22, 15 apart). Mana: 3 casts in 160 ticks, 3840 -> 2304 (256ths) = 2 mana per cast, once, not per strike (V run, equals the game's srvstfunc 37 payment, fid-pal-attack.md Q2).
Harness: `tools/emu/zd_zeal_test.cpp` (built by `zd_build.sh`, run by `zd_d2d.sh`), compiled against the d2d scratch build tree on the worker host. Its `Fight` logic is the same as the upstream read tree at e36922ea (`fight.cpp:2293`: `kicks_left > 0 && next_target()` then `set_pmode`, with no use of Param2); the scratch tree has no git metadata, so the exact commit of the linked library is U.

**Q2. What does game.exe do? (V run, field meaning I)** Same as fid-zeal.md Q1-Q3, re-run here for the exact rates d2d printed: FUN_00553b10 (called by FUN_0056e210 with Skills Param2 = 100 after every strike but the last) resets the swing's start to now and re-adds the events for the flagged frames, so the next action frame comes `ceil(256 x action / rate)` ticks later and the swing ends `ceil(256 x frames / rate)` after the strike (the end equals the full-swing rule for all 6 rates, which is the check that the harness fields are read right).

**Q3. The comparison (V run both sides; unit fields I).** PAA11HS (Short Sword class: 15 frames, action frame 7, D from AnimData.d2):

| IAS / WSM | rate (d2d `attack_speed`) | d2d gap (V run) | game.exe next strike (V run) | game.exe swing end after a strike | whole 5-strike cast d2d / game |
|---|---:|---:|---:|---:|---:|
| 0 / 0 | 256 | 15 | 7 | 15 | 75 / 43 |
| 20 / 0 | 299 | 13 | 6 | 13 | 65 / 37 |
| 40 / 0 | 332 | 12 | 6 | 12 | 60 / 36 |
| 0 / -20 (Scimitar) | 307 | 13 | 6 | 13 | 65 / 37 |
| 40 / -20 | 384 | 10 | 5 | 10 | 50 / 30 |
| 0 / +20 (War Hammer) | 204 | 19 | 9 | 19 | 95 / 55 |

(`zd_game.py 256 299 332 307 384 204`; cast totals: game = first strike + (n - 1) x spacing + the rest of the swing after the last strike, first strike = the same action event; d2d = n x full swing. The d2d first strike lands at tick 7 at rate 256, equal to the game's action frame, so the first strike is not part of the deviation.) **All 24 inter-strike gaps (6 rates x 4) differ; the d2d gap equals the full swing, the game's is 46 to 49 percent of it.** n = 2, 3, 4: 30 / 45 / 60 against 22 / 29 / 36 at 0 IAS.

**Q4. Is Zeal faster than a plain Attack on each side?** Per hit at 0 IAS: game.exe Zeal 7 ticks (cast of five: 8.6 per hit including the tail), plain Attack 15 (S, 198,988 cases); d2d Zeal 15 = d2d Attack (and d2d Attack held repeats every 16: Q1 run with skill 0: strikes at 7, 23, 39, 55, an idle tick between swings; the game's idle tick for a held plain attack is U). So on d2d, Zeal's only edge over Attack is its AR bonus and its synergy ED; on game.exe it is 1.74 times the strikes per second (x1.4 per hit counting the tail at level 1, x1.74 at level 4: 5 hits).

**Q5. Skill delay and UseAttackRate (D).** Zeal has no `delay` and no `perdelay` (blank in Skills.txt) and `UseAttackRate` blank (Attack, Sacrifice, Vengeance have 1). Whether the blank flag changes the swing speed of Zeal in game.exe (does IAS count?) is U: fid-zeal.md Q5; the d2d harness uses IAS for Zeal. Drill Z3.

## Deviations d2d vs game.exe

| id | rule on both sides | tag | differential today / buildable |
|---|---|---|---|
| D1 | game.exe: next strike `action` ticks after the last (rollback Param2 = 100); d2d: `pmode_until`, the full swing, 1.74x slower for a 5-hit cast | game V run (fields I), d2d V run | built: `zd_zeal_test` (d2d) + `zd_game.py` (game), 24 of 24 gaps differ. NOT an existing differential (new harness), no fix built, so no PR candidate. |
| D2 | d2d idles one tick between held swings (plain Attack 16, Zeal casts 16 between casts); the game's held-button repeat tick is U | d2d V run, game U | a game-side run needs a full unit loop (FUN_005dbc60 and the mode change); lab drill H1 |
| D3 | Dragon Talon kicks and Fend use the same `kicks_left` path in d2d (carried from fid-zeal.md D2, U for their Param2) | U | read Skills.txt Param2 for rows 5 / Fend: one table dump |

Why no PR candidate: the rule says a PR candidate needs a d2d deviation proven with the EXISTING differential harness and a fix with after = 0. Here the harness is new (this card), the d2d side ran on a scratch copy and the fix is not built. What a fix would be (sketch only, I): in `Fight::player_modes`, when a strike has just landed and `kicks_left > 0`, set `pmode_until = strike tick + ceil(256 x action_frame / rate) x 40 ms` for the NEXT strike instead of waiting out the swing; then `zd_zeal_test` would print gaps 7 / 6 / 6 / 6 / 5 / 9 and zd_game.py gives the same numbers (after = 0 on 6 rates x 4 gaps). The swing end and the animation drawn (the client plays the mode again from frame 0?) would also have to be matched: U, drill Z4. Build it only after the upstream owner agrees which tree "upstream main" is (the same open point as batch 7).

## Bot findings (described, not changed)

- **B1** Any bot kill-time number for Zeal measured on d2d is pessimistic by the factor in Q3 (the bot is tuned on d2d). `internal/combat/classdata/profiles.json` Paladin band `minLevel 12 requires Zeal` (lines 144-170 region): no change needed, but the soak figures of a Zeal Paladin are not a statement about the real game. I.
- **B2** The held-button model: the bot holds the left button on the target (client repeat); each Zeal cast costs 2 mana once, so the bot's `manaReserve 4` leaves two casts (profiles.json melee block). On d2d a 5-hit cast lasts 75 ticks (3.0 s), in the game 43 (1.7 s), so the mana drain per second is 1.74x higher in the real game (I; 2 mana per 1.7 s = 1.2 mana/s at level 4).

## Drill recipe (lab verbs of docs/forkdbg.md; nothing run)

Z1 (d2d, mirrors this harness): throwaway Paladin, `debug points 106 4`, `debug spawn zombie1 1`, `debug mon set <id> life 100000 maxlife 100000`, `debug cast 106 unit <id>`, then `step 1` per tick printing the monster life: strikes 15 apart, five per cast. Z2 (retarget): two monsters in reach, life 1 on the first: strike 2 hits the second. Z3 (IAS): equip a weapon with IAS, `dump stats` and repeat Z1: gaps 13 / 12 / 10 (table above). Z4 (game.exe, needs a live game and the lockstep clock, INSTRUMENTATION with the user's go): a throwaway Paladin on a hidden instance with Zeal 4 and a Short Sword, lockstep stepping, read the target monster's life every tick (read-only), expect strikes 7 apart and a five-strike cast of about 43 ticks; this also settles the rollback field meaning (I) and the held-button idle tick (D2). H1: hold the attack on a dummy with plain Attack, count swings per 400 ticks: d2d 25 (16 apart), game expected 26 or 27 (U).

## Matrix-cell updates (proposed text for the Paladin matrix)

Zeal row, speed cell: "game.exe 7 ticks per strike (action frame; V run, fields I), d2d 15 (V run through the real Fight); a 5-strike cast 43 vs 75 ticks at 0 IAS: D1, TASK-330; no PR candidate yet". See [pal-skill-matrix.md](pal-skill-matrix.md).

## Files

`tools/emu/zd_zeal_test.cpp`, `zd_build.sh`, `zd_d2d.sh` (d2d side, on the worker host through `run-on-worker.sh`), `tools/emu/zd_game.py` (game side). Re-run: `D2_LABRUN_HOST=<host> tools/emu/run-on-worker.sh --ttl 1500 zd_build.sh`, then `... zd_d2d.sh 106 4 IAS WSM TICKS` and `... zd_game.py 256 299 332 307 384 204`.

## Dead ends

The IAS runs set `player_combat.ias / wsm` directly on the actor (the same fields `set_pmode` reads) instead of equipping items; a first try that set `prate` after the swing began did nothing (the swing length is fixed in `set_pmode`). Two weapon classes only (Short Sword 15 / 7); the other records of fid-zeal.md Q3 were not re-driven on the d2d side because d2d's swing is `attack_ticks(frames, speed, ias, wsm)` for all of them (V read, setup.cpp:380).
