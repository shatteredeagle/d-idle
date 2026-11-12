---
id: TASK-341
title: 'campaign3: journey questions, top band chip, chapter events, knobs, Rogue'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-11T20:57:00Z'
labels:
  - bot
  - feature
dependencies:
  - TASK-338
priority: high
ordinal: 228000
card: "campaign3"
owns:
  - "data/campaign/"
  - "internal/campaign/"
  - "internal/task/claim.go"
  - "internal/task/guided.go"
  - "internal/bot/guided.go"
  - "internal/decide/catalogue.json"
  - "internal/watch/campaign.go"
  - "internal/watch/web/panels/journey/"
  - "internal/watch/web/panels/journey-chip/"
  - "internal/env/tuning_route.go"
  - "docs/knobs.md"
  - "docs/campaign.md"
  - "cmd/d2bot/journeywire.go"
  - "cmd/d2bot/invwire.go"
  - "internal/task/claim_test.go"
  - "internal/task/guided_test.go"
  - "internal/decide/default_rules.json"
  - "internal/decide/build_test.go"
  - "internal/decide/parity_test.go"
  - "internal/watch/cardtext.go"
  - "internal/watch/humanize.go"
  - "internal/watch/cardtext_test.go"
  - "internal/watch/whyline_test.go"
  - "internal/watch/web/shell.json"
  - "internal/watch/campaign_test.go"
  - "internal/watch/testdata/campaign/"
exclusive: []
result: "Live hero: the Rogue was claimed earlier and is dead; revive skipped for gold (price 600 to 1260, gold under 120) and ClaimNeeded sent the hero home to re-claim every half hour (fixed: an employed merc is never re-claimed). Rogue chapter, t1.journey.floor question, journey-chip, chapter done events, Cain unlock, three knobs. Not verified live."
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: stage 3 of the Guided journey (docs/campaign.md, coordinator brief 2026-11-11). Done when: the acceptance criteria pass offline on Paladin and Sorceress.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 floor question through the return-screen queue (t1.journey.floor), answers persist
- [x] #2 journey-chip panel in the top band
- [x] #3 chapter done: Chronicle and feed line, unlock seen event, Cain chapter opens Identify all
- [x] #4 knobs campaign.journeyAskWaitMin, journeyFarmBudgetMin, journeyGoAtFloor in docs/knobs.md
- [x] #5 Rogue: diagnosis and a Claim the Rogue chapter; a dead employed merc is not re-claimed
<!-- AC:END -->
