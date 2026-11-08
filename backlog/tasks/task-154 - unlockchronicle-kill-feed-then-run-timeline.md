---
id: TASK-154
title: 'unlockchronicle: kill feed, then run timeline'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T03:15:00Z'
labels:
  - watch
  - feature
milestone: m-4
dependencies:
  - TASK-153
priority: medium
ordinal: 132000
card: "unlockchronicle"
owns:
  - "internal/watch/unlockchronicle.go"
  - "internal/watch/unlockchronicle_test.go"
  - "internal/watch/unlock.go"
  - "internal/watch/funwire.go"
  - "internal/watch/returnscreen.go"
  - "internal/watch/web/**"
  - "internal/watch/testdata/chronicle-*.json"
  - "internal/unlock/**"
  - "data/unlocks/ladder.json"
  - "data/chronicle/**"
  - "cmd/d2bot/funwire.go"
  - "docs/unlock-chronicle.md"
exclusive: []
contracts: []
result: "Built and tested offline (go test ./internal/unlock ./internal/watch ./cmd/d2bot; golden payloads for Sorceress and Paladin); not verified live or in a browser. Also opens the unlock ladder in cmd/d2bot (FunConfig.Unlock) and feeds kills_of_kind. See docs/unlock-chronicle.md."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Unlock: first kill opens the kill feed; the first expedition card read opens the run timeline (M follow-up). Not on the two-week playable path. Design note: docs/research/unlockables.md (a brainstorm, not a decision). Technical check: kills must come from observed events, the same source as the tally.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 First kill opens the feed; the first expedition card read opens the timeline tape; ladder rows chronicle and chronicle_tape with teasers
- [x] #2 Feed in Diablo II wording from a data table (3+ variants per kind, seeded), kills from the same events as the tally
- [x] #3 Golden payloads for two classes, locked/open states, replay idempotency, unknown row errors naming it, a reveal line exists
<!-- AC:END -->
