---
id: TASK-1096
title: 'fix7: arbiter A1-A4 fixes, admission differ lines, target and migration doc'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-28T03:45:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 524000
card: "fix7"
lane: next
owns:
  - "internal/bot/"
  - "internal/arbiter/"
  - "internal/admit/"
  - "cmd/d2bot/"
  - "docs/design/arbiter-target-and-migration.md"
  - "docs/arbiter.md"
  - "docs/probes/"
exclusive: []
contracts: []
result: "Offline only, NOT verified live. A1: controlled recovery and gold farm pass lowLifeGate. A3: controlled hooks call Report; a declined hook and a recovery or gold farm choice no longer hold the quest step. A2: survive reads the fact safety (Fighter.Safety) and yields to an emergency trip only (a plain chores trip still yields to danger, incident questy-0251). A4: observed progress (walking, gold, potions) is the watchdog heartbeat; the slice budget is open. Admission: the old road judge asks every row (Duriel caster life 220 enforced), lean is exempt under try, the shadow counts agreeRefused, oldOnly, newOnly. Nothing build-specific: floors are data in internal/plandata/admission.json. Docs: docs/design/arbiter-target-and-migration.md, docs/probes/arbiter-families-note-2026-11-27.md. No knob changed."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: fable review part 7 (docs/design/fable-review-2026-11-28/07-arbiter-and-deciders.md) found two safety paths the arbiter control path can starve (A1, A2), an open loop (A3), a 40 min slice against the 10 min watchdog (A4) and two real differences between the old admission judges and admit.MayEnter. Done when: each has a must-fail test on two classes, the target and migration doc and the data note exist, and no sandbox knob was changed.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A1: controlled recovery and gold farm honour the low-life gate (test, two classes)
- [x] #2 A3: Report wired from the controlled hook outcomes; a declined hook does not veto a chunk the chain allows (tests)
- [x] #3 A2: survive reads Fighter.Safety and yields to a running emergency town trip (tests)
- [x] #4 A4: slice heartbeat reaches the watchdog; documented
- [x] #5 admission: lean under try and area 73 fixed; shadow reports differ lines
- [x] #6 docs/design/arbiter-target-and-migration.md and the data note written
- [ ] #7 live: one control-0 recorded night scored before any family bit beyond 14
<!-- AC:END -->
