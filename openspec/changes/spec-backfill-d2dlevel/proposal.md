## Why

Card d2dlevel (archived 2026-10-30) changed a decision in internal/bot (the Den of Evil floor yield) but merged with skip_specs; docs/spec-skip-audit.md judged it needs-spec (effort S). The floor yield is part of how a fresh character reaches the Den, so it is ADDED to early-game-policy; the existing Den of Evil first requirement is not modified.

## What Changes

- Delta ADDED to early-game-policy: the Den floor does not yield far below the floor; when it does yield (one level below, kit present)
- Nothing in internal/** or cmd/** changes
