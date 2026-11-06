---
id: TASK-31
title: 'testerd2dimage: d2d tester image, realtime entry, compose file, ports'
status: In Progress
assignee: []
created_date: '2026-11-06T14:44:00Z'
updated_date: '2026-11-06T15:12:00Z'
labels:
  - G-F
dependencies:
  - TASK-23
priority: high
ordinal: 31000
card: "testerd2dimage"
owns:
  - "deploy/d2d-container/Dockerfile.d2d"
  - "deploy/d2d-container/entrypoint.sh"
  - "deploy/d2d-container/context.sh"
  - "deploy/d2d-container/compose.tester.yml"
  - "deploy/d2d-container/tester_image_test.go"
  - "deploy/d2d-container/smoke.sh"
  - "docs/d2d-container/README.md"
exclusive: []
contracts: []
result: "Phase A offline (not live): tester profile in entrypoint.sh, fork pin lab2-freezeresume ee5e06d8ef99 (on the fork remote, V), compose.tester.yml, static+dry-run test, README env table. AC1 open: wizard step (testerd2dwiz not merged) and the built image; phase B (build+smoke) deferred, Gate 2 g2f still running."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Slice 1 (d2d route), gaps G1, G3, G7, G8. Depends on testerwatch. Build runs on the worker host. Estimate 6 h.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 image pins the d.idle fork ref the Gate 2 evidence used (coordinator names it; the commit is on the fork remote), vendor inputs reproducible from the pack, D2BOT_GAMEDIR=/game, D2D_MODE=realtime default, tap on 127.0.0.1:8893, watch on 8891 and 8892, tester entry runs the wizard then the bot
- [x] #2 compose.tester.yml: MPQ folder read-only, named volume, ladder/watch/video env, origin from a variable, token file never in the file, 127.0.0.1 publish by default, stop_grace_period; static image test pins these and that no .mpq or Game.exe is in the context
<!-- AC:END -->
