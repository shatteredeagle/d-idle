---
id: TASK-116
title: 'dietcards: stop committing docs/cards.md (generate on demand or in CI; fix the mergeq preflight and the cards_md --check use)'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-07T21:57:00Z'
updated_date: '2026-11-07T22:18:00Z'
labels:
  - ops
  - chore
milestone: m-4
dependencies: []
priority: medium
ordinal: 109000
card: "dietcards"
owns:
  - "tools/backlog/cards_md.py"
  - "tools/ci/check.sh"
  - "docs/backlog-howto.md"
exclusive: []
contracts: []
result: "docs/cards.md untracked + gitignored; cards_md.py --write builds it whole, --check only runs the generator; check.sh, howto, ops auto_done/mergeq/merging.md updated; offline, not verified live (no game needed)"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: user decision D5 (board and docs diet), implementation is this separate card.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 docs/cards.md is generated on demand or in CI and no longer committed; remove it from the repo and add a stub or ignore
- [x] #2 mergeq preflight and every cards_md --check call site updated; no cards.md merge conflicts left
- [x] #3 tools/ci/check.sh passes
<!-- AC:END -->
