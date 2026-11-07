# Trace: life, mana and stamina at character load, d2d against game.exe (TASK-101)

> status: rewritten 2026-11-07 by the fidoracle agent (Sonnet) on top of the fidfix1 pilot. The game.exe loader is found and run under the emulator; the pilot's rule ("fraction dropped") is REFUTED for the server side and replaced by "kept exactly". Needs an independent Opus verification before anything is opened (docs/research/pr-bundle-queue.md). Rows EN04, ST01. Earlier evidence: [fidelity-probe-run1.md](fidelity-probe-run1.md) (D2 file reader, scenario 15).
>
> Tags: **V read** read in the disassembly (`tools/emu/disasm.py` of upstream, addresses given); **V run** game.exe's own code run under the emulator by `tools/emu/fl_load.py` and compared with d2d's `World::enter` byte for byte (a PR claim rests only on this); **V live** observed on the running game.exe or d2d (the earlier probe); **D** data table; **I** inferred; **U** unverified.

## 1. Answers

**Q1. Where does game.exe set a loaded character's life and mana?** (V read, game.exe 1.14d addresses)

| Step | Function (address) | What it does |
|---|---|---|
| version dispatch | FUN_00534330 (0x534350 magic 0xaa55aa55; 0x534364 `cmp ebx, 0x5b`) | version above 0x5b: FUN_0056b180 (0x534387); 0x5b and below: FUN_00534020 (old format, not read, U) |
| reached from | join: FUN_00539760 (0x539804) -> FUN_005345a0 (0x5345fb) -> FUN_00534520 (0x53455e) -> FUN_00534330; a new character with no save data: FUN_005345a0 -> FUN_00532590 (0x534607) | |
| the loader | **FUN_0056b180** | header FUN_0056a090, 0056a370, 0056a3e0, 0056a470; **stat block FUN_0056a620 (0x56b270)**; then it reads **stat 6 and stat 8** with FUN_00625480 (0x56b283, 0x56b28f) into two locals; skills FUN_0056a710, items FUN_0056a7e0, corpse FUN_0056a830, mercenary FUN_0056aa50 / FUN_0056ac10; finally **FUN_0056af80 (0x56b3c3)** with those two locals |
| stat block decode | FUN_0056a620 -> FUN_0056a4f0 (version above 0x5e; 1.14d writes 0x60) | header word 0x6667 ("gf"), then 9-bit stat id, the stat's param bits (ItemStatCost +0xb), the stat's save bits (+0xa), until id 0x1ff (0x56a53e); each stored with FUN_00627260 (0x56a5d5); the value is the file's raw 256ths |
| restore | **FUN_0056af80** | gold and bank clamped to their caps (FUN_00622e70, FUN_00623460, 0x56afb4..0x56b00f); **stamina (stat 0xa) set to the maximum (FUN_00625db0 = stat 0xb), 0x56b01d**; equipment lists re-linked (FUN_0056af20, FUN_005701b0); **stat 6 set to the saved life, stat 8 to the saved mana, 0x56b0ff and 0x56b10d (FUN_00627260), no comparison with the maximum**; stats 0x44, 0x43, 0x45 set to 100 (0x56b115..0x56b128, names U), stat 0x1e (0x56b159) |

The queue's start point FUN_0052d440 -> FUN_0055df00 -> FUN_0055dbc0 is NOT the stat load: FUN_0055dbc0 re-validates the worn items (it loops the body slots through FUN_0063bde0 / FUN_00625820 / FUN_0055db00 and puts back the ones that still fit). Its code range 0x55dbc0..0x55df0f holds no write of stat 6, 8 or 0xa (V read: a scan of every `call 0x627260` whose stat argument is 6, 8 or 0xa over the whole exe; none lies in that range). d2d's `settle_requirements` is the matching step and stays where it is.

**Q2. What stat vector does the loader leave?** (V run, `tools/emu/fl_load.py`)

