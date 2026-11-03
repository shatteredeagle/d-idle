# Verifier: bundle 1 (`local/bundle-1`, ed087f88 on upstream main 48908176)

> status: verifier report (card verifybundle, 2026-11-02, Opus, one agent). Offline: no game, no d2d slot, nothing pushed or posted. Builds, oracles and tidy ran niced on the worker host at `-j2` at most, mostly one at a time. Harness and raw tables: [pr-bundle-1-verifier/](pr-bundle-1-verifier/) (`results.txt` holds every count below).

Tags: **V run** = run by this verifier (command in the harness dir); **V read** = read in the bundle diff or the repo by this verifier; **S** = an earlier doc of ours, not re-run; **U** = not verified. NOT RUN is said where a task was not done.

## Verdict: READY WITH CHANGES (provisional on the NOT RUN items)

No code change is needed. The ground component holds under an independent harness (base 476 of 3,726 hit-area cases differ from game.exe, head 0; burst equal losses 2 / 400 on base, 400 / 400 on head; game.exe rolls once per burst in 120 / 120 bursts). All five components are present unchanged on the bundle (patch-ids equal; two differ by rebase context only), the suites pass, warnings are clean on added lines, and the bundle merges cleanly with current upstream main. What blocks a push is text and process: commit bodies over the three-line rule, no Credits section and no `credit:` blocks, and a ground section in the PR body with no numbers. Nothing needs dropping or splitting out.

## Findings (ranked)

| # | sev | finding | evidence | recommendation |
|---|---|---|---|---|
| F1 | high | 7 of 10 commit bodies are over the upstream "body at most three lines" rule: 4f573c01 (6 lines), 1e344c02 (7), dabffee8 (10), 10afbc2d (8), b83f22a3 (7), ed087f88 (7); the four freeze commits have 2 lines but each is 172 to 246 characters, unwrapped. The PR draft says nothing about it | V run: `git log --format=%b` per commit, `Agent:` line excluded; rule: docs/process/upstream-pr-standard.md gate 7 and section 8, docs/process/coderabbit-prescan.md | reword each body to three lines or fewer, wrapped at 72 (the detail goes in the PR body). Rewording changes hashes only: re-run pr_guard and presubmit afterwards, no new differential |
| F2 | high | The PR body draft has no `## Credits` section, and no bundle card has a `credit:` block: `credit.py check` fails on every card of the bundle (prbuildprices2, prbuildskmelee3, prbuildskmelee3b, prbuildfreeze, skground, skground2, alignment2: "no credit: block"). No commit carries a `Traced-by:` line; whether a helper traced any component is not recorded anywhere | V run: `python3 tools/credit/credit.py check ...`; rule: docs/process/credit.md rules 4 and 5 | add the `credit:` blocks (coordinator), generate the section with `credit.py credits`, add `Traced-by:` lines only if a person traced a component |
| F3 | high | The ground section of the body draft has no counted differential ("at-scale counts ... still pending"), and the ground draft (agent/prbuildground, pr-ground-burst-draft.md) still shows placeholders (HITB_BASE, ROLLF_HEAD ...). Gate 1 of the PR standard (mismatches before > 0, after = 0, commands in the body) is unmet for this component in the text, although the claim holds | V read both drafts; this report's section 1 has the counts | put the counts of section 1 (or the author's own re-run) and the re-run commands into the body |
| F4 | medium | The footprint change reaches every standing row with a Size of 1..3, not only the two the commits and the body name (Blizzard shard, Fire Wall flame): Meteor's fire, Blaze, Immolation Arrow's fire, Eruption's cracks (Size 3), the poison / plague clouds, Shock Field, Volcano and more (189 Vel-0 rows with Size 1..3 in Missiles.txt). The four extra rows measured here (meteorfire, blaze, erruption crack 1, poisonexplosioncloud) match game.exe after and differ before, so the reach is right; it is undisclosed | V run (section 1); V read `standing_struck` callers: tick.cpp standing loop (also the shrine potion trigger and plain missiles), missiles.cpp do 5 | say in the commit body / PR body that it applies to every standing row with a Size, name the rows measured; list the shrine-potion and plain-missile paths as not measured (U) |
| F5 | medium | One roll per burst is applied to hit functions 1 and 13 only. Meteor (hit function 14) also rolls once in game.exe (skground-trace.md D5, V read / game side run there) and stays a roll per monster; Exploding / Freezing Arrow explosions (hit function 1 rows 656 / 657) now share the element roll, but a weapon share (`srcdam > 0`, `player_blow`) would still be drawn per unit; whether game.exe's record copies it is not measured | V read missiles.cpp `area` / `strike`; S skground-trace.md line 12 | Limits: name D5 (Meteor) as a known remaining deviation, and the srcdam case as unmeasured (U) |
| F6 | medium | Det scenario hashes base vs bundle: NOT RUN (needs lab builds of both and a d2d slot; this card has no exclusive resource). The body says "Not run" too | - | run det_scenarios_run.py on lab builds of 48908176 and ed087f88 before the push; expect changes only after a multi-monster Fire Ball / Glacial Spike burst, a standing missile at a footprint edge, a melee blow's fraction, an undead target, a Glacial Spike freeze |
| F7 | low | Full clang-tidy selection NOT RUN: on a test-only build `tidy-pr-select.py` falls back to every TU (1,051, deps unknown for unbuilt objects); the author's 228 needs a full build. The 8 changed .cpp TUs were tidied: 0 diagnostics in changed files (the 34 it prints are identifier-length on structured bindings in unchanged headers outdoor.hpp, merc.hpp, pets.hpp, clang-tidy 23.1.4) | V run | run the selected 228 TUs once on a full build, or accept the changed-TU pass |
| F8 | low | `Monster::half_freeze` is still set (ai.cpp, Blood Raven / the Countess) but nothing reads it after 71ea76f2 | V read: grep readers | fine as a documented keep; a reviewer may ask; one line in the body |
| F9 | low | upstream's tools/emu/freeze_rules.py docstring says "This change adds no d2d-side test binary for these cases, so `compare` has no d2d.tsv": stale once test_freeze_rules lands (8eba17d6 does not touch it) | V read | one-line docstring fix in 8eba17d6, or leave |
| F10 | low | The per-frame hit count of a standing missile is not part of any claim: on the head d2d hits a Fire Wall / Blaze / Meteor-fire victim on 4 or 5 of 6 frames (the placing frame and one more missing); the game harness's 6 of 6 rests on a hooked FUN_005adf10, so it is not a valid comparison (U) | V run (`results.txt` is sets only); counts in this section | keep the claim to the hit set, as the body does |

