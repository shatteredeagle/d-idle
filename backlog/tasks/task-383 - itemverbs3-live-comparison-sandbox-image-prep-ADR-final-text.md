---
id: TASK-383
title: 'itemverbs3: live comparison, sandbox image prep, ADR final text'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-12T20:40:00Z'
updated_date: '2026-11-12T21:14:00Z'
labels:
  - d2d
  - feature
dependencies:
  - TASK-371
priority: medium
ordinal: 242000
card: "itemverbs3"
owns:
  - "docs/design/item-verbs.md"
  - "docs/probes/itemverbs-live-2026-11-12.md"
  - "docs/probes/test-characters.md"
  - "tools/itemverbs/live_compare.py"
  - "tools/itemverbs/compare_items.py"
  - "tools/itemverbs/diff_d2d.py"
  - "internal/invdesk/mover.go"
  - "internal/town/playerstash.go"
  - "internal/town/playerid.go"
exclusive:
  - "fork-branch:fork-only/sandbox-itemverbs"
  - "fork-branch:fork-only/itemverbs"
contracts: []
result: "Live click-vs-verb comparison on throwaway Paladin and Sorceress heroes with the real d2bot (docs/probes/itemverbs-live-2026-11-12.md): equip, swap, identify, stash, take and gold deposit/withdraw give the same end state in both modes (how=verb on every step); the click drop FAILED live on both classes while the verb dropped. Found and fixed the at-chest rule (stash panel open and within 3 cells). New d2d gate --item-verbs. Fork fork-only/sandbox-itemverbs a42dba25 pushed; sandbox image d2dbot:iv1 and swap/rollback scripts prepared, sandbox NOT touched. Knob default stays off."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

User go 2026-11-12: run the live click-vs-verb comparison on a throwaway hero, fix differences, prepare the sandbox image (fork-only/sandbox-itemverbs) and rollback, then (on the coordinator's word) swap the sandbox and watch how=verb|click. Knob default stays off.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Live comparison table recorded in docs/probes/itemverbs-live-2026-11-12.md (Paladin and Sorceress, throwaway hero)
- [x] #2 Differences fixed or explained; fork branch fork-only/sandbox-itemverbs pushed with --item-verbs gate
- [x] #3 Sandbox image and rollback prepared (not swapped)
- [x] #4 ADR 0026 final text in the design doc; gate green
<!-- AC:END -->
