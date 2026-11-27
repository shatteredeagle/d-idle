---
id: TASK-709
title: >-
  boardprereq: accepting a chunk accepts its unmet prerequisite chain, a lone
  later record is repaired at board load
status: Done
assignee:
  - credit-runner
created_date: '2026-11-21T02:55:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 340000
card: "boardprereq"
owns:
  - "internal/questboard"
  - "tools/provingground/scenarios/cain.json"
exclusive: []
contracts: []
result: "Offline fix: Accept of a chunk locked only by After steps accepts the chain in questspec order; Next repairs a lone later record (log: quest board: stones waits for scroll (auto-started)); Stop withdraws dependants with a toast. Unit tests Paladin+Sorceress green, -race x20 green. Scenarios board.prereq-not-accepted and board.prereq-start-tap queued on the builder long lane, NOT run when this card was written. Not verified live."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: sandbox 2026-11-20: stones accepted, scroll step stopped; the board parked for ever. Done when: the board starts the prerequisite chain itself.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 accept of a chunk with unmet After steps accepts the chain in questspec order; a lone later record is repaired at board load with a log line
- [x] #2 Stop on a prerequisite step withdraws its dependants and the toast says so
- [x] #3 unit tests on Paladin and Sorceress; single chunk unchanged
- [ ] #4 scenarios board.prereq-not-accepted and board.prereq-start-tap run on the builder
<!-- AC:END -->
