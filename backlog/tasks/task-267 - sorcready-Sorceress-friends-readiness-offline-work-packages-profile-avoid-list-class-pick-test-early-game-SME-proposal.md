---
id: TASK-267
title: >-
  sorcready: Sorceress friends-readiness offline work packages (profile avoid
  list, class-pick test, early-game SME proposal)
status: Done
assignee:
  - credit-runner
created_date: '2026-11-10T18:46:00Z'
updated_date: '2026-11-10T19:55:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 196000
card: "sorcready"
owns:
  - "docs/sorceress-early.md"
  - "docs/smoke.md"
  - "internal/combat/classdata.go"
  - "internal/combat/classdata/profiles.json"
  - "internal/combat/sorcready_test.go"
  - "internal/combat/pure_test.go"
  - "internal/combat/castgap_test.go"
  - "cmd/d2bot/sorcready_test.go"
exclusive: []
contracts: []
result: "Offline only, not verified live. Wizard pick does NOT select sorceress_normal30 (fresh Sorceress plays fallback-lightning-v2; tested). Lightning profile now avoids Chain Lightning and Lightning (generic avoid field, tests Sorceress and Paladin). Table unchanged; SME proposal and open questions in docs/sorceress-early.md. Live plan documented, not run."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Wizard class pick vs normal30 table verified by test
- [x] #2 SME consult written as proposal, table unchanged, open questions listed
- [x] #3 Lightning profile avoids Chain Lightning and Lightning, tests on two classes
- [x] #4 Live measurement plan documented, not run
<!-- AC:END -->
