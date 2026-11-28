---
id: TASK-1111
title: 'cls1: Druid and Barbarian playable offline (decide rows, plans, scenarios)'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-28T05:32:00Z'
updated_date: '2026-11-28T08:47:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 533000
card: "cls1"
lane: next
owns:
  - "internal/decide/"
  - "internal/plandata/"
  - "internal/d2/classtraits.json"
  - "internal/d2/"
  - "internal/combat/classdata/"
  - "internal/levelplan/"
  - "data/build-cap-ratchet.json"
  - "tools/provingground/scenarios/"
  - "docs/classes/"
  - "tools/provingground/golden/"
  - "internal/watch/policysheet_test.go"
  - "internal/watch/policypreview_test.go"
  - "internal/watch/stance_test.go"
  - "internal/watch/unlockrisk_test.go"
exclusive: []
contracts: []
result: "Offline only, not verified live. Druid and Barbarian have decide rows (mana band, stance knobs, propose lead), judged act2plan and levelplan rows, merc aura rows, six scenarios (validate ok) and gap docs; cap ratchet 110 to 101 rows. Build-specific: warcry stance in profiles.json (barbarian/warcry melee block), table row order in the two endgame90 tables; class-keyed: decide rows, plan rows, merc aura (classtraits.json)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Make Druid and Barbarian playable end to end offline as data. Done when the offline boxes are ticked and the checks pass.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 decide catalogue, policy and propose rows for druid and barbarian, test on both
- [x] #2 act2plan and levelplan rows judged against docs/sme (S), classtraits mercAura rows
- [x] #3 cap ratchet lowered for the hand-kept druid and barbarian tables, consistency check green
- [x] #4 Proving Ground scenarios (level 1-30, earned act2, kit soak) for both classes validate
- [x] #5 gap docs docs/classes/druid-gap-2026-11-28.md and barbarian-gap-2026-11-28.md
- [ ] #6 live: run the new scenarios in the lab (needs live drill)
<!-- AC:END -->
