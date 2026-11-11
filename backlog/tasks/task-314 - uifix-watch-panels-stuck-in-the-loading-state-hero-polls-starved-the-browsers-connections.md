---
id: TASK-314
title: 'uifix: watch panels stuck in the loading state (hero polls starved the browser''s connections)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T18:31:00Z'
labels:
  - watch
  - bug
dependencies: []
priority: high
ordinal: 217000
card: "uifix"
owns:
  - "internal/watch/hero.go"
  - "internal/watch/invapi.go"
  - "internal/watch/herofast_test.go"
  - "internal/watch/web/shell.js"
  - "internal/watch/web/app.js"
  - "internal/watch/testdata/shell_sched.js"
  - "tools/uishots"
  - "tools/uishots.sh"
  - "docs/ui-design-system.md"
  - "backlog/merge-lines/agent-uifix.txt"
exclusive: []
contracts: []
result: "Likely cause: connection starvation (video and two event streams hold three of six connections; five panels polled /api/hero, each request built the whole inventory view). Fixed on both sides and tested (blocking-desk Go test, node scheduler test, console smoke in uishots). Not verified on the live sandbox: the browser console and network tab were not available, so the diagnosis is from code and tests."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Live regression 2026-11-11: most /watch cards stayed skeletons. Done when the acceptance criteria hold and the branch is merged.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 /api/hero never builds the inventory view on the request (test with a blocking desk)
- [x] #2 the shell sends one request per address per cycle and the page has one event stream
- [x] #3 node test: after the first poll cycle no panel stays in the loading state, a hanging endpoint does not starve the others
- [x] #4 uishots fails on an uncaught console error
<!-- AC:END -->
