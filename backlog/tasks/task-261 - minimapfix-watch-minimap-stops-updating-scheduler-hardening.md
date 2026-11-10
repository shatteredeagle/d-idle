---
id: TASK-261
title: 'minimapfix: /watch minimap stops updating, scheduler hardening'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-10T17:34:00Z'
labels:
  - watch
  - bug
dependencies: []
priority: medium
ordinal: 195000
card: "minimapfix"
owns:
  - "internal/watch/web/shell.js"
  - "internal/watch/minimapfix_test.go"
exclusive: []
contracts: []
result: "scheduler hardened (fetch abort 8 s, safe catch, safe tick); root cause not reproduced offline (headless render dpr 1 and 2 updates fine); not verified live"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: user reports the minimap stale. Done when: a stalled or throwing panel cannot freeze polling.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 shell scheduler cannot wedge a panel (abort, safe catch, safe tick)
<!-- AC:END -->
