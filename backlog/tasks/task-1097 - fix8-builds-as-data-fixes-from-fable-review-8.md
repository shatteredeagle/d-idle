---
id: TASK-1097
title: 'fix8: builds-as-data fixes from fable review 8'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-28T03:46:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 524000
card: "fix8"
lane: next
owns:
  - "internal/plandata"
  - "data/build-cap-ratchet.json"
  - "data/capabilities"
  - "tools/ci/check.sh"
  - "internal/combat"
  - "cmd/d2bot"
  - "internal/leveler"
  - "internal/task"
  - "internal/bot"
  - "internal/d2"
  - "internal/town"
  - "internal/resource"
  - "docs/plan/data"
  - "docs/sme/capability-gaps.md"
  - "docs/invariants.md"
  - "tools/rigs"
  - "internal/power"
  - "internal/delivery"
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
- [ ] #1 Data-consistency test TestBuildTablesConsistent with ratchet data/build-cap-ratchet.json and a tools/ci step
- [ ] #2 Silent Sorceress fallback removed; a profile error stops the fighter with a loud line and an invariant count
- [ ] #3 Every plan table carries a combat key, validated in the loader
- [ ] #4 TestMainAttackSkillsHaveAKey requires key and band, no skipped tables; Assassin sentries in a band
- [ ] #5 capabilities.json reqLevel/reqSkills filled from repo sources; rest listed U in docs/sme/capability-gaps.md
- [ ] #6 Merc aura literals of act2.go and mercwire.go read from classtraits mercAura
- [ ] #7 Readiness key_skill_min read; mana_heavy honoured in Normal
<!-- AC:END -->
