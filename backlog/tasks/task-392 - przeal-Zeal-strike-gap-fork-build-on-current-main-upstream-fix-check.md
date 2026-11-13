---
id: TASK-392
title: 'przeal: Zeal strike gap, fork build on current main, upstream fix check'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-12T23:58:00Z'
labels:
  - d2d
  - upstream-pr
dependencies: []
priority: medium
ordinal: 245000
card: "przeal"
owns:
  - "tools/emu/zg_build.sh"
  - "tools/emu/zg_tree.sh"
  - "tools/emu/zg_run.sh"
  - "tools/emu/zg_game.py"
  - "tools/emu/zg_grid.py"
  - "tools/emu/zg_zeal_test.cpp"
  - "tools/emu/zg_sandbox.sh"
  - "docs/upstream/zeal-gap/"
  - "docs/upstream/drafts/zeal-gap.md"
  - "docs/upstream/pr-queue.md"
exclusive:
  - "fork-branch:fid/zeal-gap-r1"
  - "fork-branch:fork-only/sandbox-zeal"
contracts: []
result: "Not a PR: upstream 276f85c0 (Dravis, 2026-11-12) already fixes the Zeal gap with the same rule (FUN_005dbc60 -> FUN_0056e210 Param2 -> FUN_00553b10 restarts the swing; next strike ceil(256 x action / rate) later). Grid on one animation record: parent 600 of 600 gaps differ (+4..+10 ticks), upstream main 200 of 600 (+1 tick, d2d action_ms scaling, not Zeal-specific). Sandbox branch fork-only/sandbox-zeal 90ac00ab pushed to the private fork, image not swapped. Verified offline on the worker host, no live game."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: decomp13 found Zeal strikes 15 ticks apart in d2d against 7 in game.exe (TASK-330). Done when: the rule is re-derived, an upstream-ready fix is built or shown to be unneeded, and a fork-only sandbox branch carries it.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Re-derive the Zeal strike gap rule from FUN_00553b10 / FUN_0056e210 / FUN_005dbc60
- [x] #2 Differential grid (weapons x IAS x WSM) on current upstream main and its parent
- [x] #3 Sandbox test branch fork-only/sandbox-zeal pushed, image not swapped
- [x] #4 Draft recorded in the PR queue
<!-- AC:END -->
