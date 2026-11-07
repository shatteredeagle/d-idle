---
id: TASK-59
title: 'custom-overlay: row-merge table overlay in the d.idle fork and Go loader'
status: In Progress
assignee: []
created_date: '2026-11-07T03:01:00Z'
labels:
  - T5
dependencies: []
ordinal: 57000
card: "custom-overlay"
owns:
  - "docs/research/custom-overlay-spec.md"
  - "docs/overlay.md"
  - "docs/research/custom-items.md"
  - "internal/overlay/"
  - "internal/data/"
  - "cmd/d2bot/d2dbackend.go"
  - "cmd/d2bot/d2dbackend_test.go"
exclusive: []
contracts: []
result: ""
---

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Overlay hook in Stack::try_read, flag-gated, vanilla hashes unchanged
- [ ] #2 Go data loader applies the same fragments and stamps them
<!-- AC:END -->
