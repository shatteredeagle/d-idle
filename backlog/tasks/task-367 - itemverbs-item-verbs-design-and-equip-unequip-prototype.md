---
id: TASK-367
title: 'itemverbs: item verbs design and equip/unequip prototype'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-12T16:32:00Z'
labels:
  - d2d
  - feature
dependencies: []
priority: medium
ordinal: 237000
card: "itemverbs"
owns:
  - "docs/design/item-verbs.md"
  - "internal/invdesk/verbs.go"
  - "internal/invdesk/verbs_test.go"
  - "internal/d2dctl/item_verbs.go"
  - "internal/d2dctl/item_verbs_test.go"
  - "internal/env/tuning_invdesk.go"
exclusive:
  - "fork-branch:fork-only/itemverbs"
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Private-fork item verbs (user decision 2026-11-12): design doc plus one vertical slice (equip, unequip, caps) in the fork and the bot executor interface. Done when the doc, the slice and the fakes tests are green.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 docs/design/item-verbs.md written (verbs, rules, errors, conformance plan, draft ADR text)
- [ ] #2 Fork branch fork-only/itemverbs: equip/unequip/caps with a d2d unit test, pushed to the private forge
- [ ] #3 Bot: verb-first equip with click fallback behind knob invdesk.verbs (default off), tests with fakes for two classes
- [ ] #4 Gate: per-package tests, whole-repo labrun gate, contracts and backlog checks, leakcheck
<!-- AC:END -->
