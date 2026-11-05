# PR body: a monster blow takes the flat reduction before the resistance

> status: DRAFT body for a pull request to psi29a/d2decomped (job nsprep, 2026-11-04; retargeted 2026-11-05). Fork branch `local/noshield-reduction-order-r2`, ONE commit on upstream main `4b402827` (the merge of #198, Energy Shield, merged upstream 2026-11-05 05:44 EST as a merge commit, so `3b8bb7df` and `8f9dd904` are in its history). It builds on #198 (merged): it uses #198's `reduce_fine_signed`, `StruckBlow::parts` and `Taken::element_raw`. The earlier branch `local/noshield-reduction-order` (head `fe434d09`) was the same change on `3b8bb7df`; the r2 commit is its cherry-pick, applied without a conflict. Private fork remote only. NOT posted, nothing pushed to GitHub, no PR opened. Tags: V = run (how stated), S = read without a run, U = unverified. The follow-up named in `docs/research/pr-esabsorb-body.md` (limit K1) and `docs/research/esverify/verdict2.md`. The counts below were measured on `fe434d09`; the r2 retarget is the same diff on a newer base and has NOT been re-measured with the native differential. An independent verification of this branch has not been done.
>
> Title: `Fidelity fix against game.exe 1.14d: the flat reduction comes off a monster blow before the resistance`

## What changes

`rules::monster_blow` resisted each element and then took the magic damage reduction (mdr) off, clamping each part at 0. game.exe's resistance pass (`FUN_0057bf80`, per field of the damage record) does it the other way round: for a field above 0 the flat reduction comes off first (physical: stat 34 then the percent; fire, lightning, cold, magic: the mdr), then the resistance on what is still above 0, and the result is stored even under 0. `FUN_0057c1e0` sums the stored fields into the record total (+0x4c) and the life update ignores a total that is not above 0 (`0057c852`): a field left under the reduction takes damage off the other fields of the record, and the total is clamped once.

Energy Shield (#198, merged) already does this for the fields the shield takes from; this commit gives a blow with no shield (and the fields a shield leaves alone) the same arithmetic. One claim, one function.

Example (V run, game.exe's own code natively, no shield): 100 fire, 50 % resistance, mdr 10 deals **45** on game.exe, 40 on d2d before this commit; at 75 % 22.5 against 15; at -100 % 180 against 190. Fire 5 plus cold 100, mdr 6: game.exe 93 (fire is -1 and is summed), d2d 94. Physical 10 under a flat 15 plus fire 50: 45 against 50.

| Code | Change |
|---|---|
| `rules::monster_blow` | the physical field is `reduce_fine_signed`; the element rolls of a type are one field (as the record has one), reduced by the mdr, then `resisted_fine`, in 256ths; the fields are summed signed and `damage` / `frac` are the total clamped once. Rolls, hit, block and chill draws are unchanged |
| `Taken` | gains `element_fine` (each field after the reduction, under 0 kept) and `sum` (the total before the clamp); `damage` and `frac` now hold the clamped total, not the physical's fraction |
| `StruckBlow` | gains `sum` and `Part::fine`; `physical_now`, `blow_total`, `store_total` keep the signed total through Bone Armor (`settle_blow`), Cyclone Armor and the shield. A blow built by hand or by another path (no `sum`) is taken as before |
| `rules::energy_shield_blows` | reads the signed fields (`part_fine`, `physical_now`) instead of the clamped ones; its arithmetic is unchanged |
| `Foe::take` | copies `sum` and the fields into the blow |

The frame's element total is now the sum of 256ths rather than of whole-point parts; that is the second thing `FUN_0057c1e0` does (it sums the stored fields) and it is the reason the remaining whole-point differences of the control frames (d2d resisted each element in whole points) go too.

## Evidence from game.exe 1.14d

| Address | What it shows |
|---|---|
| `0057bf80` (to `0x57c04d`) | field at or under 0 stored as 0; else the reduction comes off, then only if the rest is above 0 and the resistance is not 0, `MulDiv(rest, 100 - min(res, 100), 100)`; the result is stored even under 0 (V read; V run natively in the esverify job, `esv2_handgame.py`) |
| `0057c1e0` | sums the stored fields into +0x4c; `0057c852`: a total not above 0 is ignored (V read) |
| native numbers | 45 / 22.5 / 180 / 93 / 45 above (V run, `docs/research/esverify/verdict2.md`, "The no-shield finding") |

## Counts before and after (V run, the worker host)

The esverify harness (`tools/emu/esv_*`, `esv2_*`; the game side is game.exe's own `FUN_0057c6c0` / `FUN_0057c1e0` / `FUN_0057bf80` and, with a shield, `FUN_005ca6c0`) with its two probes: "before" is `3b8bb7df` (#198, now on upstream main), "after" is `fe434d09`, same cases, same game output. Seeds 31, 32 and 33 of the first generator at 20,000 frames each (new seeds; every earlier job used other ones), the second generator's edge grid (all six damage types, resistance -100 / 0 / 50 / 75 / 100, rolls 1..632, flat and mdr in (0,0) (15,0) (0,10) (15,10) (3,6) (40,40), levels 1 / 40 / 99, mana 0 / 1/256 / 300 points, two-field records, poison) plus 20,000 random frames, and the 17 hand cases. Life is compared in whole points and in 256ths; mana the same.

| Frames | Cases | Before: life differs (pt / 256ths) | After |
|---|---|---|---|
| seeds 31 + 32 + 33, all | 60,000 | 16,472 / 26,752 | **0 / 0** |
| of them no shield | 8,997 | 2,304 / 4,764 | **0 / 0** |
| of them no shield, flat or mdr non-zero | 3,223 | 1,191 / 1,723 | **0 / 0** |
| of them with a shield | 51,003 | 14,168 / 21,988 | **0 / 0** |
| edge grid + 20,000 random, all | 26,588 | 6,677 / 8,255 | **0 / 0** |
| of them no shield | 1,933 | 649 / 844 | **0 / 0** |
| of them no shield, flat or mdr non-zero | 1,449 | 593 / 681 (41 %) | **0 / 0** |
| of them with a shield (24,655, 4,418 with poison: mana only) | 24,655 | 6,028 / 7,411 | **0 / 0** |
| hand cases | 17 | 10 | **0** |

Mana differs on none of the shield cases, before or after (also the poison ones: game.exe takes no mana for poison). On the 11,006 shield cases of the grid whose control twin is exact, #198 left 7 (9 in 256ths) behind, all frames where the mana ran out and the untouched fields followed the old order (K1/K2 of the second verification); this change closes them. No case is left over: there is no difference to list with parameters.

This is a re-measurement on the same machinery, not a new claim about shield arithmetic: with a shield up the result is the same as #198's wherever #198 was right (`test_energy_shield`, 2,400 cases, unchanged and passing).

## Tests (V run, the worker host, Release, g++, `-DD2D_WERROR=ON`, head `ca7872ab` on upstream main `4b402827`)

`tests/test_combat.cpp` gets a block that needs no game data: 100 fire at 50 % with mdr 10 is 45; 75 % is 22.5; -100 % is 180; fire 5 plus cold 100 with mdr 6 is 93 with the fire field kept at -1 and `sum` 93 x 256; the same with mdr 0 is 105 (zero-flat control); fields all under the reduction give 0 with `sum` -5 x 256; physical 10 under a flat 15 plus fire 50 is 45, and 60 without the flat.

Two existing assertions encoded the old order and game.exe contradicts them (V run: 40 x 25 % less 2 versus (40 - 2) x 25 %), so they changed: the "monster blows" block of `test_combat.cpp` (`taken.damage == 76 + 8` and `taken.elemental == 8`: now the physical 76.x plus 9.5, `elemental == 9`) and the sure-hit loop of the same file (`damage == 76 + 8`: now 85 or 86, the physical's 256ths decide). `tests/game/test_energy_shield.cpp` needed no change. `tests/game/test_game.cpp` (the Bone Armor cases) needed none either; `settle_blow` keeps the old arithmetic for a blow without `sum`.

`test_combat`, `test_rules`, `test_game_rules`, `test_game`, `test_monsters`, `test_skills`, `test_monster_missile` and `test_energy_shield` return 0 on the retargeted head (V run) except `test_game` and `test_energy_shield`, which exit 77 (SKIP: no game data on the worker host at the time of the retarget; they passed with data on `fe434d09`; rerun with data before opening).

## Callers and what else moves

- `monster_blow` is the one place a hit on a unit is built (the monster melee, missiles, Charge, Whirlwind, Leap, Corpse Explosion, Diablo's cold touch); the defender may be the player, the merc, a pet or a co-op partner, and all of them get the new order. The Energy Shield exists only on the player (S).
- Not through `monster_blow`, so unchanged and still in the old order (U, not measured): the monster aura's element damage (`fight/tick.cpp`, resisted then mdr, clamped) and a burning missile (`ai.hpp`, the same). Their rows are fire / lightning / cold only; game.exe takes them through the same `FUN_0057bf80`. A follow-up if the owner wants it.
- Two rows of one element type in one blow are now one field (resisted once, as the record has one field a type); before they were resisted per row. No Monsters.txt row has two of one type that fire in the same mode (U: not checked against every row).
- The monster-only fields (+0x38, +0x3c, +0x40) are not modelled, as in #198.

## Checks (V, the worker host, head `ca7872ab`)

- Author and committer are the numeric noreply identity; the body is 3 lines and ends with `Agent: Claude Code`; no `-x` line.
- `python3 tools/pr_guard.py 4b402827 HEAD`: 0 FAIL, 0 WARN.
- `python3 tools/presubmit.py --quick --base 3b8bb7df` (run on `fe434d09`, not re-run on r2): 0 FAIL, 4 WARN: unit kinds (answered above), docs-sync (`docs/research/re/combat.md` is updated in the commit), build and gcc skipped by `--quick` (the build was done by hand, above).
- clang-tidy with the repo's `.clang-tidy` on `tests/test_combat.cpp` (no finding) and `components/game/fight/states.cpp` (no finding in a touched file; the pre-existing `readability-identifier-length` errors in untouched headers remain, as in the second verification).

## Not re-run

- MSVC, clang++ suites, the full parity lane, a live game, a soak, a determinism hash comparison; `test_game` neighbours beyond the list above.
- The monster-only fields (C4), a physical field above 4,096 points with a percent reduction (C2: pre-existing, the grid has none by design), magic damage with a percent resistance (the harness has no magic resistance column; d2d applies none to a monster's magic blow, `FUN_0057bf80` row `stat 0x25`), the state's removal at 0 mana (C3).
- Armor states (Bone Armor, Cyclone Armor) together with a shield or with negative fields: the signed total goes through them (`test_combat` Bone Armor cases still pass) but the differential does not put an armor up.
- An independent verification of this branch.

## Limits

- U: `Taken::damage` and `frac` changed meaning (the clamped total, not the physical's fraction). Every reader in the tree was read (`Foe::take`, `Fight::absorb`, `rules::absorb_blows`, `settle_blow`, `energy_shield_blows`); the callers that build a `Taken` or a blow by hand keep the old meaning because `sum` is unset.
- The differential harness is not part of this PR (it lives in the lab repo); only `test_combat` is repeatable by upstream, so the counts above are cited, not proven, for the maintainer.

Agent: Claude Code
