# Currency rows: the Raven's Mark overlay (TASK-63, TASK-60)

> status: offline part done and tested, live acceptance pending (blocker in [probes/currency1.md](../probes/currency1.md)). 2026-11-06. Design: [metacurrency-design.md](metacurrency-design.md); format: [overlay.md](../overlay.md).
> Tags: **V** verified (how is named), **S** secondary (read in the fork source, not run), **U** unverified.

## The rows (`data/overlay/didle-v1/excel/`)

| table | row | what |
|---|---|---|
| Misc | code `mrk`, name and namestr `Raven's Mark` | stackable (`stackable` 1, `minstack` 1, `maxstack` 12, `spawnstack` 1), `spawnable` 0 (never a random drop, only through the class below), `type` `key`, art and sounds of the Skeleton Key base (donor values, see overlay.md) |
| TreasureClassEx | new `DI Mark A` | `Picks` 1, `NoDrop` 2, `Item1` `mrk`, `Prob1` 1: a one in three roll |
| TreasureClassEx | replaced `Blood Raven` (Normal) | only `Item5` `DI Mark A`, `Prob5` 1 added (the row's own `Item1..Item4` stay) |
| CubeMain | appended `DI bank: mark` | enabled, `numinputs` 1, `input 1` `mrk`, `output` `isc,qty=1` |

Checks (V by `go test -run Didle ./internal/overlay` against the user's install on the worker host, 2026-11-06): `mrk` is unused in Misc, Armor and Weapons; the Misc type `key` and the sound and art names are ones the base uses; MonStats `bloodraven` has `Blood Raven` in `TreasureClass1..3`, so every rank of her uses the line; `Blood Raven` has `Item5` free; no earlier enabled one-input recipe takes `mrk`, type `key` or `any`; the bank output `isc` is a base item.

## Source and rate

Source: Blood Raven (MonStats Id `bloodraven`, Normal difficulty only; her `(N)` and `(H)` classes are untouched, the difficulty scale of the design is a later row). Her class reads (V, the loaded table): `Picks` 5, `NoDrop` 19, entries `gld,mul=1280` 14, `Act 1 Equip B` 19, `Act 1 Junk` 5, `Act 1 Good` 3, plus ours 1. Per pick the entry is 1/(19+41+1) = 1/61, then our class gives 1/3, so per kill 1 - (1 - 1/183)^5 = 0.0270, one in 37.0 (V, computed by the test from the loaded rows; a 400000-kill Monte Carlo of the fork's roll agrees within 5 sd). The brief asked for about one in 40. Side effect: every other entry of her class loses 1.6 percent of its weight (sum 60 to 61). That changes vanilla drop odds slightly in d.idle mode only; accepted, noted for the fidelity list ("custom row: not 1:1 by design").

Quantity (S, read in the fork `rules.hpp` stack roll: min + rand(spawnstack - min)): with `spawnstack` 1 every drop is one mark. U until the live run shows the quantity.

## Names

The fork shows a missing string as its key (S: `item_text.hpp` `str()` returns the key when `lookup_string` finds nothing); the Go loader falls back to the row's `name` (S: `data.nameOf(NameKey, Name)`). So `namestr` and `name` are both the plain text `Raven's Mark` and no `.tbl` is needed. U: that d2 shows it so on screen and in the stats panel.

## Bank recipe

The fork honours CubeMain (S: `rules/cube.hpp`, `World::transmute`) and a row with no output "makes nothing" and keeps the inputs, so an output is required. `isc` (one Scroll of Identify) is the stand-in output: the bot already handles it. Costs: a free ID scroll per bank. The count in the recipe is items, not quantity (S: `cube_match` counts a stack as one item), so one transmute banks the whole stack in the Cube. The pack loss is the ledger credit (design 1.4); the Observer that compares pack snapshots is job A's interface: not built here (internal/currency is not on main).

## Bot side

`internal/loot/pickit_starter.json`: rule `currency marks` (`mrk`, keep, tier S, prio 92) so the bot carries the stack and never sells or drops it. Test `internal/loot/currencymark_test.go` (default build, a Sorceress plan and a Barbarian plan, with and without gold). Build-specific: nothing; the rule is data in the pickit.

## Not done

Live acceptance and the bank recipe in the game: [probes/currency1.md](../probes/currency1.md). Normal/Nightmare/Hell scale, boss first-kill grants, chests: later rows of the design.
