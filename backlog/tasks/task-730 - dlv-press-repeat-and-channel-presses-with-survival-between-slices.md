---
id: TASK-730
title: 'dlv-press: repeat and channel presses with survival between slices'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-21T04:44:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 353000
card: "dlv-press"
owns:
  - "internal/combat/"
  - "internal/d2dcombat/"
exclusive: []
contracts: []
result: "Offline only, NOT verified live. Press rows as profile data (press.go): sliceMs per held skill, commit gate (commitLifePct: a commit press is not started under that life percent, the pick falls to the next slot), hard bounds 100..3000 ms so every hold has an explicit release. Knob combat.deliveryUpkeep 0/1/2 (existing dark-launch row; 1 counts and logs differences). Row shipped: Zeal at today's 600 ms (source S); Strafe, Fend, Arctic Blast, Inferno rows wait for a profile that plays them (no profile attacks with them yet; Inferno is not in skills.json) and for fidelity A1. d2dcombat needed no change: the verb already carries the slice as its press time. Tests on Paladin (Zeal) and Druid (Arctic Blast). Scenario combat.hold-interruptible: survival between slices is the fight tick, live drill pending."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Design: docs/design/fable-review-2026-11-21/combat-engine-d-cards.md (user-approved 2026-11-21, all Q1-Q11 as recommended). repeat and channel presses in slices on the verb path; commit gate; Zeal, Strafe, Fend, Inferno, Arctic Blast rows. Explicit release verb and expectation timeout. Depends on dlv-cast-verbs.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Scenario combat.hold-interruptible (offline: gate and slice only)
- [ ] #2 A1 attack-sequence rate known
- [x] #3 Two builds tested (Paladin + Sorceress), no class or skill literal added, literal count reported in result (may only fall)
- [x] #4 Knob combat.delivery.<driver> 0/1/2 where a driver ships; rollback is the knob
- [ ] #5 go test -race -count=20 on touched packages; tools/ci/check.sh, contracts/backlog checks and leakcheck clean
<!-- AC:END -->
