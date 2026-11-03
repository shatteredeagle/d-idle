# SEEDSPAWN: spawn list, layout and the RNG draws before spawn, game.exe 1.14d against d2d against our bot (card seedspawn, 2026-11-02)

> status: current. Docs and emulator scripts only: no game, no live run, no d2d slot, no Go change, nothing pushed. "The worker host" = the lab machine. Companion docs: `docs/research/seed-study.md` (the question, sections 0, 1.3, 3, 4, 7), `docs/research/tierb-design-20261102.md` (the A:B plan), `docs/research/upstream-gaps/item3-monster-density.md` (the earlier layout/density diff), `docs/research/monster-stats-trace.md` (stats), `docs/research/monster-mods-trace.md`.
>
> Tags: **V run** = game.exe's own code run in the emulator (upstream `tools/emu` oracles: `drlg.py`, `monsters.py`, `objgroups.py`, `store.py`, `hire.py`, plus the scripts of this card) against the real compiled d2d code (a hand build of upstream's `tools/drlg-dump` and of two probes against the lab's scratch build of upstream main 7d5b3a5c plus the Blizzard shard commits, irrelevant to level generation), counts stated; **V read** = read in a disassembly (address and instructions) or in d2d's source (file:line); **D** = derived from data (MPQ tables); **S** = shown by an existing doc or upstream proof (named); **I** = inference; **U** = unverified. A "match" in this doc always comes from a counted run, never from arithmetic alone.

## 0. Answer in ten lines

1. **Layout, preset units, room seeds and the spawn list are identical seed for seed.** Rogue Encampment (1), Blood Moor (2), Cold Plains (3), Stony Field (4) and Den of Evil (8): tiles, preset units and room seeds equal in 30 of 30 seeds (1..30), 10 of 10 seeds of each of two 32-bit ranges, and the edge seeds 0 and 4294967295; the populated room lists (class, position, mode, leader per room) equal in 5,991 of 5,991 populated rooms of levels 2, 3, 4 and 8 (section 2). V run.
2. **The game's own seed chain after level creation is identical too**: the game seed, the object seed and every container drop of a whole game (levels 1, 2, 3, 4, 8 made in turn, all rooms populated, every container opened) equal in 30 of 30 seeds; the camp's units (stepping the game seed) in 30 of 30 (`npcs` kind, 150 think steps). V run.
3. **Vendor stock and mercenary offers equal game.exe's for the same sunitproxy seed**: 20,400 of 20,400 stock lines (200 seeds x 17 vendors x 6 act/difficulty/clvl jobs, every item asked and the seed left) and 7,200 of 7,200 offer lines. V run. That d2d's proxy seed is the game seed's third step, as game.exe's, is V read on both sides and not itself compared (I that the two agree).
4. **What differs is per unit, after the unit exists** (section 3): d2d rolls a monster's life on the **world stream** (one `Rng{0x2545f491}` shared by every roll that is not a unit's or the game's), game.exe on the **unit seed**; and d2d leaves the unit seed **one step behind** game.exe's (it skips the life roll's step): 14,605 of 14,605 plain monsters whose start seed is equal end one step apart. So the same seed gives the same world and different life values and a shifted drop and AI stream per monster. V run.
5. **Statistically the life roll is the same**: 14,730 plain Normal monsters of the real populations (seeds 1..30, levels 2, 3, 4, 8): two-sample Kolmogorov-Smirnov D 0.0103 (p 0.43) on the normalised life; none of 13 (class, level) groups with 100 or more units differs at p < 0.01; mean life 6.229 against 6.246 (ratio 1.0028). Seed for seed the lives agree in 3,060 units (chance 3,079.5). V run.
6. **The old seed-study statement "d2d has one fixed-constant stream" is out of date** for the game seed, the object seed, the sunitproxy seed, unit seeds at creation, drops and chest contents: those are the game's chain now (V read `gamedata.cpp:190-206`, `loot.hpp:128-143`, V run items 2-3). What is still one shared constant stream: monster life and the other rolls listed in section 4.
7. **Bot action order can shift outcomes on d2d that game.exe does not shift the same way** (section 4): any combat roll, a gamble, a shrine, and the Cairn Stones' order draw on the shared world stream, so the number of earlier rolls (rooms woken, fights) moves them. On game.exe a combat roll is on the attacker's unit seed. The distributions are the same (item 5 for life; the combat rules are the subject of the sk-* traces); the paths are not. I.
8. **A:B verdict** (the decision this feeds): the statistical A:B of `tierb-design-20261102.md` stays the right plan; this trace finds no layout, preset or spawn-list difference that would bias it, and one rule difference at the unit-seed level (D2) whose effect is a shifted draw, not a different distribution. No PR candidate (section 7).
9. **The `.d2s` pin was not run from a file** (no saved character was readable from here, section 5): the seed went in as the map seed number, the value both sides read from the file (V read `main.cpp:378-379`; game.exe: D/V live, seed-study.md 1.1).
10. Limits and dead ends are in section 11; the drill recipe for the U items is in section 9; bot findings in section 8.

## 1. Questions answered

| Q | Question | Answer | Where | Tag |
|---|---|---|---|---|
| Q1 | Where does the map seed come from and do both sides read the same file field? | d2d: `--seed N` fixed, else the save's map id when the active-difficulty byte has bit 0x80 and the id is not 0, else random. game.exe: the same field, a saved id of 0 is reused (not treated as none). | d2d `apps/d2d/main.cpp:378-379` (`game_seed`), `:1689`; game.exe `FUN_0056a090` per seed-study 1.1 | d2d V read; game.exe D |
| Q2 | Is the level layout (tiles, preset units, room seeds) the same for one seed? | Yes: 0 differences, section 2 | `tiles`, `units`, `seeds` kinds of `diff_drlg.py` | V run |
| Q3 | Is the spawn list (class, count, position, mode, leader, per room) the same? | Yes for Blood Moor, Cold Plains, Stony Field, Den of Evil: 0 differences in 5,991 populated rooms. The Rogue Encampment is not a populated level: its monster units are preset units (cows), covered by `units`/`game`/`npcs` (section 2.3). | `monsters` kind | V run |
| Q4 | What does game.exe draw before the first monster spawns, in order? | section 3.1 | `monsters.py new_game`, `FUN_00530930` | V read, V run (the chain equal) |
| Q5 | What does d2d draw, and where does it differ? | section 3.2 and D1-D3 | d2d source, ss_unitseed | V read, V run |
| Q6 | Where does d2d's world RNG serve unrelated actions, and can a bot action order shift another outcome? | section 4 | d2d source | V read; the shift itself I/U |
| Q7 | Are d2d's monster life and drops statistically equivalent to game.exe's? | Life: yes (KS, section 3.4). Drops: the drop machinery equal for the game's container drops (V run) and for the death drop given a seed (S: upstream's drop-dump 39,980 of 39,980, seed-study 7); the death drop's start seed is shifted (D2), distribution not tested here. | section 3.4 | V run / S / U |

