---
id: TASK-158
title: 'unlockbestiary: bestiary'
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
ordinal: 136000
card: "unlockbestiary"
owns:
  - "internal/watch/unlockbestiary.go"
  - "internal/watch/unlockbestiary_test.go"
  - "internal/watch/funwire.go"
  - "internal/watch/web/"
  - "internal/watch/testdata/bestiary-*.json"
  - "data/bestiary/"
  - "data/unlocks/ladder.json"
  - "docs/unlock-bestiary.md"
exclusive: []
contracts: []
result: "Built and tested offline (go test ./internal/watch ./internal/unlock ./cmd/d2bot; golden payloads at 10 and 50 kills for Sorceress and Paladin, spoiler rule, replay, return section); not verified live or in a browser. Owns widened to funwire.go, web/, data/bestiary, ladder rows bestiary and bestiary_resists. Card tables come through FunConfig.Monsters; cmd/d2bot does not set it yet (flavour-only cards live), kind names unchecked against the real monstats (U). See docs/unlock-bestiary.md."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Unlock: 10 kills of a kind opens its page, 50 kills show resist details; silhouettes for unmet kinds. Not on the two-week playable path. Design note: docs/research/unlockables.md (a brainstorm, not a decision). Technical check: do not tie to the Cain rescue (late for a Sorceress); scouting shows categories, not spoilers.
<!-- SECTION:DESCRIPTION:END -->
