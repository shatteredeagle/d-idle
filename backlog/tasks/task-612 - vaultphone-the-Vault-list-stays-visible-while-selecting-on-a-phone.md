---
id: TASK-612
title: 'vaultphone: the Vault list stays visible while selecting on a phone'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-17T12:56:00Z'
labels:
  - watch
  - bug
dependencies: []
priority: medium
ordinal: 309000
card: "vaultphone"
owns:
  - "internal/watch/web/vault.js"
  - "tools/uishots.sh"
exclusive: []
contracts: []
result: "Starting a selection folds the Vault filter panel (vault.js), so the list and the Reveal cost line stay visible on phones; vault-reveal-multi pages run at all sizes again. Pictures viewed (menu at 1280x620 and 360x640 with the scroll body, Pause/Reset row, multi-select with the cost line at 1366x768 and phone sizes). Fit status in the merge report. Closed by closecards2: merged, ACs verified against the merge diff and tests."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Found by the menufit2 multi-select fixture: filter panel open plus the selection bar left no room for the list on a phone.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 selecting folds the filter panel so the list and the cost line stay on screen at 412x780 and 360x640
- [x] #2 vault-reveal-multi pages pass at every size
<!-- AC:END -->
