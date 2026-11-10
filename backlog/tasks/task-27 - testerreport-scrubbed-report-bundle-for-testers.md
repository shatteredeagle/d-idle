---
id: TASK-27
title: 'testerreport: exportable feedback record (scrubbed bundle) for the friend beta'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-06T14:27:00Z'
updated_date: '2026-11-10T17:11:00Z'
labels:
  - container
  - feature
  - tester
milestone: m-1
dependencies:
  - TASK-31
priority: high
ordinal: 27000
card: "testerreport"
owns:
  - "deploy/d2d-container/tester-report.sh"
  - "docs/first-session.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S
Why: beta exit criterion 3 (docs/research/project-review-2026-11-10.md section 5.1) needs a feedback path that writes into the session record and an export the friend can send. This card is the export. It absorbs the report-preview hook of TASK-49 (onboardctl). The feedback box UI is on the watch screen, owned by TASK-47, not by this card.
Done when: a docker exec script writes a scrubbed tarball (bot log, d2d log, minute lines, deaths, ladder journal, wizard inputs without paths, d2dping state); a test greps the output for token, key-shaped text, home paths and hostnames. Depends on TASK-31.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 docker exec script writes a tarball with bot log, d2d log, minute lines, deaths, ladder journal, wizard inputs without paths and the d2dping state; scrubs token, key-shaped text, home paths and hostnames; a test greps the output
- [ ] #2 the known-broken table of docs/first-session.md is current and the bundle text tells the friend what to send (beta exit criterion 6; the watch-page link is TASK-47's)
<!-- AC:END -->
