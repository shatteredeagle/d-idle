---
id: TASK-7
title: "skmiss: missile roll max fix for upstream"
status: In Progress
assignee: []
created_date: '2026-11-04T00:00:00Z'
updated_date: '2026-11-04T00:00:00Z'
labels:
  - "G-E"
  - "upstream"
dependencies: []
priority: medium
ordinal: 7000
card: "skmiss"
owns:
  - "docs/research/sk-missile-trace.md"
  - "tools/emu/sk_missile_hit.py"
  - "tools/emu/sk_missile_hit_test.cpp"
  - "docs/decomp-queue.md"
  - "docs/BACKLOG.md"
  - "docs/INDEX.md"
exclusive:
  - "fork-branch:fix/missile-roll-top"
branch: "agent/skmiss"
agent: "skmiss"
result: "Offline, nothing verified live. Spawn / mana / flight / hit traced (docs/research/sk-missile-trace.md): D1 missile roll includes the max on d2d, real builds 104,000 cases 31,459 -> 0 (seed 7: 64,000, 16,479 -> 0), local candidate fix/missile-roll-top on upstream 72132b99 (tooling 9ba8d369, claim 23e2baec), CI-class checks clean; D2 mana paid at the action frame in game.exe vs at the command on d2d (read; drill M1); bot B1-B4; verifier, det hash and the user go open."
live: ""
merge_hash: ""
overlap: ""
source: "openspec-final:openspec/changes/skmiss"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Before act progression goes further the user asked for a focused check of the Sorceress skills. SKILLINV's matrix (docs/research/sorc-skill-matrix.md) leaves the player missile pipeline UNTRACED apart from Fire Bolt's one live mana / damage proof. Decomp row 14: how game.exe 1.14d spawns, flies and lands the plan's missile skills (Fire Bolt, Charged Bolt, Ice Bolt, Frost Nova, Lightning, Chain Lightning), what d2d does, what the bot assumes.

What changes:
- Trace in game.exe (disasm + emulator) the spawn (skill start / do, mana payment, missile rows, counts, aim), the flight (reusing upstream's proven flight oracle) and the hit (the roll, resistance, life), with V/D/S/I/U tags.
- Differential oracle on real d2d builds for the hit (skill, level 1..20, synergies, mastery, resistance, pierce, life, seed); a structural comparison and a drill recipe for the stateful parts.
- In this repo: `docs/research/sk-missile-trace.md`, `tools/emu/sk_missile_hit.py` and `tools/emu/sk_missile_hit_test.cpp` (copies of the upstream-candidate oracle and test mode), decomp-queue row 14, BACKLOG and INDEX lines.
- If a deviation is proven on the real patched d2d build: a LOCAL upstream candidate branch on the worker host (nothing pushed or posted).

Next step: differential, then PR

Gates (from the card):
- differential on real d2d builds: before > 0, after 0 for any claim; pr_guard 0/0; presubmit --quick 0 FAIL; upstream tests rc 0
- leakcheck -mode fail 0

Source: git tag openspec-final, openspec/changes/skmiss/ (this task is the copy).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 1.1 Trace spawn, mana payment, flight and hit in game.exe 1.14d (tags)
- [x] #2 1.2 Read d2d's missile creation, flight and damage; the bot's casting model
- [x] #3 1.3 Differential oracle (game.exe natively vs the real d2d build), before and after
- [x] #4 1.4 Local upstream candidate (only if a deviation is proven on the real build)
- [x] #5 1.5 Docs: trace (deviations table, drill recipe), queue row 14, BACKLOG, INDEX
- [ ] #6 2.1 Independent verifier (a second agent)
- [ ] #7 2.2 Det state-hash run (needs a d2d slot)
<!-- AC:END -->
