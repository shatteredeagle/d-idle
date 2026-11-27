---
id: TASK-1037
title: 'auraupkeep: Paladin auras run through the upkeep driver'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-27T07:04:00Z'
updated_date: '2026-11-27T07:28:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 501000
card: "auraupkeep"
lane: next
owns:
  - "internal/combat/"
exclusive: []
contracts: []
result: "Offline only, not verified live. keepAura now runs under combat.deliveryUpkeep: in mode 1 the old aura code acts and the driver logs each disagreement once (upkeep shadow, what aura or aura recheck); in mode 2 the driver decides whether the profile chosen aura is up (right hand holds it, then the active aura read, then the learned Skills.txt aurastate) and re-picks it at once when it dropped between checks instead of waiting for combat.auraCheckMs. The aura choice stays the profile ladder. Tests cover Paladin Might and Holy Fire in modes 0, 1 and 2, the learned state fallback and a silent Sorceress. The live shadow run is open. closecards3 Remaining (live): #3 need the live game, a browser or a worker-host run; not verified here."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: live finding on the sandbox Paladin (knob combat.deliveryUpkeep 1): no upkeep shadow line in an hour with Might up, because keepAura in internal/combat/aura.go never consulted the upkeep driver. Done when: keepAura runs under the same knob modes as the buff pass; the aura choice stays the profile's ladder (wantAura). Build-specific data: none in Go; the aura ladder lives in the combat profile, the aura state id in Skills.txt aurastate.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Aura upkeep follows combat.deliveryUpkeep: 1 old code acts and each disagreement is logged once, 2 the driver keeps the profile's aura on and re-picks it at once after a drop
- [x] #2 Tests on two builds: Paladin with Might and Holy Fire, a non-aura class logs nothing
- [ ] #3 Live shadow run on a Paladin shows upkeep shadow lines for auras (knob 1)
- [x] #4 gofmt, vet, package tests, tools/ci/check.sh, contracts, backlog and leakcheck clean
<!-- AC:END -->
