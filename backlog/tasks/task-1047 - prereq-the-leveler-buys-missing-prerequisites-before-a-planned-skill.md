---
id: TASK-1047
title: 'prereq: the leveler buys missing prerequisites before a planned skill'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-27T14:33:00Z'
updated_date: '2026-11-27T14:52:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 508000
card: "prereq"
lane: next
owns:
  - "internal/leveler/"
  - "internal/points/autoalloc.go"
  - "internal/points/autoalloc_test.go"
  - "cmd/d2bot/autoallocwire.go"
exclusive: []
contracts: []
result: "Not verified live (offline tests only). The planner buys the missing prerequisites of a planned skill first, from the skills.txt reqskill columns, and marks every prerequisite point; the auto allocator spends a marked point even when the capability table calls the skill an unsupported attack (Sacrifice before Zeal), logs a held skill once, and logs a blocked planned skill once with its blocker. Tested on Paladin (Sacrifice then Zeal) and Necromancer (Teeth, Corpse Explosion, Bone Spear). Nothing build-specific: prerequisites come from skills.txt. closecards3 Remaining (live): #4 need the live game, a browser or a worker-host run; not verified here."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: live 2026-11-27 Doofer (zealot-lite, clvl 14) never learned Sacrifice: the capability table marks it unsupported, so the auto allocator held it for a confirmation that never came, Zeal (needs Sacrifice) was refused and the points went to later Might rows. Done when prerequisites are bought first and spent without the confirmation.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A planned skill whose prerequisites are not owned gets them bought first (skills.txt reqskill data), marked as prerequisites
- [x] #2 The allocator spends a prerequisite without the unsupported-attack confirmation and logs a held skill or a blocked row once with its blocker
- [x] #3 Tests: Paladin missing Sacrifice buys Sacrifice then Zeal; Necromancer Bone Spear without Teeth buys the chain
- [ ] #4 Live: Doofer learns Sacrifice and Zeal after deploy and logs last_skill=Zeal at kills
<!-- AC:END -->
