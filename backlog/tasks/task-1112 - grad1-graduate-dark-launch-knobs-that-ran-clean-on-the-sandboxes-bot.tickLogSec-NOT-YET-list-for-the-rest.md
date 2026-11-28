---
id: TASK-1112
title: 'grad1: graduate dark-launch knobs that ran clean on the sandboxes (bot.tickLogSec), NOT YET list for the rest'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-28T05:33:00Z'
labels:
  - bot
  - chore
dependencies: []
priority: medium
ordinal: 533000
card: "grad1"
lane: now
owns:
  - "internal/bot/ticklog.go"
  - "internal/bot/ticklog_test.go"
  - "internal/bot/shadow.go"
  - "internal/bot/broke_test.go"
  - "internal/env/tuning.go"
  - "internal/env/tuning_bot.go"
  - "data/darklaunch/bot.tickLogSec.json"
  - "docs/knobs.md"
  - "docs/probes/graduations-2026-11-28.md"
  - "docs/design/ticklog-t4.md"
  - "docs/arbiter.md"
  - "internal/arbiter/replay.go"
  - "cmd/d2scenario/arbreplay.go"
exclusive: []
result: "GRADUATED 1 of 10 candidates: bot.tickLogSec (tick line always on at a 1 s gap, knob and field deleted, key in env.RetiredKnobs, ledger row data/darklaunch/bot.tickLogSec.json set default-on with ledger.py status, graduation date in its note, docs/knobs.md regenerated, TestRetiredTickLogSecKeyLoads). Evidence V: both sandbox logs of 2026-11-27 22:59Z to 2026-11-28 05:31Z replay through cmd/d2arbiter and -follow, exit 0, panics=0 (838 and 902 ticks). NOT YET (docs/probes/graduations-2026-11-28.md): bot.zoneModel (the A/B criteria have a cited Sorceress FAIL, ratio 0.497, holdout 5; both sandboxes are Paladins), quests.stepExec (lab night NOT met), bot.arbiterShadow (starved WARN noise, lab night NOT met), combat.deliveryExpect (mode 1 log unreviewed, 1727 Zeal demotions), combat.missileDodge, combat.threatDodge, bot.checkpointEnable, bot.portalGuard (0 lines in 6.5 h: not exercised), combat.standOK (no log line). Nothing build-specific. Not verified live after the change."
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: user policy 2026-11-27, a feature that ran clean on a sandbox graduates (default on, knob deleted, ledger row closed). Done when: every candidate row has a verdict and the graduations are one tested commit each.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Each candidate row decided GRADUATE or NOT YET with cited evidence and the ledger criteria text
- [x] #2 Each graduation: default flipped, knob deleted, key in env.RetiredKnobs, ledger row enabled with graduated_at, docs/knobs.md regenerated, test that a tuning.json naming the key loads
- [x] #3 docs/probes/graduations-2026-11-28.md lists the NOT YET rows and what each waits for
<!-- AC:END -->
