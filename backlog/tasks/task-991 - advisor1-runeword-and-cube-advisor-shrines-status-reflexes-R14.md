---
id: TASK-991
title: 'advisor1: runeword and cube advisor, shrines, status reflexes (R14)'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-26T03:59:00Z'
updated_date: '2026-11-27T20:31:00Z'
labels:
  - bot
  - feature
dependencies:
  - TASK-984
priority: low
ordinal: 478000
card: "advisor1"
owns:
  - "docs/design/advisor-r14.md"
  - "internal/advisor/"
  - "internal/watch/advisorsection.go"
  - "internal/watch/funwire.go"
  - "internal/watch/returnscreen_test.go"
  - "internal/cubejobs/cubejobs.go"
  - "internal/env/tuning_items.go"
  - "docs/knobs.md"
  - "data/darklaunch.json"
exclusive: []
contracts: []
result: "Built dark, offline only (not verified live): internal/advisor (runeword table as data, matcher, cube path, class relevance, line formatter, vault adapter), watch.AdviceSection, knob items.advisor (default 0) with ledger row. Bot-side stock gathering and the SME review are open; shrines and status reflexes not in this slice."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Recommendation R14 of the 2026-11-19 design review (06-roadmap): data and advice, the player acts (runeword and cube advisor, shrines, monster aura and curse status reflexes). No card existed. Cause class: none of A to K (feature). Depends on agencyp1b. Risk: low. Enable criteria: advisor lines on the Return screen reviewed by the d2-sme agent; no behaviour change without an owner go.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Advisor lines for runewords and the cube behind a knob with a ledger row
<!-- AC:END -->
