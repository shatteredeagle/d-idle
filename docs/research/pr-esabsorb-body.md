# PR body: Energy Shield absorbs the raw field before the resistance

> status: DRAFT body for an Energy Shield pull request to psi29a/d2decomped (job esprep, 2026-11-04). Local branch `local/es-absorb-order`, head `12b78b60`, 1 commit on upstream main `2d051d77` (after PRs 189, 190, 191, 193, 194 and 195); also on the private fork remote only. NOT posted, nothing pushed to GitHub, no PR opened. Tags: V = run this session on this head (how stated), S = read without a run, U = unverified. An independent verifier has not read it yet.
>
> Title: `Fidelity fix against game.exe 1.14d: Energy Shield absorbs the unresisted field`

## What changes

`Fight::absorb` took the Energy Shield's share from the resisted total of the frame. game.exe raises the shield's absorb event (0xb, `FUN_005ca6c0`) on each damage record before the resistance pass (`FUN_0057c1e0`), so the shield works on the raw field and the resistance, the flat magic reduction and the physical reduction work on what is left.

One commit, one claim: for every blow the shield takes `MulDiv(field, calc1, 100)` of each unresisted field (physical, fire, lightning, cold, magic, in the table order of `0x6e30e0`), at most `MulDiv(mana, 16, calc2)`, drains `MulDiv(taken, calc2, 16)` mana per field, record by record, and the rest goes through the resistance.

Why it matters: a percent-limited shield gives the same result in both orders up to rounding. They diverge when the mana limit binds (mana pays a fixed amount of raw damage in game.exe, of resisted damage in d2d) and at vulnerability or immunity (game.exe pays mana for damage that is then fully resisted; d2d paid nothing at 100 % resistance).

| Code | Change |
|---|---|
| `Taken` | keeps the rolled fire / lightning / cold / magic before the resistance and what each dealt (`element_raw`, `element_dealt`) |
| `StruckBlow` | carries them with the resistance and the flat reduction they went through (`parts`, `mdr`); `Foe::take` fills them |
| `rules::EnergyShield`, `rules::energy_shield_blows` | the shield's arithmetic on a span of blows; a field it took from is put back through the resistance (`resisted_fine`, less `mdr`) or, for physical, through `settle_blow` |
| `Fight::absorb` | calls it where the old block was; what is left of the mana is stored |

## Evidence from game.exe 1.14d

| Address | What it shows |
|---|---|
| `005ca6c0` | wearer has the state; pct = calc1, ratio = calc2 (at least 1); loop over the table at `6e30e0`: per field above 0, `a = MulDiv(field, pct, 100)`, `a = min(a, MulDiv(mana, 16, ratio))`, `field -= a`, `mana -= MulDiv(a, ratio, 16)` floored at 0 (V read) |
| `6e30e0` | 8 rows {record offset, monster-only}: +8, +0x10, +0x1c, +0x24, +0x20 always; +0x38, +0x3c, +0x40 only from a monster. Poison is not in it (V read, bytes read from the image) |
| `0057c1e0` | raises event 0xb on each record BEFORE it reduces it (the loop at `0x57c378`), then runs the resistance and reduction (V read) |
| event 11 | fires once per record at queue time, before the resistance pass (S: an earlier trace under the emulator with the real `FUN_0057c6c0` and a hook on the dispatcher `FUN_005c0c30`; this session's differential relies on the same order) |

## How the claim was measured (V run, the worker host)

A differential of single-element blows (physical, magic, fire, lightning, cold, poison), resistance -100 to 100 (0, 100 and -100 weighted in), raw damage 1 to 1500 points, Energy Shield level 0 (the control) or 1 to 40, mana 0 to 600 points with fractions. 20,000 cases per seed, seeds 1, 2 and 3: 60,000 cases.

- d2d side: the real `rules::monster_blow`, `Foe::take` and `Fight::absorb`, linked from a build of each head and run on each case.
- game side: game.exe's real `FUN_0057c6c0` under the emulator (the resistance pass and the event dispatch are game.exe's code). Event 11 is a hook that does the shield node's arithmetic above on the record field, so **the order and the resistance step are game.exe's code (V), the shield's arithmetic is the formula in a hook (I)**; calc1 and calc2 come from d2d's own Skills.txt evaluation.

The blow is made exact (`sure`, `exact` in `monster_blow`: no hit roll, no draw). Without it a physical range of 0 to 0 still rolls 1 to 2 points in `monster_blow`, which put a stray point into every elemental case on both heads; the "before" numbers below are with the same exact blow.

| Run | Base `2d051d77` | This head |
|---|---|---|
| shield cases without the magic-resistance rows (37,656) | life differs in 16,858 (13,088 by more than 1 point) | **0** |
| mana in whole points, all shield cases (39,918) | differs in 10,752 | **0** |
| mana in 256ths, same cases | differs in 11,186 | **0** |
| control, no shield (10,118 cases, all non-poison) | 560 differ (only the magic-resistance rows) | 560, the same rows |
| non-magic control (9,541 cases) | 0 differ | 0 differ |
| poison (9,964 cases) | shield never touches mana, either side | same |

Per element, shield rows, base to head, life differs: cold 4,366 to 0 (of 8,119), fire 4,329 to 0 (8,045), lightning 4,222 to 0 (7,867), physical 3,941 to 0 (7,829), magic at resistance 0 0 to 0 (5,796).

