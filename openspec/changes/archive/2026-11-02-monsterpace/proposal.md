## Why

The second monster AI pass (monsterai2) and the first (monster-ai-trace) proved that game.exe starts a monster at 75 percent velocity and that its Act 1 thinks set a pace d2d leaves out. The owner authorised one bundled upstream PR for the monster movement pace, each after independent Opus verification. This card prepares the candidate on the private fork and the PR draft; it opens nothing.

## What Changes

- Private fork (not this repo): branches `local/monsterpace-a` (13 commits) and `local/monsterpace-b-velocity` (2) on upstream main 3be426d3 (the first single-bundle branches `local/monsterpace-clean` / `local/monsterpace` on 6935cf64 stay for the record).
- docs/research/pr-monsterpace-a-draft.md and pr-monsterpace-b-draft.md: the two PR drafts (think paces, Blood Raven, shaman; base velocity 75) with proofs and limits; they replace the single draft after the verifier's report.
- tools/emu/mp_run.sh, mp_patch.py: the counting replay (evidence tooling; the oracle files are monsterai2's).

## Overlap (declared)

tools/emu/mai_ais.py, mai_*.py and mai2_*.py are monsterai2's (branch agent/monsterai2, owns them): this branch carries identical copies (checked out of that branch unchanged) because mp_run.sh imports them. Same bytes, so the merges agree; whoever merges second drops no change.

## Impact

Docs and evidence tooling here; the code change is upstream-bound and lives on the fork. No spec, policy, ADR or glossary change. Nothing posted.
