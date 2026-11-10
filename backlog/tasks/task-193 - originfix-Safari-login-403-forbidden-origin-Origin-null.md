---
id: TASK-193
title: 'originfix: Safari login 403 forbidden origin (Origin null)'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-08T16:22:00Z'
updated_date: '2026-11-10T16:54:00Z'
labels:
  - watch
  - bug
milestone: m-1
dependencies: []
priority: high
ordinal: 156000
card: "originfix"
owns:
  - "internal/access/sameorigin.go"
  - "internal/access/sameorigin_test.go"
  - "internal/watch/gate.go"
  - "internal/watch/gate_test.go"
  - "cmd/d2dsetup/server.go"
  - "cmd/d2dsetup/d2dsetup_test.go"
  - "docs/watch-origin.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 SameOrigin helper with table tests; login and setup use it; Referrer-Policy same-origin
- [ ] #2 deployed to the dev container and curl-verified
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Closed in the 2026-11-10 true-up (docs/research/project-review-2026-11-10.md section 3.1): merged, offline acceptance proven; open live checks moved to docs/live-checks.md (TASK-253).
<!-- SECTION:NOTES:END -->
