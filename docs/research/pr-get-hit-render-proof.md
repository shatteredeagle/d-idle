# Proof: the get-hit DRAWN FRAME matches game.exe 1.14d's client (agent RENDERGH, 2026-10-31)

> status: research, current (card rendergh). Standard: docs/process/upstream-pr-standard.md. Offline only: no game instance, nothing pushed, posted or filed. Tags: V = run in game.exe's own code under unicorn by this agent (`tools/emu/hit_render.py`) or read in the capstone dump of the read-only copy (`/tmp/casttrace/all.txt` on the worker host); V-d2d = read or run in d2d (the worktree `~/d2d-lab/pr/render`, stacked on PR #136's eb89646e); S = upstream `docs/research/re/*`; U = unverified. The hold's LENGTH is PR #136; the decision PR #134.

## The claim (one claim, one PR)

While a monster or the merc is in its get-hit mode (mode 3; the merc is a type-1 unit on game.exe's client), the frame d2d draws at each tick of the hold is the frame game.exe's client draws: the unit's 8.8 frame position starts at 0 and, each tick, is stepped by the get-hit rate (`floor(AnimData speed x (50 + E) / 100)`, #136's `rules::get_hit_rate`) only while position + rate < frames x 256; the draw shows position >> 8. So the animation plays ONCE at the get-hit rate and stands on its last stepped frame for the rest of the hold; it never loops. A Zombie (5 frames at 152, FHR 0: rate 76, hold 17): `0 0 0 0 1 1 1 2 2 2 2 3 3 3 4 4 4`. d2d before: `(elapsed_ms / ms_per_frame) % frames` at the base rate (`0 1 2 3 4 0 1 2 3 4 0 ...`).

