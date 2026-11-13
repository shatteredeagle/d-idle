---
id: TASK-395
title: 'forkrebase: rebase the d2d fork onto upstream main (2026-11-13)'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-13T15:29:00Z'
updated_date: '2026-11-13T16:34:00Z'
labels:
  - d2d
  - chore
  - tester
dependencies: []
priority: high
ordinal: 246000
card: "forkrebase"
result: "Fork branch didle/rebase-20261113 dd70428d on upstream 73bf565b (private forge): 27 commits; dropped as upstream: Frozen Armor/freeze resume (#159), warp arrival (#490), Zeal (276f85c0), lab ctrl-click sell; ctest 297/301, the 4 failures identical on pure upstream; oracles of the dropped patches 0 differ; d2dbot:devp5 built, not swapped; friends 0.2.0 candidate = dd70428d (tarball sha256 f211cf8f...); 10-min throwaway runs: no crash, hero moves, but the new d2d halves kills with the same bot (2 seeds): swap held. Verified on the worker host, not live on the sandbox."
owns:
  - "docs/fork-rebase-2026-11-13.md"
  - "docs/probes/test-characters.md"
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
- [x] #1 Inventory of every fork branch: carried, dropped (already upstream), private-only or shippable
- [x] #2 New branch didle/rebase-20261113 on current upstream main with the carried patches, one commit each, pushed to the private forge the same day
- [x] #3 Build and ctest on the worker host: only the known upstream failures; carried patch tests and the upstream oracles of the dropped patches pass
- [x] #4 Headless boot on a throwaway volume and a 10 minute bot run on a throwaway hero (arrivals, bounces, moves, no crash) compared with the iv1 sandbox
- [x] #5 Image d2dbot:devp5 built (not swapped) and the friends 0.2.0 candidate commit named, with source tarballs and sha256
- [x] #6 docs/fork-rebase-2026-11-13.md: carried/dropped/why, ids, test results, rollback, swap commands (not run)
<!-- AC:END -->
