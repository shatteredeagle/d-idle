---
id: TASK-286
title: 'mercdump: private-fork dump of merc worn items and stats, bot reads them into d2.Merc, status and invdesk'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T01:05:00Z'
labels:
  - d2d
  - feature
dependencies: []
priority: medium
ordinal: 205000
card: "mercdump"
owns:
  - "internal/d2dsrc"
  - "internal/d2dmerc"
  - "internal/status"
  - "internal/invdesk"
  - "docs/merc-wire.md"
exclusive:
  - "fork-branch:merc-dump"
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Stage A of docs/research/merc-gear.md (TASK-283), the read side. Done when merged; the fork part lives only on the private fork.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Fork patch: dump lists the merc's worn items, level, Str, Dex (private fork, pushed)
- [ ] #2 d2dsrc parses merc items into ContMerc and MercState.Gear/GearKnown
- [ ] #3 status.InventoryData.Merc and invdesk.View.Merc filled, golden fixtures
- [ ] #4 Live checks listed in docs/live-checks.md
<!-- AC:END -->
