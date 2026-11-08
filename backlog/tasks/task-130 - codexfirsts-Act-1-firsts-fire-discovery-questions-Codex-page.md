---
id: TASK-130
title: 'codexfirsts: Act 1 firsts fire discovery questions, Codex page'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T00:26:00Z'
labels:
  - bot
  - feature
  - ladder
milestone: m-4
dependencies: []
priority: high
ordinal: 122000
card: "codexfirsts"
owns:
  - "internal/decide/firsts.go"
  - "internal/decide/firsts_test.go"
  - "internal/watch/codex.go"
  - "internal/watch/codex_test.go"
  - "internal/watch/testdata/codex-*.json"
  - "internal/decide/firsts_act1.json"
  - "docs/codex.md"
exclusive: []
contracts: []
result: "Table, tracker, Codex, /api/codex done; hook wired by TASK-140 from goal events (kill, area change, drop, death) plus the runner waypoint hook. Live smoke: 3 entries in the first minutes (Fallen, Blood Moor, Quill Rats), 2 questions asked, kill criterion (3 firsts in 10 min) met. champion and shrine readers still missing."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: ADR 0018 item 6, fun-ideas.md section 13 card 4; supersedes TASK-108 qdensity (question density in hour one comes from Act 1 firsts, not a fade variant). Kill: fewer than 3 firsts fire in the first 10 minutes at 1x, or the user says they read like a bestiary. Done when: the criteria below hold.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A first context flag in the catalogue; firsts table for Den and Corpsefire, Bishibosh, Blood Raven, plus the Shaman and the Cold Plains waypoint (event U); one discovery question each with the rule it teaches; build and class data stay in tables
- [x] #2 A first-sight hook on monster class ids and area ids in internal/bot; a Codex page counting entries; tests against two builds
- [x] #3 docs/codex.md says built vs verified live; go vet and go test -p 1 of the touched packages; leakcheck 0
<!-- AC:END -->
