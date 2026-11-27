---
id: TASK-444
title: >-
  verifyp03p05: independent verifier for upstream candidates P03 (hit-recovery
  modes) and P05 (load gold clamp)
status: Done
assignee:
  - credit-runner
created_date: '2026-11-14T21:28:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - docs
  - research
dependencies: []
priority: medium
ordinal: 259000
card: "verifyp03p05"
owns:
  - "docs/upstream/verifier-p03-p05-2026-11-14.md"
exclusive: []
result: "P05 PASS WITH CHANGES, P03 PASS WITH CHANGES (not postable: clang-tidy 3 errors, restart claim lacks an oracle, block_len.py game never run); d2d side V run on the NAS builder (17,195/34,314 -> 0, R 10 15 -> 10 10, 363/539 -> 0), game side S; not verified live, no emulator run (worker host off limits) closecards3 Remaining (live): #5 need the live game, a browser or a worker-host run; not verified here."
contracts: []
lane: now
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Opus verifier pass before posting; docs only; nothing posted. Consumer: decomp21 (TASK-440) and the S6 post step of docs/upstream/pipeline-2026-11-14.md.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Verdict per item (PASS / PASS WITH CHANGES / FAIL) with required changes
- [x] #2 Builds and d2d tests rerun on the NAS builder for oracle, claim and head commits
- [x] #3 Proven vs unproven split, game.exe side vs our docs
- [x] #4 Required worker-host emulator runs listed with exact commands
- [ ] #5 Emulator runs done on the worker host (live/emulator; blocked today, the worker host off limits)
<!-- AC:END -->
