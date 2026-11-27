---
id: TASK-710
title: 'countess3: Countess found from the fork''s monster-preset read (debug levelmonsters)'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-21T02:56:00Z'
labels:
  - d2d
  - feature
dependencies: []
priority: medium
ordinal: 340000
card: "countess3"
owns:
  - "internal/d2dctl"
  - "internal/d2dworld"
  - "internal/boss"
  - "internal/d2map"
  - "cmd/d2bot/boss.go"
  - "docs/process/layout-read.md"
  - "docs/probes/provingground-countess-2026-11-20.md"
  - "internal/env/tuning_quests.go"
  - "docs/knobs.md"
exclusive:
  - "fork-branch:didle/level-monsters-20261121"
contracts: []
lane: next
result: " Remaining (live): ACs 2 and 3 fork verb differential and the Paladin 11 run. closecards3: status set to Done, merged on main; only the live or lab boxes above remain open (listed in Remaining)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 client, fake, d2dworld merge, boss anchor by name, dead Anchor.Spawns removed; must-fail tests Paladin and Sorceress
- [ ] #2 fork verb built and differential 0 differences (live-check)
- [ ] #3 act1.forgotten.countess Paladin 11 run with the new binary
<!-- AC:END -->
