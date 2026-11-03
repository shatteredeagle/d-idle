## Why

The owner decided (2026-11-03) that the upstream batch 2 pull requests follow every maintainer rule, and that our emulator oracle scripts go into the pull requests (the maintainer's pr_guard fails a PR that cites a `tools/` script not in the tree; AGENTS.md "Evidence": the oracle goes in the same pull request). The mana, Teleport and monster damage differentials ran on harness files that live in no tree.

## What Changes

- On the private fork only (the private remote, no push elsewhere): `local/batch2-A-r4` (mana floor, Teleport room rule, Teleport nearest spot, each commit with its oracle, test, README row and structs.tsv rows) and `local/batch2-B-r1` (the monster damage roll with `tools/emu/monster_damage.py`, its test and the d2d side as a `test_combat --monster-damage` mode).
- In this repo: docs/research/pr-batch2-A-body.md and pr-batch2-B-body.md (the PR bodies to the maintainer's template: Claims table, evidence, re-check commands, limits, CodeRabbit budget), the maintainer rule list, this card.
- Nothing is pushed to GitHub or posted; the coordinator opens the PRs after the user's go.

## Impact

Docs here; the fork branches are the deliverable. Overlap (declared): the earlier batch 2 body drafts (pr-batch2-body.md, pr-batch2-body-split.md) stay as history; these two files replace them for the posting.

credit: see tasks.md.
