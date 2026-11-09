---
id: TASK-235
title: 'layouteditor: draggable layout editor for the watch screen mockup, layout manifest schema, editor renders'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-09T20:24:00Z'
updated_date: '2026-11-09T20:29:00Z'
labels:
  - watch
  - research
dependencies: []
priority: high
ordinal: 178000
card: "layouteditor"
owns:
  - "docs/mockups/ui/"
  - "docs/ui-design-system.md"
  - "docs/ui-regression-plan.md"
exclusive: []
contracts: []
result: 'Done (docs/static only): docs/mockups/ui/editor.html, vanilla JS, no requests: 22 real sample panels on a 12-column snapping grid, drag by title bar, corner resize, arrows/Shift+arrows/C/Delete, five viewports with a layout each, palette add/remove/collapse, dense/comfortable, dark/light, undo/redo, reset, presets (mockup, live, command centre), localStorage autosave, export/import/copy JSON, text summary, fit-to-width, URL params bp/preset/theme/density. Layout manifest watch.layout/1 documented in ui-design-system.md 6.4; PR 14 uilayout added to the migration plan. Five editor renders from headless Chrome under Xvfb on the worker host, checked by eye (wide, phone viewport). Served from the draft dir on port 8899. UNVERIFIED: not driven by a real pointer in a browser (only rendered); drag maths checked by reading.'
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

User request 2026-11-09: a draggable mock UI to position the real panels and iterate without back and forth. Done when the editor is merged and served and its JSON is documented as the layout manifest.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 docs/mockups/ui/editor.html: standalone vanilla-JS editor, 12-column grid with snapping, drag and resize handles, keyboard moves, viewport switcher (phone 390, tablet 768, laptop 1280, wide 1920, overlay) with a layout per breakpoint, panel palette add/remove/collapse, dense/comfortable, dark/light, undo/redo, reset, presets (live, mockup, command centre), localStorage autosave, export/import JSON, copy JSON, text summary
- [x] #2 docs/ui-design-system.md documents the layout manifest (watch.layout/1: panel id, band, per-breakpoint col/row/span, collapsed, order) in the shape the shell will consume; ui-regression-plan.md gains the PR 'shell reads a layout manifest + Edit layout mode'
- [x] #3 editor screenshots in docs/mockups/ui/renders/ and the editor served from the worker host draft directory
- [x] #4 hygiene gate, merge_lines_check, leakcheck and backlog check pass
<!-- AC:END -->
