---
id: TASK-688
title: 'k1layout: level layout read verb consumer (quests.layoutRead)'
status: Done
assignee:
  - live-agent
created_date: '2026-11-20T14:49:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - d2d
  - feature
dependencies: []
priority: medium
ordinal: 335000
card: "k1layout"
owns:
  - "docs/process/layout-read.md"
  - "internal/d2dctl"
  - "internal/d2dworld"
  - "cmd/d2bot/objects.go"
  - "cmd/d2bot/objects_layout_test.go"
  - "internal/env/tuning_quests.go"
  - "internal/d2denv"
  - "docs/knobs.md"
exclusive: []
contracts: []
result: "Public side done and tested offline (fake server, d2dworld, cmd/d2bot; not verified live): client + fake + d2dworld merge + knob quests.layoutRead (default 0). Fork verb `debug levelmap` is on the private remote, branch didle/level-layout-20261120; its build and tests/levelmap_d2d.py were queued on the builder long lane at merge time (result in docs/live-checks.md). Not built into an image, no cain.chain A/B run. closecards3 Remaining (live): #3 need the live game, a browser or a worker-host run; not verified here."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 debug levelmap client, fake, d2dworld merge, knob quests.layoutRead default 0
- [x] #2 must-fail tests with the fake: knob on, off, verb absent
- [ ] #3 fork verb built and differential 0 differences (live-check)
<!-- AC:END -->
