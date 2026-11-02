# Independent re-check of the resistance rules Q1 to Q10 (VERIFY-SKRESIST)

> status: card verifyskresist, 2026-11-02, offline (no game, no d2d slot, nothing pushed). Verifier: Claude Sonnet. Order of work: I read FUN_00573cb0, FUN_0057be00, FUN_0057bf80 and FUN_005c3540 first and wrote my own models, then ran them against game.exe natively, and only then opened `docs/research/sk-resist-trace.md`.

Tags: **V read** read in the disassembly (upstream `disasm.py`, patched game.exe 1.14d); **V run** game.exe's own code run natively under the emulator and compared with my own model; **D** MPQ table (patched stack); **U** not verified. No code is pasted.

## Harness (mine)

- `tools/emu/vr_dis.py ADDR COUNT`: disassembles the patched game.exe.
- `tools/emu/vr_tbl.py monstats.txt`: MonStats resist-column summary.
- `tools/emu/vr_resist.py N SEED`: three parts, each N cases drawn by Python's `random.Random(SEED)` with boundary values (99, 100, 101, 0, -1, -100) mixed in:
  - **A** FUN_005c3540(unit, stat, value) for every stat 0x20 to 0x30, target a monster, a player or a hireling (hook of FUN_0063ee90). Model: only the six resist stats, value <= 0, monster, base >= 100 give value/5 truncated toward 0; all else pass through.
  - **B** FUN_0057be00 on each of the six resist rows, defender plain monster or player/hireling, difficulty byte 0..2, classic penalty (0/-20/-50) and expansion penalty (DifficultyLevels value), pierce, max-resist stat. Model written from my reading: pierce only if total < 100 (or defender not plain); penalty only for non-plain and not for physical/magic; floor -100; cap 75 + max stat, at most 95 (physical without max stat: 50) for non-plain only.
  - **C** FUN_0057c6c0 (lightning field) against a plain monster with a 30,000-point life: damage taken in 256ths vs `roll*(100-res)/100` truncated, 0 when res >= 100, res after pierce (< 100 only) and floor.
- Scaffolding (unit stubs, stat-getter hooks) is `life_apply.py` from the skmana read checkout; no case, model or claim comes from `sk_resist.py`.
- Run: `tools/emu/run-on-worker.sh vr_resist.py 20000 11` and `... 50000 2025`.

Result: seed 11, 20,000 per part: **0 mismatches** in A, B and C (7,205 immune-zero damage hits). Seed 2025, 50,000 per part: **0 mismatches** (17,891 immune-zero hits). Total 70,000 cases per part.
My first model of C used round-to-nearest and failed 3,307 of 20,000: game.exe truncates (e.g. 24057 at res 99 gives 240; 16395 at res -1 gives 16558). Fixed model, rerun above.

## Verdicts

| Rule | Verdict | Evidence |
|---|---|---|
| Q1 spawn base = MonStats column of the difficulty (6 stats, hireling reads Normal) | **HOLDS WITH LIMITS** | V read FUN_00573cb0: difficulty = game byte +0x6d clamped 0..2, 0 for a hireling; stats 0x24/0x25/0x27/0x29/0x2b/0x2d from record offsets 0x144/0x14a/0x150/0x156/0x15c/0x162 plus 2*difficulty; nothing else is added in that block. D: columns are ordered Normal, (N), (H) for ResDm/Ma/Fi/Li/Co/Po. Limit: I did not RUN FUN_00573cb0 (U for a run) and did not re-check boss/champion mods. |
| Q2 immune rows per difficulty | **HOLDS WITH LIMITS** | D (vr_tbl.py, patched stack): rows with any column >= 100: ResDm 23, ResMa 11, ResFi 114, ResLi 105, ResCo 137, ResPo 134 of 735 rows. The author's per-difficulty counts not recomputed. |
| Q3 boss/champion/minion mods | **U** | not looked at (FUN_005a1370 outside my scope). |
| Q4 resistance a hit uses (pierce under 100 only for monsters; floor -100; monster has no penalty and no cap) | **HOLDS** | V run, part B, 70,000 cases, 0 mismatches; floor -100 holds for all six rows incl. physical and magic. |
| Q5 damage formula | **HOLDS** | V run, part C, 70,000 cases, 0 mismatches: MulDiv(dmg, 100-res, 100) truncated; res >= 100 gives 0; negative res up to x2. Limit: the lightning field only (fire/cold/poison fields not exercised here; B covers their resistance). |
| Q6 curses/auras on an immune, 1/5 rule | **HOLDS** | V run, part A, 70,000 cases, 0 mismatches over all stats 0x20..0x30: only the six resist stats, change <= 0, monster, BASE >= 100; truncation toward 0 (-31 gives -6, -125 gives -25). Players and hirelings take the full value. |
| Q8 pierce never breaks immunity | **HOLDS** | V run, part B (plain monster, total >= 100: pierce ignored). |
| Q10 penalty and cap only for non-plain defenders | **HOLDS WITH LIMITS** | V run, part B: monsters get neither; players/hirelings get -20/-50 (classic) or the DifficultyLevels value, none on physical/magic, and the 75+max (95 top) cap, 50 for physical with no max stat. Limit: the physical-immunity side check in FUN_0057be00 (a skill-0x2f test, FUN_00639df0/FUN_0063e990) is hooked to "no" in the harness: U. |
| Q7, Q9 | **U** | curse values and what an immune hit still does: not re-checked. |

No rule FAILS. Differences from the author's numbers: none in the rules; counts differ because my cases are my own (70,000 vs 147,628), as intended.

## Not covered (limits)

d2d itself was not built or run (D1 is the author's real-build result, not re-checked); melee and the get-hit path; the 0x2f physical special case; Q3/Q7/Q9; FUN_00573cb0 not executed.

## Deviations and bot findings

None new. Author's D1 (d2d adds the full curse/aura value on an immune) is consistent with the game rule I confirmed; a differential needs the real d2d build, not run here.

## Matrix cells

No change proposed beyond adding "independently re-checked (verify-skresist.md): Q4, Q5, Q6, Q8 HOLD" to the resist rows.

## Shared-file lines for the merger

See `openspec/changes/verifyskresist/merge-lines.txt`.
