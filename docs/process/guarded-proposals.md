# Guarded-file proposals from the roadmap decisions (2026-10-31)

> status: APPLIED 2026-10-31 by card grillapply: the decisions index has the lines, the milestones/WIP ADR is [ADR 0012](../adr/0012-milestones-and-wip-limit.md) (number 0011 went to grill 0003), the glossary terms `milestone exit criterion`, `deferred card`, `paired A/B` are in GLOSSARY.md (`late baseline` was not added). Original text follows: for the user to apply. Agents never edit `docs/decisions.md`, `docs/adr/*` or `GLOSSARY.md` (AGENTS.md guard rule). Decision date for every entry: 2026-10-31 (the user, via pop-ups). Note: the ROADMAP agent's report with its own proposed ADR text is not in the repo; the text below is drafted from docs/roadmap.md section 8 and is a proposal, not a copy.

## docs/decisions.md (one-line index entries)

- 2026-10-31 D1: Milestones M1 to M6 and a WIP limit of 3 implementation agents (read-only agents unlimited) govern the work; every job maps to a milestone card or is a bug fix with a finding (docs/roadmap.md).
- 2026-10-31 D2: d2dmesh and stream-grid are parked until M3 is done; the livestream keeps running as is.
- 2026-10-31 D3: Purchase order at gold 100 to 190 is potions first (up to 3), then a Town Portal scroll only when gold >= scroll price plus 2 potions; the merged `town.tpFirstLevel` 4 rule is changed to it (card m1-audittests).
- 2026-10-31 D4: Before the Horadric Cube recipe is available, chipped and flawed gems are sold, better gems and every rune are kept (card m1-loot).
- 2026-10-31 D5: The late baseline for XP/h and kills/h is built from a cheat-levelled clvl 12 throwaway with the M1 kit on hidden d2d, every cheat in the logged ledger; not used for deaths or gold (card m2-latebase).
- 2026-10-31 D6: Route guard numbers (SME 18/14 vs plan 20), the Andariel floor and the revive budget are settled in grill 0003, run by the user now (prebrief section 10).

## Proposed ADR: docs/adr/NNNN-milestone-roadmap.md

Title: Work is organised by milestones M1 to M6 with a WIP limit of 3.

Status: accepted, 2026-10-31.

Context: 466 merges in three days across seven streams, the gated XP/h number not converging, the same bug fixed several times, measurement debt in the gate path (docs/roadmap.md section 1).

Decision: adopt the milestone ladder of docs/roadmap.md section 3 (M1 basics, M2 early gate, M3 Act 1 clear, M4 Act 2, M5 ship, M6 idle layer) with measurable exit criteria owned by specs and closing cards. At most 3 implementation agents at once (2 on the current milestone's critical path, 1 side lane), read-only agents unlimited. Every job is a milestone card or a bug fix whose proposal cites a finding and whose gate is a scenario test of the end state. Queued cards use the status `deferred` (owns and exclusive intended, not claimed, no OWNERS line). d2dmesh and stream-grid are deferred until M3; the livestream is unaffected. A late baseline from a ledgered cheat-levelled clvl 12 character is allowed for XP/h and kills/h only.

Consequences: progress is counted in exit numbers, not merges; parked streams do not hold slots or OWNERS lines; the dispatcher brief proposed in docs/process/dispatcher-brief-proposed.md replaces the private brief when the user applies it.

## GLOSSARY.md

Proposed terms: `milestone` (M1 to M6, a goal with a measurable exit criterion), `deferred card` (a change card queued behind a named dependency or user decision), `late baseline` (the clvl 12 cheat-levelled XP/h and kills/h baseline, not valid for deaths or gold). Earlier glossary proposals: docs/research/proposed-glossary-terms.md.
