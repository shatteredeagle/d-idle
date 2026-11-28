---
id: TASK-249
title: 'reconnectux: exits and chickens shown on /watch'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-10T16:55:00Z'
labels:
  - watch
  - feature
milestone: m-1
dependencies: []
priority: medium
ordinal: 187000
card: "reconnectux"
owns:
  - "docs/onboarding.md"
  - "internal/watch/onboarding.go"
  - "internal/watch/web/panels/onboarding"
  - "internal/watch/onboarding_test.go"
  - "internal/watch/funwire.go"
  - "internal/watch/testdata/onboarding_dom.js"
  - "internal/watch/testdata/onboarding_dom.txt"
  - "data/onboarding/en.json"
exclusive: []
contracts: []
result: "Banners built 2026-11-11. Chicken count built offline 2026-11-27: GET /api/onboarding gains chickens (AwayClose events with left=chicken and a wall time, last 24 h; chickenCount, TestChickenCountWindow) and the panel shows one info line (singular and plural texts in data/onboarding/en.json; node DOM golden cases three, one and no chickens). Exits that are not chickens still have no counter. Not looked at in a browser, not verified live; whether the fight feed stamps Wall on real close events is U."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: a returning friend must see at once whether the page lost the bot or the hero was away, instead of empty panels. Done when: the banners show in the stub-DOM goldens for each state. The count of game exits and chickens needs a bot-side counter and is left open.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 banner when the page loses the bot ("Reconnecting to your hero..."), when the bot restarted, when the player was away ("Your hero was away for 2h 14m: see the Return screen" with a link to /watch/return) and when the video stream is down (d2video hook, Retry)
- [x] #2 node DOM golden for the banner states; no blank panel from the banner itself
- [x] #3 chickens (fights that ended with the hero leaving the game) shown on /watch as a count of the last 24 h; exits that are not chickens (restarts, fresh games) have no watch-side source and are not counted (see result)
<!-- AC:END -->
