# Town and loot: internal/town, internal/loot (agent TOWN)

## API summary
`internal/town`
- `town.New(e *env.Env, mv Mover) *Town` (Townsman contract). `Mover` = `GoTo, GoToUnit, ExitTo, TakeWaypoint` (subset of the NAV Mover). The Town builds its own `loot.Looter` as `Judge` (`SetJudge` to share one).
- `ToTown(ctx)` (portal scroll: right-click in pack, click portal, wait for a town area), `BackToField(ctx, level)` (town portal object, else waypoint, else `ExitTo`), `Chores(ctx, ChorePlan) (ChoreReport, error)`.
- `ChorePlan{Heal, Identify, Sell, Potions, Scrolls, Repair, Stash, Gold, Insure, Force}`; `FullPlan()`. Order: heal, identify, sell, shop (potions + scrolls), repair, stash/gold. Each step has a need check (skipped unless `Force`), a failure does not stop later steps, panels are closed at the end. Only not-in-town / not-in-game / ctx errors abort.
- `NeedsTrip(snap) (bool, reason)`: low potions, belt columns below knob, no TP scroll, pack full, durability. Pure.
- `Rebuy(ctx, budget) (RebuyReport, error)`: poverty mode after a death (potions first within `rebuyPotionPct`, then cheapest viable weapon/armor/helm/boots/gloves/belt/shield up to `rebuyGearTier`, equip by clicking into the slot, report `Missing`).
- `Stock(npc)` vendor stock; `UseSource(Source{Snapshot, Items, Owner, Stats})` for replay/tests; `*ForTest` hooks used by `cmd/townlive`.
- Sub packages: `panel` (800x600 geometry from inventory.txt, `Occupancy`), `belt` (belt model + level plan, embedded `plan.json`), `hand` (verified mouse moves: Take/Put/Transfer/ClearCursor/PanelOpen), `sim` (offline simulated town used by tests), `cmd/townlive` (live checks, `cmd/townlive` is a symlink for `tools/live.sh`).

`internal/loot`
- `loot.New(e, mv) *Looter` (`Mover` = `GoTo`), `Evaluate(d2.Item) Verdict` (contract; no base class so only quality/flag/ilvl rules can match), `EvaluateUnit(d2.Unit)` (the real entry: base row, stats when a rule needs them, needs from belt/pack), `PickUp(ctx, PickOpts) (PickReport, error)`, `Valuables(snap) ValuablesReport` (stash-before-danger insurance), `SetPickit`, `UseSource`.
- `Verdict{Action keep|stash|sell|drop, Tier S|A|B|C, Prio, Rule, Reason, NeedID}`.
- Pickit = JSON data (`pickit_starter.json`, format documented at the top of `rules.go`): ordered rules, first match wins; match on codes (globs), types (ancestors via data.IsA), qualities, quest, identified, ethereal, clvl/ilvl, stat conditions (`stats`/`anyStats` with shift), `needed` (potion, scroll, tome, key, gold).
- Pickup: candidates in radius from the ground list, filtered by tier vs strictness knob, fit-checked (belt for potions, 10x4 pack occupancy, free-cell reserve), sorted by rule prio then distance (gold/runes/gems/uniques 70-95). Per item: `Mover.GoTo` within `walkDist`, `input.Interact` (hover confirmed), wait up to `confirmMs + 90 ms/subtile` for the item to leave the ground, then prove it in memory (unit in pack/belt, gold stat rose, stackable merged), place a cursor item into the pack. Max `pickTries`, an item failing twice is skipped, `Abort` polled between and during items.
- Knobs: `env/tuning_town.go` (`town.*`), `env/tuning_loot.go` (`loot.*`) with ranges.

