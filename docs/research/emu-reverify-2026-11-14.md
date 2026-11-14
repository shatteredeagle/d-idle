# Emulator image re-verification, 2026-11-14

> status: finished 2026-11-14 (TASK-406, Sonnet). Offline, worker host only, no game, nothing posted. Image: `docs/emu-image.md`. Tags V (run or read here), S (secondary), U (unverified).

## Oracles that reproduce the recorded numbers (V run)

| Oracle | Recorded | Reproduced |
|---|---|---|
| Warp arrival differential, real Act 1 windows (`tools/emu/d12_run.sh levels SEED`; `docs/probes/fid/fid-warp-arrival.md`: d2d `nearest_free` vs game.exe FUN_0064dea0, parent d2d build) | seed 5: 26 of 52 differ; seed 82: 26; seed 7: 25; `warp_spot` 0 | seed 5: 52 arrivals, 28 moved, 26 differ, `warp_spot` 0; seed 82: 26 / 0; seed 7: 25 / 0 |
| XP level-gap grid (`xp_award.py grid` against `xpg_model.py check`; `docs/research/xp-gold-trace.md`) | 119,790 lines, 0 differ | 119,790 lines, 0 differ (this is also `emu_setup.sh verify`) |

(The `C vs game.exe` block of `d12_run.sh levels` fails for lack of a `C.tsv`: an old script artefact, not the image.)

## SHRINE (decomp17, `docs/research/shrine-trace.md`)

- `shr_dump.sh` ran on the d2d side (3 seeds, 3,706 lines, 10 s); `shr_count.py` gave the per-level counts and odds (same shape as the trace); `shr_find.py` now runs (V run): the string `shrines` is referenced at 0x6412b7 and 0x6413aa, `objgroup` at 0x6411f5, all in the table loader near 0x6411xx (so the Shrines.txt loader is there: V).
- FUN_0054f9d0, the row roll, read (V read, disassembly in the image): the roll switches on the object's Parm0 (`[obj+0x178]`): 0 = generic branch with up to 8 tries (`ebp-8` = 8): row = LCG (`x*0x6ac690c5 + carry`, game's seed pair) mod (row count - 1) + 1, re-rolled while the level id is below the row's `MinLevel` field (`[row+0xb4]`), the last draw kept after 8 failures; Parm0 1 and 2 return fixed types 2 and 3; any other Parm0 draws `r mod 10`: 0 gives type 1, anything else type 4, and calls FUN_0054f770 with the type. The result is then remapped 5 to 3, 4 to 2, 0x10 to 0x12 before it is stored. So the "8-try re-roll with a level gate" part of D4 is V read now, but the per-Parm0 grouping is NOT what `shr_count.py` models (it keys on d2d's classes): the S odds in the trace stay S until FUN_0054f770 and the row table are read. Upgrade: D4 S to "8-try re-roll V read; class split U".
- Not run: D1, D2, D5, D6 (need fake object structs; no quick harness).

## MERC2 (decomp16, `docs/research/merc2-trace.md`)

- FUN_00579c00 re-read in the image (V read): hirer test (cmp against 0xfc / 0x16f / 0x203 and the two lower ids), merc lookup FUN_00574ec0, price FUN_006637b0, payment FUN_00576d90, then FUN_00624690, FUN_00625d10 (stat 6 from the max-life accessor, as claimed for D4), FUN_00627260, placement FUN_00579aa0; error codes 0x96c (not a hirer), 0x973 (no merc), 0x994 (gold), 0x99d/0x99e/0x95d (messages). Matches section "Handler" of the trace; the order of the revive steps is now V read. D4 (life after revive equals max-life accessor): the call sequence is V read, the value on a hooked merc is not run (U).
- R1 to R8 are live drills, not run.

## Not done

Zeal gap grid (600 of 600 differ on the parent d2d) needs a scratch d2d build (`zd_build.sh`); the warp oracle was the cheaper of the two named.
