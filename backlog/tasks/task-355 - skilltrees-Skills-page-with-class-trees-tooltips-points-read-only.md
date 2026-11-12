---
id: TASK-355
title: 'skilltrees: Skills page with class trees, tooltips, points (read only)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-12T03:13:00Z'
labels:
  - watch
  - feature
dependencies: []
priority: medium
ordinal: 231000
card: "skilltrees"
owns:
  - "internal/skilltree"
  - "internal/watch/skillsapi.go"
  - "internal/watch/skillsapi_test.go"
  - "internal/watch/web/skills.js"
  - "internal/watch/web/skills.css"
  - "internal/watch/skillsui.go"
  - "docs/skilltrees.md"
  - "cmd/d2bot/skillswire.go"
  - "internal/art/skillicons.go"
  - "internal/art/skillicons_test.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 internal/skilltree model: trees, nodes, canAdd with reason, two classes tested
- [ ] #2 Skill icons from the user's own MPQs via internal/art, text tile fallback, nothing committed
- [ ] #3 GET /api/skills (cached, versioned) and /watch/skills page with hover tooltips; allocate route is a 501 stub behind skills.allocate
- [ ] #4 docs/skilltrees.md with V/U tags and Future: allocation
- [ ] #5 Gates: package tests, race on internal/watch, contracts/backlog checks, leakcheck
<!-- AC:END -->
