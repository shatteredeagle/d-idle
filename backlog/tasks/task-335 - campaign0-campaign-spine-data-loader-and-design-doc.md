---
id: TASK-335
title: 'campaign0: campaign spine data, loader and design doc'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-11T20:09:00Z'
updated_date: '2026-11-12T00:01:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: high
ordinal: 225000
card: "campaign0"
owns:
  - "docs/campaign.md"
  - "data/campaign/"
  - "internal/campaign/"
exclusive: []
result: "data/campaign/normal.json (8 chapters, campaign/1) + internal/campaign (Parse/Validate, Rate bands, Next decision, Observe, Store); tests Paladin and Sorceress; diagnosis of the level.16 grind in docs/campaign.md section 1. Build-specific part: class rows and classIntent in the data file, Andariel from plandata. Not verified live."
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the user's Guided journey direction (2026-11-11): the hero follows the Act 1 campaign as a human would, nudged, never forced. Done when: the spine is data with a validated loader, readiness bands and the next-chapter decision are tested on two classes, and docs/campaign.md holds the design. Consumers: TASK-337 (planner), TASK-338 (API and Journey strip).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 docs/campaign.md design (spine, bands, modes, intents, nudges, journey strip)
- [x] #2 data/campaign/normal.json schema campaign/1 with the 8 Act 1 Normal chapters
- [x] #3 loader validates ids, prerequisites, quests, checks, class rows; errors name the chapter
- [x] #4 readiness band per chapter tested on Paladin and Sorceress
<!-- AC:END -->
