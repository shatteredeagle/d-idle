---
id: TASK-1085
title: 'questauto-visible: show which ready quest chunks wait for the Start tap'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-28T00:27:00Z'
updated_date: '2026-11-28T01:11:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 519000
card: "questauto-visible"
lane: next
owns:
  - "internal/questboard/"
  - "internal/ops/"
  - "cmd/d2bot/opscmd.go"
  - "internal/watch/testdata/quests/"
  - "docs/operator.md"
exclusive: []
contracts: []
result: "ops quest status shows auto=on|off|never per row; ops status --why prints waiting-for-Start lines and the auto held-back reason; /api/quests chunks carry auto and waitingFor; Return screen lists chunks waiting for Start. Offline only, not verified live. Chunk start rules unchanged; auto_opt_in value not used."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 ops quest status prints auto=on|off|never per chunk row
- [x] #2 ops status --why prints a waiting-for-Start line per idle ready chunk and the auto held-back reason
- [x] #3 /api/quests chunk rows carry auto and waitingFor; Return screen Quests paused lists waiting chunks
- [x] #4 tests on two classes; ready chunk with auto off reports start_tap
<!-- AC:END -->
