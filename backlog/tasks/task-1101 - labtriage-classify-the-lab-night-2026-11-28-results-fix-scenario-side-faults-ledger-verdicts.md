---
id: TASK-1101
title: 'labtriage: classify the lab night 2026-11-28 results, fix scenario-side faults, ledger verdicts'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-28T05:18:00Z'
labels:
  - docs
  - research
dependencies: []
priority: medium
ordinal: 525000
card: "labtriage"
lane: next
owns:
  - "docs/probes/lab-night-results-2026-11-28.md"
  - "tools/provingground/scenarios/combatdrills.json"
  - "tools/provingground/scenarios/townplan.json"
  - "backlog/tasks/task-1102 - navbarracks-no-route-from-the-Barracks-up-to-the-Outer-Cloister-when-the-hero-starts-in-the-Barracks.md"
  - "backlog/tasks/task-1103 - cainspin-cain.chain-seed-11-with-bot.portalRetryWaitSec-0-spins-on-run-finished-and-is-cut-by-the-wall.md"
  - "backlog/tasks/task-1104 - summonerstall-drill.act2.summoner.stepexec-ends-in-the-runner-watchdog-after-a-nav-stuck-loop.md"
  - "backlog/tasks/task-1105 - drillskills-combat-and-necro-drills-with-hero_profile-played-run-without-the-builds-skills-no-leveler-line-last_skill-Attack.md"
  - "backlog/tasks/task-1106 - bsmnight-board.sm.restart-mid-chain-and-stop-cascades-RED-on-both-classes-assertion-and-trigger-timing-against-the-log.md"
  - "backlog/tasks/task-1107 - act2night-three-Act-2-stepexec-drill-findings-Duriel-stall-shaft-road-without-a-portal-potion-not-drunk-in-the-cube-drill.md"
  - "backlog/tasks/task-1108 - townnight-town.plan-stage-4-and-5-REDs-of-the-lab-night-route-kept-the-list-Charsi-talk-not-credited-scroll-before-potions.md"
exclusive: []
contracts: []
result: "Not verified live (no game run by this card; evidence is the three lab jobs of the night on bot 2f2e092c3). 37 non-GREEN rows classified in docs/probes/lab-night-results-2026-11-28.md: 12 product bugs, 12 wrong scenario expectations, 11 harness or setup, 2 expected RED. Scenario fixes: combat drill log markers (sorted attributes), town.plan.sell-keeps-need and short-gold-handover assertions; d2scenario validate ok. Cards TASK-1102 to TASK-1108. No dark-ledger row has its criteria met. Nothing build-specific changed."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Triage of the three Proving Ground jobs of the lab night. Done when the results doc is merged. Consumer: townflip, bsmnight, navbarracks, drillskills.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Every non-GREEN row of the three jobs is classified with log evidence
- [x] #2 Scenario-side faults fixed, d2scenario validate ok
- [x] #3 Cards written for the product bugs
- [x] #4 Dark-ledger verdicts per row with the evidence
<!-- AC:END -->
