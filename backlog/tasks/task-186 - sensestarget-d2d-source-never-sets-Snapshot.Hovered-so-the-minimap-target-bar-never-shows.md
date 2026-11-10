---
id: TASK-186
title: "sensestarget: d2d source never sets Snapshot.Hovered, so the minimap target bar never shows"
status: Done
assignee:
  - pro-runner
created_date: '2026-11-08T14:50:00Z'
updated_date: '2026-11-10T16:54:00Z'
labels:
  - bot
  - bug
milestone: m-4
dependencies: []
priority: medium
ordinal: 154000
card: "sensestarget"
owns:
  - "internal/d2dsrc"
  - "cmd/d2bot/sensesunits.go"
  - "cmd/d2bot/sensesfeed.go"
  - "internal/combat/fight.go"
  - "internal/combat/fighter.go"
exclusive: []
contracts: []
result: "Offline only. unitSource.Attacking (fed by combat.Fighter.Attacking) flags the attacked monster; d2d has per-monster HP (cur/max) so lifePct needs no fake. Not verified live: no live lockstep poll showing lifePct falling."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Live check of 2026-11-08 (docs/probes/panelslive.md, docs/panels-live.md): 169 unit samples over two lockstep runs, tier 3 open, Codex held Corpsefire and Fallen Shaman and the hero fought Fallen all the time, yet no unit ever carried target, name or lifePct. Cause (S): internal/d2dsrc/source.go builds d2.Snapshot without Hovered (grep finds no Hovered there), and cmd/d2bot/sensesunits.go takes Target from Snapshot.Hovered. Done when: the target comes from the bot's attack target or the d2d source fills Hovered.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 On the d2d backend the monster the bot attacks is flagged Target in /api/senses (name and lifePct for a Codex-met class)
- [ ] #2 Test with two classes (Sorceress, Paladin) and a live lockstep poll shows lifePct falling during a fight
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Closed in the 2026-11-10 true-up (docs/research/project-review-2026-11-10.md section 3.1): merged, offline acceptance proven; open live checks moved to docs/live-checks.md (TASK-253).
<!-- SECTION:NOTES:END -->
