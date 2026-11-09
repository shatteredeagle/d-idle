---
id: TASK-230
title: 'uishell1: design tokens, cascade layers, 12 components and the uilint panel-contract linter'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-09T19:12:00Z'
labels:
  - watch
  - feature
  - T0
dependencies: []
priority: high
ordinal: 176000
card: "uishell1"
owns:
  - "internal/watch/web/tokens.css"
  - "internal/watch/web/reset.css"
  - "internal/watch/web/components.css"
  - "internal/watch/web/utilities.css"
  - "internal/watch/web/app.css"
  - "internal/watch/web/watch.html"
  - "internal/watch/web/app.js"
  - "internal/watch/policysheet.go"
  - "internal/watch/returnscreen.go"
  - "internal/watch/testdata/policysheet/"
  - "internal/watch/testdata/returnscreen/"
  - "internal/watch/http_view.go"
  - "internal/watch/uitokens_test.go"
  - "tools/uilint/"
  - "tools/ci/check.sh"
  - "docs/ui-design-system.md"
exclusive: []
contracts: []
result: "Merged (mergeq #473). Tokens (dark default, dense), reset, 12 components, utilities, app.css layered on the tokens, tools/uilint (every rule with a failing fixture, allow-list) in check.sh. Verified: go test ./tools/uilint ./internal/watch, whole-repo vet+test on the worker host, sim viewer rendered in Firefox on a private display (dark and light). Preview deployed to the sandbox container (origin/main build). Not verified live by a person. Build-specific: none (UI only)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

First migration step of docs/ui-regression-plan.md section 5 (PR 2 and PR 3 there, merged into one), user decisions 2026-11-09: dark default, dense, overlay shares the shell. Done when the acceptance criteria hold and the preview is up.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 tokens.css declares the layer order once, dark palette and dense spacing are the default, light and auto are options
- [x] #2 components.css holds the twelve components of the design doc, tokens only
- [x] #3 app.css is layered and maps its palette onto the tokens; old cards keep working
- [x] #4 tools/uilint implements the rules of the regression plan section 2 with a failing fixture per rule and runs in tools/ci/check.sh with an allow-list
- [x] #5 preview deployed to the sandbox container
<!-- AC:END -->
