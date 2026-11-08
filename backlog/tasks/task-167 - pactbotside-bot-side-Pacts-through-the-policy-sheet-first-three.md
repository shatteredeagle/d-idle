---
id: TASK-167
title: 'pactbotside: bot-side Pacts through the policy sheet, first three'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T03:33:00Z'
updated_date: '2026-11-08T04:02:00Z'
labels:
  - bot
  - feature
milestone: m-4
dependencies:
  - TASK-166
  - TASK-128
priority: low
ordinal: 143000
card: "pactbotside"
owns:
  - "internal/pact/botside.go"
  - "internal/pact/botside_test.go"
  - "internal/pact/catalogue.go"
  - "data/pacts/pacts.json"
  - "internal/watch/policysheet.go"
  - "internal/watch/policybypact_test.go"
  - "cmd/d2bot/pactwire.go"
  - "cmd/d2bot/pactwire_test.go"
  - "cmd/d2bot/ladder.go"
  - "docs/pact-botside.md"
exclusive: []
contracts: []
result: "Offline only, not verified live. Controller, away-consent record, safe zones, Off flag, return-screen lines, wiring in cmd/d2bot (not run live). Tithe and Shadow tested with Paladin and Sorceress; Rakanishu rank 3 via the generic path, rank 1 not coded. Widened owns (catalogue limits until_clvl/death_limit/safe/consent fields, sheet accessors, wiring). Open: presence detection and a UI to choose Pacts (TASK-168)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Size: M. Not on the two-week playable path. Design note: docs/research/pact-system.md (a design note, not a decision).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 (first two Pacts; Rakanishu rank 3/5 generic, rank 1 not coded) Blood Raven's Tithe, Andariel's Shadow and Rakanishu's Bargain apply and revert through the policy sheet and tuner
- [x] #2 Tests for two builds
- [x] #3 Safe zones (first 10 minutes or clvl 5, Den until quest done, town) keep Pacts off
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
The FIRST TWO Pacts are Blood Raven's Tithe (no belt potions) and Andariel's Shadow (death limit) (user 2026-11-07, ADR 0019).
<!-- SECTION:NOTES:END -->
