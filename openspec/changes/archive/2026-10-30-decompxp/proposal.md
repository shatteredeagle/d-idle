## Why

d2d pays +60 % XP per kill in the Den of Evil and +20 % in the Blood Moor against the live original at clvl 4 to 6 (docs/d2d-fidelity.md check 4b), unexplained; XP/h is the gated fidelity metric. Trace the original's XP award in game.exe 1.14d (offline, the unicorn oracle), compare with d2d's `kill_exp`, and explain the two gaps.

## What Changes

- docs/research/xp-award-trace.md: the award path with addresses, the level-difference tables, the per-monster numbers and a ranked explanation of the gaps
- tools/emu/xp_*.py: the oracle that runs the original's level-difference function under unicorn
- docs/research/upstream-pr-candidates-xp.md: a draft PR body if d2d has a real bug (nothing posted)
