---
id: TASK-346
title: 'herounify: one Hero tile (stats merged into the Hero card)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T22:23:00Z'
labels:
  - watch
  - feature
dependencies: []
priority: medium
ordinal: 228000
card: "herounify"
result: "Hero and Hero stats merged into hero-vitals (v3); hero-detail aliased (mergedInto, shell.json) and dropped from layout.json; node goldens, alias tests, whole-repo gate exit 0, uishots viewed at 5 widths. Not verified live. No merc, build or defence fields in the hero endpoint, so not on the tile."
owns:
  - "internal/watch/web/panels/hero-vitals"
  - "internal/watch/web/panels/hero-detail"
  - "internal/watch/web/layout.json"
  - "internal/watch/web/shell.json"
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
- [x] #1 Hero and Hero stats are one panel (hero-vitals); hero-detail is an alias, layouts with both show one tile
- [x] #2 DOM goldens ready/nostats/error, alias tests, uilint and compat green
- [x] #3 uishots at 5 breakpoints viewed
<!-- AC:END -->
