---
id: TASK-395
title: 'forkrebase: rebase the d2d fork onto upstream main (2026-11-13)'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-13T15:29:00Z'
labels:
  - d2d
  - chore
  - tester
dependencies: []
priority: high
ordinal: 246000
card: "forkrebase"
owns:
  - "docs/fork-rebase-2026-11-13.md"
exclusive:
  - "fork-branch:didle/rebase-20261113"
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

The fork sits 2129 commits behind upstream main on base 52362d52; user go 2026-11-13 to rebase it. Done when: the new branch is on the private forge, verified on the worker host, devp5 built and the doc written.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Inventory of every fork branch: carried, dropped (already upstream), private-only or shippable
- [ ] #2 New branch didle/rebase-20261113 on current upstream main with the carried patches, one commit each, pushed to the private forge the same day
- [ ] #3 Build and ctest on the worker host: only the known upstream failures; carried patch tests and the upstream oracles of the dropped patches pass
- [ ] #4 Headless boot on a throwaway volume and a 10 minute bot run on a throwaway hero (arrivals, bounces, moves, no crash) compared with the iv1 sandbox
- [ ] #5 Image d2dbot:devp5 built (not swapped) and the friends 0.2.0 candidate commit named, with source tarballs and sha256
- [ ] #6 docs/fork-rebase-2026-11-13.md: carried/dropped/why, ids, test results, rollback, swap commands (not run)
<!-- AC:END -->
