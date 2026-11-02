## Why

The maintainer merged the batch PR 174 into upstream main. The four private PR candidates (freeze rules, vendor repair gate, melee fine physical D1, melee undead / Blunt D2) were rebased on the old base. The user asked for a re-rebase of every candidate onto the new main with the counts re-run. Upstream PRs stay on hold: nothing is pushed to a public remote, nothing is posted.

## What Changes

- New private fork branches (private remote only): `local/freeze-rules-v5`, `local/vendor-repair-gate-v4`, `local/skmelee3-d1-r174`, `local/skmelee3-d2-r174`, each on the new upstream main, one purpose per commit, numeric noreply identity for author and committer.
- `docs/research/rebase174-20261102.md`: per candidate old and new head and base, conflicts and resolution, base versus head counts on the new main, CI parity, verdict.
- Base-commit lines of the PR drafts `pr-freeze-rules-draft.md` and `pr-vendor-repair-gate-draft.md` updated (the two skmelee3 drafts are updated on the branch of card prbuildskmelee3b).
- No Go code, no game, no PR opened.

## Overlap (declared)

`docs/research/pr-freeze-rules-draft.md` and `pr-vendor-repair-gate-draft.md` were owned by the archived cards prbuildfreeze and prbuildprices2; only their base-commit lines change here. The skmelee3 drafts belong to card prbuildskmelee3b (unmerged); the same base-line edit is committed on that branch.
