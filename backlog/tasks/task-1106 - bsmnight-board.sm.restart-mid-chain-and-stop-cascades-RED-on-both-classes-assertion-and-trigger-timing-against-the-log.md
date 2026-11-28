---
id: TASK-1106
title: 'bsmnight: board.sm.restart-mid-chain and stop-cascades RED on both classes: assertion and trigger timing against the log'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-28T05:19:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 530000
card: "bsmnight"
lane: next
owns:
  - "tools/provingground/scenarios/board.json"
  - "internal/questboard/"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

restart-mid-chain: 'quest board: step Take the Inifuss bark' is logged again each time the chunk is re-entered after a town trip (Paladin 23:00:17 and 23:01:18 after 'resupplying', one process, 2 lines; Sorceress 5 lines and the chunk still in_progress at the end with mp 0 to 6 of 35), so 'max 1' fails without a repeated step. U: whether the restart itself happened (one 'd2bot start' in the kept log). stop-cascades: the api trigger after_s 60 fired when scroll was already done (chunk started 22:59:56, 'awaiting hand-in of search_for_cain/read' 23:00:15, stones started 23:00:16): reply 409 code done, nothing to cascade. board.park.persist-restart stayed PENDING. Evidence: docs/probes/lab-night-results-2026-11-28.md (lab night 2026-11-28, bot 2f2e092c3, not live-verified beyond the lab run).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 restart-mid-chain asserts a line that is logged once per completed step, or the step line is logged once per run of the step
- [ ] #2 stop-cascades cancels while search_for_cain/scroll is still in progress (trigger tied to a board state, not to wall seconds)
- [ ] #3 Both GREEN for Paladin 11 and Sorceress 3, then the quests.boardSM ledger row is re-read
<!-- AC:END -->
