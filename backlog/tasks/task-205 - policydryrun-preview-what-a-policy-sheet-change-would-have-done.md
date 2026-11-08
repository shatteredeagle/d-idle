---
id: TASK-205
title: 'policydryrun: preview what a policy sheet change would have done'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T19:47:00Z'
labels:
  - watch
  - feature
milestone: m-4
dependencies: []
priority: high
ordinal: 160000
card: "policydryrun"
owns:
  - "internal/decide/preview.go"
  - "internal/decide/preview_rows.json"
  - "internal/decide/preview_test.go"
  - "internal/watch/policypreview.go"
  - "internal/watch/policypreview_test.go"
  - "docs/policy-preview.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

From docs/research/incremental-arpg.md proposal 1. Done when the sheet shows a preview before saving.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Preview(row, option, window) replays recorded events and reports deltas only where the data supports it
- [ ] #2 stance and whites rows have working previews; every other row says why not
- [ ] #3 POST /api/policy/preview is token-gated, same-origin, read-only
- [ ] #4 page shows the preview card with Save and Cancel and a Filtered out list
- [ ] #5 tests: two classes, empty history, unknown row, race, golden JSON, gates
<!-- AC:END -->
