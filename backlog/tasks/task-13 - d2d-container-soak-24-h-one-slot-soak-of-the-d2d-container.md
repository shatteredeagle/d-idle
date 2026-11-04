---
id: TASK-13
title: "d2d-container-soak: 24 h one-slot soak of the d2d container"
status: Proposed
assignee: []
created_date: '2026-11-04T00:00:00Z'
updated_date: '2026-11-04T00:00:00Z'
labels:
  - "parked"
  - "soak"
dependencies: []
priority: low
ordinal: 13000
card: "d2d-container-soak"
owns:
  - "docs/d2d-container-soak.md"
exclusive:
  - "soak-slot:1"
branch: "agent/d2d-container-soak"
agent: "d2d-container-soak"
result: ""
live: ""
merge_hash: ""
overlap: ""
source: "openspec-final:openspec/changes/d2d-container-soak"
contracts: ["proposed/d2d-container-soak"]
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The one-slot 24 h soak of the d2d container (risks step 10) is the gate that, per ADR 0009, ends Game.exe in the dev and product path. Needs the user's go for the NAS template (step 9). It also carries restart and kill injection for the d2d process.

What changes:
- template and first run by a non-agent per docs/container.md (step 9, user go)
- 24 h one-slot soak with restart and kill injection, memory curve (step 10)
- apply the spec delta when this merges

Parked: the container soak is deferred (ADR 0015 item 5)

Gates (from the card):
- 24 h soak report.txt says PASS with exit 0
- memory curve and periodic d2d restart interval recorded

Source: git tag openspec-final, openspec/changes/d2d-container-soak/ (this task is the copy).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 1.1 template and first run by a non-agent per docs/container.md (step 9, user go)
- [ ] #2 1.2 24 h one-slot soak with restart and kill injection, memory curve (step 10)
- [ ] #3 1.3 apply the spec delta when this merges
<!-- AC:END -->
