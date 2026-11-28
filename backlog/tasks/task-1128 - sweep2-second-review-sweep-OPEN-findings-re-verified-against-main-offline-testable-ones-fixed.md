---
id: TASK-1128
title: 'sweep2: second review sweep, OPEN findings re-verified against main, offline-testable ones fixed'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-28T08:10:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 535000
card: "sweep2"
lane: now
owns:
  - "docs/probes/review-sweep2-2026-11-28.md"
  - "docs/arbiter.md"
  - "cmd/d2bot/survivalwire.go"
  - "internal/admit/admit.go"
  - "internal/arbiter/arbiter.go"
  - "internal/arbiter/shadow.go"
  - "internal/arbiter/sweep2_test.go"
  - "internal/bot/admission.go"
  - "internal/bot/arbiter_control.go"
  - "internal/bot/earlyroute.go"
  - "internal/bot/questboard.go"
  - "internal/bot/routeguard.go"
  - "internal/bot/sweep2_test.go"
  - "internal/questboard/exec.go"
  - "internal/questboard/fake.go"
  - "internal/questboard/readiness.go"
  - "internal/questboard/sweep2_test.go"
  - "internal/vaultkeep/equip.go"
  - "internal/vaultkeep/equipdirect.go"
  - "internal/vaultkeep/sweep2_test.go"
exclusive: []
contracts: []
result: "Offline only, NOT verified live. docs/probes/review-sweep2-2026-11-28.md gives every OPEN row of the sweep1 doc a status against main after fix1b, vault2, live1 and fix7b (V re-read, U carried over). Fixed with must-fail tests, one commit each: review 13 L2 (a parked boss chunk takes its boss area out of the leveling picks), review 10 N4 (direct Equip refuses an unwritable journal; an order with an unread reply is settled by the look) and N6 (filed once per order in any record state), review 2 finding 8 and review 10 B3 (hand-in reads the evidence after every attempt and parks with the stored counters), review 11 N8 and review 7 admission (road floors before any kit, the kit is an entry rule in admit as in the old judge, words cell counted), review 11 N1c (a declined arbiter hook backs off at most arbiter.DeclineMax 120 s). Tests on Sorceress and Paladin. Build-specific part: none (road floors stay in internal/plandata/admission.json and earlyroutes.json). No knob. Left with reasons in the doc: N8 twin Keep, B1, B4, N7, the admission equality test, L3 nav (needs live grid), 12 C2 to C4."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Re-verify the OPEN rows of docs/probes/review-sweep-2026-11-28.md after fix1b, vault2, live1 and fix7b merged; fix the offline-testable ones. Done when the sweep2 doc is written and each fix has a test.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 every OPEN row of the sweep1 doc has a status against current main in docs/probes/review-sweep2-2026-11-28.md (V re-read or U carried over)
- [x] #2 offline-testable findings fixed, one commit each, with must-fail tests (two classes where class matters)
- [ ] #3 live check of the fixes on a sandbox hero
<!-- AC:END -->
