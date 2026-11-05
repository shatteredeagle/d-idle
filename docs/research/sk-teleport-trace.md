# Teleport: game.exe 1.14d vs d2d vs the bot (decomp row 16, SKTELE)

> status: card sktele, 2026-11-01, offline (no game, no d2d slot). Decomp row 16 (SKTELE, sorc-skill-matrix rank 3): how game.exe refuses, places and finishes the player's Teleport (skill 54), what d2d does instead, and what the bot's nav hops assume. Follows docs/research/sk-missile-trace.md (SKMISS) and the SKMANA trace (branch agent/skmana, docs/research/sk-mana-trace.md, not merged when this was written). Telekinesis is listed only where it shares code.

Tags: **V read** disassembled with upstream's `tools/emu/disasm.py` by this agent (addresses given); **V run** run natively in game.exe under the emulator by this agent (`tools/emu/sk_tele_spot.py`); **D** from the patched 1.14d tables (Skills.txt, Levels.txt, PetType.txt read with `mpq-cat` from the patch stack, column offsets with upstream's `txt_fields.py`); **S** d2d source (upstream main 72132b99) or an earlier trace of ours (named); **I** inference; **U** not verified.

## Summary

- **Refusals before anything happens (V read + D).** Teleport has no srvstfunc, no `InTown` (so the start FUN_0056f640 refuses it in any town room, before the mana is touched), no `lineofsight` restriction, no `delay`, range `none`. The do (srvdofunc 27, FUN_005ca360) refuses when the level's Levels.txt `Teleport` is 0, or 2 and the line from the caster to the CLICKED point is blocked (FUN_006229f0, mask 0x804). Every Levels.txt row is 1 except Duriel's Lair (73) = 2 (D): **the lair allows Teleport along a clear line; it does not forbid it.**
- **Range is a room rule, not a distance (V run).** FUN_00554ea0 accepts the target only if it lies in the caster's room or in a room of that room's near list (FUN_00463740 over FUN_00619790); else nothing happens. With outdoor rooms of 8 x 8 tiles (40 subtiles) that is at least 40 subtiles along each axis and up to about 80, depending on where in its room the caster stands. Skills.txt has no range column for it, and nothing scales with the skill level.
- **The landing spot snaps (V run).** The point itself if a player fits there (plus-shaped footprint, mask 0x1c09: walls 0x1, player walls 0x8, objects 0x400, doors 0x800 and **units 0x1000**); else the first Chebyshev ring (out to 50) holding a free spot, the free spot of least |dx| + |dy| in it, ties to the first in scan order from (-r, -r). A blocked click or a click on a monster lands next to it.
- **A refused Teleport is free (V read).** The do returns 0 on every refusal and has no missile, so FUN_0056f7f0 returns before the payment (SKMANA's read of 0x56fa3d); the SC cast still plays. Mana is paid only for a hop that moved (24 - (L - 1) points, at the action frame: SKMANA).
- **On arrival (V read, outline):** the unit's room is updated (FUN_00554670), flag 0x10000 on unit +0xc8 (the client's reassign), and for a player: the landing goes into the 20-step ring (player data +0xa0 / +0xa8), the warp spot +0x148 is set and cleared 50 frames later (timer event 0xe, FUN_00554570), and **every pet whose PetType `warp` is set (the hireling included, D) is put at the landing spot at once** (FUN_005754b0 -> FUN_00574d90 -> FUN_00574cc0). No monster is told anything: aggro comes from the monsters' own AI.
- **d2d (S):** `spot()` puts the player AT the clicked point if `unit_blocked` (mask 0x0c09, no units) is clear, at any distance on the level, else does nothing, and the mana was paid at the command either way. Deviations D1-D7 below; D1-D3 are measured on the oracle against a line copy of d2d's rule (100,000 cases, section 6).
- **Bot:** the 18-subtile hop is the game.exe backend's screen limit, not the skill's (B1); on d2d a refused hop costs 24 mana that game.exe would not take (B2); Duriel's Lair is read as "no Teleport" in two places (B6). Findings B1-B7, nothing changed.
- **PR candidate: none.** The d2d side of the comparison is a line copy, not the real build; a C++ test mode on a real Level is the next step (section 7).

