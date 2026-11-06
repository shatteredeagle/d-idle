---
id: TASK-25
title: "testerlive: run the tester flow in the real container"
status: Proposed
assignee: []
created_date: '2026-11-06T14:27:00Z'
labels:
  - G-F
dependencies: []
priority: high
ordinal: 25000
card: "testerlive"
owns:
  - "docs/tester-live.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Slice 1. Depends on testerimage. Uses the worker host Docker only, no lab game instance; throwaway data volume removed afterwards.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 image built on the worker host; wizard game-dir to character to slot playing; watch screen opened from a second machine with the token; a card appears and an answer lands; an order is given; the video shows the full client; restart and same-volume upgrade keep character and ladder store (estimate 6 h)
- [ ] #2 every defect found is listed with a fix owner; the result says verified live or not, writes after start 0 stated
<!-- AC:END -->
