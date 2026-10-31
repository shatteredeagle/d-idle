# d2d feasibility spike (agent D2DSPIKE, 2026-10-30)

> status: spike answered: d2d is the primary backend (ADR 0009, 2026-10-30; docs/plan/STATUS.md). Numbers here are the first measurements; current ones are docs/d2d-fidelity.md, docs/d2d-pace.md and docs/d2d-loop.md.

Goal: can https://github.com/psi29a/d2decomped ("d2d", GPL-3.0, native C++ reimplementation of D2 LoD 1.14d,
no Wine) serve as a native test surface for our Go bot, and what does it tell us about the original?

Tags: **V** = observed by me in this spike (command and output below), **D2D** = d2d's own docs/source say so (not
checked against the original game), **U** = unverified.

Licence rule: nothing from d2d (code, binary, text beyond short literal values) is in this repo. Everything lives in
`~/d2d-lab/` on the worker host (outside the repo). d2d's MPQ data is our own install, copied read-only.

## 1. Verdict in five lines

1. Builds and runs headless on the worker host with no Wine, no Xvfb, no display: **V** (SDL dummy video driver, software renderer, OpenAL null).
2. The control channel is good enough to drive a character around, fight, talk to NPCs, hire a merc, die and save: **V**.
3. It answers several of our open rule questions with source-level detail (death penalty, quest persistence): **V + D2D**, see section 5. Fidelity to the real game is claimed, not proven by me: **U**.
4. It runs in real time only (60 fps cap, about 0.6 to 0.9 of a core per instance), no fast-forward flag: **V** (loaded machine).
5. Using it as a bot backend is a real project (about 2 to 3 weeks), not a drop-in, because our reading layer reads raw Game.exe memory. Recommendation in section 8: use it as a rules oracle and a small scripted test rig now, do not invest in a full adapter yet.

## 2. Build (task 1)

Commit built: `e36922eab35f22020c91293406b4a9beb887320c` (main, 2026-10-30, shallow clone).

| Item | Result |
|---|---|
| Toolchain | GCC 16.2.1 (cmake picked it by default), CMake 4.4.4, Ninja; SDL3 3.4.18, OpenAL 1.25.2, libcurl 8.23, zlib 1.3.2 from the system. **V** |
| Pulled by the build itself | StormLib 9.30 (built from source, not installed on the worker host), CLI11, sentry-native/crashpad (cloned from GitHub by CMake FetchContent), a small Bink-only FFmpeg. Needs internet on first configure. **V** |
| Not needed | Qt6 (launcher only). Not built. **V** |
| Command | `cmake -S . -B build -G Ninja -DCMAKE_BUILD_TYPE=Release -DD2D_WERROR=OFF && nice -n10 ninja -C build -j6 d2d` |
| Problem 1 | With the default `D2D_WERROR=ON` the build stops at 595/600 with two GCC 16 errors in d2d's own code (`-Werror=array-bounds` in apps/d2d/ui.cpp, `-Werror=maybe-uninitialized` in a std::optional<variant> copy). Both are false positives of the newer GCC. Fix: `-DD2D_WERROR=OFF`. **V** |
| Time | The Werror=OFF build rebuilt all 592 steps: 4 min 23 s wall, 18 min CPU (`-j6`, niced, the worker host load about 27). **V** |
| Warnings | About 2200, nearly all from third-party crashpad/sentry code and the C++26 headers. **V** |
| Size | `d2d` binary 13.5 MB; build dir 277 MB; copied MPQs 1.4 GB (9 files, no video MPQs); whole `~/d2d-lab` about 1.7 GB. **V** |

## 3. Running it (task 2)

Data: copied (cp, then chmod a-w) from the worker host's `~/Games/d2lod/proton/pfx/drive_c/Program Files (x86)/Diablo II/`:
`Patch_D2.mpq d2char d2data d2exp d2music d2sfx d2speech d2xmusic d2xtalk .mpq` into `~/d2d-lab/data/`. The two video MPQs were
left out and `--no-video` is used. Originals untouched. **V**. d2d accepted this 1.14d install as is (no `--patch`).

All state (config, saves, log) goes to a private dir through `XDG_CONFIG_HOME=~/d2d-lab/home/config`, so `~/.config/d2d` is untouched. **V**

