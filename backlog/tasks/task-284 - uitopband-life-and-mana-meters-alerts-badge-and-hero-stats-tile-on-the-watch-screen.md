---
id: TASK-284
title: 'uitopband: life and mana meters, alerts badge and hero stats tile on the watch screen'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T01:04:00Z'
labels:
  - watch
  - feature
dependencies: []
priority: high
ordinal: 204000
card: "uitopband"
owns:
  - "internal/watch/web/panels/hero-meters"
  - "internal/watch/web/panels/alerts-count"
  - "internal/watch/web/panels/hero-detail"
  - "internal/watch/hero.go"
  - "internal/watch/hub.go"
  - "internal/watch/topband_test.go"
  - "internal/watch/testdata"
  - "internal/watch/panels_test.go"
  - "internal/watch/whyline_test.go"
  - "internal/watch/web/shell.json"
  - "internal/watch/web/shell.css"
  - "internal/watch/web/layout.json"
  - "tools/uishots"
  - "cmd/d2bot/readywire.go"
  - "cmd/d2bot/ladder.go"
  - "cmd/d2bot/live.go"
  - "cmd/d2bot/d2dbackend.go"
  - "internal/bot/readinessfacts.go"
  - "docs/ui-design-system.md"
  - "backlog/merge-lines/agent-uitopband.txt"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

The top status band pieces of the approved mockup that were still missing. Done when the acceptance criteria hold and the branch is merged.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 life and mana meters with numbers sit in the top band (new panel hero-meters, /api/hero)
- [ ] #2 an alerts badge in the top band counts the alerts and links to the Alerts card (new panel alerts-count, /api/alerts)
- [ ] #3 a hero stats tile shows level, the way to the next level, gold, resistances and the readiness light (new panel hero-detail; /api/hero gains a stats block, null until reported)
- [ ] #4 existing panels are untouched; compat fixtures only added; node DOM goldens and Go tests pass for Sorceress and Paladin
<!-- AC:END -->
