---
id: TASK-155
title: 'unlockvoice: hero voice and journal'
status: Done
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
ordinal: 133000
card: "unlockvoice"
owns:
  - "internal/watch/unlockvoice.go"
  - "internal/watch/unlockvoice_test.go"
  - "internal/watch/unlock.go"
  - "internal/watch/funwire.go"
  - "internal/watch/web/**"
  - "internal/watch/testdata/voice-*.json"
  - "data/unlocks/ladder.json"
  - "data/voice/**"
  - "docs/unlock-voice.md"
exclusive: []
contracts: []
result: "Built and tested offline (go test ./internal/watch; golden payloads for Sorceress and Paladin); not verified live or in a browser. Lines are data (6+ per kind, per-class flavour optional), seeded, mute switch, diary page per milestone. See docs/unlock-voice.md."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Unlock: first level-up, first death, each Codex first; one-line hero comments and journal entries. Not on the two-week playable path. Design note: docs/research/unlockables.md (a brainstorm, not a decision). Technical check: journal lines are tied to Codex firsts; check they do not spoil undiscovered content.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Ladder row voice (first level-up, or first death); a line at the first level-up, the first death and each Codex first
- [x] #2 Lines in data/voice/lines.json, 6+ variants per kind, D2 flavour, per-class flavour optional, seeded choice, replay stable
- [x] #3 Mute switch (POST /api/voice/mute with the answer gates), diary page per milestone, no spoiler of unmet content
- [x] #4 Golden payloads for two classes, locked/open states, token gates, unknown row/class named, a reveal line exists
<!-- AC:END -->
