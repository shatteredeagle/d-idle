# Merc gear: player decides, merc paper doll (TASK-283, mercgear)

> status: research, docs only (no code, no live game). Tags: V = read in the tree or fetched this session (path or source named), S = secondary (SME advice, other docs), U = unverified or from memory. Target classic 1.14d; D2R-only facts flagged. Author of the question: user 2026-11-10 ("mercenary gear must follow the same rule as the player's gear; the player decides; the inventory page needs a merc doll with the same actions, drag and drop later").

## 0. Answer in short
1. The bot makes NO merc equip decision in production today. `merc.EquipIfBetter`, `Score`, `Allowed` and `MeetsReq` have no caller outside `internal/merc` and its tests (V: grep of the tree, 2026-11-10); the d2d reader fills no gear (`Gear` stays empty, V `internal/d2dmerc`, docs/merc-wire.md). The one bot action on merc gear is the strip before a replacing hire (`merc.PlanStrip`, `mercWire.HireStep`), and that path is dormant (no `Hirer` implementation on d2d, V docs/merc-wire.md). So "turn off under `resource.gearDecisions=player`" is one gate on the strip plus removing the dead policy code later.
2. Feasible, but not today. The game side is solid: d2d implements the real hireling screen (O key, packet 0x61, slot rules) (V, d2d source on the worker host, section 2). The gaps are all on our side: no read of merc-worn items (stock `items` lists only the player's; our dump has no merc gear), no merc panel geometry or click path, no open-panel verb, no UI. The unknown that decides the size: whether the lab fork's `dump` can list merc items (fork change, U).
3. Plan: 4 stages, about 26 to 34 agent-hours (section 6), read-only doll first.

## 1. What the bot does today for merc gear
Evidence is the tree on origin/main (2ddd3bb6) plus `docs/gear-policy.md` of branch agent/gearplayer (7c16fb5c; not on main yet, V read with `git show`).

| path | what | class | action under `gearDecisions=player` |
|---|---|---|---|
| `merc.EquipIfBetter`, `Score`, `Allowed`, `MeetsReq` (internal/merc/gear.go) | pure policy: swap at `merc.swapGainPct` gain, weights are judgments (docs/merc.md item 7) | equipment decision | no caller (V grep), nothing to turn off. Recommend deleting or marking "advisor only" when the advisor card wants it; `Allowed` and `MeetsReq` are worth keeping as the legality check of the UI (section 4) |
| `merc.Actor.Equip/Unequip` interface | only `Unequip` is called, by the strip | seam | none implemented for d2d |
| `merc.PlanStrip` + `HireStep` strip loop (cmd/d2bot/mercwire.go ~l.620) | before a REPLACING hire at Lut Gholein take the old merc's gear into the pack, because a hire destroys it (V ARR via docs/sme/mercenary.md) | unequip of merc gear | dormant today. Gate it: a replace of a merc that wears gear is not done by the bot; log `merc: replace skipped (merc wears gear: take it off on the Inventory page)` and wait for the player. Option in questions (Q3) |
| `merc.BuyPotionsForMerc()` | always false | b (consumable) | unchanged |
| `ShouldGiveHealing`, `cmd belt <col> merc` | potion to the merc | consumable | unchanged |
| hire, revive, claim, first-visit flag | not gear | | unchanged |
| pickit | keeps a weapon or armor for an empty PLAYER slot; nothing keeps items "for the merc" (V grep of internal/loot, internal/town) | | no merc loot rule exists; a future "keep for merc" mark is a player mark (section 5) |
| `resource.GearGate` | counts the player's gear only; merc state feeds only `MercKnown/Alive/Level` (V internal/resource/merc.go) | | unchanged: merc gear never gated anything |

The gearplayer change deliberately left "mercenary gear (internal/merc)" unchanged and flagged it U (V gear-policy.md table). The user's 2026-11-10 request closes that question: same rule. Proposed text for gear-policy.md (not written here): "mercenary gear: the player decides; the bot never equips, swaps or buys merc equipment; the strip before a replacing hire is refused while the merc wears gear."

## 2. What d2d exposes for the merc

### 2.1 Read side (what the bot sees)
| need | source | state |
|---|---|---|
| merc present, code, position, life, dead | dump `merc` section: `{present, code, x, y, life[cur,max], dead}` (V `internal/d2dsrc/testdata/dump_head_late.json`, `internal/d2dsrc/player.go`) | works; class maps only for code `RG` (V), other codes S/U (`tables.go`) |
| merc level, name, aura, Str/Dex | not in the dump (V player.go: `rp.unknown("merc.level"...)`) | missing; needed for requirement checks. d2d has them (the Hireling screen draws them, `View::MercSheet`, V d2d source) |
| merc worn items | stock `items` verb prints `character.items` only (the player's) (V d2d source `devctl_verbs.hpp` items handler); the merc's list is `character.merc_items` (V `components/game/character.hpp`), not exposed by any verb we use. docs/d2d-probes.md section 4: "no verb to equip on the merc was found; `items` lists only the player's items" (V, 2026-11) | MISSING. Needs a fork change (lab fork `dump merc` gear list or `items merc`), private fork, pushed the same day (AGENTS.md). Fork source I could read is the public upstream; whether the lab fork already has more is U |
| item fields for merc items | the player item row is `{id, code, loc, slot, x, y, w, h, quality, ilvl, sockets, qty, dur[], unique, set, flags, name}` (V `dump_items_late.json`) and the tooltip text from `items` (V docs/inventory-tooltip.md). The same shape would serve merc items with `loc:"merc"` (proposal, U) | merc items keep the d2s fields: location 1 (equipped), slot 1 head, 3 torso, 4 right arm, 5 left arm (V `tests/test_d2s.cpp` assert, d2d source) |

Watch API / invdesk: `status.InventoryData` has `Equipment map[string]*Item` for the hero only (V internal/status/inventory.go); `invdesk.View` mirrors it (V desk.go). Nothing for a merc. `status.Item` already carries everything the doll needs (id, code, name, quality, art, dur, tip, base, sell), so merc items reuse it unchanged.

Danger to avoid: if a fork change prints merc items with `loc=body` they would collide with the player's equipment in `d2dsrc` (`case "body"`). Use a new loc value and let `dumpItem` file it under a new container `d2.ContMerc` (U, name is mine).

### 2.2 Act side (what the game allows)
All V from the public d2d source on the worker host (`apps/d2d/hireling.{hpp,cpp}`, `components/game/world.cpp`, `components/rules/merc.hpp`, `docs/research/re/hireling.md`, a decompilation summary of 1.14d game.exe functions FUN_004929f0, FUN_0054d430, FUN_0048b290); S that the real game behaves the same (it is a reimplementation).
- Screen: UI 0x24, key O (Hireling Screen), expansion game with a merc (`view.merc_sheet`). A left-hand panel, like char, stash, cube; opening it closes those, and the inventory stays open on the right. So the player's pack and the merc doll are on screen together, the way a human drags.
- Packet C to S 0x61 with a slot. With the cursor holding an item: slot 0 gives it to the merc (a healing, antidote or thawing potion is drunk instead). With an empty cursor: slot 1 to 10 takes that worn item into the cursor. The client click handler sends slot 0 when holding and the hit box's slot otherwise.
- A bad item is refused silently and stays on the cursor (`merc_slot` returns 0). A replaced item goes into the cursor, exactly like the player's slots.
- Merc must be alive: the server returns early for `merc->hit_points <= 0` (V world.cpp). So no gear change on a dead merc, and no "take gear off the dead merc before replacing" (answers docs/merc.md UNVERIFIED 7 for d2d; S for the real game).
- The slot rule (`merc_slot`): identified, not broken; helm (and helm kinds) to head, `tors` (body armor) to torso; by monstats class: 271 bow, 338 spear or polearm, 359 shield (left hand) or one-handed sword, 560 one-handed axe, 561 sword, 560 and 561 also primal helms; then merc Str, Dex and level (WITH worn gear, `fight.merc_gear()`) against the item's. A noted d2d ponytail: ethereal and "requirements -x%" are not applied (V comment), so d2d may be stricter than the game on reduced-requirement items (U).
- Hover text: the merc's stats against the item's requirements (`FUN_0048dd90`), the game draws it.

### 2.3 What each merc may wear (1.14d; V decomp as above, S SME advice 2026-11-10)
| merc | class | slots | weapon types | notes |
|---|---|---|---|---|
| Rogue Scout (Act 1) | 271 | head, torso, right hand | bow only | crossbows illegal (V ARR). Circlets count as helms (S) |
| Desert Guard (Act 2) | 338 | head, torso, right hand | spear or polearm | javelins: the SME doc says yes, the decomp says no; treat as illegal (U, probe it) |
| Iron Wolf (Act 3) | 359 | head, torso, right and left hand | one-handed sword; shield | the shield adds defense, resists, sockets; it never blocks (S) |
| Barbarian (Act 5) | 561 (560 is the second variant in the rules) | head, torso, right hand | sword (decomp says "axe or sword" for the two classes) | primal helms allowed. Two-handed swords: unsettled (SME believes yes, U) |
Belts, gloves, boots, amulets, rings do not fit any merc (S). Druid pelts: expected illegal (U). Requirements use the merc's own level, Str and Dex, including his gear (V d2d code). D2R differs on several points (merc items from any NPC, shield for all, etc.); nothing here relies on it.
The repo's own table (`merc.Allowed`, internal/merc/gear.go) matches the decomp for 271, 338, 359 and 561, but `KindAxe1H` exists and `Allowed` has no axe for 561/560 and treats `Kind` as a bucket; reuse for the UI only after the test table (section 6) agrees with `merc_slot`. The SME doc's "Barbarian one sword" and the decomp's "axe or sword" conflict: the first live probe of stage 2 settles it (Q6).

### 2.4 UI primitives under d2d headless
- `key o` opens the screen (the `key` verb takes SDL key names, V docs/d2dctl.md; U that O is bound by default in the lab fork, V that it is the upstream default). `state` prints `hireling=<0|1>` (V devctl_verbs.hpp state line), so open-state can be read.
- Click path: `click x y` in game pixels (V d2dctl.md), the same primitive that equips the player's gear; the hireling box rectangles come from inventory.txt row `Hireling2` (V gamedata_load.cpp reads it as `merc_layout`). Our `internal/data` has `InventoryLayout(name)` and `internal/town/panel` builds the player layout from it; no merc row is built yet (V internal/town/panel/panel.go) but the data accessor takes any row name (V internal/data/db.go l.373).
- Reuse: `hand.Take(item, lay.Inventory)` picks the pack item (reuse as is), `hand.click` on a slot centre equips (the player's `doSwap` does exactly this, V internal/leveler/equip.go), `hand.ClearCursor` puts a cursor item into the first free pack cell (V hand.go). What is new: opening the hireling screen (`h.OpenHireling`, key o, wait for `hireling=1`), a `Layout.MercSlots` map from `Hireling2`, a body-slot name table for the merc (head, torso, rArm, lArm), a read of the merc's items to verify the click landed, and ensuring the left panel is the hireling one.
- Headless: nothing in the Hireling code needs a window beyond what the player's inventory already needs (it is a normal UI panel in the same framebuffer); U that no render-only gating exists.
- Prior probes: none for gear (docs/merclive.md covers claim, revive; docs/d2d-probes.md section 4 "no verb found"). The 0x61 path has never run in our lab (U live).

## 3. UI design: merc doll under the player's doll on /watch/inventory
Follows docs/ui-design-system.md section 5 components 13 and 14 and section 15.4 (the page lives in `internal/watch/web/inventory.js`, `invui.go`, `inventory-tip.js`; markup mockup `docs/mockups/ui/inventory.html`).

Component: a second `d-doll` instance with `data-owner="merc"` (the hero's gets `data-owner="hero"`, default), anatomy: header row (merc name or class, level, life meter `d-meter` data-kind life, aura or arrow-skill text when known), the doll frame, then a legend line. Slots, all `d-slot2`: `data-slot` in `mhead`, `mtorso`, `mrhand`, `mlhand` (prefix `m` keeps the hero's attribute API stable, ui-design-system 6.3). Grid areas for the merc: helm (2x2, top centre), body armor (2x3 under it), weapon (2x4 left), shield (2x4 right), same cell unit as the hero's. A slot the merc type cannot use (Rogue, Desert and Barbarian shield; all but Iron Wolf) is `data-state="na"` with a dimmed `d-badge` "no shield for a Desert Guard" and rejects actions with a reason; no belt, gloves, boots, rings or amulet slots exist, so the frame is 4 slots (this is the game's own Hireling screen, V RE doc). Empty slots show a silhouette; art keys: reuse `slot-helm`, `slot-armor`, `slot-weapon` (the shield slot shows the weapon silhouette like the hero's, V paperdoll-art-build.md), item icons `item-<invfile>` unchanged; the optional merc panel frame is a new key `panel-merc` from `PANEL\NPCInv` (U, art grammar change in `internal/art`), drawn mode first, art later. Tooltips: the existing `inventory-tip.js` works on any `[data-id]` item button; merc items carry the same `tip`/`base`/`sell` (the d2d tooltip for a merc item may show the PLAYER's requirement colours, so the "requires" line should be computed from the merc's stats, U).
Placement: directly below the hero doll in the same card (the user's "likely under the player's doll"), above the backpack grid, collapsed `d-fold` by default when no merc, `data-remember` fold state. On a phone it follows the hero doll in the single column (eight-cell width unit shared).
States (`data-state` on the merc card): `empty` = no merc hired, `d-empty` text "No mercenary hired" (plus hint "Hire one in Act 1 (free after Blood Raven) or Act 2"); `ready`; `dead` (merc dead: slots read-only, banner "Mercenary is dead: revive first; gear cannot be changed while he is dead", V section 2.2); `away` (alive but not near/unknown, only if the game needs proximity, U); `unknown` (no gear read available: shows the merc header only and "equipment unreadable on this backend"); `locked` while a request is under way. Never colour alone: a badge for PIN, MARK(sell/trash) and an up-arrow when the advisor card has a hint.
Actions on the detail card (same buttons as the hero: `Equip`, `Mark for sale`, `Trash` (and `Keep`)), plus `Take off` for a worn merc item. `CanWear` for a pack item lists merc slot keys (`mhead`...) next to the hero's, computed with the legality rules of section 2.2 against the merc's own level, Str and Dex. Drag and drop later: items already are `button` cells; dragging a pack item onto a merc slot (or a merc item to the pack) calls the same POSTs; keyboard path stays the buttons. Phone: no drag, buttons only.

Schema: `watch.inventory` `View.Schema` 2 to 3 (additive, omitempty, compat promise 0):
```json
"merc": {"owned": true, "alive": true, "class": 338, "kind": "desert", "name": "Aliza", "level": 17,
         "life": [63, 80], "str": 0, "dex": 0, "aura": "Defiance",
         "equipment": {"mhead": {Item}, "mtorso": null, "mrhand": {Item}, "mlhand": null},
         "slots": ["mhead","mtorso","mrhand"], "note": ""}
```
`slots` lists the legal slots of that merc (drives `na` rendering), unknown fields are omitted, readers of schema 2 ignore the key. Golden fixtures: `internal/watch/testdata/inventory_merc_{none,dead,desert,ironwolf}.json` plus DOM goldens for the four states (the existing `inventory_dom_*.txt` pattern, node DOM tests, V docs/inventory-tooltip.md) and a `compat` golden proving a schema 2 reply without `merc` still renders. `invdesk.ItemView` gains no field; `View` gains `Merc *MercView`. `status.InventoryData` gains `Merc *MercInv` filled by `invSource` from a new `d2.Merc` gear read. Packs the user can mark: `sell`/`keep` marks stay pack-only and keep their meaning; "Trash" is a new mark (see Q2).

## 4. Action model
Requests reuse the equip queue of the page (`POST /api/inventory/equip {id, slot}` queued, runner performs it at the gear step in town, V docs/inventory-ui.md). `agent/itemactions` (the player-side action queue) has no commit beyond main yet (V `git log origin/main..agent/itemactions` empty), so this proposes extending whatever it lands: `Request` gets `Target` (`hero` default, `merc`), kinds `equip`, `unequip`, `mark`.
| action | route (proposal) | effect | refused at once when | performed |
|---|---|---|---|---|
| Equip to merc | `POST equip {id, slot:"mhead", target:"merc"}` | pack item to the merc slot, replaced item goes to the pack | merc not owned or dead, not in town (default, like the hero), item not in the backpack, unidentified, broken, illegal type for this merc, requirement above the merc's own level, Str or Dex (checked from `merc_slot` rules with the read merc stats; refuse with reason, `Kind()` codes `merc`, `type`, `requirements`) | next safe town visit at the gear step, no fight nearby |
| Unequip from merc | `POST unequip {slot:"mhead", target:"merc"}` | merc item to the first free pack cell | merc dead, pack full (item footprint), not in town | same |
| Mark for sale | existing `mark`, allowed on pack items only | sold at the next vendor visit | same as the hero | vendor visit |
| Trash | new mark `trash` (drop to the ground at a safe moment or the existing junk path; Q2) | | worn merc items must be taken off first | next safe moment |
| Pin | a merc slot pin like the hero's (`Pin.Slot` "mhead") so the town never sells or "fixes" it | | | |
Safe opportunity: in town, merc alive, no hostile near, cursor empty, no other panel (the same preconditions as `Leveler.PlayerEquip`, V). The merc follows the player into town, so being in town is enough (S ARR: hirelings follow through portals and acts). One request at a time, a failure is shown and not retried, like the hero's.
Missing primitives and effort (agent-hours, estimates U):
1. merc item read (fork: expose `merc_items` in `dump`; Go: `d2dsrc` loc `merc`, `d2.ContMerc`, `d2.FindMerc` carries gear, `d2dmerc.Reader.State` fills `Gear`, level, Str, Dex): fork 3, Go 4.
2. merc panel geometry and open/close (`panel.Layout.MercSlots` from `Hireling2`, `Hand.OpenHireling`, `CloseAll` knows 0x24): 4.
3. click path `Hand.EquipMerc(id, slot)` and `UnequipMerc(slot)` with verify by re-read, retry like `doSwapRetry`: 5.
4. legality (`CheckMercEquip`) reusing `merc.Allowed` and `MeetsReq`, aligned to `merc_slot` incl. axes: 2.
5. desk, routes, schema 3, goldens, page markup and DOM: 8 to 10.
6. a live drill D8.g (hire via cheat, equip a helm, take off, wrong weapon refused): 3 (with the d2-live agent, hidden slot).

## 5. Staged plan
| stage | PR | what | hours | blocker |
|---|---|---|---|---|
| A | tiny | gear-policy: merc strip gated, `merc.equipIfBetter` marked dead, doc line (no behaviour change when `Hirer` is nil); test for the gate (Paladin and Sorceress both, per AGENTS.md) | 2 | gearplayer merged |
| B | medium | read side: fork `dump` merc gear (private fork, pushed same day), `d2dsrc` + `d2.Merc.Gear`, `status.InventoryData.Merc`, `invdesk` view, schema 3, read-only doll with all four states, goldens, no actions | 12 to 14 | fork change (Q5) |
| C | medium | actions: `hand` hireling panel, equip/unequip, queue `target:"merc"`, refusals, drill D8.g offline with fakes, live run on a hidden slot | 9 to 11 | B, itemactions landed, one live probe |
| D | small | drag and drop on the page (merc and hero slots), `Trash` mark, art `panel-merc`, advisor hints for merc (suggest only) | 4 to 6 | C |
Total 27 to 33 agent-hours (A 2, B 13, C 10, D 5; midpoints). Stage B is shippable alone and already answers "what does my merc wear".

## 6. Questions for the user
1. Replace-hire with a geared merc (stage A): refuse until the player takes the gear off (recommended, no bot gear action), or allow the bot to strip into the pack as today (it is not a choice of equipment, it saves it)?
2. "Trash": what should it do, drop on the ground in town, sell for gold, or delete nothing and only mark "junk, never pick up again"? (Dropping has no cost; selling is the existing "sell" mark.)
3. Equip outside town: refuse (default, like the hero) or allow in the field when calm? The merc screen needs no town in the game.
4. The pack space on unequip: should Unequip refuse when the pack is full, or may the bot stash something (the player's marks only)? Recommended refuse.
5. May the lab fork get one more dump field (merc worn items, level, Str, Dex, name) and, if cheap, an `items merc` verb? It is a private-fork patch, not in this repo.
6. Barbarian and Desert edge cases (two-handed swords on the Act 5 Barbarian, javelins on the Desert Guard): accept "legal per the decomp rule, refused otherwise" in the UI, or wait for a live probe before offering the buttons for those types?
7. Aura or gear hints for the merc (Insight polearm, Might aura) in the doll: show as plain advice text only (S, SME list), never an auto action?
8. Merc potions are still bot-driven (consumable, unchanged): confirm that is within "no equipment decisions".

## 7. Dead ends and caveats
- I did not run anything live; everything about the 0x61 path and the real game is S through d2d's source, U in our lab.
- The fork source on the worker host was the public upstream tree; the lab fork's patches may add verbs I could not see (U).
- d2d is GPL: this file gives our own summaries with file citations, no code copied (docs/decisions.md rule 3).
