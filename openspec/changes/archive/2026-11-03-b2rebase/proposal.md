# b2rebase

## Why

The maintainer's batch 10 (upstream PR 178) merged as `552acc9a`. The batch 2 bundle (`local/batch2-bundle`, on `3e073888`) must be carried onto that main, with every CI check and the counted differentials re-run on the new head, so the PR body quotes exact numbers.

## What Changes

- Private fork (not in this repo): new branch `local/batch2-bundle-r2` on upstream main `552acc9a` (the four bundle commits cherry-picked in order; the old branches stay), pushed to the private remote only.
- `docs/research/pr-batch2-body.md` and `pr-batch2-run.txt`: new base and head, counts, parity table, CodeRabbit budget table.
- `tools/emu/b2r_*.sh`: the b2b_* scripts with the base, branch and scratch names changed (the PR 178 merge-tree step dropped).

## Impact

Docs, scripts and card headers only. Nothing pushed to GitHub, no PR opened.

Overlap (declared): rewrites the two b2bundle docs (card b2bundle, archived) and copies its scripts; it edits no other card's files.
