# fid-revive-mode (TASK-316, matrix rows MA08, MA13): what a Fallen Shaman does to a corpse, and the Fallen flee rule

> status: finished 2026-11-11 by agent decomp9 (Sonnet), offline (emulator on the worker host, no game, no live d2d). Result: **the raised monster is alive at once (life = max life, mode DD 12 -> S1 8 for a Fallen, in the same server step); d2d puts it in NU (1) instead (D1, V run for the game side, V read for d2d). No PR candidate** (no d2d-side differential harness for the raise). Consumer: TASK-142 (the bot's `Alive()` and hostile filters).

Tags: V run (game.exe's own code run by the emulator), V read (disassembly; function tags only), D (MPQ tables), S (existing trace), I, U. Tools: `tools/emu/d9_revive.py` (run: `tools/emu/run-on-worker.sh --ttl 1200 d9_revive.py 0`), `d9_dotab.py` (finds the srvdofunc table), `d9_dis.sh` (disassembly ranges), `d9_flagscan.py`.

## Questions

| Q | Question | Answer | Tag |
|---|---|---|---|
| Q1 | Which skill and which do-function raise the corpse? | MonStats `fallenshaman1..5` Skill1 = `Resurrect` (Skills id 181, anim SQ, Sk1mode `seq_shamanresurrect`), `srvdofunc 97`. The srvdofunc table in game.exe is at 0x7322b0 (found by the known entry 58 = FUN_005c56c0, the necromancer's Revive); entry 97 = **FUN_005ccb10**. `Resurrect2` (id 327) is the same do. | D (tables); V read (`d9_dotab.py`) |
| Q2 | What must be true for the raise to happen? | FUN_005ccb10 looks the target up by id (FUN_00553540), then needs: unit type 1 and mode 0 (DT) or 12 (DD) (FUN_0063ea40, V read: it returns true only for those two modes); FUN_0063a320 false (a list test on the game, meaning U); the corpse's spot open to its pattern under mask 0x1c09 for a monster (FUN_0064d910; d2d's comment gives 0x3c01 for its `usable`, a different mask, U which one is right). Any failure returns 0 and nothing changes. The think (not the do) only offers DD corpses: its callback FUN_005f1380 wants mode 12 and the corpse bit, class Fallen / Fallen Shaman, within aip4 squared, not flagged 0xc (S monsterai2-trace.md Q4, V run there). | V read; S |
| Q3 | What does the revive step do to the corpse? | FUN_005cc960 (V run, `d9_revive.py`, real code on a Fallen and a Fallen Shaman made a corpse by hand): **life (stat 6) := max life (stat 7)** (Fallen 0 -> max, Shaman 0 -> max); unit flags 0x0008200e -> 0x040a200e, i.e. **+0x4000000 and +0x20000** (monsters only); back into the room list. It does **not** change the mode: mode stays 12 after it. | V run |
| Q4 | Which mode does the raised monster get, and when? | FUN_005ccb10 then asks FUN_005fd7b0(unit, 1) for the mode: it returns the byte at +0x10b of the 0x134-byte record the unit's class points to = the **MonStats2 `ResurrectMode`** column, builds a mode request with FUN_005a7e60 and applies it with FUN_005a7c20 in the same call. Run natively on a real spawned unit of every MonStats row that spawns: **703 rows; mode 1 (NU) 638, mode 8 (S1) 14, mode 14 (SQ) 51; 578 rows have a monstats2 row and all 578 equal `ResurrectMode` (NU / S1 / xx -> 1 / 8 / 14), 0 disagree**. Fallen 1..5 and Mummy 1..5 are S1; Skeleton and Skeletal Mage families are SQ; Zombie, Brute, Goatman, Fallen Shaman and the rest are NU. The full request path run on the hand-made corpses: Fallen -> unit mode **8**, Fallen Shaman -> mode 1. | V run (mode value 703 rows, request path 2 rows); D (column) |
| Q5 | So what does the bot see during a raise? | Corpse mode 12 (DD) -> the raised Fallen is **alive with full life** in mode **8 (S1)** in the same server step (no frame in mode 0 or 12 in between), then NU when the S1 animation ends. S1 for the Fallen is 24 frames at speed 256 (`FAS1HTH` in the AnimData of the same MPQ, D), so about 24 ticks (about one second at 25 fps) if the monster's S1 plays at rate 256 (the rate of a monster's skill mode, stat 69, was not read: I). What the Shaman does meanwhile is mode 14 (SQ, the `seq_shamanresurrect` sequence). Whether a unit in S1 can be targeted or hit: not read (U, see Q8). | V run + D (modes), I (length) |
| Q6 | Does a raised monster give experience or drop items? | d2d's comment says the flags 0x4020000 mean "no experience, no drop" and sets `no_exp` / `no_drop`. A scan of game.exe's `.text` for instructions that touch the flag word (+0xc4) with an immediate of 0x20000 / 0x4000000 finds only writers (about 25 sites in the AI, summon and spawn code, among them the revive step and the Foul Crow Nest "gone" ending); **no reader with such an immediate**, so the claim could not be confirmed or refuted from the disassembly (the reader may test with a register mask). | S (d2d comment), U (dead end: `d9_flagscan.py`) |
| Q7 | The Fallen flee rule | Unchanged from monsterai2-trace.md Q2 (V run there): a Fallen with a kin in mode 0 (DT, dying; not DD) within `FUN_005dc530` < 15 sets its state to 1, clears the command, gets pace +50 for its next mode (path speed Velocity x (75 + 50) / 100 = 125 percent) and backs off 12 subtiles from the target; no other branch of the Fallen think reads life. So there is **no "flee at low life" rule** for the Fallen (I from the two reads; no life test appears in the think summaries of Q2 and Q3 there). The shaman has no flee branch (Q4 there: rally, corpse raise, Skill2 fire, circle). | S + I |
| Q8 | Can the bot hit and target a unit in S1? | Not traced. FUN_0057edd0 / FUN_0057eec0 (the mode gates of a mode change) belong to the hit-recovery gap D1 of fid-hit-recovery.md; whether a hit lands on a monster in S1 is U. | U |

## Deviations (d2d against game.exe)

| id | Rule in game.exe | Rule in d2d (upstream 83fc7ba1, `components/game/ai.cpp` around line 2574) | Tag | Differential |
|---|---|---|---|---|
| D1 | The raised monster gets the MonStats2 `ResurrectMode`: S1 (8) for Fallen and Mummy, SQ (14) for the skeleton families, NU (1) otherwise | `set_mode(game_data, raised, "NU", now_ms)` for every class | game side V run (703 rows), d2d side V read | Game side built (`d9_revive.py`, lines `M ...`). d2d side: the raise is inline in the monster loop, not a function, so there is no test mode; build one as a `test_game` scenario (a Fallen Shaman, a Fallen corpse in DD within aip4, step until the corpse moves, read its `mode`); oracle = the `M` lines. Effect on the bot: a raised Fallen on d2d is in NU at once, on game.exe in S1 for about 24 ticks; both are "alive", so the bot's filter is unaffected; timing of the first attack differs by that S1 length (I) |
| D2 | Revive sets +0x4000000 and +0x20000 on the unit flags | d2d sets `no_exp` and `no_drop` | game flags V run, meaning U (Q6) | none until a reader of the flags is found (read the kill-award FUN_0057e2f0 / treasure caller with a register mask search, or one live raise with the XP read) |
| D3 | Spot test under mask 0x1c09 (FUN_0064d910, monster) | `usable` uses mask 0x3c01 (ai.cpp:1702-1708) | V read vs V read | buildable with the existing walk tests: run FUN_0064d910 natively with each mask on corpse spots next to walls and units; U which mask the game applies (the 0x1c09 immediate is V read, the call arguments after it were not decoded) |

## Bot findings (described, not changed)

- B1 `internal/d2/units.go:80` `Alive()` = mode is neither 0 nor 12. Correct for a raise: the raised unit leaves 12 for 8 or 1 in one step. No change needed. The comment at `units.go:7` says the mode numbers are "UNVERIFIED except 1": game.exe's own code agrees (FUN_0063ea40 treats exactly 0 and 12 as the dying/dead modes of a monster; monmode.txt order DT=0, NU=1, WL=2, GH=3, A1=4, A2=5, BL=6, SC=7, S1=8, S2..S4=9..11, DD=12, KB=13, SQ=14, RN=15, D from monmode.txt) so the comment can name this trace (V read + V run).
- B2 A raised Fallen can be seen in mode 8 (S1) with life > 0. Anything in the bot that reads "mode 8 = casting / skill" as a reason to ignore a monster (a hostile filter by mode) would skip it for about a second; grep found only `Alive()` and the 0 / 12 tests at `internal/combat/target.go:12`, `necro.go:110`, `necro.go:327`, `necro_corpse.go:53`, `trap.go:168`, so today it is fine.
- B3 `internal/combat/necro.go:110` and `necro_corpse.go:53` take mode 12 as "a corpse"; game.exe also accepts mode 0 (DT) for the raise (Q2) but a Shaman's think only picks DD, so a corpse is mode 12 only (consistent).

## PR candidate

No PR candidate: D1 is proven on the game side only; the existing harnesses (`mon_ais.py`, `test_monster_ai`) do not drive the corpse raise, and a PR needs mismatches before > 0 and 0 after on a differential that runs d2d's code.

## Drill recipe (fork verbs, docs/forkdbg.md and docs/forkcheats.md; written, not run)

1. D1/Q5: `debug spawn fallenshaman1 1 near`, `debug spawn fallen1 2 near`, `debug kill` one Fallen, wait for mode 12 in `dump monsters`, then `step 1` repeatedly and print each Fallen's `mode` and life per tick: game.exe (expected, this trace): 12 -> 8 for about 24 ticks -> 1, life = max at the first tick; d2d: 12 -> 1.
2. Q6: the same with the player's XP before and after killing the raised Fallen (`dump stats`): expected unchanged if the flags mean "no experience".
3. Q7: three Fallens, `debug kill` one, read the others' positions per tick for 40 ticks (monsterai2-trace.md recipe 1).

## Matrix-cell updates (proposed text)

- MA08: append "UPDATE 2026-11-11 (decomp9, fid-revive-mode.md): a Fallen Shaman's raise runs FUN_005ccb10 / FUN_005cc960 (V run): full life, mode from MonStats2 ResurrectMode (703 spawnable rows, 0 disagree with the column); d2d sets NU for all (D1)."
- MA13: no change.

## Limits and dead ends

- The harness spawns real units (FUN_00555230 in a booted game, difficulty 0) and calls the revive step with the other callees real; the mode request (FUN_005a7e60 + FUN_005a7c20) returned 1 on the two hand-made corpses with no event scheduling checked. The mode-end event that takes S1 back to NU was not run.
- The scan for readers of the flag bits (Q6) found none with an immediate mask: dead end, written down, U.
- The life values in the harness are in 256ths of the unit stat (Fallen 256 = max), not hit points as the client shows them.
