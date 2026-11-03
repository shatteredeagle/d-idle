# PR draft D1: a player blow keeps its 256ths through the resistance (SKMELEE3, part 1)

> status: DRAFT, 2026-11-02 (card prbuildskmelee3b). ON HOLD: all upstream PRs are on hold; nothing is posted, nothing is pushed to a public remote. Branch `local/skmelee3-d1-r174` on the private fork remote, head 5689c007, one commit on upstream main 48908176 (the merge of #174; the earlier `local/skmelee3-d1-final` 46ced89a was on cb716f10; counts re-run: rebase174-20261102.md). Replaces the single draft of card prbuildskmelee3 (D1 and D2 are now two independent PRs; the D2 text is pr-skmelee3-d2-draft.md). Tags: V = run, S = from an earlier doc, U = unverified. Four older branch names of the first attempts (`local/skmelee3-d1`, `-d1-v2`, `-d2`, `-d2-v2`) are stale and must not be used (the first and third have a failing or missing test, the others carry a stray file).

## PR text (for upstream; no private names)

**Title:** A player blow keeps its 256ths through the physical resistance

**Body**

`FUN_0057b420` returns the physical damage in 256ths, `FUN_0057bf80` resists that fine value, and the life stat takes it as is (`0057c852`). `player_blow` shifted the roll to whole points and raised it to 1 before the resistance, so a blow lost up to a point (0.5 on average) and a sub-point blow became a full point. The part under a point now stays in `Blow::frac`, as `missile_blow` already keeps it, and the one-point floor is gone. RNG draws are unchanged. One commit, `combat.hpp` and `tests/test_combat.cpp` only.

This also changes every caller of `player_blow`: the player's Attack, a merc's blow, a Shadow's blow, kicks and Smite, and the weapon share of a skill missile (`Fight::strike` calls `player_blow` with `srcdam`, then `missile_blow` adds the element on top, and `frac` can then pass 255; `hurt()` takes any `frac`). The missile path is covered by a new test; kicks and Smite are not counted against the game (U, see limits). The comment on `Blow::frac` says so now.

Counted differential against the original game (game.exe 1.14d's own functions `FUN_0057b660` and `FUN_0057b420` run under an emulator, the same cases on both sides, a landed blow is compared on both). Two independent case generators:

| generator, cases | compared (landed, armed, target not undead) | base differs in 256ths | this head differs |
|---|---|---|---|
| A, seed 7, 100,000 + undead grid | 112,941 | 112,486 | 0 |
| A, seed 1, 100,000 + undead grid | 113,288 | 112,840 | 0 |
| B, seed 23, 60,000 cases x resistance 0, 25, -50, 75 | 24,040 x 4 | 23,954 / 24,000 / 24,001 / 23,964 | 0 (armed, gear percent -90 and up), all four |
| B, seed 5, same | 23,793 x 4 | 23,722 / 23,760 / 23,756 / 23,717 | 0, all four |

Cases where the build rolls a miss are excluded on both sides. The mean gap on the base is +0.498 points a landed blow. The one-point floor lifted a sub-point blow to 1 in 1,194 (seed 23) and 1,154 (seed 5) of those cases at resistance 0 (V run).

Not changed and still different (counted on this head; none is touched here): barehanded blows (Strength as a damage percent, the 1 / 2 range: about 14,000 of 95,000 per seed), the -90 gear percent floor applied twice with a skill (about 1,150 per 24,000 in B), a skill's off-weapon enhanced damage, to-hit with an attack rating at or under 0.

Checks (worker host, build -j2 niced): the seven suites (test_rules, test_game_rules, test_monsters, test_monster_thinks, test_monster_ai, test_combat, test_game with the data) pass on the head; `pr_guard` 0 FAIL 0 WARN; `presubmit --quick` 0 FAIL, 2 WARN (docs-sync: no doc names this rule; build skipped by --quick); g++ and clang++ with the CI flags plus `-Wshadow -Wrange-loop-construct -Wconversion`, syntax only, over test_combat, strike, missiles, pets, merc and ai: 0 diagnostics on added lines; clang-tidy (CI's step): `tidy-pr-select` picks 227 TUs, the six above were run: 0 findings in the touched files (the 36 findings are `x` / `y` bindings in untouched `outdoor.hpp`, also on upstream main); no windows.h macro names or shadowed locals in the diff (read; the test locals are `exposed`, `fine`, `whole`, `sliver`, `share`).

Overlap with open PRs: 174 changes `player_blow` (Crushing Blow) and `Target`; a trial merge with its head 489ef83d is clean, builds and passes test_combat and test_rules (V). Semantic note: 174's Crushing Blow takes the life left as `hit_points - phys - elem` in whole points; with this PR `phys` is the whole part of the resisted fine value and `frac` is not subtracted, so the life seen by Crushing Blow is up to one point higher than the game's 256ths life if the game reads it in 256ths (U; the effect on a divisor result is under one point of life / 4).

Determinism scenarios: not run (U). Expected: every scenario with a player, merc or Shadow melee hit changes its monster-state hash by the carried part under a point; passive, cold, static and cold2 do not. Checklist: build two lab trees (upstream main plus the lab patch set) for the base and for this head, run `det_scenarios_run.py` on both with a repeat of the base, attribute each differing hash to the carry.

## Limits (U)

- Kicks (`pf_kick`) and Smite take the same fine change (same code path) but their game side was not counted.
- The fine value assumes the game's resistance path is the one `resisted_fine` already states for missiles (V read in an earlier trace).
- Mercs and Shadows use the player's blow through the same function; no merc-specific counts.

## Evidence (worker host, V unless said)

- Branch `local/skmelee3-d1-r174`, base 48908176 (upstream main after #174, 2026-11-02), commit identity the contributor's noreply address for author and committer, trailer `Agent: Claude Code` only.
- Generator A: `tools/emu/sm3_run.sh` flow (`sk_melee2.py gen 100000 SEED 1700`, `game`, build's `test_melee --melee`, `cmp`); d2d md5 head seed 7 c0dff1b9, seed 1 5819013b; base eba407be / 1809103f.
- Generator B: the independent verifier's harness (own cases, game side from `FUN_0057b660` + `FUN_0057b420`, d2d side the real `rules::player_blow` of each tree's header compiled in place; the game side resisted as `resisted_fine` states it).
- Suites and tidy logs: worker host, `sm3b-out/`.

## Shared-file lines for the merger

See openspec/changes/prbuildskmelee3b/merge-lines.txt.
