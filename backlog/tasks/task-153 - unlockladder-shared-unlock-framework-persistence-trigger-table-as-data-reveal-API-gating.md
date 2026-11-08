---
id: TASK-153
title: >-
  unlockladder: shared unlock framework: persistence, trigger table as data,
  reveal, API gating
status: Done
assignee:
  - credit-runner
created_date: '2026-11-08T03:15:00Z'
updated_date: '2026-11-08T08:35:00Z'
labels:
  - watch
  - feature
milestone: m-4
dependencies:
  - TASK-140
  - TASK-141
  - TASK-129
  - TASK-149
priority: low
ordinal: 131000
card: "unlockladder"
owns:
  - "internal/watch/unlock.go"
  - "internal/watch/unlock_test.go"
  - "docs/research/unlockables.md"
  - "internal/unlock/**"
  - "data/unlocks/**"
  - "docs/unlock-ladder.md"
  - "internal/watch/funwire.go"
exclusive: []
contracts: []
result: "Engine, ladder.json schema, reveal, GET /api/unlocks, POST /api/unlocks/buy built and tested offline (go test ./internal/unlock ./internal/watch); not verified live; cmd/d2bot does not open the engine yet; kills/retire have no emitter;."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Per-profile persistence of which unlocks are open; trigger table as DATA (kinds codex_first, quest, level, tally_spend, retire_count, kills_of_kind); a reveal moment (log line plus silhouette); API gating. Reused by the senses ladder and the ten unlockables. Depends on Codex wiring (TASK-140, TASK-141) and the tally (TASK-129, TASK-141); also depends on the card sensesunlock (TASK-149). Not on the two-week playable path. Design note: docs/research/unlockables.md (a brainstorm, not a decision). Technical check: gate API output on the unlock state without hiding basics; never unlock silently.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Trigger table as data (data/unlocks/ladder.json) with kinds codex_first, quest, level, tally_spend, retire_count, kills_of_kind; an unknown kind or entry is an error naming it
- [x] #2 Per-profile persistence (unlocks-<profile>.json), replay-safe, never relocks
- [x] #3 Reveal: Chronicle line plus return screen section Unlocked since you left; no silent unlock
- [x] #4 GET /api/unlocks (locked entries without text) and POST /api/unlocks/buy with token/origin gates, 402 when short, ledger spend entry
<!-- AC:END -->
