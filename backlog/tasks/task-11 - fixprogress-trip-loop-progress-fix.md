---
id: TASK-11
title: 'fixprogress: trip-loop progress fix'
status: Done
assignee: []
created_date: '2026-11-04T00:00:00Z'
updated_date: '2026-11-05T18:01:00Z'
labels:
  - bot
  - bug
  - gate
milestone: m-0
dependencies: []
priority: low
ordinal: 11000
card: "fixprogress"
owns:
  - "docs/fixprogress.md"
  - "internal/town/chores.go"
  - "internal/town/town.go"
  - "internal/town/fixprogress_test.go"
  - "internal/town/trip.go"
  - "internal/town/staleportal_test.go"
  - "internal/env/tuning_town.go"
  - "internal/bot/loop.go"
  - "internal/bot/guarddrink_test.go"
  - "internal/bot/fakes_test.go"
  - "internal/nav/stuckfight.go"
  - "internal/nav/stuckfight_test.go"
  - "internal/nav/walk.go"
  - "internal/nav/nav.go"
  - "internal/env/tuning_nav.go"
  - "docs/knobs.md"
  - "internal/bot/runner.go"
  - "internal/bot/ladder3_test.go"
exclusive: []
branch: "agent/fixprogress"
agent: "fixprogress"
result: "Cause from the Gate 1 rerun log (V loop, U accounting root); fix and 3 sim tests pass on the worker host; NOT verified live, recipe in docs/fixprogress.md"
live: "not run; recipe in docs/fixprogress.md"
merge_hash: ""
overlap: "- m2-tripwp owns internal/town/trip.go: this card adds two lines there (set the flag on ErrNoPortal, clear it when a portal arrives). - goldfloor owns internal/town/shop.go: this card changes one case of scrollCount there (a tome counts 0 while the flag is set)."
source: "openspec-final:openspec/changes/fixprogress"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Gate 1 rerun (docs/soaks/gate1-rerun.md on agent/gate1) failed the progress criterion: 43 virtual minutes with no kill and no XP (limit 10). The log shows a trip loop, not a stuck walk: walk to town, shop buys nothing with 600 gold, walk back, field check asks for a scroll again.

What changes:
- internal/town: the game's word "no portal source (empty)" (ToTown finds no usable scroll or charged tome) is remembered (`NotePortalEmpty`) and outranks the tome stat read in the next visit, so the shop restocks against it. Two such visits in a row silence the "no town portal scroll" trip for Fut...
- Tests: internal/town/fixprogress_test.go (sim world).
- docs/fixprogress.md: timeline, evidence, cause tags, live recipe.

Stale town portal (Gate 2 g2b, seed 11, bot 9d03a58; V log, cause U): 43 `portal click failed` lines on one portal id, then a 7-area walk and 11 minutes without a kill. BackToField (internal/town/trip.go) now drops a portal id after town.portalClickFails (default 2) failed entries, rescans, then opens a fresh portal from the tome/scroll, then the waypoint, then the walk. Log lines: `back to field: portal id=X failed N times, dropped`, `back to field: fallback=<newportal|waypoint|walk> reason=...`. Tests: internal/town/staleportal_test.go (Sorceress + Barbarian). Not verified live.

Guard drink (Gate 2 g2c seed 13 death, 2026-11-05): guardOnce (internal/bot/loop.go) dropped SafetyDrinkLife and SafetyDrinkRejuv in state clearing, so the life fell 65 to 40 to 0 inside the nav stuck ladder with potions in the belt (combat.Decide already returns DrinkRejuv at or below chickenPct with a ready rejuv, Chicken with none usable). The guard now forwards both drinks in every field state (not in town); the drink goes through Fighter.Act, so the fighter's potion cooldown and the guard's 800 ms gap rate-limit it. Live proof line: `guard: drank <healing|rejuvenation> at life N% (state=<state>)`. Tests: internal/bot/guarddrink_test.go (Sorceress + Barbarian). No knob added. Not verified live.

Stuck ladder at low life (Gate 2 g2c seed 13 death, 2026-11-05): the nav stuck ladder (internal/nav/walk.go) only yielded to the fight when a pack stood within packRadius (8) and at most 4 times; beyond that it side-stepped on through a boss pack. New rule (internal/nav/stuckfight.go): rung 2 or more, hostiles inside combat.fightRadius, life below the new knob nav.stuckFightLifePct (default 60, 0 = off) ends the walk step and calls the Yield (fight) hook, up to 4 times per walk, then the town portal (Escape) once. Also logs a return to the level a walk just left within 2 s with its reason. Live proof lines: `nav: stuck ladder -> fight|portal rung=N life=P hostiles=M`, `nav: area border crossed back area=A from=B reason=<method|replan-why|walk>`. Tests: internal/nav/stuckfight_test.go (Sorceress + Barbarian life pools). Not verified live. The Yield clear radius stays PackYield's 12 subtiles (fight radius 22 only selects when to hand over).

Failsafe ladder rung 3 cap (Gate 2 g2c seed 13, 2026-11-05): with an item stuck on the cursor the ladder (escalate, internal/bot/loop.go, case 3) left and re-entered the game about 1,100 times in 2 h 07 min, because only rung 4 (MaxFailsafeRestarts, restartCapHit) was bounded. Rung 3 now counts its cycles since the last real progress (same test as MaxFailsafeRestarts: realResetMin XP/kill/level events between ladder entries); at bot.failsafeRung3Max (default 6, 0 = unbounded) it is skipped and the ladder goes to the pause rung, whose pause doubles from 5 min per consecutive pause without progress up to bot.failsafePauseMaxSec (default 3600). Rungs 1, 2, 4 and restartCapHit unchanged. Live proof lines: `failsafe: rung 3 cap reached cycles=N window=<d> -> pause`, `failsafe: paused backoff=<sec>`. Tests: internal/bot/ladder3_test.go (Sorceress + Barbarian). Not verified live.

Next step: live proof in a seed-set record

Gates (from the card):
- go test -p 1 ./internal/town ./internal/bot passes; leakcheck -mode fail 0; tools/ci/check.sh

Source: git tag openspec-final, openspec/changes/fixprogress/ (this task is the copy).

Done when: timeline and cause from the minute log and the bot log; town: portal-empty fact and watchdog, tests; stale portal: drop id after N failed clicks, fresh portal, waypoint, walk fallback (offline tests) (and 1 more criteria below)
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 timeline and cause from the minute log and the bot log
- [x] #2 town: portal-empty fact and watchdog, tests
- [x] #4 stale portal: drop id after N failed clicks, fresh portal, waypoint, walk fallback (offline tests)
- [x] #5 failsafe rung 3 capped per progress window, growing pause backoff (offline tests)
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Closed 2026-11-05 (status reconciliation, backlogfinish). Evidence: Merge agent/fixprogress on main, 3 sim tests pass. merged, offline tested; live proof pending: Gate 1 rerun, recipe in docs/fixprogress.md (no town trip loop, progress window advances). Removed criterion 'live proof (later card)'.
<!-- SECTION:NOTES:END -->
