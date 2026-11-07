---
id: TASK-64
title: 'currency-slice: bank step, balance line and kill drill'
status: Deferred
assignee:
  - live-agent
created_date: '2026-11-07T03:13:00Z'
updated_date: '2026-11-07T21:56:00Z'
labels:
  - bot
  - feature
  - metacurrency
  - T5
  - parked
milestone: m-3
dependencies: []
priority: medium
ordinal: 62000
card: "currency-slice"
owns:
  - "internal/currencyflow/"
  - "docs/probes/currency-drill.md"
exclusive: []
contracts: []
result: "paused by direction review D1/D4/D5, 2026-11-07 (docs/research/direction-review.md section 6)"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

A currency must be banked and shown to be useful, so this adds the bank step, a balance line and a kill drill.

Done when: Town bank step transmutes the stack and commits the credit; watch state carries a currency block; Drill kills N in d.idle mode: pack count in binomial interval, credit equals loss, writes after start 0 (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Town bank step transmutes the stack and commits the credit; watch state carries a currency block
- [ ] #2 Drill kills N in d.idle mode: pack count in binomial interval, credit equals loss, writes after start 0
<!-- AC:END -->
