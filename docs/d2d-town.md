# Town layer on the d2d backend (agent D2DTOWN, 2026-10-30)

Tags: **V-d2d** observed on the private fork d2d (commit e36922e + our P1/P2 + the vendor section below) on the worker host; **U** not verified; Game.exe stays the acceptance target.
Code: `internal/d2dctl/town_verbs.go`, `internal/d2dtown` (adapter, `ClaimRogue`), `cmd/d2dshop` (smoke tool), seams in `internal/town` (`town.Source` optional fields, no behaviour change when unset).

## How internal/town reads the game, and what d2d answers

`town.Town` needs (a) `d2.Snapshot` (player, UI flags, monsters incl. NPCs, inventory) = `d2dsrc.Source` already; (b) the item unit table with vendor stock and the container owner (`Source.Items`, `Source.Owner`, existing seams); (c) unit stats for quantity and durability (`Source.Stats`); (d) pixel input: NPC dialog lines, vendor tabs/cells, inventory cells.
`internal/d2dtown.Adapter.Source()` fills (a) to (c) plus two new optional fields (dialog line point from the `menu` verb, fixed vendor tab slots). Use: `t.UseSource(a.Source())` and wrap the input with `d2dtown.HoverFirst(d2dinput.Controller)`.

## Need -> d2d verb map (V-d2d unless noted)

| Town need | d2d verb / how | Notes |
|---|---|---|
| Walk to NPC and open dialog | `cmd interact <npcs index>`, then poll `menu` | the player walks by itself, takes a few seconds; no separate approach needed. `OpenNPCMenu` |
| Read dialog | `menu` (text, x, y per line, header first) | exact click points; the pixel model of `town.entryPoint` is 41 px off on d2d (Akara: model Cancel y 194, real 235) |
| Select a line | `move` twice (3 px away, then on it, ~0.5 s) then `click` | a click without hover is ignored, as on Game.exe. `HoverClick`, `PickMenuLine` |
| Vendor open? | `dump ui` `vendor`, `state store=<vendor> store_tab=<1-based>` | |
| Vendor stock + prices | **no stock verb**; added `dump vendor` to the private fork (branch `d2dtown-vendor`, commit 7abb81d in `~/d2d-lab/fork/src`, local only) | stock items have no unit id (-1): key (tab, x, y). Prices: Akara wand 205, tome 250, TP scroll 100, ID scroll 80, minor healing 30, minor mana 60, key 45 |
| Vendor tabs | click at x = 118 + 81*tab, y 75 | d2d always draws Armor/Weapons/Weapons/Misc (four slots), the page number is the slot. Game.exe's town code compacts to the stocked pages: `town.Source.FixedTabs` |
| Buy | right-click the stock cell | game buys and places in the pack; gold drops by the listed price (V: wand -205, tome -250) |
| Sell | pick up the pack item (`debug itemat <id>` gives its pixel), click the vendor grid | right-click on the pack item did NOT sell (V). Matches `town.sellByCursor` default |
| Repair | Charsi: dialog `trade/repair`, then the right-most of four buttons under the grid at (368,461) | **V-d2d** (2026-10-30 D2DTOWN2): wand 7/15 -> 15/15, gold -13; `Client.RepairAll` confirms by `dump items` durability |
| Belt / potions | `cmd belt` only drinks; pack->belt is shift-click on Game.exe | no click modifiers on the channel: **BREAK**, see below |
| Stash / gold bank | `cmd interact <npcs "Your Private Stash">`, `dump ui` `stash`; pack coin (493,461) = deposit prompt, stash coin (165,92) = withdraw prompt; `text <n>` + `key Return` | **V-d2d**: deposit 5000 (20000/0 -> 15000/5000), withdraw 2000 (-> 17000/3000), read from `dump player` gold/stash_gold. `Client.MoveGold` |
| TP scroll/tome | open the pack (`key i`), `debug itemat <id>`, hover + right-click the item; `cmd skill 220 ... <item>` does nothing | scroll **V-d2d** (portal appears in `debug portals`, scroll consumed); tome **U** (same right-click, not run). `Client.CastTP` |
| Waypoint | `npcs` lists "Waypoint"; `cmd interact` opens the panel (`dump ui` waypoint true) | open **V-d2d**; list rows at x 210, y 138 + 35*row (act 1, from a screenshot); choosing a row **U** (only the town entry was active, no `debug` verb activates a waypoint). `Client.OpenWaypoint`, `ChooseWaypoint` |
| Free Rogue | `d2dtown.ClaimRogue`: opens Kashya's menu, watches `state merc=`, closes with cancel | no hire click needed (V-d2d probes section 1) |

