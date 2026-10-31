# PR draft: a missile's element damage keeps its 256ths

> status: DRAFT (agent PRBUILD, 2026-10-30). Nothing is pushed, posted or filed. The branch exists only in the clone `~/d2d-lab/pr/gaps` on the worker host (fork of psi29a/d2decomped, remote `upstream` fetched at af825eda). Each push and the PR need the user's own go (upstream AGENTS.md section 2). Evidence: `docs/research/upstream-gaps/item1-charged-bolt-hit.md` on branch agent/upstreamgaps 89e01d4. Tags: V = run by this agent, S = secondary, U = unverified.

## The branch (V)

- `~/d2d-lab/pr/gaps`, branch `fix/missile-damage-fine`, one commit `91a7cc92`, parent upstream/main `af825eda`. Author and committer `shatteredeagle <26069301+shatteredeagle@users.noreply.github.com>`, no trailer.
- Diff: 7 files, +47 -5. `rules::missile_blow` (components/rules/combat.hpp) keeps the part of an element roll under one point in a new `Blow::frac` (0..255) instead of `roll >> 8`; `Monster::life_frac` holds the 256ths already taken off `hit_points`; `hurt()` takes an optional `fraction`, carries it, and sets life to 0 when under one point after a hit; `Fight::land` passes `blow.frac`. A note in `docs/research/re/combat.md` under "Applying damage". Tests in `tests/test_combat.cpp` (20000 Charged Bolt rolls of 2-4, mean 3.0 +- 0.03, `frac` in 0..255) and `tests/test_game.cpp` (two halves make a point, under one point is dead).
- Scope kept to the evidence: only the element branch of `missile_blow` (fire, lightning, cold, magic; resistance applied to the fine value, 64-bit). Not changed: physical part of a missile, poison (already in 256ths), stun, chill rule (`resisted_damage > 0`), melee, thorns, burning, pets and mercs (`world.cpp` monster-on-monster `damage >> 8`), hit recovery (`damage * 8 >= max_hp` still on whole points), `soak` (armor pool; `frac` is dropped when the pool took part of the blow). Monster life in d2d is whole points; game.exe's life is 256ths, so the branch adds `life_frac` beside it rather than rescaling every consumer of `hit_points` (138 uses).
- Known small limit (U): `life_frac` is not reset by heals, resurrection or clones, and is not replicated to a co-op client; a stale value is at most 255/256 of a point. Say so in the PR if the reviewer asks.

## Rules check (V)

- `python3 tools/pr_guard.py upstream/main`: `0 FAIL, 0 WARN` (tools/ and CI untouched, so no `security-reviewed` label and no CODEOWNERS path). Note `up/main` from the brief is not a ref in the clone; `upstream/main` is.
- `python3 tools/presubmit.py --quick`: 0 FAIL, 1 WARN left after the docs line was added: `siblings` (unit kinds: the diff touches monster, not player, merc, pet, co-op partner). Answer for the PR body: the rule is the monster's life; pets and mercs take whole points through `pet_hurt` and are not covered by the evidence.
- No emoji, no short names, commit subject short with 2 body lines, no Claude trailer.

## Tests run on the worker host (V)

Fresh build dir `/tmp/prbuild-build` (cmake -G Ninja, FETCHCONTENT sources from `~/d2d-lab/src/build/_deps`, nothing installed):

```sh
ninja test_combat test_game
./tests/test_combat                                                        # ok
D2_MPQ_DIR=~/d2d-lab/data D2_PATCH_INSTALLER=~/Downloads/LODPatch_114d.exe ./tests/test_game   # OK ... (rc 0)
```

`test_game` skips without game data (ctest reports Skipped), so the reviewer needs their own MPQs and the 1.14d patch. A first run of my `test_combat` block on the shared `rng` shifted a later seeded block; the block uses its own `Rng` now. Not checked: the test failing on the old code (it cannot compile without `Blow::frac`; the old mean of 2.5 would miss 3.0 +- 0.03 by 17 x the tolerance).

## Effect on casts per kill (V-d2d, cmd/d2dcasts2)

NOT MEASURED (time box). The patched d2d is built and ready, but the lab lock (`/tmp/d2d-lab.lock`) was held by a 40 min soak plus a queue, so my 12 runs never started and I cancelled them. Before (V-d2d, docs/d2d-casts-real.md on agent/gaptime): real casts per Fallen kill 1.16 / 0.73 / 0.61 at Charged Bolt 1 / 4 / 8 (castgap 450, seeds 1-4, 15 virtual minutes). Expectation (U, arithmetic only): 17 % / about 12 % / 8 % more damage per bolt before resistance should cut casts per kill by up to that much (bounded by overkill and by whole-cast granularity), so roughly 1.0 / 0.65 / 0.57; the real number needs the run.

