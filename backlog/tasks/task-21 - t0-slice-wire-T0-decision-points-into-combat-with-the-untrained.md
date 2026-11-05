---
id: TASK-21
title: "t0-slice: wire T0 decision points into combat with the untrained default"
status: In Progress
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
  - "internal/ladder/**"
  - "internal/env/tuning_ladder.go"
  - "internal/town/ladder_test.go"
  - "internal/drill/t0.go"
  - "internal/drill/t0_test.go"
  - "cmd/d2drill/t0_lab.go"
  - "cmd/d2drill/t0_lab_test.go"
  - "docs/probes/t0drill.md"
  - "internal/bot/thoughts.go"
  - "internal/bot/thoughts_test.go"
  - "internal/combat/thoughtshook.go"
  - "docs/t0-slice.md"
exclusive: []
branch: "agent/t0-slice"
agent: "t0-slice"
result: "Phase A (offline, branch agent/t0slice, flag ladder.enabled default 0): glue package internal/ladder, call sites in internal/town (repairDue, tpBuyAllowed) and bot.EnableLadder, golden parity tests, refusal witnesses, 5 profiles x 2 builds, 30 random scripts, T0 drill group in cmd/d2drill on a fake lab. Wired t1.repair and t1.tp.lost; NO T0 point passed parity (t0.retreat.drink refused with a witness, t0.retreat.repeat has no legacy test), so AC 1 and 2 are NOT met for T0 and the drill is a town drill. Not verified live (phase B: docs/probes/t0drill.md). Phase A2 (agent/t0slice2): internal/ladder now uses the public decide API (AskWithDefault with the legacy answer as the untrained default, Subscribe/Event.Info, decide.KnownProfiles/Holdout/NewProfile; no profile copy); wired t1.return, t1.town.trigger and t1.restock behind ladder.enabled in addition to t1.repair and t1.tp.lost (5 points, all T1 town points; parity grids 4848, 12006, 280, 1080, 12006 states, none refused). t0.retreat.drink, t0.area.toohard, t0.unique.fight and t2.mercenary wait for Track 2b. See docs/t0-slice.md. Phase A3 (agent/t0slice3): wired t0.retreat.drink (combat abortFight), t0.unique.fight (combat preEngage), t0.area.toohard (bot towerWhy, tower guard) and t2.mercenary (bot askMerc, veto only) behind ladder.enabled; grids 19200, 9600, 1440, 54018 states, none refused; AC 1 met offline for all nine points (legacy answer untrained, ladder-off tests unchanged); AC 2 and 3 are NOT met: the drill gets an optional fight-time hook, only the fake lab implements it, the live lab does not. Not verified live."
live: "not run (phase B pending; a d2-live agent runs docs/probes/t0drill.md after the Gate 2 g2d runs end)"
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
- [x] #1 Hook points placed behind the store interface; with no rule file the behaviour equals today's defaults (regression test on the listed contexts)
- [ ] #2 T0 drill: fresh clvl 1, order 'Blood Moor', 10 virtual minutes, scripted answers across 3 known profiles and 2 seeded holdouts; invariants hold for every profile (no loop, no crash, no stall); a thought opened mid-fight does not pause the hero
- [ ] #3 Run record carries the rule-file hash, rules learned, answers given, thoughts faded, and the line 'writes after start: 0'
- [x] #4 Two builds: the default Sorceress and one other class or build; class-specific data in profile tables
- [x] #5 go vet and go test -p 1 of the packages; leakcheck 0; docs/t0-slice.md; say verified live or not
<!-- AC:END -->


Notes 2026-11-05 (integ): `d2bot -ladder [-ladder-profile NAME|holdout:SEED]` calls `bot.EnableLadderWith` after the Runner is built (default off, nothing changes without the flag); store under the instance state dir; drill command for the real bot in docs/probes/t0drill.md (Phase B, integrated path). Offline tested only; AC 2 and 3 stay open until the live run.
