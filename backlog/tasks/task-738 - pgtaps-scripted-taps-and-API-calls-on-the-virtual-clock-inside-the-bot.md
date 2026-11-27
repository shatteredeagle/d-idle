---
id: TASK-738
title: 'pgtaps: scripted taps and API calls on the virtual clock inside the bot'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-21T04:46:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 344000
card: "pgtaps"
owns:
  - "cmd/d2bot/ladder.go"
  - "cmd/d2scenario/tap.go"
  - "internal/scenario/taps.go"
  - "cmd/d2scenario/run.go"
  - "cmd/d2scenario/tap_test.go"
  - "internal/scenario/result.go"
exclusive: []
contracts: []
result: "Steps 1 and 2 done. Step 1: board tapper polls the bot virtual clock grid (30 virtual s), taps_at recorded. Step 2: d2bot --scripted-taps FILE (scenario.TapScript JSON: quest, chunks, every_s): scenario.TapEngine decides the Start and Hand in taps on a virtual-second grid and presses desk.Start/HandIn from d2denv.LateClock.OnSleep, i.e. on the stepping goroutine, lockstep d2d only; writes FILE.taps.json; d2scenario uses it with D2SCEN_SCRIPTED_TAPS=1 (default stays the HTTP tapper, which stays for sandbox e2e). API calls (trigger.api) stay wall clock. Offline only (fake clock and desk, -race): tap seconds equal across runs of different wall speed. AC 3 (two-run pair equal D1 and Start tick) is NOT ticked: needs the live lab pair. U: that Desk.Start and HandIn are safe called inside a bot Sleep (desk lock vs a Sleep under it)."
lane: now
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Design section 2.8 step 3, 7 card 4. Medium risk: touches the bot answer path. Design: docs/design/fable-review-2026-11-21/proving-ground-gate.md. Owner approved 2026-11-21 (all 10 questions as recommended).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 --scripted-taps file with rules evaluated on the stepping goroutine
- [x] #2 HTTP tapper stays for sandbox e2e, those runs are never stable
- [ ] #3 two-run pair of one board scenario has equal D1 and equal Start tick
- [x] #4 STEP 1: board tapper polls on virtual time (grid), taps_at recorded, wall fallback, tests
<!-- AC:END -->