## Integration breaks found (each has a fix or a note)

1. **Dialog placement**: `town.entryPoint` is a Game.exe pixel model; d2d's box is elsewhere (41 px). Fix: `Source.MenuEntry`/`MenuIndex` use `menu` points. Done, tested offline.
2. **No vendor stock verb** in stock or P1 d2d. Fix: fork `dump vendor` (local patch), `d2dctl.Client.Vendor`; stock d2d returns `ErrNoVendorDump` and the adapter then shows an empty stock (the shop code logs it).
3. **Vendor tabs** are four fixed slots (see above). Fix: `FixedTabs`.
4. **Hover before click**: every dialog/vendor click needs the two-step hover. Fix: `HoverFirst` input wrapper (+350 ms per click).
5. **Memory-address seams**: `readOwner` (Mem) and unit stats by address do not exist; the adapter mints addresses (NPC unit id, item id, vendor `0x50000000|tab<<16|y<<8|x`) and answers stats from the dump's qty/dur. Stock quantity (arrows) is not in `Stats` yet: **U**.
6. **Shift-click** (pack->belt, `hand.ShiftClick`) returns `ErrUnsupported` through `d2dinput`. Needs either a modifier verb in the fork or the pick-up/drop-on-belt path (D2D-src `put_in_belt`); **open**.
7. **`debug stat 14` bypasses the purse cap** (clvl*10000): a sale clamped gold 19545 -> 10000 (V). Cheat the level first (the smoke tool does).
8. Escape after the vendor/NPC menu is already closed toggles the game menu (probes section 2): close with the cancel line or one Escape while the window is open.
9. Prices differ from live (tome 250 vs 450): not chased (D2DTOGGLE).

## Smoke result (live, private fork, throwaway Paladin "Townz", --no-save, nice 10; V-d2d)

`d2dshop -sock ~/d2d-lab/fork/d2d.sock -steps rogue,menus,potions,tome,sell,stash` (cheats printed by the tool: clvl 20, gold 20000).
- Kashya: after `debug kill` in Burial Grounds and `goto 1`, opening her menu (the `menus` step of the first run) granted the Rogue (burial record 5, `merc=..:RG`); a later `ClaimRogue` reports `HadMerc:true, Granted:false`, as probes section 1 predicts. So the grant-on-open path is reachable through `OpenNPCMenu`; `ClaimRogue` itself was only run in the already-granted state (its Granted branch is covered by reading the same fields, not by a live run: U).
- Akara: menu `talk|trade|cancel`, trade opens (52 stock items), buy 3x minor healing (30 each, listed 30), buy 3x Tome of Town Portal (250 each, listed 250), sell an Identify scroll by cursor drop (+40), all gold deltas match the dump's prices. Escape closed the vendor window without opening the game menu.
- Stash chest opens (`dump ui` stash true once); the gold button click (140,428) is a guess (U): stash gold unchanged in run 1, run 2 did not get the panel open in time. Gold banking is **not verified**.
- Charsi, Gheed, Warriv, Kashya (second run): `OpenNPCMenu` timed out after 25 s from Akara's corner. `cmd interact` walks only part of a long way and repeating it did not get through (blocked path or walk limit: U). The town layer's `GoToUnit` (nav) must walk next to the NPC first; the shop/dialog code is NPC-independent.
- Full `town.Town` chores were not run (no nav Mover over d2d, no data.DB wiring in the tool); offline the adapter is covered by `internal/d2dtown` tests and the whole repo is green.

## Gaps closed (agent D2DTOWN2, 2026-10-30, lab2 binary, private session, throwaway Necromancer "Townt", --no-save)

