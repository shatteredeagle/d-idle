---
id: TASK-113
title: 'friendsfb: friends feedback via the tester container'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-07T21:56:00Z'
updated_date: '2026-11-10T16:54:00Z'
labels:
  - docs
  - research
  - tester
milestone: m-1
dependencies:
  - TASK-25
  - TASK-26
  - TASK-27
  - TASK-28
  - TASK-29
  - TASK-31
  - TASK-32
  - TASK-33
  - TASK-106
priority: high
ordinal: 106000
card: "friendsfb"
owns:
  - "docs/friends-feedback.md"
exclusive: []
contracts: []
result: "Status sync 2026-11-08 (stays open): AC1 open: docs/friends-feedback.md does not exist; merge was the direction record, not this card."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: direction review section 7 step 7 with the user's clarification of 2026-11-07 (D3): friends are remote, the container is the route to feedback; the review's LAN step does not apply. Done when: the one-page how-to (hand the built container to friends, what they send back, three sentences each) is merged and friend feedback is wired as the H-test audience for the design brief v1.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 One page: how a friend gets the built container, runs it with their own game files, plays a realtime session and sends back the scrubbed report plus three sentences (tester cards TASK-25 to 29, 31 to 33)
- [ ] #2 No hostnames, IPs or names; leakcheck 0; says friend feedback is the H1/H3/H4 audience for the design brief v1
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Closed in the 2026-11-10 true-up (docs/research/project-review-2026-11-10.md section 3.1): merged, offline acceptance proven; open live checks moved to docs/live-checks.md (TASK-253).
<!-- SECTION:NOTES:END -->
