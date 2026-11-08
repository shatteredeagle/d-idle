---
id: TASK-170
title: 'pactnative: verify /players, Hardcore and the Nightmare penalties offline on 1.14d'
status: Proposed
assignee:
  - live-agent
created_date: '2026-11-08T03:34:00Z'
updated_date: '2026-11-08T04:02:00Z'
labels:
  - bot
  - research
milestone: m-4
dependencies: []
priority: low
ordinal: 146000
card: "pactnative"
owns:
  - "docs/research/pact-native.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Size: S. Needs the live agent. Not on the two-week playable path. Design note: docs/research/pact-system.md (a design note, not a decision).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Live: /players 1..8 offline on 1.14d works; monster life, damage, drops and XP effect measured
- [ ] #2 Live or table check: Nightmare death XP penalty and resist penalty (-5 pct, -40) and Hell values
- [ ] #3 Whether a Hardcore hero can be created and played offline; every number tagged V or U
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Decides whether /players Pacts exist (ADR 0019).
<!-- SECTION:NOTES:END -->
