---
id: TASK-396
title: 'botadapt: debug players parse on rebased d2d (kill drop)'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-13T16:48:00Z'
updated_date: '2026-11-13T17:14:00Z'
labels:
  - bot
  - bug
  - tester
dependencies: []
priority: high
ordinal: 247000
card: "botadapt"
result: "Cause V: upstream d2d 4307d9c8 appended a level column to debug players; the end-anchored ParsePlayer misread the hero position, PathTo failed and combat dropped targets as unreachable. Fix: parse from the front (TestParsePlayerColumns). Paired 10-min runs with the fixed bot: new d2d 76/87 kills vs old 90/68, 0 unreachable. context.sh SOURCE.txt fixed. Verified on throwaway heroes on the worker host, not on the sandbox."
owns:
  - "internal/d2dctl/parse.go"
  - "internal/d2dctl/parse_test.go"
  - "deploy/d2d-container/context.sh"
  - "docs/probes/killdrop-2026-11-13.md"
  - "docs/probes/test-characters.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

The rebased fork (TASK-395) halved kills under the same bot. Done when: the cause is verified, the parser fixed and the paired runs agree.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Cause of the kill drop on dd70428d found and verified
- [x] #2 Bot fix with a unit test for both reply shapes
- [x] #3 Paired 10-minute runs, fixed bot, old vs new d2d, seeds 7 and 11
- [x] #4 context.sh SOURCE.txt names the published URL and the real ref
<!-- AC:END -->
