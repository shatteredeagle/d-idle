---
id: TASK-721
title: 'dlv-reads: flag-gated read verbs: minions, skillstate, combat events'
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
ordinal: 344000
card: "dlv-reads"
owns:
  - "internal/d2dctl/"
  - "docs/process/combat-reads.md"
  - "internal/scenario/"
  - "tools/provingground/"
exclusive:
  - "fork-branch:didle/combat-reads"
  - "d2d-lab-binary"
contracts: []
result: "Fork (private forge, branch didle/combat-reads-20261121, tip 81b191a94, code 5664e0c4 built and tested headless on the builder, NOT on a live game): read verbs debug minions, skillstate, combatlog [n] behind --read-combat (implied by --lab-verbs), caps read.minions/read.skillstate/read.combat-events, a 1024-event combat journal, every field marked visible or hidden; tests/combat_reads_d2d.py green (Necromancer golem equals the engine pets walk; Paladin and Sorceress skill state; Sorceress Fire Bolt kill with cast, hit, kill events; reads change nothing). Ticks per second: 6000 lockstep steps with 18 monsters and a golem, 0.80-1.03 s without the journal (devp15 binary) and 0.73-1.11 s with it, no measurable difference (builder under load). Bot: internal/d2dctl client + fake + tests (Necromancer, Paladin, Sorceress), docs/process/combat-reads.md; no bot consumer, no scenario assertion kinds yet (AC1 and AC6 open: no driver ships, no knob). go test -race -count=20 d2dctl, whole-repo go vet and test, check.sh, leakcheck clean. Class and skill data: no literal in Go code. closecards3 Remaining (not live): #1, #6 open, no scenario assertion kinds and no shipped driver knob yet."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Design: docs/design/fable-review-2026-11-21/combat-engine-d-cards.md (user-approved 2026-11-21, all Q1-Q11 as recommended). Fork read verbs (Q4, reads under ADR 0022) flag-gated and listed in caps; bot typed clients and scenario assertion kinds of appendix C s3. Hero acts only on reads a player can see (Q5): rows marked visible/hidden, hidden reads test-only. Fork work: same-day push to the private remote, no patch here. Depends on dlv-data.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Fork tests in the style of input_verbs_d2d; one scenario per assertion kind that fails when inverted
- [x] #2 Ticks/s measured with the flag on and off
- [x] #3 Visible/hidden mark on every read row
- [x] #4 Foundation gate: golden parity unchanged
- [x] #5 Two builds tested (Paladin + Sorceress), no class or skill literal added, literal count reported in result (may only fall)
- [ ] #6 Knob combat.delivery.<driver> 0/1/2 where a driver ships; rollback is the knob
- [x] #7 go test -race -count=20 on touched packages; tools/ci/check.sh, contracts/backlog checks and leakcheck clean
<!-- AC:END -->
