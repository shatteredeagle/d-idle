---
id: TASK-764
title: 'town11: cleanup, delete stage 0 and retired knobs'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-21T04:53:00Z'
labels:
  - bot
  - feature
dependencies:
  - TASK-715
priority: medium
ordinal: 351000
card: "town11"
owns:
  - "internal/town"
  - "docs/town.md"
  - "town-basics"
exclusive: []
contracts:
  - "town-basics"
lane: next
result: "NOT DONE, blocked on a precondition (no code changed): the card deletes stage 0, and stage 0 is today the default path (town.plan default 0, stages 3 to 5 still dark, townflip-p5 section 3 flip sequence not run, no release at stage 5). Deleting bankFirst/withdrawShop (read at internal/town/basicsfix.go:100,257,268 and stash.go:131; defined in internal/env/tuning_town.go:57,58,147,148 and mirrored in cmd/d2drill/stash_lab.go:35-43, both outside owns) would change the live default. Follow-up when stage 5 has held one release: make the stage 3 behaviour unconditional, drop the two knobs plus their rows in tuning_town.go, regenerate docs/knobs.md with go run ./tools/knobdump -write, reword the town-basics contract (bank before the shop) in the same branch, run the B4 set twice per class. Checkboxes left open."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 stage 0 code and bankFirst, withdrawShop, duplicate trip knobs removed
- [ ] #2 B4 set green twice per class
<!-- AC:END -->
