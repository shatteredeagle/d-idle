---
id: TASK-639
title: 'cain45: CAIN-4/5a cairn stones solver and the Tristram portal'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-18T15:15:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 321000
card: "cain45"
owns:
  - "cmd/d2bot/cain.go"
  - "cmd/d2bot/cain_test.go"
  - "internal/d2dsrc/items.go"
  - "internal/task/act1.go"
  - "internal/task/cainstones_test.go"
  - "data/questboard/act1.json"
  - "docs/probes/search-for-cain-45-2026-11-18.md"
  - "docs/actbot.md"
  - "internal/watch/testdata/quests"
  - "docs/questboard.md"
  - "internal/questboard/chunks_test.go"
  - "internal/questboard/state_test.go"
  - "internal/d2dsrc/source_test.go"
exclusive: []
contracts: []
result: "Offline (tests, not live): solveCairn cycles by class with lit verification (15 clicks, 25 blind), tested on all 120 orders; bkd prereq fault; portal entry checks level 38; board chunk stones supported, free_cain still not supported yet. debug portals and presets are read by the sandbox bot already; debug objects (stone NU/ON) is U there, blind fallback built. Doc docs/probes/search-for-cain-45-2026-11-18.md. closecards3 Remaining (live): #5 need the live game, a browser or a worker-host run; not verified here."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Cairn solver cycles by class with lit verification, bounded, tested on all 120 orders
- [x] #2 Key (bkd) prerequisite fault; portal entry checks level 38
- [x] #3 Board: stones chunk supported, cage/Cain still not supported yet
- [x] #4 Sandbox visibility of stone state and portal answered, fallback built
- [ ] #5 Live check on the sandbox (debug objects, 15 clicks, level 38)
<!-- AC:END -->
