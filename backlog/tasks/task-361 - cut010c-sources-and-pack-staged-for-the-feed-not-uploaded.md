---
id: TASK-361
title: 'cut010c: sources and pack staged for the feed (not uploaded)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-12T14:39:00Z'
labels:
  - container
  - chore
dependencies: []
priority: medium
ordinal: 235000
card: "cut010c"
result: "Feed SOURCE.txt now names hosted d2d and bot source tarballs with sha256 and the written offer; the signed 0.1.0 set, the source files and the pack (unguessable path) are staged on the worker host and verified with the real client, NOT uploaded (publish needs the user go). d2d tarball scan clean."
owns:
  - "docs/releases/0.1.0.md"
  - "docs/friend-checklist-0.1.0.md"
  - "tools/release/app-assets.sh"
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
- [x] #1 Feed SOURCE.txt rewritten with hosted source tarballs; staged tree verified
<!-- AC:END -->
