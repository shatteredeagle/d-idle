---
id: TASK-159
title: 'unlockrisk: risk meter'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T03:16:00Z'
labels:
  - watch
  - feature
milestone: m-4
dependencies:
  - TASK-153
priority: low
ordinal: 137000
card: "unlockrisk"
owns:
  - "internal/watch/unlockrisk.go"
  - "internal/watch/unlockrisk_test.go"
  - "internal/watch/risk.json"
  - "internal/watch/funwire.go"
  - "internal/watch/testdata/risk-*.json"
  - "data/unlocks/ladder.json"
  - "docs/unlock-risk.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Unlock: first near-death survived; ready / marginal / turn back before a boss. Not on the two-week playable path. Design note: docs/research/unlockables.md (a brainstorm, not a decision). Technical check: the meter must use the same numbers the bot uses (potions, life ratio, resists vs the zone).
<!-- SECTION:DESCRIPTION:END -->

result: built and tested offline on fixtures (docs/unlock-risk.md); NOT wired in cmd/d2bot, not verified live.
