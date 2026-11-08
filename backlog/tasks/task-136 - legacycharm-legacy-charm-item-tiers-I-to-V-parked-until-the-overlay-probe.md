---
id: TASK-136
title: 'legacycharm: legacy charm item tiers I to V (parked until the overlay probe)'
status: Deferred
assignee:
  - credit-runner
created_date: '2026-11-08T00:27:00Z'
labels:
  - d2d
  - feature
  - ladder
  - parked
milestone: m-4
dependencies:
  - TASK-135
  - TASK-59
priority: low
ordinal: 128000
card: "legacycharm"
owns:
  - "docs/research/legacy-charm.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: ADR 0018 item 7, fun-ideas.md section 13 later. Parked: starts only after the overlay probe proves the charm row in d2d (TASK-59) and the user approves game.exe use. Permanent stat points, +skills, XP bonuses, level head starts and inherited uniques are never built. Done when: the criteria below hold.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The overlay probe proves the charm row (Misc code) loads in d2d, and the user has approved the game.exe side
- [ ] #2 Tier I to V charm (flat life, mana, resistances, modest magic find; SME values UNVERIFIED) granted at hero creation from the account milestones; if the probe fails, the grant path only
<!-- AC:END -->
