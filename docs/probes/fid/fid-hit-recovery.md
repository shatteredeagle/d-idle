# fid-hit-recovery (TASK-91, matrix row SK16): player hit recovery, d2d against game.exe

> status: finished 2026-11-08 by agent decomp5 (Sonnet). Result: **NO DEVIATION in the covered paths on current upstream main (1453f853); all three get-hit drafts are merged upstream.** No PR candidate. Open (U): the mode-transition gate for a get-hit that arrives during a cast or attack, and the player's own drawn frames. Nothing posted, nothing pushed.

Tags: V run (game.exe's own code run by the emulator, compared with d2d's real functions in `test_combat`, byte for byte), V read, D, I, U.

## Q1. Is the get-hit LENGTH (FHR table) equal? (AC #1) V run

`tools/emu/hit_len.py` (upstream's) runs FUN_00623f50 from 0x624020 (mode dispatch and the get-hit branch 0x62409f..0x624385, FUN_00621740 natively: `120 v / (120 + v)`, + 50, x AnimData speed / 100) and FUN_005539b0 (the server's mode-end event at game frame + max(n, 1) + 1) natively; d2d side `test_combat --get-hit-length` (`get_hit_rate`, `get_hit_ticks`). Command: `tools/emu/d5_hit_run.sh 100000 7` on the worker host.

| Cases | Differ |
|---|---|
| 110,836 (every AnimData get-hit record of Act 1, a grid of 18 (speed, frames) records at FHR 0..300 for player and monster, plus 100,000 random incl. FHR to 1000) | **0** |

The breakpoints game.exe's rule produces (`tools/emu/d5_fhr.py`, first FHR at which the hold shortens), identical on d2d:

| AnimData record (speed, frames) | Hold at FHR 0 (ticks) | Breakpoints (FHR) |
|---|---|---|
| 256, 8 | 16 | 0, 5, 9, 14, 20, 30, 42, 60, 86, 142, 280 |
| 256, 6 | 12 | 0, 6, 13, 20, 32, 52, 86, 174 (and 600, beyond the grid) |
| 256, 5 | 10 | 0, 7, 15, 27, 48, 86, 200 |
| 256, 7 | 14 | 0, 5, 10, 16, 26, 39, 56, 86, 152 |
| 256, 4 | 8 | 0, 9, 20, 42, 86, 280 |

The row "256, 8" is the card's table 0/5/9/14/20/30/42/60/86/142 (the Sorceress and Necromancer breakpoints of the community tables, plus 280); 256/5 and 256/6 are the Paladin/Barbarian and Amazon tables. Which class uses which record is the game's AnimData (I from the community tables; the per-class record was not read from AnimData here, U). 18 records, 0 differences on the full 0..300 FHR range.

## Q2. The "2-frame wait" (riskmap `fight.hpp:715`, AC #2) V run

The old listing ("a hit taken in GH waits 2 frames in game.exe, fires at once in d2d") predates the merged PRs. The wait is the `+ 1` after `max(n, 1)` of the mode-end event in FUN_005539b0; it is part of `ticks` in the table above, so 0 differences over 110,836 cases covers it. The old line `fight.hpp:715` no longer exists in that form (get-hit code moved to `rules/combat.hpp` and `fight/`).

## Q3. Decision (does a blow cause a get-hit) V run

`tools/emu/hit_rule.py` runs FUN_0057cee0 (class fix-up FUN_0057ce30, stun skip, FUN_0057cb00's size test with the two seed rolls through FUN_00472210, the mode call) natively; d2d `test_combat --get-hit`. Command: `tools/emu/d5_hit_rule_run.sh`: **73,694 frames (21,532 with several damage records), 0 differ (player 0 of 36,904, monster 0 of 36,790).**

## Q4. Drawn frame during the hold V run (monsters and merc only)

`tools/emu/hit_render.py` (FUN_004b1280, FUN_00621810, FUN_00623f50, FUN_005539b0) against `test_combat --get-hit-frames` for the 55,089 kind-1 cases: 1,747,580 ticks, **0 differ**. Gap: the player's own frames are not compared (the client's player step differs; listed gap of pr-get-hit-render-proof.md), U.

## Merge state of the drafts (AC #2)

| Draft | Upstream | Evidence |
|---|---|---|
| pr-get-hit-rule (hit class and share of life) | merged (PR #134, merge 923382d3 2026-10-31) | upstream git log; Q3 above is 0 |
| pr-get-hit-length (50 + FHR, event offset) | merged (`get_hit_ticks`, `--get-hit-length`, tools/emu/hit_len.py are in the upstream tree) | Q1 above is 0 |
| pr-get-hit-render (drawn once, held on the last frame) | merged (e0d0bf12 and the PR #136 hold; `--get-hit-frames` in test_combat) | Q4 above is 0 |

## Deviations

None in Q1 to Q4. U items (cannot be called "no deviation" yet):
- D1 (U): FUN_005809d0 (the mode change a get-hit makes) is gated by FUN_0057edd0 (a table on the unit's current mode: +0x10, and +0x60 / +4) and FUN_0057eec0 (state 0x36 refuses it; unit mode 0 and 0x11 refuse; a cast, attack or throw mode may or may not accept a transition to mode 4 depending on a jump table at 0x57f000 and flags). The oracles above hook the mode call, so whether a hit interrupts a cast or an attack on game.exe, and whether d2d does the same in every mode, is not covered. The earlier commit 2312b0e9 ("state 0x36 is never on when struck") covers only the state gate. Differential: buildable. Run FUN_005809d0 natively with a unit in each mode 0..0x11 (and state 0x36 on/off) and the new mode 4; oracle = the unit's mode afterwards; d2d side: `Fight` with the player in each mode struck by a get-hit-sized blow (test_game scenario). Worth doing: it decides "hit recovery during casts".
- D2 (U): negative FHR (items with -FHR) and frozen/stunned units, the disclosed limits of the three proofs.
- D3 (U): the player's drawn frames (Q4 gap).

## Bot findings (described, not changed)

B1 `internal/combat` uses a fixed per-class cast recovery and a post-hit pause (not read here, no file:line): if it does not read the class's FHR breakpoints from data, a kiting plan after a hit is off by the table's steps. Not checked in this task (U); the table above (V run) is the oracle to compare with.

## PR candidate

No PR candidate: no proven deviation. The three prepared drafts are merged.

## Drill recipe (lab binary with lab verbs, docs/forkdbg.md)

1. D1: Sorceress, `debug spawn fallen1 1` next to her, `debug mon set <id> dmg <big> <big>`, `debug cast 36 <x> <y>` (Frost Nova needs no target) and `step` until the hit lands during the cast; `dump stats` raw[8] each tick: does the cast complete and mana drop on the action frame, or is it cut by the get-hit? On game.exe (an instance of the bot): watch the same sequence with the unit mode read (mode 4 after the hit). The two must match.
2. FHR table: equip +FHR items to the breakpoints 0/5/9/14, hit by a spawned Fallen, count ticks to the next idle with `dump player` mode and `step 1`: expect 16, 15, 14, 13 (record 256/8, halved by the 25 fps tick rate relation of the table, V run values above).
