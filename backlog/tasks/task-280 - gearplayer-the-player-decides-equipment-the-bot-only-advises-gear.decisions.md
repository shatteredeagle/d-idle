---
id: TASK-280
title: 'gearplayer: the player decides equipment, the bot only advises (gear.decisions)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T00:43:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 203000
card: "gearplayer"
owns:
  - "docs/gear-policy.md"
  - "internal/leveler"
  - "internal/town"
  - "internal/bot"
  - "internal/resource"
  - "internal/invdesk"
  - "internal/env"
  - "cmd/d2bot"
  - "docs/knobs.md"
exclusive: []
contracts: []
result: "Offline only, not verified live. Knob resource.gearDecisions (0 player default, 1 bot; D2BOT_RUN or D2BOT_GEAR=bot select bot). Player policy: no gear step, no gear rebuy, no found-gear equip, no belt or bow buy, gear gate never closes the field over a missing slot (amber instead). Build-specific data: none added (NeedsWeapon table in internal/resource). Tests: resource, bot, town with real tables on the worker host. Unverified: watch page rendering of View.Prompt, unarmed Paladin fight live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Size: L. User decision 2026-11-10 fully player-only equipment. Done when the player policy is the default and legacy stays behind the knob.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 gear.decisions knob (resource.gearDecisions), default player
- [x] #2 no bot equip, swap or buy of equipment under player; potions, repair, sell, pickup unchanged
- [x] #3 advice and onboarding prompt feed invdesk
- [x] #4 tests for Paladin and Sorceress; docs/gear-policy.md
<!-- AC:END -->
