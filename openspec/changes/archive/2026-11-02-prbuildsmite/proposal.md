## Why

The independent verification (card verifysmite, docs/research/verify-smiteblock.md, 20,000 of 20,000 Smite hits deal damage in game.exe 1.14d incl. 4,292 blocked ones) shows d2d ends a Smite on a block while the game forces the hit bit after the block check (0x5cebb0). This card builds the smallest correct upstream candidate on the private fork, counts it against game.exe and writes the PR draft; nothing is posted.

## What Changes

- Private fork branch `local/smite-block-v1-final` (one commit on upstream main 82ea6db4; also `local/smite-block-v1`): `player_blow` lets a blocked Smite continue, `Fight::land` ends the blow on `blocked && !hit`, tests in `tests/test_combat.cpp`.
- tools/emu/sbp_probe.cpp, sbp_run.sh, sbp_ci.sh: the d2d-side probe, the differential runner and the CI parity script (worker host).
- docs/research/pr-smite-block-draft.md: PR text, counts, animation decision (a stated limit), credits.

## Impact

Docs and emulator scripts in this repo; the code lives only on the private fork remote (GPL, never in this repo). No spec, ADR, glossary or decision change.

Overlap (declared): none of the owned paths belongs to another open card; the archived cards smiteblock and verifysmite own the harness this one reuses (vs_oracle.py, sb_*), read only.
