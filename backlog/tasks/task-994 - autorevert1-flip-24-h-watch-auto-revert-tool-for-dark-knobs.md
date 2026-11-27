---
id: TASK-994
title: 'autorevert1: flip, 24 h watch, auto-revert tool for dark knobs'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-26T04:15:00Z'
updated_date: '2026-11-27T02:03:00Z'
labels:
  - ops
  - feature
dependencies:
  - TASK-955
priority: high
ordinal: 479000
card: "autorevert1"
owns:
  - "docs/design/autorevert.md"
  - "tools/flip/*"
  - "tools/darklaunch/*"
  - "data/darklaunch.json"
  - "docs/process/dark-launch.md"
  - "docs/invariants.md"
  - "docs/operator.md"
  - "internal/bot/invariants.go"
  - "internal/bot/invariants_test.go"
  - "internal/bot/explain.go"
exclusive: []
contracts: []
result: "Built (v2 fixes the envelope: data.overrides, live knob verify, dry-run, restart mode), not verified live: bot counters (hard flag, since) on /api/explain and ops status --why; tools/flip/flip.py flip and watch with fake-sandbox tests. The AC 2 drill on a real sandbox is still open (see docs/process/dark-launch.md How to flip)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: user decision 2026-11-26: 7-day soaks are not workable and get forgotten; the standard rule is flip, watch 24 h, auto-revert on a hard invariant breach (docs/process/dark-launch.md). Needs a tool, not a person. Done when: tool and counters exist, one knob flipped and reverted in a drill. DESIGN (no build here): (1) Ops side: verb 'flipknob <sandbox> <knob>=<value> --watch 24h' snapshots the sandbox tuning file and the child's current knob values, writes the new value, restarts the child, records a ledger event (knob, old, new, start, window). (2) Bot side: the invariant monitor (TASK-955) exposes HARD counters (death, stall, potion-not-drunk, refusal-as-failure storm) as 'inv=' fields in the minute line and in ops status. (3) A watcher in the ops repo reads the minute line each minute during the window; on any hard counter above its baseline since the flip it restores the tuning snapshot, restarts the child and appends a revert event with the breach to the ledger and the board. (4) No breach after 24 h: the watcher closes the window and the coordinator sets the row default-on in data/darklaunch.json. (5) Never in a measured run (D2BOT_RUN set refuses); sandbox containers only; at most one flip window per sandbox. (6) Needs from other cards: invmon hard counters in the minute line; ops tooling lives in the private ops repo, only the counter surface and the doc are in this repo. (7) Risks: restart cost, flip during a death spiral; baseline is taken at flip time. Depends on invmon (TASK-955).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Design doc and the inv= hard-counter surface agreed with TASK-955
- [ ] #2 Drill: a knob flipped and auto-reverted on a forced breach on a sandbox
<!-- AC:END -->
