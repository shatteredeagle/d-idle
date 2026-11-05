---
id: TASK-21
title: "t0-slice: wire T0 decision points into combat with the untrained default"
status: Proposed
assignee: []
created_date: '2026-11-05T00:00:00Z'
updated_date: '2026-11-05T00:00:00Z'
labels:
  - "G-G"
dependencies: []
priority: high
ordinal: 21000
card: "t0-slice"
owns:
  - "internal/decide/t0/**"
  - "internal/bot/thoughts.go"
  - "internal/bot/thoughts_test.go"
  - "internal/combat/thoughtshook.go"
  - "docs/t0-slice.md"
exclusive: []
branch: "agent/t0-slice"
agent: "t0-slice"
result: ""
live: ""
merge_hash: ""
overlap: "internal/bot/runner.go, task.go and internal/resource are owned by open cards task-10, task-11, task-18: sequence the hook after them or add it as a new file with one call line."
source: "grill 0005 outcome (docs/grills/0005-automation-ladder/outcome.md), ADR 0016 item 24"
contracts: ["combat-loop","bot-loop"]
next: "tier order: after T4/Gate A cards per ADR 0016"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The first vertical slice of the ladder: the four T0 points (t0.retreat.drink, t0.retreat.lethal, t0.area.toohard, t0.retreat.repeat) called from the combat and retreat code, with the untrained default per spec section 5 (depends on kit alone) and a thought opened while the hero carries on. Proves ADR 0016's open item 'the hero can work while a thought is open' (item 10). Uses the store of decision-point-store; touches existing retreat code only through a small hook (call site in internal/bot or internal/resource, to be named in the card; those files are owned by open fix cards, see overlap). Drill: fresh clvl 1, scripted answers across answer profiles (3 known cautious/bold/inconsistent, 2 holdouts from a seed), 10 virtual minutes in Blood Moor, no loop and no stall for any answer, writes after start 0. Order note: tier order: after T4/Gate A cards per ADR 0016 (the four T4 and Gate A cards m3-act1run, m3-andariel, m3-merc, gameexe-retire-soak wait their turn). Ladder work is product work, outside the tooling freeze (ADR 0016 item 3). Tags: V read in repo, U unverified.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Hook points placed behind the store interface; with no rule file the behaviour equals today's defaults (regression test on the listed contexts)
- [ ] #2 T0 drill: fresh clvl 1, order 'Blood Moor', 10 virtual minutes, scripted answers across 3 known profiles and 2 seeded holdouts; invariants hold for every profile (no loop, no crash, no stall); a thought opened mid-fight does not pause the hero
- [ ] #3 Run record carries the rule-file hash, rules learned, answers given, thoughts faded, and the line 'writes after start: 0'
- [ ] #4 Two builds: the default Sorceress and one other class or build; class-specific data in profile tables
- [ ] #5 go vet and go test -p 1 of the packages; leakcheck 0; docs/t0-slice.md; say verified live or not
<!-- AC:END -->
