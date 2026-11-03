## Why

Tier B design (docs/research/tierb-design-20261102.md section 3 and 4.1) answers "is d2d within 15 percent of game.exe" statistically. The existing gate treats the baseline mean as exact; an A:B needs both arms' error, k regimes kept apart, and a pair correlation.

## What Changes

- tools/baseline/ab_compare.py: per-metric d2d/game.exe ratio, 95 % interval (log delta method, Welch df), noise band, verdict (EQUIVALENT / DIFFERENT / INCONCLUSIVE) against a margin (default 15 %), exact Poisson interval for deaths and drops, area time share, pair correlation, n per arm to settle; one table per k.
- tools/baseline/tests/test_ab_compare.py: synthetic sets (equal arms, 10 % gap, high variance, mixed k, weak lanes).
- Delta spec for fidelity-gate (specs/fidelity-gate/spec.md), proposed; openspec/specs is not edited here.
- Offline only; record format is defined in the script docstring (S: the ops-repo baseline record layout was not readable, so a thin JSON adapter may be needed). Not run on kitb/kitk2 data (U).
