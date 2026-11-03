# PR draft: the player-side missile damage roll: element top, physical in 256ths, -100 floor (PLAYERROLL)

> status: DRAFT, verified by the independent verifier (`docs/research/pr-playerroll-verifier.md`, branch agent/verifyplayerroll, READY WITH CHANGES; the changes are made here), held for the user's go (nothing opened, pushed upstream or posted). Fork branch `pr/player-roll` (pushed to the private fork remote as `pr/player-roll-v2`, because the rewritten history is not a fast-forward of the old remote `pr/player-roll` at `28c0540a`), head `e66e10ba`, five commits on upstream main `6935cf64` (read-only fetch 2026-11-02): `e5481b2b` (oracle and test mode), `21f1a179` (element roll top), `e87529b2` (physical roll in 256ths), `9808e873` (-100 floor), `e66e10ba` (three include lines in `tests/game/test_game_missiles.cpp`, see CI below). The verifier read the same code as `28c0540a` (now `pr/player-roll-v1-backup`); the rebase only rewrote the author to one identity and added the include commit, `git diff 28c0540a e66e10ba` is that file alone. The lab twin `local/playerroll-lab` (51e4eb1e) is unchanged. Inputs: the roll-top pair 9ba8d369 / 23e2baec (SKMISS: `docs/research/nec-skill-trace.md` D1, `sor-rest-trace.md`), the physical candidate `local/skabrecheck-phys` (`skabrecheck.md` claim 2) and its verifier record `docs/research/pr-physroll-verifier.md` (branch agent/verifyphys; READY WITH CHANGES). The burst roll (`sor-rest-trace.md` D1, with the Meteor deviation) is NOT part of this PR. Tags: V = run on the worker host this session, S = read, U = not verified.

Title: `A missile's element roll never reaches its max, and its physical roll keeps its 256ths (FUN_005a89a0)`

## What changes

game.exe rolls a skill missile's damage in FUN_005a89a0 (physical at record +8, elements at +0x10..+0x28): each part is 0 unless min and max are both over 0, else `min + rand(max - min)` on the missile's seed (FUN_005a8910 / FUN_0045c3e0, max exclusive), in 256ths; FUN_0057bf80 resists that fine value and FUN_0057c6c0 takes it off life in 256ths. d2d's `rules::missile_blow` differed in three ways, one commit each (all in `components/rules/combat.hpp`):

1. `21f1a179` element roll: `Rng::range(lo, hi)` is top-inclusive; now `lo + rng(hi - lo)` (ends swapped, 0 unless the lower end is over 0). Covers fire, lightning, cold, poison and magic. Side effect on the RNG stream: a zero or negative low end no longer draws (and no longer deals damage), and swapped ends now draw, as FUN_005a8910 / FUN_0045c3e0 do (n <= 0 returns 0 without touching the seed). A `lo == hi` element never drew, on base either. No skill produces a zero or negative low end or swapped ends (all classes, 35,280 pairs, V run by the verifier), so the draw count is unchanged for every skill missile; only a `MissileDamage` built elsewhere (rows without a Skill, `pet_aura`, shrine potions) could see it, U whether any does.
2. `e87529b2` physical roll: the roll was shifted to whole points before the resistance (`>> 8` then `resisted`); now the fine roll goes through `resisted_fine`, the whole points go to `damage` and the part under a point to `Blow::frac`, as the element branch already does. Same single RNG draw.
3. `9808e873` floor: for a monster defender FUN_0057be00 floors the physical resistance at -100; d2d took -150 as x2.5. One token, `std::max(target.res[0], -100)`. A separate commit because it is latent (see Limits): drop it if the maintainer prefers.

`e5481b2b` is the oracle (`tools/emu/missile_hit.py`) and `test_missile_hit --missile-hit`, unchanged from the SKMISS pair.

## Reach (callers of `missile_blow`, all with a monster target; S, caller audit in `pr-playerroll-verifier.md` section 4 (the verifier's re-read at `28c0540a`))

