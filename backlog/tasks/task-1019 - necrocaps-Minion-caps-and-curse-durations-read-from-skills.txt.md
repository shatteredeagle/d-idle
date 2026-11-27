---
id: TASK-1019
title: 'necrocaps: Minion caps and curse durations read from skills.txt'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-26T19:14:00Z'
updated_date: '2026-11-27T02:03:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 490000
card: "necrocaps"
lane: next
owns:
  - "docs/probes/necrocaps-2026-11-26.md"
  - "internal/combat/classdata.go"
  - "internal/combat/classdata/profiles.json"
  - "internal/combat/necro.go"
  - "internal/combat/necro_data_test.go"
exclusive: []
contracts: []
result: "Minion caps carry a required source tag and the reach and curse-memory numbers are profile data (necro.rules) with today values as defaults; a cap without a tag is refused at load. Reading the caps from skills.txt is NOT done (calc columns not parsed, U); capSeen stays the evidence path. Offline only."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 see docs/probes/necro-readiness-2026-11-26.md cards table
<!-- AC:END -->
