---
id: TASK-766
title: >-
  countess3b: boss presets on d2d from the navigator's world; Countess found,
  nav mouth gate blocks the walk
status: Done
assignee:
  - credit-runner
created_date: '2026-11-21T04:59:00Z'
updated_date: '2026-11-27T20:31:00Z'
labels:
  - d2d
  - bug
dependencies: []
priority: medium
ordinal: 341000
card: "countess3b"
owns:
  - "cmd/d2bot/boss.go"
  - "cmd/d2bot/boss_levelmonsters_test.go"
  - "internal/nav/nav.go"
  - "docs/probes/provingground-countess-2026-11-20.md"
  - "docs/probes/countess-nav-fixture.md"
  - "internal/nav/cellar5walk_test.go"
exclusive: []
contracts: []
lane: next
result: "Offline regression cellar5walk_test.go (2 synthetic maps, must-fail pair); the gate was already fixed by later commits, no gate change. Recorded Tower Cellar 5 grid and the live Paladin check stay live-only (docs/probes/countess-nav-fixture.md). Not verified live. Remaining (live): AC 2 Paladin 11 with the new fork binary."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 boss presets reach the script on d2d (test fails without the fix)
- [x] #3 offline: Tower Cellar 5 walk planned on two corridor maps, unwanted-exit crossing still refused (cellar5walk_test.go, synthetic maps; recorded grid is live-only)
- [ ] #2 Paladin 11 with the new fork binary: Countess located, walk reaches her (live check)
<!-- AC:END -->
