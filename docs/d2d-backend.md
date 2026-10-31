# d2d as the primary backend: adapter design (agent D2DADAPTER, 2026-10-30)

> status: design doc, decided and built: d2d is the primary backend (ADR 0009, 2026-10-30; docs/plan/STATUS.md); the adapter is docs/d2d-bot.md and docs/d2d-wire.md, the gate is openspec/specs/fidelity-gate (20 % on four metrics, not "every row").

Goal: decide how our Go bot talks to d2d (https://github.com/psi29a/d2decomped, GPL-3.0, native reimplementation of 1.14d,
local control channel) and plan the work. Docs only; nothing here was run against a game, and nothing was sent upstream.

Tags: **V** = read by me in OUR code or docs/d2d-spike.md (the spike observed it live on d2d); **D2D** = d2d's own
docs/source say so (read by me in `~/d2d-lab/src`, commit `e36922e`, read-only, or via docs/sme/d2decomped-findings*.md);
**U** = unverified or my inference. Classic 1.14d only; d2d has D2R-style extras (widescreen, gamepad, co-op), all irrelevant.
Licence rule (docs/decisions.md 2026-10-30): nothing of d2d's code or text is copied here; verb names and our own words only.

## 0. Summary and recommendation

- The channel already carries about 60 % of what the decision code reads (position, life/mana, level, exp, gold, monsters,
  NPCs, ground items, item lists, NPC menus, collision grid, warps). The rest is missing or lossy (section 1.3).
- **Recommendation: (c) + (d) as a small, additive fork** (`d2d-bot` patch set kept PR-shaped so (a) stays possible):
  one new read verb `dump` returning one JSON document of the whole player/game state, one `step`/`clock` pair plus a
  `--time-scale`/`--lockstep` main-loop option, and a handful of tiny fixes (click modifiers, move-after-load quirk).
  Do not plan on upstream accepting or timing any of it; do not build on a sidecar (section 4 explains why a sidecar cannot read
  C++ memory).
- Effort: about 6 to 8 agent-hours of C++ plus 2 to 3 agent-days of Go for a read-only adapter that runs the town/quest/nav
  logic, 4 to 6 agent-days total to reach "bot farms on d2d". Fidelity gate is separate (section 5).
- Licensing: the Go bot stays a separate program (socket only), not GPL. A fork is GPL-3.0 and its source must be offered to
  whoever receives the binary (container included). Details in section 4.5.

## 1. Snapshot field map

Our decision code reads `d2.Snapshot` (internal/d2/snapshot.go, V): `State, Player, Inventory, Monsters, Objects, Missiles,
Ground, UI, Mouse, Screen, Viewport, Hovered`. It is built by `d2.Reader.Snapshot()` from raw Game.exe memory
(`mem.Source.ReadAt`); `internal/mem.Snapshot` is only a recorded page image used by tests (V), not this struct.
Seam: `env.Env.Read` is a concrete `*d2.Reader` (internal/env/env.go, V); an adapter needs a `Reader` interface (section 6).

### 1.1 Coordinates

| Space | d2d | Ours | Mapping |
|---|---|---|---|
| World | float **cells** (x.y, 5 subtiles per cell; town 57x41 cells) V (spike 4) | integer **subtiles** (`Unit.X/Y`, `Player.X/Y`, `input.Point`) V | `sub = floor(5*cell)`; back: `cell = sub/5`. `debug grid 5` is the subtile grid V. Act origin: `debug level` returns the act-tile origin (`wx wy`) V; ours are absolute act subtiles, so add 5*origin only if d2d cells are level-relative (U: spike shows player 30.70,13.70 in town vs level 57x41, so cells look level-relative; check against our `d2map` origin on the first probe). |
| Screen | 800x600 game frame; monsters/npcs/ground return feet pixels while on screen D2D | 800x600 client, `input.Transform` Calibrated (player feet at 400,284) V | Same iso projection (D2D) but our calibration was not re-measured on d2d U (1 h: `screenshot` + `ground` sx/sy). Better: stop using pixels for world actions (section 2). |
| Unit identity | `#<unit id>` in monsters/ground; `index` for NPCs; `cmd skill/pickup` take unit ids | `Unit.ID uint32` | Same role. Index vs id for NPCs is separate (npcs index feeds `cmd interact`). |

### 1.2 Field-by-field (every field the bot reads)

"Today" = channel verb that carries it now. Gap: M = missing, L = lossy, D = derivable.

| Our field (type) | Used by | d2d today | Gap |
|---|---|---|---|
| `State` (loading/menu/ingame) | everything | `state` field `screen=` (ingame, charselect, title...) V; `info frame=` | D: map screen names to State. "Loading" = reply error or screen change; U exact names. |
| `Player.Name, Class` | leveler, goals | `state class= name=` V; class index is UI order, NOT d2s byte V (spike: `--start-class 1` = Necromancer) | L: build a remap table; verify all seven. |
| `Player.X, Y` | nav, combat, town | `debug players` (x y cells) V; `state` | D: x5. Sub-cell fraction kept (float). |
| `Player.Mode` | combat | `state pmode=` V | L: d2d pmode numbering vs D2 mode numbers U; probe table needed. |
| `Player.Area, Act` | nav, town | `debug level` (level id, act origin) V | D: act from level id. |
| `Player.Life/Mana/Stamina (+Max)` | combat, town, potions | `state life=45/45 mana= stamina=` V | none; stamina is 8.8 fixed like ours V. |
| `Player.Level, Exp` | leveler, goals | `state level= exp=` V | none |
| `Player.Gold, StashGold` | town, goals | `state gold=` V; stash gold not in `state` V | M: stash gold. |
| `Player.Strength/Dex/Vit/Energy` | leveler (stat alloc) | `state str=` only str shown V | M: dex, vit, energy, unspent points (`skillpts=` exists in `debug quest` line). `BaseAttrs` (base vs gear) also M. |
| `Player.StatPoints, SkillPoints` | leveler | `skillpts=` appears in `debug quest 1 show` V; `state` skill points D2D | L |
| `Player.Resist` | gear gate, potions | `debug attack` res=f/l/c/p V (clamped? U), magic res not shown | L: no magic res, no pre-penalty value. |
| `Player.Stats` (map id->value: FCR, FRW, gold, stat 67/68...) | combat (FCR), leveler | `debug attack` has ias, frw, fhr, fbr, fcr U (field list: ias wsm frw fhr fbr; no `fcr` in D2D doc) | M: generic stat map. |
| `Player.States` (poison, buffs) | potions, combat | none | M (poisoned is decision-critical for antidote use). |
| `Player.Skills` (id, level, bonus) | leveler, combat | `state lskill= rskill=` only V | M: skill list with levels and bonus. |
| `LeftSkill, RightSkill` | combat | `state lskill rskill` V | none |
| `Player.Difficulty` | gates | `state difficulty=` V | none |
| `Player.Curses` (via states) | combat | none | M (part of States). |
| `Monsters[]` class, X/Y, Mode, ID | combat, nav, loot | `monsters` V: class id, cell x y, hp/max, mode, boss kind, mods, level, name, unit id, anim row | D: cells x5. L: mode strings (`NU`, `DD`...) not our numeric modes U; only the current level V. |
| `Monster.LifePct` / Life/MaxLife | combat | hp/max in `monsters` V | none |
| `Monster.Flags` (champ/unique/minion) | combat target choice | boss kind column (champion/unique/superunique/minion or -) D2D | D: map to flags. |
| `Monster.Enchants`, `UniqueNo` | combat (immune checks) | MonUMod ids (mods column) D2D | D: map ids to our enchant list. |
| `Monster.States` (curses, minion mark) | friendly filter | none | M: no child states. `pets=` count in `state` D2D. |
| `Monster` stat list -> resist refinement (`refineResists`) | combat | none | M: per-monster resist/immunity (reads Game.exe heap, section 6). d2d knows it internally. |
| `Unit.OwnerType/OwnerID` | friendly filter, merc | none | M: no owner link. Our merc finder is by class anyway V (internal/d2/merc.go). |
| Merc (`FindMerc`): class, id, pos, life | merc policy | `state merc=35.5,16.1:RG` V (cell, class code); merc also shows as monster? U (spike says not in a unit table) | L: no merc life/level/aura/name; need verb. Hire offer rows come from `menu` V. |
| `Objects[]` (waypoint, portal, chest, shrines) | nav, town | `npcs` (named NPCs and objects), `debug presets/objects/portals` V | L: open state only via `mode`; no uniform object list with class ids. |
| `Missiles[]` | combat (dodge) | `state missiles=<count>` only V | M: list. Low priority: our dodge code is optional. |
| `Ground[]` item units (class, pos, flags) | loot | `ground`: code, label, screen xy, unit id V | L: no cell x y (screen only, but `npcs` has cells; add), no quality/ilvl/flags; label text is what the player would read, enough for the loot filter if parsed U. |
| `Inventory.Items` + `Container()` (inventory, stash, cube, belt, equipment) with `Item` (Quality, Flags, ILvl, FileIndex, Body, Page, GX/GY) | town, gear, loot, leveler | `items`: `[code loc slot q panel size at]` + tooltip lines V | L: `loc` is 0 inv/1 body/2 belt (V from the spike lines), stash and cube not clearly present U; ilvl, identified, ethereal, socketed only through tooltip text; unique/set index missing. |
| `Inventory.Cursor` | town | `state held item` D2D | L |
| `Inventory.ItemCount` | town | `state items=` V | none |
| `Town.readOwner(item)` (owner unit via item data) | town (ownership of shop items) | none | M: memory read; replace by "item is in our list" (U whether this check is still needed). |
| `UI` panels (`UIState` flags: inventory 1, char 2, tree 4, npc menu 8, esc 9, shop 0xC, quest 0xF, waypoint 0x14, stash 0x19, cube 0x1A, merc 0x24...) | CloseAll, town, nav | `state` fields `menu=`, `store=`, `gamemenu=0:0`, `tree=`; inventory via `items panel=` V partially | L: synthesize `UIState` from these; waypoint panel, quest log, stash, cube, party flags unknown U. |
| `Mouse`, `Screen`, `Viewport` | input transform, Bind | `info w= h=`; no mouse readback; viewport not exposed | M: mouse pos (only used for Bind/hover), viewport (only for pixel clicks; drop with semantic input). |
| `Hovered` (type, id under pointer) | `Interact` (hover then click) | `state hover=` one int, meaning U | not needed if input uses `cmd interact/pickup/skill` with ids (section 2). |
| Quest words, all acts, all difficulties (`d2.PlayerProgress.Quest[3][QuestWords]`, via questmem) | merc policy (Blood Raven done), quest, goals, gates | `debug quest <q> show`: 16 flag bits plus Act 1 record states `den,burial,tools,cain,tower,andy` V | M for acts 2-5 (word dump), D2D lists the quest verb by act index. Bits (16 per quest) exist and are enough to compute our "done" tests for Act 1 V. |
| Waypoint masks (`PlayerProgress.Waypoints`) | nav (which WPs known) | `state waypoint=` V (meaning: panel? U) | M: mask per difficulty. |
| Map collision (`d2map.Tracker`: grids from game memory, room tracking) | nav pathing | `debug grid 5` rows `.`/`#`, `debug blocked x y`, `debug warps` V | L (one request per level, whole level incl. unexplored; real game reveals rooms as you walk: nav assumptions about unexplored rooms differ, U). Better than our reader (full level at once). |
| Level id/size/origin | nav | `debug level` V | none |
| Ticks/time (frame counter) | waits, timeouts | `info frame=` V | L: no game-tick or ms clock verb; needed for lockstep (section 4.4). |
| Canary/health (`internal/canary`) | soak health | `ping` | D: ok/err plus `info frame` advancing. |

