---
id: TASK-680
title: 'cainfix3: scenario-backed Search for Cain bot fixes (ban, crowd, recovery, start priority)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-19T23:13:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 332000
card: "cainfix3"
owns:
  - "cmd/d2bot/adapters.go"
  - "data/questboard/act1.json"
  - "docs/combat.md"
  - "docs/knobs.md"
  - "docs/questboard.md"
  - "internal/bot/"
  - "internal/combat/"
  - "internal/env/tuning_combat.go"
  - "internal/questboard/def.go"
  - "internal/task/banote_test.go"
  - "internal/task/gate.go"
  - "internal/task/world.go"
  - "tools/provingground/scenarios/cain.json"
exclusive: []
contracts: []
result: "GREEN on the builder lab slots (CHEATED setups, mechanics only): cain.stones Paladin 11 and Sorceress 3, cain.cage both, cain.scroll Sorceress 3. cain.chain Paladin 11 is FLAKY, not green: GREEN once without the d2bot adapter wiring of the crowd override, RED once with it (potion trip out of Tristram returns by a stale town portal, see docs/probes/provingground-cainfix3-2026-11-19.md). F1 ban waiver and F2 override are unit-tested; no run hit a death ban. Build-specific data: Chunk.noGold in data/questboard/act1.json, knob combat.crowdAbandonQuestN. Remaining (lab): AC 3 cain.chain Paladin 11 is flaky, not green."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the Proving Ground chain runs of 2026-11-19 found the bot stalling a started Search for Cain chunk (fight gate under a level goal, purse recovery, death bans, crowd abandon). Done when: cain.stones, cain.cage and cain.chain are green on the builder. Evidence: docs/probes/provingground-cainfix3-2026-11-19.md.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 cain.stones green Paladin and Sorceress
- [x] #2 cain.cage green
- [ ] #3 cain.chain Paladin seed 11 (flaky, see result)
- [x] #4 unit tests two classes, race x20
<!-- AC:END -->
