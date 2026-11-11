---
id: TASK-347
title: 'branding: original pixel-art brand assets made with Aseprite, replacing third-party icons'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T22:31:00Z'
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
- [ ] #1 Aseprite scripts generate the mark, icons, ico/icns, banner and card
- [ ] #2 build.sh regenerates and verifies sizes and palette
- [ ] #3 Electron, watch favicon, README, friend guide use the new assets
- [ ] #4 d2d icon set (png, bmp, ico, icns, hero) ready for the fork patch
<!-- AC:END -->
