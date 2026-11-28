---
id: TASK-1124
title: 'ops4: check.py fails a card whose front matter is not valid YAML; new_task.py quotes values'
status: In Progress
assignee:
  - pro-runner
created_date: '2026-11-28T06:44:00Z'
labels:
  - ops
  - chore
dependencies: []
priority: medium
ordinal: 533000
card: "ops4"
lane: next
owns:
  - "tools/backlog/"
  - "backlog/tasks/task-185 - unlockwire-wire-bestiary-risk-order-queue-and-atlas-into-the-live-bot.md"
  - "backlog/tasks/task-597 - vaultv7b-wire-the-ETHER-wallet-Vault-page-upgrade-UI-earn-call-sites.md"
  - "backlog/tasks/task-1100 - fix4-security-fixes-from-fable-review-4-scrubber-watch-token-difficulty-go-flip-run-check-log-tail.md"
exclusive: []
contracts: []
result: "Not verified live (tooling, nothing to run live). check.py names a card whose front matter is not valid YAML (task, line, key); new_task.py quotes unsafe values; TASK-185, TASK-597 and TASK-1100 load as YAML again. Tested by unit tests and a one-off cross-check against a YAML parser over every card. Build-specific: nothing."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: a card with an unquoted result containing ': ' cannot be found by the Backlog CLI (Task N not found), so auto-Done failed on TASK-1100 and check.py passed it. Done when: the checker names such cards and the three defective cards are repaired.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 check.py fails a card whose front matter is not valid YAML and names the task and the line (unit test with the two real defect shapes)
- [x] #2 TASK-185, TASK-597 and TASK-1100 load as YAML again; check.py is clean on the tree
- [x] #3 new_task.py quotes any value that is not a safe plain scalar and refuses to leave an invalid card (unit test)
<!-- AC:END -->
