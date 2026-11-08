---
id: TASK-125
title: 'fid-revive-refill: Fight::revive refills only life; game.exe resurrect (0x54c17c..0x54c198) also refills mana and stamina'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-07T23:31:00Z'
labels:
  - d2d
  - upstream-pr
  - fidelity
  - d2d-combat
milestone: m-2
dependencies: []
priority: medium
ordinal: 112000
card: "fid-revive-refill"
owns:
  - "docs/research/fid-revive-refill.md"
  - "docs/research/pr-revive-refill-draft.md"
  - "tools/emu/fr_*"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

d2d Fight::revive heals life only. The independent verifier of TASK-101 (docs/research/review-load-life-mana-opus.md, finding F6, V read) found that game.exe's resurrect path (0x54c17c..0x54c198) also refills mana and stamina. Out of scope of upstream PR #242 (load life/mana). Evidence: review-load-life-mana-opus.md section 5 and finding F6. Done when the trace, the oracle and a draft fix exist and a verifier has checked the differential.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Trace game.exe resurrect (0x54c17c..0x54c198): which stats it refills (life, mana, stamina) and in which order, with a disassembly read
- [x] #2 Oracle or harness that runs the game.exe resurrect on a dead unit and diffs life, mana and stamina against d2d Fight::revive
- [x] #3 Fix draft (a test against two builds) for upstream Fight::revive, held in the bundle queue; no post without the user's go
<!-- AC:END -->

result: Trace docs/research/fid-revive-refill.md, draft docs/research/pr-revive-refill-draft.md, queue entry and verifier brief in docs/research/pr-bundle-queue.md. game.exe packet 0x41 handler FUN_0054c0e0 sets stats 6/8/0xa to the maxima of 7/9/0xb (0x54c17c/0x54c18a/0x54c198, V read); run for real in the emulator (tools/emu/fr_revive.py, 408 cases, 1508 on a second seed) against d2d World::respawn: 390 of 408 differ before the fix, 0 after. Candidate local/fid-revive-refill-r1 (c3dac20a, private remote only, never GitHub); test_game fails on the parent (rc 134) and passes with the fix. NOT independently verified, NOT opened; not verified live (no game was run; the repo's earlier live drill read mana 40/40 after a respawn). PR OPENED 2026-11-07: upstream PR #245 (https://github.com/psi29a/d2decomped/pull/245, head 170d920e, branch fix/respawn-refill); verifier review-revive-refill-opus.md HOLDS (HOLDS WITH LIMITS on gear sums); merge NOT ticked, not verified live. The criteria are ticked for the work done, the verifier and the opening come after.
