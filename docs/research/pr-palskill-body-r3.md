# PR body: Blessed Hammer's Concentration share and undead / demon bonus (PALSKILL D2 and D3)

> status: DRAFT body for the PALSKILL D2/D3 pull request to psi29a/d2decomped (card palprep3, 2026-11-04). Local branch `local/pal-d2d3-r3`, head `1f91a216`, 2 commits on upstream main `7591a92e` (after PRs 190 and 191); also on the private fork remote only. NOT posted, nothing pushed to GitHub. Replaces `pr-palskill-body-r2.md`. Tags: V = run this session on this head (how stated), S = read without a run, U = unverified.
>
> Title: `Fidelity fixes against game.exe 1.14d: Blessed Hammer's Concentration share and undead / demon bonus`

## What changes

d2d launched a Blessed Hammer with the skill's plain magic damage. game.exe does two more things to it:

1. the do (`005d0040`) scales the missile's magic min and max by the caster's Concentration (`006461d0`);
2. the missile row's damage function 5 (`005ad2f0`) adds half of the rolled magic damage when the monster is undead and again when it is a demon.

Two commits, one claim each, in one PR because both change the same hit (the second edits lines the first adds, and its count is measured on top of the first):

| Commit | Claim | Addresses |
|---|---|---|
| 1 | The hammer's magic min and max gain the caster's Concentration share: damagepercent x Param1 / 8 (toward 0), then MulDiv(stat, 100 + share, 100) | `005d0040`, `006461d0`, `00483360` |
| 2 | The hammer adds MulDiv(roll, 50, 100) against an undead and again against a demon, before the resistance | `005ad2f0`, `005a8c70`, `005a9270`, `0063e990`, `0063e940` |

## Evidence from game.exe 1.14d

| Address | What it shows |
|---|---|
| `005d0040` | do 73: makes the missile, then asks `006461d0` for the influence and, when it is not 0, sets stats 0x34 / 0x35 (magic min / max) to MulDiv(stat, 100 + influence, 100) (V read, `tools/emu/disasm.py`) |
| `006461d0` | 0 unless the unit has state 0x2a (concentration); else the state's damagepercent x the skill's Param1 (4 for Blessed Hammer) / 8, truncated toward 0 (V read) |
| `005ad2f0` | after `005a8c70` filled the record: if dParam1 > 0 and `0063e990` (undead), MulDiv(roll, dParam1, 100) is added to the element's field by `005a9270`; the same with dParam2 and `0063e940` (demon); both can apply (V read) |
| `00483360` | MulDiv; d2d's `rules::mul_div` already reproduces its width quirks (V read) |
| Missiles.txt | the only row with pSrvDmgFunc 5 is `blessedhammer` (index 92, dParam1 = dParam2 = 50, EType mag) (V run, dumped with `mpq-cat` from the patched data) |

## How to re-check

```sh
cd tools/emu
set -o pipefail
uv run python hammer_hit.py gen 100000 1 > all.tsv           # every monster id, 109,522 cases
uv run python hammer_hit.py gen 100000 2 plain > plain.tsv   # monsters that are neither undead nor demon, 108,598 cases
for k in all plain; do uv run python hammer_hit.py game $k.tsv > $k.game; done
for k in all plain; do ../../build/tests/test_hammer_hit --hammer-hit $k.tsv | grep -E '^[0-9]+ ' | diff $k.game - | grep -c '^<'; done
```

Good output at this head: `0` and `0` (V: both cases ran on the worker host on head `1f91a216`, g++ Release, `-DD2D_WERROR=ON`, 1.14d data). The case format and the hooks are in the docstring of `tools/emu/hammer_hit.py`. Each case is a hammer level, the two synergy levels, the damagepercent of the caster's Concentration state (or none), a monster id, its magic resistance, its life and the missile's seed.

