## Why

SKGEARAR (first pass, branch agent/skgearar) ran stats 115, 116, 123 and 124 natively and left stats 179 and 120 read-only and the d2d/bot side open. Board item SKGEARAR asks what each term does and whom it changes.

## What Changes

- docs/research/geararmore.md: stat 179 and stat 120 run under the emulator (0 differ in 100,032 and 100,648 cases), a table census of the gear carrying the six stats, d2d deviations, bot findings, drill recipe.
- tools/emu/gar_run.py, gar_dis.py, gar_census.sh (own prefix).
- Docs only: no Go change, no game, no ledger change.

## Impact

Overlap (declared): the first pass doc docs/research/sk-gearar-trace.md on branch agent/skgearar covers stats 115, 116, 123, 124; this card does not edit it.
