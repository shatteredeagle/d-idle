---
id: TASK-299
title: 'viewerfps2: deploy the d2d frame feed to the sandbox and compare viewer fps'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-11T14:29:00Z'
updated_date: '2026-11-11T16:25:00Z'
labels:
  - container
  - drill
dependencies: []
priority: medium
ordinal: 209000
card: "viewerfps2"
owns:
  - "docs/viewer-fps.md"
  - "deploy/d2d-container/entrypoint.sh"
exclusive: []
contracts: []
result: "Verified live on the sandbox: framefeed image devp2 (fork 80cc7adc) delivers 21.6 fps to a stream client (was 5.9), bot kills and legs per minute not lower; KEPT. Numbers in docs/viewer-fps.md section 6. No cheats, hero untouched."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 baseline measured before (fps, ms, CPU, bot pace)
- [x] #2 image built from fork commit 80cc7adc and sandbox recreated
- [x] #3 after numbers measured and keep/rollback decided
<!-- AC:END -->
