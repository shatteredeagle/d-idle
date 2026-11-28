---
id: TASK-1097
title: 'fix8: builds-as-data fixes from fable review 8'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-28T03:46:00Z'
updated_date: '2026-11-28T06:42:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 524000
card: "fix8"
result: "Offline, not verified live. Consistency test with ratchet (110 cap rows, no other finding), no Sorceress fallback (fighter refuses without a profile, soft invariant no-combat-profile), class and combat keys required by the loader, main-attack test covers all tables and classes, capability rows filled from the SME class tables (S), merc aura defaults in classtraits.json, key skill read for readiness, mana_heavy raises the mana need in Normal. Build-specific data: profiles.json traps (Assassin), classtraits.json default mercAura, table mana_heavy and combat keys."
lane: next
owns:
  - "internal/plandata/"
  - "data/build-cap-ratchet.json"
  - "data/capabilities/"
  - "tools/ci/check.sh"
  - "internal/combat/"
  - "cmd/d2bot/"
  - "internal/leveler/"
  - "internal/task/"
  - "internal/bot/"
  - "internal/d2/"
  - "internal/town/"
  - "internal/resource/"
  - "docs/plan/data/"
  - "docs/sme/capability-gaps.md"
  - "docs/invariants.md"
  - "tools/rigs/"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: fable review 8 found the build data unchecked as a whole and two silent fallbacks. Done when: the criteria are ticked and the whole-repo tests pass.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Data-consistency test TestBuildTablesConsistent with ratchet data/build-cap-ratchet.json and a tools/ci step
- [x] #2 Silent Sorceress fallback removed; a profile error stops the fighter with a loud line and an invariant count
- [x] #3 Every plan table carries a combat key, validated in the loader
- [x] #4 TestMainAttackSkillsHaveAKey requires key and band, no skipped tables; Assassin sentries in the profile traps list
- [x] #5 capabilities.json reqLevel/reqSkills filled from repo sources; rest listed U in docs/sme/capability-gaps.md
- [x] #6 Merc aura literals of act2.go and mercwire.go read from classtraits mercAura
- [x] #7 Readiness key_skill_min read; mana_heavy honoured in Normal
<!-- AC:END -->
