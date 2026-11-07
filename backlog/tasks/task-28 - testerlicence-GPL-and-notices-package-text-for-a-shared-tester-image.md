---
id: TASK-28
title: 'testerlicence: GPL package for the d.idle fork shipped with its source'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-06T14:27:00Z'
updated_date: '2026-11-07T14:24:00Z'
labels:
  - container
  - feature
  - tester
milestone: m-1
dependencies: []
priority: high
ordinal: 28000
card: "testerlicence"
owns:
  - "docs/tester-licence.md"
exclusive: []
contracts: []
result: "partial: docs/tester-licence.md drafted (NOTICE, SOURCE.txt, offer, ship list, decision list); AC1 stays open until vendor hashes and the Ubuntu source list are cut from a real build and the Freepik terms are checked; not verified live Status sync 2026-11-07: stays open, AC 1 needs the vendor hashes and the Ubuntu source list cut from a real build (placeholders at pack cut)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Slice 1 (d2d route), reworked: the image now contains the d.idle fork of d2d, so the package covers the fork source. Plan gap G9. No dependency; the decisions themselves are the user's (ADR 0003 amendment, GPL reviewer, publication timing). Plans the publication; does not do it. Estimate 4 h.

Done when: docs/tester-licence.md: NOTICE and SOURCE.txt text (without the NOT FOR DISTRIBUTION line), the written...; the explicit USER DECISION list: ADR 0003 amendment first, GPL reviewer named or waived, when the d.idle... (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 docs/tester-licence.md: NOTICE and SOURCE.txt text (without the NOT FOR DISTRIBUTION line), the written offer, the list of what ships (fork source tarball at the pinned ref, vendor inputs with hashes, build recipe, our source tarball, licence files), Ubuntu package source pointer, Freepik icon question, game data never shipped
- [x] #2 the explicit USER DECISION list: ADR 0003 amendment first, GPL reviewer named or waived, when the d.idle fork repo is made public, whether lab and cheat verbs are stripped or flag-gated, the user's go on the day of the first copy; no legal advice claimed; nothing is published or posted
<!-- AC:END -->
