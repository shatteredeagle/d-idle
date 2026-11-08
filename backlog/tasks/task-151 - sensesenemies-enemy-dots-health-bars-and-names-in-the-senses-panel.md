---
id: TASK-151
title: 'sensesenemies: enemy dots, health bars and names in the senses panel'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T03:12:00Z'
labels:
  - watch
  - feature
milestone: m-4
dependencies:
  - TASK-148
priority: low
ordinal: 134000
card: "sensesenemies"
owns:
  - "internal/watch/senses_units.go"
  - "internal/watch/senses_units_test.go"
  - "internal/watch/web/senses_units.js"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Design: docs/research/info-unlocks.md. Not on the two-week playable path (returnscreen, policysheet, tallystrip, codexfirsts, playsession2); after it. Done when the panel shows dots and bars from a fake source and the live LifePct check is written down.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Units list (class, x/y, life pct, mode) from the bot's d2.Reader on both backends into /api/senses; dead units dropped
- [ ] #2 Tier 2 dots and tier 3 bars/names gated per monster class by the Codex; plain dot for an unmet class
- [ ] #3 Offline test with mem.Fake against two classes of monster; live check of LifePct on damaged monsters listed as a checklist (U today)
<!-- AC:END -->
