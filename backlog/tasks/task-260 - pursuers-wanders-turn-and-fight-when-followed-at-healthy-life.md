---
id: TASK-260
title: 'pursuers: wanders turn and fight when followed at healthy life'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-10T17:29:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 195000
card: "pursuers"
owns:
  - "internal/nav/pursuit.go"
  - "internal/nav/pursuit_test.go"
  - "docs/pursuers.md"
  - "internal/nav/walk.go"
  - "internal/nav/roam.go"
  - "internal/nav/explore.go"
  - "internal/nav/exit.go"
  - "internal/nav/nav.go"
  - "internal/env/tuning_nav.go"
  - "docs/knobs.md"
exclusive: []
contracts: []
result: "Offline only, NOT live verified: nav pursuit rule + 3 knobs, tests in internal/nav/pursuit_test.go. Root cause of the stand-still report (17:23-17:27 UTC) is U: no monster positions in the log (docs/pursuers.md)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the Paladin of d2play-20261108 ran from followers and stood still under attack while exploring (docs/pursuers.md). Done when: wanders turn and fight when followed at healthy life, offline tested.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Explore, roam and exit walks hand over to the fight when a monster follows within nav.pursuitRadius at healthy life
- [ ] #2 Knobs nav.pursuitRadius, nav.pursuitPolls, nav.pursuitMinLifePct in docs/knobs.md
- [ ] #3 docs/pursuers.md records the log evidence with V/S/U tags
<!-- AC:END -->
