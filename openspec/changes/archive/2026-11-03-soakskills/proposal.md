## Why

docs/research/sorc-act-gate.md section 2 requires every Gate 2/3 soak report to carry per-skill fields so a reader can audit the skills afterwards. Whether `d2dsoak report` prints each field is unknown (U). This card finds out and adds the missing ones.

## What Changes

- Audit: which per-skill fields of section 2 (casts started, casts that made a missile or effect, casts cut before the action frame, mana spent, kills per skill and casts per kill, hits per cast, Frost Nova novas and monsters per nova, armor casts/recasts/absent seconds, staff swings, d2d branch+commit, Lightning cast length 13 or 19 ticks) are already in `d2dsoak report` / `d2dsoakstat`, and which the soak log can supply.
- Add the missing fields that the existing logs can supply; for the rest print `n/a (not logged)` and list them in docs/soakskills.md as log-side follow-ups.
- Skills are resolved by name from the data, never by a numeric literal in Go (AGENTS.md build flexibility).

## Impact

Offline only (fixtures and recorded logs), not verified live. No spec change.
