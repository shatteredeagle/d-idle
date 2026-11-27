# Upstream PR 148 (directory per subsystem): impact on the private fork

> status: archived 2026-11-27 (finished one-off report, TASK-117); stub left at docs/research/pr148-impact.md

## What #148 changes
Moves only, bodies unchanged: `fight.cpp` into `fight/` (10 files), `world.cpp` into `world/` (+ `world/quests/`), `apps/d2d/town.cpp` into `town/` (9 files), `devctl_verbs.hpp` into `devctl/verbs.hpp` with each `debug` sub-verb in `devctl/debug_*.cpp`, `test_game.cpp` and `test_monsters.cpp` into `tests/game/`, `tests/monsters/`. Pure renames (git follows them): `gamedata*`, `npc_*`, `character*`, `world_quests_actN.cpp`. Also: `.gitattributes` union merge, one-file-a-line CMake list, tests found by glob (test names and `build/tests/test_game` unchanged), 168 files in all (many only a header comment).

## The lab patch set
Only one branch is live: the click-modifier branch (it contains the lab6 set): 13 commits, 14 files, 42 hunks. The older lab..lab5 branches are earlier iterations of the same work. The fork is already 73 upstream commits behind the base #148 sits on, so a rebase is needed with or without #148.

| Area | Hunks | Result of the scripted dry run |
|---|---|---|
| `devctl_verbs.hpp` (split, 9 hunks) | 6 outside `debug` | clean: `git apply --3way` with the path rewritten to `devctl/verbs.hpp` |
| same, inside the `debug` verb (wp, layout, nearby, usage text) | 3 | FAIL, hand port into `devctl/debug_level.cpp` and `debug.cpp` (about 40 lines, mechanical) |
| `town.cpp` (2 hunks) | 2 | clean: both land in `town/update.cpp` (`Town::update` stayed one function), path rewrite plus `git apply` |
| `gamedata.*` (renamed, 3 hunks) | 3 | clean (R100) |
| `main.cpp`, `apps/d2d/CMakeLists.txt` (not split, #148 edits the same lines) | 2 | trivial conflicts (include line, source list); also conflict on a plain rebase |
| 8 untouched files, 3 new files | 28 | clean; one new fork header still includes the old `devctl_verbs.hpp` name (fix by hand) |

Control: the 13 commits cherry-pick cleanly onto #148's base without the split. Per commit onto the split tree, about 4 of 13 would stop; port the net diff once instead.

Build: after the port, every touched file compiles and `test_rules` passes. The full link was not reached: the worker's newer GCC fails `-Werror` in two files #148 and the patch set do not touch (`ui.cpp`, `town/walk.cpp`); not checked against a pristine tree. Not run live.

## Other work affected
| Item | Split-file hunks | Result |
|---|---|---|
| PR 147 (ours: belt pickup oracle) | 1 in `test_game.cpp` | cherry-picks clean on #148 (git follows the 79 % rename); no action, merges in either order |
| Local tome branch (3 commits) | `fight.cpp` 2, `world.cpp` 6, `test_game.cpp` 2 (+ `world.hpp` 3, `fight.hpp` 2) | commit 1 clean, commits 2 and 3 conflict in 4 files: about 10 hunks by hand |
| Stale upstream-PR branches (cast timing, vendor prices) | not measured | one commit each, 95 commits behind: rebase on main anyway |

## Our docs and tools
Citations of moved paths (grep, `docs/` and `tools/`): `fight.cpp` 60 + 18, `world.cpp` 31 + 4, `town.cpp` 21 + 7, `test_game.cpp` 10, `gamedata` 9 + 8, `test_monsters` 3, `devctl_verbs` 1. Almost all are `file:line` comments in evidence packs, research notes and `tools/emu/*` harness headers, pinned to an upstream commit, so they stay true as history. Nothing breaks: the harnesses include no `components/game` headers, call `build/tests/test_game` by target name (unchanged), and only `tools/riskmap/dependencies.json` (11 path entries, `fight.cpp`, `world.cpp`, `gamedata.cpp`) needs its paths updated after the merge.

## Effort
Click-modifier branch: about 1.5 h (script the net-diff apply 0.5 h, 3 debug hunks 0.5 h, main/CMake/include and compile check 0.5 h). Tome branch: about 1 h. riskmap paths: 0.25 h. Total about 3 h, once; a rebase without #148 costs about 1 h of it anyway.

## Recommendation (no posting)
1. Do not hurry the fork before #148 lands: porting twice is the only avoidable cost. Wait for the merge, then rebase the fork once onto the new main.
2. Order: refresh the base, apply the net diff of the click-modifier branch with the path rewrite (`devctl_verbs.hpp` to `devctl/verbs.hpp`), hand-port the 3 debug hunks, fix the include, compile; then the tome branch; then riskmap paths.
3. Keep patch-set changes in few, small hunks; a new fork verb goes in its own file (as #148's layout intends), which removes the problem for good.
4. Nothing to tell the maintainer: the split costs the fork about 3 hours and gives merge-conflict relief upstream. If #148 is still open when the fork next needs a rebase, port onto its head instead (same script).

update 2026-10-31: #148 merged 16:41Z; riskmap dependencies updated; fork port pending the user go

Note 2026-10-31 (UPSYNC147): the belt PR #147 was brought up to date by merging upstream/main into its branch; git followed the `tests/test_game.cpp` move to `tests/game/` with no conflict and #147 shows mergeable. Detail: docs/research/pr-belt-evidence-pack.md.

update 2026-10-31: the lab patch set is ported to the restructured upstream, builds and passes the smoke checks
