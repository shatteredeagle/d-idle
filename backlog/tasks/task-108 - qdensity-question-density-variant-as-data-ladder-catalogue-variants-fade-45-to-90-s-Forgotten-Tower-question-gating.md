---
id: TASK-108
title: >-
  qdensity: question-density variant as data (ladder catalogue variants, fade 45
  to 90 s, Forgotten Tower question gating)
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-07T21:56:00Z'
updated_date: '2026-11-11T21:25:00Z'
labels:
  - bot
  - feature
  - ladder
  - T0
milestone: m-4
dependencies:
  - TASK-19
priority: high
ordinal: 101000
card: "qdensity"
owns:
  - "internal/decide/variants.go"
  - "internal/decide/variants_test.go"
  - "docs/t0-variants.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Why: direction review section 7 step 3 and H1 experiment. Done when: a second 30-minute session with a different feel and a comparable record is possible.

Deferred 2026-11-07 (ADR 0018): superseded by codexfirsts (TASK-130); question density in hour one comes from Act 1 firsts. Reopen only if the first playsession2 (TASK-134) shows a density problem the Codex does not fix.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A design variant is a directory or data file (fight-point thresholds, 'first time I meet X' points where a call site exists, fade seconds 45 to 90), loaded without a Go change; test against two builds
- [ ] #2 The t0.area.toohard question is gated to within two areas of the Forgotten Tower; test
- [ ] #3 docs/t0-variants.md says built vs verified live; go vet and go test -p 1 of internal/decide; leakcheck 0
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Closed as superseded in the 2026-11-10 true-up: folded into the return screen by ADR 0018.

Driftfix 2026-11-11: reopened; Done had open offline boxes that no live-check row covers and the card result does not show them built.
<!-- SECTION:NOTES:END -->
