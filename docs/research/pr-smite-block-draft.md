# PR draft: a block no longer stops Smite

> status: DRAFT, 2026-11-02 (card prbuildsmite). PRIVATE: nothing is posted or pushed to a public remote; the branch is on the private fork remote only. Branch `local/smite-block-v1-final`, head 887b502a, ONE commit on upstream main 82ea6db4 (the same tree is `local/smite-block-v1`, heads 57c48e98 + a47536e2, kept because the remote takes no force-push; use `-final`). Tags: V = run, V read = read in game.exe's disassembly, S = from an earlier doc, U = unverified.

## PR text (for upstream; no private names)

**Title:** A block no longer stops Smite

**Body**

`FUN_005ce9f0` (Smite, srvdofunc 150) calls the to-hit / defense check `FUN_0057ec10` at `0x5ceba8` and then ORs the hit bit back in (`or ax, 1` at `0x5cebb0`), so a blocked Smite still deals its damage, its stun and its knockback. `player_blow` ended the blow on any block, so a monster that blocked took nothing. Now the block roll is made as before (same RNG draw count for every skill and every block chance) and marks the blow `blocked`; only a Smite continues to the damage. `Fight::land` ends the blow on `blocked && !hit`, which is every other swing as before.

Only Smite does this: of the 33 call sites of `FUN_0057ec10` only `0x5cebb0` ORs bit 0 (Zeal, Sacrifice and the player's Charge are stopped by a block; Blessed Hammer is a missile). The `Blow` of pets, missiles and deaths never has `swing.smite`, so their `blocked` still means not hit. Bosses and undead follow the same rule (the Smite chain never asks the undead test, V run).

Counted against the original game (game.exe 1.14d's own Smite function `FUN_005ce9f0` run under an emulator, with its to-hit, defense and damage helpers; the same cases on both sides; a case lands when the damage reaches `FUN_0057c6c0`):

| game cases (seed), monster defender | game lands | base lands | base differs | head lands | head differs |
|---|---|---|---|---|---|
| 20,000 (7331): 15,027 monster | 15,027 | 11,812 | 3,215 | 15,027 | 0 |
| 20,000 (99): 15,008 monster | 15,008 | 11,788 | 3,220 | 15,008 | 0 |

The cases that differ on the base are exactly the ones the game blocked (3,215 and 3,220). Landed damage is inside the game's range (FUN_0057b420: a roll in [min, max + 255] x (100 + ED + StrBonus) / 100, one point of slack) in every case, base and head. Counterfactual in the game itself (the 4 bytes at `0x5cebb0` NOPed): 1,552 of 20,000 land, so the forced hit is what makes the other 18,448 land (V run). A plain swing against a blocking defender is still stopped on the head (0 hits in all 30,035 cases).

Block chance sweep, d2d's real `player_blow`, 20,000 Smite blows each (the same RNG seed for base and head):

| block chance | base hit / blocked / damage dealt | head hit / blocked / damage dealt |
|---|---|---|
| 0 % | 20,000 / 0 / 70,011 | 20,000 / 0 / 70,011 |
| 25 % | 15,075 / 4,925 / 52,873 | 20,000 / 4,925 / 70,147 |
| 50 % | 10,009 / 9,991 / 35,005 | 20,000 / 9,991 / 69,995 |
| 100 % | 0 / 20,000 / 0 | 20,000 / 20,000 / 70,019 |

(The plain-swing columns are identical on base and head at every chance.) New tests (`tests/test_combat.cpp`, next to the Smite ones): a Smite against block 100 is `blocked`, `hit`, deals its damage, keeps stun and knockback; a plain swing, a swing with enhanced damage and stun, and a kick against block 100 are never `hit` (30 of 40 blocked, the rest missed by the to-hit roll); block 0 never marks a Smite blocked and the damage is the same at 0 and 100. The first assertion fails on the base (V run: the head's test compiled against the base headers aborts at it).

Checks (worker host, build -j2 niced): the seven suites (test_rules, test_game_rules, test_monsters, test_monster_thinks, test_monster_ai, test_combat, test_game with the data) pass on the head; `pr_guard` 0 FAIL 0 WARN; `presubmit --quick` 0 FAIL, 2 WARN (docs-sync: no doc names this rule, the comments carry it; build skipped by --quick); g++ and clang++ with the CI flags: 0 warnings on `strike.cpp` and `test_combat.cpp`; the same with `-Wshadow -Wrange-loop-construct -Wconversion` syntax only: 0 new warnings on the added lines (33 clang warnings in `strike.cpp` are on lines the diff does not touch and are the same on base). MSVC /W4 and windows.h macros, by reading the diff: no new local shadows another (the test locals are `smite_blocker`, `blocked_smite`, `swinger`, `swing_blocker`, `other`, `through`, `chance`, `defender`, `stopped_swings`), no `min` / `max` / `near` / `far` name, no new include, no struct change.

Agent: Claude Code

## Credits (for the PR body; tools/credit/credit.py credits prbuildsmite)

Traced (the forced hit): community contributor chuckado, finding D8 of repo PR 23 ("game.exe forces hit bit (or ax, 1), so Smite is unblockable", with the `combat.hpp` line); consent is only `asked`, so credit.py `check` warns and the `Traced-by:` trailer must wait for their yes (or the PR body says it without a trailer). Independently verified: the verifysmite card (Opus; own harness, seeds 7331, 99, 20261102), named in the PR body and CONTRIBUTORS.md. Author of the change: the owner (numeric noreply identity for author and committer, checked with `git log --format='%an <%ae> | %cn <%ce>'`). AI work: the `Agent:` line only.

## The animation (decision, and a limit)

What game.exe does after a blocked Smite (V read, `FUN_0057cee0`, the damage reaction of a monster defender, 0x57cf3d onward): record bit 8 (knockback) first (KB mode); else bit 0x10 (block) puts the monster in mode 6 (BL; 0x13 for a few special cases) and the bit-4 get-hit roll `FUN_0057cb00` is NOT made; only a record with bit 4 (a clean hit, not blocked) reaches the get-hit roll. So a blocked Smite without knockback plays the block animation and does not use the monster's get-hit roll; with knockback it is knocked back.

NOT implemented in this PR, stated as a limit: doing it right needs two more changes that are not small. (1) `hurt()` would take a "blocked" flag to skip the get-hit roll and set BL, and the roll is on the monster's own seed, so skipping it changes that seed's stream (a determinism question). (2) The game rolls the block only after the to-hit roll passed (a miss arrives as word 0x0001 with no defense roll, V run); d2d's Smite has no to-hit roll and rolls the block always, so d2d marks a Smite `blocked` more often than the game does (the counted run above: d2d marks blocked exactly where the game did, because the probe takes the game's own bit; with d2d's own roll the rate would be the block chance, not the chance times the to-hit chance). Giving Smite that to-hit roll adds a draw per Smite. Both are a separate small PR if the maintainer wants the animation; this PR changes damage only. After it, a blocked Smite shows the ordinary get-hit (or nothing) instead of the block animation (U: how often that is visible).

## Side quirk (not fixed here; low)

When the shield's min equals its max the game still rolls up to +255/256ths (10-10 at +100 %: game 20..21, d2d 20; V run, verifysmite `D` rows); d2d's `high > low ? ... : low` returns exactly `low`. At most one point on such a shield; separate from this PR.

## Not run (U)

Determinism scenarios (the lab's `det_scenarios_run.py` on a base and a head tree): a Smite against a blocking monster now deals damage, so any scenario with a paladin Smite and a blocking monster changes its monster-state hash; the others should not. Ready-to-run checklist: build the base and the head with the lab patch set on the worker host (-j2, nice, one at a time), run the scenarios on both with a repeat of the base, attribute each differing hash to a blocked Smite. No game run (offline; the game side is the emulator).

## Evidence (worker host, V unless said)

- Fork branch `local/smite-block-v1-final` (887b502a), base upstream main 82ea6db4 fetched 2026-11-02; builds in `~/d2d-lab/pr/smiteblock-out/head`, base compiled in place from the detached worktree.
- Game side: `docs/research/verify-smiteblock/vs_oracle.py 20000 SEED` (card verifysmite), outputs `game-7331.txt`, `game-99.txt`; d2d side: `tools/emu/sbp_probe.cpp` (`chances`, `cases`) run by `tools/emu/sbp_run.sh` against each tree's own `components/rules`; CI parity `tools/emu/sbp_ci.sh`. The author's earlier harness (`sb_*`, `docs/research/smiteblock-trace.md`) found the same: 23,347 blocked cases all landed.
- The corrected line numbers (upstream main 82ea6db4): `components/rules/combat.hpp` `player_blow` (the block roll), `components/game/fight/strike.cpp` `Fight::land`.

## Shared-file lines for the merger

See openspec/changes/prbuildsmite/merge-lines.txt.
