---
id: TASK-292
title: 'uiedit: layout editor drops into any free cells and previews a drag live with push-down'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T02:13:00Z'
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
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

User feedback on Edit layout (2026-11-10). Done when the acceptance criteria hold and the branch is merged.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 a panel (the minimap) can be dropped exactly into any free cells; hidden panels reserve nothing; regression test
- [ ] #2 while dragging or resizing, overlapped panels are pushed down with a cascade as a live preview, with a slide that honours reduced motion
- [ ] #3 Escape, a release outside the grid or moving back reverts; a drop commits (pending Save); the keyboard previews until Enter
- [ ] #4 watch.layout/1 and the export format are unchanged; edit mode only; node tests for overlap, cascade, revert, drop, no-op
<!-- AC:END -->
