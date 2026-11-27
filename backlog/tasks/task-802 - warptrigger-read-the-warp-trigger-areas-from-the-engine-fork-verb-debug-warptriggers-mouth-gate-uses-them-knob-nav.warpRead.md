---
id: TASK-802
title: 'warptrigger: read the warp trigger areas from the engine (fork verb debug warptriggers), mouth gate uses them (knob nav.warpRead)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-22T04:21:00Z'
labels:
  - d2d
  - feature
dependencies: []
priority: medium
ordinal: 367000
card: "warptrigger"
owns:
  - "docs/probes/warptrigger-2026-11-22.md"
  - "internal/d2dctl/warptriggers.go"
  - "internal/d2dctl/warptriggers_test.go"
  - "internal/d2dctl/fake/fake.go"
  - "internal/nav/mouthgate.go"
  - "internal/nav/warpread_test.go"
  - "internal/d2map/static.go"
  - "internal/d2dworld/triggers_test.go"
  - "internal/d2denv/env.go"
exclusive:
  - "fork-branch:didle/warp-triggers-20261122"
contracts: []
result: "Fork verb debug warptriggers (read-only, on the allow-list) on private branch didle/warp-triggers-20261122 (code c2a9e006, tests to 98deb79e8, pushed to the private remote): V on the builder (tests/warptriggers_d2d.py, 3 levels, 4 warps): 0 differences against debug warps, 24 listed cells take the warp and 24 cells beside the set do not in real cmd move clicks, state unchanged after reads. Public side offline only: d2dctl client and fake, d2dworld exit areas and the mouth gate by read area behind nav.warpRead (default 0, unchanged behaviour), must-fail tests on the Paladin and Sorceress Tower Cellar 5 walks. Scenario act1.forgotten.countess later run three ways by card mapverify (docs/probes/countess-knobs-2026-11-22.md): default knob GREEN on seeds 11 to 13 on lab slots, warpRead cut mouth-gate refusals 55 to 0. Finding: the engine takes a warp by a click inside its Select box, not by stepping on it, so the trigger is a screen-space rectangle (a parallelogram in cells). Build-specific: none."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Why: review docs/design/fable-review-2026-11-22/map-knowledge.md card 3 and Q3. Done when the verb and the gate use it.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Fork verb debug warptriggers (read-only, ADR 0022) on the private branch, pushed the same day, differential against the engine on 3 levels with 0 differences and state unchanged after reads
- [x] #2 d2dctl client and fake; d2dworld gives exits their trigger areas behind nav.warpRead (default 0)
- [x] #3 Mouth gate uses the read area instead of the 15 and 10 subtile guesses when present; must-fail tests on the Paladin and Sorceress walks
- [x] #4 Scenario act1.forgotten.countess still green with the default knob (done by card mapverify: docs/probes/countess-knobs-2026-11-22.md, Paladin seeds 11 to 13 GREEN with both knobs off, V on lab slots; not the sandbox)
<!-- AC:END -->
