---
id: TASK-187
title: 'areanames17: minimap names area 17 as ''Area 17'''
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T14:50:00Z'
labels:
  - bot
  - bug
milestone: m-4
dependencies: []
priority: low
ordinal: 155000
card: "areanames17"
owns:
  - "internal/status/areas.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Live check of 2026-11-08: the Sorceress run showed map.area 'Area 17' on /api/senses (docs/panels-live.md). The table behind it (internal/status/areas.go) falls back to 'Area N'. Done when: all Act 1 ids are named and a table test pins them.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Every Act 1 area id the bot visits has its name (Burial Grounds, Catacombs, Tristram, Cave levels, Tower) in the area table used by /api/senses
<!-- AC:END -->
