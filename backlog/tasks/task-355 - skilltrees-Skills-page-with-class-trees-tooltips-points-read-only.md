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
  - "internal/watch/testdata/skills_paladin.json"
  - "internal/watch/testdata/skills_sorceress.json"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the user asked for a way to see the skill trees (a tab like Inventory, hover tooltips, where points are allocated). Read only; builds the model so a later allocate action needs only an implementation. Small additive edits outside the owned files: internal/data/skills.go and schema.go (IconCel, short and long description keys, cache schema 5), internal/art/store.go (key grammar), internal/watch/funwire.go (Fun.Skills mount), watch.html and invui.go (nav link), cmd/d2bot ladder.go, d2dbackend.go, live.go (wiring). Done when: docs/skilltrees.md lists V and U; result: below.

result: Skills page /watch/skills, GET /api/skills (watch.skills/1, cached 2 s), internal/skilltree model with CanAdd reasons, skill icons from the user's own MPQs (art key skill-<class>-<cel>), POST /api/skills/allocate answers 501 behind the skills.allocate flag (off). Verified offline: package tests (Paladin and Sorceress), goldens, and a headless render at 1400 and 390 px against a real install on the worker host. NOT verified live: hero data on a running bot, tooltip hover in a real browser.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 internal/skilltree model: trees, nodes, canAdd with reason, two classes tested
- [x] #2 Skill icons from the user's own MPQs via internal/art, text tile fallback, nothing committed
- [x] #3 GET /api/skills (cached, versioned) and /watch/skills page with hover tooltips; allocate route is a 501 stub behind skills.allocate
- [x] #4 docs/skilltrees.md with V/U tags and Future: allocation
- [x] #5 Gates: package tests, race on internal/watch, contracts/backlog checks, leakcheck
<!-- AC:END -->
