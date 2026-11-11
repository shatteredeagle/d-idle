---
id: TASK-277
title: >-
  beta010: cut the first beta release 0.1.0 seq 1 (image, signed feed tree,
  friends pack; no upload)
status: Done
assignee:
  - credit-runner
created_date: '2026-11-11T00:03:00Z'
updated_date: '2026-11-11T16:24:00Z'
labels:
  - docs
  - chore
dependencies: []
priority: medium
ordinal: 203000
result: "Release 0.1.0 (channel beta, seq 1) cut offline, NOT uploaded. Image didle:0.1.0-beta1 (d2d pin f75816c616c8, built on the worker host); assets signed with the beta key, d2feed verify OK; friends pack 190815541 bytes, SHA256SUMS ok; install.sh smoke on the worker host with fake game files, throwaway project and ports 18891/18892: page up (401 without token), removed afterwards. Not verified live: a real hero, an update from the feed (feed not uploaded)."
card: "beta010"
owns:
  - "docs/friend-guide.md"
  - "backlog/merge-lines/agent-beta010.txt"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Image didle:0.1.0-beta1 built from the pinned d2d commit; assets signed with the beta key; feed tree verified with d2feed
- [x] #2 Friends pack built, checksum verified, install.sh smoke test on the worker host cleaned up
- [x] #3 Release notes in the pack guide; nothing uploaded
<!-- AC:END -->
