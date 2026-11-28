---
id: TASK-1115
title: 'fix7b: review-11 defects of fix3/fix7 (arbiter back-off, watchdog heartbeat, survive condition, starve floor, band pin, corpse retry)'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-28T05:43:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: high
ordinal: 533000
card: "fix7b"
lane: next
owns:
  - "internal/bot/"
  - "internal/arbiter/"
  - "internal/survival/"
  - "internal/combat/"
  - "cmd/d2bot/"
  - "docs/arbiter.md"
  - "docs/design/arbiter-target-and-migration.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Review docs/design/fable-review-2026-11-28/11-fix-review-combat-arbiter.md found defects the fix3 and fix7 merges brought (N1 to N6). Done when each has a must-fail test and the fix.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 N1: back-off is per (source, kind, cause); a running or emergency trip, player orders and reflexes are never backed off; farm hooks honour the chain's trip and chores prefix (tests, two classes)
- [ ] #2 N2: the watchdog heartbeat moves only on progress the acting goroutine made; a hung action is detected (hang test, two classes)
- [ ] #3 N3: survive needs a hostile near and never answers a leave verdict with a sweep (tests, two classes)
- [ ] #4 N4: the no-potion floor is never above the limit: act 1 first field fallback (test, two classes)
- [ ] #5 N5: inBand does not pin the hero in the lowest zone (test)
- [ ] #6 N6: an unusable corpse is backed off per corpse id (test, two summoner builds)
- [ ] #7 A4: slice budget against the watchdog documented
- [ ] #8 Live check on the canary with bot.arbiterControl 1: an emergency trip after declined refills goes on; writes after start 0
<!-- AC:END -->
