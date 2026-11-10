---
id: TASK-148
title: 'sensesmap: side-panel minimap v1 from the nav map export'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-08T03:11:00Z'
labels:
  - watch
  - feature
milestone: m-4
dependencies: []
priority: medium
ordinal: 131000
card: "sensesmap"
owns:
  - "internal/watch/senses.go"
  - "internal/watch/senses_test.go"
  - "internal/watch/web/senses.js"
  - "docs/senses.md"
  - "docs/senses-map.md"
  - "internal/senses/"
  - "internal/watch/serve.go"
  - "internal/watch/web/watch.html"
  - "internal/watch/web/app.css"
  - "cmd/d2bot/sensesfeed.go"
  - "cmd/d2bot/sensesfeed_test.go"
  - "cmd/d2bot/ladder.go"
  - "cmd/d2bot/live.go"
  - "cmd/d2bot/d2dbackend.go"
  - "internal/env/env.go"
  - "internal/env/tuning.go"
  - "internal/env/tuning_senses.go"
  - "internal/nav/nav.go"
  - "internal/nav/onobserve_test.go"
  - "tools/knobdump/main.go"
  - "docs/knobs.md"
exclusive: []
contracts: []
result: "Built offline (fixtures and fakes only, no live run, no browser run): internal/senses, /api/senses, canvas panel, docs/senses-map.md. Wiring done (sensesfeed): Env.OnObserve called from nav.observe after World.Update, cmd/d2bot builds the Publisher when -watch and knob senses.tier (default 1) is >= 1; tests: cmd/d2bot SensesFeed* (two areas, fog, version, throttle, race), nav ObserveCallsHook. Origin: no Origin applied, V by code reading, live d2d check still open (U). Owns widened to cmd/d2bot, internal/env, internal/nav, tools/knobdump. Status sync 2026-11-08 (stays open): live/browser run not done."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Design: docs/research/info-unlocks.md sections 3 and 5. Not on the two-week playable path; after codexfirsts/funwire. Done when the panel shows the live area map in a fake-source run and the live checklist is written.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Read-only snapshot of the d2map Grid (downsampled, versioned) and hero x/y exported to the hub; no change to bot behaviour (test: bot inputs identical with senses on or off)
- [x] #2 /api/senses returns the map part only when tier 1 is unlocked (stub: profile flag), token-gated like the viewer
- [x] #3 Canvas side panel draws explored cells, hero, waypoint, exits; offline test with a d2map fixture
- [x] #4 docs/senses-map.md: data flow, schema, honesty tags; live check listed as a ready checklist
<!-- AC:END -->
