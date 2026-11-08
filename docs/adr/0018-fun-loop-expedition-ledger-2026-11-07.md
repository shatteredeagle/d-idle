---
status: accepted
date: 2026-11-07
decision-makers: user (popup answers on docs/research/fun-ideas.md, 2026-11-07)
---
# 0018: d.idle's loop is the Expedition Ledger; Act 1 content is the Codex; carry-over is a Hall of Fame first

Amends [ADR 0016](0016-automation-ladder.md) items 10, 15, 16 and 17 (fade, worth unit, pace, altar) and builds on [ADR 0017](0017-direction-fun-first-2026-11-07.md), whose direction decisions D1 to D5 stand. The design and its sources are in [research/fun-ideas.md](../research/fun-ideas.md) (sections 3 to 5, 9, 11 to 13).

## Context and Problem Statement

ADR 0017 ranked work fun-first and asked for a design base. The direction review found no accrual, no spend and no return payoff in the ladder as built. This record fixes the loop the two-week prototype tests.

## Decision

1. **The loop.** Every town trip is an expedition with a report card (zone, length, cost, return, closest call, firsts, tally). Tally is credited in the existing ledger from observed events (kills, firsts, quests, waypoints, bosses, completed expeditions). No item first: the Raven's Mark item stays deferred.
2. **The gates.** The pacing gate is a price (charters, paid in tally only), never a wall; the hero warns. In-range trips are free and the hero's guard keeps the last word on arrival.
3. **Advancing.** The hero proposes the next zone. When the player is away it waits and keeps farming the current safe zone. No auto-advance, in range or timed. The player can also order from the map.
4. **Chores.** One policy sheet, set at hero creation, changeable any time, suboptimal choices allowed. Each row shows its effect on the sheet and as before/after cards on the return screen. Chores are not questions.
5. **Session shape.** Return screen first, live glance view second. Target: away for hours, then a 5 to 20 minute return.
6. **Content.** Act 1 firsts fire discovery questions and fill a Codex. Skills unlock by quest or find through a per-class unlock table (data in `internal/plandata`, read by the leveler's rule), never by currency; measured runs load a table with every unlock met. The player may spend ONE free-choice point on a Codex-unlocked skill the build table skips, shown with its consequence. The SME's skill clvl requirements are UNVERIFIED; they are checked against the game tables before the unlock table is written (card skillreqcheck).
7. **Legacy.** Retiring after Andariel is optional; a retired or dead hero enters a Hall of Fame. Hall and title first (ledger and UI only). The legacy charm item (SME tiers I to V) comes only after the overlay probe proves the charm row in d2d and the user approves game.exe. Permanent stat points, +skills, XP bonuses, level head starts, inherited uniques and carried gold are never built. Hardcore rebirth stays as ADR 0016 item 9.
8. **Dropped.** Level walls, gambling as a sink (Act 1), the loot filter as a standalone game (folded into the policy sheet), the faster answer fade as a sink, Paragon-style stats, the Blessings skill tree. The metacurrency stays paused as an ITEM; its tally ledger side is un-paused as part of the loop. The altar object stays parked.
9. **Unchanged.** ADR 0017 stands; the learned-brain experiment (TASK-126) is unchanged.

## Consequences

- Milestone m-4 gains the cards TASK-127 to 136 in the order of fun-ideas.md section 13. The two-week playable is returnscreen, policysheet, tallystrip, codexfirsts (TASK-127 to 130), then playsession2 (TASK-134); proposezone (TASK-131), skillreqcheck and skillunlocks (TASK-132, 133) and legacyhall (TASK-135) follow the session verdict; legacycharm (TASK-136) is parked.
- TASK-108 (qdensity) and TASK-115 (qqueue) are Deferred, folded into codexfirsts, returnscreen and proposezone; TASK-109 is the mock for returnscreen.
- The catalogue's chore points stop opening cards once the policy sheet covers them; the return screen becomes the `/watch` landing after an absence; the ledger registry gains the sources and prices of fun-ideas.md section 3.4. The metacurrency slice (TASK-60 to 64) stays paused except the ledger credits of tallystrip.
- Every pacing number in fun-ideas.md section 7 is a logged target, never a pass or fail, until a session record exists.
- Glossary candidates (not written; the user's grill session decides): Expedition, Report card, Charter, Policy sheet, Codex, Legacy charm, Hall of Fame.
