## Why

Code review 2026-10-30 (docs/research/code-review-2026-10-30.md section 2): d2dsoak exited 0 on INCONCLUSIVE and, with `-json`, on FAIL; crashed seeds counted as clean 0-death exposure; minimum sample only warned.

## What Changes

- exit codes PASS 0, error 1, INCONCLUSIVE 2, FAIL 3, usage 64, `-exit-zero` escape hatch (docs/d2d-soak.md 3c)
- invalid runs (short, crashed, early-ended, unparseable) excluded and the verdict capped at INCONCLUSIVE
- min sample, NaN/negative, duplicate seeds, parse-failure counting, repeatSites cap
