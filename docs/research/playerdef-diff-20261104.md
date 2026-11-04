# Player defence differentials: run of section 6 of playerdef-trace.md (job pldefdiff, 2026-11-04)

Scope: the five differentials of `playerdef-trace.md` section 6, against game.exe 1.14d's real code under the emulator. The runner could use the worker host only through `tools/emu/run-on-worker.sh` and read-only `labrun` (ls/grep). Result in one line: **only the existing `pd_calc.py` differential (block, defence, attack rating) was re-run, on two new seeds; the other four were not built in this job** (budget cap of the plan account). No number below is invented; "not run" rows give the ready-to-run recipe.

## 1. What ran (V run)

`tools/emu/run-on-worker.sh --ttl 600 pd_calc.py 100000 SEED` (FUN_00622720 block, FUN_006223f0 defence, FUN_00622560 attack rating executed natively from their first instruction; leaf helpers hooked; the d2d side is a line-for-line **port** of `combat.hpp` / `character.hpp`, not the compiled d2d; see playerdef-trace.md 7b).

| seed | cases | AR mismatches | block raw mismatches | block mismatches after floor at 0 | defence mismatches, positive base | defence mismatches, base <= 0 |
|---|---|---|---|---|---|---|
| 11 (earlier run, 7b) | 102,604 | 0 | 6,371 | 0 | 0 of 102,375 | 159 |
| 11 (this job, repeat) | 102,604 | 0 | 6,371 | 0 | 0 of 102,375 | 159 |
| 23 (new) | 102,604 | 0 | 6,482 | 0 | 0 of 102,399 | 137 |

The result is reproducible and stable across seeds. Block raw mismatches are all dex < 15 (game.exe returns a negative number, d2d floors at 0; a negative chance never wins the roll).

## 2. Per deviation

"Before/after the proposed rule" means d2d port vs the game, and the same after the change; a PR needs before > 0 and after 0 on the compiled d2d.

| id | question | cases run | differ (before) | differ after proposed rule | seeds | verdict |
|---|---|---|---|---|---|---|
| D1 block formula / AR (own dex vs total) | arithmetic of block and AR from stat getter values | 102,604 per seed | 0 AR; 0 block after floor | n/a | 11, 23 | arithmetic **V run equal**. The own-vs-gear question is NOT answered: the stat getter FUN_00625480 is hooked, so whether it returns own or own + gear dex was not exercised. Stays I |
| D2 block recovery (mode 9, FBR branch of FUN_00623f50) | tick count of the block mode | 0 | - | - | - | **not run** (S/U). Base: `hit_len.py` in the skmana read checkout already runs 0x624020.. and FUN_005539b0 for mode 4 (gen/game/compare, 110,836 cases there); needs: the mode-9 dispatch (0x624033/0x624038, block branch, state 0x65 via the 0x639df0 hook), stat 102, AnimData BL records, then compare with `rules::speed_frames` + `ms_per_frame` |
| D3 gear vit/energy quarter points | FUN_00626200 ops 8/9 | 0 | - | - | - | **not run** (S/I). Needs a stat-list setup for FUN_00626200 not yet built |
| D4 player magic resist (stat 37) | FUN_0057be00/0057bf80 with player ctx | 0 | - | - | - | **not run** (S/I) |
| D5 resist clamp order | same harness as D4 | 0 | - | - | - | **not run** (S/I) |
| D6 DR order (stats 34/35/36) | FUN_0057c1e0 record walk | 0 | - | - | - | **not run** (S/U); base `hit_absorb.py` (not present in the checkouts this runner could list) |
| D7 stat 171 twice (defence) | FUN_006223f0 with stat 171 | 102,604 per seed | 0 with positive base | n/a | 11, 23 | game side **V run** (`stat171 + stat16` percent, truncated, applied once to `stat31 + dex/4`); d2d's two-place addition (`panel_stats` plus `update_fighters`) is the same arithmetic only if the aura state sum enters the percent once: the double-count question is a d2d wiring question, not run (compiled d2d needed) |
| D8 per-frame integers (potions) | combat.md rule | 0 | - | - | - | **not run** (S/I) |
| D9 negative defence base (found in the earlier run) | percent moves a base <= 0 toward zero | 102,604 per seed | 159 / 137 | - | 11, 23 | V run (game) + port of d2d; unreachable in play; no PR |

## 3. PR candidates

None. No deviation has a compiled-d2d differential with before > 0 and after 0. D9 is proven against a port only and is unreachable (needs negative base defence). B1..B4 of playerdef-trace.md are bot-side findings and are unchanged by this job (B2's 12-entry bonus list is S from a real dump; B1 the double penalty is V-d2d from docs/d2d-fidelity.md).

## 4. Dead ends

- A loop over several read-checkouts (`for ... in`) is refused by the runner's allow-list; single `ls | grep` calls work. `hit_len.py` is in the skmana read checkout, not in upstream main or the gaps checkout.
- The compiled d2d side (`ms_build.sh`-style hand build) was not attempted, so every "d2d" number above is the port.

## 5. Ready-to-run commands

```
tools/emu/run-on-worker.sh --ttl 600 pd_calc.py 100000 SEED > $D2_SCRATCH/pd.txt          # done: SEED 11, 23
# D2: copy hit_len.py (skmana checkout) to tools/emu/playerdef_blk.py, set unit mode 9, read the block branch of 0x623f50 first
tools/emu/run-on-worker.sh --ttl 1500 playerdef_blk.py gen 100000 7 > $D2_SCRATCH/cases.tsv
tools/emu/run-on-worker.sh --ttl 1500 playerdef_blk.py game $D2_SCRATCH/cases.tsv > $D2_SCRATCH/game.tsv
# D3..D6: boot FUN_00626200 / 0057be00 / 0057c1e0 as in hit_absorb.py, case generator over stats 34-46, difficulty 0..2, 100,000 cases
```

Drill recipe for the live confirmation of each U: playerdef-trace.md section 7 (R1..R7), unchanged.

## 6. Matrix cell

K9 sentence (proposed, unchanged from the trace): "block, defence (positive base) and attack rating: V run 0 differ against a port of d2d's formulas on seeds 11 and 23 (102,604 cases each), `playerdef-diff-20261104.md`". D1 gear dex stays I.
