---
id: TASK-176
title: 'tallyempty: tally pays 2 credits for an expedition with 0 kills and 0 XP'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T03:59:00Z'
labels:
  - watch
  - bug
milestone: m-4
dependencies: []
priority: medium
ordinal: 143000
card: "tallyempty"
owns:
  - "internal/currency/tallyrows.json"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Found by paladinsmoke (TASK-142). With a hero that killed nothing, GET /api/tally shows '+2 expedition N completed' for every walk in and out of Blood Moor (45 credits and 32 sealed after 20 such trips; sealed 486 after 4 virtual hours with 0 kills, 0 XP). The tally should reward effort (kills, XP, first sights), not an empty trip, or a stuck hero buys charters by itself. Done when: an expedition credit needs kills or XP above a floor (data in tallyrows.json), with a test for a zero-kill trip and for a normal trip.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A zero-kill, zero-XP expedition credits 0 (offline test)
- [ ] #2 A normal expedition still credits its row
<!-- AC:END -->
