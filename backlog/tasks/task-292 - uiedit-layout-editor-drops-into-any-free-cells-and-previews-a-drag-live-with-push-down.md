---
id: TASK-292
title: >-
  uiedit: layout editor drops into any free cells and previews a drag live with
  push-down
status: Done
assignee:
  - credit-runner
created_date: '2026-11-11T02:13:00Z'
updated_date: '2026-11-11T16:25:00Z'
labels:
  - watch
  - bug
dependencies: []
priority: high
ordinal: 207000
card: "uiedit"
owns:
  - "internal/watch/web/layout.js"
  - "internal/watch/web/layoutedit.js"
  - "internal/watch/web/shell.js"
  - "internal/watch/web/shell.css"
  - "internal/watch/layout_test.go"
  - "internal/watch/testdata"
  - "tools/uishots"
  - "tools/uishots.sh"
  - "docs/ui-design-system.md"
  - "backlog/merge-lines/agent-uiedit.txt"
exclusive: []
contracts: []
result: "Bug: hidden panels reserved cells invisibly in edit mode; they reserve nothing now and are still saved. Feature: live push-down preview for drag, resize and keyboard (Enter commits, Escape reverts), slide honours reduced motion, preview never touches the layout until a drop and the stored layout until Save. Verified by node tests through go test (algorithm and the editor handlers in a stub DOM) and a headless picture of a drag in progress. Not verified: a real mouse drag in a browser; the cause of the bug was reproduced in the test, not re-observed in the user's browser."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

User feedback on Edit layout (2026-11-10). Done when the acceptance criteria hold and the branch is merged.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 a panel (the minimap) can be dropped exactly into any free cells; hidden panels reserve nothing; regression test
- [x] #2 while dragging or resizing, overlapped panels are pushed down with a cascade as a live preview, with a slide that honours reduced motion
- [x] #3 Escape, a release outside the grid or moving back reverts; a drop commits (pending Save); the keyboard previews until Enter
- [x] #4 watch.layout/1 and the export format are unchanged; edit mode only; node tests for overlap, cascade, revert, drop, no-op
<!-- AC:END -->
