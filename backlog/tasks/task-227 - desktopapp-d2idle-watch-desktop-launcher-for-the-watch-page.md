---
id: TASK-227
title: 'desktopapp: d2idle-watch desktop launcher for the watch page'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-09T18:28:00Z'
updated_date: '2026-11-10T16:54:00Z'
labels:
  - watch
  - feature
dependencies: []
priority: medium
ordinal: 175000
card: "desktopapp"
owns:
  - "apps/desktop"
  - "docs/desktop-app.md"
  - ".github/workflows/desktop-app.yml"
exclusive: []
contracts: []
result: "Launcher built (Go, stdlib, Linux+Windows cross-compile ok), unit tests and Xvfb smoke with a fake browser pass on the worker host. NOT verified live: real Chromium/Edge app window, login with persisted cookie, Windows run (no Chromium on the worker host); checklist in docs/desktop-app.md. CI workflow written, never run."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

User request 2026-11-09. Done when the launcher builds for Linux and Windows and the doc explains the same-origin rule.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Design and stack decision written in docs/desktop-app.md
- [x] #2 Address config, URL normalisation, probe and shell logic unit-tested
- [x] #3 Smoke run under Xvfb with a fake browser passes
- [ ] #4 Real Chromium or Edge app-mode window verified (needs a machine with one)
- [x] #5 CI workflow written, not run
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Closed in the 2026-11-10 true-up (docs/research/project-review-2026-11-10.md section 3.1): merged, offline acceptance proven; open live checks moved to docs/live-checks.md (TASK-253).
<!-- SECTION:NOTES:END -->
