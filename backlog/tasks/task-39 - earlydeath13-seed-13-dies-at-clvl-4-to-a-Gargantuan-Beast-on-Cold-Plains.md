---
id: TASK-39
title: 'earlydeath13: seed 13 dies at clvl 4 to a Gargantuan Beast on Cold Plains'
status: In Progress
assignee:
  - pro-runner
created_date: '2026-11-06T16:02:00Z'
updated_date: '2026-11-06T21:56:00Z'
labels:
  - bot
  - bug
  - gate
milestone: m-0
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
Size: M

docs/research/g2e-diagnosis.md E3: death at clvl 4, identical in g2d, g2e and g2f via lockstep replay (deterministic). A bot defect, not noise.

Done when: Replay the death offline or on the lockstep binary; record the state before the fight (life, mana, skills,...; Root cause named (target choice, retreat threshold, kit or route guard) with a proposed fix and a test...; A rerun of seed 13 passes the clvl 4 stretch; result says verified live or not and writes after start (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Replay the death offline or on the lockstep binary; record the state before the fight (life, mana, skills, belt, merc, position) and the bot decision chain, tagged V or U
- [x] #2 Root cause named (target choice, retreat threshold, kit or route guard) with a proposed fix and a test that fails today; fix card or fix included
- [ ] #3 A rerun of seed 13 passes the clvl 4 stretch; result says verified live or not and writes after start
<!-- AC:END -->

result: offline tested only, not verified live (AC 3 stays open; the proof is the next Gate 2 with seed 13 passing clvl 4-5; writes after start: n/a, no game run). Cause path (V by source and the g2f autopsy): at clvl 4 a caster is under the Den floor (campaign.denMinLevelCaster 5), Blood Moor was held back, so Pick fell to the first eligible step, Cold Plains (enter 5, kit_clvl 4, earlyGap 2 let a one-level gap through). Fix: plan field early_gap (Cold Plains 1) in progression-normal.json (both copies), stale-pack hold (bot.stalePack*), potion-burst retreat (bot.potionBurst*). AC1 replay: the state chain is taken from docs/research/g2f-diagnosis.md 4.1, not replayed on the lockstep binary. Status sync 2026-11-07: stays In Progress, AC 3 open; not superseded by the early-death fixes (merged) because seed 13 still failed NOKILL in Gate 2 g2g; needs a seed 13 rerun.
