## Why

PR 181 (open, maintainer's ponytail batch 11) conflicts with batch 2 in components/rules/combat.hpp. Rebase the four batch 2 commits onto its head now in the private fork, so the rebase is quick when it merges.

## What Changes

- `docs/research/b2-on181.md`: conflict resolutions with their semantics, CI and counted-differential results on that tree, and the mechanical redo once 181 merges.
- Private fork branch `local/batch2-on181` (and `local/pr181`, the read-only copy of 181's head), pushed to the private remote only.

## Impact

Docs only in this repo. Overlap (declared): reads the b2bundle and dmgroll cards' results; changes none of them.
