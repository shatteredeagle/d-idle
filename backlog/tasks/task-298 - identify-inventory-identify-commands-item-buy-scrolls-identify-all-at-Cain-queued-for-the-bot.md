---
id: TASK-298
title: >-
  identify: inventory identify commands (item, buy scrolls, identify all at
  Cain) queued for the bot
status: Done
assignee:
  - credit-runner
created_date: '2026-11-11T13:46:00Z'
updated_date: '2026-11-11T16:25:00Z'
labels:
  - watch
  - feature
dependencies: []
priority: medium
ordinal: 208000
card: "identify"
owns:
  - "internal/invdesk"
  - "internal/watch/web/inventory-actions.js"
  - "internal/watch/web/inventory-actions.css"
  - "docs/inventory-actions.md"
  - "internal/watch/invapi.go"
  - "internal/watch/invapi_test.go"
  - "internal/watch/itemactions_test.go"
  - "internal/watch/testdata/itemactions"
  - "internal/bot/playeraction.go"
  - "internal/bot/playeraction_test.go"
  - "cmd/d2bot/invwire.go"
  - "internal/town/playerid.go"
  - "internal/town/playerid_test.go"
  - "internal/town/npc.go"
  - "internal/watch/invui.go"
  - "internal/watch/mercdoll_test.go"
  - "internal/town/identify.go"
  - "internal/town/playerid_sim_test.go"
  - "tools/uishots/pages.go"
exclusive: []
contracts: []
result: "Built offline, NOT verified live: identify item, buy scrolls (reserve, tome fill) and Cain errand with desk, town, API schema 5, UI and goldens; click paths and Cain menu line are U, town sim tests skip without game tables. Live checks 8 to 12 in docs/inventory-actions.md."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Size: M

Why: identify is a player command, not an equipment decision. Done when: the three commands queue and run at the next opportunity with states; nothing verified live.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Item menu Identify, header buttons Buy identify scrolls and Identify all (Cain), queue states and cancel
- [x] #2 Town primitives: identify one item, buy scrolls with reserve, Cain identify all (offline tests, live checks listed)
- [x] #3 API schema 5 additive with golden fixtures, DOM goldens, uilint clean
<!-- AC:END -->
