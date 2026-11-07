---
id: TASK-68
title: >-
  g2gfixscroll: no town portal scroll with funded gold leaves the hero stuck
  outside
status: In Progress
assignee: []
created_date: '2026-11-07T07:05:00Z'
updated_date: '2026-11-07T07:13:00Z'
labels:
  - bot-defect
dependencies: []
priority: medium
ordinal: 66000
card: "g2gfixscroll"
owns:
  - "internal/town/chores.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
docs/research/g2g-diagnosis.md finding S1: seed 17 minute 1035 to 1050, no town portal scroll at gold 304 to 460, pocket stall in Black Marsh, 15 virtual minutes without a kill. U on why the scroll was not bought.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 town code buys a scroll before gear when the scroll count is 0 and gold covers it
- [ ] #2 offline test with a fake shop
<!-- AC:END -->
