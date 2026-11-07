---
id: TASK-95
title: 'fid-wp-tp-pr: upstream: waypoint and town portal deviations'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-07T18:04:00Z'
labels:
  - d2d
  - upstream-pr
  - d2d-levels
  - d2d-movement
  - d2d-town
  - fidelity
milestone: m-2
dependencies: []
priority: medium
ordinal: 93000
card: "fid-wp-tp-pr"
owns:
  - "docs/probes/fid/fid-wp-tp-pr.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Fidelity matrix rows: PR06 PR07. Method/kind: upstream-pr. Live game (host time): no. Size: S. Fix: upstream PR (pr-wp draft; town-portal deviations); shim R06; needs the proof standard docs (differential vs original code) before any upstream PR. Dependencies: none. Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.

Done when: Waypoint trace and wp_refuse.py re-run; 39-bit mask, refusal window; tolerance: 0 differing cases in the...; Town portal: tome use, scroll count, portal pos 0.6 cells south, expiry; deviations listed with PR state; Shim R06 (act 3-5 coordinates) removal condition recorded (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Waypoint trace and wp_refuse.py re-run; 39-bit mask, refusal window; tolerance: 0 differing cases in the oracle run; a deviation becomes a docs/d2d-deviations.md entry or an upstream PR
- [ ] #2 Town portal: tome use, scroll count, portal pos 0.6 cells south, expiry; deviations listed with PR state
- [ ] #3 Shim R06 (act 3-5 coordinates) removal condition recorded
<!-- AC:END -->
