---
id: TASK-970
title: >-
  admit1: P4 admission function MayEnter(area, purpose) and
  MayLeaveTown(purpose)
status: Done
assignee:
  - credit-runner
created_date: '2026-11-26T03:53:00Z'
updated_date: '2026-11-27T20:57:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 459000
card: "admit1"
owns:
  - "docs/design/admission-p4.md"
  - "internal/admit/"
  - "internal/plandata/admission.json"
  - "internal/plandata/earlyroutes.json"
  - "internal/plandata/povertyfarm.json"
  - "docs/plan/data/admission.json"
  - "docs/plan/data/earlyroutes.json"
  - "docs/plan/data/povertyfarm.json"
  - "internal/bot/earlyroute.go"
  - "internal/bot/povertyfarm.go"
  - "internal/bot/admit_need_test.go"
  - "internal/bot/admission.go"
  - "internal/bot/admission_test.go"
  - "internal/bot/routeguard.go"
  - "internal/bot/runner.go"
  - "internal/bot/explain.go"
  - "internal/env/tuning_bot.go"
  - "data/darklaunch.json"
exclusive: []
contracts: []
result: "Step 1: package internal/admit (MayEnter, MayLeaveTown, Refused with need and wake), floors as rows in admission.json, old judges read their numbers from it, table tests for two classes. Not verified live. Step 2: wiring in shadow (internal/bot/admission.go) for early road, poverty, kit floor, lean and the quest-walk kit; knob bot.admission (default 0 = old judges decide, 1 = function decides), dark-launch row, ops status --why admission line, property test. Offline only, not verified live; other providers not wired."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Cause classes killed: G (thresholds by incident, 20) and C at the edges; the 'left town hurt or without kit' invariants. Roadmap row P4 (docs/research/fable-logic-review-2026-11-25.md 7.1). One function with a verdict that is data (ok, or Refused{need}); the eleven judges (gear gate, kit gate, route guard, early road, Barracks, Den and Andariel floors, the Forgotten keep floor, readiness, quest floors, bans, zone model shadow) become providers behind it, called at the travel chokepoint and at the town exit. Floors are rows keyed by area and archetype (as plandata/earlyroutes.json is), no function per place, no class literal. Do it provider by provider with the shadow-then-flip habit; one ledger row per provider knob. Deletes: barracksGuardWhy, the keep guard function, denFloor, andarielLow, kitSafeTier and dungeonMouths literals. Depends on P2 (Refused). Risk: medium; this is ADR 0011's single entry point. Order: after outcome1 and invmon; before threshold rows move (thrgov). Enable criteria per provider: shadow replay verdict equals the old judge or is right in a reviewed sample, then flip with a 24 h watch and auto-revert, and the invariant row 'field entry under resume line' does not rise. Will split into several cards at start.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Design doc: provider list, row schema keyed by archetype, shadow plan
- [x] #2 First provider (town exit) in shadow with a ledger row
<!-- AC:END -->
