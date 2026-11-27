---
id: TASK-960
title: 'outcome-park-step: convert parkStep and stale-target holds to typed outcomes'
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
ordinal: 458000
card: "outcome-park-step"
owns:
  - "docs/outcomes.md"
exclusive: []
contracts: []
result: "The quest step park and the blocked journey holds are typed refusals handled once (one line per change, pause by need) instead of the spin guard. Offline tests on two classes, not verified live."
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
