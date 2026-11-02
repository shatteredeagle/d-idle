## Why

Candidate 3 of docs/research/pr-candidates-20261102.md (SKMELEE3): the player's melee blow keeps its 256ths (D1) and applies the undead / Blunt bonus (D2) in game.exe, not in upstream d2decomped. The user's go of 2026-11-02 allows up to five more upstream PRs, each after an independent verification; this card builds the candidate in the private fork and drafts the PR text. Nothing is posted.

## What Changes

- Private fork branch `local/skmelee3-v1` (from upstream main): two commits (D1 fine physical, D2 undead / Blunt), unit tests in tests/test_combat.cpp. Pushed to the private remote only.
- `tools/emu/sk_melee2.py` and `sk_melee2_test.cpp`: the oracle harness now drives the new inputs and prints the build's physical in 256ths, so base and head can be counted on the same cases (U cases scaled by an optional third `gen` argument).
- `tools/emu/sm3_probe.py`: reads the two MonStats undead masks (V run).
- `docs/research/pr-skmelee3-draft.md`: the PR text (counted differentials against the original game, limits, det-scenario impact).
- No Go code, no game, no PR opened.

## Overlap (declared)

None of the open upstream PRs (159, 160, 163) touch tests/test_combat.cpp's player-blow block or `player_blow`; see the draft for the file list.
