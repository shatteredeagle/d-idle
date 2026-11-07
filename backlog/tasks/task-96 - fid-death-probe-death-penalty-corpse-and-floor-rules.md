---
id: TASK-96
title: 'fid-death: probe: death penalty, corpse and floor rules'
status: Proposed
assignee: []
created_date: '2026-11-07T18:04:00Z'
labels:
  - fidelity
  - d2d-measure
  - d2d-combat
  - d2d-items
dependencies: []
priority: high
ordinal: 94000
card: "fid-death"
owns:
  - "docs/probes/fid/fid-death.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Fidelity matrix rows: PR08. Method/kind: d2d-measure. Live game (host time): yes (needs input on a throwaway character; game.exe never run for the rule). Size: M. Fix: upstream PR if the corpse list or floor differs (d2d many bodies, original one); needs the proof standard docs (differential vs original code) before any upstream PR. Dependencies: user answer to matrix open question 4. Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Die once on a clvl 6 throwaway on game.exe: gold toll min(clvl,20) %, floor clvl x 500, XP loss 0 % in Normal, one corpse; compare d2d 120/120 doc rule
- [ ] #2 Corpse count and item loss listed; tolerance exact
<!-- AC:END -->
