---
id: TASK-525
title: >-
  lootwire-L4: shadow stage logging disagreements between today and the Starter
  loot rules
status: Done
assignee:
  - credit-runner
created_date: '2026-11-16T02:11:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - bot
  - feature
dependencies:
  - TASK-524
priority: medium
ordinal: 285000
card: "lootwireL4"
owns:
  - "internal/loot/lootshadow.go"
  - "internal/loot/lootshadow_test.go"
  - "internal/watch/lootshadowapi.go"
  - "internal/watch/lootshadowapi_test.go"
  - "docs/lootrules.md"
  - "docs/knobs.md"
  - "internal/loot/pickup.go"
  - "internal/loot/loot.go"
  - "internal/town/tidy.go"
  - "internal/town/shop.go"
  - "internal/town/tidy_test.go"
  - "internal/env/tuning_bot.go"
  - "internal/watch/funwire.go"
  - "cmd/d2bot/ladder.go"
  - "cmd/d2bot/lootshadowwire.go"
exclusive: []
contracts: []
result: "Shadow stage wired log-only: loot.Shadow/ShadowPass fed from Looter.candidates, town tidy and sell; bot.lootShadow knob (default 1), GET /api/lootrules/shadow (watch.lootshadow/1). Starter equals today on 5040 pack and 3360 ground decisions (strictness tier folded in), 250 of 840 sell decisions disagree (documented, docs/lootrules.md section 11); about 6-9 us per decision. Not verified live."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the loot rules engine must prove it equals today before it takes over (docs/lootrules.md section 11). Done when: the shadow logs and counts disagreements at the ground, tidy and sell decisions, log only. Consumer: wave 2 switch-over.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 shadow log line, counters, GET /api/lootrules/shadow, bot.lootShadow knob
- [x] #2 zero Starter pack disagreements on fixtures, ground equivalent test, latency budget test
- [x] #3 Shadow findings section in docs/lootrules.md
<!-- AC:END -->
