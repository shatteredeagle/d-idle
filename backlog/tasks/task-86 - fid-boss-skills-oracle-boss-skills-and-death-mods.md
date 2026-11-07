---
id: TASK-86
title: 'fid-boss-skills: oracle: boss skills and death mods'
status: Proposed
assignee: []
created_date: '2026-11-07T18:03:00Z'
labels:
  - fidelity
  - oracle
  - d2d-monsters
  - d2d-skills
dependencies: []
priority: high
ordinal: 84000
card: "fid-boss-skills"
owns:
  - "docs/probes/fid/fid-boss-skills.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Fidelity matrix rows: MA13. Method/kind: oracle. Live game (host time): yes for the last step only (Andariel read of immunities through the probe). Size: L. Fix: upstream PR where skills or death triggers differ; needs the proof standard docs (differential vs original code) before any upstream PR. Dependencies: fid-boss-mods. Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Trace Andariel poison cloud, Blood Raven skeleton raise, Countess and Corpsefire skills and death triggers (monstats/skills, boss AI) in docs/probes/fid/fid-boss-skills.md
- [ ] #2 Differential for each; tolerance: 0 differing cases in the oracle run; a deviation becomes a docs/d2d-deviations.md entry or an upstream PR
- [ ] #3 Boss base stats and resists through the same-save probe (scenario at the boss level); tolerance from internal/fidprobe/rules.json (exact; life/mana/stamina relative 0.5 %), explained differences only via known.json
<!-- AC:END -->
