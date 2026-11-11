---
id: TASK-315
title: 'uifit: watch panels fit their content (rows measured from the card, game card height from its width)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T18:50:00Z'
labels:
  - watch
  - bug
dependencies: []
priority: high
ordinal: 218000
card: "uifit"
owns:
  - "internal/watch/web/layout.js"
  - "internal/watch/web/shell.js"
  - "internal/watch/web/shell.css"
  - "internal/watch/web/shell.json"
  - "internal/watch/web/layoutedit.js"
  - "internal/watch/web/panels"
  - "internal/watch/panels.go"
  - "internal/watch/layout.go"
  - "internal/watch/layout_test.go"
  - "internal/watch/testdata"
  - "tools/uilint"
  - "tools/uishots"
  - "docs/ui-design-system.md"
  - "backlog/merge-lines/agent-uifit.txt"
exclusive: []
contracts: []
result: "Rows of fit panels (all main-area panels and pre-shell cards) are measured from the rendered content and planned with; the stored span is only the fallback before the first measure. The flex rules of #572 never applied (layer order) and are removed. Verified by go test (node layout_fit.js, manifests) and headless pictures at 1280, 1440 and 1920 with a uishots check on the video rows. Not verified: a live stream in a real browser."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

User report 2026-11-11: the Game card stayed taller than its picture. Done when the acceptance criteria hold and the branch is merged.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 the video panel's height follows its width at render time whatever row span is stored; old layouts and editor exports load unchanged
- [x] #2 panels that declare fit (additive manifest key, versions raised) and the pre-shell cards take the rows of their content; panels below rise
- [x] #3 node regression test (stored span too tall at width 600 leaves no empty area) and a uishots check on the video rows
<!-- AC:END -->
