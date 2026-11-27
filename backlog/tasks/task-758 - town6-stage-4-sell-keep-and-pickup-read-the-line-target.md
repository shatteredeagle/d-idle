---
id: TASK-758
title: 'town6: stage 4 sell keep and pickup read the line target'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-21T04:52:00Z'
labels:
  - bot
  - feature
dependencies:
  - TASK-715
priority: medium
ordinal: 346000
card: "town6"
owns:
  - "internal/town/"
  - "internal/loot/"
  - "docs/design/townflip-p5.md"
exclusive: []
contracts: []
lane: next
result: "Stage 4 pickup side built behind town.plan 4 (default 0): a ground healing or mana potion the belt cannot take is wanted while the usable pack count is under the plan keep the sell chore protects, for the supply profile of the last town visit; below-floor potions never count; kinds without a plan line keep the old buffer; scrolls keep their caps. Verified offline only: unit tests and table-backed rig tests on Sorceress and Paladin pass on the worker host. NOT verified live: scenarios belt-refill-from-pack, sell-keeps-need, scroll-hoard not run. Build-specific part: none in Go; lines from supply profiles and the class belt plan."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: at town.plan 4 the sell chore keeps the plan line's target (merged in townflip); the pickup still used the fixed pack buffer, so the two could disagree. Done when: the pickup of pack potions reads the same plan keep as the sell chore, offline tests on two builds pass, and the three stage 4 scenarios are written down for the live run.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 sell keeps max(target,min); below-floor sold first; pickup want from the plan line
- [ ] #2 scenarios belt-refill-from-pack, sell-keeps-need, scroll-hoard
<!-- AC:END -->
