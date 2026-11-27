---
id: TASK-983
title: 'bsmdef: boardSM flip: quests.boardSM default 1, then remove old paths and the du'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-26T03:57:00Z'
labels:
  - bot
  - chore
dependencies:
  - TASK-955
priority: high
ordinal: 471000
card: "bsmdef"
owns:
  - "docs/design/boardsm-flip.md"
  - "docs/questboard.md"
  - "internal/questboard"
  - "internal/task"
  - "internal/env/tuning_quests.go"
  - "internal/env/tuning_quests_legacy_test.go"
  - "internal/bot/questboard.go"
  - "internal/bot/task.go"
  - "internal/bot/routeguard.go"
  - "tools/provingground/scenarios/board.json"
exclusive: []
contracts: []
result: "Prep only, no flip, default 0 unchanged. Every read of quests.boardSM goes through boardLegacy() (env, bot, task, questboard); TestBoardLegacySwitchIdle proves idle identity at knob 0 and 1 for Paladin and Sorceress; docs/questboard.md has the removal plan; board.park.persist-restart rewritten (not run). Restart drill and flip still need the live night. Not verified live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Recommendation of the quest-board review (section 5 card 10) and ADR 0025 item 11: knob default 1 after the scenario set is green on a determinism pair; one release later remove the old paths and the plan file dual-write. Cards bsm1 to bsm9 built the machine, store, exec, holds, API and panel; nothing flipped it. Steps: (1) set quests.boardSM 1 on the quest sandbox; restart drill: a park survives a restart; (2) 24 h invariant-monitor watch with auto-revert on a hard breach; (3) default 1, ledger row status default-on; (4) cleanup card after one release. Cause classes: A, B, K for quests (per-quest patch branches, 76 on one quest). Depends on P1 counts. Risk: medium. Enable criteria (data/darklaunch.json): quest sandbox drill green with boardSM 1 across a restart; 24 h invariant-monitor watch with auto-revert on a hard breach.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 boardSM 1 on the quest sandbox with the restart drill recorded
- [ ] #2 Default flipped; ledger row updated; cleanup scheduled
<!-- AC:END -->