### 1.3 What is MISSING (priority for new verbs)

1. Full player stat vector and states (strength through energy, resists, FCR/FRW/IAS/fhr, poison state, curses).
2. Skill list with level and bonus per skill.
3. Inventory/stash/cube/belt/equipment as structured items (ilvl, quality, flags, unique/set index, socket count, affixes).
4. Quest words of all acts and difficulties, waypoint masks.
5. Per-monster resists/immunities and child states (curses, minion mark); owner links; merc state (life, level, aura, name).
6. Missile list (id, owner, position), objects with class id and state.
7. Panel open flags (waypoint, quest, stash, cube, party).
8. Click modifiers (shift, ctrl) and key down/up; mouse readback.
9. Game clock and a stepping control (section 4.4).
10. Ground items with cell position and item flags.

Derivable on our side: unit modes (table), enchant flags from mod ids, difficulty penalties, potions in belt by tooltip text,
"is merc alive" from `merc=` presence.

## 2. Input map

Our `input.Controller` (internal/input/controller.go, V) is built on pixel clicks plus hover verification. d2d accepts both
pixels (`click x y`, `rclick x y`, `key <SDL name>`, `text`, `move x y`, `wheel`) and semantic commands (`cmd move X Y`,
`cmd skill <id> X Y [unit] [left] [item]`, `cmd interact <npc index>`, `cmd pickup <unit>`, `cmd belt <slot> [merc]`,
`cmd select <skill> <left>`, `cmd resurrect`, `save`, `quit`) V from the spike.

| Controller verb | Today on Game.exe | d2d mapping | Gap / note |
|---|---|---|---|
| `MoveMouse(x,y)` | helper `move x y` | `move x y` (hover) V | fine, rarely needed. |
| `Click(b,x,y,mods...)` | helper click, mods shift/ctrl | `click`/`rclick x y` V; **no modifiers** V | M: `click x y [shift] [ctrl]`. Needed: shift-click belt/merc potions (workaround `cmd belt n merc` V), shift+left to attack in place, ctrl for item moves U. |
| `Key(name)` | helper key names (a-z, esc, f1...) | `key <SDL name>` V (`Escape`, `Return`; names differ: ours `esc`, `enter`) | map table. `key shift` is an error V. |
| `Hold/Release(key)` | held keys (run toggle, force move) | none | M: key down/up, only if the bot needs held run; `cmd move` reduces need. |
| `TypeChar` / OOG typing | WM_CHAR through helper | `text <s>` D2D | none |
| `Pos()` | memory | `debug players` | adapter. |
| `ClickWorld(w)` / `MoveTo(w,opts)` | screenOf + click, re-click until arrive | `cmd move X Y` (cells) V; arrival by polling `debug players` | quirk: `cmd move` ignored after a load or `debug goto` until one `click` is sent V; add a priming click after every level change until fixed (spike 4). Path stops short on long routes V, so keep the re-issue loop. |
| `SelectSkill(hotkey)` | F-key hotkey bound by runner (`Deps.Prepare`) | `cmd select <skill> <left>` (skill id directly) V as verb, not exercised by spike | better than hotkeys; Prepare (bind hotkeys) becomes a no-op. |
| `CastAt(hotkey, w)` | select + right-click pixel | `cmd skill <id> X Y [unit] [left]` V (killed a monster with Teeth) | pass unit id for targeted skills; U for ground-targeted (teleport, blizzard) and for left-vs-right semantics. |
| `Interact(target)` | hover until `Hovered` matches, click | NPC: `cmd interact <npc index>` V (walks, may stop short; repeat); monster/object: `cmd skill` with unit id or `click` on screen xy from `monsters`; item: `cmd pickup <unit id>` V; waypoint/chest/portal objects: `debug operate <i>` is a cheat; real path = `click` on the object's screen xy from `npcs`/`debug presets` U | M: object interact by id (waypoint, stash, chest, portal). Candidate verb `cmd interact obj <id>`. |
| `CloseAll()` | Esc until no panel | `key Escape` toggles the game menu V; read `gamemenu=`, `menu=`, panel flags first | needs panel state (1.2 UI row). |
| `HoldClick(b,x,y,d)` | melee chain (Zeal) | `click` is press+release; `cmd skill ... ` repeated | M: held button or an "attack-continue" mode; Zeal-style skills U. |
| Potions | key 1-4 or click belt | `cmd belt <slot> [merc]` V | direct. |
| Buy/sell/repair/gamble | NPC menu pixel clicks, vendor panel clicks | `menu` gives each line a click point V; vendor grid by pixel from `items panel=` U | store panel item positions: `debug itemat <id>` (cheat-flavoured) gives box middle V as a verb, usable read-only U. |
| Hire merc | clicks on hire window rows | `menu` rows (name, lvl, life, def, cost) V; `click 400 164` hired V | none. |
| Take corpse | walk onto corpse | `cmd interact -3000` V (clicking the body failed V) | adapter-only. |
| Hotkey binding | `Deps.Prepare` | not needed | skip. |
| Tomes: TP/ID | right-click item | `rclick` on item pixel from `items` U | needs `at=` grid -> pixel; `debug itemat` helps. |

Decision: build `internal/d2dctl` (V: D2DCTL builds it in parallel) as client plus parser, then a **semantic controller**
`d2dinput.Controller` implementing the same `Controller` interface plus a few new optional interfaces (`UnitCaster`,
`NPCInteractor`) so combat/town call ids instead of pixels. The pixel path stays only for menu lines and vendor grids.

## 3. Lifecycle

Our flows (docs/launch.md, docs/oog.md, V): start Game.exe under Wine/Proton, start the d2input helper, walk the OOG
screens by reading D2Win control lists in memory and clicking pixels, Save and Exit returns to the main menu, "Exit Diablo II"
ends the process, a crash keeper restarts it, sandbox/subreaper/ptrace handle readability. All of it is Wine-specific.

| Step | Game.exe today | d2d |
|---|---|---|
| Start | `launch.Game.Launch` via Proton, helper, Xvfb, subreaper | `d2d --data <mpq dir> --headless --devctl <sock> --no-video [--seed N] [--no-save] --start-screen <screen> --start-class N --start-name X [--start-hardcore] [--start-classic] [--log-level]` V. Env: `XDG_CONFIG_HOME`/`XDG_DATA_HOME` isolate saves V. Plain `exec.Cmd` of a native binary: no Wine, no Xvfb, no subreaper, no setcap. |
| Per-instance isolation | prefix per instance, helper port 47012+N | one socket path and one XDG dir per instance (cheap) V. |
| New character | OOG: create screen clicks, class portraits, name typed | `--start-screen ingame --start-class N --start-name X` creates and enters at once V. Class index is UI order V. Hardcore/classic via flags V (softcore expansion default). Ladder: none. Difficulty: Normal on new; `debug difficulty` is a cheat (changes game); real progression to Nightmare unlock U (the d2s difficulty bytes) — not tested. |
| Select existing | char select slots clicked, header read | `--start-screen charselect --start-name X` then `key Return` (6 s) V. Verb to select by name from the running process: none; relaunch is the flow. |
| Enter game from title | splash, Single Player, select | `--start-screen` skips it V. |
| Save | Save and Exit through Esc menu click (400,261) | `save` verb V (writes the character file now, as leaving does). |
| Exit to menu / next char | Save and Exit returns to main menu, process stays | `quit` verb: main loop exits (`town.save()` on exit when in game D2D). No in-process return to charselect via verb; use quit+restart (6 s) U whether `key Escape` + click on "Save and Exit" works to reach charselect V-partial (spike used quit). |
| Stop | SIGTERM/SIGKILL, `wineserver -k`, reap zombies | `quit` then SIGTERM; no children. |
| Crash detection | Monitor on CPU ticks flat 45 s, BlizzardError.exe | process exit code + socket `ping` timeout + `info frame` not advancing (d2d logs "MAIN STUCK N ms"; also has its own heartbeat thread D2D). Crash reports: leave `--send-crash-reports` off. |
| Death/respawn | OOG none (in game) | in game; same as ours |
| Difficulty screen | UNVERIFIED on Game.exe | not an OOG screen in d2d for existing chars U |
| Seed | not controllable on Game.exe (we may not launch games with `-seed`) | `--seed N` fixes the map layout V; monster rolls differ per run V |
| Multiplayer/LAN (our `Flow.HostGame/JoinGame`, party) | TCP/IP screens | d2d has co-op via pads and a join mode to a game.exe host (`--join`) D2D; our party tests would need separate design. Out of scope for the first adapter. |
| Cheats/ledger | `internal/cheat` writes Game.exe memory/saves under a ledger | `debug stat/kill/goto/...` are free and unlogged V. A d2d test harness may use them; results with cheats do not transfer (spike section 7). Keep a ledger entry in our code whenever the harness calls a `debug` verb. |

Gotchas to design for: one client at a time on the socket (`err busy`) V, so the bot, the sidecar and any probe tool must
share one connection (a mux inside the adapter, or the sidecar goes through the bot's API); socket permissions owner-only V.

## 4. The decision: how do we get the missing verbs

The missing data lives in d2d's C++ memory. Our Go process cannot reach it except through the channel (or by reading a
file the engine writes). That decides the options.

### 4.1 Options

| | (a) Upstream PRs | (b) Our fork, small patch set | (c) `dump` verb (JSON of all state) | (d) fast-forward |
|---|---|---|---|---|
| What | contribute verbs as PRs | private fork rebased on upstream | one new read verb; usable via (a) or (b) | virtual clock plus `step` |
| Control | none over timing or acceptance | full | n/a (it is the content) | n/a |
| Effort | PR writing, review cycles, unknown latency | initial 6-8 h; sync about 1 h per month U | 4-6 h of C++ for the schema below U | 2-4 h U (section 4.4) |
| Risk | rejected, rewritten, or reverted; verb set churns under us | drift; merge conflicts if we touch shared files | schema creep | breaks pacing-dependent code; audio/video clocks |
| Licence | PR is GPL-compatible by definition | fork must publish source to recipients | same as host | same |

A sidecar (a separate process) can only call the same verbs, so it cannot add data; it is only useful for combining existing
verbs (the adapter in Go already does that). Rejected as an answer to "missing verbs". Reading d2d's memory from outside
(like we do for Game.exe) is not an option: C++26 object layouts, no stable addresses, GPL implications, and it throws away
the one thing d2d gives us (a deliberate state interface).

