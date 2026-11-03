# Fidelity fixes against game.exe 1.14d: vendor repair gate, melee blow, freeze rules, standing missiles

> status: FINAL BODY, not posted (cards bundlefinal and crfixbundle, 2026-11-02). Nothing is pushed to the public fork and no PR is open: the user decides. Branch `local/bundle-3` on the private fork remote (10 commits on upstream main 82ea6db4; it replaces `local/bundle-2` after CodeRabbit's review of the staging PR, see [crfix-bundle-20261102.md](crfix-bundle-20261102.md)); `stage/bundle-3` is the same ten commits on a base that carries the fork's `.coderabbit.yaml`, for a staging PR. Everything below the line is the PR text. Tags: V = run (how stated), S = secondary (an earlier run of ours, not re-run for this bundle), U = unverified.

---

## Summary

Five small fixes where d2d differs from game.exe 1.14d, ten commits, one purpose per commit. Each has a differential: game.exe's own functions run natively under the emulator in `tools/emu`, the real d2d build runs the same generated cases, and the two are compared. Size: see `git diff --shortstat upstream/main...local/bundle-3` (bundle 2 was 20 files, +759 / -42; bundle 3 adds the repair price below, tags and a test).

| # | component | commits |
|---|---|---|
| 1 | Vendor repair gate | 1 |
| 2 | Melee blow keeps its 256ths | 1 |
| 3 | Undead / Blunt bonus on a player blow | 1 |
| 4 | Freeze rules (Glacial Spike resist and divide, special monsters chilled, Blood Raven and the Countess not halved) plus the harness | 4 |
| 5 | Standing missile footprint and one burst roll, plus the harness | 3 |

## 1. Vendor repair gate

game.exe's repair price (`FUN_0062efb0`) asks `FUN_0062e660` first: no price unless the ItemTypes row has Repair or the base has a durability (`FUN_00629930`). d2d priced a drawn-durability bow below its max. Counted against game.exe: 7,908 gate cases (659 bases x 12 states) and 60,000 price cases (S, earlier independent run). "Something to repair" where d2d prices and the game refuses: 143 on main, 0 on the head; "whole item": 17 to 0; the 626 whole-item cases where the game prices 1 and d2d 0 are the Repair All sentinel and are unchanged. Multiplier cases above 0xffff: 0 differ on both. With charges missing the gate passes on them alone, and game.exe then scales the base by the durability only when `FUN_00629930` holds (`0062f9e7`): an item without durability pays its whole base, a durable one its missing share. d2d now does the same (a review finding); counted with the same oracle, the base part (the durability scaling, then the npc mult block), 3,000 cases over every non-stacking base with charges missing (V: 2,202 priced): 1,527 differ on bundle 2, 0 on bundle 3. Limits: no magic stat costs or sockets in the comparison; stacking items were not in the 3,000 (their restock path is unchanged). In vanilla data all 186 bases of the 66 types without Repair have nodurability 1, so the type half only matters for mods.

## 2. Melee blow keeps its 256ths

d2d floored the physical of a player blow to whole points before the resistance; game.exe keeps 256ths (`FUN_0057b420`, `FUN_0057bf80`) and the life stat takes the carry. Counted on landed blows with random weapons, enhanced damage, flat damage and attribute bonuses (S, earlier independent run): 80,250 of 80,557 weapon cases differ in 256ths on main (seed 7), 0 on the head. A second generator (seeds 23 and 5, resistance 0 / 25 / -50 / 75): about 24,000 of 24,040 differ on main in every run, 0 on the head for armed blows with gear percent of -90 or more. Limits: barehanded blows and the -90 floor applied twice are unchanged and still differ; kicks and Smite share the code but were not counted (U).

## 3. Undead / Blunt bonus on a player blow

Against a MonStats undead (lUndead or hUndead) `FUN_0057b660` adds 50 for a Blunt weapon plus stat 122 to the enhanced damage, only when the sum is over 0, for a player attacker or alignment 2. d2d had no undead input. Counted whole points on undead targets (S): 38,402 differ on main over two seeds, 0 on the head. With components 2 and 3 together the second generator shows 0 armed, gear >= -90 differences in 256ths in all eight runs. Limits (U): whether a hired merc or a Shadow has alignment 2 is read from the disassembly, not run live; stat 122 on a shield is summed with the rest of the gear.

## 4. Freeze rules

`FUN_0057b230` resists and divides a Glacial Spike freeze, chills (never freezes) a special monster, and does not halve Blood Raven's and the Countess's freezes. `freeze_rules.py` runs `FUN_0057c6c0` natively; `test_freeze_rules --freeze-rules` runs the real `Fight::strike`. Author's run (S): 225,748 cases (125,748 boundary cases over every boss kind, resistance -300..150, freeze length -5..250, difficulty, plus 100,000 random): 185,746 differ on main (with the test file), 0 on the head. The freeze code is unchanged from bundle 2 to bundle 3 (the review fixes touch the repair price, three doc files, one test and one tool). Re-run on the bundle head with a new seed, by the independent verifier on the earlier base and again on `local/bundle-2` (V, 155,748 cases, 125,748 boundary plus 30,000 random): 129,904 differ on main (verifier), 0 on the head (both runs), with the burst roll and the footprint of component 5 in. Limits: a freeze length above 0x100000 differs in a separate harness (6 of 16 cases; unreachable in play, S). `Monster::half_freeze` is still set but nothing reads it any more.

## 5. Standing missile footprint and one burst roll

**Footprint.** A standing missile tests its own subtile against the monster each frame (`FUN_005ae1f0`, `FUN_00641cb0`). In game.exe a Size-2 shard hits a size 1 / 2 / 3 monster on 5 / 13 / 21 subtiles and a Size-1 flame on 1 / 5 / 9; d2d hit 21 subtiles for every size. `standing_struck` now uses `missile_touches`, as a moving missile does, for every standing row with a Size of 1 to 3: that is not only the Blizzard shard and Fire Wall flame the tests name, but every such row (Meteor's fire, Blaze, Immolation Arrow's fire, Eruption's cracks, the poison and plague clouds, Shock Field, Volcano and more; 189 rows with velocity 0 and a Size of 1 to 3 in Missiles.txt). A row without a Size keeps the half cell.

Counted (V, an independent harness, a 9 x 9 offset grid, six rows, 8 monster classes of size 1 to 3, game.exe's `FUN_0059fa30` and its own server step against the real d2d `World`): 3,726 hit-area cases; base differs from game.exe in 476, the bundle head in 0. Per row, cases / base differ / head differ: Blizzard shard 648 / 72 / 0, Fire Wall 648 / 132 / 0, Meteor fire 648 / 72 / 0, Blaze 648 / 132 / 0, Eruption crack 648 / 28 / 0 (base misses the 5 x 5 corners of a size-3 monster in 4 of them), poison explosion cloud 486 / 40 / 0. Four of the six rows are not named by the commits. Author's own run (S): 225 offsets x 3 classes, shard: offsets only d2d hits 16 / 8 / 0 for size 1 / 2 / 3 (game.exe 5 / 13 / 21 of d2d's 21); Fire Wall 20 / 16 / 12 (game.exe 1 / 5 / 9). Every head set equals `rules::missile_touches` (V).

**Burst roll.** `FUN_005a9a70` (hit function 1, Fire Ball) and `FUN_005aa8b0` (13, Glacial Spike) call `FUN_005a8c70` once and hand a copy of that record to every unit; each unit resists its own copy. d2d drew a roll per monster. `missile_blow` takes an optional `MissileRoll` that the first blow fills and the others reuse; `area` passes one for hit functions 1 and 13 only. Counted (V, three monsters within a subtile, 60 bursts per spell on the game.exe side, 400 trials on the d2d side): game.exe makes one record and one element roll in 120 of 120 bursts (60 Fire Ball, 60 Glacial Spike); d2d on main gives three equal losses in 2 of 400 trials for each spell (chance ties), on the head in 400 of 400.

Limits: Meteor (hit function 14) also rolls once in game.exe and still rolls per monster here: not fixed. Exploding and Freezing Arrow explosions now share the element roll; a weapon share (`srcdam > 0`) of a shared burst would still be drawn per unit, and whether game.exe copies it is not measured (U). The per-frame hit count of a standing missile is not part of the claim, only the hit set. A monster off its subtile centre (d2d positions are floats), the shrine-potion and plain-missile paths of the same loop, and Size-0 rows are not measured (U). The shared roll shifts later draws against main (a determinism change, expected).

## Credits

- Vendor repair gate: traced by Claude Code; written by the PR author; independently verified by a second Claude Code run (own harness).
- Melee fine physical (D1) and undead / Blunt bonus (D2): traced by Claude Code; written by the PR author; independently verified by a second Claude Code run (own harness).
- Freeze rules: traced by Claude Code; written by the PR author; independently verified by a second Claude Code run (own harness; freeze re-run on the bundle head).
- Standing missile footprint and burst roll (ground): traced by Claude Code; written by the PR author; independently verified by a second Claude Code run (own harness).

No other contributor's work is in this bundle. Only the git author and the Reported-by / Traced-by lines count for the gap map; verifiers are named here only.

## Checks

- Commits: 10, subject line plus a body of at most three lines (the last is `Agent: Claude Code`), author and committer the numeric noreply identity, no co-author or AI lines (V: `git log`, `tools/presubmit.py --quick` attribution check ok).
- Suites on the head (V, Release, the 1.14d game data, this bundle rebuilt on current upstream main 82ea6db4): test_rules, test_game_rules, test_monsters, test_monster_thinks, test_monster_ai, test_combat, test_game, test_sk_ground: rc 0 each (rerun on bundle 3). `test_freeze_rules`, `test_freeze_resume` and `test_armor_hit` are oracle modes and skip (rc 77) without a case file; `test_freeze_rules --freeze-rules` was run on 155,748 cases (section 4): 0 differ.
- `tools/pr_guard.py`: 0 FAIL, 0 WARN on the range and on each of the 10 commits. `tools/presubmit.py --quick`: 0 FAIL, the attribution check ok; WARN only for ruff (not on the PATH of the check host; ruff 0.8.6 by hand on the changed tool: clean), siblings and doc-claims (answered below) and the skipped build.
- g++ 16 and clang++ 23 with the CI flags plus `-Wshadow -Wrange-loop-construct -Wconversion` (syntax only, S: run on the same patch on the earlier base): g++ 0 warnings; clang++ has no warning on an added line.
- Merge: `git merge-tree` of the head with upstream main reports no conflict (V).
- Not run: the determinism scenario hashes on lab builds of the base and the head (expected to change only after a multi-monster Fire Ball or Glacial Spike burst, at the footprint edge of a standing missile, for a melee blow's fraction, against an undead, and after a Glacial Spike freeze); MSVC (read for shadowing, `min` / `max` / `near` / `far` identifiers and unchecked narrowing only); the full clang-tidy selection on a full build (the eight changed .cpp files were tidied: 0 diagnostics in changed files); a live game.
- Independent re-runs behind the repair and melee numbers (S, not re-run for this bundle, their harnesses are not in this PR and can be attached): repair `vpr_oracle.py` with `vpr_probe.inc` and `vpr_cmp.py`; melee `vs3_game.py` with `vs3_d2d.cpp` and `vs3_cmp.py`. The ground numbers above come from a harness written separately (`vb_ground_game.py`, `test_vb_probe.cpp`), also not part of this PR.

## Answers to the presubmit warnings

- siblings (merc, pet, partner): components 2 and 3 apply to every caller of `player_blow`, so a merc's blow and a Shadow's blow take the same rules; a pet builds a simple fighter without Blunt or stat 122; component 5 touches monsters only.
- doc-claims `store.md`: "all" and "never" are true for vanilla data and the text says so.
- ruff 0.8.6 by hand on the two new Python tools: clean (S).

## How to re-run

`test_freeze_rules --freeze-rules cases.tsv` against `freeze_rules.py gen|game|compare`; `test_sk_ground`, `test_sk_ground --hit ROW FRAMES PER` and `--burst-roll NAME TRIALS` against `sk_ground_hit.py` and `sk_ground_roll.py`.

Agent: Claude Code
