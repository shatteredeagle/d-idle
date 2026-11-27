---
id: TASK-1048
title: 'hostilespin: the hostile check cannot spin'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-27T15:18:00Z'
updated_date: '2026-11-27T15:45:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: high
ordinal: 509000
card: "hostilespin"
lane: next
owns:
  - "internal/bot/observe.go"
  - "internal/bot/repairtrip.go"
  - "internal/bot/task.go"
  - "internal/bot/hostilespin_test.go"
  - "internal/combat/friendly_test.go"
exclusive: []
result: "Not a spin (V log + dump, out-pass2/arbiter255/sorceress-201): game time and kills kept advancing to the end (last kill 35 s before exit, pace ratio 1.17); one quest-board slice (40 min budget: loot pickup -> nested yield/Clear/lowMana backoffs) outlasted the 10 min d2bot watchdog, and the dump only caught the guard mid hostile scan. The soak binary was built without loophang de381e7ea (kills/xp refresh the watchdog heartbeat; V grep of its source), which already fixes this exit on main. Hardening here: a hostile scan builds the friendly owner-chain set once instead of once per unit (quadratic, 4224 vs <=192 catalog lookups for 64 units, must-fail test); owner-cycle test (self and mutual owners stay hostile, Friendlies is bounded). Class-agnostic, tests on Sorceress and Paladin. Not verified live."
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Root cause of the arbiter255 Sorceress 201 exit 4 named from the log and goroutine dump
- [x] #2 A hostile scan computes the friendly set once per snapshot (must-fail test counting catalog lookups)
- [x] #3 Owner-cycle test: Friendlies terminates on a self/mutual owner cycle
- [x] #4 Tests on two builds (Sorceress and Paladin)
<!-- AC:END -->
