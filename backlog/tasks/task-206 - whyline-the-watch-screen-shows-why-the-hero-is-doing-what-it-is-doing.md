---
id: TASK-206
title: 'whyline: the watch screen shows why the hero is doing what it is doing'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T19:51:00Z'
labels:
  - watch
  - feature
milestone: m-4
dependencies: []
priority: high
ordinal: 160000
card: "whyline"
owns:
  - "internal/watch/whyline.go"
  - "internal/watch/whyline.json"
  - "internal/watch/whyline_test.go"
  - "internal/watch/testdata/whyline-paladin.json"
  - "internal/watch/testdata/whyline-sorceress.json"
  - "internal/watch/web"
  - "docs/whyline.md"
  - "internal/bot/whyline.go"
  - "internal/bot/whyline_test.go"
  - "internal/goals/runner.go"
  - "internal/bot/goal.go"
  - "internal/bot/task.go"
  - "internal/bot/death.go"
  - "internal/bot/farm.go"
  - "internal/bot/broke.go"
  - "internal/bot/routeguard.go"
  - "internal/bot/runner.go"
  - "internal/watch/hub.go"
  - "internal/watch/bridge.go"
  - "internal/watch/http_view.go"
  - "internal/watch/funwire.go"
  - "internal/watch/sessionrec.go"
  - "internal/watch/unlockchronicle.go"
  - "cmd/d2bot/play.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: viewers (and the user) should see the hero's own reason for what it does (docs/research/incremental-arpg.md, visible reasoning). Done when: docs/whyline.md is true on the live sandbox.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 reason codes set where the bot decides (quest step, walk, fight, town trip, heal trip, guard exit, death recovery) as goals.Why on the runner status
- [x] #2 one-line why under the video card and above the tabs, from a data table of templates (two phrasings per code), unknown code visible in debug and neutral in production
- [x] #3 rate limited to 1 read per second and held 3 s unless the goal changes; last 5 reasons in the Chronicle detail
- [x] #4 tests: templates, hold, two classes, golden API JSON, page order, race; session summary has reason code counts only
- [ ] #5 verified on the live sandbox (needs the passphrase for the page; API fields by curl)
<!-- AC:END -->

result: built and tested offline (fixtures, race-free tests, golden API for Sorceress and Paladin); NOT verified live or in a browser; see docs/whyline.md.
Status sync (stays open): AC #5 (live check on the sandbox) is unticked.
