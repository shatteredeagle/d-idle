---
id: TASK-39
title: 'earlydeath13: seed 13 dies at clvl 4 to a Gargantuan Beast on Cold Plains'
status: Proposed
assignee: []
created_date: '2026-11-06T16:02:00Z'
labels:
  - bot-defect
dependencies: []
priority: high
ordinal: 39000
card: "earlydeath13"
owns:
  - "docs/research/earlydeath13.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
docs/research/g2e-diagnosis.md E3: death at clvl 4, identical in g2d, g2e and g2f via lockstep replay (deterministic). A bot defect, not noise.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Replay the death offline or on the lockstep binary; record the state before the fight (life, mana, skills, belt, merc, position) and the bot decision chain, tagged V or U
- [ ] #2 Root cause named (target choice, retreat threshold, kit or route guard) with a proposed fix and a test that fails today; fix card or fix included
- [ ] #3 A rerun of seed 13 passes the clvl 4 stretch; result says verified live or not and writes after start
<!-- AC:END -->
