---
id: TASK-756
title: 'pgearned: earned save chain and pre.from_save'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-21T04:51:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 356000
card: "pgearned"
owns:
  - "internal/scenario/earned.go"
  - "tools/provingground/earned"
  - "internal/scenario/earned_test.go"
  - "internal/scenario/setup.go"
  - "internal/scenario/scenario.go"
  - "internal/scenario/ledger.go"
  - "internal/scenario/result.go"
  - "internal/scenario/pgpage.go"
  - "cmd/d2scenario/validate.go"
  - "cmd/d2scenario/ledger.go"
  - "cmd/d2scenario/main_test.go"
  - "tools/provingground/scenarios/cain.json"
  - "docs/process/proving-ground.md"
exclusive: []
contracts: []
result: "Repo side done, offline only: pre.from_save and the milestone catalogue (tools/provingground/earned/milestones.json), SaveManifest with portability states and ResolveSave, d2scenario validate refuses from_save with setup writes, a field start or CHEATED provenance and any CHEATED setup not named CHEATED, EARNED setup tag for a fresh or from_save hero, ledger fields from_save/save_run/save_portability with refusals, ledger release-check (EARNED T4 act run needed, CHEATED excluded). Runner does not copy a save yet (BuildSetup lists from_save as Unsupported, the run is BLOCKED). AC2 save portability and the runner copy need the live game: not verified live."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Design 6.3, 7 card 16. High risk. Design: docs/design/fable-review-2026-11-21/proving-ground-gate.md. Owner approved 2026-11-21 (all 10 questions as recommended).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 pre.from_save loads an earned save
- [ ] #2 save portability verified (U in the design)
- [x] #3 release needs at least 1 EARNED act run; CHEATED excluded from release notes (Q8, from the release after this card)
<!-- AC:END -->
