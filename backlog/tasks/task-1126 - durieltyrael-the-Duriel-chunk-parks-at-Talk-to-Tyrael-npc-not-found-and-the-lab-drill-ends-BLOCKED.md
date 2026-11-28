---
id: TASK-1126
title: 'durieltyrael: the Duriel chunk parks at Talk to Tyrael (npc not found) and the lab drill ends BLOCKED'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-28T06:59:00Z'
labels:
  - bot
  - bug
dependencies:
  - TASK-1107
priority: medium
ordinal: 534000
card: "durieltyrael"
lane: next
owns:
  - "internal/questboard/"
  - "internal/task/"
  - "internal/bot/stepexec.go"
  - "internal/bot/stepexec_act2chunks_test.go"
exclusive: []
contracts: []
result: "Root cause found offline (V: the fork's drlg-dump and Objects.txt on the game data, level 73; bot logs of 10 runs): Tyrael's preset is at subtile 78,88; the arena's only way to him is the 1x5 opening at 110,207 closed by object 153 (tyrael's door: collision in modes 0 and 1, not selectable, no operate function), which game.exe opens at Duriel's death and the d2d fork never does (its own note: Tyrael's prison is not opened), so the hero's walk ends in the arena (12 coverage points, x 136 to 150, y 189 to 216 in every run) and his room never comes up. The fix at the cause is in the fork (open object 153 at Duriel's death), outside this card's owns. Bot side, all builds alike (no build-specific data): the talk row's Approach now fails with npc_out_of_reach (not found) plus a diagnostic line instead of sending the town walker, and the board counts that text once per minute, so the park reason names the cause. Tests for Sorceress and Paladin, shown to fail without the change. Box 2 is a synthetic lair state, not a recorded one. Not verified live or on the lab; box 3 needs the fork fix."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

drill.act2.duriel.stepexec Paladin 11, 3 of 3 BLOCKED on the lab (bot main 702b6cfd5, run dir provingground-out/20261128-015020 of the labops2 lab checkout, rep1 to rep3 identical): the chunk seven_tombs/duriel now starts and runs (first night: never started): place_staff needs three passes ('object class 152 not found (the whole level was walked)' once), 'Entered Duriel's Lair', 'quest.duriel/door ran but is still needed (evidence 2)', 'boss not found but dead: the kill step is done', 'Quest progress: duriel', then 11 x 'step talk_tyrael: 2 tries used: open Tyrael: town: npc not found or not reachable', 'parked seven_tombs/duriel ... gave up after 3 tries', then 'Waiting for a fresh game', 'campaign stalled: fresh game failed' and exit after 60 virtual min (the harness cannot give a fresh game, so the row reads BLOCKED). Rerun on main f9b1c3551 (nf3b, a talk row walks the level for a field NPC; run dir 20261128-025235): the walk runs (6 x 'step executor: npc out of sight, walking the level for him' class 251) and Tyrael is still not found, same park. U: whether the setup's Duriel (already dead when the hero arrives) leaves Tyrael unspawned in the fork, or the step looks for him on the town NPC road. TASK-1107 names the Duriel stall but its result says it is not addressed. Blocks the fourth scenario of the quests.stepExec ledger row (the other three are 3 of 3 GREEN, docs/probes/lab-night-results-2026-11-28b.md). CHEATED setup, mechanics only, not live-verified. Done when the criteria hold.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 The talk_tyrael step finds Tyrael in Duriel's Lair (not through the town NPC road) or reports a coded reason the board can show
- [ ] #2 Unit test for the step with a recorded Duriel's Lair state, two classes
- [ ] #3 drill.act2.duriel.stepexec GREEN 2 of 3 on the lab (Paladin 11), then the quests.stepExec ledger row is re-read
<!-- AC:END -->