1. **NPC unreachable (Charsi, Gheed, Warriv, Akara from a far corner)**: not the verb, the walk. V-d2d: `cmd interact <i>` walks straight toward the NPC and stops for good when it meets an object (Charsi from Akara: stuck at 27.7,12.7 and 34.5,11.1 for 5+ s, repeated interact changes nothing; the first `cmd move` also stopped at 29.35,11.15), while a `cmd move` to a free cell nearby and one more interact reached her (menu `talk|trade/repair|cancel`). Gheed from the stash opened in 7 s with no help. Fix: `Client.ApproachNPC` (interact, poll dialog/panel flags, on a stall of the player position (`debug level`) side-step 3.5 cells toward the NPC and interact again); `OpenNPCMenu`, `OpenStash`, `OpenWaypoint` use it; `d2dtown.Input` routes the town layer's NPC/stash/waypoint interactions through it.
2. **Stash "npc not found"**: `d2dsrc` files "Your Private Stash" as an Object with Class -1 (npcs has only a name); `town.openStash` searches Class 267 (`ObjStash`). Fix in `Adapter.classObjects` (stash -> 267, waypoint -> the act's class). The chest itself opens fine (`dump ui` stash true after ~6 s walk, V-d2d).
3. **Gold deposit/withdraw geometry**: the guessed (140,428) was wrong. V-d2d from screenshots: the coins at (493,461) and (165,92); the prompt "How much gold would you like to deposit?" takes `text <digits>` and `key Return` (the confirm tick at (265,271) is not needed).
4. **Repair**: `town.RepairAll` (369,452) lies inside the button (right-most of the four at x 350..387, y 443..479 in the screenshot), so the geometry is probably not the reason for the run's "not confirmed (durability 101% unchanged)". 101 is the town code's "no worn item with a max durability stat" (`lowestDurability` start value), i.e. the stats seam found nothing (a gear-less or freshly dead character, or items without the `dur` pair); I did NOT reproduce it. The fix I can prove is `Client.RepairAll` (hover click at (368,461) confirmed by `dump items`).
5. **Town portal**: cast = right-click the scroll in the open pack (V-d2d). Walk-in: the portal is no npc, unit or ground item; **walking onto it does nothing** (stood on it, no level change); **a left click on its sprite takes it** (V-d2d: Blood Moor -> camp). The click point comes from `ScreenOfCell`: feet at (400,292), 80 px per cell of dx-dy and 40 px per cell of dx+dy (matches the `npcs` screen columns), minus 50 for the sprite. `d2dinput.UsePortal` uses an uncalibrated transform: switch it to `d2dctl.ScreenOfCell`/`Client.EnterPortal` (not my directory; `d2dtown.Input` already routes portal targets through `EnterPortal`).
6. **Waypoint**: panel opens (V-d2d); choosing a row **U**.
7. **Tome of TP**: **U** (not bought/run in the session).

## Fork verb requests (do NOT patch from the town side; a later agent implements them in the fork)

- **Click modifiers** (pack -> belt shift-click, `hand.ShiftClick`): `click <x> <y> [shift|ctrl|alt]` and `rclick` likewise; the SDL event gets the modifier state (`SDL_KMOD_SHIFT`) for the frame the click is handled. Until then `ErrUnsupported` stays; a `cmd belt put <item id> <slot>` verb (move a pack potion into a belt slot) would also do and is simpler.
- **Object interaction**: `cmd interact` takes only `npcs` indices (that includes the stash and waypoint, V), so chests, shrines and portals have none. Proposed `cmd use <unit-or-preset id>` for `debug presets`/portal rows, walking adjacent and activating, so the town portal walk-in needs no pixel projection.
- **Panel geometry dump**: `dump ui` could add `rects` for the open panel's buttons (repair, gold coins, tabs, waypoint rows) instead of our screenshot-measured constants.
- **Waypoint activation** for tests: `debug wp <area> [on]` (no verb activates waypoints, so choosing a row could not be exercised).

## Open

Full `town.Chores` on d2d (needs a nav Mover over d2d and a data.DB), belt moves, stock quantities, identify at Cain, merc revive/hire dialogs, tome cast, waypoint choice, live re-run of `d2dshop -steps menus,repair,stash,portal,wp` against the new verbs (see below).
