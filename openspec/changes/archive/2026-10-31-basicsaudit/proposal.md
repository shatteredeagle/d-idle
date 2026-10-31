## Why

The next milestone (docs/BACKLOG.md section 0) is rock-solid town and inventory basics on d2d. The d2-sme listed 15 scenarios plus extra basics; nobody has yet checked which of them the code, the offline tests and the d2d logs already cover. This offline audit does that, so the follow-up work targets real gaps.

## What Changes

- `docs/basics-audit.md`: a coverage table (implementation, behaviour against the SME's end state, offline test, live proof, failure latch, verdict and severity), the ranked gaps, the unknowns and the proposed hidden-d2d live checks.
- `internal/town/basicsaudit_test.go`: offline scenario tests on the sim rig; matching behaviour is asserted, differing behaviour is pinned and named as a gap.
- No production code changes; no game, no d2d slot.