NOT claimed: the player's own drawn frames (d2d's `prate` stretch over the hold: measured below, left as a gap with a `// ponytail:`); what the client shows at or after its OWN end of the mode (a player: the hold's last tick; a monster: only when rate >= frames x 256); a joined game's merc (no FHR on the wire); the sub-tick time of the first step relative to the server's mode packet (both engines count from the set).

## The chain (addresses; V unless marked)

| Link | What it shows | How |
|---|---|---|
| `0x4808d5` -> `FUN_004b13a0` | the client's per-tick update of a type-1 unit: `FUN_004af400` gives the mode's row, `FUN_004b1280(unit, row)` steps, the row's column 2 (jump 0x4b15e0: 1 -> 0x4b1534, "ended" = the step's over flag) sends an ended mode to `FUN_004af6a0` | read |
| `FUN_004af400` 0x4af474..0x4af499 | the row = `0x7252e0 + mode x 12` when the MonStats record's mode bit (+0x104 + mode / 8, bit mode & 7) is set; row 3 (GH) = `[0, 1, 2]` (read with the emulator; rows 1, 2, 12..15 the same, 0 `[1, 2, 0]`, 6 `[2, 1, 3]`) | read + data |
| `FUN_004b1280` 0x4b1306..0x4b1341 | column 1 through the jump table 0x4b1388 (`0x4b1318` step always, `0x4b1327` over-else-step, `0x4b1342` over-then-stand-at-end-minus-256-rate-0, `0x4b1371` frame 1 rate 0): GH takes `0x4b1327`: `FUN_006217c0`; not over: `FUN_00623e00`; returns the over flag | read + run (the oracle calls it natively with the real row) |
| `FUN_006217c0` 0x6217fc..0x62180c | over = `+0x44 + (int16) +0x4c >= +0x48` for a plain mode (+0x30 = 0) | read + run |
| `FUN_00623e00` 0x623e55..0x623e7b | the step: `+0x44 += rate`; at or past `+0x48` it wraps (0x623ed1: minus the end, plus `FUN_00623b10` << 8), which the over-else-step order never reaches | read + run |
| `FUN_00621810` 0x621839..0x621845 | the drawn frame: `*arg2 = +0x44 >> 8` (sar); called by the unit draw at `0x4715ca` after `FUN_00620100`, the frame then fed to the composite draw | read + run |
| `FUN_00624390` 0x6243bf..0x6243d7, type-1 entry `0x624483`, tail 0x6244d3..0x6244f4 | the client's mode start: `+0x4e = 0`, `+0x44 = FUN_00623b10 << 8` (0 for GH, run), `FUN_00623f50` (the same rate as the server's: #136's chain, run from 0x624020), `+0x30 = 0`, `+0x48 = record +8 << 8` | read (stores), run (rate, start) |
| `FUN_005539b0` | the server's hold T = max(held, 1) + 1, held = `floor((frames x 256 - 1) / rate)` (#136, run natively here for each case) | run |
| `FUN_004af6a0` 0x4af80c..0x4af851 | an ended GH (no special class, +0xc4 bit 21 clear): `FUN_00463720(unit, 0x2000)` 0 -> NU through `FUN_00480e70`; it runs at tick T for held >= 1 (the first tick with k x rate >= frames x 256 = the server's end) | read; `FUN_00463720` decoded with disasm.py (0x463720..0x463735: `(unit +0xc8 & mask) != 0`, mask 0x2000; what sets that bit is not traced: U, outside the claim, after the hold) |
| `FUN_00463390` 0x463466..0x4634f9, table 0x711e00 row 4 `[0, 1, 2]`, jump 0x463710 | the player's tick: the same over-else-step (`0x463487`), then column 2 = 2 re-tests `FUN_006217c0` after the step and ends the mode there (0x463618: NU / TN through `FUN_00480e70`, or only a flag at player data +0x150 when +0x154 is set: U) -> the client ends the player's GH on the hold's last tick | read + run; not claimed |

## Differential (V): game.exe's own code vs d2d

Oracle `tools/emu/hit_render.py` runs, per case: `FUN_00623f50` from 0x624020 (the rate; hooks as hit_len.py: the stat getters `FUN_00625500` / `FUN_00625480`, the state test `FUN_00639df0` 0), `FUN_005539b0` natively (the hold T; `FUN_005417d0` hooked to record the event), then per tick 1..T-1 `FUN_004b1280(unit, 0x7252e0 + 36)` natively (a monster; it reads a planted MonStats row through the global 0x744304, class 0, byte +0x1a4 = 0) or `FUN_00463390`'s step section 0x463466..0x4634bd plus `FUN_006217c0` (a player), and `FUN_00621810` natively for the frame; -1 from the tick the client ends the mode. Cases: hit_len.py's 10,836 boundary (every FHR 0..300 x 18 records x 2 kinds) + 100,000 random (seed 7). Checksums (md5 prefixes): cases `e8083048` (identical to #136's), game `e05624f4`, before `a8f7e58d`, after `9f28da81`.

| Side | Code | Cases (kind 1) | Ticks | Differ |
|---|---|---|---|---|
| BEFORE | `tools/emu/hit_render_harness.cpp` (quotes town.cpp:195..203 rate 1.f, world_view.cpp:440, world_view.hpp:164..165 `% fpd`, gamedata.hpp:391) | 55,089 | 1,747,580 | **1,488,667 ticks in 49,641 cases** |
| AFTER | upstream's `tests/test_combat --get-hit-frames` (`rules::get_hit_frame`) | 55,089 | 1,747,580 | **0** |
| player, unchanged (not claimed) | the harness's prate lines (fight.cpp:465, town.cpp:3422) | 55,747 | 1,653,995 | 136,651 ticks in 27,840 cases |

523 kind-1 ticks (523 cases) are "client ended" before the hold's end: all have rate >= frames x 256 (the client's over at tick 1, the server's hold 2); no vanilla record (frames >= 3 at speed <= 512). The player's hold ends on the client one tick early in every case (55,739 ticks).

```sh
# the worker host; /tmp/render-emu holds the oracle copies, cases.tsv, game.tsv, before.tsv, after.tsv
cd /tmp/render-emu && source /tmp/casttrace-venv/bin/activate && export PYTHONPATH=~/d2d-lab/pr/hitlen/tools/emu D2_MPQ_DIR=~/d2d-lab/data
python hit_render.py gen 100000 7 > cases.tsv
python hit_render.py --exe /tmp/casttrace/game.exe game cases.tsv > game.tsv          # about 4 min
g++ -O2 -std=c++20 -o hit_render_harness hit_render_harness.cpp && ./hit_render_harness < cases.tsv > before.tsv
/tmp/render-build/tests/test_combat --get-hit-frames < cases.tsv > after.tsv
python hit_render.py compare game.tsv before.tsv cases.tsv 1    # 1488667 ticks in 49641 cases differ
python hit_render.py compare game.tsv after.tsv cases.tsv 1     # 0 ticks in 0 cases differ
python hit_render.py compare game.tsv before.tsv cases.tsv 0    # the player, measured only
```

## The patch (V-d2d)

`~/d2d-lab/pr/render` (a worktree of the clone, branch `fix/get-hit-render`, base #136's eb89646e), one commit `57bd4b56` "Draw a get-hit once at its rate, held on the last frame", author and committer `shatteredeagle <26069301+shatteredeagle@users.noreply.github.com>`, body `Agent: Claude Code`. 10 files, +71 -3: `rules::get_hit_frame(frames, rate, tick)` (combat.hpp); `Unit::get_hit_rate` (world_view.hpp) and `get_hit_elapsed` after `object_elapsed` at both draw sites (shadow and body, world_view.cpp); town.cpp `view_units` sets it for the merc and a monster in "GH" (not a sequence row, not buried) from the anim's speed and the unit's FHR; `View::Merc::fhr` (world.hpp) filled from `merc_fighter()` (world.cpp); the replication reader's ponytail (no FHR on the wire); the player's ponytail in `set_pmode` rewritten; test_combat asserts (the Zombie's 17 frames, the Sorceress, past the hold, rate 512, rate 0) and `--get-hit-frames`; combat.md "Drawn frame". Other modes: `get_hit_rate` stays 0 unless the mode is "GH", and `get_hit_elapsed` returns `elapsed` unchanged then (read; the 23 `test_game` scenarios pass).

Checks on the worker host (`/tmp/render-build`, Release): `ninja test_combat test_game` ok; `test_combat` -> `ok`; `D2_MPQ_DIR=... D2_PATCH_INSTALLER=... test_game` 23 OK rc 0; `pr_guard.py upstream/main` and `pr_guard.py fix/get-hit-length` 0 FAIL 0 WARN; `presubmit.py --quick`: 1 FAIL `doc-paths` (combat.md line 234 names `tools/emu/hit_len.py`: #136's text; the base's new commit 9006a688 ships that file), WARNs siblings (merc, monster, not player / pet / co-op: answered in the draft), net roles (client, not host: the draw is the client's), doc-claims ("never loops": the differential), build skipped (built).

## Residual risks and limits (say them in the PR)

1. The player's own frames: d2d's prate (`length_ms / hold`) differs from the stepped rule on 136,651 of 1,653,995 ticks (e.g. FHR 33: frame 3 vs 2 at tick 4); and game.exe's client ends the player's GH on the hold's last tick (the +0x154 path U). Left as a gap (`// ponytail:` in set_pmode).
2. rate >= frames x 256 (no vanilla record): the client ends at tick 1, the server holds 2; d2d draws the stepped frame on tick 1.
3. The merc's FHR is `merc_fighter().fhr` (its items); a joined game's merc has none on the wire (drawn at FHR 0, ponytail); a party member's merc / pets in GH keep the plain loop (ponytail).
4. The COF's frames per direction are taken as AnimData's (composite_frames divides by the same ms_per_frame; `past_cof` is unchanged); a record whose two counts differ is U.
5. The first step's tick relative to the server's set (packet order on the client) is not traced: both sides count tick 0 at the set.
6. No live or state-hash run (`tools/d2ddet`): the regression evidence is `test_game` (23) and `test_combat`.

## Base and stack

The patch is on eb89646e. During this work `fix/get-hit-length` moved to `9006a688` ("Record the get-hit renderer gap and ship the length oracle"): +8 ponytail lines in `components/game/ai.cpp` `set_mode` (the renderer gap this patch closes: drop them on rebase), combat.md's "Ported" bullet reworded (my "Drawn frame" bullet follows it: conflict), `tools/emu/hit_len.py` / `hit_len_harness.cpp` added (no conflict; the oracle `hit_render.py` could ship the same way, `tools/` needs the CODEOWNERS review). Rebase: `git rebase 9006a688` on `fix/get-hit-render`, resolve combat.md (keep both bullets), delete the ai.cpp note, re-run the AFTER differential, test_combat, test_game, pr_guard, presubmit.

## Verifier verdict (independent adversarial agent, general-purpose, Fable, read-only)

PENDING
