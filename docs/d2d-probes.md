# d2d probe batch (agent D2DPROBES, 2026-10-30)

Tag **V-d2d** = observed on d2d (a reimplementation, commit e36922e); the original game stays the acceptance target.
**D2D-src** = read in d2d's source or comments (not checked on Game.exe). **U** = not tested.
Setup and verbs: docs/d2d-spike.md. Scripts: docs/d2d-probes/ (our own code: thin client `d2p.py`, `c` wrapper, start/stop/restart).
Characters are Paladins (`--start-class 2` gives a Paladin, `1` a Necromancer: UI order, not the d2s byte).
Everything runs in `~/d2d-lab` on the worker host, `nice -n 10`, one instance, private socket and XDG dirs.

## 1. Free Rogue (Kashya, Blood Raven reward)

Commands (per character): `--start-screen ingame --start-class 2 --start-name X`, then
`debug stat 12 8`, `debug stat 14 20000`, `debug goto 17`, `debug kill`, `debug goto 1`, `d2p talk Kashya`, `state`.

| Case | Observed (V-d2d) |
|---|---|
| Clvl 8, BR killed, before talking: hire window | normal prices, 8 rows: `Aliza Lvl5 Life63 Def31 Cost130`, Amplisa 195, Kyoko 195, Klaudia 240, Fiona 195, Divo 130, Liene 160, Oriana 115 (cost not level^2 based: it is per offer). |
| Opening Kashya's menu (the interaction itself, not clicking `talk`) after BR kill | `debug quest 1 show` burial state 4 -> 5, quest 2 bits `0x2011`, **a Rogue (`merc=..:RG`) appears with gold unchanged (20000 -> 20000)**. The reward is a direct grant, no hire window price. |
| Hire list afterwards | the first offer (Aliza) is gone and a new one (Raissa, 172) is appended at the end: the free merc is the **first row of the list**. |
| Merc already owned (hired Aliza for 130 first, then BR + Kashya) | burial state still 5, bits `0x2011`, **no second merc, merc unchanged, no gold change**; reward is consumed (D2D-src `World::kashya_merc`: returns if `merc_seed` set, in LoD even for a dead merc). So hiring any merc before the reward forfeits the free Rogue. |
| Hire while owning a merc | clicking a row hires at once (gold -130), the old merc is replaced, no confirm, no refund. |
| Quest word | `debug quest 2 show` bits: `0x2012` after the kill, `0x2011` after the reward; Act 1 record `burial`: 3 entered, 4 killed, 5 reward paid. |

Needs a probe on the real game: that the reward is granted on opening the menu (original: probably a dialogue line first), exact bit values, the merc type (d2d gave a Rogue archer `RG`; the fire/cold variant is not visible in `state`).

## 2. Quest quit-safety (plan-v2 10.5 rows d2d can answer)

Method: a fresh Paladin per row, `debug stat 12 N` (level), `debug goto <level>`, `debug kill`, `save`, `restart.sh NAME` (save + quit + start `--start-screen charselect --start-name NAME` + `key Return`), then the same reads again. Levels.txt ids: 17 Burial Grounds, 37 Catacombs 4, 49 Sewers 3, 102 Durance 3.
`debug quest <q> show` prints the quest `q` bit word (`bits=`) and the Act 1 record states `den,burial,tools,cain,tower,andy` (records are rebuilt at load, the BITS are what the save keeps).

| Row | Commands / observed (V-d2d) | Verdict on d2d |
|---|---|---|
| Blood Raven (quest 2) | kill: record burial 3 -> 4, quest 2 bits `0x2012`. `save`, restart: bits `0x2012` kept, record back to 3, **Blood Raven and her pack respawn** (a new level each game). Talk to Kashya without re-killing: record -> 5, bits `0x2011`, **free Rogue granted** | **Safe (A)**: reward works after reload without a second kill. The kill itself must happen once. |
| Andariel (quest 6) | kill: record andy 4, bits `0x200a`; a town portal appears 10 ticks later (`debug portals`: `0 37 11.5 17.2` and `1 1 34.7 20.7`: a portal pair, level 37 and town; D2D-src: FUN_00596490 tick 10, so the original opens one too, plan-v2 10.5 says "no corpse portal": needs the real game). `goto 1`, `save`, restart WITHOUT talking to Warriv: bits `0x200a`, record 3, Warriv menu has `talk / go east / cancel`; talking sets bits `0x2009`; `go east` lands in Lut Gholein (level 40) | **Safe (A)**: Act 2 access survives the reload (the portal itself does not). |
| Mephisto (quest 22?) | `debug goto 102` (Durance 3), `debug kill`: the object `op46` id 342 (the hellgate, `debug objects`) goes NU -> ON; fresh character: NU. `save`, restart, `debug goto 102`: **Mephisto alive again and the hellgate object already ON** with no kill. | **Not B**: a join with the quest bit set reopens the Act 4 gate (matches d2d's notes, contradicts the user report). The kill is NOT needed again. Real game must still settle it (the user's report may refer to the portal needing a fresh kill in a still-open game, not after reload). |
| Den | spike: bits persist, den record back to 3, reward still paid at Akara | A |
| Cain scroll, Malus, Radament, Staff pieces, Khalim's orb, Summoner, Duriel | not reached: the quest objects/NPCs are inside rooms that are only built when the player walks near, `debug goto` puts you at the level's middle and `debug presets` lists only built rooms; Atma (Act 2) did not appear in `npcs` or `presets` after two 6 s walks (also `cmd move` was ignored while the game menu was open, see below). | **U** |

