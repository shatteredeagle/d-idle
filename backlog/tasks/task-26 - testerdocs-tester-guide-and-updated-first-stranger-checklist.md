---
id: TASK-26
title: "testerdocs: d2d tester guide and first-stranger checklist"
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-06T14:27:00Z'
labels:
  - docs
  - chore
  - tester
milestone: m-1
dependencies:
  - TASK-31
priority: high
ordinal: 26000
card: "testerdocs"
owns:
  - "docs/tester-guide.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Slice 1 (d2d route), reworked from the Wine version: docs/container.md (Wine) is no longer touched; the checklist lives in the guide. Depends on testerd2dimage (TASK-31); final pass after testerlive. Estimate 4 h.

Done when: docs/tester-guide.md: prerequisites, the MPQ files to bring, compose route, token, wizard, watch URL and...; the first-stranger checklist with pass criteria (compose, wizard, card, answer, order, video, restart,... (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 docs/tester-guide.md: prerequisites, the MPQ files to bring, compose route, token, wizard, watch URL and ports, what to expect (realtime), stop and update, troubleshooting, known weaknesses (g2e findings, 9 of 22 cards, nothing verified past Act 1), what to send back
- [ ] #2 the first-stranger checklist with pass criteria (compose, wizard, card, answer, order, video, restart, bundle, update) is part of the guide; no private hostnames or paths
<!-- AC:END -->
