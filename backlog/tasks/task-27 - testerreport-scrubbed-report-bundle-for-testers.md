---
id: TASK-27
title: "testerreport: scrubbed report bundle for the d2d tester"
status: Proposed
assignee: []
created_date: '2026-11-06T14:27:00Z'
labels:
  - G-F
dependencies:
  - TASK-31
ordinal: 27000
card: "testerreport"
owns:
  - "deploy/d2d-container/tester-report.sh"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Slice 1 (d2d route), reworked: path moved to deploy/d2d-container. Depends on testerd2dimage (TASK-31). Estimate 3 h.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 docker exec script writes a tarball with bot log, d2d log, minute lines, deaths, ladder journal, wizard inputs without paths and the d2dping state; scrubs token, key-shaped text, home paths and hostnames; a test greps the output
<!-- AC:END -->
