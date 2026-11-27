---
id: TASK-1040
title: 'skillhand: cmd skill sends the hand the game reads'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-27T08:08:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 502000
card: "skillhand"
lane: next
owns:
  - "internal/d2dctl/verbs.go"
  - "internal/d2dctl/fake/fake.go"
  - "internal/d2dctl/skillhand_test.go"
  - "internal/d2dctl/client_test.go"
  - "internal/d2dctl/combat_verbs_test.go"
  - "internal/d2dinput/castverbs_test.go"
  - "internal/d2dinput/controller_test.go"
  - "internal/d2dcombat/castverbs_test.go"
  - "internal/d2dcombat/input_test.go"
  - "docs/d2dctl.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

The d2d devctl reads every cmd skill tail field with atoi, so the word left read as 0 (right hand) and a left cast with no unit sent 0 instead of -1, which skips the at-the-spot attack paths. Done when the line is numeric and the fake parses like d2d.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 CmdSkill sends the hand as 1 or 0 and no unit as -1, as d2d's devctl parser reads it (stock and fork)
- [ ] #2 Must-fail test: the fake d2d parses cmd skill like the fork and the old word form fails it
- [ ] #3 Live: a Paladin Zeal or Smite with cmd skill <id> x y -1 1 swings at the spot on a sandbox
<!-- AC:END -->
