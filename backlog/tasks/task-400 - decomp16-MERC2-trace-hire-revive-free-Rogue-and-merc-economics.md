---
id: TASK-400
title: 'decomp16: MERC2 trace, hire, revive, free Rogue and merc economics'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-14T10:39:00Z'
labels:
  - docs
  - research
dependencies: []
priority: medium
ordinal: 249000
card: "decomp16"
owns:
  - "docs/research/merc2-trace.md"
  - "tools/emu/mc2_setup.sh"
  - "tools/emu/mc2_dis.sh"
  - "tools/emu/mc2_grep.sh"
  - "tools/emu/mc2_fn.py"
  - "tools/emu/mc2_run.sh"
  - "tools/emu/mc2_tab.sh"
exclusive: []
contracts: []
result: "Offline, not verified live. Revive = (merc level^2/2) x 15 max 50,000, carried then stash, gold only (FUN_00579c00 V read; price V run 256 levels + 3,600 hire offers vs the bot formulas, 0 differ); free Rogue FUN_00579180 and an Asheara Iron Wolf reward; bot findings B1-B7; no PR candidate. docs/research/merc2-trace.md"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: hero14 played hours with a dead merc (revive 600 never paid). Consumer: the hero14 follow-up cards (revive gate, gold priority) and docs/contracts/merc-policy.md. Done when: docs/research/merc2-trace.md is committed with the game.exe revive and free-Rogue paths and the price oracles re-run.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 trace doc with V/S/U tags
- [x] #2 game.exe revive path read, hire/revive prices re-run against the bot formulas
<!-- AC:END -->