## 2. The differential: layout, presets, spawn list

Command (all on the worker host, `tools/emu/ss_run.sh` wraps upstream's `diff_drlg.py` with d2d's side = `tools/emu/ss_build.sh`'s hand build of upstream `tools/drlg-dump/drlg-dump.cpp`; capacity checked first, idle 52 percent plus nice 19):

```
tools/emu/run-on-worker.sh ss_build.sh
tools/emu/run-on-worker.sh --ttl 3000 ss_run.sh 1-30 <level> tiles units seeds monsters      # levels 1 2 3 4 8 (1: no monsters kind)
tools/emu/run-on-worker.sh --ttl 3000 ss_extra.sh      # objgroups 1-30 per level; seeds 1967900452-61 and 4062638700-09 per level; npcs 1-30 level 1
tools/emu/run-on-worker.sh ss_edge.sh                  # seeds 0 and 4294967295, levels 2 and 8
tools/emu/run-on-worker.sh --ttl 3000 ss_run.sh LEVELS=1,2,3,4,8 1-30 0 game
```

What each kind compares (V read of `diff_drlg.py` and `drlg.py`/`monsters.py`/`objgroups.py` in upstream): `tiles` the level's rooms and tile picks; `units` the rooms' preset units after game.exe brought the rooms up (`FUN_0061b730`) in the stated order; `seeds` each room's seeds; `monsters` game.exe's population pass per room (`FUN_0052d0f0`, walked newest first as `FUN_0052d160` does) with the units made hooked at `FUN_00555230`, one line per room `class[!mods]@x,y[m<mode>]/<leader>`; `objgroups` each room's random object groups; `game` one game with the levels made in turn (the camp's units, the four creation steps, the object seed, every container opened), game seed and drops compared; `npcs` the camp's NPCs thinking.

### 2.1 Results (V run, game.exe in the emulator against d2d's real code; counts are seeds per kind)

| Level | seeds 1..30 | two 32-bit ranges (10 + 10 seeds) | edge seeds 0, 4294967295 | kinds | populated rooms equal |
|---|---|---|---|---|---|
| 1 Rogue Encampment | 30/30 | 10/10 + 10/10 | not run | tiles, units, seeds (no monsters, 2.3) | not applicable |
| 2 Blood Moor | 30/30 | 10/10 + 10/10 | 1/1 + 1/1 | tiles, units, seeds, monsters, (objgroups 30/30) | 839 + 280 + 290 + 21 + 28 = 1,458 of 1,458 |
| 3 Cold Plains | 30/30 | 10/10 + 10/10 | not run | same | 1,162 + 394 + 392 = 1,948 of 1,948 |
| 4 Stony Field | 30/30 | 10/10 + 10/10 | not run | same | 1,044 + 350 + 364 = 1,758 of 1,758 |
| 8 Den of Evil | 30/30 | 10/10 + 10/10 | 1/1 + 1/1 | same | 475 + 170 + 152 + 16 + 14 = 827 of 827 |

