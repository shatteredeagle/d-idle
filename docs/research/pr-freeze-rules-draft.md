# Draft PR body: freeze rules (Glacial Spike's freeze on monsters)

> status: DRAFT for the fork branch `local/freeze-rules-v4` (four commits on upstream main 48908176 (the merge of #174; earlier shape local/freeze-rules-v4 on cb716f10), which already contains #159; 8 files, +166 -9; supersedes `fix/freeze-rules-v2`, `local/freeze-rules-v3` and the stale [pr-freeze-draft.md](pr-freeze-draft.md)). Rebuilt and re-run on the final head on 2026-11-02 (numbers below). Not posted; all upstream PRs are on hold (user, 2026-11-02); it needs one independent re-check of the final head before it is opened. The maintainer is expected to merge more soon: re-rebase (`git rebase --onto upstream/main cb716f10 local/freeze-rules-v4`), re-check and re-run before opening. Evidence trail: [pr-freeze-proof.md](pr-freeze-proof.md), [freeze-rules-trace.md](freeze-rules-trace.md). CI parity and prep: the section "Prep notes" at the end.

---

**Title:** Glacial Spike's freeze follows FUN_0057b230

### What

Glacial Spike's freeze on a monster now follows game.exe's FUN_0057b230:

- The freeze length is cut by cold resistance less cold pierce, the same way the chill length is (row 6 of the element table at 0x732980, next to row 5).
- On a MonStats boss, a champion, a unique, a superunique or an act boss (MonStats boss column, or monster-data flag 8) the freeze becomes a chill for that length. They are never frozen.
- Any other monster is frozen only when its coldeffect is negative. The freeze lasts length / MonsterFreezeDivisor, and the monster gets no chill from it.
- Blood Raven's and the Countess's freezes are no longer halved. The call d2d read as stat 0x76 is `FUN_00639db0(unit, 0x76, 1)`, the state setter (state 118, corpse_noselect); it changes no stat.

Before this, d2d stood every monster for the raw length at any coldeffect and also chilled it.

Four commits, one claim each (resist / divide / coldeffect; special monsters chill; no halving; tools). Files: `components/rules/combat.hpp` (`Freeze`, `monster_freeze`), `components/game/fight/missiles.cpp` (`Fight::strike`), comments in `ai.cpp`, `ai.hpp`, `uniques.hpp`, `docs/research/re/superuniques.md`, `tools/emu/monsters.py`; new `tests/game/test_freeze_rules.cpp` (the oracle `tools/emu/freeze_rules.py` and `GameData::freeze_divisor` came in with #159).

### Proof

`tools/emu/freeze_rules.py` runs game.exe's FUN_0057c6c0 natively under the emulator, including FUN_0057c1e0, FUN_0057af80, FUN_0057b230 and the boss, flag-8 and hireling tests. `test_freeze_rules --freeze-rules` runs the real `Fight::strike` on the same cases. 225,748 cases, seed 20261101 (125,748 boundary cases over every boss kind, boss column, half_freeze, coldeffect, resistance -300..150, freeze length -5..250 and difficulty, plus 100,000 random):

| build | cases that differ from game.exe |
|---|---|
| upstream main cb716f10 (test file added) | 185,746 of 225,748 (plain 48,946, special 78,501, half_freeze 58,299) |
| this branch | 0 of 225,748 |

```sh
cd tools/emu
uv run python freeze_rules.py gen 100000 20261101 > cases.tsv
uv run python freeze_rules.py game cases.tsv > game.tsv
../../build/tests/test_freeze_rules --freeze-rules cases.tsv > d2d.tsv
uv run python freeze_rules.py compare cases.tsv game.tsv d2d.tsv
```

`freeze_rules.py flags` sweeps every monster-data flag byte against the boss column and a hireling class: only flag 8, the boss column or a hireling turns the freeze into a chill. `freeze_rules.py state` runs the Blood Raven / Countess call natively: it sets state 118, writes no stat, and stat 0x76 then reads 0.

A second harness, written separately (its own generator and seed, 0x1F00DFACE) and re-run on this head: 146,591 cases, 0 differ in every class (plain, special, half_freeze) except 6 of 16 with a freeze length above 0x100000 (the residual above).

Det state hashes: not run.

### Limits

- Only Glacial Spike's freeze. Frozen Armor goes through the same FUN_0057b230 rule in game.exe (FUN_005cac50 sends it through FUN_0057c6c0) and is the subject of #159. Freezing Arrow, Ice Blast and item freeze are not modelled in d2d; Frozen Orb and Blizzard only chill (no freeze). All are left for separate PRs.
- Which monsters chill (read from game.exe's own spawn path, 12,830 monsters): flag word 0x0d champions, 0x2d possessed, 0x4d ghostly, 0x09 uniques, 0x0b superuniques and 0x08 act bosses are chilled; minions (0x10) and plain monsters (0x00) are frozen. Only bit 8 matters; hirelings are always chilled.
- Not modelled: state 0x36 (no freeze while it is set), the freeze's events, and a length / divisor of 0 (game.exe sets state 1 for no frame).
- Residual: a freeze length of exactly 0x100001 with a nonzero resistance differs (game.exe's MulDiv divides first above 0x100000, d2d multiplies); unreachable in play.
- A freeze is still d2d's stand (`stun_until`).
- `half_freeze` keeps its name and now only marks the state-118 call.
- The hireling branch has no d2d caller, because skill missiles don't strike mercs.
- Siblings: a player's freeze is already a chill in d2d (FUN_0057b230's player branch). Pets and a co-op partner as freeze targets are not traced.
- Gameplay change, by design: a Frost Nova or Glacial Spike now chills (does not freeze) every champion, unique, superunique and act boss, and monsters with cold resistance are frozen for a shorter time, as in the original game.

### Relation to #159

#159 (Frozen Armor and the freeze resume) is merged. This branch uses its `freeze_rules.py` oracle and its `GameData::freeze_divisor`, and does not touch Frozen Armor's own copy of the rule in `states.cpp`; that copy could call `rules::monster_freeze` in a follow-up.

---

---

## Prep notes (not part of the PR body)

Base and head: private fork, branch `local/freeze-rules-v5`, head 32c4943a, on upstream main 48908176 (the merge of #174; rebased 2026-11-02 without a conflict, counts re-run: [rebase174-20261102.md](rebase174-20261102.md)); the earlier `local/freeze-rules-v4` 9caa1b9b was on cb716f10 (which contains #159 as 7c1ecf57). Pushed to the private remote only; nothing to the public fork, no PR. To re-rebase after the maintainer's next merge: `git fetch upstream && git rebase --onto upstream/main cb716f10 local/freeze-rules-v4` in the PR checkout; the four commits touch `combat.hpp`, `missiles.cpp`, `ai.cpp`, `ai.hpp`, `uniques.hpp` (comments), `superuniques.md`, `monsters.py` (docstring) and add `test_freeze_rules.cpp`. The commit bodies cite "upstream main cb716f10" as the differential base (update the hash on rebase). The earlier shapes `fix/freeze-rules-v2` (conflicted with #159 on `tools/emu/freeze_rules.py`) and `local/freeze-rules-v3` (made independent of #159) are superseded, since #159 is now in main (V: `git merge-tree` of v2 against #159's head gave the one add/add conflict in the tool file).

Counts (worker host, offline, no game, no d2d slot; cases.tsv md5 ddbd53b274ad and game.tsv md5 0168caed598e, equal to the earlier runs, game side re-run natively): upstream main cb716f10 plus the test file 185,746 of 225,748 differ (plain 48,946, special 78,501, half_freeze 58,299); v4 head 0. The separately written harness (own generator, seed 0x1F00DFACE): 146,591 cases, 6 differ, all freeze length above 0x100000 (the known residual). Suites on the v4 head with the game data dir, rc 0: test_rules, test_game_rules, test_monsters, test_monster_thinks, test_monster_ai, test_combat, test_game, test_freeze_resume, test_armor_hit; test_freeze_rules without arguments SKIP.

CI parity (V where run): g++ full build of the test set with the build's flags (-Wall -Wextra -Wpedantic -Wno-missing-field-initializers -Werror, plus -Wshadow -Wunused -Wsign-compare): 0 warnings. A syntax-only pass over the changed .cpp files with clang++ and g++ using -Wshadow -Wrange-loop-construct -Wconversion: 0 warnings on added lines (g++ -Wsign-conversion, beyond CI, flags `types[fallen->second]` at test_freeze_rules.cpp line 67; left as is). clang-tidy (repo .clang-tidy) on the changed .cpp files: 0 warnings on added lines (pre-existing x / y binding names elsewhere). pr_guard: range and each of the four commits 0 FAIL / 0 WARN. presubmit --quick: 0 FAIL, 2 WARN (siblings: the diff touches monster only; build skipped by --quick). MSVC /W4 and windows.h macro names (read the diff only, U for a real MSVC build): no min/max macro use, designated initializers are C++20 and already in the tree. Direct includes: `missiles.cpp` adds `<uniques.hpp>` for `rules::Boss`.

Against PR #174 (Ponytail batch 8, head 489ef83d, base 3852c904, 14 commits): throwaway merge of v4 and #174's head is clean (V: `git merge-tree`, no conflict). Three files change on both sides: `ai.hpp` (#174 adds `Monster::crush_divisor` near the `Target` builder; ours is a `half_freeze` comment), `combat.hpp` (#174: `Target::crush_divisor`, `Swing::missile`, Crushing Blow in `player_blow`; ours: `Freeze` / `monster_freeze` after `missile_blow`), `missiles.cpp` (#174: one line `swing.missile = true` near line 692; ours: the freeze block near line 656). The hunks are separate, no semantic overlap (V by reading the diff: #174 never touches freeze, chill_ticks, half_freeze or stun_until in the strike path). I also built the merged tree (v4 + #174 head) with the test set: 0 warnings, the same ten suites rc 0, freeze differential 225,748 cases 0 differ. No resolution is needed after either lands; if #174 lands first, the rebase is `git rebase --onto upstream/main cb716f10 local/freeze-rules-v4` with no manual step expected.

U: det state hashes (need a d2d slot); an MSVC build; the independent re-check of the final head.

Agent: Claude Code
