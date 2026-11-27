---
id: TASK-1079
title: 'relaunches: reason line per relaunch, refusal-storm top reasons'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-27T23:25:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 517000
card: "relaunches"
lane: next
owns:
  - "internal/bot"
  - "cmd/d2bot/behaviors.go"
  - "cmd/d2bot/d2dwire.go"
exclusive: []
contracts: []
result: "Logging only, knob-free: relaunch reason=<code> cause=<..> Info line at all 7 relaunch sites, minute keys relaunches/relaunchBy, refusal-storm detail top=[..]. Offline, not verified live. Behaviour change (coded park reason for repeated refusals) is a proposal in the report."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: problems 6 and 10 of docs/probes/sandbox-log-analysis-2026-11-27.md: no count of relaunches by reason, and a refusal-storm line without the refusals. Done when: each relaunch has a reason line, the storm names its top reasons.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 every relaunch writes one Info line relaunch reason=<code> cause=<..> and the minute line counts them
- [x] #2 refusal-storm incident lists the top three refusal reasons
- [x] #3 must-fail tests with two heroes
<!-- AC:END -->