Source reading (D2D-src, `docs/research/re/quests-act2.md`, not tested): Radament's join rule: with bit 0 (done) set, bit 5 (book due) set and no "ass" book carried at join, bits 0/1/15/5 are cleared and the quest is "to be done again in a later game". That is exactly "keep the book or redo" in plan-v2 10.6.

Interface notes found on the way (V-d2d):
- `state gamemenu=A:B`: A = 1 means the Options menu is open. `key Escape` after an NPC menu already closed opens the game menu and **a further `cmd move` is ignored**; `key Escape` again did not close it for several seconds, `click 400 308` (Return to Game) does. Close NPC menus by clicking `cancel`, not Escape.
- Opening an NPC's menu (the walk-up) already triggers the quest talk event (Kashya, Warriv, Akara rewards fire when the menu opens, not when `talk` is clicked).

## 3. Death and corpse (V-d2d, extends the spike; script `docs/d2d-probes/death.py`)

Commands: `debug stat 12 L`, `debug stat 14 P`, `debug stat 15 S`, `debug die`, `click 400 300`, `key Escape`, then `state`, `ground`, `debug corpses`.

| Case (Normal, Paladin) | Result |
|---|---|
| clvl 12, purse 1100, stash 0 | **no toll**: the pile on the ground is the whole 1100, purse 0. |
| clvl 11, purse 1100 | same: pile 1100. |
| clvl 12, purse 1100, stash 2000 | pile 1100 (stash untouched, total 3100 is below 12 x 500 = 6000). |
| clvl 20, purse 50000, stash 0 | pile **40000**, toll 10000 (= 20%). |
| (spike) clvl 8, 10000 + 5000 | toll 1200 (8%): the toll applies only above the `level x 500` floor. |

Reading: pile = purse - toll, toll = min(level, 20) % of (purse + stash) but never taking the total below `level x 500` (D2D-src `World::death_penalty`); purse is always set to 0. This matches our live G1 case (1125 gold at clvl 11: all of it left the purse, V OUR). So at low level a death costs only the walk back to the pile, not gold. The live game needs one confirmation of "toll 0 below level x 500".
- Gold pile: **walking onto it picks it up** (autoloot, deviation): `cmd move 31 14` collected 40000 + 1100 + 1100 (gold 1100 -> 42200) without any pickup verb; `cmd pickup <unit>` did nothing extra. The pile is not saved.
- Multi-corpse: every `debug die` adds a corpse, **also with 0 items** (`debug corpses`: 5 entries after 5 deaths; only the ones with items are restored after a reload, spike). Each holds only what was worn at that death. `cmd interact -3000 - i` takes corpse `i` (list order = oldest first) and **any index works, not only the oldest**; the list shrinks by one each time. Item restore: back to the same slot if free, else the inventory (cap, Long Sword came back).
- XP: Normal 0, Nightmare lost 575 at clvl 8 (spike); not re-measured.
- After death: belt potions and inventory kept; only worn items and the held item go to the corpse; the character respawns in the act town of the act where it died (died in Durance: Act 3 town, level 75).
- Quirk: `debug die` text "You have died" must be dismissed with a click or `key Escape`; the second Escape then opens the game menu (see section 2).

## 4. Mercenary (V-d2d, partial)

