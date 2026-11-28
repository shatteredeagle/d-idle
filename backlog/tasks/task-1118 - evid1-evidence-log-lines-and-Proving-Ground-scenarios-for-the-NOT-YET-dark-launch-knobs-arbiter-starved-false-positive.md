---
id: TASK-1118
title: 'evid1: evidence log lines and Proving Ground scenarios for the NOT YET dark-launch knobs; arbiter-starved false positive'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-28T06:03:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 534000
card: "evid1"
lane: next
owns:
  - "internal/combat/"
  - "internal/bot/"
  - "tools/provingground/scenarios/evidence.json"
  - "docs/probes/evidence-logs-2026-11-28.md"
exclusive: []
contracts: []
result: "Evidence lines (combat evidence rule=..., checkpoint idle) and seven evid.* scenarios added; arbiter-starved no longer fires for the standing leveling filler (32 of 32 sandbox WARNs); zoneModel matrix, tick count proposal in docs/probes/evidence-logs-2026-11-28.md. Nothing build-specific (shared fight and loop code). Not verified live: no scenario has run in the lab."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Follows graduations-2026-11-28 section 3: make the evidence for the NOT YET rows obtainable. Done when the lines, scenarios and the doc are merged.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 evidence line for missileDodge, threatDodge, standOK, portalGuard (allowed) and checkpoint idle, rate-limited, with tests (two classes where a class matters)
- [x] #2 scenarios evid.* validate and assert on the new lines
- [x] #3 zoneModel decision matrix per class written down
- [x] #4 arbiter-starved cause found from the sandbox logs and fixed or classified
- [x] #5 tick count rule proposal in the doc
- [ ] #6 live: the seven evid.* scenarios run in the lab, three seeds each
<!-- AC:END -->
