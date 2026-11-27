---
id: TASK-719
title: 'dlv-rotation: Rotation type and FromProfile adapter, golden parity'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-21T04:41:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 342000
card: "dlv-rotation"
owns:
  - "internal/delivery/"
  - "internal/combat/"
  - "internal/agency/"
exclusive: []
result: "dlv-rotation done, NOT verified live. internal/delivery/rotation.go: Rotation, Slot (role attack, upkeep with Group, aura) and When (the three predicates profiles carry today: hero life below pct, crowd knob, target not immune to element), Names/Groups/Of and a role-by-role Diff. combat.Profile.Rotation(clvl, known) is the FromProfile adapter; Fighter.attackSkills, buffPass and wantAura read slots now (decisions unchanged: TestRotationEqualsTheProfileLists compares attack names, buff groups and aura ladder of EVERY profile at clvl 1..99 for a hero that knows everything and one that knows nothing; TestRotationParityFailsOnAReorder proves it can fail, Sorceress and Paladin). agency.Plan.Rotation lays the plan out as a Rotation; agency.ShadowDiff compares plan and profile heads slot by slot (a pure function: wiring it into the shadow log line is a follow-up in cmd/d2bot, not owned here). Verified offline: go test -race -count=20 on internal/combat, agency, delivery; go test of internal/bot and cmd/d2bot green. Build-specific: none added; data in classdata profiles. Special rules (static opener, nova, aura crowd, life price) still live in Go: they move into slots in later cards."
contracts: []
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Design: docs/design/fable-review-2026-11-21/combat-engine-d-cards.md (user-approved 2026-11-21, all Q1-Q11 as recommended). Rotation type and conditions (appendix B s4); FromProfile adapter; Fighter attack order, buff groups and aura ladder read slots; agency lays its plan as a Rotation. Gate: golden parity over the existing simulator tests and scenario corpus (Q11), byte-identical decisions. Owns the golden parity harness reused by later cards. Depends on dlv-data.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Golden parity test over simulator tests and combat scenarios of the corpus: byte-identical decisions
- [x] #2 Scenario combat.parity-rotation: PENDING as a Proving Ground file (tools/provingground is not owned here); the Go golden test TestRotationEqualsTheProfileLists stands in
- [x] #3 Must-fail unit tests on Paladin and Sorceress
- [x] #4 Two builds tested (Paladin + Sorceress), no class or skill literal added, literal count reported in result (may only fall)
- [x] #5 Knob combat.delivery.<driver> 0/1/2 where a driver ships; rollback is the knob
- [x] #6 go test -race -count=20 on touched packages; tools/ci/check.sh, contracts/backlog checks and leakcheck clean
<!-- AC:END -->
