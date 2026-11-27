---
id: TASK-557
title: 'seqfollow: upstream follow-up PR draft for #565 (seq-rate test hardening)'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-16T18:37:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - docs
  - upstream-pr
dependencies: []
priority: medium
ordinal: 246000
card: "seqfollow"
owns:
  - "docs/upstream/bodies/seq-ias-followup-body.md"
exclusive:
  - "upstream-pr:seq-ias-followup"
contracts: []
result: "Only 310ff43d substance was missing from merged #565; branch fid/seq-ias-followup (67d2262f) pushed to fork remotes, builder run: 36 cells ok, compare 7600 same, test_game/test_combat rc 0, tidy clean, pr_guard 0, presubmit --quick 0 FAIL; body drafted, not posted. Verified on the builder container, not live."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Follow-up branch fid/seq-ias-followup (fork 67d2262f) on upstream main, pushed to the private fork and the GitHub fork, not posted
- [x] #2 Evidence run: seq-rate 36 cells, compare 7600 same, test_game, test_combat, tidy, pr_guard, presubmit --quick
- [x] #3 PR body drafted in docs/upstream/bodies/seq-ias-followup-body.md
<!-- AC:END -->
