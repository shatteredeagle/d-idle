---
id: TASK-101
title: 'fid-load-life-mana: d2d keeps the saved life and mana at load (pilot)'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-07T21:49:00Z'
updated_date: '2026-11-07T22:55:00Z'
labels:
  - d2d
  - upstream-pr
  - d2d-data
  - fidelity
milestone: m-2
dependencies:
  - TASK-55
priority: high
ordinal: 99000
card: "fid-load-life-mana"
owns:
  - "docs/research/fid-loadhp-trace.md"
  - "docs/research/pr-load-life-mana-draft.md"
  - "tools/emu/fl_build.sh"
  - "tools/emu/fl_load.py"
  - "tools/emu/fo_build.sh"
  - "tools/emu/fo_diff.sh"
exclusive: []
contracts: []
result: "loader found: FUN_0056b180 -> FUN_0056af80 (0x56b0ff, 0x56b10d), no clamp, stamina = maximum; oracle tools/emu/fl_load.py (V run, 414 cases): parent 392 differ, pilot r1 341, new branch local/fid-load-life-mana-r2 0; the pilot fraction drop was the client display copy (FUN_00548760 sar 8, FUN_0045d9b0 shl 8). r2 in a scratch clone and a bundle on the worker host, push to the private remote refused by the sandbox. Not verified live; needs the independent Opus verification. UPDATE 2026-11-07: independent verification HOLDS / HOLDS WITH LIMITS (review-load-life-mana-opus.md); upstream PR opened: psi29a/d2decomped#242 (https://github.com/psi29a/d2decomped/pull/242), head 2cf72cb1 (8308864a before the CodeRabbit fix; r2 rebased onto upstream main 9c78f391, same commit), branch fix/load-life-mana; merge pending, not ticked. Not verified live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Matrix rows EN04 ST01. Evidence: docs/research/fidelity-probe-run1.md (12 of 12 loads: d2d heals life and mana to full at load, game.exe keeps the saved value with the fraction dropped). d2d site: World::take_character (components/game/world.cpp), a deliberate refill from upstream commit 756bf206. Done when the trace, the fork branch and the draft exist and an independent verifier has checked the differential. Nothing is posted upstream without the user's go.
## Upstream batching (2026-11-07)
Joins the bundle "character load and stats" (docs/research/pr-bundle-queue.md, entry fid-load-life-mana) together with TASK-102 (max mana) if its cause is a d2d defect in the same load path; nothing from another subsystem goes in. Candidate branch `local/fid-load-life-mana-r1` on the private remote, one commit on upstream main. Maintainer rules (AGENTS.md of upstream main, read 2026-11-07): one subsystem per batch, one change per commit with its test, oracle script in the same PR tree, the Claims table, `Agent: Claude Code` last line, numeric noreply author and committer, `python3 tools/presubmit.py --quick` and `tools/pr_guard.py`. Our budget: bundle <= 40 files (hard 70), <= 1000 added+deleted lines, <= 15 commits, one push then at most one per hour, at most 3 open PRs, independent Opus verification before opening. NOT READY: no game.exe address for the load rule (evidence is live reads only, "behaviour alone is not evidence"), no oracle in tools/emu, no independent verification.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Trace docs/research/fid-loadhp-trace.md with d2d code path, original behaviour, tags
- [x] #2 Private fork branch pilot/load-life-mana builds and test_game passes after the fix and fails before it
- [x] #3 Differential vs game.exe code (emulator) built or the gap written down
- [x] #4 Draft PR text docs/research/pr-load-life-mana-draft.md, DRAFT NOT POSTED
<!-- AC:END -->
