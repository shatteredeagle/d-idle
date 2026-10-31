# Belt fill (BELTFILL)

Problem (V, RECOVERYFIX parsed Soaky's .d2s): the 40-cell pack held about 12 healing, 9 mana and 2 stamina potions while
the belt read 0 healing, so the bot kept buying potions and retreating "belt thin". Nothing moved pack potions to the
belt (S, grep).

## What was added
- `town.Town.FillBelt(ctx)` (internal/town/beltfill.go): opens the backpack, shift-left-clicks each pack potion the plan
  still has room for (per kind: plan columns x rows minus belt count, best tier first, at most the free slots), waits for
  the item to read `ContBelt` (V: belt.Read already uses that), retries `Town.MoveTries` times, then falls back to
  pick up + drop on the equipment belt slot (`Layout.Slots["belt"]`). Stops after 2 failed potions. Log line:
  `belt: moved N potions from pack (hp a->b mp c->d)` with attrs rv, dragged, failed.
- `Town.NeedsBeltFill(snap)`: pure check used by the chore step and the bot.
- Chore step `belt` runs before `shop` when `ChorePlan.Potions` is set.
- `buyPotions` / `needPotions` use `potionsWanted`: the plan shortfall minus the potions still in the pack. The
  "shop: potion plan" log gains packHP / packMP.
- Bot (`internal/bot/resource.go`): optional `BeltFiller` interface (adapter `cmd/d2bot/beltfill_adapter.go`);
  `beltFillStep` runs at the top of the goal step, in town and in the field, when no hostile is within ClearRadius,
  with a 90 s backoff. Log: `belt fill moved=N inTown=.. err=..`.
- `Hand.ShiftClick`; the input layer already supported `input.ModShift` on left clicks (V: internal/input client_test).
  No d2input helper change, so no helper redeploy. U: the helper's `click l x y ... shift` path is exercised on
  ground clicks (inputcal selftest), not yet on inventory items.
- Sim: shift-click on a pack potion goes to the belt; `World.ShiftFails` / `DragFails` simulate failures.

## Tests (offline)
`labrun worker -- go test -race -short -count=1 -p 1 ./internal/town/... ./internal/bot/...`
(TestBeltFilledFromPackNoPurchase, TestFillBeltMovesAndReports, TestFullBeltMovesNothing,
TestShiftClickRetryThenDragFallback, TestBeltFillGivesUp, TestShopCountsPackPotions, bot TestBeltFillStep.)

## LIVE CHECKLIST (d2-live agent, one instance, e.g. i3 Towni; no Battle.net; cheats only through internal/cheat)
Unverified (U) until done: (1) shift+left-click on a pack potion sends it to the belt in 1.14d; (2) which belt slot it
lands in; (3) the drag fallback target (potion dropped on the belt equipment slot) works or does nothing.
1. Character in town, belt with a free slot. Put 4+ healing potions in the pack (buy with the shop, or a cheat on the
   throwaway char with a reason in the ledger). Snapshot: belt hp=N0, pack hp=M0 (`d2snap` or the town log).
2. `tools/live.sh <you> townlive` with a chore run `Potions:true` (or call `FillBelt` from a tiny cmd). Expect:
   `belt: moved K potions from pack (hp N0->N1 ...)` with K>0, failed=0, dragged=0, and no `shop: bought` for healing
   potions that the pack covered; `shop: potion plan` shows packHP>0 before the move only if the fill failed.
3. If dragged>0: shift-click does nothing in 1.14d (record it here). Expect `dragged=K` and the belt still grows. If
   the drag also fails: `belt: move failed` warnings, cursor empty, potion back in the pack, failed=2.
4. Belt full: run again, expect no log line and no clicks (NeedsBeltFill false).
5. Field: walk to the Blood Moor, no hostile near, put potions in the pack, expect `belt fill moved=K inTown=false`
   within a tick, panel closed afterwards (UI flags clear).
6. Leave the character in town, no panels open.
