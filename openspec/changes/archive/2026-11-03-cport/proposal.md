# cport

## Why

Batch A (mana floor, Teleport) is staged as the one open upstream batch. The maintainer's rule is to add further commits of the same subsystem to that open pull request instead of opening another. SORCDIFF2 produced two private fix branches in the same subsystem (player skill casting): mana paid on the action frame (with the action-frame tick) and the armor state group. This card prepares them as commits on top of batch A and writes their rows for the pull request body.

## What Changes

- `docs/research/pr-batch2-C-rows.md`: the Claims-table rows for the added commits, how each conflict with batch A was resolved, counts on the exact head, honest limits, size against the CodeRabbit budget.
- Private fork (not in this repo): branch `local/batch2-C-r1` (batch A head plus three commits), pushed to the private remote only.

## Impact

One doc. Nothing pushed to GitHub, no PR, no comment.

Overlap (declared): reads sorcdiff2 (docs/research/sorcdiff2.md), pr-batch2-A-body.md and pr-upstream-rules-20261103.md; edits none of their files.
