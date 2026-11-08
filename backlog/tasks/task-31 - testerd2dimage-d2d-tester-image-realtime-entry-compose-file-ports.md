---
id: TASK-31
title: 'testerd2dimage: d2d tester image, realtime entry, compose file, ports'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-06T14:44:00Z'
updated_date: '2026-11-08T03:49:00Z'
labels:
  - container
  - feature
  - tester
milestone: m-1
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
result: "Status 2026-11-07 (decpass2): AC1 ticked, image built from the public fork tag didle-gate2-lab3 and run end to end (V live, docs/tester-live.md, TASK-25). Earlier: Phase A offline (not live): tester profile in entrypoint.sh, fork pin lab2-freezeresume ee5e06d8ef99 (on the fork remote, V), compose.tester.yml, static+dry-run test, README env table. AC1 open: wizard step (testerd2dwiz not merged) and the built image; phase B (build+smoke) deferred, Gate 2 g2f still running. Status sync 2026-11-07: stays open, AC 1 needs the real-container image build and the wizard hook; tester live run TASK-25 pending."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Slice 1 (d2d route), gaps G1, G3, G7, G8. Depends on testerwatch. Build runs on the worker host. Estimate 6 h.

Done when: image pins the d.idle fork ref the Gate 2 evidence used (coordinator names it; the commit is on the fork...; compose.tester.yml: MPQ folder read-only, named volume, ladder/watch/video env, origin from a variable,... (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 image pins the d.idle fork ref the Gate 2 evidence used (coordinator names it; the commit is on the fork remote), vendor inputs reproducible from the pack, D2BOT_GAMEDIR=/game, D2D_MODE=realtime default, tap on 127.0.0.1:8893, watch on 8891 and 8892, tester entry runs the wizard then the bot
- [x] #2 compose.tester.yml: MPQ folder read-only, named volume, ladder/watch/video env, origin from a variable, token file never in the file, 127.0.0.1 publish by default, stop_grace_period; static image test pins these and that no .mpq or Game.exe is in the context
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Groom 2026-11-06: wizard (TASK-32) is now merged; AC 1 still needs the built image and the entrypoint hook for d2dsetup plus the context.sh build line (deploy/ is owned here).
<!-- SECTION:NOTES:END -->
