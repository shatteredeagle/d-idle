---
id: TASK-244
title: 'dollart: real D2 art on the inventory paper doll, read at runtime from the user''s MPQs'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-10T15:47:00Z'
labels:
  - watch
  - feature
dependencies: []
priority: medium
ordinal: 182000
card: "dollart"
owns:
  - "internal/art"
  - "docs/paperdoll-art-build.md"
  - "internal/data/invfile.go"
  - "internal/data/invfile_test.go"
  - "internal/data/invfile_real_test.go"
  - "internal/watch/artapi.go"
  - "internal/watch/web/inventory.js"
  - "tools/ci/check.sh"
exclusive: []
contracts: []
result: "All six PRs built: internal/art (DC6 to PNG, cache), data columns plus DB.InvFile (schema 4), status.Item.art, /api/art and /watch/art/<key>.png, inventory page art layer with DOM goldens (art off byte-identical to the old page), CI and embed guards. Verified on the real install offline (decoder, 1977 icon names, panel quadrants, slot geometry, one visual render); NOT verified live: the FileIndex memory offset and unique/set id semantics (wrong ids fall back to the base icon). No PL2 tints, no variants, no art in the stream overlay."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Build docs/research/paperdoll-art.md. Done when all six PRs are merged, art off goldens untouched, no Blizzard art in git.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 PR A internal/art core
- [x] #2 PR B data columns and InvFile resolver
- [x] #3 PR C status.Item.art wiring
- [x] #4 PR D art endpoints
- [x] #5 PR E UI
- [x] #6 PR F guard rails
<!-- AC:END -->