## 1. game.exe: the chain

| Address | Rule | Tag |
|---|---|---|
| Skills.txt row 54 | srvdofunc 27, no srvstfunc, `InTown` empty (0), `lineofsight` empty, `delay` empty, range `none`, anim SC, `interrupt` 1, `warp` 1 (bit 0x26 of +4: who reads it was not found, U), mana 24 / lvlmana -1 / manashift 8 / minmana 1, reqlevel 18, reqskill Telekinesis | D (+ offsets V `txt_fields.py` FUN_00613f80: InTown bit 8 of +4, lineofsight +0x18f, warp bit 0x26 of +4) |
| `FUN_0056f640` (skill start) | FUN_0056c160 mana check (SKMANA); then `InTown` (+5 bit 0) clear and the unit's room a town (FUN_0061ab00): refused, the player back to neutral (FUN_005809d0), no mana taken. `lineofsight` (+0x18f) 1..5 would test the target with FUN_00645950 and mask 4 / 0x1c09 / 0x180 / 0x804 / 0x805: 0 for Teleport, skipped. No srvstfunc: no payment here | V read (0x56f640..0x56f7af) |
| `FUN_005ca360` (srvdofunc 27, table 0x7322b0 entry 27) | target point FUN_0056d2c0; the caster's room FUN_00620bb0 -> FUN_0061a1b0 -> FUN_0061db70 (the Levels row), byte +4 (`Teleport`): 0 returns 0; 2 and FUN_006229f0(caster, x, y, 0x804) non-zero (blocked) returns 0; else returns FUN_00554ea0(game, caster, room 0, x, y, 0, 0) | V read (0x5ca360..0x5ca3dd); the byte +4 is `Teleport` by the d2d comment (S, gamedata.hpp:187) and by the oracle: 0 refuses, 1 and 2 land (V run) |
| `FUN_0056d2c0` (target point) | the cast's target unit if any (FUN_00553540): its position (static path +0xc / +0x10 for types 2, 4, 5; FUN_006488c0 / FUN_00648900 for the rest); else the path's target (FUN_00648a00 / FUN_00648a10); 0 when either coordinate is 0 | V read |
| `FUN_006229f0` (line test) | from the caster's subtile (path) to the point, with the caster's size (FUN_00620510): FUN_00622920 returns "clear" when the two are closer than the sizes' sum, else trims both ends by the size and walks the line with FUN_0064e260(room, from, to, mask) | V read (outline); FUN_0064e260's walk not read (U) |
| `FUN_00554ea0` (warp a unit; args room, x, y, no-snap flag, flag) | room = FUN_00463740(the caster's room, x, y): the room itself if it holds (x, y), else the first room of its near list (FUN_00619790) holding it, else **0: return 0**. Then (no-snap flag 0) FUN_0064e7b0(room, &xy, size FUN_00620510 (2 for a player, jump table 0x62057c), mask 0x1c09, 0) -> FUN_0064dea0(..., rings 50, step 1): the nearest free spot; none: return 0. FUN_00650be0(path, unit, room, x, y) sets the position; 0: return 0 | V read + V run (section 6) |
| player tail of FUN_00554ea0 | FUN_005531c0 (event 0xc3c), FUN_00619730 / FUN_0053bc50 (room coordinates to the client), FUN_0064c040, +0xc8 \|= 0x10000 (flag arg 0; 0x800 with it), FUN_00554670 (room bookkeeping), player data (FUN_006221a0) +0x148 / +0x14c = the landing (the warp spot), GetTickCount at +0xa4, the landing written into the 20-entry ring at +0xa8 (cursor byte +0xa0, wraps at 0x14), timer event 0xe at frame + 50 calling FUN_00554570 (clears +0x148 / +0x14c), FUN_005754b0(landing) | V read (0x554f65..0x555067) |
| `FUN_005754b0` (pets follow) | for each pet type 1.. (PetType table, 0xe0 a row, flags byte +4): `warp` (bit 0) -> FUN_00574d90: each pet of that type -> FUN_00574cc0: FUN_00650be0 into the owner's room at the landing x, y, room bookkeeping, +0xc8 \|= 0x10000, FUN_00648c30(path, 0x100); else bit 4 -> FUN_00575380(game, owner, 0x640); else FUN_00574c60 | V read (outline; the pets' own placement after FUN_00650be0 not read, U) |
| PetType.txt | `warp` 1 on single, valkyrie, golem, skeleton, skeletonmage, revive, **hireable**, invis, raven, spiritwolf, fenris, totem, vine, grizzly, shadowwarrior; 0 on dopplezon and the traps | D |

