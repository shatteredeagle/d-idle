# Verify SKMANA: independent re-check of the mana cost check and payment

> status: current (card verifyskmana). Verifier: a second agent, own harness `tools/emu/vsm_diff.py` (and `tools/emu/vsm_dis.py` as a reading aid). I read the two functions and wrote the oracle BEFORE opening `docs/research/sk-mana-trace.md` or `tools/emu/sk_mana.py`. No game files, no disassembly pasted.

## Result

| claim (sk-mana-trace) | verdict | tag |
|---|---|---|
| C1 check `FUN_0056c160`: stat 8 >= `(mana + lvlmana x max(lvl-1,0)) << manashift`, no minmana floor | **HOLDS** | V run (20,000 cases, 0 mismatches) + V read |
| C2 pay `FUN_0056bfe0`: `max(formula, minmana x 256)` | **HOLDS** | V run (20,000 cases, 0 mismatches; floor binds in 12,211) |
| C3 pay with stat 8 below the cost: returns 0, takes nothing | **HOLDS** | V run (0 mismatches on result and on the stat afterwards) |
| C4 the do ignores a refused payment | **NOT TESTED**: lives in the callers (do functions), outside these two functions | U |
| C5 (side claim) mana 0 and lvlmana 0: pay returns 0, pays nothing | **HOLDS** | V run (561 cases) |
| C6 (side claim) srvdofunc 0x74 on a shapeshifted unit: check returns 1 | **HOLDS** | V run (4,985 cases with row id 0x74, half shifted) |

## How (own reading)

- Check: unit type must be a player (else 1); current skill struct from `FUN_00620250` (none: 0); an item-charge entry (`FUN_00643ad0` != -1) tests charges left > 0; else row `FUN_00644140`, level `FUN_006442a0(unit, entry, 1) - 1` floored at 0, cost as above with signed 16-bit mana and lvlmana and a byte shift; row id 0x74 plus `FUN_0063a400` true returns 1; then `FUN_00625480(unit, 8)` >= cost. The minmana field (+0x186) is never read in this function.
- Pay: same player gate; item-charge path to `FUN_0056bec0`; row from the table at `[0x744304] + 0xb98` (stride 0x23c, count at +0xba0); zero mana and zero lvlmana returns 0; level argument is used directly (no `FUN_006442a0`); `max(cost, minmana << 8)`; state 0x72 (`FUN_00639df0`) pays through `FUN_005d2b60`; else stat 8 below cost returns 0 untouched, else `FUN_006272b0(stat 8, -cost)` and success.

## Harness and reproduction

`tools/emu/vsm_diff.py N SEED` runs the real code of both functions in the emulator over a fake Skills row, a fake player unit and a stat 8 hook; the oracle is a formula written in the script from my reading (not from sk_mana.py). Hooked, not the claim: `FUN_00620250`, `FUN_00643AD0`, `FUN_00643CE0`, `FUN_00644140`, `FUN_006442A0`, `FUN_0063A400`, `FUN_00639DF0`, `FUN_00625480`, `FUN_006272B0` (recorded). Inputs: shift in {0,1,2,3,5,8}, mana -40..200, lvlmana -8..40, minmana 0..100, level 0..60, stat 8 placed at cost and floor -1/0/+1 and random.

    tools/emu/run-on-worker.sh --ttl 600 vsm_diff.py 20000 20261102 > out.txt 2>&1

Run: `cases 20000 seed 20261102 mismatch {check 0, pay 0, paystat 0}`; counts: check true 14,704, pay true 7,311, zero-cost rows 561, floor binds 12,211, row 0x74 4,985. A first version of the harness reported 7,311 pay mismatches: my bug (skill id 54 outside my 8-row table), fixed, not a game finding.

## Differences from the author's numbers

None in the verdicts. The author's run is broader (every one of the 357 Skills rows x levels 1..60, 130,840 cases) and covers the d2d side; mine is 20,000 random-plus-edge cases on game.exe only, with synthetic rows. Not a conflict.

## Limits I did not cover

- C4 (callers ignoring a refused pay) and the start-function ordering: not run here.
- Not run: non-player units (return 1, read only), the item-charge path (hooked off), the blood_mana path (state 0x72 hooked off), the client check `FUN_00647540`, the real Skills.txt rows (rows are synthetic, so extreme real values such as shift 8 with large mana are covered only by the random range), d2d's `mana_cost` (not compared; this note is about game.exe).
- Deviations for d2d and bot findings: none new; see sk-mana-trace D2 (floor in the check) for d2d, B-list there for the bot.
