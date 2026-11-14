---
id: TASK-406
title: 'emurebuild: persistent game.exe emulator image and D2_EMU_DIR'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-14T11:02:00Z'
labels:
  - ops
  - chore
dependencies: []
priority: medium
ordinal: 250000
card: "emurebuild"
owns:
  - "tools/emu"
  - "docs/emu-image.md"
  - "docs/research/emu-reverify-2026-11-14.md"
exclusive: []
contracts: []
result: "Image rebuilt at ~/d2d-lab/emu on the worker host (1.6 GB, 1.4 GB hard links), tools/emu reads D2_EMU_DIR; warp arrival (26/26/25 of 52) and XP grid (119,790, 0 differ) reproduced, run on the worker host not in a live game. docs/emu-image.md, docs/research/emu-reverify-2026-11-14.md"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 tools/emu/emu_setup.sh builds and verifies the image at ~/d2d-lab/emu
- [x] #2 tools/emu scripts read D2_EMU_DIR instead of /tmp paths
- [x] #3 two oracles reproduce the recorded numbers
- [x] #4 docs/emu-image.md and the reverify doc written
<!-- AC:END -->
