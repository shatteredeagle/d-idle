---
id: TASK-99
title: 'fid-lowvalue: model-check: low-value deviations and unreachable rows'
status: Deferred
assignee:
  - credit-runner
created_date: '2026-11-07T18:04:00Z'
labels:
  - bot
  - model-check
  - d2d-monsters
  - d2d-skills
  - d2d-time
  - fidelity
  - parked
milestone: m-2
dependencies: []
priority: low
ordinal: 97000
card: "fid-lowvalue"
owns:
  - "docs/probes/fid/fid-lowvalue.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Fidelity matrix rows: SK10 SK15 ME05 EN02 PR03. Method/kind: d2d-measure. Live game (host time): no. Size: S. Fix: none; documented, not fixed (matrix open question 2: no gated metric changes). Dependencies: none. Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.

Done when: Per row write the one-line reason it changes no gated metric: Static Field legacy only (SK10), skills the...; Re-open PR03 when T4 'clear Act 1' starts (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Per row write the one-line reason it changes no gated metric: Static Field legacy only (SK10), skills the bot never casts (SK15), merc life regen/auras before Act 2 (ME05), RNG streams differ by design (EN02), Cain/Tristram skipped until T4 (PR03)
- [ ] #2 Re-open PR03 when T4 'clear Act 1' starts
<!-- AC:END -->
