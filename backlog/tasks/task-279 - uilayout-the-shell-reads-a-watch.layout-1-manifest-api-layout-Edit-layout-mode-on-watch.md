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
  - "internal/watch/web/layout.js"
  - "internal/watch/http_view.go"
  - "internal/watch/http_answer.go"
  - "internal/watch/hub.go"
  - "internal/watch/panels.go"
  - "internal/watch/testdata/layout_plan.js"
  - "internal/watch/testdata/compat"
  - "cmd/d2bot/funwire.go"
  - "backlog/merge-lines/agent-uilayout.txt"
exclusive: []
contracts: []
result: "Built: the shell places from watch.layout/1 (GET /api/layout, presets by ?layout=), editor exports load unchanged through the id mapping of shell.json, POST /api/layout stores layout-<profile>.json, Edit layout mode, uilint layout rules, three compat layout fixtures added. Verified by go test (routes, check, node plan test, JS patterns) and by headless Chrome pictures on the worker host (default layout at five widths, three presets, the edit mode). NOT verified by hand in a browser: drag, resize, keyboard moves and Save in Edit layout. The acceptance box 4 is ticked for what the tests and the picture prove; the pointer interaction is unverified."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Migration PR 14 of docs/ui-regression-plan.md: the layout the user shaped in the mockup editor is the layout the real shell loads. Done when the acceptance criteria hold and the branch is merged.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 the shell places panels from web/layout.json (per breakpoint col, row, colSpan, rowSpan, collapsed, order) and falls back to manifest priorities without one
- [x] #2 a file exported by docs/mockups/ui/editor.html loads unchanged (editor ids map onto the real panels)
- [x] #3 GET /api/layout serves the layout; POST /api/layout (answer listener, policy gate) stores a per-profile copy
- [x] #4 Edit layout on /watch loads web/layoutedit.js only in that mode: drag, resize, keyboard moves, Save, Reset to default
- [x] #5 uilint validates layout.json and the presets: known ids, no overlap per breakpoint, spans within the columns
- [x] #6 golden fixtures only added; Go, node DOM and JS pattern tests pass; the screenshot suite renders the three presets
<!-- AC:END -->
