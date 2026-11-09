---
id: TASK-228
title: >-
  uidesign: watch UI research, design system, mockups and regression test plan
  (docs only)
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-09T18:33:00Z'
updated_date: '2026-11-09T18:33:00Z'
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
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

The watch UI grew panel by panel from many agents and is messy (overflowing header, gear strip text running together, fixed widths). Research modern dashboard, idle and ARPG UI practice and propose a design system with a panel contract so features can be added without breaking existing elements. Done when the four docs and the mockups are merged and the migration cards are proposed in the report.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 docs/research/ui-design.md covers dashboards, idle/ARPG game UIs, design systems, CSS architecture, accessibility, responsive, theming, motion; sources tagged V/S/U with URLs
- [ ] #2 docs/ui-design-system.md: principles, tokens, shell grid and breakpoints, 12 components, panel contract (manifest), information hierarchy with wireframes, naming and file layout, anti-regression rules, migration table, locked-state and overflow treatment
- [ ] #3 docs/mockups/ui/: standalone HTML (inline CSS, no external requests) for /watch wide and narrow, inventory, policy, return, built on the proposed tokens
- [ ] #4 docs/ui-regression-plan.md: headless screenshot tests per breakpoint, Go panel-contract linter, golden DOM check, migration as ordered small PRs with acceptance criteria and card sizes
- [ ] #5 tools/ci/check.sh, merge_lines_check --changed and leakcheck pass; no production UI code changed
<!-- AC:END -->
