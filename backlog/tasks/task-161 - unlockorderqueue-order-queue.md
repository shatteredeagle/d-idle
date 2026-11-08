---
id: TASK-161
title: 'unlockorderqueue: order queue'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T03:16:00Z'
labels:
  - watch
  - feature
milestone: m-4
dependencies:
  - TASK-153
priority: low
ordinal: 139000
card: "unlockorderqueue"
owns:
  - "internal/watch/unlockorderqueue.go"
  - "internal/watch/unlockorderqueue_test.go"
  - "internal/watch/orderqueue.json"
  - "internal/watch/funwire.go"
  - "internal/watch/testdata/orderqueue-*.json"
  - "data/unlocks/ladder.json"
  - "docs/unlock-orderqueue.md"
exclusive: []
contracts: []
result: "Offline proof 2026-11-08: go test ./internal/watch ./internal/unlock and ./cmd/d2bot (-run OrderQueueWired|PactHalt) pass; slots, run in turn, stop on death, halt, charter and buy flows, post gates (TestOrderQueue* in internal/watch/unlockorderqueue_test.go, TestOrderQueueWiredThroughTheOrdersPath, TestPactHaltWithoutController). Fixtures only; NOT verified live. Stays open: AC #6 (live run) unticked; the queue is in memory only and the live quest reader is U."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Unlock: Den of Evil cleared once; queue 2 or 3 orders, one slot per town service reached. Not on the two-week playable path. Design note: docs/research/unlockables.md (a brainstorm, not a decision). Technical check: check Den of Evil clear conditions against the data; orders must add agency and not make the watcher redundant.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 ladder rows orderqueue (Den of Evil cleared), orderqueue2 and orderqueue3 exist; slots open in turn and stay locked until the trigger (TestOrderQueueLockedUntilTheDenAndSlotsOpenInTurn, two classes)
- [x] #2 orders run one at a time through the normal placer and the player's own orders go first (TestOrderQueueRunsOrdersInTurn, TestOrderQueueCancelsAndPlayersOrdersGoFirst)
- [x] #3 safety: a death stops the queue and it resumes on request; Halt (Pact pause) stops it and a guard refusal goes on to the next order (TestOrderQueueStopsOnDeathAndResumes, TestOrderQueueHaltStopsAndGuardRefusalGoesOn)
- [x] #4 API: GET/POST /api/orderqueue with the answer-route gates and bad-body errors; slot purchase through the ledger once (TestOrderQueuePostGatesAndBadBodies, TestOrderQueueBuySlotThroughTheLedgerOnce, TestOrderQueueCharterFlowAndKey)
- [x] #5 cmd/d2bot wiring: Quote, Halt = Pact Paused, death stop, slot rows from quest events (TestOrderQueueWiredThroughTheOrdersPath, TestPactHaltWithoutController, TASK-185)
- [ ] #6 LIVE check on a real run: Den clear opens slot 1, a queued order is placed and run (not run; the live panels check is pending)
<!-- AC:END -->
