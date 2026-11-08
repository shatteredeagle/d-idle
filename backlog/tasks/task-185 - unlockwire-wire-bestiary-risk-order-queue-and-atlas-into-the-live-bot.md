---
id: TASK-185
title: 'unlockwire: wire bestiary, risk, order queue and atlas into the live bot'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T11:45:00Z'
labels:
  - bot
  - chore
  - G-C
milestone: m-4
dependencies: []
priority: medium
ordinal: 154000
card: "unlockwire"
owns:
  - "cmd/d2bot/funwire.go"
  - "cmd/d2bot/unlockwire_test.go"
  - "cmd/d2bot/ladder.go"
  - "cmd/d2bot/d2dbackend.go"
  - "cmd/d2bot/live.go"
  - "internal/bot/riskfacts.go"
  - "internal/watch/unlockbestiary.go"
  - "internal/watch/funwire.go"
  - "docs/unlock-ladder.md"
  - "docs/unlock-bestiary.md"
  - "docs/unlock-risk.md"
  - "docs/unlock-orderqueue.md"
  - "docs/unlock-atlas.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Wire the merged unlock features into cmd/d2bot. Done when the tests in cmd/d2bot pass.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 bestiary gets the monster table and persists kill counts
- [ ] #2 risk meter gets live facts from the runner
- [ ] #3 order queue wired to the orders path with halt signals
- [ ] #4 integration tests for Sorceress and Paladin
- [ ] #5 unlock-ladder.md lists all rows
<!-- AC:END -->
