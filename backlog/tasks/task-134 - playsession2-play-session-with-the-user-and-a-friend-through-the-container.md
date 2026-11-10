---
id: TASK-134
title: 'playsession2: play session with the user and a friend through the container'
status: Done
assignee:
  - live-agent
created_date: '2026-11-08T00:27:00Z'
updated_date: '2026-11-10T16:54:00Z'
labels:
  - container
  - drill
  - tester
  - ladder
milestone: m-4
dependencies:
  - TASK-25
  - TASK-33
  - TASK-113
  - TASK-106
  - TASK-127
  - TASK-128
  - TASK-129
  - TASK-130
priority: high
ordinal: 126000
card: "playsession2"
owns:
  - "docs/probes/playsession2.md"
exclusive: []
contracts: []
result: "Status sync 2026-11-08 (stays open): AC1 open: the 4-hour realtime session has not run; merge was the guide only (firstsession)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: ADR 0018, fun-ideas.md section 13 card 6; replaces the H1 to H4 sessions of the direction review as the playtest of the loop (TASK-107 and TASK-112 feed the same numbers). The two-week playable is cards returnscreen, policysheet, tallystrip, codexfirsts, then this one. The user and the friend do the reading (assignee is the live runner for the 4-hour run; the reading is theirs). Kill: fewer than 3 rules or policies set from the return screen in 5 minutes; no wish to leave it running. Done when: the criteria below hold.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 One 4-hour unattended realtime run through the tester container with the four cards merged (proposezone if merged); writes after start 0 stated; session record written
- [ ] #2 The user reads the return screen and sets policies; then the same with the first friend; three sentences each and the H1 to H4 numbers of the direction review logged; verdict: would I leave it running tonight
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Closed as superseded in the 2026-11-10 true-up: folded into the sessions card (see TASK-246), one record format docs/session-record.md.
<!-- SECTION:NOTES:END -->
