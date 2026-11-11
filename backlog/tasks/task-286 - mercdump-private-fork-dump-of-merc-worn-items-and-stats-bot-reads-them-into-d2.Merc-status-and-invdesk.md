---
id: TASK-286
title: >-
  mercdump: private-fork dump of merc worn items and stats, bot reads them into
  d2.Merc, status and invdesk
status: Done
assignee:
  - credit-runner
created_date: '2026-11-11T01:05:00Z'
updated_date: '2026-11-11T16:24:00Z'
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
  - "docs/live-checks.md"
  - "internal/d2"
  - "internal/merc/kinds.go"
  - "cmd/d2bot/display.go"
  - "cmd/d2bot/itemtipwire.go"
  - "cmd/d2bot/mercinv_test.go"
exclusive:
  - "fork-branch:merc-dump"
contracts: []
result: "Offline only, nothing live (2026-11-10). Fork: branch d2d.lab5-mercdump, commit f64db72a, pushed to the forge fork remote, compiled (Release d2d), NOT run; container pin unchanged, not deployed. Bot: d2dsrc reads the merc fields into Snapshot.Merc, d2dmerc fills MercState (Level, Str, Dex, Gear, GearKnown), status.MercInv, invdesk schema 3 merc view; tests TestMercGearDump, TestViewMercBlock, TestMercInv (Sorceress, Paladin). Live checks: docs/live-checks.md TASK-286."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Stage A of docs/research/merc-gear.md (TASK-283), the read side. Done when merged; the fork part lives only on the private fork.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Fork patch: dump lists the merc's worn items, level, Str, Dex (private fork, pushed)
- [x] #2 d2dsrc parses merc items into ContMerc and MercState.Gear/GearKnown
- [x] #3 status.InventoryData.Merc and invdesk.View.Merc filled, golden fixtures
- [x] #4 Live checks listed in docs/live-checks.md
<!-- AC:END -->
