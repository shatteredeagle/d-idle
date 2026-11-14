# SHRINE: where shrines are, what they do, how the bot would use one, and whether it pays (decomp17)

> status: research, 2026-11-14, agent decomp17, card TASK-402. Offline, no game, no d2d slot, nothing posted. Docs and `tools/emu/shr_*` only.
> HONEST LIMIT: the emulator image of game.exe (the worker host's scratch MPQ folder with `bin/game.exe`, and the copy under the cast-trace scratch) was GONE this session (`/tmp` is volatile, the cookbook says do not search). So NOTHING below is "V run on game.exe" and I read no game.exe function myself. The game.exe side is **S**: the upstream d2d source comments and `docs/research/re/objects.md` of the d2d checkout (written from the game.exe decompile; they cite `FUN_` addresses, which I repeat as S and did not re-read). The d2d side is **V read** (the checkout's source) and **V run (d2d only)** (its real generator, section 4). Tables are **D** (patched 1.14d stack). Section 8 is the recipe that turns each S/U into V run once the emulator image is back.

Tags: V read = read in a source; V run (d2d) = d2d's real code run on a d2d build, no game.exe in the loop; D = MPQ tables; S = existing upstream proof/notes (not re-read by me); I = inferred; U = unverified.

## Answer in six lines

1. **Spawn.** A "Shrine" object picks its Shrines.txt row ONCE, when its room is built, from objects.txt `Parm0` (1 health shrine, 2 mana shrine, 3 "other": 10 % magic, 90 % boost class) and the Levels.txt **id** of the level (not its area level) against the row's `LevelMin`. Act 1 normal outdoors: **about 4.4 shrine objects per wilderness level** (V run d2d, 150 seeds), 0.6 of them health shrines. Caves 1.7 to 2.8 (Den of Evil 1.7), catacombs about 1.1, crypts 0.1 to 0.4, Cathedral 0.2 (section 4).
2. **Effects that matter** (D + S): Experience +50 % per kill for 144 s; Skill +2 to all skills 96 s; Refill full life and mana; health shrine full life; Combat +200 % damage and to-hit 96 s (only on level ids 8 and up); Armor +100 % AC 96 s (ids 5 and up); Stamina unlimited 192 s; Recharge +400 % mana regen 96 s; resists +75 % 144 s (fire 5+, poison 21+, cold 26+, lightning 32+). One boost at a time. Bad: the magic class (10 %): Storm, Exploding, Poison, Warping about 6 to 8 % of a "boost" shrine; Portal and Gem are harmless.
3. **Interaction.** A shrine is operated like a chest: the bot's `Interact` on the field-object row (V live, drill H.obj.shrine PASS on a class 81 shrine in Blood Moor, `docs/primitives.md`). d2d's `debug objects` gives the row, the kind and the mode of the same index `cmd interact` takes (V read, section 3). In game.exe the kind is NOT visible before use (U), in d2d it is, so the bot can skip bad rows on d2d.
4. **Respawn.** Experience never (Reset 0); Refill 2 "minutes", the rest 5 (S: Reset x 1200 + 1 frames, which is 48 s per unit at 25 frames/s: 96 s and 240 s).
5. **Value.** One Experience shrine at the first-hour kill rates is +250 to +1,100 XP; the whole first hour reaches about 2 Experience shrines if every one were found, so +2 to 4 % of an hour's XP. The XPGOLD gap to clvl 13 is 4.3x. Shrines do not change the first-hour level.
6. **Recommendation:** build the SMALL version (section 7), after the kill-rate and routing work, not before. No d2d deviation is proven; **no PR candidate**.

## 1. Questions answered

