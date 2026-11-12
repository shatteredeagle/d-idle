---
id: TASK-356
title: 'cut010: first friends release 0.1.0 (build and verify, not published)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-12T03:59:00Z'
labels:
  - container
  - chore
dependencies: []
priority: medium
ordinal: 232000
card: "cut010"
owns:
  - "docs/releases/0.1.0.md"
  - "docs/friend-checklist-0.1.0.md"
  - "tools/friendpack/build.sh"
  - "tools/friendpack/checklist.sh"
  - "deploy/d2d-container/context.sh"
  - "tools/friendpack/compose.yml"
  - "cmd/d2bot/main.go"
exclusive:
  - "fork-branch:fork-only/icon-original"
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Image didle:0.1.0 built from main with the pinned fork commit, booted once with the new icon
- [ ] #2 Drills D1-D7 run on the new image and legal checklist recorded
- [ ] #3 Signed beta release set and the friends pack built (not published), sha256 recorded
- [ ] #4 docs/releases/0.1.0.md written
<!-- AC:END -->
