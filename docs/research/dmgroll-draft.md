# Monster physical damage roll: the fix on a private branch (DMGROLL)

> status: DRAFT, private. Nothing is pushed to the public fork and no PR or comment is open (user decision, 2026-11-02). Branch `local/batch2-monsterdmg` on the private fork remote, head `6f9ded7c`, one commit on upstream main `3e073888` (PR 177 merged). Tags: V = run (how stated), S = secondary, U = unverified.

## Step 1: is it still open on upstream main? (V: read of `3e073888` and a fetch of PR 178)

| Question | Answer |
|---|---|
| `monster_blow` (`components/rules/combat.hpp`, the file moved from `game/combat.hpp` named in SEEDDIST) still rolls whole points inclusive? | Yes: `rng.range(attacker.a1_min, attacker.a1_max)` (a2 for the second attack), then `phys * (100 - dr_pct) / 100 - dr_flat`, whole points. |
| A `// ponytail:` note on it? | No (V: read of the function and its comment block on `3e073888`). |
| Does PR 177's melee fix already cover the monster side? | No. It carries the PLAYER's blow in 256ths (`Blow::frac`, `player_blow`); `Taken` (a monster hit on a player) has no fraction field on main. |
| Does PR 178 touch it? | No. It changes the store, panels, `fight/strike.cpp`, `rules.hpp` (store), `skills.hpp`, `ai.hpp` (a comment and `ar_pct`) and `fight.hpp` (comments). A throwaway `git merge-tree` of the branch with the PR head (`2597875`) is clean (V). |

So the deviation (seeddist-trace.md D3, F13 of the ledger) is open and the fix was built.

## What the commit does

game.exe: `FUN_0057b420` rolls `(min << 8) + rand((max - min) << 8)` on the attacker's seed, the top never rolled; the stats it reads are the monster's mode damage (stats 0x15 / 0x16), and `FUN_0057c1e0` reduces that record in 256ths (the life stat takes it as is, `0057c852`). The differential here adds a detail the older traces did not state (V, section "Differential"): min is taken at least 1 and max at least min + 1, so an empty, inverted or 0 range still rolls and draws once (`0..0` rolls 1..2, `5..4` rolls 5..6).

d2d, six files, +69 -12:

- `rules::monster_blow` rolls `low * 256 + rng((high - low) * 256)` with that clamp (not for a missile, whose `FUN_005a89a0` bounds are taken as they stand: U), applies the percent and flat reduction to the fine value (the order is d2d's, U), and returns `Taken::damage` (whole points), `Taken::frac` (0..255) and `StruckBlow::frac`.
- `Foe::take` sums the fractions in `Foe::frac`; the player loses `(damage << 8) + frac`, unless a state absorbed part of the hit (Bone Armor, Cyclone Armor, Energy Shield: the fraction is dropped, as `Fight::land` already does for a monster). Pets get it through `pet_hurt(.., fraction)` into `hurt`, the merc through its `life_frac`.
- `tests/test_combat.cpp`: the roll equals `256 + (seed.low & 255)` for 2,000 seeds, never reaches 512, mean 1.5 within 0.05, one draw on the seed, the clamp rows `3..3`, `5..4`, `0..0`, the flat and percent reduction on the fine value.

Ponytail notes in the code: the order percent then flat; the dropped fraction when a state absorbs; the merc's death test stays `hit_points <= 0`.

## Differential on the real `monster_blow` (V, the worker host, 2026-11-02)

Game side `tools/emu/mdmg.py game`: game.exe 1.14d's `FUN_0057b420` under the emulator (the `sd_game.py` setup: `FUN_006538a0` gives each class's A1 range, the stat getters hooked as `swing()` hooks them), on a unit whose seed is set to each case's state (a fixed generator, the high word 666 or random). The d2d side `tools/emu/mdmg_d2d.cpp` calls the real `rules::monster_blow` (a sure swing, so the first draw is the roll) compiled from the headers of each tree: the base `3e073888` and the head. 20,000 cases per row, the answer is the roll in 256ths AND the seed after. "Differ" counts rolls not equal in 256ths; the seed after differs in 0 cases for the 8 classes on both builds.

| class | A1 min..max | cases | game mean / variance (hp) | base: differ / mean / variance / ratio | head: differ / mean / variance / ratio |
|---|---|---|---|---|---|
| Fallen | 1..2 | 20000 | 1.4979 / 0.0830 | 19925 / 1.5049 / 0.2500 / 3.01 | 0 / 1.4979 / 0.0830 / 1.00 |
| Zombie | 1..3 | 20000 | 1.9999 / 0.3331 | 19972 / 1.9937 / 0.6707 / 2.01 | 0 / 1.9999 / 0.3331 / 1.00 |
| Skeleton | 1..3 | 20000 | 1.9961 / 0.3338 | 19980 / 1.9858 / 0.6712 / 2.01 | 0 / 1.9961 / 0.3338 / 1.00 |
| Quill Rat | 1..2 | 20000 | 1.4959 / 0.0826 | 19904 / 1.5003 / 0.2500 / 3.03 | 0 / 1.4959 / 0.0826 / 1.00 |
| Rogue (Dark Hunter) | 1..3 | 20000 | 1.9976 / 0.3338 | 19973 / 2.0011 / 0.6682 / 2.00 | 0 / 1.9976 / 0.3338 / 1.00 |
| Gargantuan Beast | 2..3 | 20000 | 2.4974 / 0.0830 | 19930 / 2.5022 / 0.2500 / 3.01 | 0 / 2.4974 / 0.0830 / 1.00 |
| Blood Raven | 4..6 | 20000 | 4.9993 / 0.3326 | 19984 / 5.0014 / 0.6671 / 2.01 | 0 / 4.9993 / 0.3326 / 1.00 |
| Andariel | 6..19 | 20000 | 12.4833 / 14.1186 | 19994 / 12.4815 / 16.1936 / 1.15 | 0 / 12.4833 / 14.1186 / 1.00 |
| edge 0..0 | 0..0 | 20000 | 1.4991 / 0.0835 | 20000 / 0.0000 / 0.0000 / 0.00 | 0 / 1.4991 / 0.0835 / 1.00 |
| edge 0..5 | 0..5 | 20000 | 2.9967 / 1.3416 | 19987 / 2.4886 / 2.9206 / 2.18 | 0 / 2.9967 / 1.3416 / 1.00 |
| edge 1..1 | 1..1 | 20000 | 1.4969 / 0.0827 | 19938 / 1.0000 / 0.0000 / 0.00 | 0 / 1.4969 / 0.0827 / 1.00 |
| edge 3..3 | 3..3 | 20000 | 3.4976 / 0.0833 | 19921 / 3.0000 / 0.0000 / 0.00 | 0 / 3.4976 / 0.0833 / 1.00 |
| edge 5..4 | 5..4 | 20000 | 5.4976 / 0.0843 | 19919 / 5.0000 / 0.0000 / 0.00 | 0 / 5.4976 / 0.0843 / 1.00 |

