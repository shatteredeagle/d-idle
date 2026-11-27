---
id: TASK-538
title: >-
  farmloop2: farm loop thrash guards (town no-op, cooldown from the fresh game,
  thrash cap, guarded startFreshGame, noTarget forgets the game)
status: Done
assignee:
  - credit-runner
created_date: '2026-11-16T03:40:00Z'
updated_date: '2026-11-16T06:03:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 283000
card: "farmloop2"
owns:
  - "internal/bot/farmreset.go"
  - "internal/bot/farmreset_test.go"
  - "cmd/d2bot/glancewire_farmoff.go"
  - "cmd/d2bot/glancewire_farmoff_test.go"
exclusive: []
contracts: []
result: "Guards added and unit-tested offline (two classes); culprit candidates named from the code (noTarget without forgetGame, stale nav coverage), the real emitter not provable without the log: every leave now logs new game requested from=file:line; not verified live; default stays off closecards3 Remaining (live): #3 need the live game, a browser or a worker-host run; not verified here."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Reset trigger never fires in town or outside the zone; state resets after any fresh game
- [x] #2 Thrash cap, cooldown from the fresh game, guarded startFreshGame, noTarget forgets the game
- [ ] #3 Live check: loop on in the sandbox with the new-game-requested log (not run)
<!-- AC:END -->
