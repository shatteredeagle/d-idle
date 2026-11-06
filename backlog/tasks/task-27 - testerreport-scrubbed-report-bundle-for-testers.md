---
id: TASK-27
title: "testerreport: scrubbed report bundle for testers"
status: Proposed
assignee: []
created_date: '2026-11-06T14:27:00Z'
labels:
  - G-F
dependencies: []
ordinal: 27000
card: "testerreport"
owns:
  - "deploy/container/tester-report.sh"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Slice 1. Depends on testerimage.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 docker exec script writes a tarball with bot logs, minute lines, deaths, ladder journal, doctor json; scrubs token, key-shaped text, home paths and hostnames; a test greps the output (estimate 3 h)
<!-- AC:END -->
