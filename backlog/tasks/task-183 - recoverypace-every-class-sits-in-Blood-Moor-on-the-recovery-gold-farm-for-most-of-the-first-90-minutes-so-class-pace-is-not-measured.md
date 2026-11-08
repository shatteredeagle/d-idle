---
id: TASK-183
title: 'recoverypace: every class sits in Blood Moor on the recovery gold farm for most of the first 90 minutes, so class pace is not measured'
status: Proposed
assignee:
  - live-agent
created_date: '2026-11-08T11:07:00Z'
labels:
  - bot
  - bug
milestone: m-4
dependencies: []
priority: medium
ordinal: 152000
card: "recoverypace"
owns:
  - "docs/soaks/recoverypace.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Evidence: docs/probes/classpacing/minutes/bar1.jsonl (68 of 90 minutes in area 2, clvl 4 at minute 90, 238 kills, step 'Recovery: farming gold') and 'area skipped dry-recovery 900s' in the assassin run; same finding as follow-up 1 of docs/palmelee.md. Done when: the criterion holds.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Barbarian lockstep run reaches clvl 6 within 90 virtual minutes or the card result names why not; the recovery farm leaves a dry area before 15 minutes
<!-- AC:END -->
