---
id: TASK-724
title: 'dlv-mark: mark driver: curses and taunts as data'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-21T04:42:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 347000
card: "dlv-mark"
owns:
  - "internal/combat/"
exclusive: []
contracts: []
result: "First slice built offline, NOT verified live and no scenario run. internal/combat/mark.go is the mark driver: profile rows (Profile.Marks: name, when any, pack, eliteOrBoss or magicResist, group, radius, range, self, state, source) and, for a Necromancer profile without rows, its ten curse rules read as rows of group curse (Iron Maiden added to all three Necromancer profiles, a duplicate Decrepify in the summoner profile replaced). Target choice by when, a boss or elite first, a cast must cover a boss, an elite or two unmarked hostiles; area cover from the row radius; the state id comes from Skills.txt aurastate (else the row) and is trusted once seen on a marked unit, until then a cast memory (state duration from the table, else 30 s) covers the radius; one group per unit; self rows (war cries, Cloak of Shadows) cast at the own feet. Gated by the existing knob combat.deliveryUpkeep (no new knob): 0 old curse path, 1 shadow (choice logged once), 2 the driver acts, the old Necromancer curse path stands down and a mark row is never cast as a buff. Tests: Necromancer curses, a Barbarian Howl self row and a Paladin negative (a bound curse is not cast); must-fail checked by disabling the area memory and the target point. No uptime claim (fidelity A2 open); on d2d the put rule deviates, so any scenario floor taken there is a d2d number. Build-specific data: the rows live in classdata/profiles.json; no profile outside the Necromancer has rows yet (the owner adds war cry rows). Literal count unchanged (no class, skill id or skill name added in non-test Go). Not done: scenario combat.curse-covers-pack (no scenario runner; offline test TestMarkCoversPack stands in), removing the curse half of necro.go (cleanup card, once mode 2 is the default)."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Design: docs/design/fable-review-2026-11-21/combat-engine-d-cards.md (user-approved 2026-11-21, all Q1-Q11 as recommended). Target choice by when, area cover from row radius, state observed on targets, one mark group per target; ten curses as data; Taunt, Howl, Battle Cry, Cloak of Shadows as rows. On d2d the put rule deviates: scenario floor is a d2d number, stated in the result. Depends on dlv-upkeep-state.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Scenario combat.curse-covers-pack
- [x] #2 No uptime claim before fidelity A2 verdict
- [x] #3 Two builds tested (Paladin + Sorceress), no class or skill literal added, literal count reported in result (may only fall)
- [x] #4 Knob combat.delivery.<driver> 0/1/2 where a driver ships; rollback is the knob
- [ ] #5 go test -race -count=20 on touched packages; tools/ci/check.sh, contracts/backlog checks and leakcheck clean
<!-- AC:END -->