The first differing case on base, now equal: seed 1 case 9, fire, resistance -100, raw 632, level 40, mana 103,179/256: game.exe takes the shield from the raw 632 and doubles the rest (220,406/256 = 860 points of life, 1/256 of mana left); the base took it from the doubled 1,264 (1,062 points of life, 1/256 of mana left). The head gives 860 and 1/256. Moving the absorb after the resistance in game.exe's own arithmetic changes 24,222 of the 37,656 shield cases (12,942 by more than 1 point), so the order alone is the effect. `test_energy_shield` holds this case.

### The magic-resistance rows (not part of this change)

The 2,262 shield rows with magic damage and a resistance other than 0 are left out of the zero above: they differ the same 1,951 on base and head, in life only (mana equal). d2d applies no percent magic resistance to a monster's magic blow (only the flat reduction); game.exe applies stat 0x25. The control rows without a shield show the same (577 cases, 560 differ). A separate rule, a separate change.

## Tests (V run, the worker host, Release, g++, `-DD2D_WERROR=ON`)

`tests/game/test_energy_shield.cpp` has two parts. Without game data it checks `rules::energy_shield_blows` on typed blows: the seed 1 case 9 numbers (cap binds, vulnerable), calc1 0, mana 0, immune at 100 %, the flat magic reduction after the resistance, a field taken to nothing losing its element mark, mana running out across two records, and damage no field accounts for. With the data it drives the real `monster_blow`, `Foe::take` and `Fight::absorb` over a grid of 2,400 cases (all six damage types, resistances -100, -50, 0, 50, 99, 100, raw 1, 5, 50, 632, 1500, levels 0, 1, 18, 40, mana 0, 3.4, 100 and 600 points) against the arithmetic written out in the test, and the case above (860 points, 1/256). It skips with 77 when the 1.14d data is missing.

## Callers and what else moves

- `Fight::absorb` is the only caller; the other armors (Bone Armor, Cyclone Armor) run before the shield as before and are unchanged.
- Blows built without the new fields (Corpse Explosion, Diablo's cold touch, a Fire Golem's burn, a burn aura) and any damage no blow carries are taken from as they stand, which is what the shield did to everything before (S: source read; the test covers the first case).
- The shield state is not removed at 0 mana (game.exe frees it, `0x5ca8b0`); unchanged here.
- Siblings: only the player carries an Energy Shield; a merc, a pet or a co-op partner has none, so nothing else changes (S).
- `calc1` is still clamped to 0..95 where `FUN_005ca6c0` has no cap; no effect at levels 1 to 30.

## Checks (V, the worker host, head `12b78b60`)

- Author and committer are the numeric noreply identity; the body ends with `Agent: Claude Code`; no `-x` line. Size: 5 files, +339 / -20 (`git diff --shortstat 2d051d77..HEAD`), 3-line body.
- `python3 tools/pr_guard.py 2d051d77 HEAD` and on the commit: 0 FAIL, 0 WARN.
- Release build, `-DD2D_WERROR=ON`, g++: `test_energy_shield`, `test_combat`, `test_rules`, `test_game_rules`, `test_game`, `test_monsters`, `test_skills`, `test_monster_missile` return 0; `test_missile_hit` returns 77 without its case file, by design.
- clang-tidy with the repo's `.clang-tidy` on `fight/states.cpp` and `test_energy_shield.cpp`: 0 findings in them; the only output is `readability-identifier-length` in `components/drlg/outdoor.hpp` and `components/rules/merc.hpp`, files this change does not touch.
- `python3 tools/presubmit.py --quick --base upstream/main` with `D2_BUILD` pointing at a build that has `mpq-cat` and the game data: 0 FAIL, 4 WARN. The WARNs: unit kinds (answered above: only the player has the shield), docs-sync (`docs/research/re/combat.md` gets the sentence), build and gcc skipped by `--quick`.

## Not re-run

- MSVC, clang++ suites, the full parity lane, `test_game` with all data-backed neighbours beyond the list above, a live game, a soak, a determinism hash comparison.
- The differential on the final head after the last amend: the amend changed a test file and a doc, not the code it measures (the 60,000-case run was on the same sources, commit `083b6edd`).
- A real `FUN_005ca6c0` run on the game side: the shield's arithmetic on that side is the formula in a hook (I), so the claim "the arithmetic equals game.exe's" is read (V read of `5ca7b1..5ca837`), not run.

## Limits (U)

- U: absorb states registered later run first in game.exe (`FUN_005c0ad0` puts the event node at the head of the unit's list); d2d keeps armor first, shield second, as before. With an armor up the order is not matched, and a Cyclone Armor cut is not put back through the resistance (marked `ponytail`).
- U: two rows of one element type in one blow are resisted per row (as before) when no shield is up, and as one summed field when it is; no monster has two rows of one type that fire in the same mode (S: not checked against every row).
- U: the shield on the monster-only fields (+0x38 to +0x40, leech and drain) is not modelled; d2d has no such fields on a blow.
- Magic damage has no percent resistance on a monster blow in d2d (above); its field goes through the flat reduction only.
- The differential harness is not part of this PR; the test above is the in-tree check.

Agent: Claude Code
