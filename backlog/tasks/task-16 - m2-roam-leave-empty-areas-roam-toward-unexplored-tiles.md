---
id: TASK-16
title: "m2-roam: leave empty areas, roam toward unexplored tiles"
status: Deferred
assignee: []
created_date: '2026-11-04T00:00:00Z'
updated_date: '2026-11-04T00:00:00Z'
labels:
  - "parked"
  - "soak"
dependencies: []
priority: low
ordinal: 16000
card: "m2-roam"
owns:
  - "internal/nav/roam.go"
  - "internal/nav/roam_test.go"
  - "internal/nav/exit.go"
  - "internal/nav/exit_test.go"
  - "internal/bot/goal.go"
  - "internal/bot/emptyarea_test.go"
  - "docs/roam.md"
exclusive:
  - "soak-slot:3"
branch: "agent/m2-roam"
agent: "m2-roam"
result: ""
live: ""
merge_hash: ""
overlap: ""
source: "openspec-final:openspec/changes/m2-roam"
contracts: ["proposed/m2-roam"]
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Walk is 13.8 of the 18 seconds per kill on d2d (77 percent, docs/d2d-soak-full-fix.md) against about 5 on the original: the longest gaps are border ping-pong between Blood Moor, Cold Plains and Cave 1 (`nav exit`, `nav stuck`) and roaming a swept area (`clear killed nothing` 0.30 per kill against 0.09 on the original, docs/research/per-kill-time-split.md). The empty-area exit exists (`bot.emptyAreaSec`) but defaults to off (docs/killfix.md). This is the largest remaining component of the fidelity gap after the economy fixes.

What changes:
- internal/nav/roam.go, exit.go: no return to an area left in the last N minutes unless the plan asks for it; roam toward unexplored, spawn-dense tiles instead of the border.
- internal/bot/goal.go: `bot.emptyAreaSec` default on (value from the KILLFIX measurement, docs/killfix.md) for recovery and leveling sweeps.
- A before/after on one binary, 4 seeds x 2 virtual hours per arm, interleaved (tools/xpgap/ab.sh pattern), walk share and `clear killed nothing` per kill reported.
- Delta: early-game-policy ADDED `Leave an empty area`.

Parked: area-choice work (noarea cards) is archived; revisit after Act 1 runs

Gates (from the card):
- before/after on 4 seeds x 2 virtual hours per arm: walk share of kill time at most 55 percent (was 77) and `clear killed nothing` at most 0.15 per kill (was 0.30), same binary, interleaved arms, raw in docs/probes/roam
- go test -p 1 ./internal/nav ./internal/bot passes; go test ./internal/env -run EveryKnob passes with the new default
- go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes; tools/agentreap.sh m2-roam leaves nothing

Source: git tag openspec-final, openspec/changes/m2-roam/ (this task is the copy).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 1.1 Read docs/d2d-soak-full-fix.md kill-time section, docs/research/per-kill-time-split.md tables 2 and 3, docs/killfix.md; extract the border-crossing and empty-sweep counts of 4 soakfull seeds as the before arm
- [ ] #2 1.2 Roam and exit changes with offline tests on recorded maps (internal/nav fixtures)
- [ ] #3 1.3 emptyAreaSec default on; EveryKnob test; docs/knobs.md regenerated
- [ ] #4 1.4 After arm on the same seeds; docs/roam.md with both tables; changecheck, leakcheck, validate; reap
<!-- AC:END -->
