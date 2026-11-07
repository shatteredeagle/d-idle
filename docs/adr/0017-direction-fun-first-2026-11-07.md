---
status: accepted
date: 2026-11-07
decision-makers: user (popup answers to the five decisions of docs/research/direction-review.md section 9)
---
# 0017: Direction for the next two weeks: find the fun first

Amends [ADR 0015](0015-goals-acceptance-on-game-exe.md) and [ADR 0016](0016-automation-ladder.md) on the ranking of work and on when the Gate 2 waves run. Their product and acceptance text stands.

## Context and Problem Statement

The direction review ([research/direction-review.md](../research/direction-review.md), 2026-11-07) found five things:
1. Parity with game.exe is the base of the product, but the gate waves measure T4 behaviour the ladder demoted, at 24 host-hours a wave.
2. The player never has been shown the loop: no realtime play-session record exists, so "is it fun" has no number (H1 to H4).
3. The ask mechanism is untested: in the clean drills the only question that fired was the Forgotten Tower one, from route planning.
4. The process costs more than it returns in places: a metacurrency sink before H1, generated docs committed, many docs-only merges.
5. The idle-versus-attention question is open, and distribution brings licence questions.

## Decision

All five decisions are the user's popup answers of 2026-11-07.
- **D1 yes:** for two weeks, optimise for finding the fun (design iteration is the main track). Parity keeps the cheap probe slices 1 and 2 and the fix pilot; Gate 2 waves, Gate 3, Gate A runs and fidelity slices 3 to 6 pause. A 45-minute smoke per bot change is the regression floor (review section 6).
- **D2 idle:** d.idle is an idle game. Questions queue for the player's return with sensible defaults (the hero acts on its default); the 45 s fade goes as a design direction (implementation is a later card).
- **D3 no (the user chose against the review's recommendation):** the tester container work continues in parallel at normal priority; the LAN session path from the review also happens.
- **D4 yes:** the metacurrency slice pauses (merged ledger and rows stay); no further metacurrency slices until H1 shows that questions are worth answering.
- **D5 yes:** diet the board and docs (stop committing `docs/cards.md`, archive finished research docs with stubs, no new contracts, fewer docs-only merges). Implementation is a separate later job, carded.

## Consequences

- Paused (Deferred with label `parked`): fidelity cards of m-2 except TASK-55 and TASK-56; metacurrency cards TASK-60, 61, 63, 64; orders slices 2 and 3 (TASK-34, 47); unwired ladder points (TASK-35, 37, 38). Gate A (TASK-97) and any Gate 3 run wait for the design brief v1.
- Kept: the lab binary, lockstep and drills; probe slices 1 and 2; the watch screen, question cards, orders slice 1, the journal; the decision-point store and its 9 wired points; the merge queue and checks; the tester container and onboarding cards.
- New milestone m-4 "Find the fun" carries the sessions, the question-density variant, the return screen, the smoke, the LAN how-to, the design brief v1, the queue-for-return card and the three diet cards (review section 7).
- The design brief v1 (plan step 8) replaces the ranking paragraph of the Current truth through the user's grill.
- Legal and provenance risks of the review (section 8) are not decided here.
