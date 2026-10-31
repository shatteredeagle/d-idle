## Why

xpgap (2026-10-31) A/B-tested three knobs with 4 seeds x 2 virtual hours per arm and found that none beats noise: the standard error is 7 to 9 percent of the mean, only effects of 15 percent or more are visible, and about 35 runs per arm would be needed for a 5 percent effect. Every M2 card claims an effect from such a run, so the protocol (pairing by seed, runs per arm, hours per seed, the statistic) must be fixed once, with a tool, before the next claim; the lockstep fork is deterministic per seed, so paired differences are available and were not used.

## What Changes

- tools/ab/: a paired same-seed before/after analysis (`an.py` style, stdlib Python): per-seed differences of XP/h and kills/h, the paired mean and its 95 percent interval, the runs per arm needed for a 10 percent effect from the measured SD; a runner script that interleaves arms on the slot pool (reuses tools/xpgap/ab.sh if it merged).
- docs/ab-design.md: the protocol (paired seeds, at least 8 seeds per arm, 2 virtual hours, an effect is claimed only when the paired interval excludes 0), applied to xpgap's 24 runs in docs/probes/xpgap as the worked example.
- Delta: fidelity-gate ADDED `A/B protocol for bot changes`.
