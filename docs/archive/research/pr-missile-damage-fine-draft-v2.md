# PR draft v2: a missile's element damage keeps its 256ths

> status: ARCHIVED 2026-10-31 (doctrim): moved here from its old path (a stub remains there). The status line below is the one it had when written.

> status: DRAFT (agent PRPROOF, 2026-10-30), supersedes docs/research/pr-missile-damage-fine-draft.md's body. Nothing is pushed, posted or filed. Branch `fix/missile-damage-fine-v2` in `~/d2d-lab/pr/gaps` on the worker host (commits 4e6d9d39, e8dd9efd, rebased on upstream/main 32768ddb; verifier's findings applied). Proof: docs/research/pr-missile-damage-fine-proof.md. The body below is in their template (.github/pull_request_template.md), ready for `--body-file`.

````
## What changes
d2d cut each spell's rolled element damage to whole hit points before applying the monster's resistance and taking it off its life: a Charged Bolt of 2-4 dealt 2 or 3 (mean 2.5). game.exe rolls the element in 256ths, applies the resistance to that fine value, keeps monster life in 256ths and kills only under one point, so the mean is 3.0 (17 % more at Charged Bolt level 1, 8 % at level 8). Now `missile_blow` keeps the fraction (`Blow::frac`, `rules::resisted_fine` with game.exe's MulDiv branches), the monster carries it (`Monster::life_frac`), a dead monster's life is 0, and a heal (`heal()`) is capped at max life with the fraction cleared only when the cap cuts.

Scope: a missile's element damage taken by a monster. Not changed: the physical part, melee, poison, burning, thorns, pets and mercs (whole points as before; siblings warning answered: the rule is the monster's life, the other kinds take whole points through their own paths), hit recovery (still whole points), pierce against an immune (game.exe lets a plain monster's immunity be pierced, LAB_0057be20; d2d keeps it: a separate gap, not changed here).

## Evidence from game.exe 1.14d

| Address | What it shows |
|---|---|
| `FUN_005a89a0` | the missile's damage record: lightning `min + rand(max - min)` in 256ths into record +0x1c (stats 0x32 / 0x33, mastery 0x14a) |
| `FUN_005ad730` | hands the record to `FUN_0057c6c0` with the apply flag set (`LAB_005ad8c6`), no rounding |
| `FUN_0057c1e0` | folds each element of the table at 0x732980 (lightning: +0x1c, resist stat 0x29, pierce 0x14e, absorb 0x90 / 0x91) through `FUN_0057bf80`, total into record +0x4c (`LAB_0057c3bd`); a plain monster (not a `FUN_0063ee90` hireling, `LAB_0057c242`) gets no difficulty penalty and no 75 % cap (`FUN_0057be00`) |
| `FUN_0057bf80` | the resistance on the fine value: 100 % or more is nothing, 0 leaves it, else `MulDiv(dmg, 100 - res, 100)` (`LAB_0057bfea`) |
| `FUN_00483360` | MulDiv: `a * b / c` up to 0x100000, `a / c * b` above it |
| `LAB_0057c852` | record int 0x13 (+0x4c), the total in 256ths, taken off the life stat 6 (also 256ths) |
| `LAB_0057c865`, `LAB_0057c874` | life under 0x100 after the hit is set to 0: dead |
| `FUN_0057a980` | a heal: 256ths added, capped at max life (`FUN_00625d10`) |

## How to re-check

```sh
python3 tools/pr_guard.py
cmake --build build && ctest --test-dir build --output-on-failure   # test_game needs D2_MPQ_DIR and D2_PATCH_INSTALLER
# the oracle: game.exe's own FUN_0057c6c0 / FUN_0057c1e0 / FUN_0057a980 on 57,304 exhaustive boundary lines
# and 100,000 random hit/heal sequences, against d2d's real missile_blow + hurt + heal on the same cases:
cd tools/emu && uv run python life_apply.py 100000 1 > g.txt       # stderr: "500425 events: old d2d wrong on 307385, new d2d wrong on 0"
cd ../.. && build/tests/test_game --life-apply 1 100000 | grep -E '^[BMR][0-9]* ' | diff tools/emu/g.txt -   # empty
```

Measured effect (corroboration only, not the evidence): on a 36-run lockstep probe (Blood Moor Fallen, Charged Bolt 1 / 4 / 8, 4 seeds, about 200 Fallen kills per cell) the fix lowers real casts per kill at Charged Bolt 8 from 0.60 to 0.52; Charged Bolt 4 is within noise (0.75 vs 0.75 / 0.78); Charged Bolt 1 was disturbed by host load and is not claimed. Kills and xp per kill unchanged.

Limits: `life_frac` is not replicated to co-op clients (the server is authoritative); the damage-over-time paths and `soak()`'s partial absorb keep whole points as before. The oracle hooks the stat getter and setter (a dict; stats 6 / 7 in 256ths per docs/research/re/monsters.md and the 0x100 threshold), max life, the unit-kind predicates and the side effects that are not this claim (events, stun / cold / freeze / poison / burn, logging); its docstring lists them. The cases cover a plain monster with no magic damage reduction or absorb, resistance -100..150, pierce against a non-immune, whole-point heals. Other callers of `hurt()` still pass whole points; the only change they see is a dead monster's `hit_points` 0 instead of negative (two thorns asserts in test_game.cpp updated). `tools/emu/life_apply.py` touches `tools/`, so it needs the maintainer's CODEOWNERS review; the fix is the first commit and stands without it.

## Screenshots
Not applicable.

## Checklist
- [x] A test in `tests/` (fix or port): test_combat (the mean 3.0, `resisted_fine` edges), test_game (two halves make a point, under one point is dead and 0, the heal's cap; `--life-apply` mode for the oracle)
- [x] The finding in `docs/research/re/`: combat.md "Applying damage" (no todo.md row)
- [x] `python3 tools/pr_guard.py` passes

---

Agent: Claude Code
````

## Commands for the coordinator (NOT RUN; each needs the user's go; at least one hour after our last PR)

```sh
ssh worker.example 'cd ~/d2d-lab/pr/gaps && git fetch upstream && git rebase upstream/main fix/missile-damage-fine-v2 && python3 tools/pr_guard.py upstream/main && cd /tmp/prbuild-build && ninja test_combat test_game && ./tests/test_combat && D2_MPQ_DIR=~/d2d-lab/data D2_PATCH_INSTALLER=~/Downloads/LODPatch_114d.exe ./tests/test_game > /dev/null && echo READY'
ssh worker.example 'cd ~/d2d-lab/pr/gaps && git push origin fix/missile-damage-fine-v2'
# save the body above (inside the four-backtick fence) to ~/d2d-lab/pr/gaps/PR_DESCRIPTION.md, then:
ssh worker.example 'cd ~/d2d-lab/pr/gaps && gh pr create --repo psi29a/d2decomped --base main --head shatteredeagle:fix/missile-damage-fine-v2 --title "A missile element keeps its 256ths" --body-file PR_DESCRIPTION.md'
```
