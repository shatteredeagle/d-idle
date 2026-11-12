# Auto pack and stash tidying (TASK-381)

User go 2026-11-12. Tags: V verified by the offline test named, S secondary (SME advice, game tables), U unverified (needs the game). NOTHING here ran live; the town sim tests need extracted game tables and SKIP on the worker host too (U, see Tests).

## What it is
A pure policy table (`internal/tidy/policy.json`, loaded by `internal/tidy`) answers, for one backpack item, `keep`, `stash`, `sell`, `drop` or `ignore` plus a reason line ("Will stash: rune"). The town runs it as the last chore, `tidy` (`internal/town/tidy.go`, `Town.Chores`), after the existing sell, shop, cube and stash chores. Those keep their pickit-verdict rules; tidy only handles what they left behind and adds the table's own rows. The same answer feeds the inventory tooltip (additive item field `tidy`, `invdesk.ItemView`).

## Hard rules (in code, before any table row)
- Only a backpack unit is ever acted on (`Place == "pack"`); equipment, belt, cube, stash, cursor and ground answer `ignore`. V `TestNeverTouchesEquipmentOrOtherPlaces`. The Town offers only `ContInventory` units to the plan. Gear policy (docs/gear-policy.md) is untouched.
- A player mark wins: `keep` and `sell` marks answer `ignore` (the sell chore owns marked sales; tidy never moves or drops them). Both are re-checked at action time. V `TestMarksWinOverEveryRow`. The keep mark and its UI already exist (docs/inventory-actions.md, menu "Keep"), nothing was added.
- Quest items, keys, tomes and the start kit answer `keep` (`town.neverSell`, `FlagStartItem`). V `TestStartKitAndQuestNeverSoldOrDropped`.
- Unknown base item: never touched.

## Policy table (SME consult 2026-11-12, EXP; first matching row wins; `overrides.<Class>` rows come first)
| class of item | action | condition |
|---|---|---|
| tomes, scrolls, keys | keep | |
| potions | keep | belt and shop own the count |
| runes, gems, skulls | stash | never sold |
| charms, jewels | stash | |
| unique, set | stash | |
| rare/crafted ring, amulet, circlet | stash | |
| magic ring, amulet | stash | identify from the menu |
| white/superior base, 4+ sockets | stash | |
| low/normal/superior item worth < 20 gold | drop | only when the pack is >= 85 percent full; code floor `dropSafe`: magic, rare, set, unique, crafted are never dropped (V `TestDropNeverTouchesMagicOrBetterOrUnidentified`, table over qualities, identified state and types) |
| anything worth >= 20 gold | sell | the sell chore runs it |
| Amazon override: javelins | keep | throwing build; other classes sell them (V `TestPerClassOverride`) |
Stash action is withheld (item stays in the pack) once the stash is 90 percent full (`stashMaxPct`); the existing `relieveStash` owns that case. Unidentified magic weapons and armor sell as is (no identify scrolls spent: SME). The sell floor 20 equals `loot.poorMinSell` (hourfix); it lives in the table, not in a knob, because per-class rows may differ.
Item type codes (`rune`, `gem*`, `ring`, `amul`, `circ`, `jewl`, `scha`/`mcha`/`lcha`, `jave`/`ajav`) are S from the repo's own pickit and tables, U against the live `ItemBase.Type` strings.

## When a pass runs
`Chores` plan with `Stash` or `Sell` set (every ordinary town visit), the knob `town.tidyEnabled` 1, and `needTidy`: something to stash, or drops planned while the pack is at least `town.tidyMinPackPct` (60) full. No dedicated tidy trip (the chore only runs inside a visit that is already happening): SME says never spend a portal just to tidy; the existing triggers (`town.tripPackUsedPct` 80, sell items, potions) bring the hero home. A stash visit by the player's own request is unchanged.

## Safety valve
`town.tidyMaxActions` (12) stash plus drop actions per pass, stash first (best sell value first), drops last (cheapest first). Hard cap 120 s per pass (`tidyPassCap`; reaching it ends the pass normally, logged `tidy: pass cap reached`). Abort on a nearby fight (`fightNear`) or life under half, on a cancelled context, and after two failed drops (a drop whose item stayed in the pack). A held cursor is never left: `DropCursor` puts a failed drop back into the pack. Player commands keep priority: the chore runs inside the town visit, which the guard interrupts like any chore (priority ladder in docs/inventory-actions.md, unchanged) and the player's queue runs at the next safe moment.

