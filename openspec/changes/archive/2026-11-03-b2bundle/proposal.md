# b2bundle

## Why

Batch 2 for the upstream d2decomped PR (minmana floor, Teleport room limit and nearest free spot, monster damage roll) is verified commit by commit. One bundle PR needs one head to quote: a branch on the newest upstream main with the four commits in order, every CI check and the counted differentials re-run on exactly that head, and a PR body in the format of `docs/research/pr-bundle-final-body.md`.

## What Changes

- Private fork (not in this repo): `local/batch2-monsterdmg-r5` (r4 with the two text conditions of the fourth verifier applied: the `// ponytail:` note in `states.cpp`, one commit) and `local/batch2-bundle` (minmana, Teleport D1, Teleport D2, damage roll on upstream main `3e073888`), both on the private remote only.
- `docs/research/pr-batch2-body.md`: the PR title and body with counts for the bundle head, limits, the ponytail list, the CodeRabbit budget table and the Credits section. Not posted.

## Impact

Docs and card headers only. Nothing pushed to GitHub, no PR opened.

Overlap (declared): reads the cards prbuildb2 (minmana, Teleport), dmgroll (damage roll) and verifyb2 / verifydmg4 (verdicts); it edits none of their files.
