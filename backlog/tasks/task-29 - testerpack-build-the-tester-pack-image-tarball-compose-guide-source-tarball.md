---
id: TASK-29
title: "testerpack: build the d2d tester pack"
status: Proposed
assignee: []
created_date: '2026-11-06T14:27:00Z'
labels:
  - tester
dependencies:
  - TASK-31
  - TASK-28
ordinal: 29000
card: "testerpack"
owns:
  - "tools/tester-pack.sh"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Slice 1 (d2d route), reworked. Depends on testerd2dimage (TASK-31) and testerlicence (TASK-28). Building the pack locally is not publishing it; no copy leaves the machine without the user's go. Estimate 4 h.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 tools/tester-pack.sh builds a directory with the docker save gz of the d2d image, compose.tester.yml, the guide, the d.idle fork source tarball (pinned ref plus VERSION) with the vendor inputs, the d2lod-bot source tarball of the image commit, the licence files and sha256sums; contains no game file, save or token; a test checks the file list
<!-- AC:END -->
