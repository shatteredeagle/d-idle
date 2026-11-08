---
id: TASK-135
title: 'legacyhall: Hall of Fame and title, optional retire after Andariel (ledger and UI only)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T00:27:00Z'
labels:
  - watch
  - feature
  - ladder
milestone: m-4
dependencies:
  - TASK-127
priority: low
ordinal: 127000
card: "legacyhall"
owns:
  - "internal/watch/hall.go"
  - "internal/watch/hall_test.go"
  - "docs/hall-of-fame.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: ADR 0018 item 7, fun-ideas.md section 13 card 8. No charm, no stat grant. Kill: the user does not want to retire a hero that just killed Andariel. Done when: the criteria below hold.
Note 2026-11-07: the legacy charm tiers it points at (TASK-136) are per-class data; see the amendment there.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Optional retire flow after Andariel; a retired or dead hero enters the Hall (one line each); account-wide codex and tier shown as a title; the same hero may continue
- [ ] #2 docs/hall-of-fame.md says built vs verified live; go vet and go test -p 1 of internal/watch; leakcheck 0
<!-- AC:END -->