## Verified live
### Round 2 (TOWN2, 2026-10-28, instance i3 "Towni", a DIFFERENT seed: Rogue Encampment origin is about (-370,+590) from Testy's)
Setup: save edit level 8, str/dex 45, gold 6000, then memory `gold` cheats (all in the cheat ledger, actor town2).
VERIFIED LIVE:
- Seed independence: NO absolute world coordinates remain in internal/town or internal/loot. The old `Act.Approx` tables are gone; the fallback for a unit not in the table is its PRESET position (`World.Current().Presets`, kind 1 NPC / 2 object). Presets and unit tables matched on Towni (Gheed preset (5618,5577), unit (5618,5570)). `cmd/townlive` uses the real `nav.Navigator` and `mem.OpenGame()` (instance aware). Offline: `TestChoresAtOtherSeedOrigins` (three origins), `TestFallbackUsesPresetsNotConstants` (the mover only accepts the preset position).
- First NPC contact plays a speech (about 60 s) that kills hover; the menu opens by itself afterwards. `openNPC` now waits `town.greetingWaitS` (90) after a click instead of clicking again.
- Buying by right-click: potions fill the belt (3 hp1 + 1 mp1 into a belt-less char), TP scrolls (go into the tome), ID scrolls, keys (`town.keyTarget`), Tome of Town Portal. Price of potions/scrolls/keys equals `ItemBase.Cost`; gear costs 0.9-1.5x Cost. Right-click buying gear into an EMPTY matching slot EQUIPS it at once (dagger, quilted armor); into an occupied slot it goes to the pack.
- Cannot afford: pre-check `gold < Cost/2` skips the click (no modal), a failed click with `gold < 2*Cost` returns `ErrNoMoney`; the game shows a modal "You do not have enough gold for that" (no UI flag; Esc via CloseAll dismisses). Chores report `Broke=true` and no error (live: 5 gold, 0 errors).
- NeedsTrip "low on mana potions" no longer repeats: Barbarian plan has no mana column (`belt.PlanForClass`, `Town.SetPlan` for custom plans), and a visit that could not fix a reason silences it until gold grows by `town.futileGoldStep`.
- Vendor tabs: one tab per non-empty stock page, left to right (x = 118 + 81*i, y 75); Charsi has Armor, Weapons, Weapons, Misc. Computed from the stock pages (`tabPoint`).
- Repair All button = (369,452), 4th of the 4 icons (Buy, Sell at x 264, Repair at x 317); hover tooltip read "Repair all equipment: 0". The REPAIR EFFECT is UNVERIFIED: nothing could be damaged (axe 28/28 and armor 20/20 after a god-mode melee fight).
- Selling: equipment slot click -> item on cursor -> click vendor grid sells (strip of staff, dagger, armor); chore `sell` sold a gambled cap.
- Stash gold deposit: the coin icon at (487,453) (tooltip "Deposit", hit area x 484-490 only) opens "How much gold would you like to deposit?" prefilled with the full amount and no UI flag. Typed amount: WM_CHAR 8 (backspace) x n, WM_CHAR digits, Enter: exactly 5000 of 28870 moved; chore keeps `stashGoldBase + level*stashGoldPerLvl` (300 + 50/lvl) and deposits the rest. (369,451) and (433,452) are the CLOSE buttons of stash and pack; (165,91) opens WITHDRAW. The helper `backspace` key name was added (code + test) but the deployed d2input.exe predates it; WM_CHAR 8 works with every helper.
- Rebuy poverty mode: potions, then cheapest viable weapon and body armor bought and equipped; slot-swap equip by pick-up and slot click works (hand axe in, dagger back to pack).
- ToTown: scroll/tome right-click in the open pack, portal appears within 1 s, entered; area change both ways. In a fight (god mode, 2 monsters within 12 subtiles) 1.3 s. An already open portal is reused without a scroll (`town.portalReuseDist`, tome count unchanged). BackToField through the town portal works; the portal object is class 59 in town, and the FIELD side portal is gone after you came back through it (so the next ToTown reads a scroll). Interact on a portal falls back to ClickWorld.
- Ground pickup: item dropped by clicking the ground lands at the click's world position; `loot.PickUp` with a keep rule picked it (hover confirmed, item in pack at the end).
- Gambling: Gheed's gamble entry (menu line 3) opens the shop window; gamble stock is a different set, rings/amulets cost far above 30000 at clvl 8. A gambled item came back already identified.

UNVERIFIED: repair effect, identify with scroll/tome (no unidentified item found), mercenary, Cain, other acts, rebuy of helm/boots/gloves/belt/shield tiers, buy-to-pack path of full pack.

### Round 1 (TOWN, Testy clvl 1)
VERIFIED LIVE:
- NPC dialog opens via `input.Interact`; UI flag 0x08. Dialog lines: Gheed Talk/Trade/Gamble/Cancel, Akara Talk/Trade/Cancel. The box floats above the NPC, bottom-anchored (Cancel about 113-117 px above the feet, lines 15-16 px apart); `pickEntry` tries 0, +5, -5, +9, -9.
- Trade opens flag 0x0C and keeps 0x08. The backpack is drawn beside the shop WITHOUT inventory flag 0x01 (same for the stash: only 0x19), so never press `i` there.
- Vendor stock is readable from memory (item units owned by the NPC; X,Y = grid cell; `Item.Page` = tab).
- Grids: vendor 10x10 at (96,123), backpack (419,315) 10x4, stash 6x8 at (154,142), equipment slot rects match screenshots.
- Selling: right-click does nothing; pick up and drop on the vendor grid sells.
- Stash open via class 267, pack item moved in and out (`hand.Transfer`). Flag 0x11 stays set after a quest update: ignore it.

