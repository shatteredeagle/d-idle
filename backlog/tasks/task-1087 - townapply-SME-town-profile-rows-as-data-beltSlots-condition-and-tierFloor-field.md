---
id: TASK-1087
title: 'townapply: SME town profile rows as data, beltSlots condition and tierFloor field'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-28T02:21:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 521000
card: "townapply"
lane: next
owns:
  - "internal/supply"
  - "internal/town/planshadow.go"
exclusive: []
contracts: []
result: "Applied SME review 2026-11-27 rows as data (andariel belt-size kit, blood_raven mana 4, tierFloor, duriel, catacombs, tristram, countess); offline tests only, not verified live. Skipped: antidote and thawing lines, field clvl band, merc_revive reorder."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 beltSlots condition and tierFloor field in profiles.json loader, unknown keys rejected
- [x] #2 SME rows applied (andariel, blood_raven, duriel, catacombs, tristram, countess) with tests
- [x] #3 Skipped (behavior code): antidote/thawing lines, field clvl band, merc_revive reorder
<!-- AC:END -->
