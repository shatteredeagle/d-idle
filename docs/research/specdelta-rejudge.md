# specdelta: Gate 1 belt and XP rules of 2026-11-03, and the re-judge of the recorded runs

> status: current. Cards: openspec/changes/specdelta (delta spec for soak-gates). Decisions: docs/decisions.md, 2026-11-03.

Tags: V = replayed here from the minute archives with the new `d2dsoakstat` code or read from the run records; U = not verified live (no game involved).

## The rules
- BELT: a window of more than 5 consecutive minute lines outside town with no healing potion counts only if pack plus stash gold reached 30 (Minor Healing price, `PotionPrice`) at some minute of it. Otherwise it is a poverty window: reported, not failing. Why: a clvl 1-2 character with 10 to 16 gold cannot buy a 30 gold potion; the clvl 11 window of gate1-20261103-0338 held 1418 gold in the stash (bot defect, beltfix5) and must still fail.
- Old archives have no `stashGold`: stash counts as 0 and the tag stays "stash unknown (old archive)". They behave as before for windows with pack gold of 30 or more and as poverty below 30 (V test: TestBeltPovertyExcused).
- XP: one seed below the d2d baseline, inside the tolerance, with d2dsoak INCONCLUSIVE, passes with the note "one seed, inconclusive"; below baseline minus tolerance fails; at or above passes. A claim "at or above the baseline" needs the 5-seed d2dsoak verdict. Code: `xp_gate1` in tools/gate-judge.py (ops repo).

## Re-judge (V: archives docs/probes/soak-logs/gate1-*-minutes.jsonl.gz replayed; criteria other than BELT and XP unchanged from the run records)

| run | clvl and window | pack / stash gold in the window | BELT old | BELT new | XP (vs 9172, tol 1834) | XP old | XP new | verdict old | verdict new |
|---|---|---|---|---|---|---|---|---|---|
| 20261102-1939 | clvl 12, 12 min | pack 30..38, stash unknown | FAIL | FAIL (funded) | 10042 (+9.5%) | PASS | PASS | NOT PASSED | NOT PASSED (belt) |
| 20261102-2210 | clvl 9, 37 min | pack 332..366, stash unknown | FAIL | FAIL (funded) | 9754 (+6.4%) | PASS | PASS | NOT PASSED | NOT PASSED (belt, and NOKILL 11 > 10) |
| 20261103-0053 | clvl 2, 7 min | pack 10..16, stash unknown | FAIL | PASS (poverty) | 8162 (-11.0%) | FAIL | PASS (one seed, inconclusive) | NOT PASSED | PASSED (all other criteria PASS in the record) |
| 20261103-0338 | clvl 2, 7 min; clvl 11, 7 min | clvl 2: pack 10..16; clvl 11: pack 3..11, stash 1418 (beltfix5 section 5; the archive predates stashGold, a replay alone would read the clvl 11 window as poverty) | FAIL | FAIL (clvl 11 window funded; the clvl 2 window is poverty) | 8751 (-4.6%) | FAIL | PASS (one seed, inconclusive) | NOT PASSED | NOT PASSED (belt) |

Earlier runs, for completeness (V replay): 20261102 (belt 4, PASS) and 20261101 (belt 3) are unchanged by the belt rule.

Reading: the new rules turn exactly one record, 0053, into a pass; the two runs where the bot could have bought a potion (1939, 2210) and the stash defect (0338 clvl 11) still fail. The fifth run (gate1-20261103-0628) is judged by the lead.

## Not verified
Nothing here ran against a live game. The 0338 clvl 11 stash figure is from beltfix5 (V there: logs/bot-0.log of the run), not from the archive; once a run carries `stashGold` the replay needs no outside number.
