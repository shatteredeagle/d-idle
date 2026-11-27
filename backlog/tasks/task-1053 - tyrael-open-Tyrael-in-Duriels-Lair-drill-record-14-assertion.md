---
id: TASK-1053
title: 'tyrael: open Tyrael in Duriel''s Lair, drill record 14 assertion'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-27T18:50:00Z'
updated_date: '2026-11-27T19:24:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 511000
card: "tyrael"
lane: next
owns:
  - "internal/town/npc.go"
  - "internal/town/fieldnpc_test.go"
  - "internal/town/fieldnpc_int_test.go"
  - "tools/provingground/scenarios/stepexec-act2.json"
  - "docs/probes/act2-stepexec-drill-2026-11-27.md"
exclusive: []
contracts: []
result: "Offline: town opener allows Tyrael (251) in area 73 via fieldNPCAreas; scenario record 14 asserts 0x20 + 0x2000 (S). Not verified live; rerun listed in the probe doc."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Drill 6 blocked at open Tyrael: town: not in a town. Done when the hand-in talk opens Tyrael in the lair.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Town opener allows field NPCs by table (Tyrael 251 in area 73)
- [x] #2 Scenario record 14 assertion matches the kill (0x20) and the hand-in (0x2000, S)
- [ ] #3 Live rerun of drill.act2.duriel.stepexec (needs a lab slot)
<!-- AC:END -->