The oracle calls the real FUN_0056b180 on a stat block it encodes with the loaded ItemStatCost bit widths, the other file parts hooked (section 3), and reads the six stats back with the real getter. 414 cases: the two live saves, zero life, zero everything, one 256th, fractions .5 and .996, life / mana / stamina above the maximum, stamina 0 and half, a maximum of 0, and 400 random.

| Case (file, 256ths) | life | mana | stamina | maxima (life / mana / stamina) |
|---|---|---|---|---|
| Sorceress-like: life 22921 of 23296 (89.54 of 91), mana 6244 of 11520 (24.39 of 45), stamina full | **22921** | **6244** | 26112 | 23296 / 11520 / 26112 |
| Paladin-like: life 19907 of 22528 (77.76 of 88), mana 4992 of 4992 (19.5), stamina full | **19907** | **4992** | 20224 | 22528 / 4992 / 20224 |
| life 0, mana full | **0** | 11520 | 26112 | |
| life 1 and mana 1 (one 256th) | 1 | 1 | 26112 | |
| life 95 of 91 | **24320** (kept above the maximum) | | | |
| mana 50 of 45 | | **12800** (kept) | | |
| stamina 0, half, above the maximum | | | **always the maximum** | |
| maxima 0, current 5000 each | 5000 | 5000 | **0** (= the maximum) | 0 / 0 / 0 |

So: life and mana are kept **exactly**, the fraction included, with **no clamp** to the maximum; stamina is set to the maximum, always. In all 414 cases the loader's stat 6 and 8 equal the file's and stat 10 equals the file's stat 11 (V run, `fl_load.py game`; `--stages` shows the decode alone gives the same life and mana, so the restore step changes only stamina in this setup).

**Q3. Why did the live reads show the fraction dropped (89.54 -> 89)?** (V read + V run)

The probe read the CLIENT player unit. Single player runs a server and a client in one process, each with its own unit. The server's per-player update FUN_00548760 sends the unit's life, mana and stamina as whole points: **`sar 8` of stat 6 at 0x5487b7** (stat 8 at 0x548855, stat 0xa at 0x548868), packed 15 bits each (FUN_0053c320 for packet 0x95, FUN_0053c230 for 0x18); the client's handler FUN_0045d9b0 sets its own stats to **`value << 8`** (0x45d9d8 for stat 6, 0x45d9ea stat 8, 0x45d9fc stat 0xa). The oracle's `client` command runs that handler (the real FUN_0045d9b0 and FUN_0045d900 on a 0x18 packet it builds) on the server's loaded vector: Sorceress-like life 22784 (= 89 * 256), mana 6144 (24); Paladin-like life 19712 (77), mana 4864 (19). These are the six live reads (89, 77, 19; the live mana 25 is 24 plus a second of regeneration). Conclusion: the live "fraction dropped" is the client's copy; the server's unit, the one that fights, keeps the fraction. The pilot's claim "game.exe drops the fraction" read the client display and is REFUTED for the server (V run).

Still U: that nothing between the load and the first server frame changes the server unit's life (see D3), and no live read of the server unit exists (the bot reads the client's).

