# 06 Loot: pickit, pickup, ground items, cubing

Sources: KB:core/Pickit.js (Result codes, checkItem 262-340, pickItem 346-580, pickItems 620-760, fastPick 770-840), KB:core/NTItemParser.js, KB:pickit/*.nip, KB:core/Cubing.js, KB:config/Templates/Cubing.txt, KO:internal/action/item_pickup.go + step/pickup_item.go, TY:docs/features/hover-confirmed-item-pickup.md and pickit-engine.

## 1. Pickit result model
`Pickit.Result`: UNID -1, UNWANTED 0, WANTED 1, CUBING 2, RUNEWORD 3, TRASH 4 (sell for gold), CRAFTING 5, UTILITY 6.
`checkItem(item)` precedence:
1. NIP file rules -> result (UNID if the rule needs identification and item not identified).
2. If result UNWANTED and no pickit files loaded and item type is an "essential" (gold, scroll, health/mana/rejuv potion) and `canPick`: WANTED ("Essentials").
3. Ort/Ral runes when cube-repair enabled: UTILITY.
4. Crafting system / Cubing ingredient / Runeword ingredient lists: CRAFTING / CUBING / RUNEWORD.
5. Gold-poor fallback: if still UNWANTED and not ignored type and not quest item and (item already in inventory in town, or gold < LowGold, or (gold < 500k and no pickit files), or gold < repair cost): gold items WANTED; others: if inventory not fully locked: sell value per inventory square >= 2000 -> TRASH "valuable", >= 10 -> TRASH "LowGold item" (pick up to sell).
`LowGold` SP: 25,000 Normal, 50,000 NM, 100,000 Hell.

## 2. NIP format (what we need to parse)
Line: `[properties] # [stats] # [maxquantity]` comments `//`. Properties: `[type]` (item type e.g. ring, amulet, armor, shield), `[name]` (base item name), `[class]` (normal/exceptional/elite), `[quality]` (inferior/normal/superior/magic/set/rare/unique/crafted), `[flag]` (ethereal, identified, runeword), `[level]` (ilvl), `[prefix]`,`[suffix]`, `[color]`, `[wsm]`, `[2handed]`, `[sockets]`, `[minimumsockets]`; stats are item stat names (`[fcr]`, `[maxhp]`, `[strength]`, `[fireresist]`, `[sorceressskills]`, `[itemmagicbonus]`, `[enhanceddefense]`, `[itemchargedskill]` ... ) with arithmetic between stats (`[strength]+[dexterity] >= 30`), `&&`, `||`, parentheses, comparison operators; `[tier]`/`[secondarytier]` assign equip tiers (SP). Third section `[maxquantity] == n`: stop picking when you hold n matching items (inventory-fix counts items in storage owned by me).
Evaluation: for each line (compiled into closures), property part then stat part; line index/file recorded for logging ("kept by line N of kolton.nip"). Two checks: with and without tier lines. Unidentified items: stats not evaluable -> result UNID if property part matches and stat part exists, so ID first.
Example rule shape (paraphrase): rare ring with FCR 10 and to-hit and life/mana thresholds. Provided files: kolton.nip (large, endgame oriented), classic.nip, LLD.nip, gold.nip, pots.nip, keyorg.nip, follower.nip, shopbot.nip.
Our plan: implement a small NIP subset in `internal/bot/loot` (Go, no deps): tokenizer for `[name] op value`, `&&`, `||`, `()`, `+`; item property getters from `internal/d2` item data + `internal/data` base items (name, type code, class tier) and an affix/stat reader. Start with a tiny built-in rule set (runes, gems >= flawless? potions, keys, uniques/sets, rare jewelry with basic stats, charms of value) rather than whole kolton.nip.
Alternative source of truth for what to pick: pre-identification quality is visible via item flags and the item name color on the ground (quality byte in item data; mem 2.7 `quality`), so unique/set/rare/magic quality is known before pickup, but stats need identify (item identified flag in flags).

## 3. Ground-item detection
- KB: `Game.getItem()` iterates ground items (unit type 4) filtered by `onGroundOrDropping` (mode 3 on ground, 5 being dropped). Event listener `itemaction` records new gids that dropped (mode 0) for `FastPick`.
- Ours: item hash table (mem 2.5) filtered by item mode (on ground) and location; new-item detection by comparing sets of unit ids each tick (drops appear after the monster dies; wait ~1 s after the kill for drops KO `waitForDrop` = 1 s).
- `canPick(unit)` (rules):
  - quest items: skip if already have same classid.
  - gold: skip when carrying max gold (level*10000).
  - scroll: only if a matching tome exists with quantity < 20 (tome classid = scroll classid - 11).
  - key: not for assassin; skip when stack would exceed 12.
  - unique charms (Annihilus/Torch/Gheed): skip if we already own one.
  - potions: only if belt/buffer needs it: needed = sum over belt columns configured for that potion type of (beltSize) minus potions of that type in belt; if satisfied and belt full, consider buffer counts (HPBuffer, MPBuffer, RejuvBuffer); if need < 1 and a worse potion (lower classid) is in belt/inventory: drink it to make room and pick the better.
  - everything else: `Storage.Inventory.IsPossibleToFit(unit)`.

## 4. Pickup procedure (`pickItems`, `pickItem`)
```
pickItems(range = PickRange 40):
  if dead or disabled: return
  scan ground items: for gold within 4 pick immediately; else add to pickList if on ground and within range
  if potions in list: clearBelt first
  while list not empty:
    sort by distance to me (FastPick: runes and uniques first)
    item = real unit by classid+gid; drop from list if vanished
    skip if item behind a wall (walkers only; teleporters and town ignore)
    status = checkItem; if wanted and canPick:
       canFit = inventory fits or essential type
       if FieldID enabled and (no fit or inventory used > FieldID.UsedSpace): identify in the field (`me.fieldID`), recheck fit
       if not canFit: 
           if canMakeRoom (junk to sell and can TP, or wanted unid, or stash candidates): visitTown (chores), then restart pickItems if used space went down
           else mark item ignored, request mule if configured, stop
       else pickItem(...)  (failure stops this pass)
    shift
pickItem(unit, retry=3):
  validate gid/on ground; close blocking UI (inventory, npc, waypoint, shop, stash, cube) with 500 ms wait
  up to retry attempts:
     abort if dead; abort if already picked (same classid+gid in my inventory)
     wait until not moving
     telekinesis for tkable types (gold, scroll, potions) when 5<dist<20 and clear line and mana > 2*cost
     else: if dist > 6 (FastPick or first try) or > 4 (later) or wall in between: move to within 4 (Pather.move, no clearing/picking); re-check still on ground
           click the item (packet click first try; mouse click on retries)
     wait <= 1 s for item to leave ground; gold: compare gold stat
  verify: item count increased or item present in inventory by gid
```
Numbers: pick range 40 (field), fast pick range separate; gold auto pick at 4; max 3 tries; walk to within 4; 1 s verification; `Pickit.pickList` re-sorted each iteration; recursion guard via `track.lastItem`.
KO pickup (external, MIT): select items on ground within `maxDistance`; potions only if belt missing count > 0 (per type); abort pickup and fight if an enemy within 7; same item failing twice -> town to sell/stash and retry; `PickupItemStep`: stable target, move the mouse to item screen position (item world pos minus (1,1)) with archimedean spiral offsets (`helper.Spiral(attempt)`), click only when memory says hovered item matches id; up to 45 mouse attempts, 3 s after click without result -> error; if a chest is hovered instead (Andariel corpse area) click it open; item farther than 7 -> error ("too far").
TY: requires 2 consecutive stable snapshots (same area/unit/txt/position); candidate = nearest, tie lower unit id; no blind click; success = item left ground or appears in inventory with same txt id, `verify_ticks` consecutive; statuses: picked_up, target_unstable, target_lost, too_far, hover_not_found, projection_failed, pickup_failed.
Our approach: copy TY/KO exactly (we have hover/mouse target in memory, mem 2.12): world->screen via the iso transform in PLAN, hover over item label hot-spot (ground items are clickable at their sprite or label; Alt-key shows labels in D2: **holding the show-items key (default Alt) shows ground labels which make clicking far easier** UNVERIFIED, used by Koolo's `show_items` key and botty).

## 5. When to pick up (timing)
- After each `clear`/`kill` (pickit at end of fight) and after seals/chest opening; not mid-fight unless FastPick is enabled (then after each kill via drop event).
- Delay >= 1 s after boss death for drops (KO waits up to 1 s if none seen), Andariel/Duriel stuff drops in waves: `Attack.kill` returns then `delay(2000)` for minions (Andariel).
- Potions and gold: even in fights if within 4 (gold); potions only when needed.
- Never pick items in range of dangerous alive monsters (<7 for KO).
- Priorities: runes/uniques first when FastPick; otherwise nearest-first.

## 6. Cubing (KB Cubing.js)
Cube = 3x4 container; Horadric Cube quest item. Flow (`doCubing`): update ingredient lists from items; shuffle recipe order (avoids one recipe blocking others); for each recipe with all ingredients present: open stash UI, empty cube, move ingredients to cube (verify count), open cube, `transmute()`, wait 700 ms+ping, inspect results: UNWANTED dropped, WANTED kept (logged), runeword/crafting results update lists; empty cube; loop until no recipe can run. Close UI at end.
Recipe families supported: Gem upgrades (3 -> next quality: chipped..perfect), Rune upgrades, Token of Absolution, Hit Power/Blood/Caster/Safety crafts (rare item + 3 specific perfect gems/jewel + base), Unique/rare to exceptional/elite upgrades, Socket recipes (shield/weapon/armor/helm), Reroll magic/rare (grand charm, diadem), Rune recipes, Rejuv/full rejuv recipes. Config format `Config.Recipes.push([Recipe.X, itemName|classid, Roll.Eth?])`; `Config.Cubing=true` enables.
Do not hard-code recipes: parse `cubemain.txt` from the user's MPQ (`internal/data`); each row lists inputs, output and quantity, and a generic matcher covers gems, runes, tokens and crafts. (From general game knowledge, UNVERIFIED here: 3 gems of one type/quality make the next quality; 3 identical low runes make the next rune.)
Ingredient reservation: items that serve cubing are kept (Pickit CUBING) and excluded from selling; "invo quantity" caps (e.g. max 3 chipped gems).
Mapping: `internal/data` cubemain.txt, `bot/loot/cube.go`, input: cube panel slot coordinates (3x4), transmute button at 800x600 (calibrate), UI flag Cube.

## 7. Gold and system items
Gold: pick up always unless full; stash at >= StashGold; gambling later. Keys: needed for locked chests (non-assassin) and key-org; buy up to 12.
Gems/runes: WANTED by NIP or by cubing. Potions as described. Charms: SP `CharmEquip` keeps best charms by custom score rules (e.g. "ResLife" small charms with allres 5 and life 20: max 3).

## 8. Mapping
| need | package | data |
|---|---|---|
| NIP-lite evaluator | `bot/loot` | item: base code/name/type, quality, flags (identified, ethereal), ilvl, sockets, stat list (itemstatcost ids), requirements |
| ground detection | `d2` | item units with mode ground, position, id, hover flag |
| pickup | `input` + `nav` | iso transform, hover/target from memory (mem 2.12), inventory fit test |
| fit test | `bot/loot` | container grid occupancy + item size from `internal/data` |
| cubing | `bot/loot` + `data` | cubemain.txt, cube grid, transmute click |
