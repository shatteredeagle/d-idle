# PR body: Blessed Hammer's Concentration share and undead / demon bonus (PALSKILL D2 and D3)

> status: DRAFT body for the PALSKILL D2/D3 pull request to psi29a/d2decomped (card palprep, 2026-11-04). Local branch `local/pal-d2d3-r2`, head `9e9622eb`, 2 commits on upstream main `e5de6c10` (after PR 189); also on the private fork remote (Forgejo), not on the public fork. NOT posted. Replaces `pr-pal-d2d3-body.md` (head `fc2f7c48`, base `8c9a92c7`). The rebase was a clean cherry-pick (no conflict; the 8 files merged without edits). Commits are re-authored with the numeric noreply identity. Tags: V = run on this head (how stated), S = carried from the earlier head, U = unverified.
>
> Title: `Fidelity fixes against game.exe 1.14d: Blessed Hammer's Concentration share and undead / demon bonus`

## What changes

d2d launched a Blessed Hammer with the skill's plain magic damage. game.exe does two more things to it:

1. the do (`005d0040`) scales the missile's magic min and max by the caster's Concentration (`006461d0`);
2. the missile row's damage function 5 (`005ad2f0`) adds half of the rolled magic damage when the monster is undead and again when it is a demon.

Two commits, one claim each, each with its test mode and the same oracle:

| Commit | Claim or change | Evidence | Re-check |
|---|---|---|---|
| `6d6873bf` | The hammer's magic min and max gain the caster's Concentration share: damagepercent x Param1 / 8 (toward 0), then MulDiv(stat, 100 + share, 100) | `005d0040`, `006461d0`, `00483360` | run below, cases with monsters that are neither undead nor demon: 108,598 cases, 57,996 differ on main, 0 after this commit |
| `9e9622eb` | The hammer adds MulDiv(roll, 50, 100) against an undead and again against a demon, before the resistance | `005ad2f0`, `005a8c70`, `005a9270`, `0063e990`, `0063e940` | run below, all monster ids: 109,522 cases, 21,070 differ after the first commit (308 distinct monsters: the 123 undead and 185 demon ids), 0 after this commit |

## Evidence from game.exe 1.14d

| Address | What it shows |
|---|---|
| `005d0040` | do 73: makes the missile, then asks `006461d0` for the influence and, when it is not 0, sets stats 0x34 / 0x35 (magic min / max) to MulDiv(stat, 100 + influence, 100) (V read, `disasm.py`) |
| `006461d0` | 0 unless the unit has state 0x2a (concentration); else the state's stat 25 (damagepercent) x the skill's Param1 (4 for Blessed Hammer) / 8, truncated toward 0 (V read; V run on pct -7, 0, 60, 75, 105, 195, 225, 345) |
| `005ad2f0` | after `005a8c70` filled the record: if dParam1 > 0 and `0063e990` (undead), MulDiv(roll, dParam1, 100) is added to the element's field by `005a9270`; the same with dParam2 and `0063e940` (demon); both can apply (V read) |
| `00483360` | MulDiv, d2d's `rules::mul_div` already reproduces its width quirks (V read) |
| Missiles.txt | the only row with pSrvDmgFunc 5 is `blessedhammer` (index 92, dParam1 = dParam2 = 50, EType mag); dumped with `mpq-cat` from the patched data (V run) |

## How to re-check

```sh
cd tools/emu
set -o pipefail
uv run python hammer_hit.py gen 100000 1 > all.tsv           # every monster id, 109,522 cases
uv run python hammer_hit.py gen 100000 2 plain > plain.tsv   # monsters that are neither undead nor demon, 108,598 cases
for k in all plain; do uv run python hammer_hit.py game $k.tsv > $k.game; done
for k in all plain; do ../../build/tests/test_hammer_hit --hammer-hit $k.tsv | grep -E '^[0-9]+ ' | diff $k.game - | grep -c '^<'; done
```

Good output at the head: `0` and `0`. The case format and the hooks are in the docstring of `tools/emu/hammer_hit.py`. Each case is a hammer level, the two synergy levels, the damagepercent of the caster's Concentration state (or none), a monster id, its magic resistance, its life and the missile's seed. Cases: every monster id at three Concentration settings, Concentration 0 to 60 x hammer 1 to 40 on random monsters, then random.

Counts in this table are S (measured on the earlier head `fc2f7c48`; the current head is in the next section): the worker host, g++ Release, 1.14d data, the oracle running game.exe's functions natively against the real d2d build (`test_hammer_hit`). "Before" is the same test compiled with the `influenced(...)` line removed (the main behaviour), or, for the second commit, the binary of the first commit; the game-side file is identical in every run.

| Run | Cases | main | after commit 1 | after commit 2 |
|---|---|---|---|---|
| plain monsters | 108,598 | 57,996 differ | 0 | 0 |
| all monster ids | 109,522 | 68,544 differ | 21,070 differ | 0 |

## What the oracle runs for real and what it hooks

Real: `00644d50` / `00644e40` (the skill's min and max with the synergies), `006461d0`, `00483360`, `005ad2f0` with `005a8c70` / `005a8910` (the roll on the missile's seed), `0063e990` / `0063e940` on the real MonStats rows, `005a9270`, `0057c6c0` (resistance and life). Hooked, not the claim: the stat getter and setter (a dictionary), the caster's state test `00639df0` and its stat list (`006256b0` / `00625d00`, which serve the case's damagepercent), max life, the difficulty row, and the hit's side effects (events, stun, freeze, poison, logging).

## Callers and what else moves

