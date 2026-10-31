# Lut Gholein's two missing level-seed steps: traced, fixed, proven

> status: done 2026-10-31 by agent LUTPROOF (card lutproof; follow-up of docs/research/lutgholein-trace.md). Tags: V = run by this agent (command named), S = read in upstream source or docs (psi29a/d2decomped main 32768ddb), U = unverified. Raw outputs stay on the worker host under `~/d2d-lab/pr/lut-*` (lab space, not in this repo). Nothing was posted upstream; the fix sits on a local branch only.

## Answer in four lines

1. game.exe steps Lut Gholein's level seed (level +0x1c4) 51 times before the first room: two in `FUN_00667620`, the preset-unit "roll to stay" (write at `0x667744`, the `low % 3 == 0` branch, called through `FUN_00667970`, the Scan / Pops roll at generation), then once per room in `FUN_0066b3e0` (49 rooms). No file roll on the level seed for level 40 (the act layout fixes the file). V (lut_trace.py)
2. The two units are the DS1's monsters with raw ids 25 and 26, which MonPreset (act 2) maps to MonStats 204 and 205 (Act2Vendor1 / Act2Vendor2), two of the four ids `FUN_00667620` rolls (0xcc, 0xcd, 0x173, 0x174, jump table at `0x6677d4`). V (disassembly, monpreset.bin, units dumps)
3. d2d's generator (`stays`, `rolled_unit`, components/drlg/units.hpp) does the same roll, but `tools/drlg-dump` opened no `patch_d2.mpq` from the MPQ folder (only `$D2_PATCH_INSTALLER`, then d2exp and d2data), and `data\global\excel\monpreset.bin` exists only in the 1.14d patch: with no MonPreset table `ds1_units` keeps the raw ids, nothing matches `rolled_unit`, the two steps never happen, every room is seeded two rooms late. The game's own loader (`open_game_data`) does open the folder's patch, so the d2d game is not affected; the tool and upstream's `diff_drlg.py` gate were. V/S
4. Fix (one commit on the upstream clone, `tools/drlg-dump/drlg-dump.cpp` only): open the MPQs as the game does. Differential with upstream's own `diff_drlg.py`: level 40 seeds 1-20, tiles / units / seeds 0/20 -> 20/20; the other towns and the act 1 Scan / Pops level unchanged or improved (table below). V

## 1. The trace in game.exe (lut_trace.py)

