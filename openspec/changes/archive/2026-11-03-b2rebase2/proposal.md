# b2rebase2

## Why

Upstream main moved to `005c7cab` (PRs 179 and 180, then the maintainer's AGENTS.md, CONTRIBUTING.md and PR template change `549da358`: code changes batch too, one change per commit, a "Commit" column in the Claims table, `presubmit --quick` runs GCC and clang-tidy). The batch 2 bundle must be carried onto that main, the new rules read, the PR body moved to the new template, and every check and counted differential re-run so the body quotes exact numbers.

## What Changes

- Private fork (not in this repo): new branches `local/batch2-bundle-r3` (the four bundle commits on `005c7cab`), `local/batch2-mana-teleport` (commits 1 to 3) and `local/batch2-monsterdmg-pr` (commit 4), all on upstream main, pushed to the private remote only. Old branches stay.
- `docs/research/pr-batch2-body.md` (new template, new hashes and counts, budget), `docs/research/pr-batch2-body-split.md` (the two split bodies), `pr-batch2-run.txt` (raw results), `docs/research/pr-upstream-rules-20261103.md` (the new upstream rules, tagged).
- `tools/emu/b2r3_*.sh`: the b2r scripts with the base, branch and scratch names changed.

## Impact

Docs, scripts and card headers only. Nothing pushed to GitHub, no PR opened.

Overlap (declared): rewrites the b2rebase docs (card archived) and copies its scripts; it edits no other card's files.
