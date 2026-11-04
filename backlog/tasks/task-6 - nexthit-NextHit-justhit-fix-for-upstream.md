---
id: TASK-6
title: "nexthit: NextHit justhit fix for upstream"
status: In Progress
assignee: []
created_date: '2026-11-04T00:00:00Z'
updated_date: '2026-11-04T00:00:00Z'
labels:
  - "G-E"
  - "upstream"
dependencies: []
priority: medium
ordinal: 6000
card: "nexthit"
owns:
  - "docs/research/pr-nexthit-proof.md"
  - "docs/research/pr-nexthit-draft.md"
  - "tools/emu/next_hit.py"
  - "tools/emu/next_hit_mode.inc"
  - "tools/emu/next_hit_summary.py"
exclusive:
  - "fork-branch:fix/nexthit"
branch: "agent/nexthit"
agent: "nexthit"
result: "Offline, nothing verified live, no d2d slot used. Local upstream candidate fix/nexthit on the worker host (claim 8abab0ed, tooling 7901fda2, base upstream main 72132b99): NextHit hit sets unit justhit for NextDelay frames, LastCollide last unit, one unit a frame. Differential (d2d test_next_hit real Fight::fire / Fight::world contacts replayed through game.exe FUN_005adf10 / FUN_005adb00 under the emulator): Nova / Frost Nova 84,779 -> 0 of 100,198 cases; d2d 4.2-4.4 hits per touched monster per cast vs game.exe 1.04. pr_guard 0/0 per commit, presubmit --quick 0 FAIL, four upstream test binaries green, shadow / clang-tidy clean. NOT READY: verifier, det hashes; Chain Lightning hop (C2) and Lightning per frame (C3) designed, not built. One docs line in pr-bundle-queue.md (shared file, owned by no open card)."
live: ""
merge_hash: ""
overlap: ""
source: "openspec-final:openspec/changes/nexthit"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Three traces (SKLIGHT D1/D2, SKNOVA D1, SKMISS) found the same d2d deviation, D-NEXTHIT: game.exe's NextHit rule puts state 0x56 `justhit` on a struck unit for NextDelay frames, so one nova / chain / pierce missile effect hits a unit once per window and one unit a frame; d2d tracks NextHit per missile. Effects were modelled only; no differential existed.

What changes:
- On the worker host (not in this repo): local upstream candidate `fix/nexthit` from upstream main, claim + tooling commits, with a stateful differential (d2d `test_next_hit --next-hit`, game.exe replay `tools/emu/next_hit.py`).
- In this repo: the evidence pack `docs/research/pr-nexthit-proof.md`, the PR body draft `docs/research/pr-nexthit-draft.md`, copies of the tooling (`tools/emu/next_hit.py`, `next_hit_mode.inc`, `next_hit_summary.py`), a queue line in `docs/research/pr-bundle-queue.md`, this card. Shared-file lines...
- Nothing pushed or posted.

Next step: PR claim

Gates (from the card):
- changecheck 0 warnings for this card; leakcheck -mode fail 0
- PR claim only with a differential on real builds: before > 0, after 0

Source: git tag openspec-final, openspec/changes/nexthit/ (this task is the copy).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 1.1 Worktree fix/nexthit from current upstream main on the worker host
- [x] #2 1.2 Stateful differential: d2d test_next_hit mode + game.exe replay oracle
- [x] #3 1.3 Before numbers on upstream main (84,779 of 100,198 cases differ)
- [x] #4 1.4 Claim commit (justhit, LastCollide, one unit a frame); after 0 of 100,198
- [x] #5 1.5 CI-class checks (pr_guard, presubmit, tests, shadow, clang-tidy, ruff)
- [ ] #6 1.6 Chain Lightning hop search (C2) and Lightning per frame (C3): designs only
- [x] #7 1.7 Proof, draft, queue line, card; bot-measurement note
- [ ] #8 1.8 Independent verifier and det state hashes (later, another agent)
<!-- AC:END -->
