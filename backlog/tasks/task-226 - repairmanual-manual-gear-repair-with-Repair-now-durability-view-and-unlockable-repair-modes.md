---
id: TASK-226
title: 'repairmanual: manual gear repair with Repair now, durability view and unlockable repair modes'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-09T14:44:00Z'
labels:
  - watch
  - feature
dependencies: []
priority: medium
ordinal: 165000
card: "repairmanual"
owns:
  - "internal/repair"
  - "internal/watch/repairapi.go"
  - "internal/watch/repairapi_test.go"
  - "internal/watch/web/gear.js"
  - "cmd/d2bot/repairwire.go"
  - "internal/town/repairgate_test.go"
  - "docs/repair.md"
  - "internal/decide/policy_chores.json"
  - "internal/decide/testdata"
  - "data/unlocks/ladder.json"
  - "data/alerts/rules.json"
  - "internal/watch/policysheet.go"
  - "internal/watch/unlockalerts.go"
  - "internal/watch/invui.go"
  - "internal/town/chores.go"
  - "internal/town/town.go"
  - "cmd/d2bot/behaviors.go"
  - "cmd/d2bot/display.go"
  - "cmd/d2bot/invwire.go"
  - "cmd/d2bot/ladder.go"
  - "cmd/d2bot/live.go"
  - "cmd/d2bot/d2dbackend.go"
  - "internal/status/inventory.go"
  - "internal/decide/policy.go"
  - "internal/decide/policy_test.go"
  - "internal/watch/funwire.go"
  - "internal/watch/policysheet_test.go"
  - "internal/watch/unlockalerts_test.go"
  - "internal/watch/web/watch.html"
  - "internal/watch/web/alerts.js"
  - "internal/watch/testdata/policysheet"
exclusive: []
contracts: []
result: "Built and tested offline, NOT run live and the browser pieces (gear.js, policy buy button) not opened in a browser. Controller, API gates, policy locks, alerts and the two-class tests pass (go test ./internal/repair ./internal/watch ./internal/decide ./cmd/d2bot). The town gate tests (internal/town/repairgate_test.go) compile and vet but skip without the game tables, so AC 1 and 2 stay open until they run on a host with /tmp/d2mpq or live. Open choices in docs/repair.md (prices 30/50/70/90 are U, gear alerts free, named runs keep the legacy rule)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

User decision 2026-11-09: repair is not automatic at first. Done when the checklist holds.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Default repair policy is manual: the town repairs only after Repair now (or a bought mode)
- [ ] #2 Durability on the inventory page and a gear strip on /watch with warn/broken colours and an alert line
- [x] #3 Repair now POST (same gates as the inventory POSTs) and requested/done state in the UI
- [x] #4 Policy row repair with Manual plus four unlockable modes on the ladder; two-class tests
- [x] #5 docs/repair.md, tests offline, gate on the worker host
<!-- AC:END -->
