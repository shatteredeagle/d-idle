## Why

Card d2dtown (archived 2026-10-30) added internal/d2dtown but merged with skip_specs; docs/spec-skip-audit.md judged it needs-spec (effort M). The town seams (vendor stock, owners, stats, menu lines, hover-before-click, interact routing) are what town code relies on over d2d and belong in backend-contract; town-basics and early-game-policy are about shopping decisions, not the adapter.

## What Changes

- Delta ADDED to backend-contract: town Source seams, vendor stock units and owners, synthesised stats, menu line seams, object classes, hover before click, interact routing
- ClaimRogue (internal/d2dtown/claim.go) has no offline test and is not wired into the bot: left out, noted in the report