- Hire window rows are re-rolled per level of the character: at clvl 20 the same names show `Aliza Lvl 17 Life 171 Def 127 Cost 310`, Amplisa 465, Klaudia (lvl 19) 510, Liene (lvl 19) 340, Oriana (lvl 16) 295, Divo 310; at clvl 8: Aliza lvl 5 cost 130, Klaudia lvl 7 240. Cost depends on the offer (merc level and variant), not a single formula we know; the window shows `Your Gold`.
- Kashya (Act 1), Greiz (Act 2), Asheara (Act 3), Tyrael (Act 4), Qual-Kehk (Act 5) resurrect a dead merc (D2D-src `kMercNpcs`); cost `min(level^2 / 2 * 15, 50000)` taken from the purse then the stash (D2D-src `merc_resurrect_cost`), which is the user's formula. The revive line (string 0x1507 "resurrect") appears while the merc is dead. **Not seen live**: my merc (clvl 17, `merc=..:RG`) never died (Durance 3 monsters stayed passive until the player was killed, the merc followed the player into town alive).
- Hiring while owning a merc replaces it with no confirmation (section 1). Merc gear: no verb to equip on the merc was found; `items` lists only the player's items. **U**.

## 5. NPC menus (V-d2d, Act 1 Normal, clvl 20, script `menus.py`)

| NPC | Lines |
|---|---|
| Akara | `Akara, talk, trade, cancel` (the `Reset Stat/Skill Points` line is absent here: it appears only while the Den reward is open, D2D-src world.cpp Respec: stats reset once, then the quest bit is consumed) |
| Kashya | `Kashya, talk, hire, cancel` |
| Gheed | `Gheed, talk, trade, gamble, cancel` |
| Warriv | Act 1 before Andariel: `Warriv, talk, cancel`; after Andariel: `talk, go east, cancel`; Act 2: `talk, go west, cancel` |
| Charsi, Deckard Cain | did not open (not in `npcs` after the walk; Cain is only placed after the Tristram rescue). `trade` clicked at the printed coordinates opened the stash chest instead of the vendor window on one run (the NPC was not yet adjacent): after `talk`, wait about 1.5 s and re-read `menu` before clicking. |
Menu x,y are screen pixels and change with the NPC position (examples Kashya 483..620, 112..264).
Vendor prices: **U** (store items are not in `items`; reading them needs hover tooltips from screenshots, not done).

## 6. Belt, potions (partial)

- `debug equip <code> 8` puts a belt on slot 8 (`lbl` Sash accepted); belt row count by type was not observable through `items`; D2D-src reads it from Belts.txt of our own MPQ (rows per belt). **U** for the row table.
- Shift-click to belt: no modifier on `click`; `cmd belt <slot> [merc]` only drinks (spike). The inventory-to-belt move would be a real `click` on the item, then a click on the belt slot (picks the item up onto the cursor and drops it: D2D-src `put_in_belt`).
- Potion heal rates and clvl 12 regen: **not run** (time).

## 7. Waypoints

Source says (D2D-src `docs/research/re/waypoint.md`): 10 s refusal after the last level change, first touch lights a dark waypoint without the panel (second opens it). **Not tested live on d2d** (time).

## 8. Drops and economy (V-d2d, partial; script `drops2.py`, data `drops_sweep.jsonl`)

First visits only: a level is built once per game and **re-entering it does not respawn monsters** (my first loop of 148 re-entries "killed" the same 6 monsters 148 times and was discarded). Method: fresh load, `debug goto L`, `debug warp x y` over a 12-cell grid to build every room, `debug kill` at each stop, tally by unit id. Character clvl 12 Normal.

| Level | Monsters (unit ids) | Gold piles | Gold per pile | Ground items |
|---|---|---|---|---|
| Blood Moor (2) | 95 (all lvl 1) | 6 | avg 2.5, max 4 (sum 15) | 23 |
| Cold Plains (3) | 237 (lvl 1: 130, lvl 2: 105, lvl 5: 2; 1 unique, 1 superunique, 6 minions) | 29 | avg 4.5, max 10 (sum 131) | 65 |

Together 332 kills: 0.27 items per kill, 0.105 gold piles per kill, about 0.4 gold per kill. Items are mostly potions (Minor/Light Healing, Minor/Light Mana, Stamina, one Rejuvenation), arrows/bolts, TP/ID scrolls, keys, junk weapons and armor, 3 rings, 2 Sashes. Quality is not in `ground` (name only: only "Superior ..." and "Low Quality / Crude / Damaged / Cracked ..." prefixes are visible, magic and rare names did not show up as such, so magic/rare/unique rates are **not measured**). Our live G1 log (docs/g1-log.md, 167+ kills Blood Moor, "gold 0 -> 3, small gold piles skipped") agrees with tiny gold at this level. Not comparable further: live kills per hour is not reproducible with `debug kill`.
Vendor prices (Akara/Charsi buy and sell, repair, identify): **U** (see section 5).