```
# ~/d2d-lab/start.sh (my wrapper, not committed)
export XDG_CONFIG_HOME=~/d2d-lab/home/config XDG_DATA_HOME=~/d2d-lab/home/data
nice -n10 ~/d2d-lab/src/build/apps/d2d/d2d --data ~/d2d-lab/data --headless \
    --devctl ~/d2d-lab/d2d.sock --no-video --seed 3 --start-screen ingame --start-class 1 --start-name Spike
```

- Headless: `--headless` = SDL dummy driver + software renderer; screenshots still render the real frame. No Xvfb was needed, so none was started and nothing of the user's display was touched. **V**
- Startup: "first frame presented" 1.25 to 2.1 s after launch, game data 360 ms, scene 720 ms (hot disk cache). **V**
- `--start-screen ingame --start-class N --start-name X` creates a new character and enters the game at once. `--start-screen charselect --start-name X` then `key Return` (about 6 s) loads the saved character. **V**
- Class index N is the UI order, not the d2s class byte: `--start-class 1` produced a Necromancer (Wand start item, necro skill ids), not a Sorceress as d2d's README implies. Use the class table in the launch UI. **V**. Necromancer is what the probes below used.
- Seed: `--seed 3` fixes the map layout: Den of Evil monster positions were identical in every session. Monster HP and AI rolls differ per run (e.g. Fallen Shaman 7/9 then 9/9), so it is not a full replay seed. **V**
- Socket: `srwxr-xr-x` owner only, single client at a time (`err busy` for a second), no auth, a plain AF_UNIX path of our choosing. Only the owner can connect (write bit needed). **V**
- Sound: about 10 `MPQ read failed ... .wav` warnings per session (ADPCM wavs in the music/sfx MPQs, "damaged MPQ?"), sound for those files is skipped. Harmless for us. **V**; cause not investigated (their own notes say attribute CRCs of ADPCM wavs never match, **D2D**).

My client: `~/d2d-lab/ctl.py "cmd1" "cmd2" ...` (30 lines of python over the socket: send a line, read until a line `ok...` or `err...`). The reply format is: payload lines (tab separated) then `ok`.

## 4. Control channel, observed (task 3)

All outputs below are real, from the sessions in this spike (level 1 Necromancer "Spike").

```
ping                 -> ok
info                 -> w=800 h=600 frame=823 / ok
screenshot /abs/path.png -> ok 442927     (PNG, 800x600, renders the full frame incl. HUD; relative paths go to <user dir>/screenshots; "~" is NOT expanded)
state                -> one line: screen= class= name= difficulty= cam= walking= gold= str= life=45/45 mana= stamina= level= exp= pmode= lskill= rskill=
                        waypoint= items= menu= merc=35.5,16.1:RG gamemenu=0:0 ...    (about 50 fields)
debug level          -> ok 1 30.70 13.70 57 41 864 912          (level id, player x y in cells, level w h, act origin)
debug players        -> 1<TAB>Spike<TAB>30.70<TAB>13.70<TAB>-1<TAB>-1<TAB>0
npcs                 -> Kashya  812 484 1 0 35.7 13.5      (name, screen x y, has-menu, index for cmd interact, cell x y)
monsters             -> fallen1  12.10 28.70 752 436 2/2 NU - - lvl1 Fallen #2 -
                        (class id, cell x y, screen x y, hp/max, mode, boss kind, mod ids, level, name, unit id, anim row)
ground               -> gld  8800 Gold  371 331 #1 40.9 11.5
items                -> [wnd loc=1 slot=4 q=2 panel=0 id=1 size=1x2 at=0,0] + indented tooltip lines (Wand / One-Hand Damage: 2 to 4 ...)
debug quest 1 show   -> ok bits=0x0014 den=3 skillpts=0 states=3,0,0,0,0,0 ...
debug attack         -> ok dmg=2-4 ar=80 def=6 block=0 dr=0%+0 mdr=0 res=0/0/0/0 cb=0 ... (full combat sheet of the player)
debug presets        -> every object/NPC the level placed incl. hidden ones (Deckard Cain, waypoint, chests)
debug warps          -> 0<TAB>67.0<TAB>28.0<TAB>8        (index, cell x y, destination level id)
debug blocked X Y    -> ok 0|1        debug grid [n] -> '.'/'#' map rows        debug objects / portals / corpses / critters
```

