---
id: TASK-132
title: 'skillreqcheck: verify the SME skill clvl requirements and Static Field / Warmth / Teleport timing against the tables'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T00:27:00Z'
labels:
  - docs
  - research
  - ladder
milestone: m-4
dependencies: []
priority: medium
ordinal: 124000
card: "skillreqcheck"
owns:
  - "docs/research/skillreq-check.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: fun-ideas.md section 12 UNVERIFIED list; the SME's clvl requirements disagree with the author's memory of classic 1.14d (Frozen Armor, Ice Bolt, Warmth, Inferno, Ice Blast, Fire Ball, Nova, Lightning, Fire Wall). No code: a note. Done when: the criteria below hold.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Each skill of the SME schedule checked with data.DB.SkillByName(...).ReqLevel (offline, from the user's own tables); a table of SME value, table value, verdict (V with the command)
- [ ] #2 Static Field, Warmth and Teleport timing checked against docs/buildv3.md (v3 spends Teleport at 18; Catacombs run at clvl 14 to 17): recommendation for the Teleport unlock event; no Go changes; leakcheck 0
<!-- AC:END -->
