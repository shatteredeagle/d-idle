---
id: TASK-2
title: "m3-andariel: Andariel fight card with the SME decisions (docs/andarielsme.md)"
status: Deferred
assignee: []
created_date: '2026-11-04T00:00:00Z'
updated_date: '2026-11-04T00:00:00Z'
labels:
  - "G-B"
  - "soak"
dependencies: []
priority: high
ordinal: 2000
card: "m3-andariel"
owns:
  - "internal/boss/act1.go"
  - "internal/boss/andariel_d2d_test.go"
  - "docs/andariel.md"
  - "docs/probes/andariel/**"
exclusive:
  - "soak-slot:3"
branch: "agent/m3-andariel"
agent: "m3-andariel"
result: "research note only (agent andychain, no fight run): docs/research/andariel-chain-needs.md; effective Andariel floor is 20 without a merc (route guard), SME says 18 with merc and 21 without"
live: ""
merge_hash: ""
overlap: ""
source: "openspec-final:openspec/changes/m3-andariel"
contracts: ["proposed/m3-andariel"]
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Andariel has never been fought on either backend (docs/plan/STATUS.md), and the d2-sme calls her the biggest unknown of the Act 1 clear: a low-level Sorceress dies to poison nova and cold bolts at close range, to an empty belt, or to a dead merc that turns her onto the caster. The fight does not depend on the economy or walk work, so it runs in parallel with M1 and M2 on a cheat-levelled throwaway (ledgered) at the floor level.

What changes:
- internal/boss/act1.go: Andariel script on d2d: approach from Catacombs 4, antidote before the fight and on poison, Charged Bolt at range (Static Field only if a measured drill shows it helps; the pinned no-opener script stays by default), keep range, retreat line, kill detect, log `boss andariel`...
- Fights on a cheat-levelled throwaway at the fight floors (15 with a living merc, 17 without, plus a life floor: life floor = N from the diagnostic run (step 2/4)) with the kit (8 healing, 3 antidotes, 4 mana, tome with at least 2 charges). The declared setup may be goto Catacombs 4 plus a given m...
- docs/andariel.md: the fight log, what killed her or us, Static Field's boss cap in Normal (U until read from the tables first, step 2), her HP and resists from the tables.
- Delta: act1-campaign ADDED `Andariel fight rules`.

Next step: first drill per the SME card

Gates (from the card):
- series at the fight floors (15 with a living merc, 17 without, life floor = N from the diagnostic run (step 2/4)) with the kit (8 healing, 3 antidotes, 4 mana, tome with at least 2 charges): 5 clean wins in a row (or 7 of 8 with a cause for the loss), Andariel dead, character alive, antidote drunk on poison, writes after start: 0, `boss andariel` log lines with life minimum and potions used (raw in docs/probes/andariel); includes handicapped variants (merc at 50 percent life, kit at the gate minimum, tome at 2 charges); retreat by portal at 30 percent (escape) or 12 percent (chicken), one r...
- the no-merc case at least 3 times, reported (pass or fail, with the cause); test that a dead merc flips the floor between 15 and 17 mid-run
- go test -p 1 ./internal/boss passes including TestAndarielFightOrder (antidote, then Charged Bolt; Static Field only if a measured drill shows it helps) and TestAndarielScriptHasNoStaticFieldOpener unchanged
- go run ./cmd/leakcheck -mode fail finds 0; cheat ledger has one entry per cheat with a reason; tools/agentreap.sh m3-andariel leaves nothing

Source: git tag openspec-final, openspec/changes/m3-andariel/ (this task is the copy).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 1.1 Read docs/bossgate.md, internal/boss/act1.go, docs/sme/play-act-route.md section 4; write the fight order test first
- [ ] #2 1.2 Prepare the throwaway (cmd/d2cheat: level, gold, kit; merc claimed honestly or hired per merc-policy: no paid Act 1 hire before Blood Raven) on a hidden slot
- [ ] #3 1.3 the series (with the merc, handicapped variants, at least 3 without a merc; setup may be goto Catacombs 4 plus a given merc and kit, declared); docs/andariel.md; changecheck, leakcheck, validate; reap
<!-- AC:END -->
