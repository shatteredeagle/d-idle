---
id: TASK-117
title: 'dietarchive: archive finished research docs with stubs'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-07T21:57:00Z'
labels:
  - docs
  - chore
milestone: m-4
dependencies: []
priority: low
ordinal: 110000
card: "dietarchive"
owns:
  - "docs/archive"
  - "docs/research/README.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: user decision D5. Done when: the archive list is merged.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 List candidates from docs/research/direction-review.md section 3 and 6 (research docs with no consumer or finished) and archive each with git mv, leaving a one-line stub and a status banner
- [ ] #2 docs/INDEX.md lines go through merge-lines; links still resolve; leakcheck 0
<!-- AC:END -->
