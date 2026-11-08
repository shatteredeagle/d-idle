---
id: TASK-162
title: 'unlockalerts: alerts and pings'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T03:16:00Z'
labels:
  - watch
  - feature
milestone: m-4
dependencies:
  - TASK-153
priority: low
ordinal: 140000
card: "unlockalerts"
owns:
  - "internal/watch/unlockalerts.go"
  - "internal/watch/unlockalerts_test.go"
  - "internal/watch/web/alerts.js"
  - "internal/watch/testdata/alerts-*.json"
  - "data/alerts/"
  - "docs/unlock-alerts.md"
  - "internal/watch/funwire.go"
  - "internal/watch/web/watch.html"
  - "internal/watch/web/app.css"
  - "data/unlocks/ladder.json"
exclusive: []
contracts: []
result: "Offline proof 2026-11-08: go test ./internal/watch ./internal/unlock passes; rules, cooldowns, filters, golden payloads for Sorceress and Paladin, token gates, outbound off by default and never contacts the network (TestAlertRulesFiltersAndCooldowns, TestAlertsGoalEventsTwoClasses, TestAlertsAPIAndTokenGates, TestOutboundDefaultOffNeverContactsTheNetwork, TestShippedLadderHasTheAlertsRow). Senders tested against httptest only; NOT verified live or in a browser. Stays open: AC #6 (browser) and #7 (live) unticked; cmd/d2bot does not read an outbound profile into FunConfig.Alerts, low-life has no emitter; docs/unlock-ladder.md still says no alerts row (stale, not this card's file)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Unlock: tally spend on a charter; phone or desktop notifications on a unique, death, level-up, hero back in town. Not on the two-week playable path. Design note: docs/research/unlockables.md (a brainstorm, not a decision). Technical check: opt-in channels only; no hosts or tokens in the repo.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 ladder row alerts exists in data/unlocks/ladder.json and the alerts rule table loads; bad tables are errors (TestShippedLadderHasTheAlertsRow, TestAlertRulesShippedAndErrors)
- [x] #2 rules fire on unique, death, level-up and hero back in town, with cooldowns and filters; locked keeps only free rules (TestAlertRulesFiltersAndCooldowns, TestAlertsGoalEventsTwoClasses, TestAlertsLockedKeepsOnlyFreeRules, TestAlertsTownNeedsAFieldFirst)
- [x] #3 GET /api/alerts?since=N behind the viewer token gate, paging by seq (TestAlertsAPIAndTokenGates, TestAlertsSincePaging)
- [x] #4 outbound-off proof: zero config sends nothing, NewSender is nil unless enabled, no URL or token in the repo (TestOutboundDefaultOffNeverContactsTheNetwork, TestOutboundConfigErrorsAndConsent)
- [x] #5 outbound senders (webhook signed, no retry, ntfy, no redirect follow) work against httptest servers (TestWebhookSenderSignsAndDoesNotRetry, TestNtfySender, TestSenderInterfaceAndRedirectNotFollowed)
- [ ] #6 alerts toasts and notification centre render in a real browser (not run)
- [ ] #7 LIVE check on a real run: alerts appear for real events (not run; the live panels check is pending)
<!-- AC:END -->
