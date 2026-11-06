---
id: TASK-29
title: "testerpack: build the tester pack"
status: Proposed
assignee: []
created_date: '2026-11-06T14:27:00Z'
labels:
  - G-F
dependencies: []
ordinal: 29000
card: "testerpack"
owns:
  - "tools/tester-pack.sh"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Slice 1. Depends on testerimage and testerlicence.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 tools/tester-pack.sh builds a directory with docker save gz, compose.tester.yml, guide, source tarball of the image commit and sha256sums; contains no game file or token; a test checks the file list (estimate 3 h)
<!-- AC:END -->
