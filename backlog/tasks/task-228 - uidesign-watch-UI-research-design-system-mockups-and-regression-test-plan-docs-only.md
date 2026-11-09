---
id: TASK-228
title: 'uidesign: watch UI research, design system, mockups and regression test plan (docs only)'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-09T18:33:00Z'
updated_date: '2026-11-09T18:59:00Z'
labels:
  - watch
  - research
dependencies: []
priority: high
ordinal: 175000
card: "uidesign"
owns:
  - "docs/research/ui-design.md"
  - "docs/ui-design-system.md"
  - "docs/mockups/ui/"
  - "docs/ui-regression-plan.md"
exclusive: []
contracts: []
result: 'Done (docs only, not run in a browser): docs/research/ui-design.md (V/S/U sourced), docs/ui-design-system.md (tokens, shell, 12 components, panel contract, migration table, open decisions), docs/ui-regression-plan.md (uilint, golden DOM, screenshot suite on the worker host, 13 migration PRs), docs/mockups/ui/ (8 standalone HTML pages, no requests). Root cause of the gear strip: pageCSP style-src self blocks the style element gear.js injects. No production UI code changed; gates: check.sh --hygiene-only, merge_lines_check, leakcheck, backlog check all pass. UNVERIFIED: mockups not opened in a browser (HTML balance checked by parser only).'
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

The watch UI grew panel by panel from many agents and is messy (overflowing header, gear strip text running together, fixed widths). Research modern dashboard, idle and ARPG UI practice and propose a design system with a panel contract so features can be added without breaking existing elements. Done when the four docs and the mockups are merged and the migration cards are proposed in the report.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 docs/research/ui-design.md covers dashboards, idle/ARPG game UIs, design systems, CSS architecture, accessibility, responsive, theming, motion; sources tagged V/S/U with URLs
- [x] #2 docs/ui-design-system.md: principles, tokens, shell grid and breakpoints, 12 components, panel contract (manifest), information hierarchy with wireframes, naming and file layout, anti-regression rules, migration table, locked-state and overflow treatment
- [x] #3 docs/mockups/ui/: standalone HTML (inline CSS, no external requests) for /watch wide and narrow, inventory, policy, return, built on the proposed tokens
- [x] #4 docs/ui-regression-plan.md: headless screenshot tests per breakpoint, Go panel-contract linter, golden DOM check, migration as ordered small PRs with acceptance criteria and card sizes
- [x] #5 tools/ci/check.sh, merge_lines_check --changed and leakcheck pass; no production UI code changed
<!-- AC:END -->