- Element roll (commit 1): every element hit that goes through `missile_blow`: player and merc skill missiles (`Fight::strike`, `by_merc`; the act 1 and act 3 merc skills), area releases, Shiver Armor / Thunder Storm / Hurricane state hits (`states.cpp`), the element parts of Psychic Hammer / Mind Blast / Telekinesis (`spot()`), pet auras and vines (`pet_aura`, Poison Creeper in `pets.cpp`), shrine potions (`tick.cpp`) and missile rows without a Skill. Not reached: object traps (their damage goes into a hostile missile's `MonStats` and hits the player through the monster overload) and merc auras.
- Physical roll and floor (commits 2, 3): only a skill with Skills.txt MinDam / MaxDam over 0 reaches the physical branch (rows with no Skill, `missile.fixed`, auras and shrines carry no physical). Amazon: Magic Arrow. Assassin: Psychic Hammer and Mind Blast (through `spot()`), Blade Sentinel, Blade Fury. Barbarian: War Cry. Druid: Twister, Tornado, Shock Wave, Volcano, Armageddon. **No Sorceress, Necromancer or Paladin path.** Not reached: Holy Shield (Smite uses `player_blow`), the Druid summons (own attack stats), Molten Boulder and Blade Shield (rows have no Skill: no physical in d2d at all, U, separate gap).

## Side effect to name

`hurt()` now also receives the physical fraction. A physical hit under one point (damage 0, frac > 0) no longer returns early: it makes the monster aware and rolls its get-hit, and the get-hit size includes the fraction. Both follow game.exe's 256ths (the get-hit comes from the fine total, I, FUN_0057cb00 not re-traced here), but they change monster modes, not only life. Volcano and Armageddon also fire, so the fraction sum can reach 510 per call and `hurt` carries it (S).

## Proof (V, worker host, d2d builds -j2, game.exe's code under the fork emulator as oracle)

| claim | harness | before | after (head `e66e10ba`) |
|---|---|---|---|
| physical, real `missile_blow` + real `hurt()`, life in 256ths after every hit vs game.exe's FUN_0057c6c0 (the verifier's `physroll_game.py` + `test_rollprobe`, new seeds 4242 and 31337 with edge, death and random-res grids, plus a floor grid of every res from -101 to -300 x 14 edge rolls) | 110,000 cases (40,000 + 40,000 + 30,000 incl. 30,000 at res -101 to -300), monster defender | upstream main `6935cf64` differs on **81,464** (28,707 / 28,873 / 23,884) | **0 of 110,000**, every class: immune, res 0, 0 < res < 100, -100 <= res < 0, res < -100, rolls <= 0, rolls over 2^20 and 0x7FFFFFFF |
| physical arithmetic, game.exe's own roll fed to d2d (column B; the author's `skab_phys.py` + `skab_phys_test.cpp`, seed 7) | 101,626 cases | base header **84,271** differ | **0**; with commits 1 and 2 only (floor off) 16,526 differ, all at res under -100 |
| element roll at the roll level: game.exe's FUN_005a8910 run natively vs d2d's real `missile_blow` (fire at res 0), roll and the RNG state after each roll (the verifier's `elemroll_game.py` + `test_rollprobe --elemroll`, seed 8128) | 60,000 cases: all 25 Sorceress and 8 Necromancer element skills (levels 1 to 40, synergies, masteries) plus 920 edge cases | upstream main differs on **59,580** (rolls; the seed after differs only on 240 edge cases) | **0 of 60,000**, roll and seed after |
| element roll, author's runs: Sorceress `sk_rest_hit.py` (Fire Ball, Glacial Spike, 31,000 cases) and Necromancer `nec_missile_hit.py` (Teeth, Bone Spear, Bone Spirit, 36,600 cases, life after one hit) | md5 of game.exe and d2d lines | the old roll is wrong on 30,997 of 31,000; Necromancer life after one hit differs on 10,509 (a different measure from the roll-level count above) | 0 differ, md5 equal both sides |

How much it matters in play: the element fix is a fidelity fix. The old roll was `x mod (n+1)` against game.exe's `x mod n`, so single rolls differ almost always at the same seed, but d2d draws from the shared fight stream, not the missile's seed; the effect is that the top 256th is no longer rolled, a mean shift of 1/512 of a point per element hit. It is not that 30 percent of hits are wrong. The physical commits matter for the fraction (sub-point hits) and for res under -100.

What these do not prove: column A of the author's run (d2d's own roll, same seed) still differs on 83,964 of 101,626 physical cases, because d2d's physical roll is `Rng::range` (top-inclusive, and a zero min is rolled where game.exe rolls nothing): the physical roll range is deliberately not in this PR (see "Not in this PR"). The magic element Necromancer set and the Sorceress set run the roll with the same function, so they cover commit 1 only.

