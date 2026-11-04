---
id: TASK-17
title: "m2-tripwp: leave town by waypoint, portal back to the field"
status: Deferred
assignee: []
created_date: '2026-11-04T00:00:00Z'
updated_date: '2026-11-04T00:00:00Z'
labels:
  - "parked"
  - "soak"
dependencies: []
priority: low
ordinal: 17000
card: "m2-tripwp"
owns:
  - "internal/town/trip.go"
  - "internal/town/trip_test.go"
  - "internal/bot/waypoints_test.go"
  - "docs/tripwp.md"
  - "docs/probes/tripwp/**"
exclusive:
  - "soak-slot:3"
branch: "agent/m2-tripwp"
agent: "m2-tripwp"
result: ""
live: ""
merge_hash: ""
overlap: ""
skip_specs: false
source: "openspec/changes/m2-tripwp"
contracts: ["proposed/m2-tripwp"]
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
xpgap measured (24 virtual hours, 85 town trips): a trip costs about 4 minutes of which 2.4 are the walk back to the field, 14 percent of all time is `Returning to the field` at 11 kills/h, and chicken leave-game restarts are 21 percent of the long kill gaps at 129 s each (docs/d2d-xp-gap.md). The bot walks out of the Rogue Encampment even when the target area's waypoint is active and leaves the game on a chicken when a portal would do. Both are nav and trip policy, not economy.

What changes:
- internal/town/trip.go, internal/bot/waypoints.go: leave town by the waypoint when the target area's waypoint (or the nearest activated one on the way) is active; walk only when none is; return through the open portal when one exists.
- Chicken: portal to town when a scroll or tome charge is held, leave-game only without one (the restart path stays for the no-portal case).
- A paired before/after per docs/ab-design.md (8 seeds per arm, 2 virtual hours): `Returning to the field` share and restarts per hour.
- Delta: early-game-policy ADDED `Leave town by waypoint`.

Parked: same as m2-roam

Gates (from the card):
- paired before/after per docs/ab-design.md: `Returning to the field` at most 7 percent of time (was 14) and bot restarts per virtual hour at most half of before, interval excluding zero, raw in docs/probes/tripwp
- go test -p 1 ./internal/town ./internal/bot -run 'Trip|Waypoint|Chicken' passes including TestLeaveTownByActiveWaypoint and TestChickenPortalsWhenScrollHeld
- go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes; tools/agentreap.sh m2-tripwp leaves nothing

Source: openspec/changes/m2-tripwp/ (kept in place; this task is a copy).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 1.1 Read docs/d2d-xp-gap.md (trip costs), internal/town/trip.go, internal/bot/waypoints.go (the 10 minute failed-trip latch), docs/basics-audit.md E3; failing tests first
- [ ] #2 1.2 Waypoint out of town and portal return; chicken by portal when a scroll is held
- [ ] #3 1.3 Paired before/after; docs/tripwp.md; changecheck, leakcheck, validate; reap
<!-- AC:END -->
