# gate2cursor: the cursor fault and the no-area block (Gate 2 seed 13)

Run gate2-s13-20261103-2211-g2v, bot log read offline on the worker host. V = log line or code read, U = unverified.
Follows docs/gate2loop.md.

## A. "item stuck on the cursor"

Evidence (V, log):
- First hit 23:57:55 EST (line 54166). One call earlier (54157) the loot call returned `Strangling Gas Potion was worn on
  pickup and could not be taken off: hand: no free space in target`. The next calls (54166, 54168, ...) fail at once with
  `item stuck on the cursor: hand: no free space in target`; later the same in `Recovery: farming gold` at 02:50, through
  five failsafe ladder steps, a self pause and a game re-entry (the error is back on the first PickUp afterwards).
- The pack was not full by cell count: `inventory` 18/40 cells at 23:57:44, `invPct` 45 at 02:50, gold 26.
- The first failure is not a pack-full pickup: the pickit sells gas potions (sold 6 times earlier, 20 gold each), so
  the pickup itself is routine.

Code path (V, read):
1. `pickOne` clicks the item; the game wears a picked weapon-type item (a gas potion is a throwing "weapon" base) when a
   weapon slot allows it (S: KITSTALL2). `unwear` sees a worn item the class must not wear and calls `Hand.Unequip`.
2. `Unequip` clicks the slot (item to the cursor) and calls `ClearCursor`; `ClearCursor` returns `ErrNoSpace` when
   `FindSpot` finds no cell. The item stays on the cursor. `unwear` returns an error, `PickUp` aborts.
3. Every later `PickUp` starts with `Inventory.Cursor != nil`, calls `ClearCursor`, gets `ErrNoSpace` again and returns
   `item stuck on the cursor`. Nothing ever drops or places the item, so every loot call (including the gold farm of
   recovery) fails at once, gold stays at 26, no shop, the failsafe ladder fires every cycle. A held item is saved with the
   character (U), which is why re-entry does not help.

Not established (U): why `ErrNoSpace` with 18/40 cells used. Candidates: the cursor item is larger than the free block
(a displaced weapon or armor in a fragmented 4-row pack), or the occupancy view and the game disagree. The log has no
cursor item dump. The fix does not depend on the answer.

Fix: `Hand.DropCursor` (internal/town/hand/drop.go: close panels, ground click beside the character, wait for an empty
cursor). `PickUp` uses it when `ClearCursor` fails with `ErrNoSpace` (top of the loop) and when `unwear` cannot place a worn
item (returns `errDropped`, not counted as picked). The dropped id is marked failed for the rest of the call, so it is not
picked straight up again. Other `ClearCursor` users (shop, belt, waypoint) are unchanged: dropping is only safe for loot.
Risk (U live): the ground click with a held item drops it (D2 behaviour, never exercised by this bot), and a dropped
quest item is picked up again on the next call if the occupancy view says there is room.

Tests (offline, need game tables: run on the worker host):
`go test -count=1 -v -run HeldItem ./internal/loot` through labrun on the worker host
- `TestPickUpDropsHeldItemWhenPackIsFull`: 40 scrolls fill the pack, a bow is on the cursor; PickUp returns no error, the
  cursor is empty, the bow is on the ground, not re-picked. Old code returned `item stuck on the cursor`.
- `TestPickUpHeldItemWithRoomGoesToPack`: with room the item still goes to the pack. The sim now drops a cursor item on a
  `ClickWorld` (internal/town/sim/input.go).
The worn-and-no-room branch of `unwear` has no fake test (the sim cannot build a pack that has room for the candidate
check and none for the unequip); it shares `dropHeld`.

## B. "no eligible reachable leveling area" at clvl 15-16

Evidence (V, log; 7683 such lines in the run):
- First block 23:36:00 (line 41596). Just before: Burial Grounds (clvl 15 target), no town portal (tome empty), walk home
  refused by nav (`the click would cross the trigger area of an unwanted exit`), failsafe step 3 re-entered the game. In
  town the state read `Waiting: Burial Grounds banned for 3 more min`; 0.15 s wall later `area ban lifted` (idleBreak) and the
  planner chose `Cave Level 1 (past every window, last eligible (leaves at 10))`. So that first block lasted 2 slices.
- Many bans over the run (`banned after repeated deaths`, `skipped the stalled target area nav-stuck`, route guards:
  monastery needs 2 mana potions, Tower Cellar needs a merc and clvl 16).

Code (V, read): `Plan.Pick` (plan.go) returns no step when the character is past every window by more than `worthFloor` (4)
levels and the banned/cleared/guarded set (`ex`) covers the level (`windowExcluded`). In Normal Act 1 the last window
leaves at about clvl 10-12, so at clvl 15-16 every Act 1 area is "past its window" and any ban makes the picker refuse;
`LevelTo` then waits in town (idleBreak lifts the least bad ban after `AreaIdleSec`) or starts a fresh game when areas were
cleared. This is the designed policy, not a bug: the character is clvl 15-16 in Normal Act 1 with no merc and no kit,
and the way forward (Act 2 / quest progress / Nightmare) is a policy decision. Not changed (policy cards are paused).

What made it permanent after 23:57: A. Gold 26, no tome, so every trip walks home, recovery farming gold never loots, Blood
Moor gets banned by the failsafe skip, and the picker has nothing left. So B (late) is a consequence of A plus the
window policy; B (23:36) is the policy alone, and it self-cleared. Seed 11 reached clvl 16 without A (U: no gas-potion
pickup with a free weapon slot; its log was not read). Fixing A removes the permanent stall, not the slow leveling.

## Live rerun criteria

Soak seed 13, same settings, from a fresh character:
- no `item stuck on the cursor` line; if `loot: no pack cell for the held item, dropped it on the ground` appears, the next
  line must not be a loot error and the cursor must be empty in the following `inventory` line;
- if the drop click does not clear the cursor, `could not drop the held item` appears: then the ground click assumption
  is wrong (U), capture the cursor item (class, size) and the pack layout;
- gold rises in recovery (above the 300 floor), shop visits happen, `no eligible reachable leveling area` lines are
  followed within a few minutes by a level plan line, XP keeps rising past clvl 16.