- `rules::missile_blow` gets the bonus only from `MissileDamage::undead_pct` / `demon_pct`, which `Fight::strike` fills from a row whose pSrvDmgFunc is 5. That is one row, so every other missile (Sorceress, Necromancer, Amazon, monsters, shrines) has 0 and takes the old path unchanged (V read of the diff; V run of the table above for the row count).
- `Missile::influence` is 0 for every missile but a Blessed Hammer cast by the player, so `influenced` returns its argument for them.
- Melee and weapon-based hits do not use these paths (the hammer is a missile skill); the existing melee undead bonus (`Target::undead`, `0057b660`) is untouched, `Target::demon` is new and read only by `missile_blow`.
- A merc or pet never casts Blessed Hammer in d2d; the influence is read for the player's cast only (`!caster.merc && !caster.pet`).

## Counts on the rebased head `9e9622eb` (V)

Re-run this session on the worker host (g++ Release with `-DD2D_WERROR=ON`, 1.14d data, the oracle running game.exe's functions natively against the real d2d build `test_hammer_hit`), same generator seeds as above:

| Run | Cases | differ on this head |
|---|---|---|
| plain monsters | 108,598 | 0 |
| all monster ids | 109,522 | 0 |

The game-side files are byte-identical (sha1) to the ones recorded on `fc2f7c48`, so the game side did not move; only d2d's side was rebuilt. The "main" and "after commit 1" columns above (57,996 / 68,544 / 21,070 differ) are S: measured on the earlier head, not re-run on this base (they need the pre-change binaries).

## Checks (the worker host)

- Author and committer are the numeric noreply identity; both bodies end with `Agent: Claude Code`; no cherry-pick `-x` line (V: `git log --format=%B`).
- Size: 8 files, +304 / -6 (V: `git diff --shortstat e5de6c10..HEAD`).
- `python3 tools/pr_guard.py e5de6c10 HEAD` and on each commit: 0 FAIL, 0 WARN (V).
- Release build, `-DD2D_WERROR=ON`, g++ (V): `test_hammer_hit`, `mpq-cat`, `test_rules`, `test_game_rules`, `test_combat`, `test_skills`, `test_missile_blow_poison`, `test_monster_missile` build; the six tests that take no arguments return 0. `test_missile_hit` returns 77 (skipped, no game data on that run). In the CI-parity lane (g++ suites): `test_rules`, `test_game_rules`, `test_monsters`, `test_monster_thinks`, `test_monster_ai`, `test_combat`, `test_game`, `test_sk_ground` rc 0; `test_hammer_hit` without arguments rc 77 by design (its check is the oracle run above).
- clang++ suites build with the CI flags and `-Werror` (V). The extra warnings stage (`-Wshadow -Wrange-loop-construct -Wconversion`, both compilers, changed TUs): 0 on added lines (V).
- `python3 tools/presubmit.py --quick --base e5de6c10` with `D2_BUILD` pointing at a build that has `mpq-cat` and the game data set: 0 FAIL, 5 WARN (V). The WARNs: ruff and uvx are not installed on that run, so Python is unchecked; the two sibling questions answered above (unit kinds: Blessed Hammer is the player's missile skill, a merc or a pet never casts it; hit paths: melee is untouched, see "Callers"); build and gcc skipped by `--quick`.
- Tooling unit tests (V): every `tools/test_*.py` and `tools/emu/test_*.py` under presubmit OK, including `test_pr_guard.py` and `test_presubmit.py`. In the parity lane the guard stage showed `test_sk_mana.py` failing without `D2_BUILD`/`D2_MPQ_DIR` (a file this change does not touch; with the env set the same presubmit run reports 0 FAIL).
- clang-tidy over the changed TUs (V, one finding on an added line): `tests/game/test_hammer_hit.cpp:13` `misc-include-cleaner`: `#include <d2s_items.hpp>` is not used directly. It is in the first commit and has not been fixed here (a one-line removal if the maintainer wants it). All other tidy output is in files this change does not touch.
- The g++ all-targets stage failed on three targets this change does not touch (`apps/d2d/ui.cpp`, `apps/d2d/town/walk.cpp`, `tests/test_protocol.cpp`); not compared against upstream main (U).

## Not re-run on this head

- The "before" counts (S, above); `test_game` and `test_sk_ground` ran in the parity lane, but `test_missile_hit` was skipped (no data).
- MSVC, a live game, a soak, a determinism hash comparison, a drill with `debug cast 113` then `112` (see Limits).
- The parity lane with the earlier head's results (g++/clang++ on `fc2f7c48`) is superseded by this run; its tidy and g-all stages never reported to this card.

## Limits (U)

- U: the glue in `Fight::fire` and `Fight::strike` is compiled and its neighbours' tests pass, but no test drives a whole `Fight` cast with a Concentration aura on; the oracle covers the rules (`aura_influence`, `influenced`, `missile_blow`) and the loader's two columns. A drill with `debug cast 113` then `112` against a zombie would close this (H3 / H4 of the trace, live only).
- U: where the damagepercent comes from. The oracle feeds it; in `Fight::fire` it is the first friendly aura named `concentration` over the caster (its `aurastat` damagepercent calc, evaluated by `eval_calc`). That calc equals game.exe's on every Paladin aura row (21,760 cases, 0 differ: an internal earlier run, S); two sources of the same state (the player's and a merc's Concentration) are not merged (marked `ponytail`).
- U: the hammer's flight (the spiral), repeated hits of one hammer on a monster, and who else an aura reaches are other deviations (D4, D6, D7 of the trace) and are not part of this change.
- The run is at clvl 30 and Concentration damagepercent -20 to 1000; resistance -100 to 150; no pierce (magic has none); a plain monster (no hireling, magic damage reduction 0, no absorb).

Agent: Claude Code
