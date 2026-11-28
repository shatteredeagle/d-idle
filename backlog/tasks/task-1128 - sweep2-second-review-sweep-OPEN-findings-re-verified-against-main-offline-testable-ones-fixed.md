---
id: TASK-1128
title: 'sweep2: second review sweep, OPEN findings re-verified against main, offline-testable ones fixed'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-28T08:10:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 535000
card: "sweep2"
lane: now
owns:
  - "docs/probes/review-sweep2-2026-11-28.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Re-verify the OPEN rows of docs/probes/review-sweep-2026-11-28.md after fix1b, vault2, live1 and fix7b merged; fix the offline-testable ones. Done when the sweep2 doc is written and each fix has a test.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 every OPEN row of the sweep1 doc re-verified against current main, status in docs/probes/review-sweep2-2026-11-28.md
- [ ] #2 offline-testable findings fixed, one commit each, with must-fail tests (two classes where class matters)
- [ ] #3 live check of the fixes on a sandbox hero
<!-- AC:END -->
