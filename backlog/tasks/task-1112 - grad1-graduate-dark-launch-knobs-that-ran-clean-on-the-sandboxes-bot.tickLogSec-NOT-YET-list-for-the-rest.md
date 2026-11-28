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
  - "internal/bot"
  - "internal/env"
  - "data/darklaunch.json"
  - "docs/knobs.md"
  - "docs/probes/graduations-2026-11-28.md"
  - "docs/design/ticklog-t4.md"
  - "docs/arbiter.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: user policy 2026-11-27, a feature that ran clean on a sandbox graduates (default on, knob deleted, ledger row closed). Done when: every candidate row has a verdict and the graduations are one tested commit each.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Each candidate row decided GRADUATE or NOT YET with cited evidence and the ledger criteria text
- [ ] #2 Each graduation: default flipped, knob deleted, key in env.RetiredKnobs, ledger row enabled with graduated_at, docs/knobs.md regenerated, test that a tuning.json naming the key loads
- [ ] #3 docs/probes/graduations-2026-11-28.md lists the NOT YET rows and what each waits for
<!-- AC:END -->
