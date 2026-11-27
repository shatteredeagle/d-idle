---
id: TASK-801
title: 'handinbtn: quest row offers Hand in / Start for awaiting_handin, tapper presses the row''s button'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-22T04:03:00Z'
labels:
  - watch
  - bug
dependencies: []
priority: medium
ordinal: 367000
card: "handinbtn"
owns:
  - "internal/questboard/desk_action.go"
  - "internal/questboard/desk_action_test.go"
  - "internal/questboard/desk.go"
  - "internal/watch/web/panels/quest-board/panel.js"
  - "internal/watch/testdata/questui/quest_ui.mjs"
  - "internal/watch/testdata/quests/"
  - "cmd/d2scenario/tap.go"
  - "cmd/d2scenario/tap_test.go"
  - "docs/questboard.md"
exclusive: []
contracts: []
lane: next
result: " Merged (agent/handinbtn); ACs 1 to 3 checked against desk_action_test.go and the quest-board panel. Remaining (live): AC 4 scenarios merc.kashya-handin and act1.burial.blood_raven on a lab slot."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Row of an awaiting_handin part shows one button (Start without a record, Hand in with one)
- [x] #2 Table test over every board state: a button or a stated reason
- [x] #3 Tapper presses the server action of the row
- [ ] #4 merc.kashya-handin and act1.burial.blood_raven green on a lab slot
<!-- AC:END -->