Coordinates **V**: positions are floating **cells** (one cell = 5 subtiles; steps of 0.2 = one subtile; town is 57x41 cells). Our bot uses integer subtiles, so x_sub = floor(5 * x_cell) (D2D: `debug grid 5` is the subtile grid). Screen coordinates are the 800x600 game frame, the same iso projection as the original (D2D). Not calibrated by me: **U**.

Movement/action verbs (**V**):

- `cmd move X Y` (cells) walks. `cmd interact <npc index>` walks next to an NPC and opens the menu; if the path is long it may stop short, repeat it until `menu` shows lines (my `loopint.py` did that). `cmd skill <id> X Y <unit id> left` fired a necro Teeth at a monster unit and killed it (hp went to -9/9, mode DD). `cmd pickup <unit>` walked onto a dropped gold pile and picked it up (gold 0 -> 8800). `cmd interact -3000` takes the player's own corpse (see 5a). `cmd belt <slot> [merc]` drank a belt potion (the merc variant is "shift-click": the merc drinks).
- **Quirk V**: right after `debug goto`, a level change, or loading a save, `cmd move` and `cmd interact` are silently ignored (player does not move, no error) until a real `click X Y` has been sent once. Sending `click 400 500` first fixed it every time. Cause not found, **U**.
- **Quirk V**: `key Escape` toggles the in-game menu; two Escapes in a row mean it is open again. Check `state` field `gamemenu=` before assuming a menu is closed.
- `debug goto <level>` works for outdoor levels but for the Den of Evil it left an unrendered black level with no monsters; the reliable way into a cave is `debug warps` then `debug enter <i>`. For Burial Grounds (17) `debug goto 17` worked and Blood Raven was there. **V**
- `debug points <skill> <n>` and `debug skill` only accept class skills of the character; a wrong class id answers `err not a class skill`. **V**

## 5. Probes (task 4)

Character: Necromancer "Spike", started as level 1, set to level 8 with `debug stat 12 8` (Normal, then Nightmare/Hell by `debug difficulty 1|2`).
Cheats were all on d2d's own throwaway character; our cheat-ledger rules do not apply to d2d, nothing touched our live games.

### 5a. Death (V, rules read from d2d source marked D2D)

Commands: `debug stat 14 10000`, `debug stat 15 5000` (stash gold), `debug stat 13 40000` (exp), `debug die`, `key Escape` (dismisses "You have died / Press ESC to continue"), `debug corpses`, `ground`, `save`, quit, restart, `key Return`, `debug corpses`.

| Observation | Result |
|---|---|
| Normal, level 8, purse 10000 + stash 5000 | Screen text "Death takes its toll of 1200 Gold". Purse 0, a pile of **8800** appears on the ground at the death spot (`ground` -> `gld 8800 Gold`), stash untouched. 1200 = 8% of (purse+stash) = level % . |
| Normal, purse 8800, stash 0 | toll 704 (8%), 8096 on the ground. |
| Rule (D2D `World::death_penalty`, matches both numbers) | lost = min(level,20) % of purse+stash, capped so total keeps level*500 in single player, taken from the purse only; the rest of the purse is dropped at the death spot; purse set to 0. |
| XP, Normal | **unchanged** (40000 -> 40000). |
| XP, Nightmare, level 8 | 40000 -> 39425 (lost 575). D2D: 0 / 5 / 10 percent of the level's exp span (Normal / Nightmare / Hell), never below the level start + 1. 75% of the lost exp (431) goes onto the corpse. |
| Corpse | `debug corpses` -> `<level> <x> <y> <item count>`: the equipped items (and a held cursor item) move to the corpse; the character respawns in the act town at its start spot (30.7, 13.7) with an empty body. Belt, inventory and potions are kept. |
| Retrieval | `click` on the lying body did **not** work for me (two attempts). `cmd interact -3000` (= -3000 - corpse index) walked to it and gave back all items and the exp (39425 -> 39856, wand back on the body slot). D2D (`take_corpse_items`): each item back to its slot if free, else inventory; requirements are not checked. |
| Save and reload with a corpse | `save`, quit, restart, load: the corpse is back **in town** at (31.7, 14.7), one cell diagonal from the arrival spot, with its item. This holds for a corpse made in the Blood Moor (died at 40.1,22.1) as well: it is relocated to the town. A corpse with **0 items** (nothing worn) was not restored after reload. |
| Gold on the ground after reload | gone. The pile is not saved. Purse after reload 0. |
| Hardcore, resurrect | not tried (U). `cmd resurrect` is for a merc/ghost case (D2D), not tested. |

