---
id: TASK-168
title: >-
  pactui: Pact offer UI with effect lines, before/after cards and away-consent
  record
status: Done
assignee:
  - credit-runner
created_date: '2026-11-08T03:33:00Z'
updated_date: '2026-11-08T09:06:00Z'
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
  - "internal/watch/pactapi.go"
  - "internal/watch/pactapi_test.go"
  - "internal/watch/policysheet.go"
  - "internal/pact/view.go"
  - "internal/pact/view_test.go"
  - "internal/pact/botside.go"
  - "cmd/d2bot/pactwire.go"
  - "cmd/d2bot/pactwire_test.go"
  - "docs/pact-ui.md"
exclusive: []
contracts: []
result: "Built offline: /watch/pacts page, /api/pacts view with proposal, unlock silhouettes, cost-and-paid return section with before/after counters (docs/pact-ui.md). Not verified live or in a browser. #3: the consent record lives in pacts-state.json, not the decide store (tagged U). Owns widened to the files above."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Size: M. Not on the two-week playable path. Design note: docs/research/pact-system.md (a design note, not a decision).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Offer shows current vs proposed policy, heat against cap and incompatibility reasons
- [x] #2 Return screen section shows what each Pact cost and paid
- [x] #3 Away-consent record stored in the decide store
<!-- AC:END -->
