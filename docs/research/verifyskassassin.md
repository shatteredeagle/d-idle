# Verify SKASSASSIN: independent re-check of the Assassin skills first pass

> status: current (card verifyskassassin). Verifier: independent Opus pass with its own harness (`tools/emu/vsa_dis.py`, `tools/emu/vsa_mana.py`, `tools/emu/vsa_diff.sh`). Order kept: I disassembled the two mana functions and wrote my model into the script header BEFORE running, then ran game.exe, then compared with the first pass (docs/research/skassassin-trace.md) and with SKMANA (docs/research/sk-mana-trace.md, docs/research/verify-skmana.md).

## Which passes exist

- First pass: docs/research/skassassin-trace.md on main (static, table-derived; claims nothing V run, no ledger rows).
- Second pass ("credit job with runs"): NOT FOUND. No branch, ref or worktree named for it exists in this clone (searched `git branch -a`, `git for-each-ref`, `git worktree list` for `assass`/`skass`). Nothing of it is verified here; if it lands later it needs its own check.

## Own model (V read, from my disassembly of game.exe 1.14d)

- M1 check `FUN_0056c160` (ecx = unit): not a player (dword 0 != 0) returns 1; current skill entry from `FUN_00620250`, none returns 0; item-charge entry (`FUN_00643ad0` != -1) returns charges `[entry+0x38] > 0`; else Skills row `FUN_00644140`, level `FUN_006442a0(unit, entry, 1) - 1` floored at 0 (`sub eax,1; jns`); `f = (lvlmana(+0x18c, s16) * lvl' + mana(+0x18a, s16)) << manashift(+0x188, byte)`; srvdofunc (+0x2e) == 0x74 and shapeshifted (`FUN_0063a400`) returns 1; result `stat8 >= f` (signed). minmana (+0x186) is never read.
- M2 payment `FUN_0056bfe0(skill, lvl)` (edx = unit): player gate; item-charge path to `FUN_0056bec0`; row from `[0x744304]+0xb98`, stride 0x23c, bounds `+0xba0`; mana == 0 and lvlmana == 0 returns 0 (no payment); `cost = max(f, minmana << 8)` with the level argument used directly; state 0x72 (blood_mana, `FUN_00639df0`) pays from life via `FUN_005d2b60`; `stat8 < cost` returns 0 and takes nothing; else `FUN_006272b0(unit, 8, -cost, 0)`.

This agrees with verify-skmana C1-C6 (I read it only after writing M1/M2).

## Runs (V run)

1. `tools/emu/run-on-worker.sh --ttl 900 vsa_mana.py 20000 20261102`: game.exe's real `FUN_0056c160` + `FUN_0056bfe0` on the Skills table game.exe itself loaded, ids 251..280, levels 1..99 x mana at {0, f-1, f, f+1, minmana*256-1/0/+1} plus 20,000 random cases (seed 20261102). Result: `cases 39305 check-passed 25310 paid 23549 model mismatches 0`. Check passed but nothing paid only for the two passives (252: 883, 263: 878; mana and lvlmana 0). The minmana floor never binds on an Assassin row (every row with a cost has `f >= 256`, Blade Fury has minmana 0).
2. `tools/emu/run-on-worker.sh --ttl 1500 vsa_diff.sh 20000 20261102`: the same 39,305 cases through d2d's real `rules::mana_cost` (tools/emu/sk_mana_test.cpp compiled by hand against the sorcrest scratch build, patched table loaded: `Loading: patch_d2.mpq`, `Skills: 357 rows`). Result: `vsa diff cases 39305 differ 0` (the only diff hunk is d2d's 15 log lines at the top of stdout). Limit: the d2d side is the scratch build's source, not today's upstream main.
3. `tools/emu/run-on-worker.sh mpq-table.sh skills` (D): the patched skills table; columns skill, Id, charclass, srvstfunc, srvdofunc, startmana, minmana for 251..280 equal the fields game.exe loaded in run 1 (srvst +0x2c, srvdo +0x2e).

Hooked in runs 1-2 (inputs only, not the claim): `FUN_00620250`, `FUN_00643ad0`, `FUN_00644140`, `FUN_006442a0`, `FUN_00625480`, `FUN_0063a400`, `FUN_00639df0`, `FUN_006272b0` (recorded).

## Claim verdicts (first pass)

