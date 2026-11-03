## Why

The d2d gate runner samples only d2d RSS, yet the soak-gates spec requires fd, thread and bot-RSS slopes; the Gate 2 and 3 text asks for a primitives invariant that d2dsoak does not compute; planned d2d restarts, log size and inventory fill have no rule. Evidence: section 2 of docs/research/gate3-prep-20261103.md, checked against internal/d2dsoakstat and cmd/d2dsoak. A cheap smoke rung would have caught the Gate 2 failures of 2026-11-02 early.

## What Changes

- Delta spec for soak-gates: MODIFIED Resource slopes, MODIFIED Gate 2 and Gate 3 on d2d, ADDED Gate 0.5 smoke (specs/soak-gates/spec.md).
- docs/research/gate3gaps.md: sampling design, judge design, work packages with tier and hours.
- No code, no existing spec edited.
