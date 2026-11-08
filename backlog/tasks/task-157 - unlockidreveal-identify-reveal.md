---
id: TASK-157
title: 'unlockidreveal: identify reveal'
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
priority: medium
ordinal: 135000
card: "unlockidreveal"
owns:
  - "internal/watch/unlockidreveal.go"
  - "internal/watch/unlockidreveal_test.go"
  - "internal/watch/unlock.go"
  - "internal/watch/funwire.go"
  - "internal/watch/web/**"
  - "internal/watch/testdata/reveal-*.json"
  - "internal/unlock/**"
  - "data/unlocks/ladder.json"
  - "data/idreveal/**"
  - "docs/unlock-idreveal.md"
exclusive: []
contracts: []
result: "Built and tested offline (go test ./internal/watch ./internal/unlock; golden payloads face down and flipped for Sorceress and Paladin); not verified live or in a browser. Face-down cards leak no quality, colour or name. Missing reader: item affixes after identify (Fake in tests, card says not read); no identify event, the flip is by the viewer. See docs/unlock-idreveal.md. Status sync 2026-11-08 (stays open): AC5 open (item affixes need a reader that does not exist)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Unlock: card flip when the first magic or rare item is picked up; later affixes, colour, sound; first unique or set later. Not on the two-week playable path. Design note: docs/research/unlockables.md (a brainstorm, not a decision). Technical check: check how the bot handles unidentified items before the reveal; mods and uniques only after the first encounter.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 What the bot does with unidentified items is checked and written down (docs/unlock-idreveal.md)
- [x] #2 Ladder rows idreveal (first magic or rare) and idreveal_later (first unique or set in the Codex); face-down card, flip, then colour, sound and text; no quality, colour or name before the flip
- [x] #3 Unique, set and crafted cards wait for the Codex first; flip route gated (token, origin, JSON, 404 for hidden ids)
- [x] #4 Golden payloads for two classes, locked/open, replay idempotency, unknown row and table errors naming them, a reveal line exists
- [ ] #5 Item affixes after identify (needs a reader that does not exist yet; Fake only)
<!-- AC:END -->
