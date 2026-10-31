## Why

The upstream maintainer asked (2026-10-31, relayed by the user) for several bits bundled into one larger PR: fewer wasted review resources and the agent can work longer. Our rules of one claim per PR and at least one hour between PRs were written for a different situation (CodeRabbit limits, a cautious start) and now cost time. The claim stays the unit of proof; the PR becomes the unit of review.

## What Changes

- docs/process/upstream-pr-standard.md: one claim per commit; a PR may bundle several proven claims when the maintainer prefers it; each claim keeps its own differential, verifier verdict, ponytails and commit; the one-hour spacing rule becomes at most one open bundle at a time.
- docs/research/pr-bundle-draft.md and pr-bundle-evidence-pack.md: the first bundle (renderer get-hit frames + Lut Gholein tool fix), prepared, not opened.
- docs/BACKLOG.md and docs/research/README.md: the policy line and the index entries.
- Delta: upstream-pr MODIFIED `One claim per PR` and `PR spacing and bundled follow-ups`.
