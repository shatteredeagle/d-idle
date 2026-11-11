---
id: TASK-290
title: >-
  electrontray: Electron app tray icon, autostart, hide-to-tray, signed update
  check
status: Done
assignee:
  - credit-runner
created_date: '2026-11-11T01:31:00Z'
updated_date: '2026-11-11T16:24:00Z'
labels:
  - watch
  - feature
dependencies:
  - TASK-289
priority: medium
ordinal: 205000
card: "electrontray"
result: "PR2 built; V on the worker host: 32 node tests, Xvfb smoke (Tray constructed, menu template, autostart toggle, hide-to-tray, signed fake feed update). NOT verified: a real tray host (KDE/Windows), real Windows autostart, a real feed (none exists yet), notifications skipped. Live game not used."
owns:
  - "apps/desktop-electron/"
  - "docs/desktop-electron.md"
  - ".github/workflows/desktop-electron.yml"
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
- [x] #1 Tray with Open, pages submenu, reachability status line, Start with the system, hide-to-tray toggle, Quit; single instance; original icon from SVG
- [x] #2 Autostart: Linux XDG entry, Windows/macOS login item, started with --hidden
- [x] #3 Update check is check-only against the pinned-key signed feed, never downloads, no github.com; off without a configured feed
- [x] #4 Unit tests and the Xvfb smoke (Tray API, menu template, hide-to-tray, signed fake feed) pass on the worker host
<!-- AC:END -->
