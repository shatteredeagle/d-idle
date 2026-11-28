---
id: TASK-1133
title: 'road5: offline halves of the three open roadmap rows (R13 chain coverage, PG11 gate marker, DLVc literal ratchet)'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-28T10:12:00Z'
labels:
  - ops
  - feature
dependencies: []
priority: medium
ordinal: 537000
card: "road5"
lane: next
owns:
  - "tools/ci/"
  - "cmd/gatemark/"
  - "internal/gatemark/"
  - "internal/questspec/"
  - "data/questspec/"
  - "docs/roadmap-fable.md"
  - "docs/process/pg-ops-mgonejob.md"
  - "docs/questspec.md"
exclusive: []
contracts: []
result: "Offline only, NOT verified live and nothing installed in the ops repo. DLVc: tools/ci/literalcount.py ratchet in check.sh (fails on a rise; class and skill names read from the class table and the capability table, none in the tool). PG11: cmd/gatemark (ran-marker by package count, failure record with tail, flake ledger, quarantine rows); wiring list docs/process/pg-ops-mgonejob.md. R13: data/questspec/pending.json plus TestShippedCoverage; no chain written for Acts 3 to 5 (lab verdicts RED or BLOCKED, a chain is live behaviour). No knob. Build-specific: nothing; the literal names live in internal/d2 classNames and data/capabilities."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Build what the three open rows allow offline; no knob, no live game, no ops repo edit.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 DLVc: class and skill literal ratchet for internal/combat and internal/power in tools/ci/check.sh, must-fail test with two classes
- [x] #2 PG11: gatemark reads go test -json, prints a positive ran-marker, stores failing tests and tail, appends a flake ledger; must-fail tests
- [x] #3 R13: every board quest has a step chain or a pending row with a reason; must-fail test
- [x] #4 roadmap rows R13, PG11, DLVc updated; tools/roadmap/check.py without warnings
- [x] #5 Live or ops-side remainder written precisely (not run by this card)
<!-- AC:END -->
