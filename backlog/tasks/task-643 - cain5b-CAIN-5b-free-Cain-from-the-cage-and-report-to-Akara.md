---
id: TASK-643
title: 'cain5b: CAIN-5b free Cain from the cage and report to Akara'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-18T15:41:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 322000
card: "cain5b"
owns:
  - "cmd/d2bot/cain5b_test.go"
  - "cmd/d2bot/objects.go"
  - "internal/bot/task.go"
  - "internal/bot/task_acts_test.go"
  - "internal/bot/questboard_s7b_test.go"
  - "internal/quest/dialog.go"
  - "internal/task/act1.go"
  - "internal/task/testdata/plans/act1.txt"
  - "data/questboard/act1.json"
  - "internal/watch/testdata/quests"
  - "internal/questboard/chunks_test.go"
  - "docs/probes/search-for-cain-5b-2026-11-18.md"
  - "docs/questboard.md"
exclusive: []
contracts: []
result: "Offline (not live): the Tristram cage step fights the hostiles near the hero with the normal Fighter first (bounded, Danger fault without a click while they live), then opens the cage and reads success from the quest word (0x2012, bit 1), 3 tries; board chunks free_cain and report are supported; the whole-level Tristram clear is gone. Sandbox build needs no new verb (the word comes from dump quests). Tests: guards alive, freed from the word, no-word failure, no-reader, melee and caster, L2 fixtures. Live drill open. Doc docs/probes/search-for-cain-5b-2026-11-18.md."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Cage step fights the guards (bounded) and never clicks while they live
- [x] #2 Success read from quest word bit 1; report to Akara via existing talk step
- [x] #3 Board chunks free_cain and report supported; tests incl. two classes, fixtures
- [ ] #4 Live drill (not run): sandbox build, real archers, chicken logic
<!-- AC:END -->
