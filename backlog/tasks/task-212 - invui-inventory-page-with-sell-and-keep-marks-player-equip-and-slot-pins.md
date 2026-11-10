---
id: TASK-212
title: 'invui: inventory page with sell and keep marks, player equip and slot pins'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-09T13:53:00Z'
updated_date: '2026-11-10T16:54:00Z'
labels:
  - watch
  - feature
dependencies: []
priority: medium
ordinal: 163000
card: "invui"
owns:
  - "internal/invdesk"
  - "internal/watch/invapi.go"
  - "internal/watch/invapi_test.go"
  - "internal/watch/invui.go"
  - "internal/leveler/playerequip.go"
  - "internal/leveler/playerequip_test.go"
  - "internal/town/invmarks_test.go"
  - "cmd/d2bot/invwire.go"
  - "docs/inventory-ui.md"
  - "internal/watch/funwire.go"
  - "internal/watch/web/watch.html"
  - "internal/leveler/equip.go"
  - "internal/leveler/leveler.go"
  - "internal/town/town.go"
  - "internal/town/shop.go"
  - "internal/town/stashfull.go"
  - "internal/town/rebuy.go"
  - "internal/town/ammo.go"
  - "internal/bot/contracts.go"
  - "internal/bot/runner.go"
  - "internal/bot/leveling.go"
  - "cmd/d2bot/behaviors.go"
  - "cmd/d2bot/d2dwire.go"
  - "cmd/d2bot/d2dbackend.go"
  - "cmd/d2bot/live.go"
  - "cmd/d2bot/ladder.go"
exclusive: []
contracts: []
result: "Offline only: desk, routes and wiring tested; table-dependent town and leveler tests compile but skip without game tables (U). Not verified live. Deployed to the preview sandbox, see docs/inventory-ui.md."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Player inventory control on the watch screen. Done when the page, marks, equip and pins pass the offline tests and the live checklist.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Inventory page at /watch/inventory shows equipped slots, backpack, stash and belt with name, quality, props and a why line
- [ ] #2 Sell and keep marks persist in the state dir and the town honours them (never-sell items refused)
- [ ] #3 Equip request is queued, refused with a reason (not in town, class or stat requirements), done at the gear step through the normal click path
- [ ] #4 A player equip pins the slot; the bot never undoes it; only the player's unpin ends a pin
- [ ] #5 POST routes use the answer listener gates; two-class tests; live drill checklist in docs/inventory-ui.md
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Closed in the 2026-11-10 true-up (docs/research/project-review-2026-11-10.md section 3.1): merged, offline acceptance proven; open live checks moved to docs/live-checks.md (TASK-253).
<!-- SECTION:NOTES:END -->
