---
id: TASK-323
title: 'nocursor: the watch video shows no game pointer (fork verb cursor off, d2dview asks for it)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T19:10:00Z'
labels:
  - watch
  - feature
dependencies: []
priority: medium
ordinal: 218000
card: "nocursor"
owns:
  - "cmd/d2dview"
  - "docs/viewer-fps.md"
  - "backlog/merge-lines/agent-nocursor.txt"
exclusive:
  - "fork-branch:d2d.lab3-nocursor"
contracts: []
result: "Fork: draw-only verb cursor on|off, built and tested on the worker host (headless throwaway profile, no character), pushed as d2d.lab3-nocursor c418e235. Viewer: -hide-cursor (default true) sends it, the viewer socket lets it through, an older d2d is left alone. Not verified: the two together on the sandbox; the sandbox image was not rebuilt or touched (steps in docs/viewer-fps.md section 8)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

User request 2026-11-11: hide the game's pointer in the watch video. Done when the acceptance criteria hold and the branch is merged; the sandbox image rebuild is the coordinator's.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 the fork has a draw-only cursor on|off verb, built and tested, pushed to the private remote
- [x] #2 d2dview sends cursor off by default (-hide-cursor), lets the verb through the viewer socket and leaves a d2d without it alone
- [x] #3 docs/viewer-fps.md names the pushed fork commit and the image rebuild steps
<!-- AC:END -->
