---
id: TASK-11
title: "fixprogress: trip-loop progress fix"
status: In Progress
assignee: []
created_date: '2026-11-04T00:00:00Z'
updated_date: '2026-11-04T00:00:00Z'
labels:
  - "G-C"
dependencies: []
priority: low
ordinal: 11000
card: "fixprogress"
owns:
  - "docs/fixprogress.md"
  - "internal/town/chores.go"
  - "internal/town/town.go"
  - "internal/town/fixprogress_test.go"
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
Gate 1 rerun (docs/soaks/gate1-rerun.md on agent/gate1) failed the progress criterion: 43 virtual minutes with no kill and no XP (limit 10). The log shows a trip loop, not a stuck walk: walk to town, shop buys nothing with 600 gold, walk back, field check asks for a scroll again.

What changes:
- internal/town: the game's word "no portal source (empty)" (ToTown finds no usable scroll or charged tome) is remembered (`NotePortalEmpty`) and outranks the tome stat read in the next visit, so the shop restocks against it. Two such visits in a row silence the "no town portal scroll" trip for Fut...
- Tests: internal/town/fixprogress_test.go (sim world).
- docs/fixprogress.md: timeline, evidence, cause tags, live recipe.

Next step: live proof in a seed-set record

Gates (from the card):
- go test -p 1 ./internal/town ./internal/bot passes; changecheck -cli 0 warnings; leakcheck -mode fail 0; tools/ci/check.sh; openspec validate --all --strict

Source: git tag openspec-final, openspec/changes/fixprogress/ (this task is the copy).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 timeline and cause from the minute log and the bot log
- [x] #2 town: portal-empty fact and watchdog, tests
- [ ] #3 live proof (later card)
<!-- AC:END -->