- The 8 Act 1 classes of SEEDDIST: base differs in 19,904 to 19,994 of 20,000 (the rest are cases where both land on the same value), head in 0 of 160,000; the head's variance ratio against game.exe is 1.00 in every class (base 1.15 to 3.03), the means agree within the sampling error (the base means were already within 0.05).
- The edge rows (not part of the 8): the base draws nothing for an empty or inverted range, so its seed after differs in 20,000 of 20,000 cases and every later draw of that monster shifts; the head matches in value and seed.
- Limits: the reduction (percent, flat) is covered by unit tests only, not by the emulator (the game's order is U); the missile path of `monster_blow` (`FUN_005a89a0`) shares the roll but its edge clamp is not measured (U); the pet and merc carry is compiled and exercised by the existing suites, not by a new game-level test.

Reproduce (the worker host): `tools/emu/mdmg_build.sh TREE LABEL` for each tree, `tools/emu/run-on-worker.sh mdmg.py game 20000`, `tools/emu/run-on-worker.sh mdmg_d2d.sh base head`, `tools/emu/run-on-worker.sh mdmg.py compare base head`.

## Consequences (I, reasoned, not measured in a soak)

- A monster hit now varies as in game.exe: a 1..2 Fallen deals 1.00..1.99 instead of 1 or 2 (mean unchanged, variance a third of d2d's before). Expected damage taken per fight is the same; the per-hit spread falls, so a bot that sizes potion thresholds on the hit's maximum sees a lower bound (SEEDDIST B2: the largest hit is strictly below max).
- A monster with `min == max` or `min == 0` now rolls up to one point more than its table says (the game's clamp): the mean rises by 0.5 hp per hit for those rows (e.g. `3..3` deals 3.0..3.99). Which rows are affected is U (not counted over MonStats here).
- Determinism: the roll draws the same number of times except for an empty range, which now draws once; recorded hashes change after the first monster blow.

## Checks (V unless stated)

- Suites on the head (Release build with the 1.14d game data, -j1, nice 19): `test_rules`, `test_game_rules`, `test_monsters`, `test_monster_thinks`, `test_monster_ai`, `test_combat`, `test_game`, `test_sk_ground`, plus `test_monster_missile` and `test_missile_blow_poison`: rc 0 each.
- g++ and clang++ with `-Wall -Wextra -Wpedantic -Wno-missing-field-initializers -Werror` plus `-Wshadow -Wrange-loop-construct -Wconversion`, `-fsyntax-only` on the three changed translation units: 0 diagnostics on an added line (the first test version had two `-Wshadow` hits and one `-Wrange-loop-construct`, fixed before the commit; clang prints `-Wsign-conversion` notes in untouched headers). clang-tidy on the three: 0 diagnostics in the changed files. Direct includes of the test: `<algorithm>`, `<cmath>`, `<utility>`, `<cstdint>` present. MSVC /W4 and `windows.h` names: read only (every `std::max` / `std::min` qualified, no `min` / `max` / `near` / `far` identifier added) (S).
- `tools/pr_guard.py upstream/main`: 0 FAIL, 0 WARN. `tools/presubmit.py --quick`: 0 FAIL, 4 WARN (siblings: merc / pet / monster paths are covered, a missile reaches the same function and rolls the same shape; hit paths: melee is the swing, skills the same call; docs-sync; build skipped by `--quick`).
- Commit: author and committer the numeric noreply identity, subject plus three body lines, last line `Agent: Claude Code`, no co-author or AI line. Size: 6 files, 81 changed lines, 1 commit.
- Not run: a live game, a soak or a determinism hash comparison; MSVC; the other branches of the private fork were not touched.

## Commit text

```
game: a monster's physical damage is rolled in 256ths with the top excluded

FUN_0057b420 rolls min << 8 + rand((max - min) << 8) on the attacker's seed (min at least 1, max at least min + 1); d2d rolled whole points inclusive.
monster_blow keeps the 256ths in Taken::frac, which the player's life, pets and the merc carry; the reduction applies to the fine value.
test_combat checks the roll against the seed, the clamp and the reduction.

Agent: Claude Code
```

## For the independent check

Re-run the differential with a new seed (change the generator constant in `mdmg.py`), read `monster_blow` and `Foe` / `tick.cpp` for the carry (especially the absorbed-hit rule and the merc), and judge whether the clamp for a `min == max` row is wanted before the commit is carried anywhere.