Counted as seed x level x kind comparisons: levels 2 and 8: 120 + 80 + 8 + 30 = 238 each, levels 3 and 4: 120 + 80 + 30 = 230 each, together **936** comparisons with 0 mismatches; level 1: 3 kinds x (30 + 20) = 150, plus `npcs` 30 = **180** with 0 mismatches; the `game` kind (levels 1, 2, 3, 4, 8 in one game): **30** of 30. The 10-seed ranges are the two map ids seen live in a joined game (seed-study 1.1, D/V live: 0x7543c724 = 1967900452 and 0xf221f26c = 4062638700, each plus nine) so the seed size of a real save is covered.

### 2.2 The two mixed-up runs (dead ends, kept)

- First attempt of the `game` kind with LEVELS=2,3,4,8 failed before comparing: the oracle needs the `mpq-cat` binary beside `drlg-dump`; `ss_run.sh` now copies upstream's built `mpq-cat` and `drop-dump` into the scratch build dir (read only). The rerun (LEVELS=1,2,3,4,8) passed 30 of 30. No result was lost.
- The `monsters` kind on level 1 reports "0/69 populated rooms match" (and 0/21, 0/23 on the two ranges): game.exe's population pass makes 2 to 3 units in the camp (class 179, the cow, positions 45,65 / 142,68 / 98,155 in subtiles for seeds 1 and 2; two for seed 3: `tools/emu/ss_town.py`), while `drlg-dump ... monsters` of a preset level prints nothing (V read `gamedata.cpp:307-321`: a camp room has no population; its preset units, the cows and the NPCs, are made on game seed steps, `:311-319`). This is the tool's limit, not a deviation: the units are compared by `units` (30/30) and their seed steps by `game` (30/30) and `npcs` (30/30).

### 2.3 The camp

