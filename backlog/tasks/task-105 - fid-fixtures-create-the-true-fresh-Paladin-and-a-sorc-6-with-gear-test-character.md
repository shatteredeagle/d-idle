---
id: TASK-105
title: 'fid-fixtures: create the true fresh Paladin and a sorc-6 with gear test character'
status: Proposed
assignee:
  - live-agent
created_date: '2026-11-07T21:49:00Z'
labels:
  - ops
  - chore
  - fidelity
milestone: m-2
dependencies:
  - TASK-55
priority: medium
ordinal: 103000
card: "fid-fixtures"
owns:
  - "docs/probes/test-characters.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Run 1 and 2 of the same-save probe lack a true fresh Paladin (Pally is clvl 4) and a separate sorc-6 with gear (Freshe is the only clvl 6; scenarios 3 and 4 are the same character); see docs/research/fidelity-probe-run1.md. PROVENANCE RULE (user 2026-11-07, AGENTS.private.md 'Test characters: provenance rule'): agents may create test characters on hidden instances only via OOG tooling, never delete others'; every character is logged with its creation and any memory write; CHEATED/IMPOSSIBLE characters may test mechanics but are NEVER evidence about game behaviour, balance, parity with game.exe or any tuning decision. Prefer plain fresh characters with zero setup writes.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Fresh Paladin (clvl 1) created normally through OOG on a hidden instance, zero setup writes, logged in docs/probes/test-characters.md
- [ ] #2 Sorceress clvl 6 with gear: genuinely played to clvl 6, or a SETUP write used and the character tagged CHEATED
- [ ] #3 Each log row: name, class, how created, writes used, date, CHEATED/IMPOSSIBLE tag
<!-- AC:END -->
