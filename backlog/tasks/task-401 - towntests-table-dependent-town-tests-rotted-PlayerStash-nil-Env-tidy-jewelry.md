---
id: TASK-401
title: 'towntests: table-dependent town tests rotted (PlayerStash nil Env, tidy, jewelry)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-14T10:39:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 249000
card: "towntests"
owns:
  - "docs/towntests.md"
  - "internal/town/playerstash_test.go"
  - "internal/town/tidy_test.go"
  - "internal/town/invstash_test.go"
exclusive: []
contracts: []
result: three stale town tests fixed; no product bug; verified with tables on the worker host, not live
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Failing table-dependent tests found and fixed
- [x] #2 go-tables gate proposal written in docs/towntests.md
<!-- AC:END -->
