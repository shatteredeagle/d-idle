---
id: TASK-253
title: 'livechecks: docs/live-checks.md and the closing rule'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-10T16:55:00Z'
updated_date: '2026-11-10T16:55:00Z'
labels:
  - docs
  - chore
milestone: m-4
dependencies: []
priority: low
ordinal: 191000
card: "livechecks"
owns:
  - "docs/live-checks.md"
  - "tools/backlog/drift.py"
  - "tools/backlog/test_drift.py"
  - "tools/ci/check.sh"
exclusive: []
contracts: []
branch: "agent/autodone"
result: "Auto-Done is ON in the worker (env file, daemon restarted when the queue was empty); live-check boxes become docs/live-checks.md rows at merge; catch-up scan applied (counts in the merge summary); drift warning in tools/ci/check.sh. Verified on the real worker log and a scratch-repo rehearsal; the live-check rows themselves are unverified live by nature."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: finished cards stayed open for boxes that only ask for a live check. Done when: auto-Done is on, such boxes move to docs/live-checks.md at merge, the catch-up scan closed the merged cards, and CI prints a drift warning.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 docs/live-checks.md exists with one row per moved check and the closing rule of review section 3.1
- [x] #2 auto_done.py moves live-check boxes to docs/live-checks.md and closes the card (ops repo, tests in tools/tests/test_auto_done.py); MERGEQ_AUTO_DONE=1 durable in the worker env file; catch-up scan applied
- [x] #3 drift report printed as a CI warning (check.sh runs drift.py --summary, exit 0)
<!-- AC:END -->
