---
id: TASK-371
title: 'itemverbs2: identify, drop, move, gold verbs and bot wiring'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-12T17:03:00Z'
labels:
  - d2d
  - feature
dependencies:
  - TASK-367
priority: medium
ordinal: 238000
card: "itemverbs2"
owns:
  - "docs/design/item-verbs.md"
  - "internal/invdesk/mover.go"
  - "internal/invdesk/mover_test.go"
  - "internal/invdesk/desk.go"
  - "internal/invdesk/stash.go"
  - "internal/invdesk/identify.go"
  - "internal/d2dctl/item_verbs.go"
  - "internal/d2dctl/item_verbs_test.go"
  - "internal/town/town.go"
  - "internal/town/playerid.go"
  - "internal/town/playerstash.go"
  - "cmd/d2bot/invverbs.go"
  - "cmd/d2bot/d2dwire.go"
  - "tools/itemverbs/diff_d2d.py"
  - "tools/itemverbs/compare_items.py"
exclusive:
  - "fork-branch:fork-only/itemverbs"
contracts: []
result: "Fork branch fork-only/itemverbs commit 59ae7810 (private forge): item move/identify/drop and gold deposit/withdraw verbs, stash only at the chest; headless differential verb vs manual command sequence matched 3 of 3 runs. Bot: VerbMover/VerbTrasher wired on the d2d backend behind knob invdesk.verbs (default OFF), how=verb|click on every settled request; live click-vs-verb differential prepared (docs/design/item-verbs.md 9a, tools/itemverbs), NOT run live. 0x27 field order settled by disassembly: item first, scroll second."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

User go 2026-11-12: identify, drop, move (pack/stash/cube) and gold verbs in the fork (stash strictly at the chest), the bot executors wired in cmd/d2bot behind invdesk.verbs (default off) with per-action how=verb|click, and the live click-vs-verb differential prepared as docs plus tools. Done when the fork tests, the fakes tests and the gate are green.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Fork: item move/identify/drop and gold verbs with unit and headless differential tests, pushed to the private forge
- [x] #2 Bot: VerbMover/VerbTrasher wired in cmd/d2bot behind invdesk.verbs (default off), how=verb|click per action, tests with fakes for two classes
- [x] #3 Differential recipe and tools (docs/design/item-verbs.md 9a, tools/itemverbs) and the offline differential result
- [ ] #4 Gate: whole-repo labrun, race on touched packages, checks, leakcheck
<!-- AC:END -->
