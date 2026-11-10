---
id: TASK-275
title: >-
  feedpublish: feed-publish.sh and d2feed, a host-agnostic static feed tree with
  client verification
status: Done
assignee:
  - credit-runner
created_date: '2026-11-10T22:52:00Z'
updated_date: '2026-11-10T23:06:00Z'
labels:
  - container
  - feature
  - tester
dependencies: []
priority: medium
ordinal: 201000
card: "feedpublish"
owns:
  - "cmd/d2feed"
  - "tools/release/feed-publish.sh"
  - "docs/friend-feed.md"
exclusive: []
contracts: []
result: "Built and tested offline with a throwaway key (go test ./cmd/d2feed, incl. the wrapper with an upload hook and the real update client). Not run against a real host or the real key; upload step is a user-supplied hook or rsync."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

PR 3 of docs/research/friend-build.md section 10 (user go 2026-11-10). Done when merged.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 d2feed builds the GitHub-shaped feed tree from a signed release, refuses unsigned, wrong-key, tampered and older input, is idempotent and keeps old releases
- [x] #2 d2feed verify runs the real update client with the pinned key against the tree; withdraw hides a release
- [x] #3 feed-publish.sh signs with a key path given by the user, verifies, then runs the user's upload hook or rsync; runbook and static server snippets in docs/friend-feed.md
<!-- AC:END -->