## 1. Ground (own harness, first job)

Built from the sources and the claim only; the author's sk_ground_hit.py / sk_ground_roll.py / test_sk_ground modes were not used or read before these counts. Different from the earlier D4 run (sk_gnd2_hit.py): another map seed (11), other site picks, monster classes 2 / 5 / 38 / 58 / 104 (size 2), 156 (size 3), 190 / 191 (size 1), four more rows, a 9 x 9 offset grid with the collision words restored around the unit.

- game.exe side `vb_ground_game.py` (upstream's emulator): a unit of the class at a free subtile, the row made by the real FUN_0059fa30 at an offset, stepped by its own server do (which runs FUN_005ae1f0), FUN_005adf10 hooked to record the unit. Burst mode: FUN_005adf10 NOT hooked; three hostile units within a subtile of a point; a Fire Ball (row 62) or Glacial Spike (row 96) is fired at them; code hooks count entries of FUN_005a9a70 / FUN_005aa8b0 (hit functions), FUN_005a8c70 (the record), FUN_005a8910 (element roll), FUN_005a89a0 (physical roll) and FUN_0057c6c0 (a record applied to a unit).
- d2d side `test_vb_probe.cpp` (scratch test, both builds, not for upstream): the real `World` / `Fight::world`, one monster with a million life and zeroed resistance, the row launched standing at the offset with a fixed 10-point damage; a hit = life lost in the frames. Burst mode: three `fallen1` at the point and +-1, a level-10 Fire Ball / Glacial Spike launched with Range 1 so it goes off there, the life each loses.
- builds: Release (asserts on), the bundle head ed087f88 and its base 48908176, separate worktrees, from clean.

| row (Size) | cases (class x offset) | base: differ from game.exe | head: differ | game.exe footprint s1 / s2 / s3 |
|---|---|---|---|---|
| blizzard1 159 (2) | 648 | 72 | 0 | 5 / 13 / 21 |
| firewall 69 (1) | 648 | 132 | 0 | 1 / 5 / 9 |
| meteorfire 240 (2) | 648 | 72 | 0 | 5 / 13 / 21 |
| blaze 67 (1) | 648 | 132 | 0 | 1 / 5 / 9 |
| erruption crack 1 462 (3) | 648 | 28 (4 only game.exe: base MISSES the 5 x 5 corners on a size-3 unit) | 0 | 9 / 21 / 25 |
| poisonexplosioncloud 217 (2) | 486 (no size-1 class on the game side) | 40 | 0 | - / 13 / 21 |
| total | 3,726 | 476 | 0 | |

Base d2d hits 21 subtiles for every size and row (half a cell). Every head set equals `missile_touches`: V run.

| burst | game.exe (60 bursts each, 3 units in reach) | d2d base (400 trials) | d2d head (400 trials) |
|---|---|---|---|
| Fire Ball (hit function 1) | FUN_005a9a70 1, FUN_005a8c70 1, element roll 1, applied to 3 units, in 60 / 60 | equal losses 2, unequal 398 | equal 400, unequal 0 |
| Glacial Spike (13) | FUN_005aa8b0 1, FUN_005a8c70 1, element roll 1, applied to 2 units, in 60 / 60 | equal 2, unequal 398 | equal 400, unequal 0 |

Break attempts: (a) a Size-3 row (Eruption), outside the claim's two rows: holds; (b) a size-3 unit, where base under-hits instead of over-hitting: fixed by the head; (c) rows the body does not name (Meteor fire, Blaze, a poison cloud): hold; (d) monster classes not used before: hold. Not tried: a monster off its subtile centre (d2d float positions; the author lists it as U), the shrine potion and plain-missile paths of the same loop, a missed (ToHit) first blow in a shared burst, Size-0 rows (keep the half cell; game.exe's rule for them U).