## Dead ends and gotchas
- Fixed dialog pixel rows do not work; the box moves with the NPC.
- Right-click selling; reading the vendor stock only after the shop opens is unnecessary but the units are only complete once the NPC was opened.
- The d2input helper of an instance dies now and then (connection refused): `tools/game.sh helper iN` restarts it.
- Class index of an item unit = position in `data.DB.Items` (weapons, armor, misc). Checked: 63 = short staff, 587 = hp1, 529/530 = tsc/isc (docs/mem.md calls them tomes).
- Vendor buy price is not `ItemBase.Cost` (leather armor cost column 481); purchases are confirmed by the gold change, prices are only a budget estimate.

## Interface notes for others
- BOT: call `NeedsTrip` each tick, `ToTown`, `Chores(FullPlan())` (set `Insure` before dangerous trips), `BackToField`; after a death `Rebuy(ctx, budget)`. Add `Rebuy` to the `Townsman` interface.
- NAV: needs `GoToUnit` to stop 3-5 subtiles from the unit; `GoTo` for fallback NPC positions (act tables in `acts.go`).
- INPUT: `backspace` key added; redeploying d2input.exe is the main agent's call (shared by all instances).

## Mercenary (LEVELERCLASS, 2026-10-28; `internal/town/merc.go`)
- Data: hireling.txt via `MercCandidates` (V real tables): Normal Act 2 Desert Mercenary Id 6 Prayer, 7 Defiance, 8 Blessed Aim (Might and Holy Freeze only from Nightmare), seller class 198 (Greiz).
- V live (Palli, i2): Greiz stands beside the Sewers hatch; a nav walk to him (GoToUnit) stepped onto the hatch and killed the character twice, so merc talking clicks him from a distance (`Town.noWalk`). His dialog is Talk / Hire / Cancel (`MercMenu`, `PickLine(1,3)`). The hire window is a list of seven rows "Name - Lvl - Life - Def - Cost" with the type text under the name (Defensive = Defiance, Combat = Prayer, Offensive = Blessed Aim); the order differs between visits and the type is not in memory. Hovering at x 400 (x 250 did nothing) highlights a row; one click hired it: gold 20000 -> 19650 (row said 360), merc portrait and a spear Desert Mercenary appeared. Done by hand through `townlive talk 198 1:3 h:400:269 w:10 h:402:271 w:10 k:402:271`.
- Not verified: `Town.Hire` end to end. Three runs of the coded path (`townlive hire ROW`) with a merc already alive opened the window, highlighted the row and closed it without spending gold; probably a living merc blocks a second hire (U). Rows below the seventh, the aura of a row without a screen reader (`MercWant.Row` or `RowPicker` is required, else `ErrMercRowUnknown`), revive (`Revive` is a skeleton returning `ErrMercUnverified`), merc inventory.

## TOWNFIX (offline, 2026-10-29)

Offline fixes for live findings (docs/summon.md "Assassin live results"). None re-verified live: checklist below.

- Greeting stall (V live as a symptom: first Akara visit sat 3 min; cause U): `waitGreeting` waits for the NPC menu in 20 s slices up to `greetingWaitS`; each slice without the menu ends with Esc (`In.CloseAll`), then `openNPC` clicks the NPC again. Esc is never sent while the menu is open.
- `NeedsTrip` is affordability-aware by default: gold-costing reasons (potions, belt columns, scrolls, ammo, repair) are not raised below `Tune.Resource.TripMinGold` (60). `ignoreTownAdvice` still silences the whole advice.
- Every reason a visit leaves standing is remembered (not just the first; the old single-reason memory let two reasons alternate forever): quiet for `futileMinutes` (10), and for gold reasons also until gold grew by `futileGoldStep`. The bot also stays quiet about "inventory N% full" for the same time after a trip that left the pack full.
- Equip into small slots (EQUIPFIX, 2026-10-29): live soak i4 (Soaky) failed every amulet into `neck` and a ring into `rHand` ("item N did not reach slot neck"), while gloves and rArm equips worked through the same code. The cause is NOT proven (U). Slot rectangles (800x600, Sorceress): neck (609,95)-(632,119), rHand (495,240)-(518,264), lHand (609,240)-(632,264); centre clicks are inside. Leading hypothesis (U): the game refuses the item (hidden affix level requirement above clvl 6; the pack amulets were found/gambled magic items), which leaves it on the cursor, exactly what the log shows. Second (U): the real 23x24 box is offset from inventory.txt. Code now: one slot table `hand.SlotNames` (+ `hand.SlotPoint`: try 1 and 2 click inside the box off-centre), a refused click logs `leveler: slot click refused` (item, slot, point, stillOnCursor, clvl) and the item goes back to the pack with the cursor proven empty, failures back off per item (skipped until clvl rose by its failure count, max 4) and per slot (two items failed at one level blocks the slot for two levels). Sim: neck, rings, belt, boots, gloves slots, `World.EquipRefuse`. Live check: tools/live.sh on a clvl<8 char with a magic amulet in the pack: run the gear equip, read `slot click refused ... stillOnCursor`; if true with a low-level item the cause is a requirement, if equips work on a new point the box offset is real.
- Leveler equip: `doSwapRetry` retries a failed swap once after closing panels and re-reading memory (the item may have landed late, or in the other ring slot).
- `tools/jobs.sh kill` kills the job's whole session plus descendants (setsid leader, `timeout` puts d2bot in its own process group, so the old pid/child kill missed it), TERM then KILL, and exits non-zero if anything survives. Dry run: `tools/jobs.sh _killer <pidfile> [grace]` against a local setsid+timeout+TERM-ignoring tree printed "killed", 0 members left (V, local).