`tools/emu/lut_trace.py` (this repo; copied next to upstream's `drlg.py`) boots upstream's emulator, builds the act server-side with `FUN_006194a0` exactly as `drlg.py <seed> 40` does, finds level 40, boots again fresh (the bump allocator puts the level record at the same address) and hooks a memory-write watch on level +0x1c4..+0x1cb plus code hooks at `FUN_0045c370` / `FUN_0045c3e0` (never fired: every step in this path is inlined). Nothing is hooked out. Run: `cd ~/d2d-lab/pr/lut/tools/emu && D2_MPQ_DIR=/tmp/ug-emu /tmp/casttrace-venv/bin/python lut_trace.py 161464580 40` (output `~/d2d-lab/pr/lut-trace-161464580.txt`). V

| # | write eip | function | what |
|---|---|---|---|
| 1 | `0x681f4f` | `FUN_00681e09` | memset of the level record (`FUN_00642ae0` on the stack) |
| 2, 3 | `0x650e40` | `FUN_00650e40` | seed set `{act+0x470 + 40, 0}` then `{.., 666}` (from `FUN_00642ae0`, then `FUN_006424a0`), as drlg.md "Seeds" says |
| 4, 5 | `0x667744` | `FUN_00667620` | the `% 3` roll of a monster id in 0xcc..0x174 via the table at `0x6677dc`; `FUN_00667970` at `0x667a3a` on the stack |
| 6..54 | `0x66b451` | `FUN_0066b3e0` | room allocation, one per room (`FUN_00666680` at `0x666694`, `FUN_00667ed0` at `0x6680ae` on the stack) |

So 2 + 49 steps; the room list's seeds (list order) end the output and equal `drlg.py`'s. The stack lines are emu.backtrace's heuristic (code addresses on the stack); the eip of each write is exact.

## 2. What game.exe rolls (disassembly, `disasm.py --exe /tmp/casttrace/game.exe 667620`)

`FUN_00667620(seed in [ebp+8], unit list in ebx)`: type 1 (monster) with id below the MonStats count (`[0x744304]+0xa80`): `id - 0xcc` indexed into the byte table at `0x6677dc` (two cases, `0x6677d4`): 0xcc, 0xcd, 0x173, 0x174 take the roll at `0x667721` (step, keep when `low % 3 == 0`), every other id is kept without a step. MonPlace ids 0x21 / 0x22 / 0x23 and objects 0xc4 / 0x105 / 0x245 have their own rolls (as units.hpp `stays` ports them). V (read), S (drlg.md "Preset units")

`monpreset.bin` (1.14d patch, 920 bytes, 229 rows `{act, kind, u16 id}`): act 2 index 25 = (1, 204), 26 = (1, 205). `LutW.ds1` (header act field 1 = act 2) carries monsters 25 and 26; game.exe's units dump shows ids 176, 177, 195.. (mapped) and, for seed 161464580, no 204 / 205 at all: both rolls failed to stay. d2d before the fix listed `1:25` at room 24,8 (23,34) and `1:26` at room 32,16 (14,24), and every other monster by its raw id. V (`drlg.py 161464580 40 units` vs `drlg-dump ... 40 units`)

## 3. Why d2d skipped them

- `tools/drlg-dump/drlg-dump.cpp` (upstream main) built its stack twice by hand: `if ($D2_PATCH_INSTALLER) push_installer(it); d2exp.mpq; d2data.mpq`. No `patch_d2.mpq` from the folder. S (lines 490-492, 704-706 of main)
- `mpq-cat ~/d2d-lab/data "data\global\excel\monpreset.bin"`: not found (the folder's `Patch_D2.mpq` is never opened by the tool, and d2exp / d2data have no such file); the same against `/tmp/ug-emu` (lowercase `patch_d2.mpq` symlink): 920 bytes. V
- `components/drlg/outdoor_data.hpp` fills `ids.monpreset[act]` from that file only; `units.hpp ds1_units` leaves an id alone when the act's table is shorter than the id; `rolled_unit` then sees 25 / 26, not 204 / 205. S
- `components/game/gamedata_load.cpp open_game_data`: `$patch`, else `find("patch_d2.mpq")` (case-insensitive, `d2d::install::find_file`), else `LODPatch_114d.exe`, then the base MPQs. The game and drlg-dump's `monsters` / `objgroups` / `game` / `npcs` modes go through it; the plain / tiles / units / seeds dump and `layout` did not. S
- `diff_drlg.py` sets `D2_PATCH_INSTALLER` to `~/Downloads/Diablo II + LoD/patch/LODPatch_114d.exe` by default; with that file (the maintainer's setup) the tool had the patch and upstream's 20/20 rows hold; without it `push_installer` fails ("MPQ open failed") and the tool prints nothing; with the variable unset the tool silently ran on CD-era data. V (before binary run three ways)

## 4. The fix (clone `~/d2d-lab/pr/lut`, branch `fix/lut-gholein-preset-rng`, commit 632d5cc2, author shatteredeagle, no trailer)

`static d2d::mpq::Stack open_mpqs(const fs::path& dir)` in drlg-dump.cpp: `$D2_PATCH_INSTALLER` when set and present (a stderr note when set but missing, as the game warns), else the folder's `patch_d2.mpq`, else `LODPatch_114d.exe` (both via `d2d::install::find_file`), `.mpq` pushed as an archive, `.exe` as an installer, then d2exp and d2data; both hand-built stacks call it. `#include <install.hpp>`, `<cctype>`. 27 insertions, 6 deletions; `pr_guard.py upstream/main HEAD`: 0 FAIL 0 WARN; `presubmit.py --quick`: 0 FAIL, 3 WARN error-path (throwing `fs::exists` overloads, the same call the pre-existing d2exp / d2data line makes) + the skipped build. V

## 5. Differential (upstream's gate, before and after)

`cd ~/d2d-lab/pr/lut/tools/emu; D2_BUILD=build-<k> D2_MPQ_DIR=/tmp/ug-emu /tmp/casttrace-venv/bin/python diff_drlg.py <range> <level> tiles units [seeds]`, where `build-before/tools/drlg-dump/drlg-dump` and `build-after/...` are wrappers that unset `D2_PATCH_INSTALLER` and exec the upstream-main build (`~/d2d-lab/pr/lut-drlg-dump.before`) or the fixed one (`.after`); the only difference between the two runs is the binary. Outputs `~/d2d-lab/pr/lut-diff/<k>-<level>.txt`. V

| level | seeds | kind | before | after |
|---|---|---|---|---|
| 40 Lut Gholein | 1-20 | tiles / units / seeds | 0/20 / 0/20 / 0/20 | 20/20 / 20/20 / 20/20 |
| 1 Rogue Encampment | 1-10 | tiles / units | 10/10 / 0/10 | 10/10 / 10/10 |
| 2 Cold Plains (Cottages 2 Pops) | 1-10 | tiles / units | 10/10 / 6/10 | 10/10 / 10/10 |
| 75 Kurast Docks | 1-10 | tiles / units | 10/10 / 0/10 | 10/10 / 10/10 |
| 103 Pandemonium Fortress | 1-10 | tiles / units | 10/10 / 0/10 | 10/10 / 10/10 |
| 109 Harrogath | 1-10 | tiles / units | 0/10 / 0/10 | 10/10 / 10/10 |

Plain dumps (`drlg-dump DATA a-b LEVEL DIR` vs `drlg.py a-b LEVEL DIR`, `cmp`): level 40 before 0/20, after 20/20; levels 1, 2, 75, 103, 109 10/10 before and after. `lut_seeds.py` on level 40: before "equal only with a two-room shift 20/20", after "all rooms equal 20/20". The units rows show the same missing table in every act (raw ids); Harrogath's tiles come from the patch's own act 5 files, which the tool now reads as the game does. Upstream tests: `test_combat` ok, `test_game` with `D2_MPQ_DIR=/tmp/ug-emu` OK (exit 0); the fix touches no library. V

## 6. Limits

- One tool (`drlg-dump`), two call sites; the d2d game's generation of Lut Gholein was not measured by this agent (its loader reads the patch, S; LUTGHOLEIN's live comparison was game.exe vs drlg-dump). U for the in-game result, outside the claim.
- The trace is one map seed (161464580); the 20-seed differential covers the rest. The call chain above `FUN_00667970` is from a heuristic backtrace.
- A user with the installer at diff_drlg.py's default path saw no difference before; the change reaches users whose folder has `patch_d2.mpq` (every installed 1.14d game) and no installer, and makes a set-but-missing `$D2_PATCH_INSTALLER` fall back with a note instead of failing.
- `tools/` needs the maintainer's CODEOWNERS review. The maintainer's own gate had the installer (`diff_drlg.py`'s default path), so for them the change is a no-op; without that file their default run failed outright before. mpq-cat / MpqCache still look for a case-sensitive `patch_d2.mpq` (follow-up candidate, not this claim).

## Verifier

Independent adversarial read-only verifier: HOLDS WITH LIMITS (report verbatim and what was acted on in docs/research/pr-lut-gholein-evidence-pack.md).
