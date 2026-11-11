---
id: TASK-279
title: 'uilayout: the shell reads a watch.layout/1 manifest, /api/layout, Edit layout mode on /watch'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T00:07:00Z'
labels:
  - watch
  - feature
dependencies: []
priority: high
ordinal: 203000
card: "uilayout"
owns:
  - "internal/watch/layout.go"
  - "internal/watch/layout_test.go"
  - "internal/watch/web/shell.js"
  - "internal/watch/web/shell.css"
  - "internal/watch/web/shell.json"
  - "internal/watch/web/layout.json"
  - "internal/watch/web/layouts"
  - "internal/watch/web/layoutedit.js"
  - "internal/watch/web/watch.html"
  - "tools/uilint"
  - "tools/uishots"
  - "tools/uishots.sh"
  - "docs/ui-design-system.md"
  - "docs/ui-regression-plan.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Migration PR 14 of docs/ui-regression-plan.md: the layout the user shaped in the mockup editor is the layout the real shell loads. Done when the acceptance criteria hold and the branch is merged.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 the shell places panels from web/layout.json (per breakpoint col, row, colSpan, rowSpan, collapsed, order) and falls back to manifest priorities without one
- [ ] #2 a file exported by docs/mockups/ui/editor.html loads unchanged (editor ids map onto the real panels)
- [ ] #3 GET /api/layout serves the layout; POST /api/layout (answer listener, policy gate) stores a per-profile copy
- [ ] #4 Edit layout on /watch loads web/layoutedit.js only in that mode: drag, resize, keyboard moves, Save, Reset to default
- [ ] #5 uilint validates layout.json and the presets: known ids, no overlap per breakpoint, spans within the columns
- [ ] #6 golden fixtures only added; Go, node DOM and JS pattern tests pass; the screenshot suite renders the three presets
<!-- AC:END -->
