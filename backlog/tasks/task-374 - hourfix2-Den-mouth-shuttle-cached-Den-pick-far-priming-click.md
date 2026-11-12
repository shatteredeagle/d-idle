---
id: TASK-374
title: 'hourfix2: cached Den pick after the claim'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-12T18:27:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 240000
card: "hourfix2"
result: "Cached Den pick dropped after the claim (V by test); the kick change was dropped in favour of agent/bounce3. Not verified live."
owns:
  - "docs/hourfix2.md"
  - "internal/bot/goal.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Fix for the second fresh hour. Done when tests pass; live effect unverified.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 cached Den pick dropped after the claim
- [x] #2 (dropped: bounce3 owns the priming click)
- [x] #3 gates green
<!-- AC:END -->
