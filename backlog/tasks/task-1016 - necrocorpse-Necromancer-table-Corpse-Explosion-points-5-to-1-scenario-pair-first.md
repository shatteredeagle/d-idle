---
id: TASK-1016
title: >-
  necrocorpse: Necromancer table: Corpse Explosion points 5 to 1, scenario pair
  first
status: Done
assignee:
  - credit-runner
created_date: '2026-11-26T19:13:00Z'
updated_date: '2026-11-27T21:25:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 487000
card: "necrocorpse"
lane: next
owns:
  - "docs/probes/necrocaps-2026-11-26.md"
  - "internal/plandata/build-necromancer_summoner_ce1.json"
  - "docs/plan/data/build-necromancer_summoner_ce1.json"
  - "internal/plandata/build-catalog.json"
  - "internal/leveler/necrocorpse_test.go"
  - "tools/provingground/scenarios/necro.json"
  - "tools/provingground/scenarios/tiers.json"
exclusive: []
contracts: []
result: "Offline part done: opt-in table necromancer_summoner_ce1 (Corpse Explosion 1 point; freed points to Raise Skeletal Mage x3 and Clay Golem; Teeth main attack), test TestCorpseExplosionPointsPair, scenario pair necro.corpse-ce5 and necro.corpse-ce1 (seeds 11 to 13, thresholds U). Default table NOT changed: the 5 to 1 move waits for the live pair; SME numbers U. Not verified live. Build-specific: Necromancer only, data in internal/plandata."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Candidate table with Corpse Explosion 5 points to 1 (opt-in necromancer_summoner_ce1), pair test, and the scenario pair necro.corpse-ce5 / necro.corpse-ce1 (offline; default table unchanged)
- [ ] #2 Live: run the pair on seeds 11 to 13; move the default table to 1 point only if ce1 reads equal or better (live check, may stay open)
<!-- AC:END -->
