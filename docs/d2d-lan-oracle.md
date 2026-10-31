# d2d LAN oracle demo (original Game.exe host, d2d client)

> status: demo done 2026-10-30 by agent LANORACLE on hidden instance i1. Tags: V = VERIFIED LIVE (run in this demo), S = read in source or docs, U = UNVERIFIED. Scope: LAN TCP to a local original host on 127.0.0.1:4000 only (user-approved 2026-10-30); no Battle.net. Raw logs stay on the worker host in `~/d2d-lab/lan/`; none are in this repo.

## Result

| Step | Result |
|---|---|
| Original Game.exe (i1, Wine, Xvfb) hosts a TCP/IP game via `oog "host NAME"`, TCP 4000 bound | V |
| Headless `net-join` (upstream tool, lab fork build) joins it, 10 s, clean leave | V: 187 S>C and 10 C>S packets, 0 unknown packets |
| Same join through `d2proxy` (TCP MITM) | V: 172 S>C, 10 C>S in both the proxy log and the client log, 0 unknown; same map seed on re-join of one game |
| `d2d.lab6 --headless --join 127.0.0.1` joins (char select, Return) | V: "in 127.0.0.1's game: act 1, map seed 0x5cd102ca"; d2d lays out act 1 for that seed; "5 of our 5 items match the host's" |
| whisper (`ijl11.dll` stand-in) loads in original Game.exe under Wine and answers | V: `ping`, `state`, `peek 463dd0 2` = `7a6a70a1 ccccc300` (1.14d check passes), `units`, `quests`, `town` |
| Rogue Encampment, host seed 1891490528: whisper `town` vs d2d `drlg-dump ... 1 tiles` | V: 35/35 rooms match (upstream's claim reproduced) |
| Lut Gholein (level 40), host seed 831026623: same comparison | V: loaded rooms all DIFFER at the first tile line; looks like a systematic tile-origin or ordering shift (first game.exe tile is at x+1, d2d at x), not 21 independent bugs. U which side. Not investigated |

## Exact commands (all on the worker host, under a d2d slot)

1. Build (no source edits, build dir of the fork): `ninja -C ~/d2d-lab/fork/src/build net-join d2proxy drlg-dump`; copies in `~/d2d-lab/lan/`.
2. Instance and host: `tools/game.sh start i1; tools/game.sh helper i1; D2_INSTANCE=i1 tools/live.sh lanoracle oog "host Navi"` (the agent name is not in instances.conf, hence `D2_INSTANCE`). Leave: `oog leave`.
3. Direct join: `D2_GAME_EXE=<Game.exe of the i1 prefix> ~/d2d-lab/d2dslot.sh run -- ./net-join Palli.d2s 127.0.0.1 --seconds 10 --log join-direct.log`. The joiner `.d2s` is a COPY of an i1 character; the host rewrites the joiner's save, so use a copy.
4. Via proxy: the game port is fixed at 4000 and the host binds 0.0.0.0:4000, so the proxy cannot also listen on 4000. Run `d2proxy --host 127.0.0.1 --port 4000 --listen 4010 --log-dir proxy1 --level debug` and point the client at it with a 12-line `LD_PRELOAD` connect() shim that rewrites port 4000 to `$SHIM_PORT` (`portshim.c`, kept on the worker host; it is a lab helper, not repo code). A second machine or netns would not need it.
5. d2d client: `XDG_CONFIG_HOME=<dir with d2d/save/Palli.d2s> d2d.lab6 --data ~/d2d-lab/data --headless --no-video --devctl SOCK --no-save --join 127.0.0.1 --start-screen charselect --start-name Palli`, then devctl `key Return` selects the character and joins. `debug level` afterwards: `1 ... 57 41 1024 1256`, the same level origin `drlg-dump` gives for the seed.
6. whisper: no mingw on the worker host (and no root), so it was built with a user-local `pip install ziglang` in a venv: `python -m ziglang c++ -target x86-windows-gnu -std=c++20 -O2 -shared -s -o ijl11.dll whisper.cpp ijl11.def -lws2_32 -lgdi32` (V: builds, loads, works). Install ONLY in the i1 prefix: `mv ijl11.dll ijl11_real.dll`, copy the new dll in as a new file, write `whisper.txt` (token line, port 4100), restart the game. The i1 prefix is a hardlink clone of the shared install (link count 6): rename plus a new file leaves every other instance's inode untouched (V: i2's `ijl11.dll` still has link count 6). Undo (done): delete `whisper.txt`, `mv ijl11_real.dll ijl11.dll` (back on the shared inode), restart i1.
7. Compare: `python3 whisper.py 127.0.0.1 TOKEN town` (host level, seed, loaded rooms), `drlg-dump ~/d2d-lab/data SEED LEVEL tiles > d2d.txt`, `whisper.py ... town d2d.txt` prints one line per room.

## Facts worth knowing

- The `[03] load act` packet carries the map seed (bytes 3..6 little endian) and area of the JOINER's save, not of the host's player: Palli (act 1) got area 1 while the host Navi stood in Lut Gholein. whisper's `town` reads the host player's level. To compare a given town, host a character that stands in it (V: hosting Palli gave level 1).
- Seeds are chosen by the host per game (V: 0x5cd102ca, 831026623, 1891490528 in three games); one game keeps its seed across re-joins. Observable, not controllable. (U: whether the save header's map id steers it.)
- `town` only prints rooms near the player as loaded; in Rogue Encampment all 35 were loaded.
- Joined players make it a 2-player game: monster HP and XP use the 2-player rows (+50 % each, S net-host.md), so a joined session is not a 1-player fidelity sample until d2d applies the same scaling.
- whisper `units` on the host (level 1): 266 units (1 player, 14 NPCs, 245 objects, 5 items). `drlg-dump ... units` lists 16 type-1 and 26 type-2 entries per room, with its own class numbering; counts and ids are not comparable without a mapping table. Not done.
- Port 4000 is exclusive per machine: nothing else may host while this runs (card exclusive `port:4000`).

## Limits

One original host per machine (fixed port); the host is a real game on a core; joiner saves get rewritten; whisper addresses are 1.14d only; token auth only (bound 0.0.0.0:4100 while it ran, so keep the window short); no raw logs or binaries in the repo.

## Next step proposal: spawn/XP diff harness

1. Mapping table for unit classes between whisper `units` and `drlg-dump units` (class id, position in tiles) and a Go or Python checker that diffs NPCs per seed in Rogue Encampment (all seeds equal layout, so it validates the mapping): 3-4 h.
2. Investigate the Lut Gholein tile mismatch (is it whisper's room origin, or d2d's act 2 town?): 2-3 h.
3. Outdoor spawn diff: join with d2d to the host's seed, run d2d `--seed` to that value, compare monster lists for Blood Moor from whisper `units` after walking there, with the 2-player scaling applied: 10-15 h, feasibility risk is per-room monster seeds (U).
4. XP/damage per packet (0x1c/0x1d, damage packets via d2proxy) for scripted kills: 6-10 h plus a delta spec for the fidelity gate.
Total 15-25 h (matches the WHISPERLOOK estimate); this demo took about 1.5 h of agent time.
