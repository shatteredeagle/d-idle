---
id: TASK-347
title: >-
  branding: original pixel-art brand assets made with Aseprite, replacing
  third-party icons
status: Done
assignee:
  - credit-runner
created_date: '2026-11-11T22:31:00Z'
updated_date: '2026-11-12T01:38:00Z'
labels:
  - docs
  - chore
dependencies: []
priority: medium
ordinal: 228000
card: "branding"
owns:
  - "assets/branding/"
  - "tools/branding/"
  - "docs/branding.md"
  - "apps/desktop-electron/build/"
  - "apps/desktop-electron/scripts/make-icons.js"
  - "apps/desktop-electron/package.json"
  - "internal/watch/web/watch.html"
  - "README.md"
  - "docs/friend-guide.md"
  - "docs/desktop-electron.md"
exclusive: []
result: "Pixel-art set drawn in assets/branding/src/didle.lua with Aseprite 1.3.18 batch on the worker host (V: ran, viewed 16/32/48/256/banner/card); packed by tools/branding/pack.py; Electron, watch favicon, README, friend guide wired; d2d set in out/d2d for the fork patch (fork not touched, not built). CC0 1.0 artwork. Not verified: real macOS menu bar, Windows shell, d2d build."
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Aseprite scripts generate the mark, icons, ico/icns, banner and card
- [x] #2 build.sh regenerates and verifies sizes and palette
- [x] #3 Electron, watch favicon, README, friend guide use the new assets
- [x] #4 d2d icon set (png, bmp, ico, icns, hero) ready for the fork patch
<!-- AC:END -->
