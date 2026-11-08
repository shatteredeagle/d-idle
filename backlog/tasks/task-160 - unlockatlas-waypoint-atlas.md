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
result: "Built and tested offline (go test ./internal/watch ./internal/unlock ./cmd/d2bot; golden payloads for Sorceress and Paladin, dark and lit proposal node, token gates, replay, errors); not verified live or in a browser. Owns widened to funwire.go, web/, data/atlas, ladder row atlas. Waypoint order and area ids are data (data/atlas/waypoints.json); lit comes from the Codex waypoint firsts, so the bot's existing waypoint hook needs no change. See docs/unlock-atlas.md."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Unlock: first waypoint taken; lit nodes, silhouettes for locked, the hero's proposed next zone. Not on the two-week playable path. Design note: docs/research/unlockables.md (a brainstorm, not a decision). Technical check: view only; check which waypoint data the bot already records.
<!-- SECTION:DESCRIPTION:END -->
