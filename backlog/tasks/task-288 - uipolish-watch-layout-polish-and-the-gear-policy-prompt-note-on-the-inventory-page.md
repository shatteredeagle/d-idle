---
id: TASK-288
title: 'uipolish: watch layout polish and the gear-policy prompt note on the inventory page'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T01:14:00Z'
labels:
  - watch
  - feature
dependencies: []
priority: medium
ordinal: 205000
card: "uipolish"
owns:
  - "internal/watch/web/layout.json"
  - "tools/uishots"
  - "internal/watch/invui.go"
  - "internal/watch/web/inventory.js"
  - "internal/watch/gearprompt_test.go"
  - "docs/ui-design-system.md"
  - "backlog/merge-lines/agent-uipolish.txt"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Finishing polish of the watch UI migration plus the gear-policy prompt (TASK-280 follow-up). Done when the acceptance criteria hold and the branch is merged.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 the tablet layout leaves no half-width panel alone on a row
- [ ] #2 the inventory page shows View.prompt as a note above the equipment (role=note, info or warn, hidden when the prompt is gone, no focus or scroll change)
- [ ] #3 pictures of the five widths and the three presets were looked at; Go and node tests pass
<!-- AC:END -->
