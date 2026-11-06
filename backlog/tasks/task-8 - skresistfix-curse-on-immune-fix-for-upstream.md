---
id: TASK-8
title: 'skresistfix: curse-on-immune fix for upstream'
status: Done
assignee: []
created_date: '2026-11-04T00:00:00Z'
updated_date: '2026-11-05T18:00:00Z'
labels:
  - upstream
dependencies: []
priority: medium
ordinal: 8000
card: "skresistfix"
owns:
  - "docs/research/pr-curseimmune-draft.md"
  - "docs/research/pr-curseimmune-proof.md"
exclusive:
  - "fork-branch:fix/curse-immune"
branch: "agent/skresistfix"
agent: "skresistfix"
result: "Offline, nothing verified live, no d2d slot, no game window. Local upstream candidate fix/curse-immune on the worker host (claim 6a04cb3e, tooling 557d14e3, base upstream main 72132b99): a curse or aura lowering a resistance counts a fifth (toward 0) when the monster's base resistance is 100 or more (game.exe FUN_005c3540). Differential (real Fight::target_of and rules::missile_blow vs game.exe natively): adj 88,101 -> 0 of 147,628 (seed 7: 88,224 -> 0), dealt 14,355 -> 1,958 (seed 7: 14,289 -> 1,977); the 1,958 left are all D2 (magic total under -100, not in the claim, unreachable in 1.14d data); a line-for-line port of the fixed rule equals the real build on every case. pr_guard 0/0, presubmit --quick 0 FAIL, five test binaries rc 0, gcc shadow and clang-tidy clean on touched lines, ruff clean, merges clean with #152 #153 #155 #156 #157 #158, fix/armor-hit, fix/freeze-resume, fix/frozen-armor-and-resume-2c. Independent verifier (VERIFYCURSEIMMUNE) READY, three draft text edits applied; still owed: det state hashes (no d2d slot) and the user's go."
live: ""
merge_hash: ""
overlap: ""
source: "openspec-final:openspec/changes/skresistfix"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
SKRESIST (docs/research/sk-resist-trace.md, branch agent/skresist) proved deviation D1 on the real d2d build: a curse or enemy aura that lowers a monster's resistance counts its full value on an immune monster, while game.exe divides a negative change by 5 when the monster's base resistance is 100 or more (FUN_005c3540). Hell and Nightmare immunities break on d2d that game.exe keeps. The trace stopped at "harness ready, no fix"; this card builds the local upstream candidate.

What changes:
- On the worker host (not in this repo): local branch `fix/curse-immune` in a new worktree `skresistfix` of the PR checkout `skresist` (base upstream main 72132b99): one claim commit (`Fight::apply_target_stats` takes the monster's base resistances and counts a negative change as a fifth when the b...
- In this repo: `docs/research/pr-curseimmune-draft.md` (PR body draft), `docs/research/pr-curseimmune-proof.md` (commands, counts, checks), this card, OWNERS lines.
- Nothing is pushed or posted; the private fork is not touched (a later card carries it after an independent verifier).

Next step: PR claim

Gates (from the card):
- leakcheck -mode fail 0
- PR claim only with a differential on real builds: before > 0, after 0

Source: git tag openspec-final, openspec/changes/skresistfix/ (this task is the copy).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 1.1 Worktree skresistfix, branch fix/curse-immune from upstream main 72132b99 (worker host)
- [x] #2 1.2 Claim commit; test mode test_sk_resist (real target_of) and oracle sk_resist.py as the tooling commit
- [x] #3 1.3 Differential before and after on two seeds (147,628 cases each)
- [x] #4 1.4 CI-class checks and merge check with the open PR branches and local candidates
- [x] #5 1.5 Draft, proof, card, OWNERS
- [x] #6 1.6a Independent verifier (VERIFYCURSEIMMUNE): READY
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Closed 2026-11-05 (status reconciliation, backlogfinish). Evidence: Merge agent/skresistfix on main; upstream PR #160 (curse / aura lowering counts a fifth on base resistance >= 100) MERGED. Removed 1.6 (det state hashes, user's go): the upstream merge supersedes it.
<!-- SECTION:NOTES:END -->
