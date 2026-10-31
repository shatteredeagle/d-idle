# Lut Gholein: why whisper `town` and d2d `drlg-dump` disagree

> status: investigation done 2026-10-30 by agent LUTGHOLEIN on hidden instance i1 (Navi). Tags: V = VERIFIED LIVE or by a run named here, S = read in upstream source or docs (psi29a/d2decomped, lab fork `~/d2d-lab/fork/src` at 7abb81d1), U = unverified. Follow-up of docs/d2d-lan-oracle.md ("Lut Gholein ... all DIFFER"). Raw captures and the scratch tool stay on the worker host in `~/d2d-lab/lan/lut/`; none are in this repo.

## Answer in three lines

1. The room *lists* are the same on both sides (49 rooms, same positions, same DS1 file, same walls and shadows). What differs is (a) which room owns the tiles on a shared room edge and (b) the random `ground.dt1` floor variants. Both are known order/seed effects, not a tile-origin shift. V
2. (a) is a comparison artifact: `drlg-dump` brings all 49 rooms up in list order; the live client brought up 20 rooms around the player in the order "the player's room, then its 3x3 near list row by row, then the ring". With that order fed to d2d's own `level_room_tiles` every per-room tile count matches. V
3. (b) is a real d2d generator bug for Lut Gholein: the live room2 seed word of row-major room k is d2d's seed of room k+2, for 47/47 comparable rooms, and game.exe's own DRLG run in upstream's emulator gives the live seeds for 49/49 rooms (0/49 agree with d2d). With the order from (2) plus that two-room seed shift, d2d reproduces all 20 live rooms cell for cell on both seeds tried (20/20 and 20/20; without the shift 16/20 differ, without the order 20/20 differ). The drlg code involved is identical to upstream/main (32768ddb) apart from comments, so it is an upstream bug, not a lab-fork regression. What game.exe rolls twice more is not traced (U).

## What was compared

