---
id: TASK-22
title: 'decisions-mirror: ADRs mirrored read-only into backlog/decisions for the board'
status: Deferred
assignee: []
created_date: '2026-11-05T19:25:00Z'
updated_date: '2026-11-06T16:00:00Z'
labels:
  - tooling
  - decisions
dependencies: []
priority: low
ordinal: 22000
card: "decisions-mirror"
owns:
  - "tools/backlog/decisions_mirror.py"
  - "backlog/decisions/"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Decided 2026-11-05: ADRs and docs/decisions.md stay the source of truth (guard files, 547 ADR references, grill workflow). This card builds a one-way generated mirror so the Backlog.md board shows the decision log next to the tasks. Do later; no conversion of the ADRs.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 tools/backlog/decisions_mirror.py generates backlog/decisions/decision-N files from docs/adr/*.md one way (ADR 0016 becomes decision-16; status mapped from the ADR header; body copied; file marked 'generated, do not edit')
- [ ] #2 a --check mode fails CI when the committed mirror differs from the generated one, wired into tools/ci/check.sh like cards_md.py
- [ ] #3 unit tests: status mapping (accepted, amended, superseded), amendment blocks kept, idempotent output, ADR guard rules unchanged
- [ ] #4 the NAS-hosted board's decisions page lists all ADRs after the next mirror sync
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Groom 2026-11-06: drift line D is a false positive: the merge was the card itself (Deferred on purpose); tools/backlog/decisions_mirror.py does not exist. Stays Deferred.
<!-- SECTION:NOTES:END -->
