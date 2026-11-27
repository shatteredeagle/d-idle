---
id: TASK-532
title: 'intake: observed damage intake (lifeLost per minute) in minute line, glance and ledger'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-16T02:57:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 284000
card: "intake"
owns:
  - "internal/bot/hpfields.go"
  - "internal/survival/ledger.go"
  - "internal/watch/glance.go"
  - "tools/levelsim/shadow.go"
  - "docs/survival/ledger.md"
  - "internal/bot/observe.go"
  - "internal/bot/areametrics.go"
  - "internal/bot/zoneshadow.go"
  - "internal/d2dsoakstat/minute.go"
  - "cmd/d2bot/glancewire.go"
  - "cmd/d2bot/survivalwire.go"
exclusive: []
contracts: []
result: "minute line lifeLost/hitsTaken/hpPolls, glance intake, ledger intakeLife per zone, levelsim and d2dsoakstat readers; offline tests only, not verified live Closed by closecards2: merged, ACs verified against the merge diff and tests."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 minute line carries lifeLost, hitsTaken, hpPolls; max-life changes never count
- [x] #2 /api/glance intake {lifeLostPerMin, window}; ledger per-zone intake; levelsim -shadow reads lifeLost
- [x] #3 docs/survival/ledger.md section Observed intake; gate green
<!-- AC:END -->
