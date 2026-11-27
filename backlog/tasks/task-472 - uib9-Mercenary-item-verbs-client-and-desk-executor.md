---
id: TASK-472
title: 'uib9: Mercenary item verbs, client and desk executor'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-15T04:59:00Z'
updated_date: '2026-11-27T02:03:00Z'
labels:
  - d2d
  - feature
dependencies: []
priority: medium
ordinal: 266000
card: "uib9"
owns:
  - "internal/d2dctl/item_verbs_merc.go"
  - "internal/d2dctl/item_verbs_merc_test.go"
  - "internal/invdesk/mercverbs.go"
  - "internal/invdesk/mercverbs_test.go"
  - "internal/invdesk/sets.go"
  - "cmd/d2bot/invverbs.go"
  - "cmd/d2bot/invverbs_test.go"
  - "docs/research/merc-gear.md"
exclusive: []
contracts: []
result: "Go half done and tested offline for two hirelings and each refusal; fork half NOT done (no fork checkout here); not verified live."
lane: now
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the player cannot give gear to the mercenary or take it off without the click path, which does not exist. Done when: the Go half (client verbs, desk executor on the instant lane, wiring) is merged and refuses as before until a fork build lists the capabilities; the fork half is a separate job on the worker host with the private fork (criterion 1, open). Live drill: docs/research/merc-gear.md section 8.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Fork tests green on the worker host; the fork commit is on the private remote and named in the card (NOT DONE: no fork checkout on this host)
- [x] #2 Go tests with a stub server for both verbs and each refusal code; Desk tests for two mercenary types
- [x] #3 usable.merc and the verb agree (one Judge); disagreements listed in docs/research/merc-gear.md section 8
- [x] #4 Compat goldens unchanged
<!-- AC:END -->
