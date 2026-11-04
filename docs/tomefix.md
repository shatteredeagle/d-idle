# Tome stat 70 false-empty audit (tomefix)

> status: current, offline only, NOT verified live. Tags: V = read in code or run as a test here, S = inference, U = unverified live.

## Result
Every consumer-side path (town, combat, reader) reads the tome by unit id and keeps "not empty" under the field conditions tested (stale or shifted snapshot, ground id collision, cursor, panel). No bot change was needed. The only way a tome in the pack reads 0 is a missing or qty-less record in `Extras.Items` for its id, i.e. a producer-side loss; the collision in `d2dsrc` (docs/tomestat.md) was one cause. Whether another producer-side loss exists in the field stays U until the drill below.

## Audit table (reader of stat 70)
| # | Path | file:line | Reads | Tag |
|---|------|-----------|-------|-----|
| 1 | `combat.PortalQty`: no stat70 or error -> 0; nil stats -> 1 | internal/combat/portal.go:16-27 | `Stats(u.Addr)` | V |
| 2 | `combat.PortalSource`: pack tomes only (`ContInventory`), skips qty 0 | internal/combat/portal.go:35-46 | snapshot arg + #1 | V |
| 3 | `bot.hasPortalSource` -> debounce `tpSeen.look`, `inv-tp-dump` | internal/bot/resource.go:328-343, tpdebounce.go:33 | `r.e.Read.UnitStats` | V |
| 4 | `Fighter.portalQty` / `openPortal` | internal/combat/fighter.go:482 | #1 | V |
| 5 | `town.qty` (tbk/ibk without stat 70 -> 0, else 1) | internal/town/shop.go:123-136 | `unitStats` = statsFn or `Read.UnitStats` | V |
| 6 | `town.scrollCount`, `tomeQty`, `tpTool`, `NeedsTrip` "no town portal scroll" | shop.go:386-409, trip.go:42-52, chores.go:93 | #5, pack only | V |
| 7 | `d2denv.Reader.UnitStats`: `last.Extras.Items[id]`, stat 70 only if `Qty > 0` | internal/d2denv/reader.go:117-149 | LAST snapshot's extras, keyed by id (player id/Addr first) | V |
| 8 | Producer `d2dsrc.dumpItem`: `Extras.Items[id]` with `Qty` | internal/d2dsrc/items.go:141-147 | `dump items` | V |
| 9 | Producer stock path `itemsVerb`: no `Qty` ever, so a tome reads empty | items.go:188 | `items` verb | V (code); U whether it can occur with the dump verb present |
| 10 | Producer when `dump items` fails: empty `Inventory`, no tome at all | items.go:19-32 | error path | V (code), U in the field |
| 11 | Ground items into `Extras.Ground` (not `Items`) | items.go:436 | tomestat fix | V (test `TestGroundIDDoesNotShadowCarriedItem`) |
| 12 | Other readers `cmd/d2bot/d2dwire.go:337,388`, `cmd/d2drill/portal_copy.go:141,192`, `portal_lab.go:97`, `bot/tpdump.go:77` | | #1 | V |

Notes (V code): a tome on the cursor is not a pack source (`Where != ContInventory`, #2 and #6) but its count is still readable; the debounce (120 s virtual, about 4 s wall at 29x, docs/tomestat.md "Left") carries a short excursion. `UnitStats` reads `last`, which is replaced on each successful Snapshot, never mixed: unit id is stable across looks, so a stale snapshot asks the current count.

## Regression tests (all pass)
- `internal/combat/tomefix_test.go` (`go test -p 1 ./internal/combat -run TomeFix`): charges -> source; shifted/stale snapshot; ground unit with the tome's id; cursor (qty kept, not pack source); panel flag; contract that only absent extras / stat error read empty.
- `internal/d2denv/tomefix_test.go` (`go test -p 1 ./internal/d2denv -run TomeFix`): Reader with ground/carried id collision, `last` replaced between looks (reads 12 not 0), cursor tome, qty-less record reads empty (contract).
- `internal/town/tomefix_test.go` (needs gear tables; skips here, ran PASS on the worker host: `labrun orion -- sh -c 'go test -count=1 -p 1 -v ./internal/town -run TomeFix'`): 13 charges raise no trip; moved cell; scroll beside empty tome; no stat list reads empty (contract).

## Proven safe offline vs U
- Safe (V by test): paths 1 to 7 and 11 against stale/shifted snapshot, id collision, cursor, panel state.
- U: paths 9 and 10 (a one-look fallback to the stock verb or a failed dump would read empty), whether the fork's `qty` is ever 0/absent for a loaded tome (S: no), and Game.exe parity. Paths 9/10 would show in the log as `items verb`/`dump items failed` warnings next to a `no70` line.

## Live-check recipe (later drill, U)
1. Setup: character with a Tome of Town Portal of 13+ charges and no loose scrolls, in a field (Stony Field, Burial Grounds) with at least 10 filtered ground items; bot running with the d2d dump verb.
2. Observable: grep the run log for `inv-tp-dump` lines with `tome` and `raw70=no70` or `counted=0`, and `tp-debounce-hold|flip`; at each hit note the neighbouring `dump items failed` / `items verb` warnings; take `d2dsnap` (d2d verbs `dump items`, `ground`) at the first hit and compare `extras.Items[tomeID]` with `extras.Ground`.
3. Pass: over at least 30 virtual minutes in field areas, zero `no70` lines for the tome with a real id, zero `tp-debounce-flip` while the tome has charges, no `no town portal scroll` trip at a tome count above 0.
4. Fail branches: `no70` with `dump items failed` nearby -> path 10; `items verb` -> path 9; neither -> fork `qty` dropout (capture `dump items` raw JSON for that id, then an upstream/fork ticket).

## Proposed shared-file text
BACKLOG: tomefix: offline audit of the false-empty tome verdict (14 regression tests, no bot change needed; paths 9/10 and fork qty stay U), live drill recipe in docs/tomefix.md
