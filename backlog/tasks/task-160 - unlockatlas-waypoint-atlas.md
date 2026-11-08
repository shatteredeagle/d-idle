---
id: TASK-160
title: 'unlockatlas: waypoint atlas'
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
ordinal: 138000
card: "unlockatlas"
owns:
  - "internal/watch/unlockatlas.go"
  - "internal/watch/unlockatlas_test.go"
  - "internal/watch/funwire.go"
  - "internal/watch/web/"
  - "internal/watch/testdata/atlas-*.json"
  - "data/atlas/"
  - "data/unlocks/ladder.json"
  - "docs/unlock-atlas.md"
exclusive: []
contracts: []
result: "Offline proof 2026-11-08: go test ./internal/watch ./internal/unlock and ./cmd/d2bot (-run BestiaryWired, which also fetches the atlas) pass; golden payloads for Sorceress and Paladin, dark and lit proposal node, token gates (TestAtlasGoldenTwoClasses, TestAtlasReplayAndGates, TestAtlasTableAndErrors). Fixtures only; NOT verified live or in a browser. Stays open: AC #5 (browser) and #6 (live) unticked."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Unlock: first waypoint taken; lit nodes, silhouettes for locked, the hero's proposed next zone. Not on the two-week playable path. Design note: docs/research/unlockables.md (a brainstorm, not a decision). Technical check: view only; check which waypoint data the bot already records.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 ladder row atlas opens on the first waypoint taken (Codex waypoint), offline (TestAtlasGoldenTwoClasses)
- [x] #2 GET /api/atlas returns watch.atlas/1: lit nodes from the Codex, silhouettes for locked, the first pending proposal as the hero's next zone, on a dark and a lit node (TestAtlasGoldenTwoClasses)
- [x] #3 viewer token gates and replay safety (TestAtlasReplayAndGates); waypoint table errors name the row (TestAtlasTableAndErrors)
- [x] #4 cmd/d2bot needs no config: NewFun builds the atlas from the ladder row, Codex and proposal desk (docs/unlock-atlas.md, TestBestiaryWiredWithTableAndRestart, TASK-185)
- [ ] #5 atlas panel renders in a real browser (not run)
- [ ] #6 LIVE check on a real run: nodes light as waypoints are taken (not run; the live panels check is pending)
<!-- AC:END -->
