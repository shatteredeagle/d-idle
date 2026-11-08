---
id: TASK-89
title: 'fid-skill-damage-pr: upstream: skill damage min-synergy gate and MulDiv'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-07T18:04:00Z'
updated_date: '2026-11-08T00:27:00Z'
labels:
  - d2d
  - upstream-pr
  - d2d-combat
  - d2d-skills
  - fidelity
milestone: m-2
dependencies: []
priority: medium
ordinal: 87000
card: "fid-skill-damage-pr"
owns:
  - "docs/probes/fid/fid-skill-damage-pr.md"
  - "tools/emu/sdmg_run.sh"
  - "tools/emu/sdmg_items_patch.py"
exclusive: []
contracts: []
result: "no PR candidate: every proven skill-damage deviation is already merged upstream (PR #146); 116,000 cases x 2 seeds on upstream main 9c78f391 = 0 differ, blvl/lvl split 0 of 116,000; not verified live (offline oracle only); docs/probes/fid/fid-skill-damage-pr.md"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Fidelity matrix rows: SK02. Method/kind: upstream-pr. Live game (host time): no. Size: S. Fix: upstream PR (pr-skill-damage draft); needs the proof standard docs (differential vs original code) before any upstream PR. Dependencies: none. Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.

Done when: skill_damage 116,000 cases re-run on current main; expected 0 differ for the 13 skills after the fix;...; Charged/Fire/Ice Bolt and Frost Nova stay clean (no regression) (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 skill_damage 116,000 cases re-run on current main; expected 0 differ for the 13 skills after the fix; tolerance: 0 differing cases in the oracle run; a deviation becomes a docs/d2d-deviations.md entry or an upstream PR
- [x] #2 Charged/Fire/Ice Bolt and Frost Nova stay clean (no regression)
<!-- AC:END -->
