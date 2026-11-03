## Why

Staging batch A on the fork (fork PR 3) showed that the maintainer's guard workflow runs `tools/pr_guard.py` from the base checkout, so any pull request that adds a `tools/emu` oracle and cites it fails "isn't in the tree", although his rule says the oracle goes in the same PR. The lead decided the fix goes upstream as its own one-claim PR, and batch A follows it as the batch PR (decision of 2026-11-03).

## What Changes

- Private fork branch `local/guardfix-3`: one commit, the cited-tool check reads the PR head tree with git instead of the base working tree, plus two tests.
- Private fork branch `local/batch2-A-r9`: batch A without the guard commit (r6 minus its first commit, on upstream main 9785632c), CodeRabbit's real findings folded in.
- Two upstream PR texts as files: `docs/research/pr-guardfix-body.md`, `docs/research/pr-batch2-A-final-body.md`. Nothing is posted.

## Overlap (declared)

Supersedes the staging card `stagea` (r6 carried the guard fix as its first commit) and the earlier `oracleport` body for r5; no shared owned paths.
