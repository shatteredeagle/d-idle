---
id: TASK-28
title: "testerlicence: GPL package for the d.idle fork shipped with its source"
status: Proposed
assignee: []
created_date: '2026-11-06T14:27:00Z'
labels:
  - G-F
dependencies: []
ordinal: 28000
card: "testerlicence"
owns:
  - "docs/tester-licence.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Slice 1 (d2d route), reworked: the image now contains the d.idle fork of d2d, so the package covers the fork source. Plan gap G9. No dependency; the decisions themselves are the user's (ADR 0003 amendment, GPL reviewer, publication timing). Plans the publication; does not do it. Estimate 4 h.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 docs/tester-licence.md: NOTICE and SOURCE.txt text (without the NOT FOR DISTRIBUTION line), the written offer, the list of what ships (fork source tarball at the pinned ref, vendor inputs with hashes, build recipe, our source tarball, licence files), Ubuntu package source pointer, Freepik icon question, game data never shipped
- [ ] #2 the explicit USER DECISION list: ADR 0003 amendment first, GPL reviewer named or waived, when the d.idle fork repo is made public, whether lab and cheat verbs are stripped or flag-gated, the user's go on the day of the first copy; no legal advice claimed; nothing is published or posted
<!-- AC:END -->
