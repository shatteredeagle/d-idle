---
id: TASK-1105
title: 'drillskills: combat and necro drills with hero_profile played run without the build''s skills (no leveler line, last_skill Attack)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-28T05:19:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 529000
card: "drillskills"
lane: next
owns:
  - "internal/scenario/"
  - "cmd/d2scenario/"
  - "tools/provingground/scenarios/combatdrills.json"
  - "tools/provingground/scenarios/necro.json"
exclusive: []
contracts: []
result: "NOT fixed. Premise corrected (V, lab results json of run 20261128-000130): the drill heroes DO learn skills (skillsspent 23 to 30, skill_levels set, Zeal 5, Whirlwind 1, hero_casts show Blessed Hammer, Bash, Teeth); the played hero spends points via cmd skillpt, so no leveler line is expected. The RED is the attack binding: the library binds the build's main attack (Barbarian left skill Bash, Paladin Blessed Hammer, Necromancer right Teeth/Bone Spear) while the drills expect Whirlwind, Zeal, Corpse Explosion; minions 0 for ce1/ce5 because setup casts none. Next: bind the drill's skill in trigger or pre, and give the necro setup a summon."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Ten RED rows of job 3 share one cause candidate. In all of them the bot log has no 'leveler: plan', 'stats: allocated' or 'skills: allocated' line (the nav runs with hero_profile banked have them), 'class-aware start' says combatBuild=blizzard for Barbarian and Paladin, every 'combat kill' carries last_skill=Attack (Paladin), there is no 'combat move', 'combat corpse-explosion' or 'combat minion' line, and minlife stays 100 in every fight (Fallen do not hurt a clvl 24 to 30 hero, so escape and potion triggers cannot fire either). The Necromancer read 'debug minions' (journal) and had 0 minions at the end. cmd/d2scenario/smokegame.go says 'debug points is accepted and changes nothing (U)'. U: whether setup spends the points and the bot then has nothing to allocate, or neither does. The marker patterns were fixed in labtriage (sorted attributes). Encounter note: use monsters that hurt at the hero's level. Evidence: docs/probes/lab-night-results-2026-11-28.md (lab night 2026-11-28, bot 2f2e092c3, not live-verified beyond the lab run).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A combat drill log shows the skills of trigger.build learned before the first fight (leveler or setup line)
- [ ] #2 combat.whirlwind-clears, combat.hold-interruptible, combat.teleport-reposition, combat.corpse-explodes and necro.corpse-ce1/ce5 rerun; each verdict is about the driver, not the setup
- [ ] #3 Tested for two classes
<!-- AC:END -->
