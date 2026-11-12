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
  - "docs/upstream/drafts/zeal-gap.md"
  - "docs/upstream/pr-queue.md"
exclusive:
  - "fork-branch:fid/zeal-gap-r1"
  - "fork-branch:fork-only/sandbox-zeal"
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Re-derive the Zeal strike gap rule from FUN_00553b10 / FUN_0056e210 / FUN_005dbc60
- [ ] #2 Differential grid (weapons x IAS x WSM) on current upstream main and its parent
- [ ] #3 Sandbox test branch fork-only/sandbox-zeal pushed, image not swapped
- [ ] #4 Draft recorded in the PR queue
<!-- AC:END -->
