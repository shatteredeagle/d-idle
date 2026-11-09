---
id: TASK-231
title: 'uishell2: watch shell with panel registry; status band, video+map stage and whyline as panels'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-09T19:27:00Z'
labels:
  - watch
  - feature
  - T0
dependencies:
  - TASK-230
priority: high
ordinal: 177000
card: "uishell2"
owns:
  - "internal/watch/web/shell.css"
  - "internal/watch/web/shell.js"
  - "internal/watch/web/shell.json"
  - "internal/watch/web/panels/"
  - "internal/watch/panels.go"
  - "internal/watch/panels_test.go"
  - "internal/watch/web/watch.html"
  - "internal/watch/web/app.js"
  - "internal/watch/web/app.css"
  - "internal/watch/web/senses.js"
  - "internal/watch/senses.go"
  - "internal/watch/senses_test.go"
  - "internal/watch/watchlayout_test.go"
  - "internal/watch/whyline_test.go"
  - "internal/watch/uitokens_test.go"
  - "internal/watch/http_view.go"
  - "internal/watch/testdata/"
  - "docs/ui-design-system.md"
  - "docs/senses-map.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Second migration step (docs/ui-regression-plan.md section 5, PR 4 and the status and stage part of PR 7a). Done when the acceptance criteria hold and the preview is up.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 shell.js places panels from /watch/panels.json (manifests embedded from web/panels/*/panel.json) into the status, stage and grid bands, with one fetch scheduler (SSE change plus per-panel polls), focus keeping and a ctx API
- [ ] #2 the status band (hero status, links, theme), the video and map stage and the whyline are panel modules; the pre-shell markup and code for them are gone
- [ ] #3 every other card works unchanged in the compatibility slot of the grid band (no endpoint or behaviour change)
- [ ] #4 ?mode=overlay renders the status band alone (overlay is a mode of the shell)
- [ ] #5 uilint passes, node tests of the video and map panels pass, preview deployed to the sandbox container
<!-- AC:END -->
