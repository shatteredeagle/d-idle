---
id: TASK-306
title: 'uiedit2: game picture fits its panel (aspect-locked), every watch card editable (Hero now, Up next), hide and show'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T15:56:00Z'
labels:
  - watch
  - bug
dependencies: []
priority: high
ordinal: 211000
card: "uiedit2"
owns:
  - "internal/watch/web/layout.js"
  - "internal/watch/web/layoutedit.js"
  - "internal/watch/web/shell.js"
  - "internal/watch/web/shell.css"
  - "internal/watch/web/panels/video"
  - "internal/watch/web/layout.json"
  - "internal/watch/layout.go"
  - "internal/watch/layout_test.go"
  - "internal/watch/testdata"
  - "tools/uishots"
  - "tools/uishots.sh"
  - "docs/ui-design-system.md"
  - "backlog/merge-lines/agent-uiedit2.txt"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

User feedback on Edit layout, second round (2026-11-10). Done when the acceptance criteria hold and the branch is merged.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 the game picture is contained and centred in its panel at any cell size, the head stays visible, and the editor keeps the video cell 4:3 (hint on the handle)
- [ ] #2 Hero now, Up next and every other card on /watch can be dragged, resized, collapsed and hidden or shown in Edit layout, are saved and take part in the push-down
- [ ] #3 regression tests: every section on /watch has a layout id and an edit frame; video fit; pictures of edit mode
- [ ] #4 watch.layout/1 stays compatible (hidden is an additive key, absent on shown entries); fixtures only added
<!-- AC:END -->
