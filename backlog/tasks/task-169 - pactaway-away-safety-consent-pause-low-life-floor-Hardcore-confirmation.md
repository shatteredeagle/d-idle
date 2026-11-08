---
id: TASK-169
title: 'pactaway: away safety: consent, pause, low-life floor, Hardcore confirmation'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-08T03:33:00Z'
labels:
  - bot
  - feature
milestone: m-4
dependencies:
  - TASK-166
priority: low
ordinal: 145000
card: "pactaway"
owns:
  - "internal/pact/away.go"
  - "internal/pact/away_test.go"
  - "internal/pact/botside.go"
  - "internal/pact/catalogue.go"
  - "internal/watch/pactapi.go"
  - "internal/watch/pactapi_test.go"
  - "internal/watch/funwire.go"
  - "internal/watch/serve.go"
  - "cmd/d2bot/pactwire.go"
  - "cmd/d2bot/pactwire_test.go"
  - "docs/pact-away.md"
exclusive: []
contracts: []
result: "Built offline: presence, consent API, low-life pause, Hardcore flag + stub (docs/pact-away.md). Tests with fakes/httptest only; not verified live. Owns widened to the files above. #2 read as: the Pact pauses under the life floor and resumes (the hero is not held)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Size: M. Not on the two-week playable path. Design note: docs/research/pact-system.md (a design note, not a decision).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A Pact is active away only with a typed dated per-Pact consent
- [x] #2 Low-life floor pauses the hero in town
- [x] #3 Hardcore needs its own typed confirmation, off for away by default
<!-- AC:END -->
