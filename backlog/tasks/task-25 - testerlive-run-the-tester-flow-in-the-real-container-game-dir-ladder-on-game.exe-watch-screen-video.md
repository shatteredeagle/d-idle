---
id: TASK-25
title: "testerlive: run the tester flow in the real d2d container"
status: Proposed
assignee: []
created_date: '2026-11-06T14:27:00Z'
labels:
  - tester
dependencies:
  - TASK-31
  - TASK-32
  - TASK-33
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
Slice 1 (d2d route), reworked from the Wine version. Depends on testerd2dimage (TASK-31), testerd2dwiz (TASK-32) and testerrt (TASK-33). Uses the worker host Docker only, no lab game instance; throwaway data volume removed afterwards. The game data is a copy of the owner's own MPQs mounted read-only. Estimate 8 h.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 image built on the worker host; wizard to character to the bot playing; watch screen opened from a second machine with the token; a card appears and an answer lands; an order is given; the video (tap source) shows the game; stop, start and a same-volume image upgrade keep the character and the ladder store
- [ ] #2 the minimal MPQ set (d2data, d2exp, d2char, Patch_D2) is tried and the answer recorded in docs/tester-live.md (required set confirmed or widened)
- [ ] #3 every defect found is listed with a fix owner; the result says verified live or not, writes after start 0 stated
<!-- AC:END -->
