## Why

The fidelity verdict is the number every M2 card is judged by, and docs/research/code-review-2026-10-30.md (H1) found that a d2dsoak verdict can exit 0 on INCONCLUSIVE or FAIL and that a seed with no minute lines or a d2d that died early is counted as clean exposure for the death metrics. Until the tool is fail-closed, a PASS cannot be trusted and a FAIL can be missed by a script.

## What Changes

- cmd/d2dsoak: exit 0 only on PASS, 2 on INCONCLUSIVE, 3 on FAIL and NO-DATA, for `run` and `compare` alike; the verdict line names the code.
- cmd/d2dsoak: a seed with zero minute lines, or whose d2d exited before the wall limit, is excluded from the exposure hours and listed under `excluded seeds` in the report; the gate's minimum-seed rule counts only included seeds.
- docs/d2d-soak.md: the exit-code table and the exclusion rule.
- Delta: fidelity-gate ADDED `Verdict exit codes` and `Excluded seeds`.