**Q4. Dead and zero-life saves, stamina, old saves?** (V run, V read)
- Life 0 is stored as 0 (V run, `zero_life`, `zero_all`). The loader has no dead-state branch for stat 6 (V read: the only other stat 6 write in 0x56b180..0x56b3d0 is FUN_0056ac10's, at 0x56acc4, on the MERCENARY unit when FUN_005541b0 says so, U for when). What the game does with a character whose life is 0 on entering is not traced (U).
- Stamina: the maximum, always (V run). d2d already refilled stamina; unchanged by the fix.
- Old-format saves (version 0x5b and below): FUN_00534020 (calls FUN_00532da0 at 0x534126; sets stats 6 and 8 at 0x5342a0 / 0x5342ae): not read in detail (U). d2d reads version 0x60 saves.

**Q5. The Paladin maximum mana (TASK-102)?** The loader keeps the maximum exactly (4992 -> 4992, V run, row `pally_life`), and d2d's `World::enter` returns 4992 for the same input (V run, 0 mismatches after the fix; the earlier 19.0 read did not come from stat 9 at load). So the 19.0 appears after the load, in a recompute with the worn items (U; TASK-102 stays open; this run has no Paladin items).

## 2. Deviations

| # | d2d | game.exe | tag | differential |
|---|---|---|---|---|
| D1 | life, mana and stamina = maximum on entering a game (upstream 756bf206, `World::take_character`) | life and mana = the save's, exactly; stamina = maximum | V run: 392 of 414 cases differ before the fix (the 22 equal ones are saves already at full) | YES: `tools/emu/fl_load.py` + `test_game --load-stats`, in the fork branch (section 4) |
| D2 | the pilot candidate r1 (76013e22): fraction dropped, clamped to the maximum | kept exactly, no clamp | V run: 341 of 414 differ | the same; r1 is SUPERSEDED by r2 and must not be opened |
| D3 | current life, mana and stamina do not follow a later change of the maximum | FUN_0055b800, the unit's stat-change callback (registered with FUN_00626d40 at 0x534914 and 0x573e5a), scales the current life by new max / old max when the maximum changes (V read, 0x55b8ec..0x55b95e: `life * new / old`, at least 1; its mana and stamina arms write stats 8 and 0xa at 0x55ba11..0x55bab9, not read in detail) | V read, behaviour U | NOT built. How: run the real FUN_00627260 on a unit whose stat list has the callback registered, set stat 7 from A to B with stat 6 = L, oracle = stat 6 afterwards; against d2d's equip, level-up and Battle Orders paths. It touches the load only if the join's FUN_0055dbc0 removes an item after FUN_0056af80 |
| D4 | n/a | a new character takes FUN_00532590 -> FUN_005706d0 (0x5325e0) | U | no |

## 3. What the oracle runs and hooks (honest limits)

`tools/emu/fl_load.py` (fork branch, section 4) boots the real 1.14d tables through the emulator (the patched stack), makes a player unit with an empty base stat list, writes a 'gf' stat block and calls **FUN_0056b180 for real**. Run for real inside it: FUN_0056a620, FUN_0056a4f0, FUN_00627260 / FUN_006270b0 / FUN_00626920 (the stat setter), FUN_00625480 / FUN_00624f60 (the getter), FUN_00625db0, FUN_00622e70, FUN_00623460, FUN_006253b0, FUN_0056af80. Hooked, returning "ok" (not the claim): FUN_0056a090 (makes the unit), FUN_0056a370, 0056a3e0, 0056a470 (moves the cursor to the stat block), 0056a710, 0056a7e0, 0056a830, 0056aa50, 0056ac10, 0056ae50, FUN_005531c0, FUN_005390d0, FUN_00539220 (the equipment slot object: empty), FUN_006221a0 (class record: zero), FUN_00611800. Limits:
- The unit has no gear, no level and no registered stat-change callback (D3); the maxima are the file's. A real character's items load before FUN_0056af80 and can raise the maxima; the restore still sets the saved values (V read), but that is not run with items.
- The save header, quest, waypoint, NPC, skill, item, corpse and mercenary parts are not run.
- The server-to-client update FUN_00548760 is V read only; the client handler is V run.
- The decode branch for versions 0x5c..0x5e (FUN_0056a620 at 0x56a643) is not run.

## 4. d2d side, the fix, the harness

Private fork branch `local/fid-load-life-mana-r2`, one commit on upstream main da2d96cf (the hash is in the report; it lives in a scratch clone and a bundle on the worker host: the push to the private remote was refused by the sandbox and waits for the coordinator, so the same-day rule of AGENTS.md is open). Files: `components/game/world/setup.cpp` (`take_character` no longer touches life and mana; stamina is still refilled), `tests/game/test_game.cpp` (the unit assertion and the mode `--load-stats CASES`, which prints `World::enter`'s stat vector), `tools/emu/fl_load.py` (the oracle). The oracle is in the same commit as the change: upstream's rule is "the oracle goes in the same pull request", one change per commit with its test (AGENTS.md "Evidence").

Run (worker host, nice 19; copies of the fork's flow are `tools/emu/fo_build.sh` and `tools/emu/fo_diff.sh` of this repo): 414 cases (`fl_load.py gen 400`) through game.exe (`fl_load.py game`) and through d2d (`test_game --load-stats`), then `diff`:

| d2d build | cases | differing from game.exe |
|---|---|---|
| upstream da2d96cf (before: refill all three) | 414 | 392 |
| r1 76013e22 (fraction dropped, clamped) | 414 | 341 |
| r2 (kept exactly; stamina refilled) | 414 | **0** |

`test_game` (full) with r2: "OK: a save's life and mana are kept on entering a game", rc 0 (worker host). With the parent's `setup.cpp` the new assertion aborts: `hero.character.stats.values[kLife] == 89 * 256 + 138` failed (rc 134, worker host).
`python3 tools/pr_guard.py upmain HEAD`: 0 FAIL, 0 WARN; `presubmit.py --quick`: 0 FAIL, 3 WARN (docs-sync, answered in the PR body; build and gcc skipped by --quick).

## 5. Bot findings

B1. Unchanged from the pilot: the bot has no load-time full-heal assumption in `internal/` (grep). The soak counts "Save and Exit" as reported, not gated (`internal/d2dsoakstat/invariants.go:120,304`); on d2d before the fix a chicken-and-rejoin returns full where the original returns hurt, so a chicken rate measured on d2d over-states survival (I). Described, not changed.
B2. The bot reads the CLIENT unit's life through `mem`: it sees whole points (floor) while the server's unit holds the fraction. A check that compares the bot's life with a computed one to the last 256th is off by up to 255/256 of a point (I; the mechanism is Q3). Described, not changed.

## 6. Drill recipe (fork verbs, to confirm U later)

Lab binary with `--lab-verbs` (docs/forkdbg.md, docs/d2d-fork.md, docs/forkcheats.md); no game is run by this task.
1. D3: hurt a Paladin to half life, equip and remove a +life item, `dump` stat 6 and 7 before and after on d2d; on game.exe do the same by play and read the client's life (it floors): current life should scale by new max / old max.
2. Dead save: take a character that died (corpse) and read stats 6, 8, 0xa at 0, 1 and 3 s after entering, on both.
3. Server value: hurt a character by 10 percent, save and exit, re-enter, and read both builds' life at 0 s; on game.exe only the client's floor is visible, so compare floors, and the fraction only on d2d.
4. A save with life above the maximum cannot be made by play: the oracle rows `above_max_*` stay V run only.

## 7. Matrix proposals

- EN04 (character load): "game.exe loader FUN_0056b180 -> FUN_0056af80 (0x56b0ff, 0x56b10d): life and mana kept exactly, no clamp, stamina = maximum (V run, fl_load.py, 414 cases); d2d refills all three (392 of 414 differ); fix on private branch local/fid-load-life-mana-r2, 0 differ (TASK-101)".
- ST01 (stats at load): "the live 'fraction dropped' is the client's whole-point copy (FUN_00548760 sar 8 at 0x5487b7, FUN_0045d9b0 shl 8 at 0x45d9d8; V run); the server keeps the fraction; the Paladin maximum 19.0 does not come from the load (TASK-102)".
- New row candidate: "stat-change callback FUN_0055b800 scales current life / mana / stamina with the maximum (D3, V read)".

## Shared-file lines for the merger
See `backlog/merge-lines/agent-fidoracle.txt`.
