---
id: TASK-373
title: 'archify: Go import-graph diagrams of d.idle (tools/arch)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-12T18:08:00Z'
labels:
  - ops
  - chore
dependencies: []
priority: medium
ordinal: 239000
card: "archify"
owns:
  - "tools/arch"
  - "docs/arch.md"
exclusive: []
contracts: []
result: "tools/arch/arch.sh (gen.py + pinned Archify 7e45f36c, Apache-2.0) renders overview, package and --diff drift HTML from go list into gitignored docs/arch/out; offline except the one-time fetch; screenshot-checked in headless Firefox on the worker host; not verified live in the game (n/a)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 tools/arch/arch.sh renders overview and package diagrams and a --diff drift view into a gitignored dir
- [x] #2 docs/arch.md records usage, pinned Archify commit, licence, trust notes and blind spots
- [x] #3 first diagram opened and looked at (headless Firefox screenshot)
<!-- AC:END -->