Preset units of the camp equal in all 50 seeds (`units`); the camp's NPC thinking for 150 steps equal in 30 of 30 seeds (`npcs`, V run). d2d makes the camp's units "in the DS1's order reversed, each on a step of the game seed" (`gamedata.cpp:307-319`, V read), as game.exe's room pass does for the camp's room records. Client-side critters (the chickens) come from `FUN_0046c1a0` on the room seed and are not part of this comparison (S `re/critters.md`; d2d `Town::critter_tick`, seed 3: 23 chickens in the camp, U against a live camp: the earlier critter note counts 15 in a reference 1.14d client's camp).

## 3. RNG draws before and at spawn

### 3.1 game.exe, in order (V read: `monsters.py new_game` calls the game's own functions; V run: the chain is equal, 2.1)

| # | Step | game.exe | d2d | Equal |
|---|---|---|---|---|
| 1 | game seed `{map seed, 666}` (`g+0xd0`, `g+0xd4`) | `FUN_0052c280` / set by the harness as with `-seed` | `gamedata.cpp:197` `Rng{map_seed}` | V run (game seed after levels equal) |
| 2 | four steps: regions `FUN_00547d20`, object seed `FUN_00546c60`, sunitproxy `FUN_00536070`, quests `FUN_00545d80` | each takes one step of the game seed | `gamedata.cpp:198-201` same order | V run (game seed, object seed, drops of 30 games) |
| 3 | the camp's units made | each `FUN_00555230 -> FUN_00552df0` steps the game seed, unit seed `{low, 666}` | `gamedata.cpp:311-319` | V run (`game`, `npcs`) |
| 4 | act and level seeds `{act seed + level id, 666}`, room seeds | per level / per room | `drlg` | V run (`seeds` kind) |
| 5 | per room pass `FUN_0052d0f0`: presets, object groups, `FUN_0054ec90` population on the room seed | room seed | `gamedata.cpp:297-` | V run (`monsters`, `objgroups`) |
| 6 | each monster made: unit seed = game step (start), then `FUN_00573cb0` | look `FUN_005739d0` (call at `0x573d43`), variant (`0x573d6e`/`0x573d9b`, rows 0x137/0x138), then **life roll `0x573f8f`: `FUN_0045c3e0` with ECX = unit+0x20, EDX = max - min + 1**, then stats; no further draw on the unit seed | `ai.cpp:140-143`: look, variant, props on `Rng{spawn.seed}`; **life on the world stream** (`ai.cpp:2935`) | start equal (V run); end differs, D1 D2 |

Disassembly read this session (upstream `disasm.py` on the patched 1.14d game.exe, `tools/emu/mst_dis.sh`): `0x573cb0 push ebp ... test esi,esi` (ESI = unit); `0x573d43 call 0x5739d0` (look); `0x573f7e call 0x6538a0` (the stat rows); `0x573f83..0x573f8f mov edx,[ebp-0x58]; sub edx,[ebp-0x5c]; lea ecx,[esi+0x20]; add edx,1; call 0x45c3e0` (life = min + rand(max-min+1) on unit+0x20); `0x573f94..0x573faf` the player-count percent and the 0x7fffff cap. S: `docs/research/monster-stats-trace.md` Q3 reads the same.

### 3.2 What d2d draws (V read)

- Game chain: `start_spawning` (`gamedata.cpp:190-206`): `Rng game{map seed}`, then four `game.next()` for region, objects, proxy, quests.
- Every spawn gets `spawn.seed = game.next()` (populate, V run: start seeds equal, 3.3). `spawn_monsters` (`ai.cpp:128-146`): `monster = make_monster(..., rng, ...)` with `rng` = **the caller's world stream** (`tick.cpp:708`, `objects.cpp:680`), then `unit_seed{spawn.seed}` for look, variant, props; `monster.seed = unit_seed` (`ai.cpp:160`).
- `make_monster` (`ai.cpp:2919-2941`) on the world stream: one draw per component layer (`:2929`), `unit.dir = rng(16)`, `wait_until = rng(4000)` (`:2932-2934`) and the stats, **life included** (`:2935` `monster_stats(..., rng, ...)`).
- The world stream: `d2d::rules::Rng rng{0x2545f491u}` in `World` (`world.hpp:349`), handed by reference to `Loot` and `Fight` (`:351-352`).

### 3.3 Unit seeds, game.exe against d2d (V run, `ss_unitseed_run.sh`, seeds 1..30 per level)

Game side: `tools/emu/ss_unitseed.py` hooks `FUN_00573cb0` (entry 0x573cb0, ESI unit, seed at unit+0x20/+0x24), the point before the life roll (0x573f7e) and the return (0x5742e9). d2d side: `tools/emu/ss_seed_test.cpp`, the real `populate_level` + `spawn_monsters`, printing `spawn.seed` (start) and `Monster::seed` (end). Compared per unit of rooms with equal unit counts (`ss_unitseed_cmp.py`); d2d boss units (`B`) counted apart.

| Level | game rooms with units / d2d | non-boss units | class equal | start seed equal | end: game = d2d end stepped once | end equal | other |
|---|---|---|---|---|---|---|---|
| 2 Blood Moor | 860 / 839 | 2,770 | 2,770 | 2,770 | 2,770 | 0 | 0 |
| 3 Cold Plains | 1,162, all rooms of equal unit count | 5,788 | 5,788 | 5,763 | 5,763 | 0 | 25 |
| 4 Stony Field | 1,044, all equal count | 4,147 | 4,147 | 4,096 | 4,096 | 0 | 51 |
| 8 Den of Evil | 475, all equal count | 2,006 | 2,006 | 1,976 | 1,976 | 0 | 30 |
| together | | 14,711 | 14,711 | 14,605 | **14,605** | **0** | 106 |

(Boss units: 0, 237, 488, 180 in levels 2, 3, 4, 8, not compared.) The game's room count in level 2 is larger than d2d's (860 against 839, 830 rooms of equal unit count) because this script lists every unit game.exe makes, including NPC-class rows that `monsters.py` and d2d's population leave out; every pair compared has the same class.

Reading: for every unit whose start seed is equal (14,605), game.exe's seed after `FUN_00573cb0` is **exactly one LCG step** after d2d's final unit seed: d2d's final seed equals game.exe's seed just before the life roll. The 106 others (0.7 percent) start from a different seed: game.exe's start seed there has a high word other than 666 (for example `bbb426e1.43bb630d`), i.e. not a plain game step: the unit takes its seed from elsewhere (a pack member seeded from its leader, I; cause not traced, U), where d2d takes `game.next()`. All 106 have a start-seed high word other than 666 and are Fallen (class 19: 85 units) or Carver (class 20: 21 units) members, in rooms whose first unit is a Fallen Shaman (class 58) (V run, `ss_unitseed_cmp.py`: 25 in Cold Plains, 51 in Stony Field, 30 in the Den, none in Blood Moor); later units of the room start equal again, so d2d's game stream is stepped as often as game.exe's. The shaman pack path is the likely source (I; the code of that path was not read).

### 3.4 Life: seed for seed and in distribution (V run, `ss_life_run.sh`, `ss_life_stats.py`)

Method: game.exe's real population (upstream `ms_spawn.py`, seeds 1..30, levels 2, 3, 4, 8, Normal, one player) gives each unit's life (stat 7 / 256); d2d's real `make_monster` (`tools/emu/ss_life_test.cpp`) is fed the same class sequence on **one** persistent world `Rng{0x2545f491}`, as d2d spawns them. 15,676 units, 14,771 plain (flags word & 0x1c = 0, the test of `ms_spawn_cmp.py`), 905 others left out; 41 plain units (the cows, MaxHP = MinHP = 0 in d2d's row) have no range and are left out of the tests: **14,730 compared**. Game life outside d2d's [min, max]: 0 of 14,771.

