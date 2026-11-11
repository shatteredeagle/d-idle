---
id: TASK-298
title: 'identify: inventory identify commands (item, buy scrolls, identify all at Cain) queued for the bot'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T13:46:00Z'
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
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Size: M

Why: identify is a player command, not an equipment decision. Done when: the three commands queue and run at the next opportunity with states; nothing verified live.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Item menu Identify, header buttons Buy identify scrolls and Identify all (Cain), queue states and cancel
- [ ] #2 Town primitives: identify one item, buy scrolls with reserve, Cain identify all (offline tests, live checks listed)
- [ ] #3 API schema 5 additive with golden fixtures, DOM goldens, uilint clean
<!-- AC:END -->