What the oracle runs for real: `00644d50` / `00644e40` (the skill's min and max with the synergies), `006461d0`, `00483360`, `005ad2f0` with `005a8c70` / `005a8910`, `0063e990` / `0063e940` on the real MonStats rows, `005a9270`, `0057c6c0` (resistance and life). Hooked, not the claim: the stat getter and setter, the caster's state test and its stat list (they serve the case's damagepercent), max life, the difficulty row, and the hit's side effects.

A note on the random draw: on a real hit game.exe rolls the hammer's damage twice from the missile's seed, once for the whole record (`005a89a0`) and again inside the undead / demon function (`005a8c70`, which overwrites the first magic roll before adding the bonus). The oracle starts at `005ad2f0`, so its roll is the second draw, the one that counts. d2d draws from one shared random stream per fight rather than a per-missile seed, so the distribution is the same and gameplay is unaffected.

## Results on this head (V, the worker host)

| Run | Cases | Differ on this head |
|---|---|---|
| the harness above, plain monsters | 108,598 | 0 |
| the harness above, all monster ids | 109,522 | 0 |

Before the change, on the same inputs: 57,996 of the plain cases differed on the earlier main and 21,070 of the all-ids cases differed after commit 1 alone (S: counts from the earlier base `8c9a92c7`, not re-run on this base).

A second, independently written comparison (not part of this PR) ran the same functions in game.exe's real hit order (record roll `005a89a0`, then the damage function through the missile table, then `0057c6c0`) against the real `missile_damage`, `aura_influence`, `missile_blow` and `hurt` of this head, on three other seeds, every MonStats row twice, hammer level 1 to 99, clvl 1 to 99, synergies 0 to 99, damagepercent edges from -100 to 400000, resistance -100 to 150, life 1 to 3000. 734 rows, 100 of which d2d never lets a hit land on (`targetable()` false) were left out and counted:

| Seed | Cases in domain | differ on this head |
|---|---|---|
| 20261104 | 88,826 | 0 |
| 777 | 45,776 | 0 |
| 4242 | 5,155 | 0 |

A run with every row marked both undead and demon (the real tables have no such row) also gave 0 differing. The same cases on the base build differ on 99,698 of 139,757 (V run on the earlier base `e5de6c10`, which had no hammer change), and on none of the plain monsters without a Concentration. These 0s compare with the second draw described above; with d2d's stream used as is, the life field differs on the cases where the first draw moves the roll, and equals again once that draw is accounted for.

## Callers and what else moves

- `rules::missile_blow` gets the bonus only from `MissileDamage::undead_pct` / `demon_pct`, which `Fight::strike` fills from a row whose pSrvDmgFunc is 5. That is one row, so every other missile (Sorceress, Necromancer, Amazon, monsters, shrines) has 0 and takes the old path unchanged (V read of the diff).
- `Missile::influence` is 0 for every missile but a Blessed Hammer cast by the player, so `influenced` returns its argument for them.
- Melee and weapon-based hits do not use these paths: the hammer is a missile skill, and the existing melee undead bonus (`Target::undead`, `0057b660`) is untouched; `Target::demon` is new and read only by `missile_blow` (S: source read).
- A merc or pet does not cast Blessed Hammer in d2d; the influence is read for the player's cast only (`!caster.merc && !caster.pet`) (S: source read, no test drives it).
- The bonus code in `missile_blow` is magic-only in effect: for poison game.exe's base is the roll x 25, which it does not model. A comment says so; only `blessedhammer` (magic) has pSrvDmgFunc 5 today.
- Upstream's own Sorceress missile differential (`missile_hit.py gen 20000 99`, 64,000 cases) has 0 differing on the earlier base `e5de6c10` and on this change applied to it (V run there, S for this head: not re-run on `7591a92e`).

## Checks (V, the worker host, head `1f91a216` on upstream `7591a92e`)

- Author and committer are the numeric noreply identity; both bodies end with `Agent: Claude Code`; no cherry-pick `-x` line.
- Size: 8 files, +307 / -6 (`git diff --shortstat 7591a92e..HEAD`).
- `python3 tools/pr_guard.py 7591a92e HEAD` and on each commit: 0 FAIL, 0 WARN.
- Release build with `-DD2D_WERROR=ON`, g++: `test_hammer_hit`, `test_combat`, `test_rules`, `test_game_rules`, `test_game`, `test_monsters`, `test_skills` build; all return 0 except `test_hammer_hit`, which without arguments returns 77 by design (its check is the run above).
- The g++ all-targets stage fails on three targets, `apps/d2d/ui.cpp`, `apps/d2d/town/walk.cpp` and `tests/test_protocol.cpp`; the base `7591a92e` built the same way fails on the same three, so this change does not cause or touch them.
- clang-tidy 23.1.4 with the repo's `.clang-tidy` on the changed translation units (`fight/missiles.cpp`, `gamedata_load.cpp`, `tests/game/test_hammer_hit.cpp`): 0 findings in them. The only output is `readability-identifier-length` at `components/drlg/outdoor.hpp:1052` and `:1090`, a file this change does not touch (the same on base).
- `python3 tools/presubmit.py --quick --base 7591a92e` with `D2_BUILD` pointing at a build that has `mpq-cat` and the game data: 0 FAIL, 4 WARN. The WARNs: the two sibling questions are answered under "Callers" (unit kinds: a merc or pet does not cast it; hit paths: melee is untouched); build and gcc are skipped by `--quick`.

## Not re-run

- MSVC, clang++ suites and the full parity lane on this head, a live game, a soak, a determinism hash comparison.
- The "before" counts (S, above).

## Limits (U)

- U: the glue in `Fight::fire` and `Fight::strike` is compiled and its neighbours' tests pass, but no test drives a whole `Fight` cast with a Concentration aura on; the oracle covers the rules (`aura_influence`, `influenced`, `missile_blow`) and the loader's columns.
- U: where the damagepercent comes from. The oracle feeds it; in `Fight::fire` it is the first friendly aura named `concentration` over the caster (its `aurastat` damagepercent calc, evaluated by `eval_calc`). The calc gives 60 + 15 x (level - 1) for levels 1 to 40 (V run), which matches the known 1.14d values (S, not compared with game.exe here). Two sources of the same state (the player's and a merc's Concentration) are not merged (marked `ponytail`).
- The hammer's flight (the spiral), repeated hits of one hammer on a monster, and who else an aura reaches are other deviations and are not part of this change.
- game.exe has 22 other missile rows with their own pSrvDmgFunc (1, 2, 3, 4, 7 to 14, for example firewall, icearrow, moltenboulder) that d2d does not model; the loader now keeps the column for every row, and only function 5 is used.
- The runs cover clvl 30 (author's harness) and 1 to 99 (second comparison); no pierce (magic has none); a plain monster (no hireling, magic damage reduction 0, no absorb).

Agent: Claude Code
