## Why

The PALSKILL D2/D3 fork candidate (Blessed Hammer: the caster's Concentration share of its magic damage, and the half-roll bonus against the undead and the demons) claims 0 differing cases against game.exe on the author's harness. An independent verifier run is required before the PR is opened.

## What Changes

- docs/research/palverify/: my own harness (game.exe oracle `hammer_game.py`, d2d probe `test_palprobe.cpp` dropped into scratch copies only, build and run scripts), the run results and the verdict doc `verdict.md`.
- No change to the code under test, to the fork, or to any Go code. Nothing pushed or posted.

## Impact

Docs only. Decides whether the candidate goes into the upstream PR as is.
