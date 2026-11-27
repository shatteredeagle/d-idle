---
id: TASK-972
title: >-
  townflip: P5 finish and flip town.plan: stage 2 on sandboxes, then stages 3 to
  5
status: In Progress
assignee:
  - live-agent
created_date: '2026-11-26T03:54:00Z'
updated_date: '2026-11-27T18:49:00Z'
labels:
  - bot
  - chore
dependencies:
  - TASK-955
priority: high
ordinal: 460000
card: "townflip"
owns:
  - "docs/design/townflip-p5.md"
  - "internal/town/sellplan.go"
  - "internal/town/sellplan_test.go"
  - "internal/town/shop.go"
  - "internal/town/town.go"
  - "internal/town/chores.go"
  - "internal/env/tuning_town.go"
exclusive: []
contracts: []
result: "Not verified live. Stage 4 sell side built behind town.plan 4 (default stays 0): the sell chore keeps the plan line's max(target, min) potions belt plus pack, best tier first, the rest is surplus; tested offline on Sorceress and Paladin. Runbook with the flip sequence, live check list and counts table in the design doc. Stage 2 is on in the dev sandbox after a 45 minute bundle window only; the farm sandbox and the 24 hour window are not recorded, so no criterion is ticked. Stage 3 waits on the owner look and the malus-errand-gold scenario, the pickup side of stage 4 on town6, stage 5 on town8. Build-specific data lives in the supply profiles and the class belt plan, none in Go. Remaining (live): stage 2 on both sandboxes with the 24 h watch, and the per-stage default flips (ACs 1 and 2). Dependencies checked 2026-11-27: town5 (TASK-757) and town6 (TASK-758) stage code is merged. Remaining (live): ACs 1 and 2 (24 h sandbox watch, default flips). Remaining (live): AC 1 stage 2 on both sandboxes with the 24 h watch, AC 2 default flipped per stage with counts."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Cause class killed: C supply and economy (49, the largest). Roadmap row P5 (docs/research/fable-logic-review-2026-11-25.md 7.1; ADR 0028 as decided). `town.plan` is built to stage 3 and default 0 (town1 to town5 merged or in flight); this card owns the FLIP, not the stage code: set stage 2 on both sandboxes (quest and farm), scenario set green, then flip stage 2 with a 24 h invariant-monitor watch and auto-revert on a hard breach, then flip the default to 2, then stages 3, 4 (town6) and 5 (town8) one at a time, then town11 deletes the old counters and resource.* knobs. Updates the dark-launch row on each step. Depends: P1 counts exist before the flip; stage code cards town5, town6, town7, town8. Risk: medium, mitigated by before/after counts. Enable criteria (from data/darklaunch.json): stage 2 scenario set green, scenario/drill green, flip on the sandboxes, 24 h invariant-monitor watch, auto-revert on a hard breach (the standard rule of user decision 2026-11-26); later stages the same, one at a time. Owner pause rule of ADR 0028: user looks before purse, sell, route stages.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Stage 2 set on both sandboxes with the 24 h watch recorded
- [ ] #2 Default flipped per stage with counts in the card result; darklaunch row updated each step
<!-- AC:END -->
