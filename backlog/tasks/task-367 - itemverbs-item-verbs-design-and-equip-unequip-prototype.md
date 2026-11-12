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
  - "internal/env/tuning.go"
  - "internal/invdesk/desk.go"
  - "internal/d2dctl/fake/fake.go"
  - "docs/knobs.md"
exclusive:
  - "fork-branch:fork-only/itemverbs"
contracts: []
result: "Design doc docs/design/item-verbs.md plus the equip/unequip/caps slice: fork branch fork-only/itemverbs commit 68318c097aa6 (private forge; fork unit test and headless e2e green, verb vs manual command sequence); bot VerbEquipper (verb first, click fallback, how=verb|click) behind knob invdesk.verbs (default off, not wired into cmd/d2bot). Verified offline and headless only, NOT live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Private-fork item verbs (user decision 2026-11-12): design doc plus one vertical slice (equip, unequip, caps) in the fork and the bot executor interface. Done when the doc, the slice and the fakes tests are green.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 docs/design/item-verbs.md written (verbs, rules, errors, conformance plan, draft ADR text)
- [x] #2 Fork branch fork-only/itemverbs: equip/unequip/caps with a d2d unit test, pushed to the private forge
- [x] #3 Bot: verb-first equip with click fallback behind knob invdesk.verbs (default off), tests with fakes for two classes
- [ ] #4 Gate: per-package tests, whole-repo labrun gate, contracts and backlog checks, leakcheck
<!-- AC:END -->