Telekinesis (43): srvstfunc 12 (table 0x732140 entry 12, 0x5c91c0), srvdofunc 21 (0x5c9b50), `InTown` 1, Param1 25 "on-screen cast range" (D); not read further (the bot never casts it; SKPASS row 23).

Not found: any distance clamp on the server side; any use of the skill level in the destination; any monster notification in the warp (V read of FUN_00554ea0's calls). The client side (what the click can reach on screen, whether the client refuses a far click itself) was not traced (U).

## 2. The destination rule, measured (V run, `tools/emu/sk_tele_spot.py probe`)

World: 5 x 5 rooms of 40 x 40 subtiles, caster at (100, 100) in room (2, 2) = [80, 120). Its near list covers [40, 160) on both axes.

| Case | game.exe |
|---|---|
| open ground, target (120, 100) / (79, 100) / (40, 40) / (159, 159) | lands there |
| target (39, 39), (160, 160), (199, 100), (-1, 100): outside every near room | do 0, nothing moves |
| target subtile blocked by 0x1 / 0x8 / 0x400 / 0x800 / 0x1000 | lands (109, 99): the corner of ring 1 |
| target word 0x2 / 0x4 / 0x10 / 0x100 / 0x2000 | lands on the target (not in the mask) |
| a wall one subtile east of the target / two east | lands (109, 100) / on the target: the footprint reaches one subtile, a plus |
| blocked squares of radius 1, 2, 3, 5 round the target | (108, 98), (107, 97), (106, 96), (104, 94): ring r + 1 corner, before ring r + 2's straight spots of smaller \|dx\| + \|dy\| |
| Levels Teleport 0 | do 0 |
| Levels Teleport 2 | FUN_006229f0 called with (110, 100), mask 0x804: the clicked point, before the snap |

So the search is Chebyshev ring first (out to 50), then least |dx| + |dy| in the ring, ties to the scan order starting at (-r, -r) (V run). Mask 0x1c09 (V run).

## 3. d2d's version (S, upstream main 72132b99)

- Command: `World` (components/game/world/commands.cpp:609-628): not `InTown` and in town: the "not in town" cue, no cast; a live monster under the click: walk up to it and attack with the skill (`attack_skill = 54`), then cast at it from where the attack path decides (tick.cpp:153); else `Fight::cast_missile` (fight/missiles.cpp:218): mana check and **payment at the command**, SC.
- Action frame: `Fight::spot` (fight/strike.cpp:661, Teleport branch :703-750): refused (break) when `in_town(level)`, `level->unit_blocked(x, y)` (plus footprint, mask 0x0c09: no units; the Crowd is not consulted), Levels `Teleport` 0, or 2 and `clear_line` (samples every 0.2 cell with `unit_blocked`); else the player is put at exactly (x, y), path cleared, `reassigned` (the client's 0x15).
- Not done on a Teleport: no room test (`near_rooms` exists, gamedata.hpp:1105, unused here), no snap (`nearest_free` exists, used by Leap, but searches in its own order: rings at 0.2 cell, first free in a fixed side order, out to 200), no ring entry (the next tick's footstep rule records a step over about 6.7 subtiles, world/tick.cpp:366), no warp spot (`warp_spot` is set by level travel only and never cleared, world/travel.cpp:155), no pet or merc move (they follow by their own think: the merc's leash, docs/decomp-queue.md row 8), rooms come up on the next tick as an ordinary move (`rooms_up(..., arrived false)`, world/tick.cpp:442).
- Lockstep: as for every skill, one 40 ms frame per step; the hop lands on the SC action frame in both.

## 4. The bot's use (our repo)

- `internal/nav/teleport.go`: `defaultTeleportRange` 18 subtiles ("about 18 subtiles ... V live, docs/act2.md"), `defaultTeleportMana` 24 (upper bound), `teleportSettle` 700 ms, `teleportManaWait` 40 s (Refill or 1 s sleeps), `teleportMinHop` 4, three hops that did not move end it; `hopCell` picks the known walkable cell within range closest to the target and at least 2 nearer; `hopAt` waits for mana, `CastAt(F2, cell)`, sleeps 700 ms, counts a move of 2+ subtiles, else marks the cell bad.
- `internal/nav/teleport_islands.go`: A* over known walkable cells joined within cast range (Arcane Sanctuary), blind casts at unknown cells on the target's side, budget 12 blind failures, 80 hops, stall after 8 hops without 2 subtiles of progress.
- Backends: game.exe: key + right click on the screen pixel of the cell, clamped to the screen (`internal/input`); d2d: `cmd skill 54 x y` with unit 0 (`internal/d2dinput/controller.go:262` -> `Cast(id, w, 0)`), the ground-target form docs/d2d-backend.md:416 left open; it reaches `cast_missile` (section 3).
- Gating: the rig turns the key off in town (40) and in Duriel's Lair (73) (docs/act2.md "syncTele"); `internal/boss/act2.go:96` says "Teleport into the lair is forbidden by levels.txt (V, Teleport column 2)"; `internal/data/levels.go:13` comments the column as "0 no, 1 yes, 2 forbidden...".

Bot findings (not changed):

| id | finding | tag | impact |
|---|---|---|---|
| B1 | the 18-subtile hop is the game.exe backend's screen clamp, not the skill: on an 800 x 600 view the screen edge is about 17.7 subtiles away left / right and 26.5 up / down (a subtile is 32 x 16 pixels in the iso view); the skill's own limit is the near-rooms rule (40+ subtiles along each axis on 8 x 8-tile rooms) | I (screen geometry, view size assumed) + V run (room rule) | on game.exe the bot cannot hop further by mouse; on d2d (`cmd skill`, no screen) it could hop about twice as far per cast within game.exe's rule, halving hops and mana. A per-backend range needs the room rule, not a bigger number (D1) |
| B2 | on d2d a refused hop (blocked cell, Levels 0) costs the full mana at the command; in game.exe it is free (D4) | V read + S | every "did not move" hop on d2d drains 24 mana (level 1) that the real game keeps: more `waitMana` time on d2d than on game.exe; the bot's three-bad-hop rule caps it |
| B3 | the bot aims only at known walkable cells; game.exe would also land a click on a wall or a monster at the nearest free spot (D2, D3) | V run | conservative and correct on both backends; a blind cast into the void lands only if its point lies in a near room (the void between islands has no room data, docs/act2.md Y3: I), which explains Y4's "blind casts stop at the first void edge" |
| B4 | `hopCell` ignores monsters: in game.exe a hop aimed at a monster's subtile lands next to it, on d2d on top of it (D3) | V run + S | small; it changes melee exposure after a hop on d2d only |
| B5 | `teleportSettle` 700 ms vs the action frame (about frame 8-9 of 13 at 0 FCR, docs/research/cast-trace-gameexe.md) and the SC end (13 frames, 520 ms) | S + I | enough; the next hop cannot start before 520 ms anyway; with FCR the settle could shrink (casts per second are the hop rate) |
| B6 | Duriel's Lair (73) is read as "Teleport forbidden" (boss/act2.go:96, data/levels.go:13, docs/act2.md D1 / line 41); game.exe's value 2 allows a hop along a line clear of mask 0x804 (doors 0x800, missile barrier 0x4) | V read + D | turning it off is safe but costs walking in the lair; fix the comments and docs when someone next touches them |
| B7 | `defaultTeleportMana` 24 is exact at level 1 and an upper bound later; from level 25 the formula is <= 0 and a cast is free when mana is below 1 point (SKMANA D2) | S (SKMANA) | harmless |

## 5. Deviations (d2d vs game.exe)

| id | what | d2d | game.exe | tag | magnitude / bot impact | differential |
|---|---|---|---|---|---|---|
| D1 | range | any point on the level (and neighbour levels' grids) | only a point in the caster's room or its near list | V run (oracle) + S | 25,120 of 100,000 random cases (game.exe refuses, d2d lands); none at the bot's 18 subtiles; any longer hop on d2d is fake | oracle exists (`sk_tele_spot.py`); real-build side not built |
| D2 | blocked target | refused | nearest free spot (ring first, then \|dx\| + \|dy\|, out to 50) | V run + S | 1,323 of 100,000 (game.exe lands nearby, d2d refuses); the bot aims at walkable cells (B3) | as D1 |
| D3 | monster at the target | ignored: lands on it (point) | 0x1000 in the mask: lands next to it | V run + S | 4,805 of 100,000 (4,643 both land elsewhere, 138 d2d refuses for a wall in the footprint, 24 a monster in the plus footprint) | as D1 |
| D4 | mana on a refused hop | paid at the command | none (do 27 returns 0, no missile, FUN_0056f7f0 skips the payment) | V read (do 27 + SKMANA 0x56fa3d) + S | 24 points a refused hop at level 1 (B2); also SKMANA's D1 (pay at the command vs the action frame) | stateful: drill T4 |
| D5 | a unit target | walks up to a live monster and casts at it; lands on its exact spot | teleports to the unit's position (FUN_0056d2c0), snapped off it | V read + S | the bot never sends a unit (unit 0) | drill T3 |
| D6 | Levels Teleport 2 line | samples every 0.2 cell with `unit_blocked` (0x0c09, plus footprint) | FUN_006229f0: caster to the clicked point, mask 0x804, ends trimmed by the sizes | V read (outline) + S | Duriel's Lair only; walls (0x1) refuse on d2d but not in game.exe, missile barriers (0x4) the reverse | needs FUN_0064e260 read; oracle hook point exists |
| D7 | arrival bookkeeping | no ring entry (next tick's footstep), no warp spot, no pet or merc move | landing in the 20-step ring, warp spot for 50 frames, every `warp` pet (the hireling) moved to the landing at once | V read (outline) + S | the merc stays with a hopping Sorceress in game.exe but falls behind on d2d until its leash teleport; changes who tanks after a hop. Surprising against common play experience: the drill must confirm before anyone builds on it (I) | drill T5 |

Equal (S + V read): town refusal (start, no mana in both: d2d refuses at the command before `cast_missile`), Levels Teleport 0, the action-frame timing of the move, the mana formula (SKMANA).

## 6. Differential (V run for game.exe; d2d by line copy)

`tools/emu/sk_tele_spot.py` (hooks in its docstring): `game` runs FUN_005ca360 natively down to the position set on synthetic rooms; `d2d` is a line copy of `spot()`'s Teleport test on the same grid (refuse at Teleport 0, `unit_blocked` plus footprint with mask 0x0c09 on the target, off the level blocked, else land at the target). A line copy is allowed as "before" only (upstream-pr-standard.md); it is not a claim about the d2d build.

Cases (`gen 100000 1`): 5 x 5 rooms of 40 subtiles, 0-40 random rectangles of 0x1 / 0x9 / 0x5 / 0x400 / 0x800 / 0x4 / 0x8, 0-12 unit subtiles (0x1000), the caster in the centre room, the target 40 % within 40 subtiles, 30 % anywhere, 15 % on a unit, 15 % past the map edge; Levels Teleport 0 in 5 %.

| Run (the worker host, upstream main 72132b99 tools) | same | D1 room not near: game.exe refuses, d2d lands | D3 monster at or next to the target | D2 blocked target: game.exe lands nearby, d2d refuses |
|---|---|---|---|---|
| `gen 100000 1` | 68,752 | 25,120 | 4,805 (4,643 both land, other spot; 138 d2d refuses; 24 a monster in the plus footprint, not on the target) | 1,323 |
| `gen 20000 7` | 13,822 | 4,902 | 968 (938 + 25 + 5; the 5 not checked one by one) | 308 |

md5 (seed 1): cases 6d300a90..., game 93bd61e8..., d2d 95841201.... Every one of the 31,248 differences (seed 1) is D1, D2 or D3; no other kind (the 24 "both land elsewhere" off the target were checked one by one: each has a unit subtile in the target's plus footprint).

The 4,000-case pilot gave the same split (2,738 same; 1,019 room not near; 179 on a monster; 63 blocked target; 1 other). Every difference falls in D1, D2 or D3.

## 7. PR candidate

None. Why: the d2d side above is a line copy; the rules want the real build. What a candidate needs (local only, the user's go before anything is pushed): a C++ test mode (`tests/game/test_tele_spot.cpp`, the SKMISS pattern) that loads the patched tables, builds a real `Level` with the case's grid and rooms, calls the Teleport branch of `Fight::spot`, and prints the landing; then a fix in `spot()`: the target must lie in a `near_rooms` rect of the caster's position (D1), snap with a game.exe-order search and mask 0x1c09 including the Crowd (D2, D3), no refusal on a blocked point; D4 belongs with SKMANA's pay-at-the-action-frame change (one fix for every skill). D6 needs FUN_0064e260 read first. D7 is a separate, merc-facing change after drill T5.

## 8. Drill recipe (live; a throwaway Sorceress, one hidden slot, offline, lab binary with the FORKDBG verbs)

Verbs (docs/forkdbg.md on branch agent/forkdbg, docs/cheats.md): `debug goto`, `debug points 54 <n>`, `debug stat 8`, `debug mana full`, `debug cast 54 <x> <y>` / `unit <id>`, `debug spawn`, `debug pin on|off`, `blocked X Y` (read), `dump` / `dump stats` (position, mana raw[8] in 256ths), `step <n>`. Every cheat through `d2dctl.SkillDebug` / `d2drill -probe` (ledgered, with a reason). Positions in cells (subtile = cell x 5). The game.exe side of each drill is the expected value from this trace (a game.exe run reads the same by memory on a hidden instance; no cheats there beyond the logged tool).

- T1 snap (D2): `debug goto 2`, `debug points 54 1`, `debug mana full`; find a wall or tree subtile 3-5 cells away with `blocked`; `debug cast 54 <it>`; `step 14`; `dump`. game.exe: lands at the nearest free spot (ring order of section 2). d2d today: does not move, mana down 24.
- T2 room limit (D1): open ground, `debug cast 54` at a point 25 cells (125 subtiles) away along x: game.exe refuses (outside the near rooms), mana kept; d2d lands. Then 9 cells (45 subtiles): game.exe depends on the caster's place in its 8 x 8-tile room (lands when the target's room touches hers).
- T3 monster (D3, D5): `debug spawn zombie1 1 near`; `debug cast 54 <its cell>` and `debug cast 54 unit <id>`: game.exe lands next to it both times; d2d lands on it (point), walks then casts (unit).
- T4 free refusal (D4): `debug stat 8 6144` (24 points), cast at T2's far point: game.exe keeps 6144; d2d 0. With `step 1` + `dump stats` a tick: a hop that lands pays at the action frame in game.exe (SKMANA M1).
- T5 merc (D7): needs a hired merc on the throwaway; hop 15 subtiles on open ground, `dump` the merc's position on the landing tick: game.exe at the landing spot at once; d2d running behind.
- T6 lair line (D6): `debug goto 73`, hop across a doorway (0x800) and along a clear corridor: game.exe refuses the first, lands the second.
- T7 town (equal): `debug goto 40`, cast: refused, mana kept, in both.

## 9. Matrix cell updates (docs/research/sorc-skill-matrix.md section 3.7 and section 5)

| Row | Cell | New value |
|---|---|---|
| 3.7 Teleport | (a) mana | 24 - (L - 1) [D]; paid at the action frame only for a hop that moved, none on any refusal [V read: do 27 returns 0, FUN_0056f7f0]; free under 1 point from level 25 [SKMANA V run] |
| 3.7 Teleport | (a) rules in game.exe | refused in any town room at the start, no mana [V read FUN_0056f640, D InTown 0]; Levels Teleport 0 none, 2 a clear line caster -> clicked point mask 0x804 (only Duriel's Lair, D) [V read FUN_005ca360]; the target must lie in the caster's room or its near list (40+ subtiles each axis on 8 x 8-tile rooms) [V run]; landing = nearest free spot, plus footprint, mask 0x1c09 incl. units, ring then \|dx\| + \|dy\|, 50 rings [V run]; no level scaling, no distance clamp [V read]; unit target: its position [V read]; arrival: 20-step ring, warp spot 50 frames, `warp` pets moved with her [V read, drill T5 to confirm] |
| 3.7 Teleport | (b) d2d status | D1 no room limit, D2 no snap, D3 lands on monsters, D4 mana on a refused hop, D5 unit target walks, D6 line mask, D7 no pet move (sk-teleport-trace.md); oracle exists, real-build harness not built [V run vs line copy] |
| 3.7 Teleport | (c) bot use | 18 subtiles = the game.exe screen clamp (B1, I); refused hops cost mana on d2d only (B2); lair read as forbidden (B6) |
| section 5 row 3 (SKTELE) | status | done 2026-11-01 (sk-teleport-trace.md); no PR candidate; drills T1-T7 |

## 10. Not traced (U)

- FUN_0064e260 (the line walk for Levels Teleport 2) and the `warp` Skills bit's reader.
- What the pets do after FUN_00650be0 puts them at the landing (FUN_00648c30 0x100; their own free-spot search), and whether play shows the merc keeping up (drill T5).
- The client: whether it refuses a click outside the near rooms itself, and the exact screen reach (B1).
- Room sizes of the Act 2 maze levels (sewers, tombs, Arcane Sanctuary): the 40-subtile figure is for 8 x 8-tile outdoor rooms.
- The real d2d build on the same cases (section 7).

## Shared-file lines for the merger (this card does not own these files)

- docs/decomp-queue.md row 16, last column: `done 2026-11-01 (SKTELE, Opus): do 27 FUN_005ca360 + warp FUN_00554ea0 traced and run natively (tools/emu/sk_tele_spot.py, 100,000 cases): range = the caster's room or its near list (not a distance), landing = nearest free spot (mask 0x1c09 incl. units, plus footprint, ring then |dx|+|dy|, 50 rings), refused hops free, Levels Teleport 2 = clear line (lair allowed), warp pets moved with the owner; d2d D1 no room limit, D2 no snap, D3 lands on monsters, D4 mana on refusal, D5-D7; no PR candidate (line copy only); bot B1 (18 = screen clamp), B2, B6 (lair read as forbidden); drills T1-T7`; Evidence column add `docs/research/sk-teleport-trace.md; tools/emu/sk_tele_spot.py`.
- docs/BACKLOG.md (decomp / fidelity section): `- SKTELE (docs/research/sk-teleport-trace.md): d2d Teleport deviations D1-D3 (room limit, snap, units) need a real-build test mode before an upstream candidate; D4 joins SKMANA's pay-timing fix; drill T5 (merc moved with the owner) before any merc work relies on it; bot B6 (Duriel's Lair read as no-Teleport in boss/act2.go and data/levels.go).`
- docs/INDEX.md (research traces): `- [sk-teleport-trace.md](sk-teleport-trace.md): Teleport in game.exe 1.14d (refusals, near-room range, landing snap, arrival) vs d2d and the bot's nav hops (decomp row 16).`
- CHANGELOG.md (unreleased, docs): `- Teleport decomposition (SKTELE): game.exe's destination rule traced and run under the emulator; d2d deviations and bot findings listed (docs/research/sk-teleport-trace.md).`
