---
id: TASK-20
title: "watch-screen: Thoughts feed, orders, uptime, rules list and journal on d2dview"
status: Proposed
assignee: []
created_date: '2026-11-05T00:00:00Z'
updated_date: '2026-11-05T00:00:00Z'
labels:
  - "G-G"
dependencies: []
priority: medium
ordinal: 20000
card: "watch-screen"
owns:
  - "cmd/d2dview/**"
  - "internal/decideapi/**"
  - "docs/watch-screen.md"
exclusive: []
branch: "agent/watch-screen"
agent: "watch-screen"
result: ""
live: ""
merge_hash: ""
overlap: "cmd/d2dview is also named by task-15 (stream-grid, Deferred); coordinate the web files before starting."
source: "grill 0005 outcome (docs/grills/0005-automation-ladder/outcome.md), ADR 0016 item 24"
contracts: ["dashboard-pub"]
next: "tier order: after T4/Gate A cards per ADR 0016"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The one player screen of ADR 0016 items 20 to 23 on cmd/d2dview (docs/d2d-view.md): game frame, Thoughts feed (questions with answer buttons and 'not sure', idle dialogue, incidents; replaces the question card), orders box, uptime, rules list with its context in plain language (revoke one or all, a 'why did you do that' link from journal lines), and the journal as the feed's permanent history including missed questions. The viewer stays read-only: answers, confirms, revokes and orders go through a separate bot route (names in spec section 6). It first runs on the simulated runner (U: whether d2dview can take the goals sim is not read; cmd/d2dash already serves the sim). Needs the store API of decision-point-store; build against its interface with a fake. Story and dialogue text must be our own (docs/contracts/licence-policy.md); this card ships placeholder lines only. Order note: tier order: after T4/Gate A cards per ADR 0016 (the four T4 and Gate A cards m3-act1run, m3-andariel, m3-merc, gameexe-retire-soak wait their turn). Ladder work is product work, outside the tooling freeze (ADR 0016 item 3). Tags: V read in repo, U unverified.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The feed shows a question thought with answer buttons, fades it after the fade time (virtual clock in tests) and writes it to the journal as missed
- [ ] #2 Rules list shows each rule with plain-language context; revoke one and revoke all work through the bot route, the viewer endpoints stay read-only (test: no write method on the viewer mux)
- [ ] #3 Orders box and uptime (share of time making progress; deaths, recovery and town stalls lower it) render from the simulated runner
- [ ] #4 Runs on the simulated runner with no game (U: say in the result whether d2dview or d2dash hosts it); not verified live
- [ ] #5 go vet and go test -p 1 of the packages; leakcheck 0; docs/watch-screen.md written
<!-- AC:END -->
