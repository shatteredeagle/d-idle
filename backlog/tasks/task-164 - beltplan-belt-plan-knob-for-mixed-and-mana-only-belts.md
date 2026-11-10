---
id: TASK-164
title: 'beltplan: belt plan knob for mixed and mana-only belts'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-08T03:18:00Z'
updated_date: '2026-11-10T16:54:00Z'
labels:
  - bot
  - feature
  - G-C
milestone: m-4
dependencies:
  - TASK-128
priority: medium
ordinal: 131000
card: "beltplan"
owns:
  - "internal/resource"
  - "internal/town"
  - "internal/env"
  - "internal/decide"
  - "internal/bot"
  - "internal/loot"
  - "docs/belt-plan.md"
  - "docs/knobs.md"
  - "docs/policy-sheet.md"
  - "internal/watch"
  - "cmd/d2bot/resource_test.go"
exclusive: []
contracts: []
result: "Built and tested offline, whole-repo gate green on the worker host. NOT verified live: the live drill was skipped (host busy: 18 game processes, load 7; no free hidden instance claimed). Drill checklist in docs/belt-plan.md. Status sync 2026-11-08 (stays open): live drill skipped."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the policy sheet options 'potion mix' and 'belt fill' only moved drink lines; no path bought an all-mana or mixed-ratio belt, and the user wants players to be able to make suboptimal choices and learn. Done when: resource.beltPlan and resource.beltCol1..4 choose the columns, and the shop, belt fill, loot pickup, kit gate and policy sheet follow them (docs/belt-plan.md).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 belt plan templates and per-column overrides as data (resource.beltPlan, resource.beltCol1..4)
- [x] #2 restock, belt fill, loot pickup and kit gate follow the plan
- [x] #3 policy sheet rows potion mix and belt fill apply through the knob
- [x] #4 table tests for each template, two classes, default regression, unknown template error
- [ ] #5 live drill all mana (or skipped with reason)
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Closed in the 2026-11-10 true-up (docs/research/project-review-2026-11-10.md section 3.1): merged, offline acceptance proven; open live checks moved to docs/live-checks.md (TASK-253).
<!-- SECTION:NOTES:END -->