Unit tests: `tests/test_combat.cpp` asserts the element top (`elo 256 / ehi 257` never gives a fraction, and nothing without a min over 0) and, for the physical, a roll of 300 / 256ths: 1 point and 44 / 256 at res 0; 0 and 150 at res 50; 0 and 0 at res 100; and at res -150 exactly the -100 result (2 points and 88). Checks (V, the verifier's ctest at `28c0540a`, data tests with the MPQ folder and patch installer set): 67 tests (66 on base, which has no `test_missile_hit`), 56 pass, 7 skip, 4 fail, and the same four fail on upstream main: `test_mpq` (abort on this MPQ folder), `smoke_d2d`, `quests_act1`, `quests_act4`; `test_missile_hit` skips without arguments by design. Re-run at `e66e10ba` (V, own Release build, -j2): `test_combat`, `test_rules`, `test_game_rules`, `test_game` exit 0; `pr_guard.py 6935cf64` 0 FAIL, 0 WARN; `presubmit.py --quick --base 6935cf64` 0 FAIL, the same 2 WARN. `pr_guard.py 6935cf64`: 0 FAIL, 0 WARN. `presubmit.py --quick --base 6935cf64`: 0 FAIL, 2 WARN (answered under Limits), ruff clean.

Warnings-as-errors check (coordinator's rule after PR 164): the CI flags are `-Wall -Wextra -Wpedantic -Wno-missing-field-initializers -Werror` (GCC / clang) and `/W4 /WX` (MSVC) in `CMakeLists.txt`. The worker host's build is `D2D_WERROR=OFF` (the 2,250 warnings of the full build are in untouched files; none in `combat.hpp`, `test_combat.cpp`, `test_missile_hit.cpp`). I compiled `test_combat.cpp` and `test_missile_hit.cpp` with the target's own command line on g++ 16.2.1 and clang++ 23.1.4, adding `-Wshadow -Wrange-loop-construct -Wconversion -Wsign-conversion -Wunused`: this found one error in my test (`hit` shadowed an outer `hit`, MSVC C4456), fixed in the same commits (renamed `phys_hit`); after it 0 warnings on every line this PR adds on both compilers. Hand review for MSVC /W4: no new locals besides `phys_rng`, `phys`, `half`, `phys_hit`, no size_t / double conversions, `std::max(int, int)` with the literal `-100`. MSVC itself was not run (U).

## Deterministic scenarios (the lab patch set; base = lab `ee5e06d8`, head = lab + these four commits, branch `local/playerroll-lab`)

The fork's det harness (`tools/emu/det_scenarios_lab.py`, `det_scenarios_run.py`) on two lab binaries, seed 3, lockstep: 14 scenarios plus a new one, `hit_phys` (this card): a Druid casts Twister, Shock Wave and Tornado one after the other point blank on a fresh plain zombie, the physical missile skill the verifier said was still owed.

| scenario | head vs base |
|---|---|
| passive, fight, cold, static, cold2, hit_immune, merc, follow | identical (mon and state) |
| pet | differs only in the Spirit Wolf start position (44.1,21.1 vs 44.3,21.0) at `start`, before any blow: a run-to-run flake of the pet's start position (it hit 3 of the verifier's 6 non-base runs, both values occur on binaries that cannot differ there); reruns equal base |
| hit_plain, hit_champion, hit_unique, hit_boss, hit_minion | differ in the target's hit points only (monster lines; one point here and there), the claimed element roll: the physical-only lab binary (`ee5e06d8` + `5b13a2c4`, no roll-top) hashes the same as base on all five, so commit 1 alone moves them |
| hit_phys (new) | differs from base in 5 of 27 snapshots, monster hit points only, from Tornado on (`phys skill 245`, 125 against 126 of 147); identical to the physical-only binary, so commit 2 alone moves it, repeatable (`base2` and `player2` reproduce both hashes). Twister and Shock Wave did hit (5 and 13 points on base's numbers) but did not move a hash: they deal the same whole points on both binaries and no fraction carry crosses a point inside the scenario |

So 9 of the 15 scenarios are byte-identical (the pet after its flake rerun), 6 differ in exactly the claimed field by the claimed commit (five by commit 1, `hit_phys` by commit 2).

## Limits

- Commit 3 (floor) is latent: MonStats physical resistance is not below 0 and Amplify Damage gives exactly -100, so res under -100 needs a stacked source. It is a separate commit for that reason; with it the verifier's harness is 0 of 60,000 at all res (without it 1,456).
- The physical roll range is not changed: game.exe's `min + rand(max - min)` and `0 unless both ends over 0` (D3 / D5 of `skabrecheck.md`) for the physical still differ (column A above). Same function, a next commit once the maintainer has taken this one; the roll-top commit's comment in `docs/research/re/missiles.md` already names the physical as the same rule.
- Not in this PR: the srcdam residual (weapon share whole-point in `player_blow`, then the skill's fine part: Magic Arrow, Blade Fury), `Fight::land` soak using whole-point `resisted(blow.phys, ..)` and dropping `frac` when soak takes part (only monsters with an armour pool), Molten Boulder / Blade Shield carrying no physical, the percent stats FUN_005a89a0 applies (0 for skill missiles, I), the Sorceress burst roll (`sor-rest-trace.md` D1).
- Presubmit WARNs, answered: *doc-claims, `docs/research/re/missiles.md` line 106 ("never")*: the line states game.exe's rule (FUN_005a8910 never rolls the max); d2d now implements it for every element branch (commit 1) and the physical branch is named above as not yet; *build skipped (--quick)*: answered by the ctest run above.
- Not run: an MSVC build, a live game, the full `presubmit` build. The CI step "Names and includes (Linux)" was run as the workflow does (see below); CI's clang-tidy is older than the local LLVM 23 and not identical.

## CI: Names and includes (Linux) (V run on the worker host)

The workflow tidies every file that includes a changed header, here all 124 files that include `combat.hpp`. `tests/game/test_game_missiles.cpp` is unchanged from upstream main but had three missing direct includes (`character_store.hpp`, `combat.hpp`, `skills.hpp`), exactly the three PR 164 fixes, so this PR would have failed the step. Commit `e66e10ba` adds the same three lines (identical text, so merging this branch with the 164 branch `fix/monster-missile-record-and-poison` is conflict-free: merged in a throwaway worktree, no conflict). Run as the workflow does (`tools/tidy-pr-select.py` against the build, then `clang-tidy -p build --quiet --extra-arg=-Wno-unknown-warning-option` one file at a time, niced, local LLVM 23.1.4): 124 files selected, **0 include-cleaner findings in any of them and 0 findings of any kind in our files** (`combat.hpp`, `test_combat.cpp`, `test_missile_hit.cpp`, `test_game_missiles.cpp`). The only findings are 377 `readability-identifier-length` errors on short structured-binding names in untouched files (`outdoor.hpp`, `merc.hpp`, `pets.hpp`, `input.cpp`, `setup.cpp`, `panels.cpp`, `main.cpp`, `frontend.cpp`), a check of LLVM 23 that CI's older clang-tidy does not apply (the verifier saw the same on base).

## How to review

1. `components/rules/combat.hpp`, `missile_blow` (the whole change: three small hunks).
2. FUN_005a89a0, FUN_005a8910, FUN_0045c3e0, FUN_0057be00, FUN_0057bf80, FUN_0057c6c0 with `tools/emu/disasm.py`.
3. Elements: `tools/emu/missile_hit.py` + `test_missile_hit --missile-hit`. Physical: `tools/emu/skab_phys.py gen / game` and `cmp` against `skab_phys_test.cpp --phys`; the real-`hurt()` probe is in the verifier's record (`pr-playerroll-verifier.md`, Reproduce).
4. Det: `det_scenarios_run.py run BINDIR OUT hit_phys,... base,head` inside a d2d slot.

## What the independent verifier must check (narrow, third party)

- The combined head `e66e10ba` on upstream main: `git diff 6935cf64 e66e10ba` is the 5 commits above; `git merge-tree` clean against current upstream main.
- The floor: res under -100 at head against game.exe on a res grid that includes -101 .. -300 and a monster defender; and that FUN_0057be00's floor is only for monster defenders (every caller here has a monster target).
- The det scenario `hit_phys` exercises a real physical hit (the five moved snapshots) and the five cold hit scenarios move only through commit 1.
- The RNG draw change of commit 1 (a zero or negative low end no longer draws, swapped ends draw) is consistent with FUN_0045c3e0; `lo == hi` was never a change.
- Warnings-as-errors: build with the CI flags on a toolchain that has them (MSVC /W4 /WX was not run here).

## Shared-file lines for the merger

In `openspec/changes/playerroll/merge-lines.txt`.