| Check | Result |
|---|---|
| life equal seed for seed | 3,060 of 14,730 (chance: 3,079.5) |
| two-sample KS on (life - min + 0.5) / span, game against d2d | D 0.0103, p 0.43 |
| 13 (class, level) groups with at least 100 units, two-sample KS p < 0.01 | 0 groups (0.1 expected by chance); smallest p 0.077 (class 0 at level 2, 677 units: means 9.05 and 8.88) |
| mean life, all compared units | game 6.229, d2d 6.246 |

Limits: d2d's stream here carries only the spawn draws (in the real game, combat rolls interleave, which only adds mixing); the discrete data make the KS p conservative; the test is Normal, one player, plain monsters; bosses, champions and Nightmare/Hell are not covered (`monster-stats-trace.md` D1 for the level column).

## 4. Where d2d's world stream serves unrelated actions (V read, sorcrest-d2d = upstream main)

| Consumer | Source | game.exe's own | Tag |
|---|---|---|---|
| Monster creation draws: component layers, direction, wait, **life** | `ai.cpp:2929-2935` (called from `tick.cpp:708` as rooms wake, `objects.cpp:680` for shrine/object monsters) | life on the unit seed (3.1); direction and wait: not traced | V read; U for dir/wait |
| Worm young, clones, "laid" monsters: unit seed = `Rng{rng.next()}` | `tick.cpp:337-339`, `ai.cpp:1515`, `ai.cpp:1581` | their unit seeds come from the game step like any unit (I) | V read |
| Every blow, block, crit, damage roll of the player, merc, pets and monsters; missile spots and splits; stun, knockback, potion finds, summon variant and angle | `fight/strike.cpp`, `fight/missiles.cpp`, `fight/pets.cpp`, `fight/states.cpp`, `fight/setup.cpp:195`, `fight/summons.cpp:152,163,213,306`, `fight/deaths.cpp:184,289`, `fight/tick.cpp:301,390-412,614-656` | on the attacker's / missile's unit seed (seed-study 7.4, D `re/combat.md`) | V read; distribution equivalence per rule is the sk-* traces' |
| Gamble buy | `world/commands.cpp:72` `store_gamble(..., rng)` | not traced | V read; U |
| Imbue (Charsi quest) | `world/commands.cpp:242` | not traced | V read; U |
| Gem shrine, shrine monsters and bosses | `world/objects.cpp:468-500` (the shrine monster count uses `spawning.objects` at `:667`, its spawns `game.next()` at `:679`, faithful) | object seed / game seed | V read; U for the first three |
| Cairn Stones' order, drawn from `rng.low, rng.high` at the first stone | `world/quests/shared.cpp:646-647` (the code's own comment `:601`: stands in for the quest seed at game +0x10f4) | the quest seed | V read; U |
| Quest drop codes, missile spots | `shared.cpp:113`, `:446` | quest seed | V read; U |

Not on the world stream (so equal in kind to game.exe's): vendor stock and hire lists on `spawning.proxy` (`commands.cpp:55-56`, `shared.cpp:343-349`), container contents on `spawning.objects` (`objects.cpp:81`), item and gold drops on the game seed plus the monster's unit seed (`loot.hpp:84-143`), merc offers from the hire entries' seeds (V run, 3.5).

Can a bot action order shift another outcome? Yes, in both backends, by different routes (I; the order effect is not run here):
- **Both**: every unit made steps the game seed, so the order rooms wake in (the route the bot walks) decides every later unit's start seed, loot seed and, on game.exe, life; the first vendor opened takes the first draws of the proxy stream (d2d keeps each vendor's stock after its first open, `commands.cpp:54`).
- **d2d only**: the number of combat rolls, gambles and woken monsters before an event moves that event's roll, because they share `rng`.
- **game.exe only**: a unit's rolls depend on its own seed and on what that unit did, not on other units' fights.
The statistical A:B is not touched by this (each outcome has the same distribution on both sides, 3.4), but a **paired** comparison per map seed will not pair outcomes below the world level (section 0 item 4).

### 3.5 Vendor stock and merc offers (V run, `ss_store.sh`)

