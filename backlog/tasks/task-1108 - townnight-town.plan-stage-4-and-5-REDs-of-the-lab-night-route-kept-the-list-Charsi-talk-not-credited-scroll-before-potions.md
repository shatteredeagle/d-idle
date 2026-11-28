---
id: TASK-1108
title: 'townnight: town.plan stage 4 and 5 REDs of the lab night (route kept the list, Charsi talk not credited, scroll before potions)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-28T05:19:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 532000
card: "townnight"
lane: next
owns:
  - "internal/town/"
  - "internal/supply/"
  - "tools/provingground/scenarios/townplan.json"
exclusive: []
contracts: []
result: "Not live-verified. one-visit: product gap fixed in internal/town/route.go (a chore whose NPC has no preset no longer drops the route to the fixed list; test TestReinsertChoresKeepsPrecedence); rerun to see the stops line. malus-errand-gold: not the Charsi talk (optional greeting, skipped), the chunk parks on the route guard (life 180 needed, max 55): scenario note records it, U setup life and which record-3 bit. gold-short-priorities: marked pending for card town7; the setup also left a belt of 4 healing and a scroll so the plan saw verdict=ready. short-gold-handover and sell-keeps-need were fixed in labtriage; rerun."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

one-visit RED both classes: the bot logged 'town plan: route kept the list (no preset position for cain, chore identify)' instead of the stops line (internal/town/route.go). malus-errand-gold RED: 'quest.tools_of_the_trade/talk-charsi/home ran but is still needed (evidence 2)' repeats, record 3 stays 0 (U fork or bot). gold-short-priorities RED both classes: with 228 gold the bot bought a Scroll of Town Portal (100) before healing potions, no 'verdict=short-gold' line, gold 325 at the end; the scenario note says the priority table is stage 7 (card town7), so this row is an expected RED until then. sell-keeps-need: the assertion was fixed in labtriage (potions were drunk, not sold); rerun before judging. belt-refill-from-pack BLOCKED 'no room' is fixed by fix6. Evidence: docs/probes/lab-night-results-2026-11-28.md (lab night 2026-11-28, bot 2f2e092c3, not live-verified beyond the lab run).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 town.plan.one-visit logs 'town plan: route stops=' (Cain gets a position or the route tolerates a chore without one)
- [ ] #2 town.plan.malus-errand-gold: the Charsi talk step sets quest record 3 or the cause is named
- [ ] #3 town.plan.gold-short-priorities: decision D3 holds (potions before the first scroll) or the scenario is marked as waiting for card town7
<!-- AC:END -->
