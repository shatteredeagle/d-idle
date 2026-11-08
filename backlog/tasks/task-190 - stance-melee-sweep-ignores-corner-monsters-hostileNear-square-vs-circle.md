---
id: TASK-190
title: 'stancefix: melee sweep ignores corner monsters (hostileNear square vs circle)'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-08T15:22:00Z'
updated_date: '2026-11-08T16:01:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: high
ordinal: 154000
card: "stancefix"
owns:
  - "internal/bot/observe.go"
  - "internal/bot/friendly_test.go"
exclusive: []
contracts: []
result: "Fixed hostileNear (square vs circle) so a corner monster no longer starts the futile-clear window; regression test added. Log evidence in docs/stance.md. Offline only, NOT verified live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Part of the 2026-11-08 feedback (Paladin does not fight). See docs/stance.md.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 hostileNear uses the fighter's circular scope; regression test
<!-- AC:END -->
