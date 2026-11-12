---
id: TASK-381
title: 'tidy: auto pack and stash tidying'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-12T20:18:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 242000
card: "tidy"
owns:
  - "internal/tidy"
  - "internal/town/tidy.go"
  - "internal/town/tidy_test.go"
  - "docs/tidy.md"
  - "cmd/d2bot/tidywire_test.go"
  - "internal/env/tuning_town.go"
  - "internal/invdesk/desk.go"
  - "cmd/d2bot/invwire.go"
  - "internal/town/chores.go"
  - "internal/town/town.go"
  - "docs/knobs.md"
  - "internal/loot/ignore_test.go"
exclusive: []
contracts: []
result: "offline only, NOT verified live. Policy table + Tidy chore (stash/drop) + tooltip field + 3 knobs, wired via wireInv (test in cmd/d2bot). Town sim tests skip without game tables (U). Build-specific part: policy.json overrides per class (Amazon javelins)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the bot should keep pack and stash tidy by a data table, never touching equipment or marked items (user go 2026-11-12). Done when: the Tidy chore runs in town, the tooltip says what will happen, docs/tidy.md.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Pure policy table (internal/tidy) with per-class rows and tests for two classes
- [x] #2 Tidy chore stashes by policy and drops pure junk from a full pack, never equipment or marked items
- [x] #3 Tooltip line (tidy field) and knobs town.tidy*
- [x] #4 Wired in production (wireInv) and covered by a cmd/d2bot test
- [x] #5 docs/tidy.md with V/U tags and a live-check list
<!-- AC:END -->
