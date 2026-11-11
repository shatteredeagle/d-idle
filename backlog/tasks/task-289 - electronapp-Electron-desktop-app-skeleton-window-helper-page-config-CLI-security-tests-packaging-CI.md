---
id: TASK-289
title: >-
  electronapp: Electron desktop app skeleton (window, helper page, config, CLI,
  security tests, packaging, CI)
status: Done
assignee:
  - credit-runner
created_date: '2026-11-11T01:18:00Z'
updated_date: '2026-11-11T21:25:00Z'
labels:
  - watch
  - feature
dependencies:
  - TASK-265
priority: medium
ordinal: 204000
card: "electronapp"
result: "PR1 built; V on the worker host: 20 node tests, Xvfb smoke against fake servers (nav lock, no preload in game view, window.open denied, redirect locked), AppImage+deb built, fuses read back, packaged binary starts under Xvfb. NOT verified: real login, Windows build (CI only), macOS, Ubuntu 24.04 sandbox. Live game not used."
owns:
  - "apps/desktop-electron/"
  - "docs/desktop-electron.md"
  - ".github/workflows/desktop-electron.yml"
  - "docs/desktop-app.md"
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
- [x] #1 App opens the configured origin in a sandboxed preload-less view; helper page offline/warming with retry; config {address} 0600; flags --setup --address --page --print
- [x] #2 Security posture tests (node:test) and an Xvfb smoke against fake servers pass on the worker host
- [x] #3 electron-builder config builds AppImage and deb on Linux with fuses flipped; CI job builds Windows NSIS and portable and publishes nothing
- [x] #4 Whole-repo Go gate unaffected (stub go.mod); docs/desktop-electron.md and docs/desktop-app.md updated
<!-- AC:END -->
