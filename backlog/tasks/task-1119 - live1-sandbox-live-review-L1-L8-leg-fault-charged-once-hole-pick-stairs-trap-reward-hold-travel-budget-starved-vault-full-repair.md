---
id: TASK-1119
title: 'live1: sandbox live review L1-L8 (leg fault charged once, hole pick, stairs trap, reward hold, travel budget, starved, vault full, repair)'
status: Proposed
assignee:
  - live-agent
created_date: '2026-11-28T06:04:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: high
ordinal: 534000
card: "live1"
lane: next
owns:
  - "internal/bot/"
  - "internal/survival/"
  - "internal/questboard/"
  - "internal/arbiter/"
  - "cmd/d2bot/survivalwire.go"
  - "docs/live1.md"
exclusive: []
contracts: []
result: "Offline only, NOT verified live. L1: travelTo names the failed leg; the second failure of one leg skips no target and portals to town. L2: survival marks a hole with no margin (Hold); the bot refuses a hole walk with Hold or while a plan zone is skipped for a bot fault, takes the act first field if offered, else picks nothing. L3: an identical no-path refusal counts once per minute on the board and quest travel portals out after the same exit fails twice; the nav gate itself is unchanged. L4: a trip is held for a reward only when the board runs or awaits a chunk, a campaign goal runs or the legacy claim list has it; one log line per hold. L5: a finished upkeep trip to town clears the travel budget. L6: no arbiter-starved hit for the leveling filler while the quest step is the training slice. L7 and L8 are owner designs (full vault never removes or picks, repair mode), classified in docs/live1.md. Nothing build-specific; no knob."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Why: the live review of both sandbox heroes after the fix7 deploy (docs/design/fable-review-2026-11-28/13-sandbox-live-review.md) found L1 to L8. Done when: each has a must-fail test and a fix, or a classification with its source, in docs/live1.md.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 L1 a failed leg is charged once, the repeat leaves by town portal (test, two classes)
- [x] #2 L2 a hole is not walked to with no margin or while fault bans made it (tests, two classes)
- [x] #3 L3 identical no-path retries inside a minute are one try and the quest travel portals out before the retry (tests)
- [ ] #4 L3 nav: a different approach cell or an exempt start cell at the exit-trigger gate (needs the recorded Tower Cellar 3 grid, docs/probes/cellar3-gate-capture.md)
- [x] #5 L4 the reward hold needs somebody who walks the reward home, the hold logs once (test, two classes)
- [ ] #6 L4 a vendor visit must not pay an owner-only quest reward (owner decision needed, docs/live1.md)
- [x] #7 L5 travel budget accepts the quest step after a finished town trip (test)
- [x] #8 L6 arbiter-starved silent on a training step (test)
- [x] #9 L7 vault full: classified against docs/vault.md (owner decision 2026-11-21), no change
- [x] #10 L8 repair at 0 percent: classified against docs/repair.md (repair mode), no change
- [ ] #11 live check of L1 to L6 on a sandbox hero
<!-- AC:END -->