Comparison with the author (read after the counts): the earlier D4 run (sk-ground2-trace.md Q23) gives the same footprints for 159 and 69 (S, now V run here with other classes); the burst numbers agree in shape with sor-rest-trace.md Q5 (0 of 400 equal on d2d there, 2 of 400 here by chance ties). The author's at-scale counts do not exist yet (F3). No disagreement.

## 2. Integration on the bundle head

- Components: the five component branches' commits are all in the bundle, patch-ids equal for 8 of 10; dabffee8 vs 10e16ad5 differs by one blank line, b83f22a3 vs 42b1c9e0 by context lines only (V run `git patch-id`, interdiff). All component heads are on the private fork remote (V run `git ls-remote`).
- Shared code (V read): `strike` gets both the freeze rule (`monster_freeze`) and the shared roll; the freeze length is per unit and not rolled, so they do not interact. `missile_blow`'s new pointer defaults to null: every other caller (states.cpp, strike.cpp, pets.cpp, tick.cpp, missiles.cpp 873) is unchanged. `area` passes the roll only for hit functions 1 and 13 (callers at missiles.cpp 742 / 744; 743, 750, 758, 784 keep null). D1 / D2 change `player_blow`; inside a burst it is reached only for a `srcdam > 0` row (F5). D2 follows alignment2-trace.md: the bonus is in `player_blow`, so the player, the merc (its own `make_fighter`: Blunt and stat 122 from its gear) and the Shadow take it; pets build a `simple_fighter` (no Blunt, stat 122 = 0), matching the trace's "no visible effect" for units with no inventory.
- Counted differentials on the bundle head: ground (section 1) and freeze rules (section 2a), V run. Repair, D1, D2: their harnesses are the authors' emulator tools, not in the repo or the bundle; NOT RUN on the bundle head beyond the suites (their unit tests in test_combat / test_rules pass).
- Suites on the head (V run, Release, the 1.14d data): test_rules, test_game_rules, test_monsters, test_monster_thinks, test_monster_ai, test_combat, test_game, test_sk_ground: rc 0 each. test_freeze_rules, test_freeze_resume, test_armor_hit: rc 77 without a case file (oracle modes); test_freeze_rules with cases in 2a.
- Det scenarios: NOT RUN (F6).

### 2a. Freeze rules on the bundle head

V run with the existing oracle (upstream's tools/emu/freeze_rules.py `gen 30000 7717`: 125,748 boundary cases plus 30,000 random on a seed the author did not use), game.exe's FUN_0057c6c0 under the emulator, and the bundle's `test_freeze_rules --freeze-rules` on each build (on the base the bundle's test file was copied into the scratch worktree, the code under test untouched): **155,748 cases, base 129,904 differ, head 0** (`freeze_rules.py compare`, resfail 0; `vb-freeze.sh`, `vb-freeze-base.sh`). The freeze claim holds on the bundle head with the burst roll and the footprint in.

