---
id: TASK-1109
title: 'cls2: Amazon and Assassin playable offline as data'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-28T05:30:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 533000
card: "cls2"
result: "Offline, not verified live. Amazon and Assassin: Act 2 and level plan rows follow the caster trait (melee archetype), merc aura rows per difficulty, decide rows (mana band, zone lead, stance), capability rows Inferno Sentry, Royal Strike and the Fire Golem alias filled from the skill extract (Dragon Flight stays U), ammo path moved from a Go switch into the table key ammo, ten Proving Ground scenarios validated (none run), two gap docs. Build-specific data: classtraits.json mercAura, build table ammo and combat keys, profiles.json traps, decide class rows."
lane: next
owns:
  - "internal/plandata/"
  - "internal/d2/"
  - "internal/decide/"
  - "data/capabilities/"
  - "internal/combat/"
  - "cmd/d2bot/"
  - "tools/provingground/"
  - "internal/leveler/"
  - "docs/classes/"
  - "docs/sme/capability-gaps.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: fable review 8 found the Amazon and the Assassin not playable end to end: plan rows contradicting the class traits, no decide rows, capability rows without level facts, no scenarios. Done when: the data is consistent and tested on two builds per class, scenarios validate and the gap docs say what needs the lab.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 classtraits, act2plan and levelplan rows agree for Amazon and Assassin (test)
- [x] #2 decide rows for both classes (test on two builds each)
- [x] #3 the 4 open capability gaps sourced or left U with a note
- [x] #4 bow vs javelin profile by combat key; Assassin trap driver and Death Sentry corpse share tested
- [x] #5 Proving Ground scenarios for both classes validate
- [x] #6 gap docs amazon-gap and assassin-gap written
- [ ] #7 live drill of the new scenarios (needs live)
<!-- AC:END -->
