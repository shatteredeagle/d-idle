---
id: TASK-291
title: 'electronmac: Electron app macOS universal dmg+zip, template tray, darwin behaviours, CI job'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T01:47:00Z'
labels:
  - watch
  - feature
dependencies:
  - TASK-290
priority: medium
ordinal: 206000
card: "electronmac"
result: "V on the Mac mini: universal (x86_64+arm64) ad-hoc signed app, dmg+zip built, packaged app starts and probes a fake server, 37 node tests, gate on the worker host green. NOT verified: real menu bar icon, Gatekeeper dialogs, Intel slice running, GitHub macos CI job (never run). Live game not used."
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
- [x] #1 mac universal dmg+zip config and icns
- [x] #2 tray template image and darwin dock/quit/login behaviours with tests
- [x] #3 CI macos job builds with publish never
- [x] #4 built on the Mac mini, sha256 reported
- [x] #5 docs macOS section
<!-- AC:END -->
