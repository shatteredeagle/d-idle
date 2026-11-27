# Vault poll cost, 2026-11-27

> status: measured and fixed (branch agent/vaultpoll, TASK-1054). Follows vault-perf-2026-11-23.md ("Left" list). V = measured here with the command named; U = not measured.

Command (V): `go test -p 1 -run xxx -bench 'VaultPoll|VaultPage' -benchtime 20x ./internal/watch` on the worker host (Ryzen 7 5800X3D). The benchmark drives the real `/api/vault` handler with a synthetic vault (every 5th record unidentified), the stub actor and the mass-reveal filter. Poll is `?limit=1` (the page sends it every 6 s), page is `?limit=200&facets=1`.

## Cause (V, profile)
A poll was O(records): `view` ran `Query` for the unidentified records, `MassReveal`, the Reveal bar and the unrevealed count over all of them, then `Query` with `limit=1` collected and stable-sorted EVERY live record to return one. The real bot adds, per poll, three more full `Dump()` copies of the vault (BackfillProps, SellAllCount, the sell offer), not in the benchmark (S, code read; not timed here, so the real saving is larger than the table).

## Before / after (V)
| records | poll before | poll after | poll alloc before / after | page before | page after |
|---|---|---|---|---|---|
| 400 | 0.19 ms | 0.09 ms | 109 KB / 43 KB | 0.62 ms | 0.49 ms |
| 2,400 | 0.82 ms | 0.14 ms | 399 KB / 55 KB | 1.22 ms | 0.54 ms |
| 5,000 | 1.24 ms | 0.25 ms | 778 KB / 71 KB | 1.81 ms | 0.59 ms |
| 20,000 | 5.78 ms | 0.54 ms | 3.2 MB / 163 KB | 6.63 ms | 1.01 ms |

What is left per poll is one pass over the records to count the matches (no allocation proportional to the vault) and the three small status reads.

## Changes
- `vault.Store.Version()` (bumped in `apply`, so every write changes it) and `Store.Scan` (visit records under the lock, no copy).
- `internal/watch/vaultcache.go`: the mass preview, the Reveal bar, the unrevealed count and the type facets are cached per bound desk, keyed by store version plus the price inputs (Free, charges, Ether, scroll gold, gold info, order state). A price change with no store write recomputes. The Bind call creates the cache; a desk built without Bind is uncached.
- `Store.Query`: a page (`Limit > 0`) of the default, "new" or "old" order reads the source from the matching end and keeps only offset+limit rows, counting the rest; other orders and unpaged queries are as before.
- vaultkeep `BackfillProps`, `SellAllCount` and the sell offer use `Scan` instead of `Dump`.

## Guards
`TestVaultPollCacheFollowsWrites` (cached answer equals an uncached desk's after pin, discard, sell mark and a price change), `TestVaultQueryWindowMatchesFullSort` (window equals full sort for every offset, 3 filters, 3 orders), existing `TestVaultPollCost`, `TestVaultWireWeight`.

## Not done / U
First load already asks for 60 rows with `facets=1` and the detail card is per item (no change needed); the browser waterfall is unmeasured (U). The real Keeper's poll was not benchmarked (needs the game-side stubs); the three `Dump()` removals are by code reading.
