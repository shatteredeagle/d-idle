---
id: TASK-245
title: 'firsthour: fresh Paladin 60 realtime minutes at 1x daily table'
status: Proposed
assignee:
  - live-agent
created_date: '2026-11-10T16:55:00Z'
labels:
  - ops
  - drill
milestone: m-4
dependencies: []
priority: high
ordinal: 183000
card: "firsthour"
owns:
  - "docs/first-hour.md"
  - "cmd/d2firsthour/"
  - "tools/firsthour.sh"
  - "docs/probes/firsthour.md"
exclusive: []
contracts: []
result: "tooling built and fixture-tested (cmd/d2firsthour, tools/firsthour.sh); NOT run live, no daily rows yet; AC1 stays open"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: the headline metric (user-approved 2026-11-10) that makes sessions comparable. Done when: the table has three consecutive daily rows.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 daily table logged: fresh Paladin, 60 realtime minutes at 1x: clvl, kills, gold, chickens, zone entries
<!-- AC:END -->
