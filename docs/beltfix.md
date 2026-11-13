# beltfix: healing columns, belt repair, pack drink (SURVIVAL bug)

> status: built and tested offline; nothing ran live. Tags: V = verified (how is said), S = secondary, U = unverified. Task card: see `backlog/tasks/` (card `beltfix`).

## Evidence (V, read-only `docker logs` of the play sandbox, 22:17Z-23:40Z of 2026-11-12, previous hero then the hero reset at 23:25Z)
- `no healing potion in the belt` 59 times in the log window read; `minute` lines show `beltBoxes` with six mana potions in boxes 0,1,2,4,5,6 (82 minute lines list boxes 0,1,2 every time, 67 list 4,5,6, box 3 only 42 times) and `packHP` 1-2.
- `shop: potion plan wantHP=0 wantMP=0 wantRV=0 beltHP=0 beltMP=6 packHP=1 slots=4` (every visit) and `loot: potion skip ... beltFree=-3` (130+ lines): the belt read as 4 slots while six boxes were in use, so free = 4 - 6 = -2 (loot said -3). `belt.Want`, `beltRoom` and `beltFillCandidates` all clip by that free count, so the town bought nothing, the fill moved nothing and ground pickups were skipped.
- `belt fill` / `belt:` fill lines: 0 in the window (the fill never ran).

## Root cause (V by the log numbers above; the origin of the 8-box layout is U)
1. `belt.Slots` and `combat.BeltCapacity` trust the equipped belt's table row and fall back to 4; the hero held potions in rows 0 and 1 without a belt the equipment read showed. Where those boxes came from is U (the equipment list in the `inv-tp-dump` shows no belt item). Fix: slots never fewer than the boxes in use.
2. Even with the slots right, nothing stopped mana from taking every column: `PlaceFor` reserved an empty column only for kinds that had no column at all, and let a kind open as many columns as were empty. With the game's rule (potion into the first empty column, belt-trace.md) mana fills 0-2 and healing is left to column 3 (one box).

## Layout rule (data, not code)
Planned columns per kind come from the band (`resource.PlanFor` by `ManaNeed(class)`, `PlanForBuild` for the mana-heavy builds, the player's `resource.beltPlan` over it, docs/belt-plan.md): Paladin/Amazon/Assassin hp,hp,hp,mp; Sorceress/Necromancer/Druid hp,hp,mp,mp; Barbarian hp x4. No new knob: the numbers are that table.
- `belt.PlaceFor`: a potion joins a column of its kind freely; it opens an empty column only if (a) its kind has no column yet, or (b) its kind holds fewer columns than planned AND enough empty columns remain for every other kind's missing planned columns. A kind at its planned column count opens no more. The pickup (`loot`, ground potions the game would belt are left on the ground when refused, since a click cannot be steered), the fill candidates and the shop all use it.
- Shopping buys healing first (`resource.PlanBuys` order healing, mana, rejuvenation, unchanged).
- A plan the player chose on purpose (all mana, no healing column) is never "repaired" (`Surplus` needs a planned kind with room and pack potions).

## Repair (town chore, `internal/town/beltrepair.go`)
In town only, inside `FillBelt` (chore steps `belt` and `beltafter`, and the bot's belt-fill step): when a planned kind (healing first) has room, the pack holds some, and `PlaceFor` finds no box, `belt.Surplus` picks the mana column with the fewest potions among mana columns above the plan, and its potions go belt -> pack, top first, then the fill runs again (at most 4 passes). Only potions of the belt, only mana donates (healing and rejuvenation never leave the belt), never equipment. Log: `belt layout: violated, no column for hp (...) emptying N mp potions` and `belt: moved ... repaired=N`. Executors: the item verb (`Town.mover.Move`, d2d, U for a belt item) first, then the click path (left click the belt box, put the cursor in the pack; U on d2d, the sim models game.exe). No hysteresis problem: `PlaceFor` refuses to refill the freed column with mana, and `Surplus` offers nothing once the kinds are at their planned columns. Player keep marks cover pack items only (docs/inventory-actions.md refuses belt items for marks), so no marked item is touched; player commands keep priority because the repair is a town-chore step.

## Emergency pack drink (`internal/combat/packdrink.go`)
When the belt has no healing and no rejuvenation potion, a hostile is in the fight radius, life is under `combat.drinkRejuvPct` (35) and above the chicken line, and the pack holds a healing (else rejuvenation) potion, `Fighter.Safety` returns DrinkLife and `drink()` opens the inventory (`i`), right-clicks the strongest pack potion, checks it left the pack and closes the panel; log `combat potion: pack drink`, event `Drank a healing potion from the pack`. With a portal available the cast still comes first, but `drinkIfDue` now drinks from the pack before each cast try. A failed drink backs off 3 s. Cost: panel open (`waitUI`) + 100 ms hover + up to 900 ms for the effect, roughly 1.2 s (U). Chicken (leave the game) stays above it. D2 rule "right click on a pack potion drinks it" is S (belt-trace.md Q3 table); the same click opens town portals from the pack on d2d (V live), whether d2d drinks a potion the same way is U: first live check below.

## Tests (offline)
`go test -count=1 -p 1 ./internal/town/belt ./internal/combat` (any box); town and loot need the game tables: `labrun <worker> -- sh -c 'D2_MPQ_DIR=<tables dir> go test -count=1 -p 1 ./internal/town/ ./internal/loot'` (known baseline failures there: PlayerStash*, TestUnidentifiedJewelryWaitsWhileThePackHasRoom, TestTidyStashesByPolicy... fail on main too).
- `belt.TestSurplusMana`, `TestPlaceFor` (Paladin and Sorceress bands, the exact 6-mana state, no flip-back, all-mana plan untouched).
- `town.TestTownMovesSurplusManaOutAndFillsHealing` (Paladin + Sorceress, second visit changes nothing), `TestBeltRepairLeavesPlayerPlanAlone`.
- `loot.TestBeltLayoutKeepsTheLastColumnForHealing` (Paladin + Sorceress).
- `combat.TestPackDrinkWhenBeltHasNoHealing` (6 mana / 0 healing / pack healing / 30 percent life, Amazon + Sorceress), `...NotUsedWhenBeltHasHealing`, `...Lines` (50 percent nothing, 20 percent chicken), `...FailureBacksOff`, `...BeforePortalCast`.
- `cmd/d2bot` wiring row for `BeltFiller` on the real town adapter.

## Live checklist (not run)
1. Hidden instance, throwaway Paladin, belt: three mana potions in columns 0-2, two healing potions in the pack, in town: expect `belt layout: violated`, mana back in the pack, healing in the belt.
2. Field, belt empty of healing, one healing potion in the pack, life under 35 percent in a fight: expect `combat potion: pack drink`; check the potion left the pack and life rose. If the right click does nothing on d2d: `error ... pack potion was not used`, then the verb layer (`item use`) is the next step.
3. Where do 6 potions in 8 boxes come from on a belt-less hero (the root of the slot misread)? Dump the equipment read and `Slots` on the sandbox character.
