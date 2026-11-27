---
id: TASK-988
title: 'inputl4: Input layer I4: fork verbs buy, sell, repair, take-waypoint, menu-choos'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-26T03:58:00Z'
labels:
  - d2d
  - feature
dependencies:
  - TASK-986
priority: medium
ordinal: 475000
card: "inputl4"
owns:
  - "docs/design/input-i4.md"
  - "internal/input/repairverbs.go"
  - "internal/d2dctl/repair_verb.go"
  - "internal/d2dctl/use_verb_test.go"
  - "internal/d2dctl/fake/fake.go"
  - "internal/d2dinput/repairverbs.go"
  - "internal/town/repairverb.go"
  - "internal/town/repairverb_test.go"
  - "internal/town/chores.go"
  - "internal/town/portalverb_export_test.go"
  - "internal/town/portalverb_unit_test.go"
exclusive: []
contracts: []
result: 'Client side of I4 built offline: CmdRepair, MenuChoose, input.Repairer/MenuChooser, town repair chore on the verb behind input.planner (0). Fork differentials not run (fork half is the owner checklist in docs/design/input-i4.md); live U.'
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Recommendation R17 of the 2026-11-19 review (07 table I4; ADR 0021). Fork branch work: one verb per game client message with a differential each. Needs `exclusive: fork-branch:<name>` and the same-day push rule in AGENTS.md; the implementing agent sets them when it starts. Patches never go into this repo. Risk: medium. Enable criteria: one differential per verb.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Fork verbs with a differential each, pushed to the private remote the same day (fork half not done here: checklist in docs/design/input-i4.md section 3)
- [x] #2 Client side: repair and menu-choose verbs, repair chore adopts repair behind input.planner (offline tests, two classes)
<!-- AC:END -->
