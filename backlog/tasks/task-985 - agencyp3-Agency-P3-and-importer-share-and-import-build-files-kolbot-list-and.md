---
id: TASK-985
title: 'agencyp3: Agency P3 and importer: share and import build files, kolbot list and'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-26T03:57:00Z'
labels:
  - bot
  - feature
dependencies:
  - TASK-984
priority: low
ordinal: 473000
card: "agencyp3"
owns:
  - "docs/design/agency-p3.md"
  - "internal/buildfile/share.go"
  - "internal/buildfile/share_test.go"
exclusive: []
contracts: []
result: "Done offline, not verified live. ReadShared, ImportKolbotBuild, export/import identity test over all shipped plan tables, Paladin+Sorceress kolbot import tests; NIP importer and /api/hero/track not built."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Recommendation of the 2026-11-19 design review (06-roadmap R15; prototype branch agent/fable-proto-buildfile, didle.build/1). Build files shareable and importable; kolbot-list import; NIP importer. Cause class: none of the A..K treadmill classes (feature work). Risk: low. Enable criteria: round trip of the eight shipped plan tables stays exact.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Import and export round trip test with two builds
<!-- AC:END -->
