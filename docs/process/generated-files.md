# Generated and single-source files (dedupdata, 2026-11-27)

> status: current. Reason: most merge conflicts of the speed review (docs/research/speed-review-2026-11-27.md) were in generated or duplicated files.

## The rule for agents

1. Plan data has ONE source: `internal/plandata/*.json`. `docs/plan/data/*.json` (and `builds-endgame/`) is a mirror that the merge step regenerates (`go run ./tools/plandatasync`, run by `tools/sharedgen/sharedgen.py regen`). Edit the `internal/plandata` file only; do NOT copy it to `docs/plan/data`. Tables that exist only under `docs/plan/data` (danger-table, hotkeys, mercs, pickit-starter, potions-belt, runewords) have no twin and stay hand-edited.
2. `docs/knobs.md` (the block between the `knobdump` markers) is generated at merge (`go run ./tools/knobdump -write`). Do not hand-merge it and do not regenerate it in your branch; add the knob in code. `go test ./tools/knobdump` still checks that every Tuning field has a knob; the stale-table comparison runs only with `KNOBDUMP_STRICT=1` (merge side).
3. A new invariant policy row goes into its own file `cmd/d2scenario/invariants.d/<feature>.json` (same keys as `invariants.json`, loaded and appended by `loadInvariantPolicy`), not into `invariants.json`.
4. Strict mirror check, when you want it: `PLANDATA_STRICT=1 go test ./internal/plandata ./tools/plandatasync` (fails if a mirror differs from the source; fix with `go run ./tools/plandatasync`).

## Why it works

A branch that changes only the source never touches the mirror or the knob block, so git has nothing to conflict on; sharedgen rewrites both in the `shared files (generated)` commit after the merge, byte-stable. If a branch still edited a mirror by hand, the next regeneration overwrites it with the source.
