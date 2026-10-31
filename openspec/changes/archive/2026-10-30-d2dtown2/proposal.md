## Why

D2DWIRE's live runs hit town gaps on the d2d backend: repair unconfirmed, stash unreachable, NPC interact timeouts from a far corner, town portal walk-in, shift-click, waypoint and tome untested.

## What Changes

- find root causes by live probing on the private fork and fix them in internal/d2dtown, internal/d2dctl/town_verbs.go
- fork verb requests (modifiers, object interaction) are written down in docs/d2d-town.md, not patched
