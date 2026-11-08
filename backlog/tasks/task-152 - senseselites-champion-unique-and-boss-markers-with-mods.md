---
id: TASK-152
title: 'senseselites: champion, unique and boss markers with mods'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T03:12:00Z'
labels:
  - watch
  - feature
milestone: m-4
dependencies:
  - TASK-151
priority: low
ordinal: 135000
card: "senseselites"
owns:
  - "internal/watch/senses_elites.go"
  - "internal/watch/senses_elites_test.go"
  - "internal/decide/champmods.json"
  - "internal/decide/champmods.go"
  - "internal/decide/champmods_test.go"
  - "internal/watch/senses_units.go"
  - "cmd/d2bot/sensesfeed.go"
  - "docs/senses-units.md"
exclusive: []
contracts: []
result: "Fixtures only, not run live: elite kind from MonsterInfo flags, mod ids to names from decide/champmods.json (S, monumod.txt not in the checkout), Codex first-encounter gate (champion entry or the superunique own entry), tests for a champion with two mods, a superunique and an unknown mod id for Sorceress and Paladin. The champion Codex reader is still missing, so tier 4 opens today by the superunique alt only. Live checklist in docs/senses-units.md. Owns widened."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Design: docs/research/info-unlocks.md. Not on the two-week playable path (returnscreen, policysheet, tallystrip, codexfirsts, playsession2); after it. Done when markers and mod names render from a fixture.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Marker kind from IsChampion/IsUnique/UniqueNo/Boss flag; mod ids mapped to names by a data table (missing today)
- [x] #2 Tier 4 gated by the first champion or superunique Codex entry
- [x] #3 Offline test with a champion with two mods and a superunique
<!-- AC:END -->
