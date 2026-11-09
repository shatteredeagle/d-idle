---
id: TASK-207
title: 'd2dsrcmana: the d2s reader loses the fraction of max mana (19.5 reads 19.0)'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-08T20:31:00Z'
updated_date: '2026-11-08T21:06:00Z'
labels:
  - bot
  - bug
milestone: m-4
dependencies: []
priority: low
ordinal: 161000
card: "d2dsrcmana"
owns:
  - "internal/d2dsrc"
exclusive: []
contracts: []
result: "Fixed in internal/d2dsrc/player.go playerStats (not a d2s reader: the d2d dump pair is whole points; the 256ths are in dump stats raw). Life/mana/stamina and maxima use raw when it agrees with the pair, else the old shift. Fixtures only, not verified live; no recorded Paladin dump exists in the repo, the test uses synthetic raw 4992 (Paladin) and 25664 (Sorceress). Whole-point fields and Stats.Fixed unchanged, so threshold callers see the same values; fidprobe has no expectation on the old value."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 reader returns the saved fixed-point value (19.5) and a test with the Paladin fixture
<!-- AC:END -->
