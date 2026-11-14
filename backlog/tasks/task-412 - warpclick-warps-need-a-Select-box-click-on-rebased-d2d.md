---
id: TASK-412
title: 'warpclick: warps need a Select-box click on rebased d2d'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-14T11:46:00Z'
updated_date: '2026-11-14T12:34:00Z'
labels:
  - bot
  - bug
  - tester
dependencies: []
priority: high
ordinal: 251000
card: "warpclick"
result: "Cause V: upstream 66aece1a takes a warp only for a click in LvlWarp's Select box (game.exe rule); the bot clicked the tile corner, outside it at the Den, Cave, Underground Passage, Hole. Plus 0ddb6fe7: the walk to a unit on a blocked subtile stops short (level 20 entrance 0/8). Fork didle/rebase-20261114b 75891650 reports a click point and walks to the nearest free spot; bot pushes the click point. devp7 built, not swapped: fresh heroes entered the Den on seeds 7 and 11. Verified headless and in throwaway bot runs on the worker host, not on the sandbox."
owns:
  - "internal/d2dctl/parse.go"
  - "internal/d2dctl/parse_test.go"
  - "internal/d2map/static.go"
  - "internal/d2dworld/world.go"
  - "internal/nav/exit.go"
  - "internal/nav/warpclick_test.go"
  - "docs/probes/warpclick-2026-11-14.md"
  - "docs/probes/test-characters.md"
  - "docs/upstream/pr-queue.md"
exclusive:
  - "fork-branch:didle/rebase-20261114b"
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

The hour-4 run on devp6 never entered the Den of Evil. Done when: the cause is verified, the fork and bot fixed, devp7 built.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Cause of the Den of Evil never entered on devp6 found and verified
- [x] #2 Fork debug warps reports a click point; bot pushes it; unit tests
- [x] #3 Entrances checked headless: which work before and after
- [x] #4 d2dbot:devp7 built, not swapped; swap and rollback commands written
<!-- AC:END -->
