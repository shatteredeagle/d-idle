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
result: "Fork branch d2d.lab4-overlay (private remote) built on the worker host (D2D_WERROR=OFF: GCC 16 trips -Werror in untouched ui.cpp/command code); ctest test_overlay (merge, malformed, hash pin) and overlay_d2d (start off/on/malformed against real MPQs) pass, verified live-free. AC 1 stays open: vanilla table-hash equality not measured (only by code path); full ctest suite not run. Go side: internal/overlay, internal/data, d2bot flag tested on the worker host. Not pinned in gates."
---

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Overlay hook in Stack::try_read, flag-gated, vanilla hashes unchanged
- [x] #2 Go data loader applies the same fragments and stamps them
<!-- AC:END -->
