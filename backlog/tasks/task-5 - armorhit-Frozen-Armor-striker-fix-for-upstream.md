---
id: TASK-5
title: "armorhit: Frozen Armor striker fix for upstream"
status: In Progress
assignee: []
created_date: '2026-11-04T00:00:00Z'
updated_date: '2026-11-04T00:00:00Z'
labels:
  - "G-E"
  - "upstream"
dependencies: []
priority: medium
ordinal: 5000
card: "armorhit"
owns:
  - "docs/research/pr-armorhit-proof.md"
  - "docs/research/pr-armorhit-draft.md"
  - "docs/research/pr-bundle-queue.md"
  - "tools/emu/armor_hit.py"
  - "tools/emu/armor_hit_mode.inc"
exclusive:
  - "fork-branch:fix/armor-hit"
branch: "agent/armorhit"
agent: "armorhit"
result: "Offline, nothing verified live, no d2d slot, no game window. Local upstream candidate fix/armor-hit on the worker host (claim 36cbe015, tooling d5bd19c1, base upstream main 72132b99): Frozen Armor's striker effect follows game.exe FUN_0057b230 (resisted by cold res less pierce, specials chilled only, plain monsters frozen at coldeffect < 0 for length / MonsterFreezeDivisor). Differential (d2d test_armor_hit through the real Fight::buff_events vs game.exe FUN_005cac50 natively under the emulator): 95,920 -> 0 of 108,506 cases (every damaging case differed before), seed 2 95,986 -> 0. pr_guard 0/0 per commit, presubmit --quick 0 FAIL / 3 WARN answered, upstream tests green, gcc shadow / clang-tidy clean on touched lines, merges clean with #155 and #156 both ways. NOT READY: independent verifier, det hashes."
live: ""
merge_hash: ""
overlap: ""
source: "openspec-final:openspec/changes/armorhit"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The user asked (2026-11-01) whether Frozen Armor behaves on d2d as in game.exe. SKARMOR (docs/research/sk-armor-trace.md D1) proved the game side (FUN_005cac50 follows the freeze rule) and read d2d's `buff_events` as stunning and chilling every damaging striker, but no d2d-side run existed, so no PR candidate could be proposed.

What changes:
- On the worker host (not in this repo): local upstream candidate `fix/armor-hit` (base upstream main 72132b99): a d2d test mode `test_armor_hit --armor-hit` that drives the real `Fight::buff_events`, the game-side oracle `tools/emu/armor_hit.py` (FUN_005cac50 natively), and a claim commit that mak...
- In this repo: docs/research/pr-armorhit-proof.md (evidence pack, the answer to the user's question, the bot implication, a fork proposal), pr-armorhit-draft.md (PR body, limits first), a queue line in pr-bundle-queue.md, copies of the tooling (tools/emu/armor_hit.py, tools/emu/armor_hit_mode.inc)...
- Nothing pushed to the public repo or posted upstream; the private fork is not touched (proposal only).

Next step: PR claim (one batch at a time)

Gates (from the card):
- leakcheck -mode fail 0
- PR claim only with a differential on real builds: before > 0, after 0

Source: git tag openspec-final, openspec/changes/armorhit/ (this task is the copy).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 1.1 Worktree fix/armor-hit from current upstream main on the worker host
- [x] #2 1.2 d2d test mode test_armor_hit (real buff_events) + game oracle armor_hit.py (FUN_005cac50 natively)
- [x] #3 1.3 Before numbers on upstream main (95,920 of 108,506)
- [x] #4 1.4 Claim commit; after 0 of 108,506; second seed 95,986 -> 0
- [x] #5 1.5 CI-class checks; merge check with #155 / #156
- [x] #6 1.6 Proof, draft, queue line, card, OWNERS
- [ ] #7 1.7 Independent verifier and det state hashes (another card)
<!-- AC:END -->
