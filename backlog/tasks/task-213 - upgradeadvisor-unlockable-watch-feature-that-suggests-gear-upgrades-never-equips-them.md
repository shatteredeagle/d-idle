---
id: TASK-213
title: 'upgradeadvisor: unlockable watch feature that suggests gear upgrades, never equips them'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-09T13:53:00Z'
labels:
  - watch
  - feature
dependencies:
  - TASK-212
priority: low
ordinal: 164000
card: "upgradeadvisor"
owns:
  - "docs/unlock-upgradeadvisor.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Hook point: invdesk.Config.Advice(item) fills ItemView.Advice, drawn under the item detail on /watch/inventory. Unlock row style: ADR 0017/0018/0019 and docs/unlock-ladder.md (ui group, trigger e.g. first magic item or tally price). Data sources: data tables (armor, weapons, itemtypes, treasureclassex, runes, cubemain) through internal/data, the stat reader for identified items, internal/leveler Candidate scoring for the comparison. Done when every tier passes its offline tests.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 T0: ladder row and rough hint line in the hero's voice on the inventory page and journal (hmm, I think this is better), shown only when the unlock row is open
- [ ] #2 T1: exact delta per stat, damage and defence of the candidate against the equipped item, from the item tables and the stat reader (identified items only; unidentified say so)
- [ ] #3 T2: where better bases drop: areas, monsters and treasure classes from the TC and item-type tables, per difficulty
- [ ] #4 T3: further advice (socket count, runeword hints, upgrade by cube recipe) from runeword and recipe data, each line tagged with its source
- [ ] #5 Never equips or unpins: suggestions only (user decision 2026-11-09); a pinned slot still shows suggestions, marked as such; two-class tests (Sorceress, Paladin)
<!-- AC:END -->
