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
result: "Built and tested offline (go test ./internal/watch ./internal/unlock ./cmd/d2bot; rules table, cooldowns and filters, golden payloads for Sorceress and Paladin, token gates, senders against httptest only); not verified live or in a real browser. Owns widened to funwire.go, web/, data/alerts, ladder row alerts. Outbound senders exist but are OFF by default and nothing reads a profile file into them yet. Low-life has no emitter. See docs/unlock-alerts.md."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Unlock: tally spend on a charter; phone or desktop notifications on a unique, death, level-up, hero back in town. Not on the two-week playable path. Design note: docs/research/unlockables.md (a brainstorm, not a decision). Technical check: opt-in channels only; no hosts or tokens in the repo.
<!-- SECTION:DESCRIPTION:END -->
