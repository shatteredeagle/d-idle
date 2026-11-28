---
id: TASK-1116
title: 'ops2: watch token cookie lifetime and 401 hint, Proving Ground smoke in the hygiene gate (review 12 C1, D1)'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-28T05:53:00Z'
labels:
  - watch
  - bug
dependencies: []
priority: high
ordinal: 533000
card: "ops2"
lane: now
result: "Not verified live (offline tests only; phone check open). Watch token cookie now lives 30 days and is renewed by a page load; a page load without credentials gets a hint page instead of a bare 401; ?token= accepted like ?t=; token stripped from the query before the handlers. Proving Ground smoke block moved above the hygiene exit of check.sh (runs in the merge gate, preflight and light CI job when its paths changed), PG_BASE for pushes to main. Nothing build-specific. Autodeploy hold, green rule, flip-window skip and mergeq union fixes are in the ops repo branch agent/ops2."
owns:
  - "internal/watch/serve.go"
  - "internal/watch/serve_test.go"
  - "tools/ci/check.sh"
  - ".github/workflows/ci.yml"
  - "docs/process/proving-ground.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Review docs/design/fable-review-2026-11-28/12-fix-review-security-pg-data-pipeline.md findings C1 and D1. The autodeploy and mergeq parts of ops2 are in the ops repo, branch agent/ops2. Done when the four offline boxes are ticked.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 tokenGate: the d2dview cookie has Max-Age 30 days and a page load with the cookie renews it (go test ./internal/watch -run TestTokenGate)
- [x] #2 a page load without valid credentials answers 401 with a hint page that names the ?t= link and echoes no input; API reads stay a plain 401
- [x] #3 ?token= is accepted like ?t=; the handlers behind the gate never see either in the query; a // path cannot redirect to another site
- [x] #4 tools/ci/check.sh runs the Proving Ground smoke block before the --hygiene-only exit when its paths changed; ci.yml passes PG_BASE on a push
- [ ] #5 live check: open the canary watch link on a phone, bookmark the clean address, reopen after closing the browser
<!-- AC:END -->
