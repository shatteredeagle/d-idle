## Why

Coverage gap 10 (`itemtypes`, 98 rows, 0 percent in docs/coverage/rule-coverage.md): the item-type table (Equiv chain, body locations, Beltable, class, socket caps, quiver and throwable columns) feeds the vendor, the pickup, the equip and the hands rules in d2d and in our bot, and only the price and repair gate had been differenced (docs/research/price-trace.md).

## What Changes

- Trace in game.exe 1.14d the type walk (`FUN_00504a20`), the closure matrix (`FUN_00629bb0`), the own-row getters, the equip slot (`FUN_0062ed50`), the two-hands rule (`FUN_0063dbc0`) and the socket cap (`FUN_0062bc20`); run them natively in the emulator over every row pair, item and slot and compare with d2d's real code (`tools/emu/cit_*`).
- Doc: `docs/research/covitemtypes-trace.md` (questions, latent deviations D1-D7, bot findings B1-B6, drill recipe, limits).
- Coverage: one ledger line for `itemtypes` (all 98 rows, TESTED-0-DIFF, 553,647 cases), the gap row dropped, `rule-coverage.md` regenerated.
- Shared-file lines in `merge-lines.txt` (BACKLOG, INDEX, CHANGELOG); no Go, no live game.

## Overlap (declared)

`docs/coverage/ledger.tsv`, `docs/coverage/gaps.tsv` and `docs/coverage/rule-coverage.md` are edited by every coverage trace (one appended line, one dropped gap row, the regenerated table); the merger resolves line conflicts by keeping both sides and re-running `python3 tools/coverage/coverage.py --write`.
