---
id: TASK-78
title: 'fid-belt-pr: oracle: belt pickup lands in belt first'
status: Deferred
assignee:
  - credit-runner
created_date: '2026-11-07T18:03:00Z'
updated_date: '2026-11-08T17:18:00Z'
labels:
  - d2d
  - upstream-pr
  - d2d-items
  - d2d-loot
  - fidelity
  - parked
milestone: m-2
dependencies:
  - TASK-92
priority: medium
ordinal: 76000
card: "fid-belt-pr"
owns:
  - "docs/probes/fid/fid-belt-pr.md"
exclusive: []
contracts: []
result: "NO DEVIATION, already upstream (2026-11-08, agent decomp5): belt pickup 390,000 and shift-click 390,000 oracle cases on upstream 1453f853, 0 differ (V run, game.exe functions unhooked); docs/probes/fid/fid-belt-pr.md. AC 3: shim R04 stays until the lab binary carries the fix (needs TASK-92). Offline."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Fidelity matrix rows: IT09. Method/kind: upstream-pr. Live game (host time): no (belt-pickup-trace is a V game.exe run). Size: S. Fix: upstream PR (pr-belt-draft.md) and retire shim R04 in the bot; needs the proof standard docs (differential vs original code) before any upstream PR. Dependencies: lab-binary rebase for the shim. Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.

Done when: Belt PR evidence pack (pr-belt-evidence-pack.md) re-run on current main: pickup of a potion with free belt...; Autobelt and belt size/column data checked for belt base types in the plan (belt sizes tie to belt base); After the fix: shim R04 removed from the bot or kept with a removal condition in docs/d2d-deviations.md (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Belt PR evidence pack (pr-belt-evidence-pack.md) re-run on current main: pickup of a potion with free belt slots lands in the belt; tolerance: 0 differing cases in the oracle run; a deviation becomes a docs/d2d-deviations.md entry or an upstream PR
- [x] #2 Autobelt and belt size/column data checked for belt base types in the plan (belt sizes tie to belt base)
- [ ] #3 After the fix: shim R04 removed from the bot or kept with a removal condition in docs/d2d-deviations.md
<!-- AC:END -->
