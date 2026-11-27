---
id: TASK-741
title: 'pghero: hero library with played and banked profiles; survival prep uses it'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-21T04:47:00Z'
updated_date: '2026-11-27T07:24:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 347000
card: "pghero"
owns:
  - "internal/scenario/hero/"
  - "tools/ab/prep/"
  - "tools/survival/"
exclusive: []
contracts: []
lane: next
result: "Hero library internal/scenario/hero: played profile spends stat and skill points by the build table (leveler plan, quest skill points declared), binds the main attack from the build or combat profile, banked keeps every point; heroes are tagged CHEATED. Minimal self-check (level, points, attack learned and bound, weapon). The A/B prep resolves its hero from the library, self-checks before the save, writes hero.json with the run id; the survival run logs the hero line and abreport drops pairs with a failed or banked hero. Tested offline on Sorceress, Paladin and Necromancer; not verified live. Build-specific data stays in the plan tables and combat profiles."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Design section 6.2, 7 card 7. Design: docs/design/fable-review-2026-11-21/proving-ground-gate.md. Owner approved 2026-11-21 (all 10 questions as recommended).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Library with played and banked profiles
- [x] #2 survival prep resolves heroes from it
- [x] #3 scenario run id in the result (changes harness behaviour)
<!-- AC:END -->