`store.py` (game.exe's `FUN_00576980` per vendor, seeds `index * 0x9e3779b1`, 17 vendors x 6 jobs) against `drlg-dump store`: **20,400 of 20,400** lines equal (every item asked: code, quality, item level, and the proxy seed left), seeds 1..200. `hire.py` (the offer) against `drlg-dump hire`: **7,200 of 7,200** lines equal (200 seeds x 3 acts x 3 difficulties x 4 character levels). V run. Since the seed left is part of each line, chaining vendors in any order is the same chain on both sides (I from the run). Not compared: the 4-minute restock timer of game.exe (seed-study 7.3, U) against d2d's kept stock (`commands.cpp:54`): a stock that d2d keeps for a whole game may be rerolled by game.exe; drill T1.

## 5. The `.d2s` pin

- **Not run from a file.** The worker host's d2d save directory is empty and no character file is in the repository; no existing throwaway copy was readable. The seed entered the oracles as the map seed number (1..30, two 32-bit ranges, 0 and 4294967295).
- d2d (V read `main.cpp:378-379`): `if (!g_seed_fixed && header.difficulty[active] & 0x80 && header.map_id) return header.map_id;`, called at scene start (`:731`, `:1350`, `:1498`); `--seed` (`g_seed_fixed`) wins over the file; otherwise a random `std::random_device` seed (`:1689`).
- game.exe: the same field (d2s 0xAB), same bit (seed-study 1.1; D/V live in `docs/saves.md`). Edge: a saved id of 0 is reused by game.exe, not by d2d (S `re/deviations.md` Improvements #2). The generator at seed 0 equals game.exe's (V run, edge seeds: level 2 and 8 equal), so only the "file to seed" selection differs (D7, U).
- A pinned honest character therefore gives both backends the same world for the same file, **if** the bot starts d2d without `--seed` and without `--no-save` (B1).

## 6. Deviation list (d2d against game.exe)

| ID | Rule on both sides | Tag | Differential |
|---|---|---|---|
| D1 | Monster life: game.exe `min + rand(max - min + 1)` on the unit seed (`0x573f8f`); d2d on the shared world stream (`ai.cpp:2935`). Same function, same range, different stream: 3,060 of 14,730 lives equal (chance 3,079.5); distribution equal (KS p 0.43; 13 groups, none below p 0.01; mean ratio 1.0028). | V run | **Yes, built**: `ss_life_run.sh` + `ss_life_stats.py` (game side `ms_spawn.py`, d2d side the real `make_monster`). Statistical only; a seed-for-seed oracle would need life moved to the unit seed (the fix of D2). |
| D2 | Unit seed after creation: game.exe draws look, variant, **life**, then nothing; d2d draws look, variant, props and no life: d2d's final unit seed is game.exe's seed before the life roll; game.exe's is one LCG step later in 14,605 of 14,605 units with equal start. Everything that rolls on the unit seed later (death drop start, AI draws, blows) starts one draw apart. | V run | **Yes, built**: `ss_unitseed_run.sh` (`ss_unitseed.py` hooks `FUN_00573cb0`; `ss_seed_test.cpp`; `ss_unitseed_cmp.py`). A fix would make `spawn_monsters` roll life on `unit_seed` between variant and props; then before: 14,605 differ, after: expected 0 (not built, section 7). |
| D3 | 106 of 14,711 non-boss units (0.7 percent; Fallen and Carver pack members) start from a seed with a high word other than 666 in game.exe; d2d starts them from a game step. | V run (the 106); cause U | Yes, same harness; cause: print FUN_00555230's caller for those units. |
| D4 | The world stream serves unrelated actions (section 4). | V read | d2d side: a lockstep drill (T2); game.exe side by emulator is not needed (its rolls are per unit). |
| D5 | Gamble buy and imbue draw on the world stream. game.exe: not traced. | V read d2d; U game.exe | Could be built: run `FUN_` of the gamble (find it from the store code, `store.py` hooks) with a fixed proxy and unit seed against `store_gamble`; oracle = the item made. |
| D6 | Cairn Stones' order drawn from the world stream at the first stone (d2d, `shared.cpp:646-647`); game.exe: the quest seed (the code's own comment says so, U). | V read d2d; U game.exe | Could be built: quest init `FUN_00545d80` and the stones' operate function in the emulator, oracle = the five-stone order for a map seed. |
| D7 | Saved map id 0: d2d treats it as none (random seed), game.exe reuses it (S `re/deviations.md`). | S; the generator at seed 0 equal (V run) | Yes by d2s: a file with the difficulty bit and map id 0 loaded in the lab binary (drill T3), oracle `drlg.py 0 <level>`. |
| D8 | Monster initial direction and wait (`rng(16)`, `rng(4000)`, `ai.cpp:2932-2934`) on the world stream. game.exe: not traced (U). | V read d2d; U | Needs the creation path of `FUN_00555230` traced for those fields. |

Not deviations (checked and equal): layout, presets, room seeds, spawn list (section 2), the game, object and proxy seed chain (V run item 2), container drops, vendor stock, merc offers.

## 7. PR candidates

**No PR candidate.** D2 is proven by a harness that is new (`ss_unitseed_*`), not by an existing differential, and its fix (roll the life on `unit_seed` in `spawn_monsters`, drop the world-stream life draw) is not built or run in the private fork's checkout (not touched here). The existing differentials (`diff_drlg.py monsters/game`, `ms_spawn_cmp.py`) show 0 mismatches before and after because they do not read unit seeds. To make it a candidate: build the fix on a local branch of the fork checkout, run `ss_unitseed_run.sh` (before: 14,605 of 14,605 units one step apart; after: 0), plus `diff_drlg.py 1-30 <level> monsters game` and `ms_spawn_cmp.py` (stay 0), and hand the pair to an independent verifier. Note the monster-count visible effect is nil (D1 equal in distribution); the value is seed-for-seed pairing of drops and AI.

## 8. Bot findings (described, not changed)

| ID | File:line | Finding | Tag |
|---|---|---|---|
| B1 | `internal/d2dinput/lifecycle.go:90`, `cmd/d2bot/d2dbackend.go:59,144` | The d2d launch passes `--seed` only when the seed is non-zero; with 0 and no `--no-save`, d2d takes the loaded save's map id (V read, section 5), with `--no-save` it takes a random one. A pinned A:B arm must start d2d with seed 0 and the save loaded, or `--seed <the file's 0xAB>`; `--seed` overrides the file. | V read |
| B2 | `internal/saves/progress.go:181,194`, `internal/saves/snaplib.go:55,185,287` | The bot already reads header 0xAB (`MapID`) and the snapshot manifest carries and verifies it: the pin check for an A:B character exists (compare `MapID` of both arms' `.d2s` before each run). The `.map` file repeats the seed at offset 8 (`progress.go:244`). | V read |
| B3 | `internal/drill/dfield.go:413` | A drill puts the character at cell (25,69) because "Cold Plains' waypoint stands at about cell (23,69) on map seed 11": a layout constant of one seed. Under a pinned or random seed other than 11 it lands elsewhere. The layout is seed-for-seed equal on both backends (section 2), so with a pinned seed the constant is valid on both. | V read |
| B4 | `cmd/d2dspawn/main.go:129` | The spawn census row keys on the d2d `--seed`; the generator's whole-level count is the reference (`item3-monster-density.md`), the lab's lower count is the activation-by-distance of the census walk (U there). | V read |
| B5 | `docs/primitives.md:31` (P19) | The bot has no gamble code, so the gamble draws (D5) cannot yet shift a bot outcome; the town order (which vendor opens first) can (section 4), identically on both backends given an equal proxy seed (I). | V read |

## 9. Drill recipe for the fork verbs (U to confirm later; writing only, nothing run)

Lab: hidden slot, throwaway Sorceress `--no-save`, lockstep, as `docs/forkdbg.md` (vetting method) and `docs/d2d-fork.md` section 5 (`det.py`: `--lockstep --seed 3 --start-screen ingame --start-class 1`, `debug stat 12 8`, `debug goto 2`, steps of 250, `monsters` after each block); verbs `monsters`, `info`, `step N`, `dump player,time`, `debug critters`, `debug level`, `debug goto L`, `debug spawn` (`docs/forkdbg.md`, `docs/forkcheats.md`). Every `debug` call is ledgered, so these runs never count toward a gate. Check capacity first.

| Drill | Question | Steps | Oracle / expected |
|---|---|---|---|
| T1 | Does game.exe reroll a vendor's stock and does d2d keep it (D-store)? | d2d: `--seed 3`, open Akara's trade twice 5 virtual minutes apart (`step`), compare `dump` of the stock | d2d: identical (kept, `commands.cpp:54`); game.exe: U (seed-study 7.3 says it rerolls every 4 minutes): needs a live read, not the lab |
| T2 | Does a world-stream consumer shift a later roll (D4)? | Two lockstep runs, same `--seed 3`, same inputs except run B first does `debug goto 3` (wakes Cold Plains rooms) then `debug goto 2`: `monsters` of Blood Moor after 250 ticks in both | d2d predicted: monster positions equal, hit points of the Blood Moor monsters differ between A and B (their life is drawn from the stream, `tick.cpp:708`); equal hit points would refute D1's stream claim |
| T3 | Map id 0 and the file pin (D7, Q1) | Place a copy of a throwaway `.d2s` with the active-difficulty bit set and 0xAB = 0 (a setup edit, declared), start d2d without `--seed`; read the seed from the log and `debug level`; repeat with id N | d2d: id 0 gives a random layout, id N gives `drlg.py N 2`'s layout (diff the `monsters` positions against `ss_run.sh`-style oracle output for N); the game.exe side is the SEEDPIN probe of seed-study 6 (P3-0), live, not mine |
| T4 | Cows in the live camp (section 2.2) | d2d: `--seed 1`, in town `monsters` (cows are camp units, they may not list) and `debug level`; game.exe: the client monster table in Rogue Encampment on the pinned seed | emulator says 3 cows at subtiles (45,65), (142,68), (98,155) for seed 1; U live |
| T5 | Life distribution live (D1) | d2d: `--seed 3`, `debug goto 2`, `monsters` hit points per class, repeat over seeds 1..30 | compare with `ss_life_run.sh` output of game.exe for the same seeds: means per class equal within sampling error; not equal seed for seed |
| T6 | Unit seed exposure | none of the verbs shows a unit seed; a read-only `dump monsters` with `seed` would turn D2 into a live check | needs a new verb (a fork card), not available now |

## 10. Matrix-cell updates (proposed text; shared files left untouched)

`docs/coverage/` (written by this card, see section 11): new subsystem `levels.layout` (Act 1 areas 1-4, 8) with the two ledger rows, evidence this doc, cases 936 and 180.

Proposed rows for the sorceress-style matrices and for `docs/research/seed-study.md` (a status banner or the cells below, by its owner):

| Cell | Was | Now |
|---|---|---|
| seed-study.md section 0 item 4 and 7.2 ("d2d has one fixed-constant stream; life and drop rolls differ from the first monster") | RNG stream after level generation does not match | Partly out of date. The game, object, proxy and quest seeds and unit start seeds are the game's chain (V run: 30 games, 14,605 units). Differs: life stream (D1, equal in distribution), unit seed one step behind (D2), world-stream consumers (section 4). |
| seed-study.md section 7 item 3 (vendor stock, gambles, hire lists "differ from the first shop open") | differs | Stock and offers equal for the same proxy seed (V run 20,400 + 7,200 lines); gamble draws on the world stream (D5, U). |
| seed-study.md section 8 card 2/3 (SEEDLAYOUT, SEEDSPAWN) | open | Done offline for Act 1 outdoor levels and the camp (this doc); the live half (P3-1/P3-2 on a live game.exe) remains. |
| tierb-design-20261102.md A:B ("paired by map seed") | pair per map seed | Pair at the level of the world (layout, presets, spawn list); do not pair monster life, drops or fights below it. |
| `docs/coverage/rule-coverage.md`, `monster.stats` row | stats equal | add: unit seed after creation (D2) is a separate aspect; this card's `levels.layout` rows are about layout and spawn list only. |

## 11. Dead ends and what was not run

- **No `.d2s` read** (section 5); the file pin was not exercised.
- No live game.exe: the camp's cows, chickens, the vendor restock timer and every live-observation cell stay U (drills T1, T4).
- The `monsters` kind on level 1 (section 2.2): a tool limit, kept in the doc.
- Boss, champion, Nightmare and Hell life and seed paths not compared (905 of 15,676 units left out of D1; NM/Hell area-level mismatch is `monster-stats-trace.md` D1).
- Run order matters for nothing here; the d2d binaries are a hand build of upstream main, not the private fork (`I` that the fork equals it on these paths: its lockstep patch does not touch population or RNG, S `d2d-fork.md`).
- The four unit-seed runs shared one compile output name (parallel builds of `ss_seed_test`); all four finished and agree with the one-level smoke run, so no run was corrupted (I: they were the same source).

## 12. Shared-file lines for the merger

See `openspec/changes/seedspawn/merge-lines.txt` (BACKLOG, INDEX, CHANGELOG). Not edited here: `docs/decomp-queue.md`, `docs/BACKLOG.md`, `docs/INDEX.md`, `CHANGELOG.md`, `docs/research/README.md`.

## 13. Glossary terms missing (for the report, not added)

"world stream" (d2d's `World::rng`), "unit seed", "sunitproxy seed", "game seed chain".

## 14. Files of this card

`tools/emu/ss_build.sh` (hand build of `drlg-dump`), `ss_run.sh`, `ss_extra.sh`, `ss_edge.sh`, `ss_town.py`, `ss_store.sh`, `ss_life_build.sh`, `ss_life_test.cpp`, `ss_life_run.sh`, `ss_life_stats.py`, `ss_cppbuild.sh`, `ss_seed_test.cpp`, `ss_unitseed.py`, `ss_unitseed_cmp.py`, `ss_unitseed_run.sh`. Run each with `tools/emu/run-on-worker.sh [--ttl N] SCRIPT [ARGS]`.