### 4.2 Recommendation

**(b)+(c)+(d) with (a)-friendly patches.** Concretely:

1. Fork `psi29a/d2decomped` on our forge (private) as `d2d-bot`; pin to a commit (we start at `e36922e`).
2. Patch set P1: `dump` (and `dump <section>`), P2: `clock`/`step`/`--lockstep`/`--time-scale`, P3: small fixes
   (`click ... shift|ctrl`, move after load, `unit` verb for one unit's stat list, `waypoints`/`quests` inside dump).
3. Keep patches **additive**: new files `apps/d2d/devctl_dump.cpp` and `devctl_clock.cpp` plus one-line registrations in the
   existing verb table; touch main.cpp only at the clock seam (4 call sites). Any change to shared files stays below ~50
   lines and each patch is one commit with a message that reads like a PR description (section 4.6).
4. Sync policy: **rebase onto upstream main at most monthly, or when upstream changes the verb table/main loop that we
   touch.** Drift allowed: at most the patch set (about 500 to 800 lines in new files, under 100 in shared files). If a rebase
   needs more than a day, stop and reduce the patch (move logic into the new files). Run their `tests/smoke_d2d.py` and our
   fixture contract test (section 5, Phase 0) after every rebase.
5. After our patches work, offer P1 (dump) upstream as PRs **only if the user decides to** (the user reviews the drafts in
   4.6). Never required: we run on the fork either way.

Reasons: the bot needs structured state far more than it needs d2d to merge anything; a dump verb is one place to look when the
schema changes; our soak statistics need fast-forward (the SME's biggest lever: statistics take wall-clock hours on Game.exe,
about 0.6 to 0.9 core per instance at real speed on d2d V); and a fork is the only option whose schedule we control.

### 4.3 Proposed `dump` schema (our design)

Request: `dump [sections]` where sections is a comma list of `player,skills,stats,items,quests,waypoints,monsters,units,ground,objects,missiles,ui,merc,time` (default all but `units`).
Reply: one line of UTF-8 JSON, then `ok` (a single line avoids breaking the line-oriented framing; the 64 KiB line cap D2D means
`items`, `monsters` and `stats` may exceed it, so the verb must either raise the cap for this reply or split sections:
`dump player` and `dump items` as separate calls. Our adapter requests per section; a full dump is for fixtures.)

```
{
 "v": 1, "build": "<git sha>", "frame": 123456, "game_ms": 2055000, "tick": 51375,
 "screen": "ingame", "difficulty": 0, "hardcore": false, "expansion": true,
 "player": {
   "id": 1, "name": "Soaky", "class": "sorceress",          // names, not UI indices
   "level": 12, "exp": 6543, "gold": 1200, "stash_gold": 300,
   "act": 1, "area": 2, "x": 40.3, "y": 22.1,                 // cells, float (level-relative; origin given below)
   "origin": [864, 912], "level_size": [57, 41],
   "mode": "walking", "life": [88,120], "mana": [20,40], "stamina": [100,120],
   "stat_pts": 0, "skill_pts": 1, "lskill": 36, "rskill": 54,
   "stats": { "str": 20, "dex": 25, "vit": 20, "ene": 30,
              "base": { "str":10, "dex":25, "vit":10, "ene":35 },
              "res": {"fire":0,"cold":0,"light":0,"poison":0,"magic":0},  // before difficulty penalty
              "fcr":0,"frw":0,"fhr":0,"fbr":0,"ias":0,"mf":0,"gf":0, "ar":80, "def":6, "block":0, "dmg":[2,4] },
   "all_stats": {"<stat id>": <raw value>},                   // every stat as the engine holds it, ints (x256 kept)
   "states": [ <state id>, ... ],                             // states.txt ids: poison, buffs, curses
   "skills": [ {"id":36,"lvl":4,"bonus":1,"hand":"left"} ]
 },
 "merc": {"present":true,"alive":true,"class":271,"unit":812,"x":35.5,"y":16.1,"life":[63,63],"level":5,"name":"Aliza","aura":0},
 "items": [ {"id":9,"code":"wnd","loc":"body","slot":4,"x":0,"y":0,"w":1,"h":2,"quality":2,
             "ilvl":3,"flags":{"identified":true,"ethereal":false,"socketed":false,"runeword":false},
             "unique":0,"set":0,"sockets":0,"qty":1,"dur":[20,20],"name":"Wand","tip":["Wand","One-Hand Damage: 2 to 4"]} ],
   // loc: inventory|stash|cube|belt|body|cursor|vendor (open store)|ground
 "ground": [ {"unit":771,"code":"gld","x":40.9,"y":11.5,"qty":8800,"quality":0,"flags":{},"label":"Gold"} ],
 "monsters": [ {"unit":752,"class":2,"x":12.1,"y":28.7,"hp":[2,2],"mode":"NU","level":1,"kind":"normal",
                "mods":[7],"name":"Fallen","owner":0,"states":[],"res":{"f":0,"c":0,"l":0,"p":0,"m":0},"immune":["fire"]} ],
 "objects": [ {"unit":40,"class":119,"x":..,"y":..,"mode":1,"name":"waypoint","operable":true} ],
 "missiles": [ {"unit":5,"class":6,"x":..,"y":..,"owner":1} ],
 "quests": { "diff": [ [ <u16 x 96> ], [..], [..] ] },       // quest words, 3 difficulties x 96 words, same layout as the .d2s block
 "waypoints": [ <u64 mask>, <u64>, <u64> ],                  // bit n = waypoint n, per difficulty
 "ui": { "inventory":false,"char":false,"skills":false,"quest":false,"waypoint":false,"stash":false,"cube":false,
         "npc_menu":true,"vendor":false,"esc_menu":false,"mini_panel":true },
 "menu": [ {"text":"Kashya","x":812,"y":484} ],
 "held": 0
}
```

Rules: numbers in cells for position, plus screen pixels where the engine already has them; ids not indices; every field
named after what the engine's data tables call it, with a `v` bump on breaking change; unknown sections return `err`.
The quest and waypoint layout deliberately mirrors the save file so our existing `saves`/`questmem` decoders run unchanged on
it (V: `internal/saves`, `d2.PlayerProgress` use the same word layout, docs/questmem.md).
Cost: reading is O(units); at one call per bot tick (about 4 Hz) the single-line JSON is small except `items` and
`monsters` (hence sections).

Mapping into `d2.Snapshot` (adapter in Go, fully offline-testable from recorded `dump` fixtures): `Player` from `player`,
`Monsters` from `monsters`, `Ground` from `ground`, `Inventory.Items` from `items` (`Container` from `loc`), `UI` from `ui`,
`Objects`/`Missiles` from lists, `PlayerProgress` from `quests`/`waypoints`. Missing in `dump` means `ok=false` fields in the
adapter, not zero.

### 4.4 Fast-forward (d): smallest change

What d2d does today (D2D, source read at e36922e, line numbers of `apps/d2d/main.cpp`):

- The main loop (`run_windowed`) takes `frame_start_ms = SDL_GetTicks()` (~754) and `now_ms = SDL_GetTicks() - start_ticks`
  (~1032); the world and UI advance from `now_ms` and `last_ms` (`town.update(..., now_ms, last_ms)` ~1365), with fixed
  40 ms steps inside (`for (; frame_ms - rain_ms >= 40; ...)` pattern in town.cpp; the server tick is 25 Hz D2D). The loop ends with
  `pace_frame(frame_start_ms)` (~1567), a `SDL_Delay` up to `kFrameBudgetMs = 16` (~419). A heartbeat thread watches
  real ticks (~726). 5 other `SDL_GetTicks` uses exist outside main.cpp (counted by grep). Headless still renders every frame
  with the software renderer (8 to 20 ms per frame V), which, not the pacer, is the real limit.
- So game time = wall time; there is no scale option V (spike 6).

Smallest design, in order of cost:

1. **Clock seam**: replace the SDL_GetTicks reads that feed game logic with one function `clock::now_ms()`. Default: SDL time.
2. **`--time-scale N`** (float; 1 = today): `virtual_ms = base + N * real_elapsed`. The world sees N times larger `now_ms - last_ms` per frame;
   if the world only consumes fixed 40 ms steps with a catch-up loop, large deltas become many steps per frame, which is fine
   as long as the catch-up loop has no cap U (check `town.update` before relying; this is the main unknown). Skip `pace_frame`
   when scale > 1. Limit: the frame loop must still finish N * 16 ms of game work in 16 ms of wall time.
3. **`--lockstep` + `step <n>`** (deterministic, what a bot wants): virtual time advances only when told: each `step n` runs `n`
   world frames of 40 ms back to back and replies `ok frame=<f> ms=<game_ms>` after the last. No pacer, no sleeping. The bot's
   Clock (our `env.Clock`, V) becomes the d2d game clock, so every `Sleep(d)` in behaviours is `step(d/40ms)`. This removes
   the race between "bot polls state" and "game moves" and makes a replay reproducible when combined with `--seed` (monster
   rolls come from d2d's own RNG, same odds, U for exact reproduction D2D).
4. **`--no-render` while stepping**: in lockstep mode, skip the render/present phases unless `screenshot` is pending; this is
   where the cost is (render 8 to 20 ms per frame V). Expected speed: game logic only is likely 20x or more real time U.
5. `clock` verb: `ok frame=<f> game_ms=<ms> tick=<t> scale=<N> lockstep=<0|1>` (read-only).

Watch-outs: audio and video use wall time (headless: `--no-video`, OpenAL null V); town/level-load code uses `std::async`
loading (`std::future<void>& loading` at main.cpp signature) which must finish in lockstep mode before stepping continues
(step blocks on the future); the "MAIN STUCK" heartbeat must use the same clock; save timestamps use wall time (harmless).
Estimated effort: 2 to 4 agent-hours for 1+2, another 2 for 3+4 U. If upstream already steps on a fixed dt internally, 3
may be a one-line change; this must be confirmed by reading `Town::update` before committing to the estimate.

What it buys: the soak (docs/soak-hosts.md hours) shrinks to minutes; 500+ kill drop samples, death-rate and XP/h
measurements per build become routine; recovery paths (death loops, full inventory) are exercised daily rather than weekly.
What it breaks: anything that depends on wall-clock behaviour of the original (timeouts that gate unlocks in the original
engine are game-time in d2d, so they scale; our own bot timeouts must be game-time too, via Clock).

### 4.5 Licensing

- d2d: GPL-3.0-or-later D2D (their README/LEGAL). Our Go bot talks to it over a socket, as a separate program; GPL does not
  propagate to it (U: not legal advice; interoperability by socket between separately built programs is the standard
  position). Do not paste their code, tables or doc passages into this repo (docs/decisions.md rule 3; this file is our own words).
- A fork is a derivative: patches we write against d2d code are GPL-3.0 and must be made available **to anyone we give the
  binary to** (source offer or the source itself, with our patches). Private use on the worker host needs no publication.
- **Container**: if a container image ships the d2d binary (or any user receives it), the image must carry or point to the
  corresponding source (our fork's commit) and the GPL text; build the binary in the image from the pinned fork commit so the
  offer is satisfied by the image's own recipe plus a published tag. The Blizzard MPQs remain the user's (as today: mounted, not
  in the image; docs/container.md).
- Their tree also pulls third-party code (StormLib, sentry-native/crashpad, CLI11, SDL3): each has its own licence to carry in
  the notices file if we ship; turn Sentry off (`--send-crash-reports` off by default D2D) and consider building without it.
- Our own licence decision stays deferred (docs/decisions.md): shipping a container that contains GPL code does not force the
  Go bot's licence, but mixing them in one image makes the licensing story harder to explain; keep them in separate layers and
  document that.
- Their `LEGAL.md` makes an interoperability argument and admits the scope is debated D2D; their position, not ours. Our MPQ
  data stays out of git and images as now.

### 4.6 Draft PR/issue text (for the user to review; NOTHING is posted)

Issue (feature request): *"Control channel: a read-only `dump` verb returning the whole player/game state as JSON."*
Body outline: we drive d2d as a bot test rig; `state`, `monsters`, `items` etc. are tab/space text with partial coverage; we
propose one versioned JSON document per section (player, stats, skills, items, quests, waypoints, merc, monsters, ui) so
tools need no per-verb parsers; read-only, no new engine state; section list and field names as in section 4.3 (rewritten as
their schema); we will write the code and tests (`tests/smoke_d2d.py` additions) if the shape is agreeable.

Issue 2: *"Deterministic stepping: `--lockstep`, `step <n>`, `clock`, `--time-scale`."* Body: headless tools cannot control game
time; propose a clock seam (one function replacing `SDL_GetTicks` in the main loop), `--time-scale`, and a stepping mode that
skips pacing and rendering; opt-in only; default behaviour unchanged; benefits for CI quest smoke tests (their smoke run takes
about 1:45 D2D) and bots.

Issue 3 (bugs seen in the spike, good first PRs): `cmd move`/`cmd interact` silently ignored after a level change or load until
a `click` is sent (V, spike 4); `click` has no modifier for shift-click; `debug goto 8` renders the Den black while `debug warps`
+ `debug enter` works.

All three would be filed under the user's decision only; the user decides authorship and account.

## 5. Phased adapter plan

Effort in agent-hours (1 agent-hour = one focused agent hour incl. tests; estimates U). Fidelity gate (end of Phase 3 and 4):
the same character, same seed, both backends, replicated runs (at least 3 each), compare death rate, XP/h, gold/h; a gap
above about 20 % on any of them is a fidelity bug to investigate before trusting d2d for tuning (docs/plan/plan-v2.md section
11 principle). Variance is high (docs/plan/plan-v2.md section 11: judge on rates, not one run), so the gate is about means and
CIs, not a single run.

| Phase | Work | Acceptance checks | Effort |
|---|---|---|---|
| **0 Probe tools** (D2DCTL, parallel) | `internal/d2dctl`: socket client (one conn, line framing, timeouts, `err busy`), parsers for `state, monsters, npcs, ground, items, menu, debug level/players/quest/warps/grid`, a recorder that stores raw replies as fixtures. Contract test: replay recorded replies and compare parsed values. Measure the `Transform` offsets on d2d (1 h). | `go test ./internal/d2dctl` offline against fixtures; one live smoke on the worker host (spike's character) reproduces `state` fields; fixtures committed. | owned by D2DCTL (about 4 h) |
| **1 Read-only adapter** | Introduce `d2.StateReader` interface (Snapshot, Player, Units, UI, UnitStats, RawStatList...). `d2dsrc.Reader` builds `d2.Snapshot` from the channel (partial: section 1.2 rows marked none/derivable), unit mode/class mapping tables, `x5` subtiles, class remap, UI synthesis. Fields without a source return not-ok, bot code already tolerates (`BaseAttrs` returns ok=false). `env.Env.Read` becomes the interface; direct `Mem` readers get an optional nil path (section 6). | `go test` of the adapter with recorded fixtures (all D2DCTL fixtures); a replay test fills `d2.Snapshot` and the existing pure policy tests (merc, potion, town) pass on d2d-derived snapshots; live: `cmd/d2snap`-style dump of a snapshot on a running d2d matches `debug players`. C++ prerequisite: P1 `dump` (4-6 h C++) makes this 2 days instead of 3 to 4 days of parsing. | Go 20-24 h without `dump`; 12-16 h with `dump`; C++ 4-6 h |
| **2 Input adapter** | `d2dinput.Controller`: `Click` via `click/rclick`, `Key` map, `MoveTo` via `cmd move` + arrival polling + priming click, `CastAt`/`Interact` via ids (new optional interfaces in `internal/input`), `CloseAll` with synthesized panel flags, belt via `cmd belt`. Cheats go through the cheat ledger shim. | Offline: fake socket server test of every verb mapping; live smoke on d2d: walk town loop, talk Kashya, hire merc, buy potion, cast at a monster, pick up gold/item, take corpse, all without Game.exe. | 12-16 h Go; C++ 1-2 h (`click` modifiers, move-after-load) |
| **3 Lifecycle + soak harness** | `launch/d2d` flow: start/stop/restart native process, create/select character, save/quit, crash keeper (exit code, ping, frame advance), per-instance socket+XDG dir. `d2soak`/`soak24.sh` for d2d: samples from `dump`/`state` (level/exp/gold/deaths) instead of `/proc/pid/mem`; health from `ping` and `info frame`; no Wine reap checks; sidecar alert codes that mention Wine/ptrace are off. P2 lockstep/time-scale for fast soak. | `tools/soak` run of 30 game-minutes at time-scale 10 on one slot reports XP/h, gold/h, deaths, recoveries; d2soak analysis passes on its output unchanged except procs; 3 replicated runs on d2d vs 3 on Game.exe (same character spec) satisfy the fidelity gate for death rate, XP/h, gold/h within 20 %. | 16-24 h Go; C++ 4-6 h (P2) |
| **4 Container recipe** (SME holds until the A/B report) | Dockerfile building the pinned fork, `--headless --no-video`, MPQs mounted, socket path inside the container exposed to the bot process only, GPL source/notices in the image, non-root, no Xvfb/Wine layers. | `tools/container-checklist.sh` variant passes; one-slot 24 h soak in the container; image build is reproducible from a tag. | 8-12 h after the A/B report |

Critical path: P1 (dump) -> Phase 1 -> Phase 2 -> Phase 3. Phase 0 and the C++ patches can start together. Total about 6 to 8 agent-days
if all go well; a read-only town/quest/nav testbed (Phase 1 plus half of 2) is usable after about 3 agent-days.

## 6. Bot-side prerequisites (keep backend-agnostic)

Places that depend on Game.exe specifics (V: found by grep in this worktree, 2026-10-30):

| Dependency | Where | What to do |
|---|---|---|
| Concrete `*d2.Reader` in `Env` | internal/env/env.go (`Read *d2.Reader`, `Mem mem.Source`), `Env.Snapshot()` | Define `d2.StateReader` interface; `Env.Read` uses it; `Mem` becomes optional (`nil` on d2d). |
| Direct heap reads (not through Snapshot) | internal/combat/fighter.go (`refineResists` reads monster stat list at unit+0x5C, `readU32`); internal/combat/aura.go (`ActiveAura` walks stat lists via `e.Mem`; has a `auraFn` hook); internal/town/town.go `readOwner` (item owner via memory); internal/leveler/leveler.go `BaseAttrs` (base stat vector at p.Addr+0x5C), `UnitStats` calls | Move behind methods on `StateReader` (`MonsterResists(unit)`, `ActiveAura()`, `ItemOwner(item)`, `BaseAttrs()`); d2d implements from `dump` or returns not-ok (the code already falls back: `ok=false`). |
| Unit addresses (`Unit.Addr`, `Player.Addr`) as identity | d2 snapshot, combat caches (`resCache` keyed by ID), `d2.Snapshot`, `questmem` (`FindServerPlayer`) | Use `Unit.ID`; keep `Addr` zero on d2d; audit users with grep `\.Addr`. |
| Map reading from memory | internal/d2map (Tracker/Reader over `mem.Source`, room tracking, grid), cmd/d2mapdump | Provide `d2map.Source` interface fed by `debug grid 5`/`debug blocked`; Tracker's RescanEvery logic becomes "fetch on level change". |
| Quest/waypoint memory | internal/questmem, internal/d2/questmem.go (`PlayerProgress`, region scans over /proc maps, `RegionsOfPID`), cmd/d2bot/task.go (`memQuests`, `questmem.New(e.Read)`) | `PlayerProgress` source interface; d2d fills from `dump quests/waypoints`; save-file fallback stays. |
| /proc/pid/mem, ptrace | internal/mem/proc.go, internal/memfd, internal/input/procmem_linux.go, cmd/d2bot/game.go, cmd/oog/*, tools/live.sh grants setcap through worker-sudo | Not used on d2d. Keep behind `//go:build` or runtime backend selection; live.sh gets a `--backend d2d` mode that skips setcap. |
| ptrace readability / subreaper | internal/launch/subreaper.go, managed.go, sandbox.go, reap.go, zombies.go, recycle.go; `launch.BecomeSubreaper()` | Wine-only; skip for d2d. |
| Wine/Proton paths and prefix | internal/launch/{proton,config,instance,install,settings,mounts}.go, internal/paths, internal/setup/*, internal/data/extract.go | Backend switch in config; d2d needs `--data` MPQ dir only. |
| d2input helper TCP (47012+N) | internal/input/client.go (`Client`, `AddrFor`, `Bind`, `ResolveInstance`), internal/launch/helper.go, cmd/d2input, cmd/d2bot/live.go (`input.NewGame(NewClient(o.helper), input.Reader{Src: g.src})`) | `input.Controller` implementation for d2d; `Bind` (mouse readback handshake) not needed. |
| `input.Reader` (memory Player/Mouse/Viewport/Hover) | internal/input/state.go, controller.go `Game.R`, `HoverFunc`, `UIFunc` | Interface `input.StateSource`; semantic Controller does not need Hover/Viewport. |
| OOG screens via D2Win control lists | internal/launch/oog.go, oog_table.go (coordinates), oog_keeper.go, oog_wire.go; cmd/d2bot/behaviors.go (`launch.Flow{Mem: e.Mem...}`), setup/steps.go | `launch.Flow` becomes an interface (`Start`, `LeaveGame`); d2d implementation is process start flags plus `save`/`quit`. |
| UI panel flag array semantics | internal/d2/ui.go (indices 0x01..0x24), CloseAll in input | Keep our `UIState`; d2d adapter fills it from `dump ui`/`state`. |
| Hotkey binding | bot `Deps.Prepare`, input `SelectSkill(hotkey)` | `cmd select` by skill id; keep the hotkey layer behind the Controller. |
| Monitor/health | internal/launch/monitor.go (CPU ticks from /proc/pid/stat, BlizzardError.exe), internal/canary, cmd/d2soak/sample.go and winedev.go (procs, winedevice plateau checks), internal/metrics/proc.go | Backend-agnostic health interface: `Alive()`, `Progress()` (frame advance), RSS from /proc of the d2d pid (fine, native). |
| Cheats | internal/cheat/* (memory and save edits, ledger) | d2d `debug` verbs wrapped by the same ledger API. |
| Wine-only soak tools | cmd/d2soak/winedev*.go, tools/winereap.sh, tools/live.sh | Skip for d2d; separate soak recipe (Phase 3). |
| Tests/fixtures | internal/d2/testdata, mem fixtures (page images) | Add `dump` fixtures next to page snapshots (envelope kinds, internal/mem/envelope.go supports kinds, V). |

Rules for new code (as in docs/decisions.md rule 1): behaviours take interfaces (`StateReader`, `Controller`, `Clock`,
`Flow`), never `mem.Source`; no `/proc` reads in decision code; positions through `d2.Unit` not addresses; time through
`env.Clock`; anything needing Game.exe-only data is an optional capability with a documented fallback.

## 7. Fidelity-test checklist (SME list mapped to d2d probes)

All are tests of d2d against Game.exe; d2d can run them cheaply and fast (with Phase 3 time-scale), Game.exe is the
reference. "Both" means the same probe spec runs on both backends and numbers are compared (gap > 20 % = fidelity bug).

| SME item | d2d probe | Game.exe counterpart | Missing for the probe |
|---|---|---|---|
| Monster aggro and leash | place player at distance d from a sleeping pack (`debug warp`), step N ticks, record `monsters` mode/position per tick; vary d; measure wake distance and give-up distance | our live probe (docs/probes.md) with `mem` snapshots | tick clock (`clock`/`step`); `monsters` is current level only V; D2D notes say d2d updates monsters within 30 cells, more than Game.exe's active rooms (docs/sme/d2decomped-findings-2.md section 8): so aggro range on d2d is an upper bound, expect known divergence. |
| Regen (life, mana, stamina) | read `state` each second while idle after `debug stat 12 N` | read memory each second | per-second sampling in game time; stamina 8.8 V. |
| Drops over >= 500 kills | `debug kill` on a fixed monster class (counts as real kills with drops V) at fixed area level and difficulty; collect `ground` after each; classify with `dump items`/`ground` quality+ilvl; compare against drop tables and Game.exe farm logs | farm soak drops (our logs) and 500 kills of one boss | `ground` lacks quality/ilvl (dump P1); d2d's drop engine was diffed against game.exe's 100000/100000 D2D, so d2d mostly tests ours; real value is the Game.exe side. |
| Prices (buy/sell/repair/gamble) | open vendor, read `menu` rows and the `items` tooltips of stock; sell known items; repair cost; gamble costs by item type | vendor reads on Game.exe (our live `town` runs) | vendor restock after 4 min NOT implemented in d2d D2D: compare before restock only; `items` for vendor panel U. |
| Resist penalties by difficulty | `debug difficulty 1|2` then `debug attack` res | read stats on Game.exe | `debug attack` clamps? U; magic res missing; `dump` all_stats fixes. |
| Hit recovery / FCR / FRW timing vs breakpoint tables | cast/attack, record animation frame counts via `monsters ... anim row` (monster) and player `pmode` per tick in lockstep; compare to breakpoint tables | Game.exe frame timing from `mem` mode sequence | lockstep `step 1` loop; player animation row/frame not exposed for the player U (add to `dump player`: `mode`, `anim_frame`). D2D notes say FHR/FBR counts are off by 1 to 4 frames at high stats D2D (findings-2 section 8): d2d is not the oracle for high-end breakpoints. |
| Merc AI | hire (`menu` rows V), follow, fight, potion use (`cmd belt n merc` V), death and revive; log merc life/position per tick | Game.exe merc logs (`FindMerc`) | merc state verb (dump `merc`); d2d merc auras not built D2D (findings) so aura behaviour is not testable. |
| Death rule (gold, XP, corpse) | done in spike V (docs/d2d-spike.md 5a) | probe on Game.exe still open | none (confirm on original). |
| Quest persistence / safe-quit table | done in spike V (5b) | confirm on original | Act 2-5 words (dump quests). |
| Pathing / collision | `debug grid 5` vs our `d2map` grid on the same level seed (seed on both: Game.exe `-seed` is real but we may not launch games ourselves) | d2map dumps | Game.exe seed control (U), unexplored room reveal differences. |

Procedure: each probe is a small Go program over the adapter (so it also tests the adapter) writing a CSV; the analysis
compares distributions (means, spread), not single runs; probes that need cheats record the `debug` verbs used in the ledger.

## 8. Open questions and next steps

1. Does `Town::update` (and the server tick) consume time in fixed 40 ms steps with an uncapped catch-up loop? Decides the
   effort of `--time-scale` (4.4). One hour reading, no running.
2. Are d2d cells level-relative or act-absolute? One `debug level` + `debug players` + `debug grid 5` probe at the town start
   (compare with our `d2map` origin for the same spot).
3. Does `items` include stash and cube contents when their panels are open, and what are `loc` values beyond 0/1/2?
4. Is the mode table of `monsters` (NU, DD, ...) the same set as D2 unit modes? Build a table from the spike's output and
   the `monsters` code (read-only) in Phase 1.
5. Does `cmd skill` accept ground-targeted skills (teleport, blizzard, hydra) with no unit? Needed for Sorceress farming.
6. Does `Escape` plus the game-menu "Save and Exit" click reach charselect without a process restart (faster character swap)?
7. Decision for the user: the fork lives where (the forge private repo name), who reviews GPL obligations before any container
   ships the binary, and whether any upstream contact happens (we do none).

## Phase 1 status: read-side adapter `internal/d2dsrc` (agent D2DSRC, 2026-10-30)

Done: `internal/d2dsrc` builds a `d2.Snapshot` from a running d2d with no `/proc/pid/mem`; `cmd/d2dsnap` prints it. Additive and
read-only: not wired into `cmd/d2bot`, no decision code touched, no verb that changes the game is ever sent. Tags: **V** =
verified (offline against the scripted server and the fork source, not against a live d2d: see below), **U** = unverified.

### What it is

- `d2dsrc.New(client, Options)` gives a `*Source` with `Snapshot() (*d2.Snapshot, error)`, the same method the bot calls on
  `*d2.Reader` today (so putting an interface around `env.Env.Read` in the wiring phase needs no change here), and
  `Read(ctx) (*Result, error)` returning the snapshot plus `Report` (present / derived / unknown fields, per section source)
  and `Extras` (facts the d2 types cannot hold: item qty/durability/socket count/ground label, NPC names, menu lines, the dump
  clock, corpses, warps). `Progress(ctx)` returns a `*d2.PlayerProgress` (quest words of all 3 difficulties, waypoint masks)
  from `dump quests`, same word layout as the save, so the questmem/saves decoders run unchanged; stock d2d answers `ErrNoProgress`.
- Round trips per tick (fork): `info` (once), `state`, `debug players`, `dump player,stats,skills,ui,merc,time`, `dump items`,
  `monsters`, `npcs`, `ground`, `debug portals`, plus `menu` while an NPC menu is open, `debug level` only with `AddOrigin`.
  Sections are split so one reply stays far below the 64 KiB line cap (a full dump of a loaded character may not).
- Stock d2d (no `dump`): degrades to `state` + `debug players` + `debug level` + `items`; `Report.Source` says `verbs` and the
  fork-only fields are unknown. A log line says so once.
- Transport errors fail the call; a game that is not readable gives `State` loading or menu (screen names other than `ingame`
  are menu, `loading`/empty are loading, `dump` answering `err no scene` is loading), like the memory reader.

### Honesty rules (how "unknown" works)

Every field d2d cannot provide is listed in `Report.Unknown` with the reason, counted in `Source.UnknownCounts()` per snapshot
and logged once per process (`Options.Logf`). Representation in the d2 types: `Class` -1 when no table resolves it, monster/player
`Mode` -1 for a code outside the mode table (and `monster.mode.<code>` is listed), ground `X,Y` -1 without a cell, `Stats` holds
only keys d2d gave (`Stats.Has(id)` is false for the rest), `Monster`/`Item` pointers are always set. Where a d2 type has no "absent" the zero stays and
only the Report tells the truth: never read a value of a field named in `Unknown`. Derived fields (use an assumption) are in
`Report.Derived` with the assumption.

### Mapping decisions

- Positions: d2d float cells to subtiles with `d2dctl.SubTile` (floor(5*cell), V: lossless for the 2-decimal multiples of 0.2
  d2d prints). Cells are level-relative (V: `debug players` equals `debug level`); our world units may be absolute act subtiles, so
  `Options.AddOrigin` adds 5*origin of `debug level` (U: unit of the origin). Off by default; nav needs this settled first.
- Unit ids: monsters and ground items use d2d's unit id; NPCs `NPCIDBase + index` (what `cmd interact` takes), the merc
  `MercID` (the dump has no merc unit id), town portals `PortalIDBase + row`. The input adapter decodes these.
- Monster modes: two-letter codes via the standard MonMode.txt order (DT 0, NU 1, ... DD 12, KB 13, SQ 14, RN 15), equal to our
  `MonMode*` constants (S). Player modes via PlrMode.txt order by name (d2d's own numbers are not used, U). Unknown code = -1.
- Monsters: class key (`fallen1`) resolves through `Tables` (`NewDBTables(*data.DB)`; nil Tables = Class -1). Boss column to flags
  (champion, unique/superunique, minion), MonUMod ids to `Enchants` (first 9). Hostility is NOT provided (see missing list).
- NPCs: with Tables a row whose name is a monstats entry becomes a `Monsters` unit (town NPC, like the memory reader's table),
  anything else an `Objects` unit with Class -1 and the name in `Extras.Names`. Without Tables the menu flag decides.
- Items: `loc` body/belt/inventory/stash/cube/cursor to `Container`, `slot` to the body location (d2s slot numbers equal our
  Body* constants, S), quality (d2s numbers), ilvl, flags identified/ethereal/socketed/runeword, unique/set id to `FileIndex`,
  item base row via `Tables.ItemRow` (index into `data.DB.Items`). `socketed` gems/runes are skipped (not units of the list in
  our reader either). Ground items: code, label, cell from the trailing columns (V in the spike transcript), Quality 0 + unknown.
- Player: class name to `d2.Class`, area to act by town boundaries, life/mana/stamina pairs also written into `Stats` as 24.8 so
  `Stats.Fixed` works, base attributes + `stats.bonus` as effective (bonus order str,dex,vit,ene is U), resists from the panel
  (4 of 5), skills = class base id (Amazon 6, Sorc 36, Necro 66, Pal 96, Barb 126, Druid 221, Assassin 251) + index in
  `base_levels` (ascending order U). `lskill/rskill` known; hotkey bindings are not needed (skills are selected by id).
- UI: stash, cube, npc menu, vendor, skill tree, waypoint, esc menu, mini panel, automap map onto our panel indices.

### Unknown or lossy fields (what the snapshot cannot say today)

Always unknown: `mouse`, `viewport`, `hovered`, `missiles` (count only), `unit.owner`, `unit.flags` (no hostile/friendly/dying),
`player.states` (poison, curses, buffs), `player.resist.magic`, `player.skills.bonus`, `player.stats.effective` (FCR, FRW, IAS,
FHR, MF, gear/aura sums), `player.hotkeys`, `monster.states`, `monster.resists`, `monster.unique_no`, `merc.level/name/aura`,
`ui.inventory/character/quest/party/trade/merc/chat` (a bot that closes panels would not see these open), `objects.other`
(waypoints, chests, shrines have no uniform list; `npcs` lists named ones, `debug presets/objects` are static level data),
`object.class/mode`, `ground.quality/ilvl`, item flags beyond the four, `inventory` socket count (in Extras only).
Fork-only (unknown on stock d2d): everything from `dump`, quest words, stash gold, stamina, stat points, skills, UI, merc, clock.
Lossy: `Player.Mode` (name based, U), act from the area, stamina units guessed from magnitude when the pair
looks like raw 8.8 (U), item `Unit.ID` taken from the dump `id` (U that it is unique across lists).

### What the control channel / dump still lacks for a bot, ranked

1. **Player states** (poison, curses, buffs) in the dump: antidote and curse handling are blind without it.
2. **Hostility / ownership per monster** (hostile flag, owner id, summon marks): without it summons and the merc cannot be told from
   enemies except by class tables; `pets=` is only a count.
3. **Panels**: inventory, character, quest log, party, trade, merc flags in `dump ui`; `CloseAll` needs them.
4. **Effective stat vector** (FCR/FRW/IAS/FHR/MF, magic resist, skill bonuses) and per-monster resists/immunities (combat's
   `refineResists`).
5. **Objects with class id and state** (waypoints, chests, shrines, portals) and missiles as lists; ground items with
   quality/ilvl/flags.
6. Merc unit id/level/name/aura; item unit ids unique across lists; `dump items` for vendor stock (open store) and stash while closed.
7. Click modifiers and key down/up (input side, section 2); mouse readback only for pixel flows.
8. Lockstep clock is already in `dump time` (consumed as `Extras.Clock`); the dump should also carry a unit-id namespace note.

### Verified and not

- V (offline): `go test -race -p 1 ./internal/d2dsrc` runs the mapping against a scripted socket server fed by the fork-source
  field names (`internal/d2dsrc/testdata`, values synthetic, fresh Necromancer in camp and a late Sorceress with merc, items in
  every container, quests): fields, unknown reporting, counters, one-time log, stock-d2d degradation, state mapping, origin option.
- U: nothing ran against a real d2d (another agent's d2d was running on the worker host, so the live check was skipped by the one-at-a-time
  rule). The fixtures follow the fork's `devctl_dump.hpp` field names; a changed real layout shows as unknown fields, not a panic.
- Ready-to-run live check (the worker host, fork, no other d2d running): `cd ~/d2d-lab/fork && nice -n10 ./start.sh --seed 3 --start-screen ingame
  --start-class 1 --start-name Snap &`, then on the worker host in the repo `D2D_SOCK=$HOME/d2d-lab/fork/d2d.sock go test -count=1 -run TestRealD2D -v ./internal/d2dsrc`
  (prints present / derived / unknown per field) and `go run ./cmd/d2dsnap -sock $HOME/d2d-lab/fork/d2d.sock -progress -corpses -warps`.
  Add `-tables <cache dir>` (the `data.Load` cache) to resolve item rows and monster classes.
- Fake server: `internal/d2dctl/fake` has no `dump` verb (not our directory); d2dsrc tests use their own scripted server. Adding
  `Fixtures.Dump map[string]string` to the fake would let other tests share it.

### Next phase: input adapter and lifecycle

1. `d2dinput.Controller` implementing `input.Controller` over `Click/Key/CmdMove/CmdSkill/CmdInteract/CmdPickup/CmdBelt/CmdSelect`;
   decode `NPCIDBase/MercID/PortalIDBase` unit ids; world moves use `cmd move` with `Cell(sub)` and re-issue until arrival from the
   next snapshot; the kick quirk is already handled by `d2dctl`.
2. An interface around `env.Env.Read` (`Snapshot() (*d2.Snapshot, error)`) so `d2dsrc.Source` can replace `*d2.Reader`; the bot-side places
   of section 6 (memory-only paths: `refineResists`, `readOwner`, server-player scan for quests) switch to `Progress`/Extras.
3. Lifecycle: start/stop d2d as a child (`--devctl` socket per instance, private XDG dirs), crash detection by `ping` + `info frame`,
   restart through `quit` + relaunch; one client per socket, so the Source and the input adapter share one `d2dctl.Client`.
4. Settle `AddOrigin` and the monster `Mode` mapping against the real game with the first live check above, then run nav and combat
   on recorded `Result` fixtures (`d2dsnap -n` output) before wiring.

## Phase 2a: interface audit (agent D2DIFACE, 2026-10-30)

Behaviour-preserving refactor, no decision code touched. Tags: **V** = verified by build, vet and tests (offline), **U** = unverified.

### What changed (V)

- `d2.StateReader` (internal/d2/statereader.go): `Snapshot, Player, UI, Units(typ), UnitStats, ReadStatList, ReadBaseStatList, RawStatList`.
  These are exactly the methods the bot calls through `env.Env.Read` (grep of `.Read.` and `e.Read` over internal/ and cmd/).
  `env.Env.Read` is now a `d2.StateReader`; `*d2.Reader` satisfies it (compile-time assertion). `nav` already used its own
  one-method `stateSource`.
- `d2.ErrUnsupported`: "this source cannot answer by design". Never a zero value posing as data.
- `*d2dsrc.Source` implements it (internal/d2dsrc/statereader.go): `Player`, `UI`, `Units(Monster|Object|Missile)` are derived from
  one `Snapshot`; `UnitStats`, `ReadStatList`, `ReadBaseStatList`, `RawStatList`, `Units(Player|Item)` answer `ErrUnsupported` (d2d has no unit addresses).
- `internal/d2/fake.Reader`: fixed-snapshot implementation for tests. `TestStateReaderConformance` (internal/d2dsrc) runs memory reader
  (over `mem.Fake`), d2d source (scripted server), fake and empty fake through every method.
- Left on the concrete type (too entangled): `questmem.New(*d2.Reader)` walks server-side process memory (`FindServerPlayer`,
  `ReadProgress`, `ServerPlayerOK`). `cmd/d2bot/task.go` now type-asserts `e.Read.(*d2.Reader)`; with another source there is no
  in-memory quest reader and the save-file reader is used, as before when `Read` was nil. d2dsrc has `Progress(ctx)` for the later swap.
  Also concrete: `canary`, `sentinel`, `cheat`, `cmd/d2watch|d2qmem|d2bot dry.go` (diagnostic tools).

### Audit: process-memory dependencies a d2d backend lacks

Effort: S under half a day, M a day or two, L needs a new d2d verb or design.

| file:line | reads | d2d source / next abstraction | effort |
|---|---|---|---|
| internal/combat/aura.go:33-50 (`ActiveAura`, `readU32(f.e, p.Addr+uoStatList)`, child stat list chain) | active aura id from the player's child stat lists (`e.Mem`) | `f.auraFn` seam exists. Needs aura/state in `dump stats` or a `Player.ActiveAura` snapshot field; fallback to right skill already exists | S-M |
| internal/combat/fighter.go:652-657 (`readU32(f.e, t.Unit.Addr+0x5C)` then `Read.ReadStatList`) | monster stat list for live resists | `statsFn`-style seam: a `MonsterStats(unit id)` method keyed by unit id, d2d `monsters` resist columns (U) or fall back to the static tables (the `HasRes=false` path already exists) | M |
| internal/combat/gearcheck.go:31-32, internal/town/shop.go:115-134, internal/leveler/equip.go:74-75, internal/loot/loot.go:104-105 (`Read.UnitStats(u.Addr)`) | per item stat list (identified affixes) | `Item` already carries what d2dsrc fills (Extras); abstract as `ItemStats(Unit)` keyed by item id; d2d `dump items` stats (U) | M |
| internal/leveler/leveler.go:230-240 (`mem32(l.e.Mem, p.Addr+0x5C)` + `ReadBaseStatList`) | base (item-free) attributes | d2d dump gives base+bonus split (d2dsrc computes `Strength=base+bonus` V); expose `Player.BaseAttrs` in the snapshot | S |
| internal/town/town.go:178-190 (`readOwner`: item -> ItemData -> owner inventory) | whose container an item sits in (shop/stash/vendor attribution) | d2d `items` lists container per row (U); abstract as `Item.Owner` enum | M |
| internal/leveler/equip.go:231, leveler.go:76/492/548, cmd/d2bot/live.go:116, probe.go:28-33 (`Read.Player`, `Read.RawStatList`) | player struct, raw stat bytes (probe only) | `Player()` done (derived). `RawStatList` is a debug probe: leave unsupported | done / n/a |
| internal/combat/deathfacts.go:111, cmd/.../deathrec.go (`Read.Units(TypePlayer)`) | other players (party/corpse facts) | `Units(Player)` unsupported; d2d `debug players` is read by d2dsrc but not mapped to units; add to Snapshot as `Players []Unit` | S |
| cmd/d2bot/task.go:416 + internal/questmem (`questmem.New(*d2.Reader)`) | server-side quest words by process memory | `d2dsrc.Source.Progress` returns the same word layout (V offline, fork only); put `quest.Source` behind a factory chosen by backend | M |
| internal/env/env.go:37 `Env.Mem mem.Source` and every `e.Mem` use (aura.go, fighter.go, leveler.go, town.go above; `cmd/bossrun/main.go:105` io.Closer) | raw process memory | the whole `Mem` field would be nil on a d2d backend: all four users above already nil-guard (`f.e.Mem == nil`), so nil is the d2d value; `town.go:180` does not guard (only reached via shop owner check) | S |
| internal/d2map/tracker.go:75 `NewTracker(mem.Source)`, reader.go:101-110 (player unit, act, path, rooms from memory) wired at cmd/d2bot/live.go:75, bossrun:152, probes:132 | collision grid, rooms, level seed, `Update()` | nav already uses the `worldSource` interface (Update, Current, Level). Implement it from `debug grid 5` / `debug level` / `debug warps` (spike V in section 1): a `d2dsrc.World` returning `d2map.LevelMap` with the whole level known (nav's unexplored-room assumptions differ, U) | L |
| internal/input/state.go:21 `input.Reader{Src mem.Source}` (player mode, mouse, viewport, skills, UI vars, hover; used by `input.NewGame` at cmd/d2bot/live.go:81) | the input controller verifies its own effects through memory (mouse, panels, cursor) | Input phase: a d2d controller replaces `input.Game` wholesale (verbs, not synthetic input), so `input.Reader` is not needed there; but any shared `Controller` code reading `Reader` must go behind its own interface | L (input phase) |
| internal/launch/oog.go:36-101, 301-340 (`ReadPhase`, `ReadControls`, `PlayerName`, `uiState` over `mem.Source`; `Flow.Mem` at cmd/d2bot/behaviors.go:100) | menu/out-of-game phase and controls by memory | d2d `state screen=` and `menu` give the phase (V in section 1); needs a phase source interface behind `launch.Flow` | M |
| internal/mem/proc.go (`/proc/<pid>/mem`, ptrace), internal/input/procmem_linux.go, internal/memfd, internal/sentinel/observe_linux.go | process access, CAP_SYS_PTRACE (`worker-sudo` setcap via tools/live.sh) | none needed on d2d (socket only); keep behind build use, nothing to abstract | n/a |
| internal/launch/{proton,sandbox,mounts,game,system,managed,instance}.go, internal/paths, internal/fleet/{runtime,spec,bot}.go, internal/setup/*, tools/d2-prefix.sh | Wine/Proton prefixes, `-w -ns`, Xvfb displays, `/tmp/d2lod-game-<id>.lock`, helper port 47012+N | launch layer is per-backend: d2d is started as its own process; `fleet` needs a backend choice in `spec` (U: d2d launch command not designed) | L |
| internal/replay (`Fixture.Mem`, Recorder), d2map `Image`/`Recorder` | memory-image record/replay fixtures | d2d fixtures would be recorded replies (d2dsrc testdata already does this); no change for now | n/a |

### Top 5 abstractions still needed (in order)

1. World/collision source: implement `nav.worldSource` from d2d grid verbs (the only large blocker for nav; L).
2. Input: a d2d `input.Controller` plus removal of `input.Reader`/`mem.Source` from the controller path (L, Phase 3).
3. Per-unit stat access by id instead of address: `UnitStats/ReadStatList` callers (monster resists, item stats, aura): one `UnitStats(Unit)`-style method on `StateReader` (M, replaces four seams).
4. Quest/progress source factory so `questmem` is chosen per backend (M).
5. Phase source for `launch.Flow` (out-of-game menus) plus a backend choice in launch/fleet (M-L).
## Phase 2b: input status (agent D2DINPUT, 2026-10-30)

Package `internal/d2dinput` (stdlib only, on `internal/d2dctl`): `Controller` implements `input.Controller` and the new optional
`input.SemanticInput` (`internal/input/semantic.go`, additive: `ErrUnsupported`, `SemanticInput`, `AsSemantic`), `Process`
implements `Lifecycle`. Not wired into `cmd/d2bot` or `internal/env`. One `d2dctl.Client` is shared with the d2dsrc `Source`
(one client per socket). `Controller.Ctx` bounds every call (the interface has no context). `Controller.Origin` mirrors
`d2dsrc.Options.AddOrigin`; both must agree.

Tags: V = observed on d2d in earlier probes, S = secondary (d2d notes, not run by us), U = unverified. Everything below was tested
against the fake channel only (V offline), nothing against a real d2d.

### Mapping

| input.Controller | d2d line(s) | Tag / note |
|---|---|---|
| `MoveMouse(x,y)` | `move x y` | V verb. Game pixels = our client pixels, frame 800x600 (V `info`). |
| `Click(Left/Right,x,y)` | `click` / `rclick x y` | V |
| `Click(..., ModShift/ModCtrl)` | `ErrUnsupported`, nothing sent | gap G1 (fork P3 not done). Potions: `Belt`. |
| `Key(name)` | `key <d2d name>` through `KeyName` | table below |
| `TypeChar(r)` | `text <r>` | S (verb exists, not exercised) |
| `Hold` / `Release` | `ErrUnsupported` | gap G2: no key down/up |
| `Pos()` | `debug players` (first, seat -1) x5, floor to subtiles, plus `Origin` | V verb |
| `ClickWorld(w)` | `debug level` (once per level id), `click 400 500` when needed, `cmd move x y` | cells = (sub+0.5)/5. V quirk |
| `MoveTo(w,opts)` | loop: `cmd move`, poll `debug players` every 100 ms, stall -> one extra `click 400 500`, 3 stalls -> `ErrMoveFailed` | long paths stop short (V), hence the loop |
| `SelectSkill(hk)` | `cmd select <id> 0` if `Skills[hk]` is set, else `key <hk>` | `cmd select` verb V, args U |
| `CastAt(hk,w)` | `cmd skill <id> x y` if mapped, else key + `rclick` at a pixel from the player position | pixel path U (transform not calibrated on d2d) |
| `Interact(t)` | NPC id (`NPCIDBase+i`) -> `cmd interact i`; type 4 -> `cmd pickup id`; monster -> `cmd skill 0 x y id left`; portal id -> `UsePortal` | monster attack id/button U |
| `Interact` on object / merc | `ErrUnsupported` | gap G3: no object verb (chest, waypoint, stash) |
| `CloseAll()` | `state`: game menu open -> `click 400 308`; else `menu` has `cancel` -> click it; repeat up to 4x | V (Escape is avoided on purpose). Open panels are invisible on the channel: gap G4 |

Not implemented on purpose: `HoldClick` (Zeal chains, gap G5), so callers that type-assert it fall back.

Semantic extras (`input.SemanticInput`): `Move`, `Cast(skill,w,unit)` = `cmd skill`, `InteractNPC(index)`, `Pickup(unit)`,
`ChooseMenu(text)` (reads `menu`, clicks the line; header skipped), `UsePortal(row)` (walk next to the `debug portals` cell, click
the computed pixel: U), `Belt(slot, merc)` = `cmd belt`.

Key names (`KeyName`, lower-case input, modifiers refused with `ErrUnsupported`, unknown names an error):
`esc`->`Escape` V, `enter`->`Return` V, `a`..`z`->same S, `0`..`9`->same U, `f1`..`f12`->`F1`.. U, `tab`->`Tab` S, `space`->`Space` S,
`backspace`->`Backspace` U (SDL spelling; the channel may want another). Unknown to us: arrows, page keys, `Delete`. First live task: run each
once and fix the table.

### Lifecycle (`d2dinput.Process`, os/exec only)

Binary from `$D2D_BIN` (else `ErrNoBinary`), data dir `$D2D_DATA` optional, one socket and one `XDG_CONFIG_HOME/XDG_DATA_HOME` per instance,
optional `nice`. Flags per docs/d2d-spike.md section 3 (V): `--data --headless --devctl SOCK --no-video [--seed N] [--no-save]
--start-screen ingame --start-class N --start-name X [--start-hardcore] [--start-classic]` for a new character (class = UI order),
`--start-screen charselect --start-name X` then `key Return` for a saved one. `WaitInGame` polls `state` for `screen=ingame`, then
sends the priming click (V quirk after a load). `SaveExit` = `save` + `quit`, SIGINT after 10 s. `Restart` = SaveExit + Start + WaitInGame.
Tested against a python stand-in for d2d (argument line, load flow, quit, restart). Not done: difficulty on start (`Spec.Difficulty != 0`
is `ErrUnsupported`: only the `debug difficulty` cheat exists), crash detection and auto-restart (design: `ping` timeout + `info frame`
not advancing, then `Restart`), selecting a character from the running process (relaunch is the only V flow), hardcore/classic flags are
V in the docs but unused by our probes.

### Gaps, ranked

1. G3 object interaction (waypoint, stash, chests, shrines): the biggest blocker for a semantic loop. Needs a `cmd interact obj <id>` verb in the fork, or pixel clicks from `debug presets`/`objects` screen coordinates.
2. G4 UI state: open panels (inventory, stash, vendor) are only partly in `state` (`items panel=`); `CloseAll` cannot close a panel without an extra read.
3. G1 click modifiers (shift-attack in place, ctrl item moves, shift-click to belt): fork P3.
4. G5 held button (Zeal, Fury) and G2 key down/up (run toggle): a hold or a repeat-attack mode.
5. Skill ids for `Skills` (hotkey map) must come from the class tables in `internal/data`; `Deps.Prepare` becomes a no-op.
6. Monster attack through `cmd skill 0 ... left` and ground-targeted spells are U.

### What a semantic bot loop needs next

Wire both sides behind one shared `d2dctl.Client`: Source for reads, Controller for writes (needs D2DIFACE's `Env.Read` interface); a
`Skills` map built from the character; the bot's Mover/Fighter/Townsman prefer `AsSemantic(env.In)` for move, cast by unit id, NPC
interaction and potions; vendor and stash flows keep pixels (`ChooseMenu`, `items` coordinates). Then the first live run: `D2D_SOCK=... go test -run TestRealD2D -v ./internal/d2dinput`
(moves a throwaway character about 10 subtiles in town, hovers, clicks, closes menus, toggles the inventory key), and verify the key table.

## Phase 3: world source status (agent D2DWORLD, 2026-10-30)

`internal/d2dworld` (new, additive; `cmd/d2bot` and nav untouched). `*d2dworld.World` satisfies nav's `worldSource` (`Update`, `Current`, `Level`; compile-time check in its test) and builds `d2map.LevelMap` values from d2d verbs. Tags: V verified (live on the private fork, the worker host, 2026-10-30, or by the offline tests), S secondary, U unverified.

### What works
- `debug level` gives the rectangle (cells x5 = subtiles) and the player position; `debug grid 5` gives rows of `.`/`#`. **V live:** the reply has exactly 5*W columns and 5*H rows (town 285x205, Blood Moor 480x280), row 0 / column 0 is the level's top-left, so `grid[y][x]` is subtile (x, y) of the level. `debug warps` gives exits as cell + destination id: **V live** Blood Moor warp `0 67.0 28.0 8` became `Exit{Dest 8, Pos (335,140)}`; exit count equals the `debug warps` line count. Rogue Encampment has **no warps** (V): its way out is a level border, see gaps.
- `Update()` fetches grid, warps and presets on arrival in a level (also on every return visit), then again when the player has moved `RefetchDist` (60 subtiles) and `MinRefetch` (2 s) have passed; `Refresh()` forces it. Levels are cached by id (`Level(no)`), the grid is merged into the same `*d2map.Grid` in place so nav's field cache keys keep working.
- Unknown is never open: only `.` and `#` set a cell; any other character, a missing row, an error or an `err` reply leaves cells unknown (`Grid.Known` false, `Walkable` false). Failing `debug level` returns `d2map.ErrNotInGame` (nav maps it to "loading"). A reply over `MaxCells` (default 4096*4096, nav's `maxFieldCells`) is ignored whole.
- Presets: `debug presets` object ids become `d2map.Preset{PresetObject}`; `LevelMap.Waypoints()` works through the real object ids (V live: id 119 at the Rogue Encampment waypoint). `World.Waypoints/Doors` also match by name. `World.NPCs(ctx)` gives town NPC positions in subtiles incl. the stash ("Your Private Stash", V live) and the waypoint pseudo-NPC.
- Tools: `go run ./cmd/d2dmap -sock S [-origin none|tiles|subtiles]` prints the level as ASCII, exits, presets and NPCs; `D2D_SOCK=S go test ./internal/d2dworld -run Live -v` prints the grid and compares exits with `debug warps`.

### Coordinates and the origin question (open question 2 of section 8)
- d2d cells are level-relative in everything we saw (V: `debug players` position == `debug level` position, within the level's 57x41 / 96x56 cells; the grid is indexed from the level corner). Default `OriginNone`: subtile = floor(5*cell), the same as `d2dsrc` with `AddOrigin=false`. The two must agree: use `OriginTiles` here exactly when `d2dsrc.Options.AddOrigin` is true.
- `wx wy` of `debug level` (864 912 in town, 920 904 in Blood Moor): how to decide what it is. Our Game.exe `d2map` rectangles are absolute act subtiles. Run the Game.exe bot's `d2mapdump` for the same level and compare its `Rect.X,Y` with `wx*5` (tiles) or `wx` (subtiles). U: not done, needs a Game.exe instance on the same act seed. Until then nothing in the repo mixes the two backends, so OriginNone is consistent.

### How rooms and levels are built (corrected in round 2, see "Phase 3 round 2")
- Superseded: the first version of this section guessed that unlaid rooms read as `#`. Round 2 shows the collision (`Level::walk`) is built for the whole level when the level is created, so there are no unlaid regions in `debug grid`.
- A level is generated once per game and re-entering does not respawn monsters (probes doc); presets list only built rooms (probes doc), so far-away quest objects can be missing until the player walks near.

### Gaps and effort left
1. (DONE in round 2, see below) Level borders (S, M): outdoor transitions (Rogue Encampment <-> Blood Moor, wilderness to wilderness) are not warps. nav uses `LevelMap.Adjacent` / `BorderCells` for them. Need the neighbour level rectangles: either a static table of act layouts or `debug level` after stepping across. `Adjacent` is empty today, so `ExitTo` toward a neighbour level from town finds no gate. This is the first thing to fix before wiring.
2. Doors (M): collision of a closed door in the grid is U. `Doors()` is by preset name only; `ClosedDoor` flags are never set (every cell is 0 or `BlockWalk`). Nav door handling (`gate`, `badGates`) needs the open state: `debug objects` mode field and a re-fetch after opening.
3. Dynamic flags (S): the Game.exe grid carries `Player/NPC/Object/DeadBody` bits; here only walkability. Nav's monster avoidance already uses snapshot monsters, so mostly harmless; anything reading `DynamicMask` sees zero.
4. Traffic (S): a full grid is up to ~1 MB of text per level (480x280 subtiles = 134 KB; big wilderness more). One fetch per level change plus the rate limit is fine; do not call per tick.
5. Other levels' rectangles: `Level(no)` only knows levels visited this game.
6. `Rooms`, `State.Room/NearRooms/ActiveRooms`, `Warps` (Level.WarpX) stay empty. Nav features needing Game.exe-only data: border detection through Room2 near links (`Adjacent`), `room2Sig` rescans, room-based "explored" counts (it uses `Grid.KnownCount`, which is constant here, so the "map grew" checks in `recentNoExit` and `exitSearch` never see growth between refetches), and tracker `Reset` after menu re-entry (use `World.Reset`).
7. Pathing parity: nav's A* needs no change for walls; a replay of a real route on d2d (walk Town -> Blood Moor -> Den) is the acceptance test and needs the input phase.

### Phase 3 round 2 (agent D2DWORLD, 2026-10-30): borders, growth, unlaid rooms, coordinates

All live checks on the private fork on the worker host (load 25 to 31), d2d stopped afterwards. Tags V/S/U as above.

**1. Outdoor transitions (V live, needs two fork verbs).** Stock d2d has no verb that lists neighbours, but its source (`Level::nearby`, `Level::blocked`, `World::cross_level`, `drlg::Placed`) holds all we need, so the fork gets `debug layout` (the act's placements `level x y w h` in act tiles, from `GameData::layout_of`) and `debug nearby` (levels already linked past this level's edge: `level dx dy`). Patch: `~/d2d-lab/fork/patches/0003-P3-debug-layout-nearby.patch` on the worker host only (never in this repo: GPL; our fork, 12 lines in `apps/d2d/devctl_verbs.hpp`; branch `p3-layout-verb` in `~/d2d-lab/fork/src`; the fork binary `build/apps/d2d/d2d` now includes it, the old one is `~/d2d-lab/fork/d2d.before-layout`). Without the verbs `World` degrades to no `Adjacent` (note "no layout verb", fake server test).
- d2d crossing mechanics (S source, V live): `debug blocked x y` past the grid edge asks the neighbour level's collision when it is linked; a unit past the edge is moved to the neighbour (`cross_level`) with coordinates shifted by (dx,dy). V live: from Blood Moor, `debug warp -0.5 25.3` landed in the town at (55.5, 17.3), `debug level` said level 1.
- nav needs no new exit type: its border gates come from `LevelMap.Adjacent` + `BorderCells` (`borderGates` sets `border: true` itself). `d2dworld` fills `Adjacent` from the layout: neighbours whose edge coincides with ours and overlaps on the other axis (`abut`; unit test with corner touch and one-tile gap), one `Adjacent` per run of cells along the layout-edge strip that are walkable inside and walkable across (`debug blocked` just past the grid edge, one probe per tile, only for levels `debug nearby` already lists). Levels not yet linked count as `pending` and make `Update` refetch every `MinRefetch` (max 20 tries) even without movement.
- V live results: Rogue Encampment: level 2 contact `x=279, y 75..76, 82..88, 93..94` (the 15-cell west-to-east gap of the camp, walkable cells only; the wall cells beside it are not walkable and so not border cells). Blood Moor: level 3 (Cold Plains) `x=479, y 170..193`, level 1 (town) `x=0, y 122..128`. `BorderCells(2)` in town is non-empty, so `ExitTo(2)` has a gate; layout numbers `town 864,912 56x40`, `Blood Moor 920,904 96x56`, nearby `2 56 -8` agree.
- Gotcha (V live): d2d's grid is one tile larger than the layout (`debug level` 57x41 for the 56x40 town; Game.exe's Level rectangle is 56x40, docs/map.md). The extra margin is walkable beside the town wall (x 280..284 open for 144 rows), so the gate must be the layout-edge column, not the grid's last column; probes look past the grid's edge. `LevelMap.Rect` stays the grid size.
- Still open: whether `ExitTo` really walks through from town to Blood Moor needs the input phase; neighbour levels' own grids are not fetched (nav only uses them to filter border cells, so border cells are kept as is).

**2. Map growth (V live + S).** `KnownCount` cannot grow here: the first fetch already knows every cell (`58425 = 285*205` town, `134400 = 480*280` Blood Moor). What can change is cell values: three grids of Blood Moor taken at far apart positions differ by 5 to 10 cells (a plus-shaped stamp, d2d's unit collision: monsters and critters are in `unit_blocked`), so a refetch is rarely identical. nav keys its cached planner field on the Grid pointer and `KnownCount` only (`nav.go fieldFor`) and replans on `KnownCount` change (`walk.go:116`): a refetch that changes cells would be invisible. Fix in `d2dworld` (nav untouched): a refetch whose cells differ hands out a NEW `*d2map.Grid` (pointer changes, field rebuilt); an identical refetch keeps the old one. Recommended one-line nav change before wiring: add `Grid.Version()` to `fieldKey` and to the walk's replan test. nav's "map grew" exploration stops (`exit.go:62/71/407/459`) treat the d2d world as fully known: `ErrExplored`/no-exit answers come at once, which is right.

**3. Unlaid rooms (S source, V live).** Not an issue: `Level::walk` is filled for the whole DS1 when the level is built (`gamedata.cpp` around the `walk.assign` calls); `relevel` only re-picks tiles when rooms come up (it affects tile choice and spawn areas, `debug laid` lists the order). Live: grids at three corners of Blood Moor were full-size with at most 10 differing cells. So the first-round claim "unlaid rooms probably read as #" is withdrawn; nothing in `debug grid` is unknown. The d2map.Grid known-mask therefore stays all-known; unknown only means "no reply". (Room contents like presets and monsters are still built when the player comes near, probes doc.)

**4. Coordinates (V for units, S for equality with Game.exe).** `wx wy` of `debug level` and the `debug layout` coordinates are act TILES (5 subtiles each): V live, Moo Moo Farm (anchor level 39) layout `5000 1148` equals its Levels.txt `OffsetX/Y` `5000,1148` (our extracted `levels.tsv`), and Stony Field (4) is at `1000 1000`, the Levels.txt offset of the act anchor. Game.exe stores Level pos and size in tiles and our reader multiplies by 5 (`d2map/reader.go`: `Rect{pos*5, ...}`; docs/map.md V live: town at 1192,968 tiles, 56x40, absolute subtile 5960,4840). Both are the DRLG act frame (d2d reproduces it: drlg.md "2000/2000 seeds, rectangles"), so `absolute subtile = 5*tile + subtile offset inside the level` and `OriginTiles` is the right mode when a Game.exe-compatible absolute frame is wanted (use the same mode as `d2dsrc.Options.AddOrigin`). Consistency check: the Game.exe doc's Blood Moor 5480,4800 480x280 abuts that town on its west side (a placement the DRLG rules allow), and the d2d seed here put it on the east; the layout is seed dependent in both, so positions of neighbours cannot come from a static table, only from `debug layout`. Not compared cell by cell against a Game.exe dump (needs a running Game.exe; not started). Default stays `OriginNone` (consistent with d2dsrc's default).

**Verified vs not.** V: layout/nearby verbs, town and Blood Moor borders, crossing by `debug warp`, grid size and whole-level knowledge, tiles for wx/wy (Moo Moo Farm). Offline tests: abutment geometry, probe-limited contacts, margin case, pending then linked, no-layout degrade, new grid on change. Not verified: walking through a gate with nav, Adjacent for acts 2 to 5 (the layout verb covers all acts' placements, only act 1 seen), cave levels (they use warps), doors.

## Launch toggles: autoloot off by default (agent D2DTOGGLE, 2026-10-30)

Our d2d start scripts now pass `--toggle autoloot=off,autoparty=off` (override with env `D2D_TOGGLES`), details in docs/d2d-fork.md section 10. Consequence for the adapter: gold piles
are only picked up by `Controller.Pickup(unit)` (`cmd pickup <unit id>`, V-d2d for gold, the same call the live loot loop makes with a click); walking over a pile does nothing, as in the original.
Fidelity runs made before this date had autoloot on (see the dated correction in docs/d2d-fidelity.md section 9).

## Phase 4: the bot boots on d2d (agent D2DBACKEND, 2026-10-30)

See `docs/d2d-bot.md`: `internal/d2denv`, `d2bot --backend d2d`, the closed open items (stats by unit id, quest source factory, Flow choice,
socket/instance config, seed flag) and the combat/town seams.
