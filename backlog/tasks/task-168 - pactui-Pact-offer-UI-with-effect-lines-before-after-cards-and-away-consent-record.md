---
id: TASK-168
title: 'pactui: Pact offer UI with effect lines, before/after cards and away-consent record'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T03:33:00Z'
labels:
  - watch
  - feature
milestone: m-4
dependencies:
  - TASK-166
  - TASK-127
priority: low
ordinal: 144000
card: "pactui"
owns:
  - "internal/watch/pactui.go"
  - "internal/watch/pactui_test.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Size: M. Not on the two-week playable path. Design note: docs/research/pact-system.md (a design note, not a decision).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Offer shows current vs proposed policy, heat against cap and incompatibility reasons
- [ ] #2 Return screen section shows what each Pact cost and paid
- [ ] #3 Away-consent record stored in the decide store
<!-- AC:END -->
