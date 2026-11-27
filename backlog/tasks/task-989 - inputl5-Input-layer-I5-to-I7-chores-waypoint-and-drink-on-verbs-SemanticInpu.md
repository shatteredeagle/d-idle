---
id: TASK-989
title: 'inputl5: Input layer I5 to I7: chores, waypoint and drink on verbs; SemanticInpu'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-26T03:59:00Z'
labels:
  - bot
  - feature
dependencies:
  - TASK-988
priority: low
ordinal: 476000
card: "inputl5"
owns:
  - "docs/design/input-i5-i7.md"
  - "tools/ci/pointercount.py"
  - "tools/ci/test_pointercount.py"
  - "tools/ci/pointercount-baseline.json"
  - "tools/ci/check.sh"
exclusive: []
contracts: []
result: 'CI pointer-call ratchet (tools/ci/pointercount.py, baseline 107) wired into check.sh; repair on verbs via TASK-988; hotkey map already existed and is documented. SemanticInput-only refactor NOT done. Offline only, live U.'
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Recommendation R17 of the 2026-11-19 review (07 table I5, I6, I7). Bot side after the fork verbs: town chores on verbs; SemanticInput as the only interface of behaviour packages with a CI count of pointer call sites outside adapters (target 0 on the d2d path); the hotkey map generated from planner slots (knob input.hotkeyMap, dark). Risk: medium. Enable criteria: CI pointer count 0 on the d2d path; invdesk.verbs and mgmtfast ledger rows decided.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 CI pointer-call count wired and falling (ratchet wired, baseline 107; falling is future work)
- [x] #2 Hotkey map generated and read back (already built, V by cmd/d2bot/hotkeymapwire_test.go; live U)
<!-- AC:END -->
