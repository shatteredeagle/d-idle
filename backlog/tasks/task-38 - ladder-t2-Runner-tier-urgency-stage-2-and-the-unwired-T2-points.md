---
id: TASK-38
title: 'ladder-t2: Runner tier: urgency stage 2 and the unwired T2 points'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-06T16:02:00Z'
labels:
  - bot
  - feature
  - ladder
  - T2
dependencies: []
priority: medium
ordinal: 38000
card: "ladder-t2"
owns:
  - "docs/ladder-t2.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

ADR 0016 tier T2 Runner. Unwired per docs/t0-slice.md: t2.failstreak, t2.reset.game; urgency stage 2 (combat computes urgency instead of five fixed life bands, ADR 0016 staging note). The altar stays parked.

Done when: docs/ladder-t2.md: design of urgency stage 2 (inputs, bands, how it replaces the five life bands with...; t2.failstreak and t2.reset.game wired with the legacy default as untrained answer; parity tests on two builds; The altar is listed as parked with the reason; go vet and go test -p 1; leakcheck 0; says verified live or not (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 docs/ladder-t2.md: design of urgency stage 2 (inputs, bands, how it replaces the five life bands with equal default behaviour) with V or U tags
- [ ] #2 t2.failstreak and t2.reset.game wired with the legacy default as untrained answer; parity tests on two builds
- [ ] #3 The altar is listed as parked with the reason; go vet and go test -p 1; leakcheck 0; says verified live or not
<!-- AC:END -->
