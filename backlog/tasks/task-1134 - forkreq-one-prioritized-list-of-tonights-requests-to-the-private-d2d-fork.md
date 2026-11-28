---
id: TASK-1134
title: 'forkreq: one prioritized list of tonight''s requests to the private d2d fork'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-28T10:40:00Z'
labels:
  - docs
  - research
dependencies: []
priority: medium
ordinal: 538000
card: "forkreq"
lane: next
owns:
  - "docs/design/fork-requests-2026-11-28.md"
exclusive: []
contracts: []
result: "Docs only: docs/design/fork-requests-2026-11-28.md orders the fork requests of 2026-11-28 (save safety first, then the branch decision, Tyrael's door, order ids, game-side write count, differentials, nine read-only questions, the switch last). No fork source was read, nothing re-run, nothing verified live; sizes and risks are guesses (U). Nothing build-specific."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: the night of 2026-11-28 left fork requests in five places (save safety after a lost hero, order ids, Tyrael's door, review findings, the rebase round); the owner needs one ordered list for a fork session. Consumer: the owner's fork session and the cards durieltyrael (TASK-1126), savesafe, vault2. Done when the criteria hold.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 docs/design/fork-requests-2026-11-28.md lists every fork request of 2026-11-28 with what, why, lab acceptance, risk, size and order, with a Do first section
- [x] #2 leakcheck, contracts check and backlog check pass
<!-- AC:END -->
