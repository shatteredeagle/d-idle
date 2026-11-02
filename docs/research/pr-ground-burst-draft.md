# PR draft: standing missiles hit by their footprint, and a burst rolls once (ground and burst pipeline)

> status: READY FOR THE INDEPENDENT VERIFIER, held (nothing pushed to the public fork, opened or posted). Local bundle: branch `local/ground-burst-v1` in the private fork on the worker host, three commits on upstream main 48908176 (the merge of #174): **b04d6d0f** tests and oracles, **42b1c9e0** burst roll, **73ca62cf** standing footprint. Evidence: [sk-ground2-trace.md](sk-ground2-trace.md), [sor-rest-trace.md](sor-rest-trace.md), [sk-ground-trace.md](sk-ground-trace.md); candidate list: [pr-candidates-20261102.md](pr-candidates-20261102.md) (C4). Card prbuildground (2026-11-02), run offline on the worker host, no game.

Tags: **V** run or read in this card (how is said); **S** from an earlier trace of ours; **U** not verified.

## Inventory: what is proven and what is only read

"Proven" here means a counted differential in which game.exe's own code and the real d2d build both ran (V run), and a d2d fix small enough to review.

| id | deviation | evidence | cases | fix | in the bundle |
|---|---|---|---|---|---|
| D1 shard offsets | -6..6 against -6..5 | sk-ground2-trace.md section 5 | 14,926 / 14,957 of 100,000 shards | merged (#163) | no (merged) |
| D4 standing hit area | a shard (Size 2) hits a size 1 / 2 / 3 unit on 5 / 13 / 21 subtiles in game.exe, 21 in d2d; a Fire Wall flame (Size 1) 1 / 5 / 9 | sk-ground2-trace.md Q23, section 5 (V run both sides, 225 offsets x 3 classes) | re-run here at scale: see Proof | yes, one helper | yes |
| burst roll (Fire Ball, Glacial Spike) | game.exe rolls the record once per burst, d2d once per monster | sor-rest-trace.md Q5 (V run both sides, 400 trials, 0 of 400 equal on d2d) | re-run here at scale: see Proof | yes, an optional record in `missile_blow` | yes |
| D5 Meteor roll | same shape as the burst roll | skground-trace.md (V read, game side run; d2d side by reading) | none on the d2d side | would be one more argument at the call site | no: no d2d run |
| D2 shard seed, D3 shard on a blocked spot | determinism, blocked spots | sk-ground-trace.md (V read, V run game side) | none on the d2d side | not small | no |
| D6 Meteor blast shape | disc on integer subtiles against d2d's float hypot | skground-trace.md (V run game; d2d at subtile centres equal, off-centre U) | rim only | needs a statement of the float to subtile mapping | no |
| D7 skill delay | none for the player in d2d | sk-ground-trace.md (V read) | none | stateful | no |
| D9 Fire Wall burns per missile, D10 maker one tick short | one hit per overlapping flame; the last drop missing | sk-ground2-trace.md (V run game; D10 both sides, 80 cases) | 80 | not small, small effect | no |
| Hydra heads one subtile apart | read only (V read) | sk-ground2-trace.md Q24 | none | not built | no |

Only the two rows marked yes are proven and fixable; everything else is read (V read, S, U) or lacks a d2d-side run, and stays out.

## What changes

1. **Burst roll** (42b1c9e0). FUN_005a9a70 (hit function 1, Fire Ball) and FUN_005aa8b0 (13, Glacial Spike) call FUN_005a8c70 once and hand a copy of that one record to every unit found; each unit resists its own copy. `rules::MissileRoll` holds the physical and the element roll; `missile_blow` takes an optional pointer to one that the first blow fills and the others reuse; `Fight::area` passes one for hit functions 1 and 13 only. Other hit functions keep a roll per monster.
2. **Standing footprint** (73ca62cf). A standing missile strikes from its own subtile each frame (FUN_005ae1f0, FUN_00641cb0): `standing_struck` uses `rules::missile_touches`, as a moving missile already does, for rows with a Size of 1 to 3, in the two places that tested half a cell (`Fight::missile_tick` do 5 and the hit loop in tick.cpp). A row with no Size keeps the half cell it had.
3. **Tests and oracles** (b04d6d0f). `test_sk_ground` with no argument checks 5 / 13 / 21 and 1 / 5 / 9 subtiles and the equal losses of one burst (it fails on this commit and passes after the other two); `--hit` and `--burst-roll` print the d2d side for `tools/emu/sk_ground_hit.py` and `sk_ground_roll.py`, which run game.exe's functions natively under the emulator and compare. A unit check of the shared roll is in `tests/test_combat.cpp`.

One line in docs/research/re/skills.md each for the roll and the footprint.

## Proof (V run, the worker host, Release builds, upstream main 48908176)

game.exe natively (the emulator) against the real d2d build, base = the tests commit b04d6d0f, head = 73ca62cf.

| part | cases | base (differ) | head (differ) |
|---|---|---|---|
| standing Blizzard shard, Size 2, every offset -7..7, every MonStats class both sides can run | HITB | HITB_BASE | HITB_HEAD |
| standing Fire Wall flame, Size 1, same | HITF | HITF_BASE | HITF_HEAD |
| Fire Ball burst, three monsters, one record | ROLLF | ROLLF_BASE | ROLLF_HEAD |
| Glacial Spike burst, three monsters, one record | ROLLG | ROLLG_BASE | ROLLG_HEAD |

A hit-area case is one (class, offset): game.exe's hit or no hit against d2d's over 9 frames. 658 classes ran on both sides; the comparison skips 69 (blizzard row) / 62 (firewall row) that d2d never hits (hidden or untargetable rows), 33 that walk off their subtile in d2d whatever the test says, and 6 that game.exe never hits. A roll case is one burst: game.exe's three copied records byte-equal (every trial), d2d's three monsters (one class, same resistance) losing the same life.

Suites on the head (Release, the game's data): test_rules, test_game_rules, test_monsters, test_monster_thinks, test_monster_ai, test_combat, test_game, test_sk_ground: rc 0 each.

## Checks (V)

- pr_guard 0 FAIL / 0 WARN on the range and on each commit; `tools/presubmit.py --quick`: 0 FAIL (the attribution check passes on the three commits; the remaining WARNs are answered below).
- g++ 16 and clang++ 23 with the CI flags (-Wall -Wextra -Wpedantic -Wno-missing-field-initializers -Werror) plus -Wshadow -Wrange-loop-construct -Wconversion over missiles.cpp, tick.cpp, test_sk_ground.cpp and test_combat.cpp: g++ clean; clang++ -Wconversion reports only what upstream main already has in headers (identical diagnostics on main and on the head).
- clang-tidy (the repo's identifier-length and include-cleaner) over the four .cpp files: nothing in a changed file (what it reports is in headers this change does not touch).
- ruff 0.8.6 on the two Python tools: clean.
- MSVC /W4 and windows.h macros: not built; by reading the diff there is no `min`, `max`, `near`, `far` or `small` identifier, no narrowing without a cast, and the new locals do not shadow.
- Merge: the bundle rebased onto main 48908176 with no conflict; #174 is in this base.

## Review notes (answers to the presubmit WARNs)

- siblings: only monsters are touched. A monster's missile on the player or a merc (ai.hpp, the foe loop) keeps its own 0.4-cell test for standing missiles; not proven here (U).
- doc-claims: the roll and footprint lines in skills.md say "each unit found" and name the two hit functions and the Size rows measured.

## Limits

- The differential is the SET of hit offsets and the equality of the burst's losses, not damage values or the rng draw order: the shared roll takes the first monster's draw, so later rolls shift against main (a determinism change, expected).
- Rows with no Size keep the half cell (explosions, trails): game.exe's rule for them is not measured here (U).
- Monsters off the subtile centre are not run: d2d's float position against game.exe's integer words is the same open question as D6 (U).
- Not run: det state hashes on base and head builds (needs game slots; checklist below), an MSVC build, a live game.

Det checklist (ready to run when game slots are free): build base and head, run the repo's determinism scenarios plus a Sorceress casting Blizzard, Fire Ball and Glacial Spike at a pack and Fire Wall under a size 1 and a size 2 monster on both, and expect the state hashes to differ only after the first multi-monster Fire Ball or Glacial Spike burst (the shared roll) and for monsters at the footprint edge of a shard or flame.

## How to review

1. `components/rules/combat.hpp`: `MissileRoll` and the two `shared` lines in `missile_blow`.
2. `components/game/fight/missiles.cpp`: `area` (one roll for hit functions 1 and 13), the do 5 loop.
3. `components/game/ai.hpp`: `standing_struck`, next to `missile_struck`; `tick.cpp` calls it.
4. Re-run: `test_sk_ground`, `test_sk_ground --hit blizzard1 9 800`, `--burst-roll fireball 100000`, then `tools/emu/sk_ground_hit.py game ...` / `cmp` and `sk_ground_roll.py game ...` / `cmp` (headers of both scripts).

## Shared-file lines for the merger

See openspec/changes/prbuildground/merge-lines.txt.
