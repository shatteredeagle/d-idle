---
id: TASK-1115
title: >-
  fix7b: review-11 defects of fix3/fix7 (arbiter back-off, watchdog heartbeat,
  survive condition, starve floor, band pin, corpse retry)
status: Done
assignee:
  - credit-runner
created_date: '2026-11-28T05:43:00Z'
updated_date: '2026-11-28T08:47:00Z'
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
result: "Offline only, NOT verified live. N1: arbiter back-off is per (source, kind, cause) and never applies to a running or emergency trip, a player order or a reflex; the gold farm and recovery hooks run a running trip and due chores first. N2: the d2bot heartbeat is observer.beat (kill, xp, level, area change, 20 subtiles walked, gold gained or spent in town, a pickup), not the guard's reflex drink or monster life; hang test. N3: survive needs a drink verdict with a hostile near; leave verdicts stay with the guard. N4: the no-potion floor of a hero in a later act's town is act 1's first field, reached by the town waypoint (wpActTown, U live). N5: inBand keeps the below-band zones when only the current zone below the band is safe. N6: a corpse is aimed at twice per area. A4: interplay documented, no slice budget (no data row). Nothing build-specific; no knob changed. Open: N7, N8, a cap for decline back-off, review 11 section 2."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Review docs/design/fable-review-2026-11-28/11-fix-review-combat-arbiter.md found defects the fix3 and fix7 merges brought (N1 to N6). Done when each has a must-fail test and the fix.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 N1: back-off is per (source, kind, cause); a running or emergency trip, player orders and reflexes are never backed off; farm hooks honour the chain's trip and chores prefix (tests, two classes)
- [x] #2 N2: the watchdog heartbeat moves only on progress the acting goroutine made; a hung action is detected (hang test, two classes)
- [x] #3 N3: survive needs a hostile near and never answers a leave verdict with a sweep (tests, two classes)
- [x] #4 N4: the no-potion floor is never above the limit: act 1 first field fallback (test, two classes)
- [x] #5 N5: inBand does not pin the hero in the lowest zone (test)
- [x] #6 N6: an unusable corpse is backed off per corpse id (test, two summoner builds)
- [x] #7 A4: slice budget against the watchdog documented
- [ ] #8 Live check on the canary with bot.arbiterControl 1: an emergency trip after declined refills goes on; writes after start 0
<!-- AC:END -->
