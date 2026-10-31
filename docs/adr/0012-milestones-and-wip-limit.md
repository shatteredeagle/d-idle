---
status: accepted
date: 2026-10-31
decision-makers: user
---
# 0012: Work is organised by milestones M1 to M6 with a WIP limit of 3

## Context and Problem Statement

466 merges landed in three days across seven streams, the gated XP/h number did not converge, the same bug was fixed several times, and there was measurement debt in the gate path (docs/roadmap.md section 1). The user decided the shape of the fix on 2026-10-31 (roadmap decisions D1 to D5). The roadmap proposed this record as "0011"; that number went to grill 0003, so it is 0012.

## Considered Options

* A: milestones M1 to M6 with measurable exit criteria and a WIP limit of 3 implementation agents (docs/roadmap.md sections 3 to 5).
* B: keep the streams, add a weekly review of the numbers.
* C: freeze feature work until the gate number converges.

## Decision Outcome

Chosen: A.

* Milestones M1 basics, M2 early gate, M3 Act 1 clear, M4 Act 2, M5 ship, M6 idle layer; each has a milestone exit criterion owned by a spec and the closing card.
* At most 3 implementation agents at once (2 on the current milestone's critical path, 1 side lane); read-only agents are unlimited.
* Every job is a milestone card or a bug fix whose proposal cites a finding and whose gate is a scenario test of the end state.
* Queued cards use the status `deferred` (no OWNERS line, no exclusive claim).

## Consequences

* d2dmesh and stream-grid are parked until M3 (deferred, OWNERS lines removed); the livestream keeps running as is.
* "Needs a G1/G2 check" notes are dropped: Game.exe is frozen, verify on d2d or tag U.
* d2d-container-soak is deferred until the licence decision (ADR 0010) so it holds no slot.
* D3 buy order (potions first up to 3, then a Town Portal scroll only with gold >= scroll plus 2 potions) and the D4 gems rule (sell chipped and flawed before the Cube, keep better gems and every rune) are carried by cards m1-audittests and m1-loot.
* D5: a late baseline from a ledgered cheat-levelled clvl 12 throwaway, for XP/h and kills/h only (card m2-latebase).
* Readiness for dangerous zones follows [ADR 0011](0011-readiness-power-vs-danger.md) (power against danger, interim clvl floors).
* Progress is counted in exit numbers, not merges; parked streams hold no slots.

## Sources

docs/roadmap.md sections 1, 3 to 5 and 8; docs/process/guarded-proposals.md