## 9. Perf and memory (V-d2d; scripts `sample.sh`, series `perf_idle.txt`, `perf_fight.txt`)

`sample.sh OUT SECONDS 30` logs `seconds RSS_kB cpu% load` for the d2d process.
- **Idle in Act 1 town, 20 minutes:** RSS flat at 471 MB (471340 -> 472256 -> 472032 kB, +0.2%), CPU 89-91% of one core (60 fps loop, niced, the worker host load 26-35).
- **Level-cycling "fight" run, 20 minutes** (a freshly loaded clvl 12 Paladin with a merc, every 8 s `goto town` then `goto Blood Moor` and `debug kill`: level changes, not continuous combat; dead monsters after the first round): RSS 564 MB at start (load + merc + visited levels), 564 -> 560 MB at the end, never above 564, CPU 91-93%. So no unbounded growth in 20 minutes. The spike's 515 -> 746 MB came from many first-visit level builds and difficulty switches (each new level costs RAM, a revisit does not). First visits cost: the sweep of two outdoor levels (about 330 monsters) is not in these series; budget up to ~750 MB for a long exploring session. **Real continuous combat (monsters alive and fighting) was not sampled.**
- Wall: one `debug goto` + 3 s wait per level hop; a full 12-cell sweep of Cold Plains (80 x 80) took about 2 minutes.

## 10. What this changes (plan-v2 / BACKLOG)

| Probe | Result | Change |
|---|---|---|
| Free Rogue | granted when Kashya's menu opens after BR (state 3 -> 5), price 0, only with no merc at all; own hire first forfeits it; list drops the first offer | plan-v2 10.5 "free Rogue" step: do the quest BEFORE hiring anyone; no hire window needed |
| BR / Andariel / Den quit-safety | all A (reward after reload); BR respawns | keep 10.5 rows; Andariel: a town portal DOES appear (D2D-src and V-d2d), update the "no corpse portal" note |
| Mephisto | reload with the quest bit reopens the Act 4 gate, Mephisto respawns; kill not needed again | 10.5 / 10.6 row "Mephisto B" becomes C: take the portal anyway until the live probe; user report is the acceptance test |
| Death gold | no toll below level x 500; pile = purse; walking over it picks it up; any corpse index claimable, 0-item corpses accumulate | BACKLOG: gold-loss estimate = min(level,20)% of the part above level x 500 (our clvl 11-12 deaths lose nothing but the pile) |
| Merc revive | cost formula and NPC list from source; not seen live | keep merc.* knob formula level^2/2*15, cap 50000 (D2D-src agrees with the user) |
| NPC menus | Akara reset line only while Den reward open; Warriv `go east` after Andariel | town/quest code: do not wait for a Reset line |
| Perf | RSS flat 20 min idle and level-cycling, 0.9 core | one d2d instance is cheap enough for a nightly rig |
| Not done | Cain scroll, Malus, Radament, Staff, Orb, Summoner, Duriel, potion rates, regen, belt rows, vendor prices, waypoint rules, merc gear, real fight perf | needs room exploration (objects live in rooms built only near the player) or screenshot reading; each is a half-day on d2d, or one live probe on Game.exe |

Still needs a probe on the real game: free Rogue grant mechanics, Mephisto reload, Andariel portal, death-toll floor, merc revive price, everything tagged U.

## Repeatable scripts (docs/d2d-probes/, our own code, run on the worker host next to `~/d2d-lab`)

`d2dstart.sh ARGS` / `d2dstop.sh` (one niced instance on the private socket), `restart.sh NAME` (save + quit + load character), `d2p.py` (thin client plus `talk`, `pick`, `goto`), `c "cmd" ...` (run it from here over ssh), `hirewin.py`, `menus.py NPC...`, `death.py`, `drops2.py OUT LEVEL...`, `sample.sh OUT SECONDS`. They expect `~/d2d-lab` (d2d build, `ctl.py`, `start.sh` as in docs/d2d-spike.md); none contains d2d code.
Example: `d2dstart.sh --seed 3 --start-screen ingame --start-class 2 --start-name X; c "debug stat 12 8" "goto 17" "debug kill" "goto 1" "talk Kashya" "debug quest 2 show"`.

State at the end: d2d stopped, `~/d2d-lab` kept (1.7 GB, the worker host `/home` 140 GB free), characters Probea..Probeg in `~/d2d-lab/home/config/d2d/save`.
