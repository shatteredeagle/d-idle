---
id: TASK-128
title: 'policysheet: chores policy sheet with effect lines and before/after cards'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-08T00:26:00Z'
labels:
  - watch
  - feature
  - ladder
milestone: m-4
dependencies:
  - TASK-127
priority: high
ordinal: 120000
card: "policysheet"
owns:
  - "internal/watch/policysheet.go"
  - "internal/watch/policysheet_test.go"
  - "internal/watch/testdata/policysheet/**"
  - "internal/decide/policy.go"
  - "internal/decide/policy_test.go"
  - "internal/decide/policy_chores.json"
  - "docs/policy-sheet.md"
  - "internal/decide/store.go"
  - "internal/watch/returnscreen.go"
  - "internal/watch/funwire.go"
  - "cmd/d2bot/funwire.go"
  - "cmd/d2bot/funwire_test.go"
  - "cmd/d2bot/ladder.go"
  - "cmd/d2bot/live.go"
  - "cmd/d2bot/d2dbackend.go"
  - "cmd/d2dsetup/page/app.js"
exclusive: []
contracts: []
result: "Offline fixtures only, not live. Table internal/decide/policy_chores.json (14 rows, 2 builds), store op policy answers before cards (source policy), watch sheet page plus POST route, effect counters on a fixture journal, before/after section with change-it-back, wired in cmd/d2bot. Gaps in docs/policy-sheet.md: no all-mana belt path (bot card), identify/potion tier/stash full not asked by the bot yet (stored only), no repair or gold events, /api/rules lacks source policy."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: ADR 0018 item 3, fun-ideas.md section 13 card 2. One sheet set at hero creation, changeable any time, suboptimal choices allowed. Needs returnscreen only for the before/after cards (the sheet itself needs nothing). Kill: set once and never touched in two sessions. Done when: the criteria below hold.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 The sheet is a page; each row writes a ladder rule with source policy into the decide store (internal/decide, internal/ladder policy keys); chore points (repair, restock, potion tier, town trigger, tp lost, return, identify, stash full) stop opening cards when a policy covers the key; test against two builds
- [x] #2 Per-row effect line from the cards since the change, on the sheet AND as before/after cards on the return screen
- [x] #3 docs/policy-sheet.md says built vs verified live; go vet and go test -p 1 of the touched packages; leakcheck 0
<!-- AC:END -->
