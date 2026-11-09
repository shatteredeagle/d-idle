---
id: TASK-86
title: 'fid-boss-skills: oracle: boss skills and death mods'
status: Deferred
assignee:
  - live-agent
created_date: '2026-11-07T18:03:00Z'
updated_date: '2026-11-07T21:55:00Z'
labels:
  - d2d
  - oracle
  - d2d-monsters
  - d2d-skills
  - fidelity
  - parked
milestone: m-2
dependencies:
  - TASK-85
priority: high
ordinal: 84000
card: "fid-boss-skills"
owns:
  - "docs/probes/fid/fid-boss-skills.md"
exclusive: []
contracts: []
result: "Partial: boss_deaths ok and boss_level only the explained lifeless family (V run); boss skills have no differential (table side only). No AC ticked. Doc docs/probes/fid/fid-boss-skills.md. Not verified live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Fidelity matrix rows: MA13. Method/kind: oracle. Live game (host time): yes for the last step only (Andariel read of immunities through the probe). Size: L. Fix: upstream PR where skills or death triggers differ; needs the proof standard docs (differential vs original code) before any upstream PR. Dependencies: fid-boss-mods. Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.

Done when: Trace Andariel poison cloud, Blood Raven skeleton raise, Countess and Corpsefire skills and death triggers...; Differential for each; tolerance: 0 differing cases in the oracle run; a deviation becomes a...; Boss base stats and resists through the same-save probe (scenario at the boss level); tolerance from... (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Trace Andariel poison cloud, Blood Raven skeleton raise, Countess and Corpsefire skills and death triggers (monstats/skills, boss AI) in docs/probes/fid/fid-boss-skills.md
- [ ] #2 Differential for each; tolerance: 0 differing cases in the oracle run; a deviation becomes a docs/d2d-deviations.md entry or an upstream PR
- [ ] #3 Boss base stats and resists through the same-save probe (scenario at the boss level); tolerance from internal/fidprobe/rules.json (exact; life/mana/stamina relative 0.5 %), explained differences only via known.json
<!-- AC:END -->
