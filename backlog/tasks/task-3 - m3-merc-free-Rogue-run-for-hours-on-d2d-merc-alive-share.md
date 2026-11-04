---
id: TASK-3
title: "m3-merc: free Rogue run for hours on d2d, merc alive share"
status: Proposed
assignee: []
created_date: '2026-11-04T00:00:00Z'
updated_date: '2026-11-04T00:00:00Z'
labels:
  - "G-B"
  - "soak"
dependencies: []
priority: high
ordinal: 3000
card: "m3-merc"
owns:
  - "internal/merc/**"
  - "internal/bot/merc_test.go"
  - "docs/merc-d2d.md"
  - "docs/probes/merc/**"
exclusive:
  - "soak-slot:3"
branch: "agent/m3-merc"
agent: "m3-merc"
result: ""
live: ""
merge_hash: ""
overlap: ""
skip_specs: false
source: "openspec/changes/m3-merc"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The free Rogue is coded (merc-policy spec, docs/merc-wire.md) behind `merc.enabled` default 0 and was never run for hours on d2d; the route guard needs a living merc to enter the Monastery road below its level floor, so without a merc that works for hours the Act 1 clear waits for clvl 18 or more (d2-sme 2026-10-31). This card makes the claim, the keep-alive and the revive work on d2d for hours and turns the knob on for d2d.

What changes:
- internal/merc: revive per the revive rule (merc-policy), stop rule, potions to the merc below the knob; the claim once after Blood Raven; log lines with price and gold delta.
- (resource.go: sequenced after fix-portal (HasTP now counts a portal source); re-add when m3-merc starts.) Wanted change in internal/bot/resource.go: `merc.enabled` default 1 on the d2d backend (Game.exe default unchanged, frozen).
- docs/merc-d2d.md: 2 seeds x 4 virtual hours from clvl 9 (cheat-levelled, ledgered) after Blood Raven: uptime, revives, gold, what killed her.
- Blood Raven bailout: after 3 failed attempts or at clvl 12 without the kill she counts as skipped for good (`BloodRavenSkipped`) and the paid Rogue fallback applies.
- Revive only when gold minus the kit reserve >= `merc.reviveGoldMult` (2) x price; `merc.reviveMaxPerHour` (2), `merc.stopGoldMult` (5), `merc.stopDeathsPerHour` (3) unchanged.
- Keeper versus throwaway: a build's target merc or a merc wearing target gear is a keeper (always revived); a throwaway is re-hired when the hire price is below the revive price. Until builds define a target merc the bot always revives.

Next step: the soak (revive knob and hire drill already archived)

Gates (from the card):
- 2 seeds x 4 virtual hours from clvl 9 after Blood Raven on hidden d2d: the Rogue claimed exactly once per seed, 0 paid Act 1 hires, merc alive at least 70 percent of field time, every revive logged with price and gold delta within the revive rule (raw in docs/probes/merc)
- go test -p 1 ./internal/merc ./internal/bot -run 'Merc|Claim' passes; go test ./internal/env -run EveryKnob passes with the d2d default
- go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes; tools/agentreap.sh m3-merc leaves nothing

Source: openspec/changes/m3-merc/ (kept in place; this task is a copy).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 1.1 Read docs/merc-wire.md, docs/merc.md, merc-policy spec, ADR 0011 and the grill 0003 outcome; failing tests first (claim once, revive rule with kit reserve, stop rule, Blood Raven bailout at 3 failures or clvl 12, keeper versus throwaway)
- [ ] #2 1.2 Revive, stop rule and merc potions wired for d2d; bailout and keeper/throwaway rule (always revive until a build names a target merc); knob default per backend
- [ ] #3 1.3 2 x 4 virtual hour runs; docs/merc-d2d.md; changecheck, leakcheck, validate; reap
- [ ] #4 1.4 Open questions kept in the proposal: build merc targets and the live probes (dead-merc forfeit, revive quote, hire windows, merc Str/Dex, merc state after reload); note which the 2 x 4 hour runs answered
<!-- AC:END -->
