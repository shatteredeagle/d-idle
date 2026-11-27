---
id: TASK-1063
title: 'classkeys: main-attack keys for every class, profile skills by name, merc aura note'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-27T20:14:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 512000
card: "classkeys"
lane: next
owns:
  - "internal/combat/classdata"
  - "internal/combat/profileskills_test.go"
  - "internal/plandata"
  - "docs/plan/data"
  - "cmd/d2bot/necroplay_test.go"
  - "cmd/d2bot/hotkeydrift_test.go"
  - "data/capabilities/capabilities.json"
  - "tools/provingground/scenarios/necro-epic.json"
  - "docs/design/merc-aura-by-archetype.md"
exclusive: []
contracts: []
result: "Not verified live (offline tests only). New combat profiles druid/tornado, barbarian/warcry, amazon/javelin, paladin/hammerdin_zeal (table combat key set on those plan tables, hammerdin tables named hammerdin); Amazon default layout gains Guided Arrow f7 and Dopplezon f8. Table hotkeys now equal the profile layout for Amazon, Barbarian, Druid and hammerdin (ratchet entries removed; rows with no key dropped: Iron Skin, Grim Ward, Hurricane, Summon Grizzly, Fanaticism). TestMainAttackSkillsHaveAKey covers every class. New TestProfileSkillsExistForTheirClass (names in capabilities.json for the profile class; fixed Fire Trauma class druid to assassin there). Skill ids 73/84 in necro-epic.json are tagged S/V from docs/research; skill_points takes ids only, no name mechanism. Merc aura note: docs/design/merc-aura-by-archetype.md. Build-specific data: internal/combat/classdata/profiles.json and the plan tables."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Every class's plan tables pass TestMainAttackSkillsHaveAKey and the hotkey ratchet is zero for Amazon, Barbarian, Druid, hammerdin
- [x] #2 Every skill a profile names exists in the skills table for its class (test)
- [x] #3 Merc aura by archetype design note
<!-- AC:END -->
