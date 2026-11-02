## Why

Upstream PRs #159 and #160 show merge conflicts and #163 is behind main. The maintainer's rule (AGENTS.md section 6) is to keep a branch current with `git rebase origin/main`, which rewrites the PR branch and so needs a force-push. Standing rule: no force-push without the user's go.

## What Changes

- Rebased copies of the three PR branches on upstream main (4bf911b3) under new local branch names, verified, pushed only to the private fork remote. Nothing is pushed to the public fork or posted.
- Log entry in docs/research/openpr-log.md.

## Impact

docs/research/openpr-log.md (one appended section), this card. No spec delta.
