---
id: TASK-1090
title: >-
  farmresetretire: farm loop always on (flip bot.farmReset to default, remove
  the gate)
status: Done
assignee:
  - credit-runner
created_date: '2026-11-28T02:48:00Z'
updated_date: '2026-11-28T03:24:00Z'
labels:
  - bot
  - chore
dependencies: []
priority: medium
ordinal: 522000
card: "farmresetretire"
lane: next
owns:
  - "internal/bot"
  - "internal/env"
  - "internal/watch"
  - "cmd/d2bot"
  - "tools/uishots"
  - "data/darklaunch.json"
  - "docs/knobs.md"
  - "docs/farmloop.md"
  - "docs/questboard.md"
exclusive: []
result: "DECISION: FLIP-TO-DEFAULT, not retire. The premise of docs/probes/dark-queue-final-2026-11-27.md (retire-farmreset: 'farm reset never fired', 45 min) is stale: V docs/probes/sandbox-postdeploy-audit-2026-11-27.md shows resets firing on both sandbox heroes (farmreset.go:291, 85% cleared) with 0 deaths and 0 chickens in 70 min; the knob has been 1 in the sandbox tuning file since 2026-11-16 (V docs/probes/farmloop-live-20261116.md), and the thrash of 2026-11-15 was fixed by the farmloop2 guards (still in force: cooldown, cap 3 per 30 min, 180 s hold). Modelled gain is small (2.5 percent, S docs/farmloop.md section 3), so the value rule stays the gate; xp/h effect on the live heroes is not measured (U). Done: FarmReset field and knob row deleted, all gates always on (farmreset.go, FarmResetShare), viewer zone-clear panel kept for exploredPct, farmResetPct and cooldown (on/off toggle and the inverted order rule removed), ledger row enabled, docs/knobs.md regenerated. A tuning.json that still names bot.farmReset loads: the key is in env.RetiredKnobs (test TestRetiredFarmResetKeyLoads), so no sandbox edit is needed. Not verified live."
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Decision recorded with evidence
- [x] #2 Knob gate and viewer toggle removed, behaviour at default equals knob=1 today
- [x] #3 Ledger row enabled, docs/knobs.md regenerated
<!-- AC:END -->