- Live side: whisper `town` (upstream `apps/whisper/whisper.py`, the `town` verb mirrors `tools/emu/drlg.py tiles_dump`: room2 list from the player's level, rooms sorted by (y, x), per room the wall/floor/shadow entries of room2 +0x54 with tile x,y relative to the room, DT1 name and tile index). S whisper.py, drlg.py.
- d2d side: `drlg-dump DATA SEED 40 tiles` (upstream `tools/drlg-dump`), which calls `generate_preset` and then `level_room_tiles(rooms, ..., order = units_order(...))`; for `tiles` the order is empty, i.e. list order, every room up. S drlg-dump.cpp lines 61-76, 122.
- Captures: LANORACLE's `town.txt` (map seed 831026623, LutW) and a fresh host on i1 today (map seed 161464580, LutN; `state`: Navi at subtile 5153,5203, room 24,40, 20 rooms with tiles out of 49). V

## Findings, in the order they were made

1. Per room, masks and DT1 lists match, but tile multisets differ and no translation (dx, dy in -9..9) nor axis swap makes a room equal. Level-wide at absolute tile coordinates and zero shift: 1249 cells identical, 555 differ only in the tile index, 0 missing. 548 of the 555 are `ground.dt1` floor variants (orientation 0, same cell), the other 7 are walls counted twice by my level-wide sum (a shared-edge wall listed by two d2d rooms, one of them not up live). So positions agree; the room origin is not shifted. V (scratch `cmp.py`, offline on the recorded files)
2. The live rooms' floor windows are not 0..7: interior rooms show 1..8, edge rooms 0..8 or 0..7; in Act 1 both sides show 0..7 with 0..8 only on the level's far edge. Reason (S drlg.md "Room tiles", "Preset-room details"): a room keeps a (w+1)x(h+1) slice, and an edge tile already laid by a near room that is up is kept by that room, so the *earlier* room owns the shared edge. The windows therefore encode, for every adjacent pair, which room came up first: 31 constraints, consistent with exactly one simple order: the player's room 24,40 first, then the other 8 rooms of its 3x3 near list in (y, x) order, then the ring of 11 rooms in (y, x) order. Upstream calls this "the server's order (the arrived room first, FUN_0056cf40, then the near list)". V (constraints computed from the capture; S for the FUN)
3. A scratch build of `drlg-dump` with `ORDER=file:PATH` (a room list in bring-up order, passed as `order` with `up_count` to `level_room_tiles`; patch kept at `~/d2d-lab/lan/lut/drlg-dump-order-shift.patch`, source reverted, fork tree clean) reproduces every room's wall/floor/shadow counts with that order on both seeds; the 4 rooms of column x=40 (canal tiles only, no random pick) match cell for cell, the other 16 differ in `ground.dt1` variants from their first floor pick on, with no RNG offset (shift test on the pick sequence: no shift matches). V
4. Live room records (tools/emu/lut_rooms.py, over whisper `peek`): room2 +0x4 of row-major room k equals d2d's generation seed of room k+2 for all 47 comparable rooms (shift 0, 1, 3: none). The last two live rooms (40,48; 48,48) hold seeds d2d gives no room, and d2d's first two seeds appear in no live room. The not-up rooms show the same shift, so it is set at level generation, not by bringing rooms up. V (seed 161464580; d2d seeds from `drlg-dump ... 40 seeds`)
5. With `SEEDSHIFT=2` (room k takes room k+2's seed words, same scratch build) and the order of (2): 20/20 rooms identical for seed 161464580 and 20/20 for LANORACLE's 831026623 capture; `SEEDSHIFT=2` without the order: 0/20. V
6. Rogue Encampment (LANORACLE, 35/35 with variants) shows no such shift, so it is not a general client-side effect on every preset level. V (docs/d2d-lan-oracle.md)

## Emulator result (server-side game.exe)

Upstream's fidelity table says Lut Gholein rooms, tiles and units were diffed 20/20 seeds against game.exe's own DRLG in the emulator (`diff_drlg.py 1-20 40 tiles`, S drlg.md fidelity table). That path runs the server-side act allocation (`alloc_act`). I ran upstream's `tools/emu/drlg.py 161464580 40` and `... 40 tiles` (the casttrace venv `/tmp/casttrace-venv`, game.exe copy `/tmp/casttrace/game.exe`, a symlink dir with `bin/game.exe` and lowercase `patch_d2.mpq` as `D2_MPQ_DIR`; 6 s per seed):

- emulator room seeds == live client room seeds: 49/49 (shift 0); emulator == d2d: 0/49 same room, 47/47 with d2d shifted by two rooms. V
- emulator tiles (all rooms up, list order) vs d2d `drlg-dump ... tiles` (same order): 12 rooms identical, 37 differ (the 12 are rooms without random picks). V
- So the server-side game.exe generator has the shift too; the live client merely shows the same seeds. Upstream's "20/20 seeds" claim for level 40 does not hold on this tree and data; the 20-seed differential below quantifies it.
- Twenty-seed differential (map seeds 1..20, `drlg.py S 40` vs `drlg-dump DATA S 40`, room seed lists compared, `~/d2d-lab/lan/lut/diff20/`): seeds where d2d's 49 room seeds equal game.exe's: 0/20; seeds where d2d equals game.exe shifted by two rooms (47/47 comparable rooms): 20/20, for both DS1 files. V
- Fork vs upstream/main for `components/drlg/{maze,units,room_tiles,outdoor}.hpp`: only comment wording differs in maze.hpp (crash notes), nothing in the seed path. V (`git diff upstream/main`)

## Interpretation and PR status

- Comparison artifact, part 1: `drlg-dump ... tiles` is not the right model for a live client's `town` output in a big town; `level_room_tiles` with the client's bring-up order (and `up_count`) is, and d2d already has it (`relevel`, `client_order`, S drlg.md). LANORACLE's "first tile at x+1" is exactly the shared-edge ownership of (2). Fix for the oracle: give the comparison the live order (my ORDER=file patch, or `d2d --join` + `debug tiles` as upstream does).
- Part 2, the two-room seed shift, is a d2d generator bug for Lut Gholein (level 40; Act 1 town is right): d2d's `generate_preset` steps the level RNG two times fewer than game.exe before the per-room `seed.next()` loop (`preset_rooms`, FUN_00666680). Its mechanism is NOT traced (U): candidates are the Pops/Scan unit roll (`stays`, FUN_00667970: d2d may roll for fewer units than game.exe, e.g. units outside every room, or a unit type missing from its list; LvlPrest 301 has Scan 1 and Pops 3) and the file roll (FUN_00666ed0, one step in both as far as read). Every ground variant in the town is wrong as a result (cosmetic in a town; the same path serves other presets with Scan/Pops, U which).
- PR candidate: PROOF incomplete. What exists: the emulator differential (game.exe's own code vs d2d, 20 seeds, room seeds; below) showing the bug, a two-seed live reproduction cell for cell with a one-line test knob (SEEDSHIFT=2 is a diagnostic, not the fix: the last two rooms get no seed), and the room-order model for the live comparison. What a PR needs (docs/process/upstream-pr-standard.md): the game.exe function that does the two extra steps (V by emulator hooks on the level RNG, FUN_ listed, pr_guard clean), the fix in `generate_preset`/`stays`, the same emulator differential at 0 mismatches after (`diff_drlg.py 1-20 40 tiles`, upstream's own gate), the independent verifier, and a check of every other Scan/Pops preset level. No PR branch was prepared.

## Exact commands (the worker host, lab space)

```sh
# live (instance i1 started by this agent; whisper installed only in the i1 prefix, undone after)
tools/game.sh start i1; tools/game.sh helper i1; tools/mute-d2.sh
D2_INSTANCE=i1 tools/live.sh lutgholein oog "host Navi"
cd ~/d2d-lab/lan/lut
python3 -I lut_rooms.py 127.0.0.1 ../whisper/token state   > state1.txt
python3 -I lut_rooms.py 127.0.0.1 ../whisper/token rooms   > rooms1.txt
python3 -I ../whisper/whisper.py 127.0.0.1 TOKEN town       > town-live1.txt
# d2d, scratch build (patch applied to src/tools/drlg-dump/drlg-dump.cpp, ninja drlg-dump, binary copied, source reverted)
ORDER=file:order-arrived.txt ./drlg-dump-order ~/d2d-lab/data 161464580 40 tiles  > d2d-s2-arrived.txt
ORDER=file:order-arrived.txt ./drlg-dump-order ~/d2d-lab/data 161464580 40 seeds  > d2d-s2-arrived-seeds.txt
SEEDSHIFT=2 ORDER=file:order-arrived.txt ./drlg-dump-order ~/d2d-lab/data 161464580 40 tiles > d2d-s2-arrived-shift2.txt
# order-arrived.txt: 24,40 then 16,32 24,32 32,32 16,40 32,40 16,48 24,48 32,48 then 8,24 16,24 24,24 32,24 40,24 8,32 40,32 8,40 40,40 8,48 40,48
# compare: scratch cmp.py (per room: exact list, multiset, best translation); whisper.py's `town D2D.txt` gives the same verdicts room by room
```

## Dead ends

- Translations, axis swaps and flips of whole rooms: none fits (finding 1).
- Two-batch client order from drlg.md (near list then ring, no arrived-room-first): tile counts differ in rooms 16,40 and 24,40, so the edge ownership rules it out for a save-load arrival; upstream measured it after a waypoint arrival.
- An RNG offset within a room (same seed, extra rolls): no offset in -3..3 matches the pick sequence.
- Upstream `diff_drlg.py` on the worker host: `uv` and `unicorn` are not installed system-wide; the casttrace venv works for `drlg.py` with `D2_MPQ_DIR` pointing at a symlink dir that has `bin/game.exe` and lowercase `patch_d2.mpq`.

## Instance and cleanup

Navi hosted one TCP/IP game on i1, stood in Lut Gholein, no movement, no cheats; left with `oog leave` (save and exit, V), i1 stopped (`tools/game.sh stop i1`, V: Game.exe pid 2761408 and Xvfb :101 gone), whisper uninstalled from the i1 prefix (V: `ijl11.dll` back on the shared inode, link count 6; no `whisper.txt`, no `ijl11_real.dll`), ports 4000/4100 closed, no jobs or processes of this agent left. The scratch `drlg-dump-order` binary, its patch and the captures stay in `~/d2d-lab/lan/lut/` (lab space; the fork tree is clean). Bring it back: `tools/game.sh start i1; tools/game.sh helper i1; D2_INSTANCE=i1 tools/live.sh <agent> oog "start Navi"`.
