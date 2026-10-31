# Proof: the get-hit (hit recovery) LENGTH matches game.exe 1.14d (agent HITLEN, 2026-10-31)

> status: research, current (card hitlen). Standard: docs/process/upstream-pr-standard.md (differential vs game.exe's own code, zero mismatches after, one claim, limits disclosed, adversarial check). Offline only: no game instance, nothing pushed, posted or filed. Tags: V = run in game.exe's own code under unicorn by this agent (`tools/emu/hit_len.py`) or read in the capstone dump of the read-only copy (`/tmp/casttrace/all.txt` on the worker host); V-d2d = read or run in d2d (upstream main 32768ddb, the patched clone); S = upstream `docs/research/re/*`; U = unverified. The decision (which hits cause a get-hit) is PR #134, not this.

## The claim (one claim, one PR)

When a unit goes into its get-hit mode on the server (a player's mode 4, a monster's or the merc's mode 3), d2d now holds it for the ticks game.exe does: the rate is `floor(AnimData speed x (50 + E) / 100)` clamped to 0..0x7fff with E = `FUN_00621740(unit, row 1)` = `floor(120 v / (120 + v))` of stat 99 (FHR, v != 0), and the mode-end event sits at the game frame + `max(held, 1) + 1` where held = `floor((frames x 256 - 1) / rate)` (a rate of 0: the next frame). A Sorceress (SOGHHTH 8 frames at 256) holds 16 / 15 / 14 / 13 / 12 / 11 / 10 / 9 / 8 / 7 / 6 ticks at FHR 0 / 5 / 9 / 14 / 20 / 30 / 42 / 60 / 86 / 142 / 280 (the community table's 15 / 14 / ... / 5 frames is the client's animation, one fewer); a Zombie (5 at 152) 17, a Fallen (7 at 256) 14. d2d before: the player `speed_frames` (100 + E on the frame count: 8 ticks for the Sorceress), monsters the plain animation length over (100 + anim_pct) (Zombie 9, Fallen 7), the merc the plain length.

NOT claimed: the block length (mode BL: 50 + FBR, 100 under state 0x65: d2d keeps `speed_frames` with FBR); the client's drawn frame count; which hits cause a get-hit (PR #134); the knockback's slide (d2d's `knock_slide`, "ponytail" in fight.cpp); the stunned / frozen gates (the decision's).

## The chain (addresses; V unless marked)

| Link | What it shows | How |
|---|---|---|
| `0x624089`..`0x62409d` | FUN_00623f50's dispatch: type 0 with mode 4, type 1 with mode 3 -> the get-hit branch | read + run (the oracle enters at 0x624020, the instruction after the cast test, and takes this dispatch itself) |
| `0x62409f`..`0x6240b1`, `0x62435c`..`0x624385` | rate = `FUN_00621740(unit, 1)` + 50, x record +0xc (speed), x 0x51eb851f >> 37 (/ 100, floored), at least 0, at most 0x7fff, stored at unit +0x4c. No other stat, no state test (the block branch 0x62403d..0x624084 is the one with `FUN_00639df0(unit, 0x65)`) | read + run |
| `FUN_00621740` (0x621740..0x62177e) | row 1 of the 12-byte table at 0x6e8e24: flag 1, factor 120, stat 99 through `FUN_00625500(unit, 99, 0)`; v != 0: 120 v / (120 + v) (idiv, toward zero); else v | read + run (the getter hooked to the case's FHR) |
| `FUN_005533d0` (0x5533d0..0x553469) | the server's mode start (SUnit, S char-select.md): +0x4e = 0, +0x30 = 0, +0x44 = 0, `FUN_00623f50`, +0x48 = record +8 (frames) << 8 for a unit with a path (+0x2c) | read |
| `FUN_005539b0` (0x5539b0..0x553af9) | the mode's events: start = `FUN_00623b10(unit)` << 8 (0: 0x623b61 non-player, 0x623b6b..0x623b74 a player's mode outside 7..16), rate = +0x4c, end = +0x48; rate 0: event 1 at frame + 1 (0x5539ee..0x553a00); else cur = start + rate, while cur < end: frame + 1 (0x553a36), the record's per-frame event bytes (+0x10, zero here), cur += rate (0x553ab2..0x553ac0); then frame + 1 if it still equals the game frame (0x553ac9..0x553ad1), + 1 (0x553ad8), event type 1 there (`FUN_005417d0(game, unit, 1, frame, 0, 0)`, 0x553ad4..0x553ae2) | read + run (`FUN_005417d0` hooked: recorded) |
| `FUN_005416b0` 0x5416e3..0x5416ed | an event frame <= the game frame becomes game frame + 1 (never in the past) | read |
| callers of `FUN_005539b0` | `game-calls.tsv`: 0x57fdc0 (BL), **0x57fe30 (the player's mode-4 handler, table 0x6e1740[4])**, 0x57fe90, 0x57fef0, 0x580ec0 (DT), **0x5a7c20 (the monster's mode set, 0x5a7e2b, after `FUN_005533d0` at 0x5a7db1)**; `FUN_005809d0` dispatches the player's mode through 0x6e1740 (0x580a54..0x580a60); `FUN_0057cee0` plays the get-hit with `FUN_005809d0(mode 4)` / `FUN_005a7e60(mode 3)` (HITREC) | read (S for the tsv) |
| the end | `FUN_005414d0` runs the frame's events (S network.md: before the unit updates); the player's default handler `FUN_00581220` -> 0x6e1810[1] = `FUN_00581020`, which plays NU through `FUN_0057f020(game, unit, 1, 0, 0)` at 0x5810a7..0x5810b1 for any mode but 0 and 0x13; a monster's mode table at 0x6e2298 (16 bytes a mode) ends mode 3 with `FUN_005a8030`, which goes to NU through `FUN_005a7c20` (S monster-ai.md) | read (the tables read with the emulator) |

Closed U items from DECOMPREC: the server's monster mode end is the same event as the player's (no `- 1`: the client tick `FUN_00463390` shows one frame fewer because it tests before and after its step); record +0x64 belongs to the decision, not the length; a frozen / stunned unit's exemption is the decision's too; chill (stat 69) and an aura's attackrate / other_animrate (stats 68 / 69) are not read by the get-hit branch (the oracle feeds -50 / 20 / 100 there and the rate does not move).

## Differential (V): game.exe's own code vs d2d

Oracle `tools/emu/hit_len.py` runs, per case, `FUN_00623f50` from 0x624020 (hooks: `FUN_00625500` / `FUN_00625480` return the case's stat by id, `FUN_00639df0` 0) and `FUN_005539b0` natively (`FUN_00623b10` natively; `FUN_005417d0` hooked to record) and prints the rate and the event frame minus the case's frame. Cases: 10,836 boundary (every FHR 0..300 for 18 (speed, frames) records of both kinds, Sorceress 8/256 .. Zombie 5/152) + 100,000 random (kind, speed 1..1024, frames 1..60, FHR 0..1000, a monster's anim_pct, stats 67 / 68 / 69 as noise, the game frame), seed 7.

| Side | Code | Cases | Differ |
|---|---|---|---|
| BEFORE | `tools/emu/hit_len_harness.cpp` (quotes fight.cpp:457..460 `speed_frames` x `ms_per_frame`, ai.cpp:172 `length_ms` x 100 / (100 + anim_pct), the 40 ms tick) | 110,836 | **102,980** |
| AFTER | upstream's own `tests/test_combat --get-hit-length` on the patched clone (`rules::get_hit_rate` / `get_hit_ticks`) | 110,836 | **0** |

Checksums on the worker host (md5 prefixes): cases `e8083048`, game `fcad023b`, before `aa48eb19`, after `1c22417d`.

```sh
# the worker host; the clone ~/d2d-lab/pr/hitlen, read-only game.exe copy, never run as a game
cd ~/d2d-lab/pr/hitlen/tools/emu && cp ~/code/d2lod-wt/hitlen/tools/emu/hit_len.py . && source /tmp/casttrace-venv/bin/activate
python hit_len.py gen 100000 7 > /tmp/hitlen-cases.tsv
D2_MPQ_DIR=~/d2d-lab/data python hit_len.py --exe /tmp/casttrace/game.exe game /tmp/hitlen-cases.tsv > /tmp/hitlen-game.tsv   # about 1 min
g++ -O2 -std=c++20 -o /tmp/hit_len_harness ~/code/d2lod-wt/hitlen/tools/emu/hit_len_harness.cpp   # (copy it over first)
/tmp/hit_len_harness < /tmp/hitlen-cases.tsv > /tmp/hitlen-before.tsv
/tmp/hitlen-build/tests/test_combat --get-hit-length < /tmp/hitlen-cases.tsv > /tmp/hitlen-after.tsv
python hit_len.py compare /tmp/hitlen-game.tsv /tmp/hitlen-before.tsv /tmp/hitlen-cases.tsv   # 102980 of 110836 differ
python hit_len.py compare /tmp/hitlen-game.tsv /tmp/hitlen-after.tsv /tmp/hitlen-cases.tsv    # 0 of 110836 differ
```

## The patch (V-d2d)

`~/d2d-lab/pr/hitlen` (fresh clone of the user's fork; `upstream` = psi29a/d2decomped, main 32768ddb), branch `fix/get-hit-length`, one commit `a2f4a647` (amended after the verifier: the FHR comment) "Get-hit length by AnimData speed x (50 + FHR)", author and committer `shatteredeagle <26069301+shatteredeagle@users.noreply.github.com>`, no trailer, body three lines. 7 files, +86 -9: `rules::get_hit_rate` / `get_hit_ticks` (combat.hpp); `game::get_hit_ms` (ai.hpp: ticks x 40 ms on the mode's AnimData record); `Fight::set_pmode` (the GH branch; BL keeps `speed_frames` with FBR); `set_mode` (ai.cpp: a monster's GH, its `MonProps::fhr` from MonProp's kFasterGetHitRate, new in monsters.hpp); the merc's two GH sets (fight.cpp `merc_turn`'s `set` and the struck-frame set, `merc_fighter().fhr`); `test_combat` asserts (the Sorceress table, the Zombie, a rate of 0, frames 1) and `--get-hit-length`; combat.md's section "Hit recovery: the length" and the FBR line.

Checks on the worker host (`/tmp/hitlen-build`, Release, FETCHCONTENT from `~/d2d-lab/src/build/_deps`): `ninja test_combat test_game` ok; `./tests/test_combat` -> `ok`; `D2_MPQ_DIR=~/d2d-lab/data D2_PATCH_INSTALLER=~/Downloads/LODPatch_114d.exe ./tests/test_game` -> 23 `OK`, rc 0 (unchanged scenarios: the arrow get-hit at test_game.cpp:551 / 580 only asserts the mode, not its length); `python3 tools/pr_guard.py upstream/main` 0 FAIL 0 WARN; `python3 tools/presubmit.py --quick` 0 FAIL, 2 WARN (siblings: "merc, monster, not player": the player IS in the diff (set_pmode), pets have no GH set in d2d, the co-op partner is a player; build skipped) answered in the draft body.

## Residual risks and limits (say them in the PR)

1. The generator keeps FHR >= 0: d2d's `effective_speed` returns v for v <= 0 while `FUN_00621740` divides (120 v / (120 + v), toward zero) for any v != 0; a negative FHR (no vanilla item) would differ. Not claimed.
2. d2d times in ms (ticks x 40) on a 40 ms world tick (town.cpp:2945); the mode ends at the first tick with now >= until, which is the same tick count. A d2d build with another tick length is not covered.
3. The monster's FHR is MonProp's kFasterGetHitRate only (stat 99 from auras / states is not modelled; game.exe reads the full stat); the merc's is `merc_fighter().fhr` (its items). Act 1 Normal monsters carry none (U: MonProp rows not listed).
4. Not ported: the block length (50 + FBR), the client's drawn frame (prate is derived from the new length), the per-frame event bytes of the record (zero for every GH record: U, the oracle zeroes them; they only add type-0 events, not the end).
5. No live or state-hash run: `tools/d2ddet/pair.sh` runs the full bot on the lab fork's binary (determinism of one binary, not before / after upstream trees); `cmd/d2dcasts2` needs the lab fork's debug verbs. Both left undone; the regression evidence is upstream's `test_game` (23 scenarios) and `test_combat`.
6. The order inside a tick (the end event runs before the unit updates in game.exe; d2d ends the mode in the unit's own frame step) is the same for both engines' own callers but was not traced for every reader of the mode in the same tick.

## Verifier verdict (independent adversarial agent, general-purpose, Fable, read-only)

**CLAIM HOLDS WITH LIMITS.** It hand-read `FUN_005539b0` (0x5539b0..0x553af1), `FUN_005416b0`, `FUN_00623f50`'s prologue / dispatch / get-hit tail, `FUN_00621740`, both branches of `FUN_005533d0`, `FUN_0057fe30`, `FUN_005a7c20`'s tail, `FUN_005a6b10` and its table (0x6e23d0 row 3 = [0, 1, 0]: a monster's mode 3 takes `FUN_005539b0`, never `FUN_00553f00`), `FUN_00581020`, `FUN_005a8030` (0x5a8270..0x5a8322) and `FUN_00553990` (both handlers clear queued type 0 / 1 events first: no stale end), read the patch and ran 25 adversarial cases through the oracle and `test_combat --get-hit-length` (`/tmp/hitlen-verify-adv/` on the worker host). Findings and what was done: (1) T = max(held, 1) + 1 confirmed at 0x5539ec / 0x553a30..0x553ac0 / 0x553ac9 / 0x553ad8; `FUN_005416b0`'s bump never fires. (2) The paths hold; a unit without a path (+0x2c = 0) skips `FUN_00623f50` in `FUN_005533d0`: server units always have one. (3) Event 1 ends the mode at that frame on both kinds; `FUN_005a8030`'s 0x5a8270 reads the bss byte table 0x73c6d0[mode] (no writer in the dump; S monster-ai.md: set for WL and RN only, so mode 3 goes to NU through `FUN_005a7c20`): listed as S. (4) LIMIT: the tail divides unsigned and `FUN_00621740` divides for any v != 0, d2d's `effective_speed` only for v > 0 and the clamp floors at 0: FHR -42 / -100 / -200, speed -256, FHR 2e7 differ (FHR -120 divides by zero in game.exe); 0 such cases in the 110,836 and none reachable in vanilla: limit 1, and the patch's comment now says so (amended into `a2f4a647`). (5) In-range edges agree (speed 0 -> 1 tick, frames 0 -> 2, speed 0x7fff / 0xffff -> 2, frames 100000 -> 200000, game frame 0 and 2^20 - 1, anim 50, stats 67..69 at 100; stat 99 is read through the same getter for both kinds). (6) d2d side: BL unchanged, `prate` follows the new length (intended), `MonProps::fhr` from MonProp only, the merc `merc_fighter().fhr`. Not checked by it: whether any MonProp row carries stat 99; the oracle's skipping of `FUN_00620f00` / `FUN_00645270` (the record lookup; the branch reads only [record + 0xc]). After the amend: test_combat ok, 0 of 110,836 differ (after md5 unchanged `1c22417d`), pr_guard 0 / 0.
