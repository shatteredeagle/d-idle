---
id: TASK-1054
title: 'vaultpoll: cheap vault poll (cached preview, windowed query)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-27T18:52:00Z'
labels:
  - watch
  - bug
dependencies: []
priority: medium
ordinal: 511000
card: "vaultpoll"
lane: next
owns:
  - "internal/watch"
  - "internal/vault"
  - "internal/vaultkeep"
  - "docs/probes/vault-perf-2026-11-27.md"
exclusive: []
contracts: []
result: "Poll at 20k records 5.8 ms to 0.54 ms (3.2 MB to 163 KB alloc), page 6.6 to 1.0 ms; preview and facets cached by store version, Query windowed, no Dump copies on the poll path. Offline benchmark only, not verified live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Benchmarks at 5k and 20k records, before and after
- [x] #2 Mass preview and facets cached by store version, tested against writes
- [x] #3 No Dump copies on the poll path; Query reads a page window
<!-- AC:END -->
