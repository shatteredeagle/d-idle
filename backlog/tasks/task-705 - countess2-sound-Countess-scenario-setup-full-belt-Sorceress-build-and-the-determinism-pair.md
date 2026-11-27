---
id: TASK-705
title: >-
  countess2: sound Countess scenario setup (full belt, Sorceress build) and the
  determinism pair
status: Done
assignee:
  - credit-runner
created_date: '2026-11-21T00:34:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - ops
  - probe
dependencies: []
priority: medium
ordinal: 338000
result: "setup fixed (Light potions, town.spareKeep 30, own Sorceress scenario with the fire build; V builder job 20261120-203007-6700). Paladin RED (reaches Tower Cellar 5; the Spawns walk resolves to one blocked click; Countess not found), Sorceress BLOCKED (mana potions used up, 35 max mana). Determinism pairs NOT run. CHEATED setup, mechanics only. closecards3 Remaining (live): #2, #3 need the live game, a browser or a worker-host run; not verified here."
card: "countess2"
owns:
  - "tools/provingground/scenarios/act1.json"
  - "tools/provingground/countess-check.sh"
  - "docs/probes/provingground-countess-2026-11-20.md"
exclusive: []
contracts: []
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 full belt in the setup, Sorceress scenario with the fire build
- [ ] #2 Paladin and Sorceress GREEN on the builder (CHEATED mechanics)
- [ ] #3 determinism pair per green class
<!-- AC:END -->
