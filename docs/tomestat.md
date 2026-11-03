# Tome stat 70 lost in the field (tomestat)

> status: current. Tags: V = counted in the Gate 2 log or read in code, S = inference from code, U = unverified live.

## Where it is lost
- Log gate2-20261102-0823 on the worker host (V): 2067 `inv-tp-dump` lines; 436 show the tome with a real id and `raw70=no70`, 58 with id 0xffffffff. Dropouts sit in field areas (4 and 17 above all), the following town visit reads the tome again (8, 10, 15, 18, 20). Hold lengths 19 s median, p90 95 s, max 148 s virtual; flips carried 1000 to 3000 looks.
- Path (V, code): bot `dump items` -> `d2dsrc.dumpItem` writes `Extras.Items[id]` with `Qty` -> `d2denv.Reader.UnitStats(id)` builds stat 70 from `Qty` (only when > 0) -> `combat.PortalQty`. Field and town call the same verbs and the same fork code (`devctl_dump.hpp`, `character.items`, `qty` field, V in the fork at the Gate 2 binary commit): no dump verb differs by place.
- Cause (V in code, S that it is the whole story): `d2dsrc.ground()` wrote ground items into the SAME map, `Extras.Items`, keyed by their unit id, after the inventory. In the fork the ground id is `Loot::next_id` and the carried id is `World::next_item_id`; both start at 1 (S: separate counters, V: both declared `int ... = 1` in loot.hpp and world.hpp). A ground item whose id equals the tome's id (9 in the run) replaced the tome's extras with a record without `Qty`, so stat 70 vanished while the item lay on the floor. Towns hold few ground items; filtered loot lies in the fields (the `loot: ... skipped` lines). The id 0xffffffff cases are items not yet numbered by the world (`item_ids()` gives ids at the next tick): several of them share the key -1.
- Not a d2d defect: real unit ids are per unit type, so equal numbers for an item on the floor and one in the pack are expected. No fork patch, no upstream candidate. The `ground` verb prints `#<loot id>` and `dump items` prints the character's id.

## Fix (bot side, offline tests)
- `d2dsrc.Extras.Ground` holds ground items; `Extras.Items` only carried ones. `d2dloot.Enrich` reads `Ground`. Test: `TestGroundIDDoesNotShadowCarriedItem` (internal/d2dsrc/groundid_test.go): ground ids 11 and 9 against the carried scroll stack 11 (qty 20) and wand 9.
- Run: `go test -p 1 ./internal/d2dsrc ./internal/d2dloot ./internal/d2denv ./internal/d2dtown`.

## Left
- Debounce (internal/bot/tpdebounce.go) counts 120 s of `Clock.Now()`, which is virtual time; at 29x that is about 4 s wall. With the collision gone the dropouts should vanish; if a live run still logs `no70`, replace the span by a count of consecutive looks or a game-tick window (U).
- Items with id -1 (just created, not numbered) still share a key; a stack bought that tick reads without quantity for one look (S, short).

## Live check (U)
Run a bot on a character with a Tome of Town Portal with at least 5 charges and at least 10 filtered items lying in a field (Stony Field, Burial Grounds): grep `inv-tp-dump` for `tome ... raw70=no70` (expect none with a real addr) and `tp-debounce-flip` with raw=false while the tome has charges (expect none). Also `d2dsnap` in such a field: `extras.Ground` has the floor items, `extras.Items` the pack.
