---
id: TASK-234
title: >-
  uicommand: watch command centre above the fold, full-width stage, restyled
  cards, screenshot suite
status: Done
assignee:
  - credit-runner
created_date: '2026-11-09T20:16:00Z'
updated_date: '2026-11-09T20:57:00Z'
labels:
  - watch
  - feature
  - T0
dependencies:
  - TASK-231
priority: high
ordinal: 178000
card: "uicommand"
owns:
  - "internal/watch/hero.go"
  - "internal/watch/whychange.json"
  - "internal/watch/web/panels/"
  - "internal/watch/web/shell.css"
  - "internal/watch/web/shell.js"
  - "internal/watch/web/shell.json"
  - "internal/watch/web/app.css"
  - "internal/watch/web/app.js"
  - "internal/watch/web/watch.html"
  - "internal/watch/web/components.css"
  - "internal/watch/panels.go"
  - "tools/uishots/"
  - "tools/uishots.sh"
  - "tools/uilint/"
  - "docs/ui-design-system.md"
exclusive: []
contracts: []
result: "Command centre beside a full-width game video in one 12-column grid (doing now, hero vitals with life/mana/xp numbers and gold, map, gear chips with Repair now, three-line Why, tally, marks); GET /api/hero (read-only); whyline, tally and currency strips out of the top band; old cards restyled and packed under the video. Verified: go test ./internal/watch ./tools/uilint ./tools/uishots, whole-repo vet+test on the worker host, tools/uishots.sh renders (headless Chrome, private Xvfb, frozen fixture viewer) looked at 1440 and 1920. Preview deployed; not seen live by the agent (token). Build-specific: none."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

User layout feedback 2026-11-09 on the first shell preview (black page, dead space, nothing useful above the fold). Done when the acceptance criteria hold and the preview is up.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 above the fold at 1280 to 1920 px: stage left 7 of 12 columns edge to edge, command centre right (hero vitals with life, mana, xp numbers and gold, doing now with order progress and the next 3, Why card with three lines, gear, tally, marks); stacks under 900 px
- [x] #2 GET /api/hero is read-only and composes only what the bot already reports
- [x] #3 the whyline and the tally and currency strips are out of the top band
- [x] #4 the pre-shell cards are grid cells with the component look in the mockup order; no endpoint or behaviour change
- [x] #5 tools/uishots.sh renders a frozen fixture viewer with headless Chrome on a private Xvfb on the worker host and checks the DOM; preview deployed
<!-- AC:END -->
