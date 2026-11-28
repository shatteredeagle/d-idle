---
id: TASK-1093
title: 'pgfix: legacy scenarios banked by default, ledger tag for blocked runs'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-28T03:00:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 523000
card: "pgfix"
lane: next
owns:
  - "cmd/d2scenario"
  - "internal/scenario"
  - "tools/provingground/scenarios"
  - "docs/process/proving-ground.md"
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
- [x] #1 Scenarios without pre.hero_profile are banked; played is opt-in
- [x] #2 A run blocked before setup still gets its ledger setup tag
<!-- AC:END -->
