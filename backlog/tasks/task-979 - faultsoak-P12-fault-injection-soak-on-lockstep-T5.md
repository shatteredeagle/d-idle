---
id: TASK-979
title: 'faultsoak: P12 fault-injection soak on lockstep (T5)'
status: Done
assignee:
  - live-agent
created_date: '2026-11-26T03:56:00Z'
updated_date: '2026-11-26T16:05:00Z'
labels:
  - bot
  - drill
dependencies:
  - TASK-955
priority: medium
ordinal: 467000
card: "faultsoak"
owns:
  - "docs/design/faultsoak-p12.md"
  - "docs/probes/faultsoak-design-2026-11-26.md"
  - "internal/faultproxy/"
  - "cmd/d2scenario/"
  - "internal/scenario/"
  - "tools/provingground/scenarios/faultsoak.json"
exclusive: []
contracts: []
result: "Runner side built and tested offline: fault proxy (late and dropped replies, delayed input, cancel storm), fault sets as data, soak command with invariant hit and survival report, five soak.fault rows. Not run live; the fork verbs for NPC remove, portal expire and belt clear are proposed in the design note. closecards3 Remaining (live): #2 need the live game, a browser or a worker-host run; not verified here."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Cause classes killed: E, F and the recovery paths nobody walks on purpose. Roadmap row P12 and T5 (docs/research/fable-logic-review-2026-11-25.md 5.3, 7.1). Soak with injections on lockstep: empty the belt, zero the purse, drop life, fail the portal, time out the control socket (14 such timeouts in the quest log), remove an NPC from the unit list; P1 monitors are the oracle. Injections are lab verbs of the fork (ADR 0014 DIAGNOSIS; never in a measured run; recorded in the ledger). Night window ADR 0027 for T3/T4 runs. Risk: low (lab only). Enable criteria: soak runs nightly report-only; each hit becomes an incident1 scenario.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Injection list and the monitor oracle documented (docs/probes/faultsoak-design-2026-11-26.md)
- [ ] #2 First soak run recorded with writes ledger (live check: d2scenario soak -id soak.fault. -duration 1h on the worker host in the night window)
<!-- AC:END -->
