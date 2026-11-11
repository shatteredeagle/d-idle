---
id: TASK-249
title: 'reconnectux: exits and chickens shown on /watch'
status: Proposed
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
exclusive: []
contracts: []
result: "Reconnect/restart/away/stream banners BUILT offline 2026-11-11 in the onboarding panel (docs/onboarding.md). Exit and chicken counts not built: no counter exists on the watch side. Not verified live."
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
- [ ] #3 game exits and chickens (about 67 exits a day) as a /watch count: not in this card's change, needs an exit counter from the bot (see result)
<!-- AC:END -->