### 5b. Quest persistence (V)

`debug quest 1 show` prints the flag bits and the Act 1 record states `den,burial,tools,cain,tower,andy`.

| Step | Output |
|---|---|
| New character | `bits=0x0000 den=1` |
| Opened Akara's menu (no kill yet) | `bits=0x0004 den=2` |
| Entered the Den through the cave mouth (`debug warps`, `debug enter 0`) | `bits=0x0014 den=3` (their bugs.md #6: entering sets a bit) |
| `debug kill` (9 monsters) | `bits=0x2016 den=4`, exp +145 |
| `save`, quit, restart, load | `bits=0x2016`, **den record state back to 3**, character in town |
| Entered the Den again | 9 monsters **respawn** (levels are rebuilt per game, same positions by seed) |
| Talked to Akara (`click` on the "talk" line) | `bits=0x2001 den=5 skillpts=1 states=5,1,...` : the reward (skill point) was granted and Sisters' Burial Grounds started |

So the quest-cleared flag survives save-and-exit before reporting to Akara, and the reward is still paid; the cave itself is not "cleared" again after a reload. For the original game this matches what we believe, but I did not compare against Game.exe: **U**.
Quest bits are per difficulty (Nightmare started at `bits=0x0000`). **V**
Blood Raven (`debug goto 17`, `debug kill`): burial state 3 -> 4, then Kashya's menu gained `talk / hire / cancel` (**V**).

### 5c. Mercenary hire (V, free Rogue case U)

- Kashya's menu at clvl 1 to 7: header, `talk`, `cancel`. At clvl 8: header, `talk`, `hire`, `cancel` (D2D: the hire line appears above clvl 7, not quest-gated: `docs/research/re/npc-menu.md`).
- Clicking `hire` (the `menu` line coordinates) opens the hire window and `menu` lists it as lines (screen text, centre x 400):

```
Your Gold: 20000     Hire which Mercenary?     400 136
Aliza - Lvl: 5  Life: 63  Def: 31  Cost: 130   400 164
Amplisa - Lvl: 5  Life: 63  Def: 31  Cost: 195 400 199
... (8 rows, 35 px apart)                       ...
cancel                                          400 444
```

- `click 400 164` hired Aliza: gold 20000 -> 19870 (cost 130), `state` shows `merc=35.5,16.1:RG` (cell x,y : class code RG = Rogue archer), the merc follows and fights. The merc is kept across save/load. **V**
- The offer rows carry name, level, life, defence, cost only; the merc type (fire/cold arrows, skill) is not shown in the row text. A bot must infer it from the name list or the class code. **V**
- Free Rogue (D2D `World::kashya_merc`): talking to Kashya after Blood Raven gives the first offer of her list for free, only if the character has no merc at all (LoD: not even a dead one). I had a merc already, so I could not test it. **U**.

### 5d. NPC menu lines (V, Act 1 Normal, clvl 8)

`menu` returns `<text> <x> <y>` with the header first; coordinates are game pixels and depend on where the NPC stands on screen, so a bot must read them each time (lines sit about 15 px apart: header, then +18, +33, +48 px). Clicking the exact y of a line is safe; I once clicked 6 px above `talk` and still hit it.

| NPC | Lines (in order) |
|---|---|
| Charsi (index 7) | `Charsi`, `talk`, `trade/repair`, `cancel` |
| Akara (index 4) | `Akara`, `talk`, `trade`, `Reset Stat/Skill Points`, `cancel` |
| Kashya (index 0) | `Kashya`, `talk`, `hire` (only above clvl 7), `cancel`; after Blood Raven (merc present) the same |
| Gheed (index 12) | `Gheed`, `talk`, `trade`, `gamble`, `cancel` |

The `Reset Stat/Skill Points` line on Akara in Normal is shown at clvl 8 here (the 1.14d respec is free once per difficulty; U, not tested).

### 5e. Belt (V partly)

`cmd belt 0` drank the first belt potion for the player (4 -> 3 potions), `cmd belt 1 merc` gave one to the merc (3 -> 2), a third `cmd belt 2` healed the player from 4/45 to 45/45 over 6 s. Real shift-click on a belt slot is not reproducible through the channel: `click` has no modifier (`key shift` is `err unknown key`); the `merc` argument of `cmd belt` is the equivalent. Belt contents are visible in `items` as `loc=2 ... at=col,0`. **V**

## 6. Performance and headless facts (task 5)

| | Observed (the worker host, load average 25 to 28, niced) |
|---|---|
| CPU idle in town | 90% of one core (10 s average) |
| CPU while casting at a pack in the Den | 58% of one core. Lower than idle because the machine's load changed; do not read it as "combat is cheaper". Use 0.6 to 0.9 core as the figure. |
| Frame rate | 250 to 300 frames per 5 s (50 to 60 fps), software render 8 to 20 ms per frame. Hard cap `kFrameBudgetMs = 16` in the main loop (D2D source). No flag to run faster. Game time follows the wall clock (U: I did not try to scale it). |
| Memory | RSS 515 MB after load, 590 MB after Den visits, 754 MB after more level changes and difficulty switches (a level rebuild keeps growing it: leak or cache, U). |
| Startup | 1.3 to 2.1 s to first frame (data cache warm). Load of a saved character plus town about 6 s. Difficulty switch 5 to 6 s with a one-frame stall ("MAIN STUCK 1022 ms in ingame:world-floor" logged once). |
| Stability | Ran about 11 minutes through load, 3 reloads, level hops, deaths, difficulty 0/1/2, hire: no crash. Final check: 10 min 31 s uptime, ping ok, 5 slow-frame warnings (all at level loads), RSS 746 MB. |
| Fast forward | None. A time-scale would need a source change (loop pacing and a dt multiplier). U. |
| Determinism | Map layout by `--seed`; combat/HP rolls not reproduced across runs. |

## 7. Integration with our bot (task 6)

What we have: `internal/mem` is a byte-level `Source` over Game.exe's process memory (`ReadAt(addr)`); `internal/d2.Reader` decodes the original's structs into `d2.Snapshot{State, Player, Inventory, Monsters, Objects, Missiles, Ground, UI, ...}`; `internal/env.Env` holds a concrete `*d2.Reader` (internal/env/env.go:38); `internal/input.Controller` (click/key/hold/ClickWorld/MoveTo/Interact/CastAt) drives `d2input` over TCP; `internal/bot/contracts.go` holds the consumer interfaces (Mover, Fighter, Townsman, Looter...).

An adapter has two possible seams:

1. **Fake the memory (not recommended).** Synthesise a 1.14d process image from d2d state so `d2.Reader` works unchanged. The structs, linked lists, stat vectors and skill pointers are many and exact; every field the bot touches would need emitting. Big and brittle.
2. **Adapter at `d2.Snapshot` level (recommended if we do it).** Introduce a small `Reader` interface (`Snapshot()`, `Player()`, `Units()`, `Inventory()`, `UI()`) that both `*d2.Reader` and a new `d2dsrc.Reader` implement, change `Env.Read` to that interface, and build `Snapshot` from the channel:

| Our field | d2d source | Gap |
|---|---|---|
| Player pos, life/mana/stamina, level, exp, gold, class, difficulty, merc | `state`, `debug players`, `debug level` | stamina is 8.8 fixed (matches ours after /256); **stash gold, attributes beyond str, resistances, all other stats** are not in `state` (`debug attack` gives res and damage). A generic "dump all stats" verb is missing. |
| Skills (id, level, left/right) | `state` has lskill/rskill only | **no skill list with levels** |
| Monsters | `monsters` (class, cell, hp/max, mode, boss kind, mod ids, level, unit id) | only the **current outdoor level** is documented, in the Den it worked too (V); no owner/pet link, no immunities text |
| NPCs, objects | `npcs`, `debug presets`, `debug objects`, `debug portals` | waypoint/chest/portal states partly (mode); no generic object list with open state |
| Ground items | `ground` (code, label, screen xy, unit id) | no quality/ilvl, label text only |
| Inventory/belt/equipment | `items` (code, loc, slot, q, size, grid at=) + tooltip text | stash and cube contents are not clearly included (U); no structured affixes (tooltip text only) |
| Missiles | `state missiles=<count>` only | **no missile list** |
| Collision / map | `debug grid n`, `debug blocked x y`, `debug warps` | usable, better than our map reader, but per request (one verb per grid) |
| UI panels | `state` fields (inv via `items panel=`, store=, menu=, tree=) | partial; `key i` opens panels by SDL key name |
| Quests | `debug quest N show` for Act 1 and other acts' words in the same line (V for Act 1 only) | **no whole-quest-word dump for all acts** |
| Hover | `state hover=` | one int, meaning U |
| Time | `info frame=` | no game tick clock for waiting |

Coordinates: d2d cells (float, 5 subtiles per cell) vs our integer subtiles: `sub = floor(5*cell)`. Screen 800x600 like ours; the projection is the same iso (D2D) but our `Transform` calibration (player feet at (400, 284)) was not re-measured on d2d: **U**, a one-hour test.

Input: map `Controller` calls onto `click`, `rclick`, `move`, `key <SDL name>`, `text`, `wheel` (all V as verbs; I only used `click` and `key Escape` / `key Return`). Missing: held buttons and modifiers (shift-click, ctrl), `hold=MS`, key down/up separately, mouse position read-back. Better: a second, semantic path through `cmd move|skill|interact|pickup|belt` (unit ids, cells) that needs no pixel math, but see the quirk in section 4.

What is risky:
- **Fidelity.** d2d says "behaviour copy" and lists its own deviations. For bot testing, the monster AI, skill timings, pathing and drop tables are all re-implemented and may differ from the original in ways nobody has measured; anything we tune on d2d needs a final check on Game.exe. U.
- **Unfinished parts** (their README table): Act 5 uber levels' monster groups not exact; Joining/multiplayer partial; dedicated server not started; 494 `ponytail:` shortcuts in the source (their count on 2026-10-29).
- **Unauthenticated dev socket.** It can write files through `screenshot <path>` and drive the game. Keep it an owner-only unix socket in a private dir (V: owner-only by default) and never build it into anything shared. d2d's own doc says never enable it in shipped builds.
- **Third-party churn.** Fast-moving project (their repo had commits the same day); the channel verbs change; pin a commit and keep our adapter small.
- **Licence.** GPL-3.0: keep it out of our repo; talking to a separately built binary over a socket is fine; copying its logic into our Go code would not be.
- **Cheats are free** (`debug stat`, `debug kill`, `debug goto`, `debug equip`, `debug difficulty`) with no ledger: a bot test harness could rely on them, but results would not transfer to cheat-free runs.

## 8. Recommendation and effort

- **Now (cheap, worth it):** use d2d as a *rules oracle and sandbox for experiments*: run it on demand on the worker host, read its source docs (`docs/research/re/*.md`, GPL text stays in their repo) for game rules, and use the `ctl.py`-style channel for one-off checks (death penalty, quest bits, hire list). Cost: nothing more than this spike.
- **Maybe (about 3 to 5 days):** a thin Go `d2dctl` package: client for the channel + `Reader` that fills a *partial* `d2.Snapshot` (player, monsters, ground, NPCs, items, collision grid). With that, the **town/NPC/merc/quest-flow logic** of the bot (internal/town, internal/quest, internal/nav on d2d's `debug grid`) could be exercised on any machine without Wine. Needs the `Env.Read` interface refactor (1 day) and the missing verbs below.
- **Not now (2 to 3 weeks, high risk):** running the full farming/combat bot against d2d. Combat timing and AI fidelity are the point of our bot and the thing we cannot trust from the spike.
- Verbs worth asking upstream for (or patching in a private fork, never in our repo): a full stat dump, skill list with levels, missile list, structured items with affixes, a quest-word dump for all acts, modifier keys for `click`, a `--speed N` time scale and a vsync/no-limit headless mode, a `cmd move` that does not need a priming click.

## 9. Deviations that matter for bot testing (task 7)

From d2d's `docs/research/re/deviations.md` (D2D, all visible for a bot unless noted):

- **Gold is picked up by walking within half a cell** (autoloot, on by default; `--toggle autoloot=off` restores clicking). A bot's "click gold" step is unnecessary, and its pickup logic is hidden from tests. **V** (my gold came back when I walked to the corpse).
- **Roofs turn transparent around the player** and the light grid is larger: only matters for screen-based vision.
- **An NPC's window closes after the player is 2 cells away** (original: the panels cover the screen so you cannot walk away). A bot that walks off with a store open behaves differently.
- **Palette blends, lit tiles, shadows** are approximations (pixel-level): matters for any template matching of the screenshot, not for state reads.
- **Town portals open at a slightly different spot** (0.6 cells south of the caster; original: nearest free spot of size 3).
- **World loads behind the title** (the usual save is built before OK is clicked): load times are shorter than Game.exe's.
- **Co-op/gamepad/widescreen/TCP join** are additions, off by default or irrelevant.
- **Level-save differences I saw:** the map is re-generated at each game start from the seed (same as the original); ground items and gold piles are not saved (probably the same in the original, U); corpses are relocated to town (D2D, V).

From `bugs.md` (original's own bugs, d2d copies them): the interesting ones for us are the unchanged ones: #6 (Den entry sets quest bit in a way that bit 3 can only come from the other branch), #22 (Zakarum priests heal the last unit found, not the weakest), #23 (radius search of will-o-wisps takes every unit of the near rooms), #26 (the Uber Baal/Mephisto/Diablo AIs are empty in 1.14d, the realm event is not in single player), #34 (missiles aimed inside their own room vanish short of range). `unverified.md` lists the few places d2d took behaviour from other projects instead of tracing Game.exe (DCC/DS1 reader details, item bit layout, Throw level, converted-monster hit test); none affects state reads.

## 10. What to do next

1. Decide whether the oracle use is enough. If yes: stop here, keep `~/d2d-lab` for the next question (set `XDG_CONFIG_HOME` as in `start.sh`, never run it against our saves).
2. If a partial adapter is wanted: write `internal/d2dctl` (client + parsing of `state/monsters/npcs/ground/items/menu`), introduce a `Reader` interface for `Env.Read`, and run the town/NPC/quest-flow tests against it. Measure the `Transform` offsets on d2d first (1 hour, `screenshot` + `ground` screen coordinates).
3. Ask upstream (their Discord/issues) for the missing verbs, or hold a private patch outside our repo.
4. Test the free Rogue case with a character that has no merc: new char, `debug stat 12 8`, kill Blood Raven (`debug goto 17`, `debug kill`), talk to Kashya (`click` the `talk` line).
5. Verify two of d2d's rules on the real game before trusting them: the death gold formula (8% at clvl 8) and the corpse-to-town relocation.
6. Check `cmd move` ignored-after-warp quirk with upstream; it silently loses commands.

## Reproduce

```
ssh worker
mkdir -p ~/d2d-lab && cd ~/d2d-lab
git clone --depth 1 https://github.com/psi29a/d2decomped src && (cd src && git rev-parse HEAD)   # e36922e...
cd src && cmake -S . -B build -G Ninja -DCMAKE_BUILD_TYPE=Release -DD2D_WERROR=OFF
nice -n10 ninja -C build -j6 d2d
mkdir -p ../data && cp "<game dir>"/{Patch_D2,d2char,d2data,d2exp,d2music,d2sfx,d2speech,d2xmusic,d2xtalk}.mpq ../data/
export XDG_CONFIG_HOME=~/d2d-lab/home/config XDG_DATA_HOME=~/d2d-lab/home/data
nice -n10 build/apps/d2d/d2d --data ../data --headless --devctl ~/d2d-lab/d2d.sock --no-video --seed 3 --start-screen ingame --start-class 1 --start-name Spike &
python3 ~/d2d-lab/ctl.py ping info "screenshot ~/d2d-lab/s.png" state npcs monsters "debug quest 1 show"
```
