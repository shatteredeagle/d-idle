---
id: TASK-294
title: 'electronmac2: macOS first-run fixes (Local Network, no blank window, log file, switch passthrough)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T02:24:00Z'
labels:
  - watch
  - bug
dependencies:
  - TASK-291
priority: medium
ordinal: 207000
card: "electronmac2"
owns:
  - "apps/desktop-electron/"
  - "docs/desktop-electron.md"
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
- [x] #1 NSLocalNetworkUsageDescription in Info.plist
- [x] #2 failed, stalled or empty page load shows the helper and retries with backoff
- [x] #3 --debug log file, Open log folder, Chromium switch passthrough
- [x] #4 smoke asserts non-blank views, stall and failure recovery
- [x] #5 universal dmg/zip rebuilt, downloads replaced
<!-- AC:END -->
