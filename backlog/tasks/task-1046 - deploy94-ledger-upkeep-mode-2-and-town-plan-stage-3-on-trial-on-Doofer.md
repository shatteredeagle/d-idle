---
id: TASK-1046
title: 'deploy94-ledger: upkeep mode 2 and town plan stage 3 on trial on Doofer'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-27T13:40:00Z'
updated_date: '2026-11-27T14:26:00Z'
labels:
  - ops
  - chore
dependencies: []
priority: medium
ordinal: 507000
card: "deploy94-ledger"
lane: next
owns:
  - "data/darklaunch.json"
exclusive: []
contracts: []
result: "verified live on sandbox Doofer: deploy 94 (main 4d2f020b7, release 0.0.261127133829), combat.deliveryUpkeep 2 + town.plan 3 flipped 13:40Z, 45 min window closed clean (0 restarts, 0 errors, 0 deaths, no hard breach); rows kept on trial"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Owner approved 2026-11-27 08:35 EST: deploy 94 (main 4d2f020b7) on sandbox Doofer with combat.deliveryUpkeep 2 and town.plan 3 in one flip bundle (flip.py, 45 min window, auto-revert). Done when the ledger rows say trial and the watch closed clean.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 data/darklaunch.json rows combat.deliveryUpkeep and town.plan set to trial (owner approved 2026-11-27)
- [x] #2 45 min watch on sandbox Doofer clean (no hard invariant breach, no revert)
<!-- AC:END -->
