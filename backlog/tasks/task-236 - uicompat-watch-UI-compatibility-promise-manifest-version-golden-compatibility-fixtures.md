---
id: TASK-236
title: 'uicompat: watch UI compatibility promise, manifest version, golden compatibility fixtures'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-09T20:58:00Z'
labels:
  - watch
  - feature
  - T0
dependencies:
  - TASK-234
priority: high
ordinal: 179000
card: "uicompat"
owns:
  - "docs/ui-design-system.md"
  - "docs/ui-regression-plan.md"
  - "internal/watch/web/panels/"
  - "internal/watch/panels.go"
  - "internal/watch/layout.go"
  - "internal/watch/compat_test.go"
  - "internal/watch/testdata/compat/"
  - "tools/uilint/"
exclusive: []
contracts: []
result: "Compatibility promise (design doc section 0) with checklist in the panel contract; version field in every panel manifest and uilint rule manifest-version; loaders ignore unknown keys; internal/watch/compat_test.go with golden fixtures (v1 panel manifests, editor mockup export, token names, /api/hero fields). Verified: go test ./internal/watch ./tools/uilint, whole-repo vet+test on the worker host. No UI change; docs and tests."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

User decision 2026-11-09: keep the mockup layout and the editor, do not break compatibility in the future. Done when the acceptance criteria hold.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 docs/ui-design-system.md has the Compatibility promise near the top (contracts, rules, enforcement) and a checklist line in the panel contract; the regression plan points to it
- [x] #2 every panel manifest has a version field and uilint fails one without it
- [x] #3 loaders ignore unknown keys; golden v1 manifests, an editor layout export, token names and /api/hero fields are tested to keep loading
<!-- AC:END -->
