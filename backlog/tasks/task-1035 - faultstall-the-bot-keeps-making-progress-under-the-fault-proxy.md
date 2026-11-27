---
id: TASK-1035
title: 'faultstall: the bot keeps making progress under the fault proxy'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-27T06:21:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 500000
card: "faultstall"
lane: next
owns:
  - "internal/faultproxy/"
  - "cmd/d2scenario/faults.json"
  - "docs/probes/faultsoak-design-2026-11-26.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Cause of the 10 minute watchdog stops under the fault sets found from the row logs
- [x] #2 Must-fail offline test reproduces it with a fake single-client d2d
- [x] #3 Fix in the proxy or the bot, tests pass
- [ ] #4 Live soak row under the mixed set logs minute lines (live check)
<!-- AC:END -->