Patched build (private, throwaway, not the lab binary, lost on reboot of /tmp): source `git archive` of `~/d2d-lab/fork/wt-lab5` (lab5 217dd20) plus `git diff upstream/main -- components` of the PR branch, built in `/tmp/prbuild-fork-b` with `-DD2D_WERROR=OFF -DCMAKE_BUILD_TYPE=Release` (default WERROR failed on gcc 16 warnings), binary `/tmp/prbuild-d2d.new` (the fork tree and branches were not touched). Rebuild it the same way, then per level (clvl 2 / 5 / 9):

```sh
flock -w 3000 /tmp/d2d-lab.lock env D2D_BIN=<patched d2d> bash docs/probes/run-d2d.sh OUT "1 2 3 4" "d2dcasts2 -tables ~/d2lod-live/combat/data -area 2 -cb L -clvl C -castgap 450 -virtual 15m"
```

(`run-d2d.sh` and `d2dcasts2` are on agent/gaptime; the CSV columns and mana/cost formula are in docs/d2d-casts-real.md.)

## Draft PR body (their template)

````
## What changes
d2d cut each spell's rolled damage to whole hit points before taking it off a monster's life: a Charged Bolt of 2-4 dealt 2 or 3 (mean 2.5). game.exe rolls the element in 256ths, keeps monster life in 256ths and only kills below one point, so the mean is 3.0 (17 % more at Charged Bolt level 1, 8 % at level 8). The element roll in `missile_blow` now keeps its fraction in `Blow::frac`, and the monster carries it in `life_frac`.

Scope: a missile's element damage taken by a monster. Not changed: the physical part, melee, poison, pets and mercs, hit recovery, synergy (all untested for this claim).

## Evidence from game.exe 1.14d

| Address | What it shows |
|---|---|
| `FUN_005a89a0` | a missile's damage record: lightning is `min + rand(max - min)` in 256ths into record +0x1c, via `FUN_005a8910` (stats 0x32 / 0x33, mastery stat 0x14a) |
| `FUN_005ad730` | hands the record to `FUN_0057c6c0` unchanged (no rounding) |
| `LAB_0057c852` | record int 0x13, the total damage in 256ths, subtracted from life (stat 6, also 256ths) |
| `LAB_0057c865`, `LAB_0057c874` | life under 0x100 after the hit is set to 0: dead |
| `FUN_005adf10` | the unit-hit entry: no to-hit roll for a Missiles.txt row with ToHit 0 (Charged Bolt, row 56) |
| `FUN_004edea0` | the tooltip code; ranges and bolt counts equal d2d's (22,800 lines, zero diff), so only the fraction differs |

## How to re-check

```sh
python3 tools/pr_guard.py
cmake --build build && ctest --test-dir build --output-on-failure   # test_game needs D2_MPQ_DIR and D2_PATCH_INSTALLER
# the oracle (game.exe's own FUN_005a89a0 on 3000 random cases, equal to the port):
cd tools/emu && uv run python missile_roll.py 3000                    # prints ok  (script not part of this PR, see below)
cd tools/emu && uv run python skill_tips.py 20 0 > g.txt && ../../build/tests/test_game --skill-tips 20 0 | grep '^[0-9]' | diff g.txt -   # empty
```

Not done: the oracle script `missile_roll.py` is not in this PR (it would touch `tools/`, which needs the maintainer's CODEOWNERS review); I can send it as a second PR. Synergy on the real unit path and hit recovery are untraced. Siblings: pets and mercs still take whole points.

## Screenshots
Not applicable.

## Checklist
- [x] A test in `tests/` (fix or port)
- [ ] The finding in `docs/research/re/` (a note under "Applying damage" is in the diff; no todo.md row)
- [x] `python3 tools/pr_guard.py` passes

---

Agent: Claude Code (Claude Sonnet 5.5)
````

## Commands for the coordinator (NOT RUN; each needs the user's go)

```sh
ssh worker.example 'cd ~/d2d-lab/pr/gaps && git push origin fix/missile-damage-fine'
# save the body above (the text inside the four-backtick fence) to ~/d2d-lab/pr/gaps/PR_DESCRIPTION.md, then:
ssh worker.example 'cd ~/d2d-lab/pr/gaps && gh pr create --repo psi29a/d2decomped --base main --head shatteredeagle:fix/missile-damage-fine --title "A missile element keeps its 256ths" --body-file PR_DESCRIPTION.md'
```

Before the push: fetch `upstream` again and rebase if main moved; rerun pr_guard.