| claim | verdict | evidence |
|---|---|---|
| 30 rows Id 251..280, charclass `ass` | VERIFIED | D (run 3) |
| srvdofunc per row (34 x3, 35 x3, finishers 42/46/50/52, Shock Field 43, sentries 45, Blade Sentinel 44, shadows 49, buffs 18, 33/47/48/51/54, Fire Trauma none, passives none) | VERIFIED | D (run 3) and the fields game.exe loaded (run 1): 30 of 30 equal |
| "Fists of Fire / Claws of Thunder / Blades of Ice: srvstartfunc 1" | WRONG | srvstfunc is 23 for all six charge-up rows (254, 259, 265, 269, 274, 280); the 34/35 split is srvdofunc only. The first pass also omits srvstfunc for 13 rows (253: 22, 255: 24, 260: 25, 266: 26, 270: 27, 275: 12, 277: 28, plus the six above). This matters: per SKMANA (S) rows with a srvstfunc pay mana at the start, the rest at the action frame |
| Quickness at 258 is the Burst of Speed row, Venom at 278 | HOLDS WITH LIMITS | internal names D (run 3); the display-name mapping is I (string table not read) |
| mana formula `max(minmana, ((mana + lvlmana*(lvl-1)) << shift) >> 8)` as "the game.exe rule" | WRONG as a statement of game.exe | that is d2d's rule (SKMANA D2: floor in the check). game.exe's check has no floor (M1) and pays in 256ths, not whole points. For the 30 Assassin rows the two give the same answer (run 2: 0 differences) because the floor never binds |
| mana table L1/L20 (60 cells) | HOLDS WITH LIMITS | all 60 equal floor(game.exe cost / 256) (run 1); five rows pay a fraction the table truncates: Fire Trauma L20 5.375, Psychic Hammer L20 8.75, Blade Fury L20 3.375, Shadow Warrior L20 36.5, Shadow Master L20 44.5 |
| table misses startmana | omission | Blade Fury has startmana 6 (D, both runs); SKMANA D7 (client status below startmana x 256) applies to it |
| Q2 "these rows are not yet fed to the SKMANA differential" | WRONG | SKMANA's `gen` walks every one of the 357 rows (S, sk-mana-trace section 3), and no Assassin row is among its mismatching skills; now also run here for levels 1..99 with 0 differences (V run, run 2) |
| D1 candidate (rounding of shift 5..7 rows) | resolved: no deviation | V run, both sides executed (run 2), not a re-derivation: the oracle is game.exe's own code and d2d's own `mana_cost` |
| Q3 charge-ups, Q4 traps / shadows | NOT VERIFIED (nothing claimed) | U in the first pass, still U |
| "no ledger rows claimed" | correct | nothing in the first pass was V run, so no row was earned by it |

## What each pass got right and wrong

- First pass: right on the row set and every srvdofunc; wrong on srvstartfunc and on attributing d2d's floored rule to game.exe; its mana numbers are right in their integer part only by luck of the data (no Assassin row has `f < minmana`). Its "no deviation proven" was honest.
- Second pass: not available; no verdict.

## Deviations d2d vs game.exe (Assassin rows)

- None in the mana check or payment amounts for 251..280 at levels 1..99 (V run, 39,305 cases, 0 differences).
- SKMANA's timing deviations D1 (pay at command vs action frame), D3 (interrupted cast), D4, D5, D7 (Blade Fury startmana) apply unchanged (S, not re-run).

## Bot findings

- B1 `internal/data/skills.go:132` `ManaCost` = `max(formula/256, MinMana)`, rounded up at `internal/combat/skills.go:110`: conservative against game.exe (pays fractions, checks without the floor); for Assassin rows the floor never binds, so only the ceil differs (Fire Trauma L20: bot 6, game 5.375). No defect; described, not changed.

## PR candidate

No PR candidate: the differential shows 0 differences for these rows.

## Ledger and matrix

- Ledger rows earned by this verification: the mana check and payment cell for skills 251..280 (V run, both sides, levels 1..99). docs/coverage/ledger.tsv is not owned here; the merger may mark those 30 rows' mana cell tested with a pointer to this doc. Every effect cell stays untested.
- Matrix text (proposed for the Assassin rows): "mana: V run, game.exe = d2d, levels 1..99 (verifyskassassin); srvstfunc 23 for the six charge-ups; effect: U".

## Drill recipe (U left)

Q3/Q4 need stateful runs, not this harness: with the forkdbg verbs (docs/forkdbg.md) give an Assassin Tiger Strike and Dragon Claw, cast the charge-up three times, read the progressive state and its counter, finish, read the state again; for traps place Lightning Sentry and count its shots and lifetime. Not run.

## Shared-file lines for the merger

See openspec/changes/verifyskassassin/merge-lines.txt.
