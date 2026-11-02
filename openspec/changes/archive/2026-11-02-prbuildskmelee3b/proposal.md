## Why

The single SKMELEE3 candidate (card prbuildskmelee3) was verified by an independent Opus run (card verifyskmelee3): READY WITH CHANGES, split in two. This card rebuilds it as two independent upstream-ready branches on the current upstream main and applies the verifier's findings. Upstream PRs are on hold: nothing is posted or pushed to GitHub; the branches live on the private fork remote only.

## What Changes

- Private fork branches `local/skmelee3-d1` (fine physical, no one-point floor, missile weapon-share test, corrected `Blow::frac` comment) and `local/skmelee3-d2` (undead / Blunt bonus with the corrected gate: no `not_player`), each one commit on upstream main, independent of each other.
- Differential, suites, CI parity and determinism results on each head, recorded in the two drafts `docs/research/pr-skmelee3-d1-draft.md` and `pr-skmelee3-d2-draft.md`, including a throwaway merge with the maintainer's open batch PR 174 and the prepared resolution.
- The oracle harness files (`tools/emu/sk_melee2*`, `sm3_*`) are carried here from the unmerged prbuildskmelee3 branch so the counts can be re-run.
- No Go code, no game, no PR opened.

## Overlap (declared)

Supersedes card prbuildskmelee3 (its draft and branch `local/skmelee3-v1`); the harness files it owns are carried here unchanged except as noted in the drafts. PR 174 (upstream) touches `player_blow`, `Target`, `ai.hpp` and the missile path: see the drafts.
