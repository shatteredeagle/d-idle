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
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 tools/emu/emu_setup.sh builds and verifies the image at ~/d2d-lab/emu
- [ ] #2 tools/emu scripts read D2_EMU_DIR instead of /tmp paths
- [ ] #3 two oracles reproduce the recorded numbers
- [ ] #4 docs/emu-image.md and the reverify doc written
<!-- AC:END -->
