---
id: TASK-657
title: 'forkrebase2: regular fork rebase routine, script and first round (devp13)'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-19T14:06:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - d2d
  - chore
dependencies: []
priority: medium
ordinal: 246000
card: "forkrebase2"
owns:
  - "docs/process/fork-rebase.md"
  - "docs/process/sandbox-devp13.md"
  - "tools/fork-rebase.sh"
exclusive: []
contracts: []
result: "first round done 2026-11-19: didle/rebase-20261119 tip f32837d8 (rebase 89ea1d19, 50 commits on upstream da98b7f89, 236 behind, 3 text conflicts + 1 semantic), d2d built on the builder (532 s), 10 of 11 fork verb tests pass (incl. new item create identified=0|1 flag for the vault reveal), smoke_d2d fails identically on pure upstream main; bot TestRealD2D (d2dctl, d2dsrc) pass against it; image d2dbot:devp13 97e985ca5eb5 loaded on the sandbox host, container still on devp12. Upstream ctest and a full d2drill leg not run. Verified on the builder, not live on the sandbox."
lane: now
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the private d2d fork must follow upstream regularly (user decision 2026-11-19). Done when: routine doc, script and first round (devp13 loaded, not switched) are merged.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 docs/process/fork-rebase.md: routine, conflict policy, tests, regression checklist
- [x] #2 tools/fork-rebase.sh measure/rebase/test/build, stops on conflict
- [x] #3 first round: rebase onto upstream main (didle/rebase-20261119), tests, image devp13 loaded not switched
- [x] #4 docs/process/sandbox-devp13.md: payloads, diff vs devp12, rollback
<!-- AC:END -->