Live checklist: fresh character, first Akara click (greeting gone within about 20 s or Esc dismisses it); 0-gold Sorceress leaves town without a potion trip; soak job then `tools/jobs.sh kill`, `pgrep d2bot` on the worker host empty.

## Shop order: town portal scroll first
`shopVisit` first calls `buyFirstTP`: with no tsc or tome in the pack and gold >= the vendor's tsc price, it buys one scroll
(`resource.BuyKit`, priced from the actual stock; V in the sim, price on live ~100 is U). Only then `buyPotions` (so
potions get the remainder, and `potionCap` counts the scroll in `rep.Spent`), then `buyScrolls`. Ammo still runs before
the shop step. `TestShopBuysTownPortalBeforePotions` (120 gold, no scroll).

## Imbue and sockets (agent IMBUESIM; every UI step U live, offline-tested in the sim)
Code: `internal/town/imbue.go` (shared flow, Charsi), `socket.go` (Larzuk). Interfaces `town.ItemImbuer` / `town.ItemSocketer`, implemented by `*Town`; `cmd/d2bot/task.go` `dialogAdapter.Talk` calls them for (Charsi, "imbue:best") after the plain Talk, and (Larzuk, "sockets") instead of it.

Flow tables (the only place with rows and coordinates; `ImbueFlow`, `SocketFlow`):

| Flow | NPC | Dialog click (idx of n) | Confirm | Proof |
|---|---|---|---|---|
| imbue | Charsi 154 (V class) | line 2 of 4 (Talk, Trade, Imbue, Cancel; U, mirrors quest.Menus) | none (U) | pack item in the same cell has quality >= magic |
| socket | Larzuk 511 (V class) | line 0 of 3 (the reward comes with Talk; U) | none (U) | same item id has the socketed flag or stat 194 > 0 |

Model (U): after the line the inventory shows and the game waits for ONE left click on a pack item; a worn target is taken off first (`hand.Unequip`, V live path), so the leveler's gear step re-wears the result. The target is chosen BEFORE the dialog opens, so a missing target (`ErrNoTarget`) never burns the one-time reward.

Choice rules (pure, `PickImbue`, `PickSocket`):
- Candidates: pack and worn (main set) items that are normal or superior, not ethereal, not socketed, usable by the class (leveler.Candidate), not bows or quivers.
- Imbue: gain = leveler score of the same item as an unidentified rare minus the score worn in its slot; the biggest gain wins, bases a runeword wants lose to those none wants; gain <= 0 means no target.
- Socket: expected sockets = gemsockets capped by the type's MaxSock tier for the item level (S rule "maximum"); value = best `gear.RunewordScore` among offline-available runewords whose rune count equals that and whose type fits; else base score; worn wins ties. V (real tables): Spirit and Insight are ladder rows in 1.14d runes.txt here, so Smoke (Nef El, body armor) is the best 2-socket target the table knows.

Live checks (all needed, nothing flipped yet): 1. With the Malus: Talk to Charsi, does the Imbue line appear at index 2 of 4 and what does clicking it show (inventory by itself? cursor change? a confirm dialog and where)? 2. Does a worn item work, or only a pack one? Is the imbued item the same id (the proof uses the cell)? 3. After Shenk: Larzuk Talk, does the picker open straight away or after a second line? 4. The socket count the game gives (stat 194) against `ExpectedSockets`. 5. Fill `ServiceFlow.Confirm` and the `N`/`Idx` values from what is seen.
