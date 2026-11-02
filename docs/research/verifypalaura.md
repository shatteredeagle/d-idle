# VERIFYPALAURA: independent re-check of `palaura-trace.md`

> status: card verifypalaura, 2026-11-02, offline. Independent verifier (Sonnet). Scope was limited by a points cap: two claims re-run with my own script and seeds (`tools/emu/vp_check.py`); the rest is marked as not re-run. Marks: VERIFIED (counts), HOLDS WITH LIMITS, NOT VERIFIED, WRONG.

## Method

`vp_check.py` reuses only the world builder `palaura_area.World` (real Blood Moor game, real player / monster units from game.exe's `FUN_00555230`) and `palaura_apply.prep` (calc hooks). The model, the placement and the seeds are mine: world seed 9 / 11 (the author used 5), caster seed set by me, unit placement different. The oracle for both runs is game.exe's own code executed natively under the emulator (`FUN_005cf3a0` do 66, `FUN_0056b7e0` area search), not a re-derivation of d2d's formula.

## Results

| Claim (palaura-trace) | My run | Mark |
|---|---|---|
| Enemy roll is `min .. max-1` (D4; Holy Fire 1 table 256 .. 768) | `vp_check.py roll 4000`: 4,000 pulses, observed min 256, max 767, 512 distinct values, 768 never, 0 outside `[lo, hi)` (10 pulses at exactly the minimum, so the bottom is included) | **VERIFIED** (game side, V run). Independent seed; the d2d side `min .. max` stays S (source reading) and matches SKMISS D1 |
| Area search is an inclusive euclidean subtile disc `d2 <= r2` (Q1, D6 game side) | `vp_check.py disc 7`: 237 units, 237 agree, 2 units exactly on the circle both found; `disc 16`: 349 units, 349 agree, 1 on the circle found | **VERIFIED** for radii 7 and 16 inside the near rooms. The author's 4 off-model units (radius 43, outside the near-room list) I did not reproduce: **HOLDS WITH LIMITS** for large radii |
| Filter bit table, 0x12003 / 0xa783 / 0x4a283 pass lists | not re-run | NOT VERIFIED (independently) |
| Line of sight bit 0x200 mask | not re-run; the author already calls the mask a best fit (1,183 of 1,200) | NOT VERIFIED; the mask stays I |
| Prayer: pays mana and sets state 85 only when a life rose; no heal below cost | not re-run | NOT VERIFIED (independently) |
| State 85 regen effect (`FUN_005806f0`) | not re-run | NOT VERIFIED (independently) |
| Resist formula `roll * (100 - eff) // 100`, pierce | not re-run | NOT VERIFIED (independently) |
| Holy Freeze skips prime evil, Andariel | not re-run | NOT VERIFIED (independently) |

## "No deviation" and oracle check

- D8 (pierce equal) and "not deviations" (resist arithmetic, calcs): the author's oracle is game.exe code for the game side, but the d2d side is source reading and no number was compared with a real d2d build (the trace says so in its section 9). So these are "equal by reading", not a differential. Agreed, not stronger than that.
- D4: the game side is now double-confirmed. The only d2d-proven part remains SKMISS D1 on `missile_blow`; the aura call site is S.
- The trace's "no PR candidate" is correct under the rule (no existing d2d differential runs `Fight::aura_pulse`).

## Open for the next pass

Re-run with own seeds the filter table (`palaura_area.py filter` equivalents), Prayer cases (four rows of trace section 5) and the regeneration cases, then the hit resist run with pierce. Nothing found WRONG so far.

## Shared-file lines for the merger

In `openspec/changes/verifypalaura/merge-lines.txt`.
