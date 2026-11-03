# crfixbundle

## Why
CodeRabbit reviewed the fork's staging PR 2 (stage/base-2 <- stage/bundle-2) with 6 inline comments and a summary. Two were functional (the charge-repair price, the ground-roll comparison); the rest were confidence tags and test notes the fork's review rules ask for. Each needed a verdict with evidence before the staging branch is pushed again.

## What Changes
- Private branches `local/bundle-3` and `stage/bundle-3` on the fork's private remote: the ten bundle commits with the valid fixes folded into the commit each belongs to (bundle 2 untouched, no force-push).
- `docs/research/crfix-bundle-20261102.md`: verdict, action and evidence per comment, the reply texts for the coordinator, and a throwaway merge with upstream PR 176.
- Harness for the repair base part against game.exe in `docs/research/crfix-bundle-20261102/`; the PR body draft updated where a claim changed.

Exclusive: fork-branch:local/bundle-3, fork-branch:stage/bundle-3. Nothing is pushed to the public fork or posted anywhere.
