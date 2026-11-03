## Why

Each Gate 1 failure cost 30 to 60 minutes of hand analysis (find belt and no-kill windows, read gold, stash, belt and area at those minutes, chase the log text). `tools/baseline/gate_triage.py` prints it on one screen.

## What Changes

- New `tools/baseline/gate_triage.py` (stdlib only) and a fixture test; examples in `docs/research/gate-triage-examples.md`.
- No Go, spec or ops change. The `gate-run.sh --status/--collect` hook is documented in the examples doc only.

## Overlap (declared)

`agent/xphshort` adds `tools/baseline/xph_budget.py` (not on main yet); this tool computes its small XP-by-band and area share itself so it works without it.
