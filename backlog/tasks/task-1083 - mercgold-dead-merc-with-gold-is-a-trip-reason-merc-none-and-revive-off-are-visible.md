---
id: TASK-1083
title: >-
  mercgold: dead merc with gold is a trip reason, merc none and revive-off are
  visible
status: Done
assignee:
  - credit-runner
created_date: '2026-11-28T00:17:00Z'
updated_date: '2026-11-28T00:38:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 518000
card: "mercgold"
lane: next
result: "Wire revive ran only on unrelated trips (legacy chore hidden by the mercTown wrapper): a dead merc the policy would pay for is now a trip reason; merc none and revive-off log lines, status lastDecision and /api/explain refusal. Paid Act 1 hire (no hire window reader) and the Blood Raven start NOT done; knob stays off. Offline only, not verified live."
owns:
  - "cmd/d2bot/mercwire.go"
  - "cmd/d2bot/mercwire_test.go"
  - "internal/bot/resource.go"
  - "internal/bot/merc_test.go"
  - "docs/probes/merc-revive-live-checklist.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Problem 2 of the sandbox postdeploy audit. Done when the above hold.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 dead merc with gold >= revive bar asks for a town trip (two classes, offline)
- [x] #2 no merc and revive-off are visible in log, status and /api/explain
- [x] #3 live checklist written; paid Act 1 hire and Blood Raven start reported as not done
<!-- AC:END -->
