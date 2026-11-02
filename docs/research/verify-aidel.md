# VERIFY-AIDEL: independent re-check of the monster think period

Verifier: Claude Sonnet, own harness, written before reading the author's numbers beyond the function names. No game file or disassembly is pasted. Tags: V run (game.exe code under the emulator against an oracle written here), D (MonStats data), S (upstream/author), I inferred, U unverified.

## Verdicts

| Claim | Verdict | Basis |
|---|---|---|
| C1. MonStats `aidel` is 15 on the Normal column of the rows read, and later columns differ per difficulty | HOLDS (D) | Patched stack, 735 rows: `aidel` = 15 on 710, empty on 19, 1 on 5, 6 on 1. `aidel(N)` mostly 14 (394) / 15 (255) / 13 (47); `aidel(H)` mostly 13 (393) / 15 (255) / 12 (31) / 11 (13). 455 rows differ between the three columns. |
| C2. FUN_005a73e0 files the next think at frame + aidel, aidel read by game type and difficulty, 0 -> 15, nothing filed when a pending type-2 event is later than the frame, frame + 0x2d when state 0x15 is on | HOLDS (V run) | 352,800 cases, 0 differ (below). |
| C3. d2d `set_off` uses `aidel`, 15 when 0 | HOLDS WITH LIMITS (S) | Matches the rule above; I did not run d2d code (no d2d probe build in this job), I only compared game.exe with the stated rule. |
| C4. "55,268 equal cases" | HOLDS WITH LIMITS | Different harness, same verdict. The author's oracle is a Python copy of the d2d rule fed synthetic aidel 0..255; mine uses the real table rows. Their count is not reproduced (mine is 352,800 over a different grid); no difference found. |
| C5. Resumes thinking at freeze end (FUN_005a7c20 path, 0x57b170 callback) | NOT CHECKED (U) | Out of this job's time; FUN_005a7c20 was not driven. See limits. |

## Method and command

`tools/emu/va_aidel.py` summarises the table (columns, empties, per-difficulty counts). `tools/emu/va_aidel_run.py` runs game.exe's FUN_005a73e0 natively (upstream `emu.py`, hooks only on FUN_005415a0 pending frame, FUN_00639df0 state 0x15, FUN_005417d0 the filed event). For every one of the 735 MonStats rows, in classic and expansion mode, difficulties 0..2, 20 random frames each (seed 11), six situations (no pending, pending frame-1, frame, frame+1, frame+50, stunned state 0x15). Oracle written independently: filed frame = frame + A, A = aidel byte (+difficulty in expansion), 0 -> 15; nothing when pending > frame; frame + 0x2d when stunned.

```sh
tools/emu/run-on-worker.sh mpq-table.sh monstats 2>/dev/null | python3 -I tools/emu/va_aidel.py
tools/emu/run-on-worker.sh --ttl 900 va_aidel_run.py 11 20    # last line: rows 735 cases 352800 differ 0 seed 11 (44 s)
```

Result: **352,800 cases, 0 differ, seed 11.**

## Differences from the author's numbers

- Empty `aidel` rows: I count 19 (author: 18). Rows with 15/13/12: author 30, my H column shows 31 rows at 12 across all groups (grouping differs, I did not tie out individual rows). Both are D counts from a script over the same table; cause unchecked. Neither changes the rule, since empty -> 15 is run-verified.
- The row-offset mapping (columns `aidel`, `aidel(N)`, `aidel(H)` to row bytes +0x4f..+0x51) is taken from the author (S); my harness writes the bytes at those offsets, so it proves the function's use of them, not the loader's column-to-byte mapping (U).

## Limits (not covered)

- FUN_005a7c20 and the freeze-end callback were not driven; the "resumes at freeze end" claim is not re-checked here (see pr-freezeresume-proof.md for the author's differential).
- game +0x6a (game type) value 3 for single player: harness used 0 and 1 only; the author says 3 selects the difficulty column. I did not run 3.
- No d2d binary was run; C3 compares against the rule text.
- Stunned path uses the function's own constant, no d2d twin.

## Deviations and bot findings

None found. No bot finding (the bot was not read in this job).

## Matrix-cell proposal

Think period row: "aidel per difficulty, 0 -> 15; game.exe FUN_005a73e0 re-run over all 735 real rows (verify-aidel.md): V run, 0 differ".

## Shared-file lines for the merger

See openspec/changes/verifyaidel/merge-lines.txt.
