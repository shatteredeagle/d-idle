---
id: TASK-276
title: >-
  betakey: pin the beta signing key beside the release key and dry-run the first
  feed release
status: Done
assignee:
  - credit-runner
created_date: '2026-11-10T23:47:00Z'
updated_date: '2026-11-11T16:24:00Z'
labels:
  - container
  - feature
dependencies: []
priority: medium
ordinal: 202000
card: "betakey"
owns:
  - "deploy/d2d-container/pubkeys"
  - "tools/friendpack"
  - "tools/release"
  - "docs/friend-feed.md"
  - "docs/self-update-release.md"
  - "internal/selfupdate/feed_test.go"
  - "deploy/d2d-container/pubkeys_test.go"
exclusive: []
contracts: []
result: "beta.pub pinned; two-key tests (beta, current, unknown) pass; dry run 0.1.0 channel beta built, signed with the real beta key, feed tree verified with beta.pub in scratch, nothing uploaded. Not verified live (no real image, d2d fork binary; placeholders used)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 beta.pub pinned beside current.pub; two-key tests pass
- [x] #2 dry-run release tree builds and verifies with beta.pub (no upload)
<!-- AC:END -->
