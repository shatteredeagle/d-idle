---
id: TASK-173
title: "pactnightmare: the Pact of Nightmares, same hero into Nightmare after Baal in Normal"
status: Deferred
assignee:
  - credit-runner
created_date: '2026-11-08T03:34:00Z'
updated_date: '2026-11-10T16:54:00Z'
labels:
  - bot
  - feature
  - parked
milestone: m-4
dependencies:
  - TASK-166
  - TASK-169
  - TASK-170
  - TASK-94
priority: low
ordinal: 149000
card: "pactnightmare"
owns:
  - "internal/pact/nightmare.go"
  - "internal/pact/nightmare_test.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Size: L. Parked: needs the bot past Act 1. Not on the two-week playable path. Design note: docs/research/pact-system.md (a design note, not a decision).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Unlock on Baal killed in Normal; same hero moves to Nightmare with the Pact bonus
- [ ] #2 Nightmare entry warnings shown only with verified numbers; away off after the move
- [ ] #3 Blocked until the bot plays past Act 1 (class and act coverage)
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Unlocks at Baal killed in Normal; same hero; ONE Pact (user 2026-11-07, ADR 0019).

Parked in the 2026-11-10 true-up: first Pacts need Andariel killed on a live hero.
<!-- SECTION:NOTES:END -->
