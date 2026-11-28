---
id: TASK-1119
title: 'live1: sandbox live review L1-L8 (leg fault charged once, hole pick, stairs trap, reward hold, travel budget, starved, vault full, repair)'
status: Proposed
assignee:
  - live-agent
created_date: '2026-11-28T06:04:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: high
ordinal: 534000
card: "live1"
lane: next
owns:
  - "internal/bot"
  - "internal/survival"
  - "internal/nav"
  - "internal/questboard"
  - "internal/arbiter"
  - "internal/town"
  - "internal/items"
  - "docs/live1.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 L1 a failed leg is charged once, the repeat leaves by town portal (test, two classes)
- [ ] #2 L2 a hole never travels to a margin-low or hardest zone (test)
- [ ] #3 L3 exit-trigger start cell exempt and same-cell retries are one try (tests)
- [ ] #4 L4 reward hold derives from the board claim, log rate-limited, no passing claim (test, two classes)
- [ ] #5 L5 travel budget accepts the quest step after a finished town trip (test)
- [ ] #6 L6 arbiter-starved silent on a training step (test)
- [ ] #7 L7 vault full: fixed or classified against docs/vault.md
- [ ] #8 L8 repair before 0 percent: fixed or classified (test)
- [ ] #9 live check of L1-L4 on a sandbox hero
<!-- AC:END -->