## 3. CI parity (bundle head)

- g++ 16.2.1 and clang++ 23.1.4, `-fsyntax-only` with the CI flags (-Wall -Wextra -Wpedantic -Wno-missing-field-initializers) plus -Wshadow -Wrange-loop-construct -Wconversion over the 8 changed .cpp TUs (which pull in every changed header): g++ 0 warnings; clang++ 206 warnings, none on an added line (the ones in changed headers are ai.hpp 860 / 919 / 920, -Wsign-conversion, outside the diff hunks). V run `vb_parity.py`.
- MSVC /W4 by reading (V read): no new local shadows another (`size` in the tick.cpp if-init is scoped to that `if`; `roll` in `missile_blow` stays in two disjoint blocks); every float/int conversion is an explicit cast; no `min` / `max` / `near` / `far` / `small` / `interface` / `IN` / `OUT` used as a new identifier (the existing local `max` in `repair_cost` is not followed by `(`, so windows.h's function-like macro does not expand it).
- clang-tidy: F7.
- pr_guard: 0 FAIL / 0 WARN on the range and on each of the 10 commits (V run).
- presubmit --quick against 48908176: 0 FAIL, 4 WARN: ruff (absent on that host; ruff 0.8.6 by hand: the two new tools clean, monsters.py 10 findings on base and on head alike), siblings, doc-claims store.md 208-209, build skipped. Attribution: ok (V run). The siblings and doc-claims answers in the draft stand.

## 4. Hygiene

- Author and committer of all 10 commits: `shatteredeagle <26069301+shatteredeagle@users.noreply.github.com>`; the only trailer is `Agent: Claude Code`; no co-author, AI or anthropic line, no emoji (V run).
- No private path, machine name, e-mail or tool name in the messages or the diff (V run grep). 
- Body length: F1. Credit: F2.

## 5. PR body draft (pr-bundle-1-draft.md, afef8cef)

- Repair, D1, D2, freeze numbers: as in their own drafts and verifier reports (S, not re-run here; freeze re-run in 2a: base 129,904 of 155,748 differ, head 0).
- Overclaims: "Author and committer ... trailer `Agent: Claude Code` only" true; "pr_guard 0 / 0" true; "presubmit 3 WARN" (4 here, the extra is ruff missing on the host). No claim of a number that did not run, except that section 5 says the hit areas are 5 / 13 / 21 and 1 / 5 / 9 without counts (F3).
- Limits: honest on det, MSVC and live; missing F4, F5 (Meteor's roll), F10.
- Credits section: missing (F2).

## 6. Size against the review budget

20 files, +759 / -42 = 801 changed lines, 10 commits: within 40 files, 1,000 lines and 15 commits (V run `git diff --shortstat`, `git rev-list --count`). CodeRabbit's default path filters: the exact list is not documented in coderabbit-limits-20261102.md (U); none of the 20 files is a lockfile, generated or minified file, so the count is not expected to change (S), and docs would only lower it.

## 7. Merge with current upstream main

`git fetch upstream main` (read-only): 82ea6db4 (the merge of #175), 3 commits past the base, touching components/game/objgroups.cpp and four docs/research/re notes, none in the bundle. `git merge-tree --write-tree upstream/main ed087f88`: clean (exit 0). V run. A rebase onto 82ea6db4 is not needed for the merge; rebuild after a rebase if the maintainer asks for one.

## Exact changes before the push

1. Reword the 7 commit bodies to three wrapped lines or fewer (F1); re-run pr_guard and presubmit.
2. Add `credit:` blocks to the bundle cards, a `## Credits` section from `credit.py credits`, and `credit.py check` exit 0 (F2).
3. Fill the ground section of the body with counted numbers and commands (F3) and name the rows the footprint reaches (F4).
4. Limits: Meteor's single roll (D5) not fixed, the srcdam case of a shared burst, the per-frame count (F5, F10).
5. Run the det scenarios on lab builds (F6) and, if wanted, the full tidy selection on a full build (F7).

## What I left on the worker host

Scratch worktrees `~/d2d-lab/verifybundle/{base,head}` (detached at 48908176 / ed087f88, each with the untracked scratch test test_vb_probe.cpp, the head with a `build` symlink and changed.txt / tidy_files.txt), builds `build-base`, `build-head`, outputs under `~/d2d-lab/verifybundle/`. No process left running (agentreap). Remove with `git -C ~/d2d-lab/fork/src worktree remove --force` on both when no longer needed.
