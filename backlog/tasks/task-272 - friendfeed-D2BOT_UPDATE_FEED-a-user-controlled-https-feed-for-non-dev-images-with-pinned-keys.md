---
id: TASK-272
title: >-
  friendfeed: D2BOT_UPDATE_FEED, a user-controlled https feed for non-dev images
  with pinned keys
status: Done
assignee:
  - credit-runner
created_date: '2026-11-10T21:56:00Z'
updated_date: '2026-11-10T22:26:00Z'
labels:
  - bot
  - feature
  - tester
dependencies: []
priority: medium
ordinal: 199000
card: "friendfeed"
owns:
  - "internal/selfupdate/feed.go"
  - "internal/selfupdate/feed_test.go"
  - "cmd/d2bot/update.go"
  - "cmd/d2bot/update_test.go"
  - "docs/friend-feed.md"
exclusive: []
contracts: []
result: "Built and unit-tested offline (selfupdate, d2bot); not run against a real feed or the live image."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

PR 2 of docs/research/friend-build.md section 10 (user decision 2026-11-10). Done when merged.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Non-dev image reads D2BOT_UPDATE_FEED (https only, exact host allow-list)
- [x] #2 Keys stay pinned; feed adds no trust
- [x] #3 Tampered, wrong-key and unsigned manifests refused (tests)
<!-- AC:END -->
