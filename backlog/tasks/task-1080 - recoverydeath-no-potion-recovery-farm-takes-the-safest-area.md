---
id: TASK-1080
title: 'recoverydeath: no-potion recovery farm takes the safest area'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-27T23:27:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 517000
card: "recoverydeath"
lane: next
owns:
  - "internal/bot/broke.go"
  - "internal/bot/povertyfarm.go"
  - "internal/bot/recoverydeath_test.go"
  - "internal/bot/invariants.go"
  - "internal/admit/rows.go"
  - "internal/plandata/admission.json"
exclusive: []
contracts: []
result: "Recovery farm with no healing potion limited to monster level <= clvl-3 (admission.json poverty.recoveryFromLevel, data, default-changing); field-entry-no-kit detail carries gold. Offline only, not verified live. Problem 5 cause not reproduced."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A recovering hero with no healing potion farms only areas at monster level <= clvl minus poverty.noPotionGap (first field when none); two builds, clvl 4 and 9/12
- [x] #2 field-entry-no-kit incident detail carries the gold
<!-- AC:END -->