| Q | Question | Answer | Tag |
|---|---|---|---|
| Q1 | Which objects are shrines | objects.txt rows with InitFn 1 and OperateFn 2 (93 InitFn 1 rows, 92 of them OperateFn 2 + the obelisk row with OperateFn 17). Act 1 normal uses 2, 77, 81, 83, 84 (wilderness: 2, 81, 83 Parm0 3, 84 Parm0 1), 77 and 165 (crypts/barracks, Parm0 1 and 2), 260, 262, 263..265 (monastery, Parm0 3/1/2), 275, 276, 277 (caves, Parm0 1/2/3). All have `OperateRange` 2, `Selectable0` 1, `Restore` 1, mode 0 at start | D (objects.txt; the ids from the generated Act 1 objects of section 4) |
| Q2 | How the row is chosen | `FUN_0054f9d0` on the game's one object seed: Parm0 0 any row `1 + rand(count - 1)` (no Act 1 row uses 0); 1 class 2, 2 class 3, else one step `% 10 == 0` class 1 (magic), else class 4 (refill and boosters); a class picks `rand(n)` of its rows in row order; both re-roll while level id < LevelMin, **8 tries, the last roll stands even if too low**; exchange rows 4/5 become 2/3, Enirhs (16) becomes the gem shrine (18) | S (`rules::roll_shrine`, comment cites FUN_0054f9d0 / FUN_0054f770 / FUN_00546c60) + V read of the port |
| Q3 | Odds per row on an Act 1 level | section 2 table, computed from the class lists and LevelMins by `tools/emu/shr_count.py` | D + S (rule) |
| Q4 | Where shrines are placed | outdoors: 5 candidate cells per wilderness level, a shrine preset DS1 stamped in each free one (`FUN_00674e40`, Levels `SubShrine` 5; the DS1's object is one of 2/81/83/84); indoors: object groups with PopulateFn 2 (`FUN_00552b50`): at most 10 per level and one per 8 rooms, 50 subtiles apart, and late in a level with no health shrine yet it forces up to 30 tries and makes the next one a health shrine (`FUN_00552ac0`) | S (d2d `outdoor.hpp`, `objgroups.cpp` comments) |
| Q5 | How many per level, Act 1 normal | section 4 | V run (d2d only) |
| Q6 | What each row does | section 2 effect table | D (Shrines.txt) + S (functions) |
| Q7 | Stacking, merc, party | boosters are States.txt "curse = 1" states 128..137: a new one ends the old one (`FUN_0056e970`); the same one again only moves its expiry; only the operator gets it (S); Storm hits every player and monster in range, merc included; Recharge/Refill fills only the operator | S |
| Q8 | Activation | an object operated at OperateRange 2 with objdata +0xc == 0 and mode 0: mode 1 (`FUN_00624690`), message ShrMsg (string 3683 + row) over it for 157 frames, effect table at 0x6e1850 by Code, then a Reset timer (S: `FUN_00583c70`) | S |
| Q9 | Do shrines come back | rows with Reset > 0 return to mode 0 (virgin) Reset x 1200 + 1 frames after use in d2d; the game's timer 5 handler is not traced upstream (5 taken as the reset) | S + U |
| Q10 | How would the bot use one | section 3 and 7 | V read (bot) + V live (H.obj.shrine) |

## 2. The numbers

### 2.1 Shrines.txt (D, patched stack; columns by name)

Effect table, durations in frames at 25 frames/s (d2d: `Duration x 40` ms, V read `world.cpp` operate).

| Row | Name | Code | Arg0 / Arg1 | Duration (frames, seconds) | Reset | rarity col | Class | LevelMin | Paladin, hour 1 |
|---|---|---|---|---|---|---|---|---|---|
| 1 | Refill | 1 | 100 / 100 | 0 | 2 | 1 | 4 | 1 | take (life and mana to max) |
| 2 | Health Boost (the health shrine) | 2 | 200 / 400 | 0 | 5 | 2 | 2 | 1 | take, life set to max (S) |
| 3 | Mana Boost (the mana shrine) | 3 | 200 / 400 | 0 | 5 | 2 | 3 | 1 | take, mana to max |
| 4, 5 | Health / Mana Exchange | 4, 5 | 50 / 500 | 0 | 0 | 3 | 2, 3 | 2 | never placed: 4 becomes 2, 5 becomes 3 (S) |
| 6 | Armor Boost | 6 | +100 % AC | 2400 (96 s) | 5 | 2 | 4 | 5 | take |
| 7 | Combat Boost | 7 | +200 % to-hit, +200 % damage | 2400 (96 s) | 5 | 2 | 4 | 8 | take: triples damage |
| 8 | Resist Fire | 8 | +75 % | 3600 (144 s) | 5 | 2 | 4 | 5 | take |
| 9 | Resist Cold | 9 | +75 % | 3600 | 5 | 2 | 4 | 26 | take |
| 10 | Resist Lightning | 10 | +75 % | 3600 | 5 | 2 | 4 | 32 | take |
| 11 | Resist Poison | 11 | +75 %, poison length 0 | 3600 | 5 | 2 | 4 | 21 | take |
| 12 | Skill Boost | 12 | +2 all skills (state 134) | 2400 (96 s) | 5 | 2 | 4 | 1 | take |
| 13 | Recharge Boost | 13 | +400 % mana regen (stat 27) | 2400 | 5 | 2 | 4 | 1 | take (small for a Paladin) |
| 14 | Stamina Boost | 14 | unlimited stamina | 4800 (192 s) | 5 | 3 | 4 | 1 | take |
| 15 | Experience Boost | 15 | +50 % per kill (stat 0x55) | 3600 (144 s) | 0 | 3 | 4 | 1 | take, never respawns |
| 16 | Shrine of Enirhs | 16 | | 0 | 0 | 3 | 1 | 1 | becomes the gem shrine (S) |
| 17 | Portal to unknown | 17 | opens a town portal 5 subtiles from the player | 0 | 0 | 2 | 1 | 3 | harmless, wastes a detour |
| 18 | Gem Upgrade | 18 | first inventory gem up one grade, else a chipped gem | 0 | 0 | 3 | 1 | 4 | harmless, small gain |
| 19 | Storm Shrine | 19 | all players and monsters lose 50 % of CURRENT life, 16 fireballs (level clvl / 5) | 0 | 0 | 2 | 1 | 1 | SKIP |
| 20 | Warping Shrine | 20 | nearest monster becomes a unique | 0 | 0 | 3 | 1 | 3 | skip under clvl 10 |
| 21 | Exploding Shrine | 21 | 5 to 10 exploding potions at the feet + 6 explosions | 0 | 0 | 3 | 1 | 1 | SKIP (the potions are free throwables, U) |
| 22 | Poison Shrine | 22 | 5 to 10 poison potions + 6 clouds | 0 | 0 | 3 | 1 | 1 | SKIP |

The "rarity" column of Shrines.txt is NOT read by the row roll (the class pick is uniform over the class's rows; S: `roll_shrine` has no weights). `Reset` units are 1200 frames each in d2d (S: `FUN_00583c70` timer 5), not minutes. The "(OUT)" rows 2 to 5 are marked unused in the table but row 2 and 3 are what Parm0 1 and 2 give (class 2 and 3 lists are rows 2, 4 and 3, 5; the exchanges are mapped away).

Names: the SME's "Fire / Light / Poison shrine" are rows 21 (Exploding), 19 (Storm), 22 (Poison); "Monster" is row 20 (Warping). Row names are the table's, not the in-game tooltip (U).

### 2.2 Odds of a row for a Parm0 3 shrine (D + S rule; `python3 -I tools/emu/shr_count.py DUMP OBJECTS SHRINES LEVELS`)

A Parm0 3 shrine is 10 % magic (rows 16..22 in LevelMin order), 90 % boost class (rows 1, 6..15, 11 rows). Exact enumeration of 8 tries with the LevelMin re-roll:

| Level id | Refill | Skill | Recharge | Stamina | Experience | Armor | Fire res | Combat | Gem | Storm | Exploding | Poison | Warping | Portal |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 2 Blood Moor | 17.9 | 17.9 | 17.9 | 17.9 | 17.9 | 0.1 | 0.1 | 0.1 | 2.5 | 2.5 | 2.5 | 2.5 | | |
| 3, 4 Cold Plains, Stony Field | 17.9 | 17.9 | 17.9 | 17.9 | 17.9 | 0.1 | 0.1 | 0.1 | 1.7 / 2.9 | 1.7 / 1.4 | 1.7 / 1.4 | 1.7 / 1.4 | 1.7 / 1.4 | 1.7 / 1.4 |
| 5, 6 Dark Wood, Black Marsh | 12.9 | 12.9 | 12.9 | 12.9 | 12.9 | 12.9 | 12.9 | | 2.9 | 1.4 | 1.4 | 1.4 | 1.4 | 1.4 |
| 8 and up (Den of Evil on) | 11.2 | 11.2 | 11.2 | 11.2 | 11.2 | 11.2 | 11.2 | 11.2 | 2.9 | 1.4 | 1.4 | 1.4 | 1.4 | 1.4 |

Percent of Parm0 3 shrines. The 0.1 values are the 8-low-rolls tail (a row below its LevelMin that survives all re-rolls). Bad (Storm + Exploding + Poison + Warping) per Parm0 3 shrine: 7.5 % on id 2 (Warping not yet allowed), 6.8 % on id 3, 5.6 % on id 4 and up. Parm0 1 and 2 shrines are always the health and mana shrine (100 %).

## 3. Interaction: d2d vs game.exe vs the bot

- **d2d, V read** (checkout source): `cmd interact <index>` sets the player walking to the object (`World::apply` Interact, `walk_to`) and the object is usable when it is an operable kind, not preoperated and not in `operated`. When the player is within **2 cells** (`hypot < 2`) `World::operate` runs (comment "ponytail: 2 cells, not D2's per-object operate range"). `operate` on a shrine: `shrine_recharge`, then `shrine_boost` into `actor->boost` with `now + Duration x 40 ms`; stamina fill; sounds; the row's special action for 17..22; `operated[...]` records the time; the frame loop sets mode OP, then ON, then back to NU after the Reset time, and the message ShrMsg (3683 + row) for 157 frames. A shrine already `operated` is not offered again (`usable`).
- **`debug objects`** (d2d devctl verb, V read in `apps/d2d/devctl_verbs.hpp`, parsed by `internal/d2dctl/parse.go:439-460`, listed by `internal/d2dctl/verbs.go:405`): one line per operable object `<npc index> <x> <y> <shrine|chest|opN> <Shrines.txt row | trap | object id> <locked|-> <mode>`. The index is the **same `level->npcs` index** that `cmd interact` takes (V read: both loops index `level->npcs[i]`). `docs/fixobj.md` and `internal/d2dsrc/items.go:315` still say the mode is "under another index" (U): from the source it is the same one. `debug operate <i> [row]` is a lab verb that forces an operate and a row.
- **Bot, V read:** `internal/d2dsrc/items.go:281-316` lists `debug presets` rows as snapshot objects (Class = objects.txt id, ID = NPCIDBase + preset index, name "Shrine") and NEVER reads `debug objects`, so the snapshot knows a shrine's place and class id (2, 81, 83, 84, ...) but not its row and not whether it is spent (`rp.unknown("object.mode", ...)` at line 315). `internal/d2dinput/controller.go:463-483` `Interact` sends `cmd interact <index>` for ids in the NPC range, objects included (`docs/fixobj.md`). Drill row `internal/drill/dfield.go:127` `H.obj.shrine` PASSED: "the bot's Interact operates ... a shrine (class 81)" (`docs/primitives.md:143`, V live).
- **game.exe, S/U:** the object-operate click goes through the same object path as a chest (OperateFn 2 vs 4, `FUN_00583c70` vs `FUN_00585f60`, S). The row IS known client-side for codes 6 to 15 (the client table at 0x6da8c0 puts the shrine's icon overlays on the object at init, S) so the kind may be visible on game.exe too (U); the tooltip names it only after use (U).
- **Reach:** objects.txt `OperateRange` is 2 for every shrine (D), d2d uses 2 cells for everything. Same number; the unit game.exe uses is U (D1).

## 4. Counts per level, Act 1 normal (V run, d2d's real generator only)

Command (worker host, 150 seeds, normal, `ORDER=shuffle`, levels 2..37 in turn, the same `drlg-dump ... game` mode the upstream differential `objgroups.py` compares with game.exe: S that it matches for placement; I did not re-run that diff):
`tools/emu/run-on-worker.sh --ttl 900 shr_dump.sh 1 150 > dump.txt` then `python3 -I tools/emu/shr_count.py dump.txt objects.txt shrines.txt levels.txt` (tables from `tools/emu/mpq-table.sh`). A shrine object = objects.txt InitFn 1 and OperateFn 2.

| Level id | Name | shrines per game (mean) | of them Parm0 1 / 2 / 3 | games with none |
|---|---|---|---|---|
| 2 to 7 | Wilderness 1..6 (Blood Moor ... Tamoe) | 4.35 / 4.39 / 4.38 / 4.36 / 4.46 / 4.35 | 0.6 / 0 / 3.75 each | 0 of 150 each, max 5 |
| 8 | Cave 1 (Den of Evil) | 1.67 | 0.57 / 0.60 / 0.51 | 0 |
| 9 to 12 | Cave 2..5 | 2.15 / 2.80 / 2.05 / 2.05 | about 0.7 / 0.7 / 0.7 | 1 of 150 each |
| 13 to 16 | Cave treasure levels | 0.37 / 0.27 / 0.45 / 0.23 | | 84 to 116 |
| 18, 19, 21 to 24 | Crypts, Forgotten levels | 0.1 to 0.4 | mostly health | 97 to 137 |
| 27 to 31 | Courtyard, Barracks, Jails | 0.68 to 1.13 | | 23 to 64 |
| 33 | Cathedral | 0.24 | | 117 |
| 34 to 36 | Catacombs 1..3 | 0.93 / 1.08 / 1.24 | | 18 to 43 |
| 37 | Catacombs 4 (Andariel) | 0.05 | | 142 |

Expected shrines you can use per visit (shrines per game x row odds, section 2.2): Experience on level 2, 3, 4: 3.78 x 17.9 % = **0.68, 0.67, 0.67 each**; Dark Wood and Black Marsh 0.48; the Den 0.06. Refill the same as Experience. Health shrine 0.6 per wilderness level. The count is per GAME, not per visit: shrines are placed once and the object stays; Experience is gone for good after use.

Why 4.4: five candidate cells per wilderness level, nearly all free (`outdoor.hpp` `shrines(5)`), each stamped with a shrine DS1 (S). This is high against what a player remembers (U: checked against nothing but d2d itself; the game.exe diff of the DRLG presets is upstream's, S).

## 5. Value for the first hour

Inputs (V run + D, `docs/research/xp-gold-trace.md`): plain kill 22.0 XP (Blood Moor), 25.6 (Cold Plains), 37.3 (Stony Field) at clvl 1..6; ideal route 14.6 kills/min; clvl 13 needs 112,725 XP; hour-1 totals 17,000 observed, 26,000 ideal route.

- Experience shrine = +50 % of every kill award for 144 s = 2.4 min (stat 0x55 in the award, S/V read: `docs/research/xp-award-trace.md`). At 8 kills/min x 25.6: +246 XP; at 14.6/min: +449; at 25/min on Stony Field (37.3): +1,119.
- Expected Experience shrines met in the first hour if every one on the route were found: Blood Moor 0.68 + Cold Plains 0.67 + Stony Field 0.67 = 2.0. Realistically half: +250 to +450 XP of 17,000 to 26,000, **1 to 3 %** (4 % if all found). (I)
- A detour of about 20 s (40 cells, there and back) costs 20 s x 14.6 kills/min x 25.6 / 60 = 125 XP of farming, so an Experience shrine pays 2 to 9 times its detour; a Refill or a health shrine pays in potions (1 potion per 17 kills, `xp-gold-trace.md`).
- Combat Boost (+200 % damage, 96 s) and Armor need level id 8 and 5; they appear from the Den and Dark Wood on, 11 % each there. They matter more than XP in a hard pack; not in the hour-1 numbers (I).

Not a level lever: the clvl 13 gap is 4.3x the ideal route; shrines are about 3 % of that.

## 6. Deviations d2d vs game.exe (all U or S; none proven)

| D | Rule, game.exe | Rule, d2d | Tag | Differential |
|---|---|---|---|---|
| D1 | operate reach: objects.txt `OperateRange` (2 for shrines), checked by the server operate path (function not read) | fixed `hypot < 2` cells for every object | U | yes: run the game's server-side operate check for a shrine object at distances 0..4 tiles on a built fake game (the `objgroups.py` populate harness), compare with d2d's `hypot < 2.f` over a distance grid; the unit (subtile vs tile) is the question |
| D2 | Reset: timer 5 handler not traced (S: upstream took it as "back to NU") | NU again after Reset x 1200 + 1 frames | U | yes: run `FUN_00583c70` then the timer queue on a fake object, count frames until mode 0 |
| D3 | Storm/Exploding/Poison missiles hit players and monsters (S: Storm "all players and monsters"; missiles friendly flags not read) | missiles hit monsters only, poison cloud one burst (upstream's own ponytail list) | S (known simplification) | no: needs the missile engine; stat deltas of a player under a Storm are the cheap part |
| D4 | row roll `FUN_0054f9d0` incl. 8-try re-roll | `rules::roll_shrine` | S (upstream "emu-checked" for the object seed stream, not the row) | yes: emulate `FUN_0054f9d0` for every (Parm0, level id) with the same seed stream as `shr_count.py`, 100,000 seeds per cell, compare the row histogram with `roll_shrine` through a d2d probe (the cit_d2d_test pattern). Needs the fake game struct with the Shrines table |
| D5 | effect functions 0x582860..0x583bf0 by Code (stats written, state, expiry) | `shrine_boost` / `shrine_recharge` | S | yes: call the table entry for Code 1..15 on a fake player, compare the stat list; same pattern as `sk_pass_*` |
| D6 | Experience: stat 0x55 added in the award `FUN_0057e480` | `exp += exp x psum / 100` | S (XPGOLD port equals game.exe on 119,790 lines but with stat 0x55 = 0; "the experience shrine, 0 here" in xp-award-trace.md) | yes: rerun `tools/emu/kill_exp.py`-style grid with stat 0x55 = 50 |

D4 and D5 are the ones a PR could rest on. **No PR candidate**: nothing is proven (no emulator image), and `docs/upstream/pr-queue.md` gets no draft.

## 7. Build or not, and the minimal design

**Recommendation: build the small version AFTER the kill-rate and routing cards, as an optional, default-on-for-d2d module.** Expected gain +2 to +4 % hour-1 XP, free heals, and early Combat/Skill boosts in caves; cost about 150 lines of Go plus one data table; risk limited by skipping bad rows (zero if the row is known). Do NOT build for game.exe parity runs: the row is probably not readable there (U).

Minimal design (nothing built here, B findings below):
1. **Recognise** (data, not Go): the set of shrine object ids is the objects.txt rows with InitFn 1 and OperateFn 2 (the bot already parses these in `internal/data/items.go:148-188`; the `ShrineFunction` column it reads is NOT the discriminator, all shrine rows have 0 there: read `InitFn`/`OperateFn` instead). Act 1 ids seen: 2, 77, 81, 83, 84, 165, 260, 262..265, 275, 276, 277.
2. **Read the row**: for each snapshot object whose class is in the set, read `debug objects` once per area entry and after every 50 subtiles (join by npc index, which is the preset index). Fields: row (Shrines.txt row) and mode (NU virgin, OP/ON used). A used shrine (mode OP/ON) is not offered again; Experience never returns, others return after Reset.
3. **Take / skip table** (build profile data, with today's values as defaults): take rows 1, 2, 3, 6..15 and 18; skip 17 (portal), 19 (storm), 21, 22 and 20 under clvl 10; resist rows are optional (take, they are free). Under clvl 10 skip nothing else. Never take at all when `row` unknown (the real client).
4. **Interact**: walk to the shrine, one `cmd interact <index>`; do it only when no hostile within about 20 cells (a shrine message and the effects are lost on death), life above 50 % for rows that can hurt, and not while a UI panel is open. Success = the object's mode leaves NU, or the player's state list gains 128..137 (States.txt shrine states; `internal/d2/skilllevel.go:15` already knows 134).
5. **Priority** just below gold pickup: within about 25 cells of the route for Experience, Skill, Combat, Armor; within 15 for Refill and Stamina; the health shrine only when life is under 60 % (it is a free heal, keep it for when it pays, but it does not reset for 240 s). Detour budget 20 s per shrine.
6. **Log**: a first-event line of kind `shrine` (`internal/decide/firsts.go:57` already has that kind), with the row name.

## 8. Drill recipe (the fork verbs; running a game was not allowed here)

Binary and verbs: `docs/forkdbg.md`, `docs/d2d-fork.md` (lab binary), `docs/forkcheats.md`. One hidden slot, a throwaway character, `debug goto 2`, lockstep, reasons in the ledger.

| U | Recipe |
|---|---|
| spawn counts (section 4) on the product binary | `debug goto 2` ... `debug goto 7`, `debug objects` each time; count lines of kind shrine, 10 fresh games; compare with 4.4 per level |
| row visible before use (D "row known") | `debug objects` on a fresh Blood Moor: the `row` column of a shrine, then `debug operate <i>` and read `dump stats` raw values and `debug states` |
| D1 reach | `cmd interact <i>` from distance 1.9 / 2.0 / 2.1 cells (a lab `debug teleport` or `cmd move` then `step`); log whether `debug objects` mode leaves NU |
| D2 reset | `debug operate <i> 1` (row 1, Reset 2), `step` 2400 ticks, `debug objects` mode; row 15 never returns |
| D3 | `debug operate <i> 19` (Storm) with a monster near and the merc out, `dump stats` life of player, merc and monster |
| effect numbers | `debug operate <i> <row>` for 6, 7, 12, 15 on a fresh Paladin; `dump stats` armor, tohit, damage percent, all-skills +2, add-experience 50; then `step` the Duration frames and confirm they end |
| bot path | `d2drill -group H -only H.obj.shrine` (exists, PASS); extend with `debug objects` to assert the mode changed |

Emulator once the image is back: section 6 D4/D5/D6 with `tools/emu/run-on-worker.sh` (same venv as `sk_pass_*`), disassembly via `cit_dis.sh 54f9d0 583c70 583b30`.

## 9. Bot findings (described, NOT changed)

| B | Finding | File:line |
|---|---|---|
| B1 | no shrine logic anywhere: the bot never reads `debug objects`, so it knows neither the Shrines.txt row nor whether a shrine is spent | `internal/d2dsrc/items.go:281-316` (reads presets only, `object.mode` unknown at 315) |
| B2 | parses the shrine kind and row but nothing consumes it | `internal/d2dctl/parse.go:439-460`, `internal/d2dctl/verbs.go:405` |
| B3 | `data.Object.ShrineFunction` (objects.txt "ShrineFunction") is 0 on every shrine row; the discriminator is InitFn 1 and OperateFn 2, which `Object` does not carry | `internal/data/items.go:158-188` |
| B4 | the loot rule never opens or operates anything ("chests stay shut"), so a shrine is only used when a quest/field primitive asks for it | `internal/loot/pickup.go:29`, `cmd/d2bot/objects.go:52-96` (`objSpecs` has waypoint-class and quest objects only) |
| B5 | gold-change logging counts a shrine's gold effects as pickup ("a shrine or a refund inside the field would count too"); fine as logging, wrong if a shrine module is added | `internal/bot/areametrics.go:10` |
| B6 | the doc claims `debug objects` is "under another index"; from the d2d source it is the same npc index `cmd interact` takes (V read), so a join is trivial | `docs/fixobj.md`, `internal/d2dsrc/items.go:315` |

## 10. Matrix cell and doc updates (proposed text; the owners apply them)

- `docs/levelplan.md`, section "Shrines: finding and proposal": replace "Proposal (not built)" point 2 with: "Row names are Shrines.txt rows; take 1, 2, 3, 6..15 and 18, skip 17, 19, 20 (under clvl 10), 21, 22. LevelMin compares the Levels.txt id, not the area level (Combat from id 8, Armor and Fire resist from 5, Poison 21, Cold 26, Lightning 32)." and add: "Wilderness has about 4.4 shrines per level (d2d generator, 150 seeds), 17.9 % Experience on ids 2..4; one Experience shrine is +250 to +1,100 XP at hour-1 kill rates, 1 to 4 % of the hour. See docs/research/shrine-trace.md."
- A future `docs/research/pal-skill-matrix.md` row "shrine Skill Boost": +2 to all skills for 96 s (state 134), Zeal level +2 while it lasts; value U.
- `docs/primitives.md` row P39: the claim "no code path chooses a shrine to use" stays true; add "the row is readable through `debug objects`".

## Dead ends

- The emulator image (`bin/game.exe` in the scratch MPQ folder) does not exist this session, so no `FUN_` of game.exe was read or run by me. Every address above is quoted from the d2d checkout's own comments (S).
- A first `shr_find.py` (string xref finder for game.exe) was written and never run for that reason.
- The objects.txt column "ShrineFunction" is not the shrine flag (all 0); do not use it.

## Reproduce

`tools/emu/shr_dump.sh` (seeds FIRST LAST, via `tools/emu/run-on-worker.sh`), `tools/emu/shr_count.py` (counts and odds), `tools/emu/shr_find.py` (needs the emulator image), tables via `tools/emu/mpq-table.sh shrines|objects|levels|objgroup`.
