## Why

Stale in-progress card (soak-minutes-virtual, branch merged as 12c11b94) still owned files that later cards also list, so changecheck reported overlaps on branches such as agent/stashfield. A read-only inventory of worktrees and branches is also wanted before a cleanup go.

## What Changes

- Archive soak-minutes-virtual the way merge.sh does (git mv, archive header, OWNERS rows dropped) and apply its soak-gates spec delta (task 1.5 of that card) by hand.
- docs/research/housekeep-20261103.md: read-only inventory (nothing deleted).

## Impact

docs/OWNERS, openspec/specs/soak-gates/spec.md, one archived card, one doc. No code.
