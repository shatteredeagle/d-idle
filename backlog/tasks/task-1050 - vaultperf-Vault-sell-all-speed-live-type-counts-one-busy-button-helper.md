---
id: TASK-1050
title: 'vaultperf: Vault sell-all speed, live type counts, one busy-button helper'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-27T18:28:00Z'
labels:
  - watch
  - bug
dependencies: []
priority: medium
ordinal: 510000
card: "vaultperf"
lane: next
owns:
  - "internal/watch"
  - "internal/watch/web"
  - "internal/vaultkeep"
  - "internal/vault"
exclusive: []
contracts: []
result: "Sell all batches its writes (SellNowBatch: one archive write, one log write, one fsync); every reload carries the type counts; dBusy in toast.js gives every POST button an immediate working/done/failed state. Verified in Go and node tests only, not on a real page (first-load wire work was already done by the earlier vaultperf probe). Remaining (live): AC 5 first load and button states on a real page; the card then closes. closecards3: status set to Done, merged on main; only the live or lab boxes above remain open (listed in Remaining)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

User report 2026-11-27: slow first load, Sell all 5-10 s, counts stale, no feedback on buttons (Hand in). Done when the four are fixed.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Sell all writes its records in one batch (one fsync), not two per record
- [x] #2 A reload always carries the type counts, so counts move after Sell all without a refresh
- [x] #3 One busy-button helper (dBusy in toast.js): a click marks the button before the request returns, swallows a second tap, ends done or failed
- [x] #4 Node and Go tests for the batch sale, the counts and the helper; asset budget green
- [ ] #5 First load and the button states checked on a real page (live check, not done here)
<!-- AC:END -->
