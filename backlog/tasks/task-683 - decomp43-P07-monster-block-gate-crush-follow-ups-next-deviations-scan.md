---
id: TASK-683
title: 'decomp43: P07 monster block gate, crush follow-ups, next deviations scan'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-20T03:12:00Z'
labels:
  - d2d
  - research
dependencies: []
priority: medium
ordinal: 246000
card: "decomp43"
owns:
  - "docs/research/decomp43-monster-block-trace.md"
  - "docs/upstream/bodies/monster-block-body.md"
  - "tools/emu/mb_explore.py"
  - "tools/emu/mb_gate.py"
  - "tools/emu/mb_act2.py"
  - "tools/emu/mb_col.py"
  - "tools/emu/mb_block.py"
  - "tools/emu/mb_mut.py"
  - "tools/emu/mb_mut2.py"
  - "tools/emu/mb_edge.sh"
exclusive:
  - "fork-branch:fid/monster-block-r1"
contracts: []
result: "P07 monster block gate built as local private-fork branch fid/monster-block-r1 (oracle e5425263, claim e0236ed1, README row 91e3d4c5, on upstream 8ae5cfcb, pushed to the private remote only): a monster blocks only with MonStats NoShldBlock or a shield (item type shie) in its layer-7 look entry, ToBlock at most 75; the per-spawn shield is derivable (d2d already rolls the look into npc.comp[7]); parent 228 of 2,469 cases differ, claim 0; item-code rule 659 codes 0 differ; cap V run; 4 of 15 mutants run (all abort), the rest scripted but blocked by six stopped compiler processes of my cancelled job left on the builder. Crush follow-ups: missile already in r3 of #637, merc and pet not built (no early reach). Scan: three open items ranked, none provable yet. Offline, no live game; whole-tree ctest not run; verifier pending. Remaining: AC 3 mutants and pins of the claim."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Consumer: the independent verifier of fid/monster-block-r1 (the coordinator schedules it).

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 trace doc: P07 monster block rule with addresses and tags, d2d and bot compared
- [x] #2 oracle differential of the block chance against game.exe (V run), parent mismatches, claim 0
- [ ] #3 mutants and pins of the claim
- [x] #4 crush follow-up (merc, pet, missile) decided
- [x] #5 scan of the next player-visible deviations
<!-- AC:END -->
