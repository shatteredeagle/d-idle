---
id: TASK-963
title: >-
  outcome-kit-gate: convert kit gate holds and town chores refusals to typed
  outcomes
status: Done
assignee:
  - credit-runner
created_date: '2026-11-26T03:42:00Z'
updated_date: '2026-11-27T02:03:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: low
ordinal: 460000
card: "outcome-kit-gate"
owns:
  - "docs/outcomes.md"
exclusive: []
contracts: []
result: "Kit gate holds and town chore refusals are typed holds: one line per change with need and wake fact, no pause or failure count. Offline tests on two classes, not verified live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Size: S. Follow-up of outcome1 (docs/outcomes.md, not yet converted list). Needs outcome1 merged.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 path converted behind bot.typedOutcome, row added to docs/outcomes.md, must-fail test on two builds
<!-- AC:END -->