## Cost (S, SME)
Target one pass 60 to 90 s, hard cap 120 s; the chore budget `town.choreBudgetS` (150) still bounds the whole visit. Stashing uses the stash visit's open panel (one open per pass); a drop costs one pick-up plus one ground click. The hourfix target (town 5 to 8 percent of the hour) is not measured here (U).

## Telemetry
`tidy: plan` (stash, drop, sell_left_to_vendor, pack_pct, stash_pct), `tidy: stashed` / `tidy: dropped` per item with the why, `tidy: drop failed`, `tidy: done` (stashed, dropped, refused, stopped). `ChoreReport.Tidied` counts the actions.

## Knobs
`town.tidyEnabled` (1, conservative: stash by table, drop only from a nearly full pack), `town.tidyMaxActions` (12), `town.tidyMinPackPct` (60). docs/knobs.md.

## Build and class flexibility
The table is data; no class or skill literal is in Go. Per-class rows: `overrides.<Class>` (key = `d2.Class.String()`); an unknown class simply has no override rows (common rows apply; this is a table lookup, not a Sorceress fallback). A per-build override is not built (U); the class key is the extension point.

## Wiring (V)
`cmd/d2bot/invwire.go` `wireInv` calls `desk.SetTidyFor(tw.TidyWhyFor)`; both stacks (Wine path `behaviors.go`, d2d `d2dwire.go`) call `wireInv`. V `cmd/d2bot` `TestTidyHookIsWiredIntoTheDesk` (real `buildD2DStack`). The chore is a step of `Town.Chores` itself, so every caller reaches it. The `tidy` item field is additive; `ViewSchema` stays 6 (no golden changed; a bump is the invdesk owner's call).

## Tests
V: `internal/tidy` (table, equipment, marks, start kit and quest, two-class override, stash-full valve, plan cap and order, bad policies refused); `internal/invdesk` `TestTidyForFillsBackpackItemsOnly` (Paladin and Sorceress); `cmd/d2bot` `TestTidyHookIsWiredIntoTheDesk`. U: `internal/town` `TestTidyStashesByPolicyAndNeverTouchesEquipmentOrMarks` (Sorceress and Paladin) and `TestTidyOffDoesNothing` compile and vet but SKIP without extracted game tables (no tables on this box or the worker host), like the other sim tests of that package. The drop click path reuses `hand.DropCursor` (docs/inventory-actions.md Trash, also U live).

## Live checks (not run)
1. Pack with 3 runes, a chipped gem, a magic ring, 2 white clubs worth < 20, a keep-marked rune: at the next town visit the log shows `tidy: plan stash=5` and the stash holds them, the marked rune stays in the pack, nothing worn changed.
2. Pack at 90 percent with only junk under 20 gold and no sell mark: junk dropped beside the hero, `tidy: dropped` lines, the cursor empty, the pack no longer full; the looter does not pick the drops back up (U: trash ignore list is player-only; check, a follow-up may feed drops to `Desk.Ignored`).
3. Tooltip on the Inventory page shows "Will stash: rune" / "Will sell: vendor junk, value 30" on backpack items only.
4. `town.tidyEnabled` 0: no `tidy:` lines.
5. Fight in the camp during a pass: pass stops (`stopped` count).
6. Stash at 90+ percent: no stash actions, items stay in the pack ("Kept: the stash is full").
7. Time the tidy chore with the log timestamps; target under 90 s, town share of the hour.
Drop-pickup loop (closed): drops are remembered by `tidy.Drops` (id plus print, or the same print within 12 subtiles of the drop spot, 2 h, 64 entries) and `wireInv` feeds them to `Looter.SetIgnore` together with the player's trashes. V `TestDropsIgnoreByIDOrSpotAndExpire`; looter test `TestPickUpLeavesTidyDrops` (Sorceress and Paladin, gold/other items still picked) skips without game tables (U).
