## Why

Card d2dcombat (archived 2026-10-30) changed code under internal/d2dcombat and internal/d2dctl but merged with skip_specs; docs/spec-skip-audit.md judged it needs-spec (effort M). The d2d combat wiring (belt keys, unit-targeted casts, viewport, meter) is observable behaviour of the d2d backend and belongs in backend-contract.

## What Changes

- Delta ADDED to backend-contract: belt keys drink by column, cast targets the monster at the cell, pseudo hotkeys, viewport anchor, meter totals
- Nothing in internal/** or cmd/** changes; no requirement restates an existing backend-contract requirement
