# Draft PR body: bundle 1 (local/bundle-1)

> status: DRAFT (card rebase174, 2026-11-02). NOT posted, nothing pushed to the public fork: the coordinator runs the final integration check first. Branch `local/bundle-1` on the private fork remote, head `ed087f88`, 10 commits on upstream main `48908176` (the merge of #174). Per-component detail and old heads: [rebase174-20261102.md](rebase174-20261102.md). Tags: V = run (how stated), S = secondary, U = unverified.

## Size against the review budget (V: `git diff --stat upstream/main...local/bundle-1`)

20 files, +759 -42, 10 commits (limits from the research note coderabbit-limits-20261102.md on branch agent/crlimits: files <= 40 target, 70 ceiling; lines <= 1000 target; commits <= 15). Within budget; no component needs to move to a second PR. Non-doc files: 17 (3 are `docs/research/re/*.md` notes). CodeRabbit's default path filters are not documented in the limits note (U): if docs are filtered the counted files drop to 17, which only helps. No `.coderabbit.yaml` upstream.

## Summary

Ten small fidelity fixes against game.exe 1.14d, each with a differential that runs game.exe's own functions under an emulator and compares them with the real d2d build on the same generated cases. One purpose per commit; the order below is the commit order.

| # | component | commits | files |
|---|---|---|---|
| 1 | Vendor repair gate | 1 | rules.hpp, gamedata_load.cpp, test_rules.cpp, store.md |
| 2 | Melee blow keeps its 256ths (D1) | 1 | combat.hpp, test_combat.cpp |
| 3 | Undead / Blunt bonus on a player blow (D2) | 1 | combat.hpp, ai.hpp, d2s_items.hpp, test_combat.cpp |
| 4 | Glacial Spike freeze, special monsters chilled, Blood Raven / Countess not halved, harness | 4 | combat.hpp, missiles.cpp, ai.cpp, ai.hpp, uniques.hpp, superuniques.md, test_freeze_rules.cpp |
| 5 | Standing missile footprint and one burst roll (ground) | 3 | combat.hpp, ai.hpp, fight.hpp, missiles.cpp, tick.cpp, skills.md, test_sk_ground.cpp, test_combat.cpp, two tools/emu scripts |

## 1. Vendor repair gate

game.exe's repair price (`FUN_0062efb0`) asks `FUN_0062e660` first: no price unless the ItemTypes row has Repair or the base has durability. d2d priced a drawn-durability bow below its max. Counted against game.exe: 7,908 gate cases (659 bases x 12 states) and 60,000 price cases. "Something to repair" where d2d prices and the game refuses: 143 on main, 0 on the head; "whole item": 17 -> 0; the 626 whole-item cases where the game prices 1 and d2d 0 are the known Repair All sentinel, unchanged. Multiplier cases above 0xffff: 0 differ on both; the 8 buy / 10 repair residuals below 0xffff are equal on both (harness, not the change). Limits: no magic stat costs or sockets in the comparison; with charges missing the game still adds the full base price, d2d charges the charges only (not claimed).

## 2. Melee fine physical (D1)

d2d floored the physical of a player blow to whole points before the resistance; game.exe keeps 256ths (`FUN_0057b420`, `FUN_0057bf80`) and the life stat takes the carry. Counted on landed blows with random weapons, enhanced damage, flat damage, Strength and Dexterity bonuses: 80,250 of 80,557 weapon cases differ in 256ths on main (seed 7; the seed 1 run reproduces the earlier count), 0 on the head. Second harness (own cases, seeds 23 and 5, resistance 0 / 25 / -50 / 75): about 24,000 of 24,040 differ on main in every run, 0 on the head for armed blows with gear percent -90 or more. Limits: barehanded blows (Strength term) and the -90 floor applied twice are unchanged and still differ; kicks, Smite and conversion paths share the code but were not counted (U).

## 3. Undead / Blunt bonus (D2)

Against a MonStats undead (lUndead or hUndead) `FUN_0057b660` adds 50 for a Blunt weapon plus stat 122 to the enhanced damage, only when the sum is over 0, for a player attacker or alignment 2. d2d had no undead input. Counted whole points on undead targets: 38,402 differ on main over two seeds, 0 on the head; the second harness, 60,000 rows: whole-point differences 24,439 -> 8,811 (seed 23) and 24,585 -> 8,911 (seed 5), the remainder being the one-point floor (a component 2 matter), barehanded blows and the -90 floor; armed, gear >= -90, not floor: 13,250 -> 0 and 13,316 -> 0. With D1 and D2 together (this bundle) the second harness shows 0 armed, gear >= -90 differences in 256ths in all eight runs. Limits (U): whether a hired merc or a Shadow has alignment 2 is read from the disassembly, not live; stat 122 on a shield is summed with the rest (the game takes the unit's own stat).

## 4. Freeze rules

`FUN_0057b230` resists and divides a Glacial Spike freeze, chills (never freezes) a special monster, and does not halve Blood Raven's and the Countess's freezes. Counted: `freeze_rules.py` runs `FUN_0057c6c0` natively; `test_freeze_rules --freeze-rules` runs the real `Fight::strike`; 225,748 cases (125,748 boundary cases over every boss kind, resistance -300..150, freeze length -5..250, difficulty, plus 100,000 random): 185,746 differ on main (with the test file), 0 on the head. Limit: a freeze length above 0x100000 differs in a separate harness (6 of 16 cases; unreachable in play, S).

## 5. Standing missile footprint and the burst roll (ground)

A standing Blizzard shard (Size 2) hits a size 1 / 2 / 3 monster on 5 / 13 / 21 subtiles in game.exe, d2d on 21 for all; Fire Wall (Size 1) 1 / 5 / 9. A Fire Ball or Glacial Spike burst rolls its damage once for the whole burst in game.exe, d2d once per monster. `test_sk_ground` checks both and fails without the two code commits. The at-scale counts of this component belong to its own draft (pr-ground-burst-draft.md, still pending numbers there); only `test_sk_ground` was re-run on this bundle (rc 0). Limits: the shared roll shifts later draws against main (determinism, expected); rows without a Size keep the half cell (U).

## Checks (V, worker host, the bundle head)

- The seven suites (test_rules, test_game_rules, test_monsters, test_monster_thinks, test_monster_ai, test_combat, test_game) with the game data, plus test_sk_ground: rc 0. Release, `-DD2D_WERROR=ON`.
- g++ and clang++ with the CI flags plus `-Wshadow -Wrange-loop-construct -Wconversion` (syntax only) and clang-tidy over the 12 changed or including TUs: 0 diagnostics on added lines; `tidy-pr-select` picks 228 TUs for the whole bundle.
- `pr_guard` 0 FAIL 0 WARN on each of the 10 commits; `presubmit --quick` 0 FAIL, 3 WARN (below), including the attribution check (no co-author or AI lines in the messages). Author and committer on every commit: the contributor's numeric noreply identity; trailer `Agent: Claude Code` only.
- Not run: determinism scenario hashes, MSVC /W4 (by reading: no `min`, `max`, `near`, `far` identifiers, no narrowing without a cast), a live game.

## Answers to the presubmit WARNs

- siblings (merc, pet, partner): component 2 and 3 apply to every caller of `player_blow`, so a merc's blow and a Shadow's blow take the same rules; component 5 touches monsters only (a monster's missile on a player keeps its own test, U).
- doc-claims `store.md` lines 208-209 ("all", "never"): true for vanilla data (186 of 186 bases with no Repair have nodurability 1) and the text says so.
- build: skipped by `--quick`; built by hand above.

## How to re-run

`test_freeze_rules --freeze-rules cases.tsv` against `freeze_rules.py game|compare`; `test_sk_ground`, `--hit ROW FRAMES PER`, `--burst-roll NAME TRIALS` against `sk_ground_hit.py` / `sk_ground_roll.py`. The repair and melee harnesses are the author's emulator tools (not in this PR); their case files can be attached.
