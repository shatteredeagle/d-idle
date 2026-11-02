# Opening the Frozen Armor bundle and the curse-on-immune PR (card openpr)

> status: done 2026-11-01. Upstream PRs #159 and #160 are open; nothing else was posted. CodeRabbit follow-ups belong to a later card.

## Posted

- posted: 2026-11-01 go: the user, popup answer "Open both, bundle first".
- posted: 2026-11-01 fetch upstream: main 52362d52 (#158 merge); both candidates were on 7c3a50af, rebased with `git rebase --onto upstream/main 7c3a50af` without conflicts. The cherry-pick `-x` lines were stripped from the commit messages (identity unchanged: the numeric-ID noreply, `Agent: Claude Code`, no other trailer).
- posted: 2026-11-01 push (plain, new branch) to the fork: fix/frozen-armor-and-resume-final, head d8cc3b47 (5aea44af Frozen Armor claim + tooling, d8cc3b47 freeze-resume claim + tooling).
- posted: 2026-11-01 PR opened: https://github.com/psi29a/d2decomped/pull/159 (13 files, 2 commits; title "Frozen Armor freezes by the freeze rule; a frozen monster thinks again at the freeze end + aidel"; body from pr-bundle-armor-resume-body.md without the internal header, hashes updated, ends `Agent: Claude Code`).
- posted: 2026-11-01 push (plain, new branch) to the fork: fix/curse-immune-final, head 4e2ec701 (9281a06d claim, 4e2ec701 tooling).
- posted: 2026-11-01 PR opened: https://github.com/psi29a/d2decomped/pull/160 (5 files, 2 commits; title "Curses and auras: a lowering counts a fifth on a monster whose base resistance is 100 or more"; body from pr-curseimmune-draft.md at status READY, two wording fixes, ends `Agent: Claude Code`).

## Checks at open time (worker host, upstream main 52362d52)

| | bundle (#159) | curse (#160) |
|---|---|---|
| pr_guard range / each commit | 0 FAIL 0 WARN / 0 and 0 | 0 FAIL 0 WARN / 0 and 0 |
| Release build, test_monsters / rules / combat / protocol / game | all rc 0 | all rc 0 |
| tooling unit tests | 17 + 7 ok (1 skip) | 17 + 7 ok (1 skip) |
| presubmit --quick | 0 FAIL / 3 WARN (answered in the body) | 0 FAIL / 3 WARN |
| differential | armor 0 mismatches of 108,506; resume 0 differ of 12,548 frozen | adj differ 0 of 147,628; dealt differ 1,958 (D2, not claimed) |

Not re-run after the rebase: gcc shadow sweep, clang-tidy, ruff (the bundle body says so). PR page checked with `gh pr view`: state OPEN, 2 commits each, file counts as above.

Body ending convention (copied from merged #157 and #158): the body ends with the line `Agent: Claude Code`; CodeRabbit appends its own release-notes block afterwards. No "Generated with" line, no session link.

## CodeRabbit follow-up on #160 (card crfollow160)

- posted: 2026-11-01 CodeRabbit review (state COMMENTED, 3 actionable) read: three inline findings, all on the tooling and test side: tests/game/test_sk_resist.cpp:77 (Minor: an unopened case file gives an empty run that returns success), tools/emu/sk_resist.py:90 (Minor: parse() silently drops a malformed case line), tools/emu/sk_resist.py:167 (Major: cmp() skips case ids missing from either result file and could report zero differences). Own check (CHECKPRS) added ruff B905 at sk_resist.py:186 (zip without strict=).
- decision: all three valid, each a trivially safe guard in test or tooling code; the claim commit 9281a06d is untouched. Fixes: the test returns 1 with a message when the case file does not open; parse() skips only blank lines and raises ValueError with file:line otherwise; cmp() exits on an empty case set and on any requested id missing from the game or d2d results (the old skip is removed); tbl() zip gets strict=False (same form as freeze_resume.py). +18 -6 lines in two files.
- check: ruff (presubmit rules plus B905) clean, test_tooling 7 ok (1 skip), pr_guard 0 FAIL / 0 WARN for the commit and for upstream/main..head, Release build of test_sk_resist; the d2d side re-run against the stored game-side data: seed 1 adj differ 0 of 147,628, dealt differ 1,958; seed 7 adj differ 0 of 147,628, dealt differ 1,977 (both as before, so behaviour is unchanged). Negative checks: missing case file rc=1, a three-field line names its line, empty case file and a truncated d2d file are rejected.
- posted: 2026-11-01 push (plain, no force) to the fork: fix/curse-immune-final 4e2ec701 -> 802dee68 (one follow-up commit, numeric-ID noreply identity, `Agent: Claude Code`). Branch tip and the PR head were both 4e2ec701 before the push. PR #160 now has 3 commits.
- posted: 2026-11-01 replies to the three CodeRabbit inline threads (comment ids 4437023955, 4437023961, 4437023965; replies 4437044889, 4437044955, 4437045021): each valid, fixed in 802dee68. No re-review was requested (the included allowance was used up by the first review).
- non-inline: walkthrough, finding list and review info repeat the three findings; no merge-risk note, pre-merge check or docstring-coverage item needs action.

## Shared-file lines for the merger

- BACKLOG: upstream #159 (Frozen Armor + freeze resume) and #160 (curse on immune) are open; watch for the maintainer and CodeRabbit (follow-ups need their own card).
- INDEX: docs/research/openpr-log.md: posted log for #159 and #160.
- CHANGELOG: Docs: opened upstream PRs #159 (Frozen Armor and freeze-resume bundle) and #160 (curse lowering on immune monsters); posted log in docs/research/openpr-log.md.

## CodeRabbit follow-up on #159 (card crfollow159)

- posted: 2026-11-01 CodeRabbit review (state COMMENTED, 1 actionable, 0 nitpicks) read; one inline finding on tools/emu/freeze_rules.py line 6 (Minor, maintainability, quick win): the usage text names a `test_freeze_rules --freeze-rules` binary the PR does not add, so `compare` cannot run as documented. Its walkthrough and risk note repeat the same point; no pre-merge check or docstring-coverage item was flagged.
- decision: valid. freeze_rules.py is only a shared helper in this PR (armor_hit.py subclasses its Game runner and has its own d2d side, test_armor_hit). The fix is docstring only: drop the two usage lines and say where the d2d side lives (+4 -2 lines). No claim changes. The bot's suggestion to add the missing target was not taken (it would widen the PR).
- check: the Python code is AST-identical before and after (docstring excluded), tooling test_tooling 7 ok (1 skip), ruff (presubmit rules plus B905) clean; the differential was not re-run because no code changed.
- posted: 2026-11-01 push (plain, no force) to the fork: fix/frozen-armor-and-resume-final d8cc3b47 -> c4e9db97 (one follow-up commit, numeric-ID noreply identity, `Agent: Claude Code`); pr_guard 0 FAIL / 0 WARN for the commit and for upstream/main..head. PR #159 now has 3 commits.
- posted: 2026-11-01 reply to the CodeRabbit inline thread (comment id 4436799936, reply 4436827126): valid, fixed in c4e9db97, docstring only.

## Blizzard / Eruption shard offsets (card openpr-blizzard)

- posted: 2026-11-01 go: the user's popup answer "Open it now" for the Blizzard shard-offset PR (one PR, upstream psi29a/d2decomped).
- posted: 2026-11-01 fetch upstream: main 7d5b3a5c (unchanged since the candidate's base, so no rebase and no new branch was needed; the candidate's tree and identity are as built).
- posted: 2026-11-01 push (plain, new branch name, no force) to the fork: fix/blizzard-shard-offsets-v2 head 3d6f3fe7 (46c55621 tests and tooling, then 3d6f3fe7 claim; numeric-ID noreply identity, `Agent: Claude Code`, no other trailer). The first candidate fix/blizzard-shard-offsets (9a1a90eb, 681d52a9) was not used.
- posted: 2026-11-01 PR opened: https://github.com/psi29a/d2decomped/pull/163 (5 files, 2 commits; title "Blizzard and Eruption shards fall -(R-1)..(R-2) of the centre, not -(R-1)..(R-1) (FUN_005a9820)"). State OPEN, base 7d5b3a5c, head 3d6f3fe7 as checked with gh pr view. Body: the draft's body section with the status line, internal links and internal wording removed; it ends with a blank line and the bare `Agent: Claude Code` line, like #157 and #158 (no "Generated with" line, no session link).
- check (all V, run on the worker host on head 3d6f3fe7 before the push):

| check | result |
|---|---|
| pr_guard, range origin/main..head and each commit | 0 FAIL / 0 WARN, rc 0 each |
| Release build (`-ffp-contract=off`), test_game (with the data dir) and test_rules | build rc 0, both pass |
| presubmit --quick | 0 FAIL, 1 WARN (build skipped by --quick); tooling unit tests ok |
| ruff 0.8.6, repo rules plus B905, py312 | clean |
| test_sk_ground --blizz, base build (tooling only, source of main) | seed 1: 14,926 of 100,000 off the -6..5 grid; seed 7: 14,957 |
| same on the head | 0 and 0 |
| independent harness (verify_blizzard.py gen-d2d 5000 777, two Release probe builds) | base 16,676 of 112,964 shards off the grid (= Blizzard 14,804 + Eruption 1,872 of the claim's split); head 0; shards per cast {25: 3997, 13: 1003}; the per-skill split itself was not re-tabulated |
| git merge-tree (write-tree) head against #160 | clean; against main clean; against #159 conflicts in ai.hpp (a file this PR does not touch, the known #159 conflict) |

- not re-run: the game-side emulator comparisons (64,200 and 120,000 cases), the world-rng-state check, gcc -Wshadow and clang-tidy (clean on the first candidate, identical source), det state hashes, MSVC, a live game. The body says so.
- nothing else posted: no comment, label or reviewer request; CodeRabbit follow-ups are a later task.

## Shared-file lines for the merger

- BACKLOG: #159 got a docstring follow-up (c4e9db97) for CodeRabbit; #160 review still pending after the rate limit (own card).
- BACKLOG: #160 got a tooling and test follow-up (802dee68) for CodeRabbit; watch for the maintainer.
- INDEX, CHANGELOG, decomp-queue, research README: no change.

## CodeRabbit follow-up on #163 (card crfollow163)

- posted: 2026-11-01 CodeRabbit review (state COMMENTED, 1 actionable, 1 nitpick) read: inline finding on tools/emu/sk_ground_blizz.py:147 (Minor, quick win): cmp still accepts -(R-1)..(R-1) as d2d's range, so +(R-1) counts as valid for d2d; bounds, range and share lines and the docstring need the new upper end R-2. Nitpick in the review body (not inline): tests/game/test_sk_ground.cpp lines 66-67 and 86, short loop variable names (x, y, i) against the repo's clang-tidy rule.
- decision: inline finding valid (the claim code in missiles.cpp is right: rng(2 reach) - reach is -6..5 at radius 7; only the tool's idea of d2d's range was stale). Fixed in tooling only: lo/hi, the d2d-only value set, the printed range and the docstring use -(R-1)..(R-2); radius 1 keeps 0..0 (it draws nothing; without that clamp the check flagged 246 radius-1 shards); the old 14.8 percent share is kept as a labelled "before the fix" line. +8 -5 lines in one file.
- decision: nitpick not applied: the brief's standing go covers the one inline thread and keeps the tests unchanged; presubmit's short-names check passes (it only looks at added lines). Left for the author to decide.
- check: ruff (presubmit rules plus B905, py312) clean, tooling tests 31 ok, presubmit --quick 0 FAIL (1 WARN: build skipped by --quick), pr_guard 0 FAIL / 0 WARN for the commit and for origin/main..head. Differential re-run on the stored data: Blizzard 14,926 / 14,957 of 100,000 off-grid on base and 0 / 0 on head (seeds 1 / 7), cmp over 42,100 cases: reading differs 0, game offsets outside d2d's range 0. No game-side or d2d-side code changed, so behaviour is unchanged. No Release build rerun (no C++ touched).
- posted: 2026-11-01 push (plain, no force) to the fork: fix/blizzard-shard-offsets-v2 3d6f3fe7 -> 3ac63fa0 (one follow-up commit, numeric-ID noreply identity, `Agent: Claude Code`). The PR head was 3d6f3fe7 before the push and is 3ac63fa0 now.
- posted: 2026-11-01 reply to the CodeRabbit inline thread (comment id 4437340451, reply 4437363818): valid, fixed in 3ac63fa0. No re-review was requested.
- non-inline: walkthrough repeats the claim and tools; effort 3 (about 25 minutes); possibly related issue 103 (RNG stand-ins), no action. Merge risk Low, with the same note as the inline finding (now fixed). Pre-merge checks: 4 passed, 1 warning: docstring coverage 11.11 percent against 80 (scoped to touched functions with complexity of 2 or more); a style gate on the tooling script, no action taken (not a defect; the author may add docstrings). Status checks: guard success, CodeRabbit neutral; review decision REVIEW_REQUIRED.

## Shared-file lines for the merger

- BACKLOG: #163 got a tooling follow-up (3ac63fa0) for CodeRabbit; watch for the maintainer.
- INDEX, CHANGELOG, decomp-queue, research README: no change.

## Monster missile record and poison units, bundled (card propenmissile)

- posted: 2026-11-02 go: the user, explicit "open the two now", then corrected before anything was opened: "if they haven't been posted I meant to bundle them". One PR, two commit groups, no squash.
- posted: 2026-11-02 fetch upstream: main 6935cf64 (unchanged since both candidates' base, so no rebase). The record candidate (5 commits, local head 59d319c8) and the skill-rows candidate (3 commits, local head f52df8d4) were rewritten for the PR: numeric-ID noreply identity as author and committer, the trailers replaced by the bare `Agent: Claude Code` line (no session link, no model trailer); trees identical to the candidates. The skill-rows commits were cherry-picked onto the record commits (no conflict), record first.
- posted: 2026-11-02 push (plain, new branch, no force) to the fork: fix/monster-missile-record-and-poison head ee8a2093 (92f9a9dc test mode, 02205609 record, 809023e9 poison units, 85b8fa3e owner poison, 28400999 SrcDamage byte, 6d4fa1bf skill-sourced poison, 50175311 trap poison, ee8a2093 wording). The same branch was pushed to the private remote.
- posted: 2026-11-02 PR opened: https://github.com/psi29a/d2decomped/pull/164 (9 files, 8 commits; title "A monster missile's launch record and poison units as game.exe fills them (FUN_0064b860): physical brackets, HitShift, row element; skill-sourced and trap poison per frame"). State OPEN, base 6935cf64, head ee8a2093 as checked with gh pr view. Body from pr-missilerecord-draft.md and pr-missilepoison-draft.md merged into one, ends with the bare `Agent: Claude Code` line.

| check (worker host, head ee8a2093) | result |
|---|---|
| pr_guard, range and each of 8 commits | 0 FAIL / 0 WARN, rc 0 each |
| presubmit --quick with the CI-pinned ruff 0.8.6 | 0 FAIL, 4 WARN (unit kinds, hit paths, docs-sync, build skipped; answered in the body) |
| presubmit --quick with ruff 0.17.3 (a newer one on the box) | 1 FAIL: B905 `zip()` without strict= in mm_diff.py:179 and mm_launch.py:42; not the CI version, not changed |
| tooling unit tests | 17 ok; test_tooling 7 ok (1 skip) |
| Release build, ctest test_game (data dir), test_combat, test_game_rules, test_rules, test_monster_ai, test_monsters, test_monster_missile, test_missile_blow_poison | 8 of 8 pass |

Not re-run on the combined head: the differentials (emulator comparisons), clang-tidy, the gcc shadow sweep; they were run on each group separately (the body says so). Nothing else posted: no comment, label or reviewer request.

## Shared-file lines for the merger (propenmissile)

- BACKLOG: #164 (monster missile record + poison units, bundled) is open; watch for the maintainer and CodeRabbit (follow-ups need their own card).
- INDEX, decomp-queue, research README: no change.

## CodeRabbit follow-up on #164 (card crfollow164)

- posted: 2026-11-02 CodeRabbit review read: two inline findings, both Minor on tools/emu/mm_diff.py: line 222 (the MMTAB table is read on every cmp even with zero differences, so a clean comparison fails with FileNotFoundError when ~/mmtab is absent), line 226 (cmp skips cases without a d2d line and returns only the mismatch count, so an incomplete d2d output can exit 0). Own check: ruff 0.8.6 (CI-pinned) is clean; B905 (zip without strict=) shows only on newer ruff at mm_diff.py:179.
- decision: line 222 valid: the table is needed only for the row data of a difference, so it is read only when by_row is non-empty. Line 226 valid: a missing case is a failure, cmp now returns bad["any"] + missing (same class as the #160 sk_resist finding). B905 at mm_diff.py:179 fixed with strict=False (clean on 0.8.6 too); mm_launch.py:42 not touched. Claim commits untouched. +9 -8 lines in one file.
- check: ruff 0.8.6 with the presubmit rules plus B905 clean, tools test_tooling 7 ok, pr_guard 0 FAIL / 0 WARN for the commit and for upstream/main..head. Negative checks with MMTAB pointing at a missing directory: clean case rc 0 (before: traceback, rc 1), a case with no d2d line rc 1 (before: rc 0), a mismatch still reads the table. No game or d2d code changed, so behaviour on a complete run is unchanged; no differential re-run needed (tooling output only).
- posted: 2026-11-02 push (plain, no force) to the fork: fix/monster-missile-record-and-poison ee8a2093 -> cd01212f (one follow-up commit, numeric-ID noreply identity, `Agent: Claude Code`); also pushed to the private remote. PR #164 now has 9 commits.
- posted: 2026-11-02 replies to the two inline threads (comment ids 4439301404, 4439301382; replies 4439323712, 4439323832): each valid, fixed in cd01212f. No re-review was requested.
- non-inline: none read beyond the two inline items.

## Shared-file lines for the merger (crfollow164)

- BACKLOG: #164 got a tooling follow-up (cd01212f) for CodeRabbit; watch for the maintainer.
- INDEX, CHANGELOG, decomp-queue, research README: no change.

## Player missile damage roll (card propenplayerroll)

- posted: 2026-11-02 go: the user, explicit authorisation of up to 4 more upstream PRs overnight, bundled commits; this is number 1. Only this PR was opened.
- posted: 2026-11-02 fetch upstream: main 6935cf64 (unchanged; the five candidate commits are a clean fast-forward on it). The candidate (pr/player-roll-v2, e66e10ba) was rewritten for the PR: numeric-ID noreply identity as author and committer (the candidate's committer was a generic one), only the `Agent: Claude Code` trailer; trees identical (git diff e66e10ba..head empty). Messages, branch name and body scanned for host names, home paths, emails, tool names, session links, model attribution: none.
- posted: 2026-11-02 push (plain, new branch, no force) to the fork: fix/player-missile-damage-roll head 1c273ef1 (610c0474 oracle and test mode, 4f51d89c element roll top, 1ea79c57 physical in 256ths, d67720e0 -100 floor, 1c273ef1 three include lines). Also pushed to the private remote. The old pr/player-roll (28c0540a) was not touched.
- posted: 2026-11-02 PR opened: https://github.com/psi29a/d2decomped/pull/166 (7 files, 5 commits; title "A missile's element roll never reaches its max, and its physical roll keeps its 256ths (FUN_005a89a0)"). State OPEN, base 6935cf64, head 1c273ef1 as checked with gh pr view. Body from pr-playerroll-draft.md without the internal sections (lab-binary scenarios, verifier checklist, merger lines), ends with the bare `Agent: Claude Code` line; it says the include lines duplicate #164's and either merge order works.

| check (worker host, head 1c273ef1) | result |
|---|---|
| throwaway merge with fix/monster-missile-record-and-poison | clean, no conflict |
| pr_guard, range and each of 5 commits | 0 FAIL / 0 WARN |
| presubmit --quick | 0 FAIL, 2 WARN (doc-claims "never" at missiles.md:106, build skipped; both answered in the body) |
| tooling unit tests | 17 ok; test_tooling 7 ok (1 skip) |
| Release build with -Wall -Wextra -Wshadow, test_combat, test_rules, test_game_rules, test_game, test_game_missiles (data dir set) | exit 0 each; 0 warnings in combat.hpp, test_combat, test_missile_hit, test_game_missiles |
| diff scan for `for (const auto [..] :` copying loops | none added |
| CI "Names and includes" selection (tidy-pr-select, clang-tidy 124 files) | 0 include-cleaner findings; 0 findings in our files; 377 readability-identifier-length in untouched files (a newer-LLVM check, same as upstream main) |

Not re-run: the emulator differentials and the deterministic scenarios (run on the candidate head by the author and the verifier; the tree is identical), an MSVC build. No comment, label or reviewer request was posted.

## CodeRabbit follow-up on #166 (card crfollow166)

- posted: 2026-11-02 CodeRabbit review read: three inline findings, all Minor: missiles.md:106 (the physical roll is described as max-exclusive and pointed at rules::missile_blow, which rolls it inclusive), test_missile_hit.cpp:55 (an unreadable case file returns success), tools/emu/README.md:45 (the oracle row can pass on empty input). Non-inline nitpicks (loop names `i` in test_combat.cpp:437 and `k` in test_missile_hit.cpp:62) were read and not acted on.
- decision: missiles.md valid: the max-exclusive wording is now for the elements, and a separate paragraph says the physical roll is inclusive, rolled whenever phys_hi > 0, not examined and not covered by the differential. test_missile_hit valid: it prints an error and returns 1 when the file does not open. README valid: the row sets pipefail and chains the stages with &&; only this row (the table's other rows keep their form). Claim commits untouched; +15 -6 lines in three files.
- check: pr_guard 0 FAIL / 0 WARN for the commit and for upstream/main..head; presubmit --quick 0 FAIL, 3 WARN (ruff not installed there, build skipped, the older "never" wording at missiles.md:105 that holds for the elements); test_missile_hit built with g++ and clang++ with the CI flags plus -Wshadow -Wrange-loop-construct -Wconversion, 0 warnings in it (D2D_WERROR off because unrelated files such as ai.cpp shadow under -Wshadow); test_combat ok, test_game_missiles ok with the data dir; negative check: a missing case file now exits 1 and the pipefail row fails; the gen output (47,000 rows) still runs through test_missile_hit. The game-side oracle was not re-run (the guard does not change a result).
- posted: 2026-11-02 push (plain, no force) to the fork: fix/player-missile-damage-roll 1c273ef1 -> 7d4f6f0e (one follow-up commit, numeric-ID noreply identity, `Agent: Claude Code`); also pushed to the private remote. PR #166 now has 6 commits.
- posted: 2026-11-02 replies to the three inline threads (comment ids 4440416918, 4440416929, 4440416938; replies 4440512312, 4440512543, 4440512785): each valid, fixed in 7d4f6f0e. No re-review was requested.
- non-inline: the nitpick loop names (i, k) are style notes on lines this PR added; not acted on, no reply (not inline threads of ours to answer under this go).

## Shared-file lines for the merger (crfollow166)

- BACKLOG: #166 got a follow-up (7d4f6f0e) for CodeRabbit; watch for the maintainer.
- INDEX, CHANGELOG, decomp-queue, research README: no change.

## Shared-file lines for the merger (propenplayerroll)

- BACKLOG: #166 (player missile damage roll) is open; watch for the maintainer and CodeRabbit.
- INDEX, CHANGELOG, decomp-queue, research README: see merge-lines.txt.

## Monster think paces (card propenmonsterpacea)

- posted: 2026-11-02 go: the user, standing authorisation of up to 4 more upstream PRs overnight, bundled commits; this is number 2. Only this PR was opened.
- posted: 2026-11-02 fetch upstream: main 3be426d3 (unchanged since the candidate's base, so no rebase). The candidate (local/monsterpace-a, 260eb34f, 13 commits) was rewritten for the PR: numeric-ID noreply identity as author and committer, the trailers replaced by the bare `Agent: Claude Code` line (no session link, no model trailer); trees identical (same tree hash). Messages, diff and body scanned for host names, home paths, emails, tool names, session links, model attribution: none.
- posted: 2026-11-02 push (plain, new branch, no force) to the fork: fix/monster-think-paces head bf7671a7 (2bb1de00 Zombie, 7ad32869 Brute, 9d7579d4 Blood Hawk, 7f3d0e16 Rogue, 18115b2d Lancer, dccb6756 Fallen flee, 6d77436b Blood Raven paces, 99a9c30b Blood Raven home rule, 94ac4cfb tools, a622ef7d replay cases, 29785180 unit asserts, 2b1a2f55 shaman corpse reach, bf7671a7 docs). Also pushed to the private remote.
- posted: 2026-11-02 PR opened: https://github.com/psi29a/d2decomped/pull/167 (11 files, 13 commits; title "Monsters carry the pace game.exe's Act 1 thinks give them, Blood Raven goes home by FUN_005dc480's distances, and a unique Fallen Shaman's corpse reach is FUN_005f1380's"). State OPEN, base 3be426d3, head bf7671a7 as checked with gh pr view. The body states its limits: the Brute's act and the Rogue's n-only fields untouched, the superunique flag bit unverified, no MSVC build. The base-velocity change is not opened (waits for this one).

| check (worker host, head bf7671a7) | result |
|---|---|
| pr_guard, range and each of 13 commits | 0 FAIL |
| presubmit --quick | 0 FAIL, 3 WARN (unit kinds, doc-claims, build skipped; answered in the body) |
| tooling unit tests | 17 ok |
| Release build, test_monster_thinks, test_monster_ai, test_monsters, test_rules, test_game_rules, test_combat, test_game (data dir) | 7 of 7 exit 0 |
| g++ and clang++, TU compile commands plus -Wall -Wextra -Wpedantic -Wshadow -Wrange-loop-construct -Wconversion -Wsign-conversion, ai.cpp and the two test TUs, against a base build | rc 0; 0 new warning messages vs 3be426d3; corpse harness 0 warnings |
| clang-tidy on the three changed TUs | 0 findings, 0 include-cleaner |

Not re-run: the emulator differentials (run on the identical tree by the author and a verifier), the deterministic scenarios, the full CI tidy selection, an MSVC build.

## Shared-file lines for the merger (propenmonsterpacea)

- BACKLOG: #167 (monster think paces) is open; watch for the maintainer and CodeRabbit. The velocity change waits for it.
- INDEX, decomp-queue, research README: see merge-lines.txt.

## Nightmare and Hell monster start level (card propennmlevel)

- posted: 2026-11-02 go: the user, standing authorisation of up to 4 more upstream PRs overnight, bundled commits; this is number 3. Only this PR was opened.
- posted: 2026-11-02 fetch upstream: main 3be426d3 at first (the candidate local/nmlevel-v5, 94562869, 2 commits, was on it); a plain push of that head to the fork as fix/nightmare-hell-monster-level was made and is not used (no PR). Then #167 merged: main 5df54372. Rebased as a new branch (`cherry-pick` of the two commits, no conflict, identities already the numeric-ID noreply, only `Agent: Claude Code`).
- posted: 2026-11-02 push (plain, new branch, no force) to the fork: fix/nightmare-hell-monster-level-2 head 9b04f5b1 (0afc3f36 the rule, 9b04f5b1 oracle and test mode). Also pushed to the private remote.
- posted: 2026-11-02 PR opened: https://github.com/psi29a/d2decomped/pull/170 (11 files, 2 commits; title "Nightmare and Hell monsters start at the area MonLvl, not their own Level(N) / Level(H) (FUN_00573cb0)"). State OPEN, base 5df54372, head 9b04f5b1 as checked with gh pr view. Body from the verified draft without internal sections; states the limits (network client copy and pets/Revive not covered, levels 83/110/111/117 crash the oracle, the level-0 clamp is shared, the two XP samples named, the pet flap pre-existing); ends with the bare `Agent: Claude Code` line.

| check (worker host) | result |
|---|---|
| det scenarios, rebuilt lab binaries from the v5 head and from its base 3be426d3, 14 scenarios, each binary twice | 13 identical in monster and state hash; pet differs (position flap) |
| pet flap control, 5 runs per build | base alone gives both end states (3 of 5 / 2 of 5), head the same two (1 of 5 / 4 of 5); monster hash equal throughout |
| 7 suites (test_game with data, combat, rules, game_rules, monsters, monster_ai, monster_thinks), Release -Wall -Wextra -Wshadow | exit 0 each on v5 head and on the rebased head; 0 compiler warnings |
| pr_guard, range and each commit, rebased head | 0 FAIL / 0 WARN |
| presubmit --quick, ruff 0.8.6 | 0 FAIL, 4 WARN (answered in the body) |
| grep of make_monster / spawn_monsters callers on the rebased head | no new caller from #167 |

Not re-run on the rebased head: the det scenarios (run on the same two commits over 3be426d3), clang-tidy, an MSVC build. Nothing else posted.

## Shared-file lines for the merger (propennmlevel)

- BACKLOG: #170 (Nightmare/Hell monster start level) is open; watch for the maintainer and CodeRabbit.
- INDEX, CHANGELOG, decomp-queue, research README: see merge-lines.txt.

## Monster base velocity (card propenmonsterpaceb)

- posted: 2026-11-02 go: the user, standing authorisation of up to 4 more upstream PRs overnight, bundled commits; this is number 4 of 4 and the last. Only this PR was opened.
- posted: 2026-11-02 fetch upstream: main 5df54372 (the #167 merge; the candidate local/monsterpace-b-velocity, 8a77a4ab, was on 3be426d3). Both commits were cherry-picked onto it as a new local branch (local/monsterpace-b-v2). One conflict: a comment block over the walk sum in `ai.cpp` that #167 had reworded; resolved by keeping #167's sentences and adding the 75 sentence (no code line conflicted). Then rewritten for the PR: numeric-ID noreply identity as author and committer, the trailers replaced by the bare `Agent: Claude Code` line (same tree as the gated head). Messages, diff and body scanned for host names, home paths, emails, tool names, session links, model attribution: none. A throwaway merge with the open #170, #163 and #159 is clean (#160 conflicts with main in `missiles.cpp` regardless).
- posted: 2026-11-02 push (plain, new branch, no force) to the fork: fix/monster-base-velocity head f8447aff (8bf07f12 code and oracle, f8447aff monster-ai.md). Also pushed to the private remote, with local/monsterpace-b-v2 (the pre-rewrite head).
- posted: 2026-11-02 PR opened: https://github.com/psi29a/d2decomped/pull/171 (4 files, 2 commits; title "A monster starts at 75 percent velocity, not 100 (FUN_00573cb0 SetStat(0x43, 0x4b))"). State OPEN, base 5df54372, head f8447aff as checked with gh pr view. The body says plainly that it changes every awake monster's walking speed, that chill and slows become relatively stronger (a coldeffect -50 walker hits the 25 percent floor), that Diablo's run and the whirl step stay at 100, that d2d mercs and pets are unchanged although in game.exe they probably also start at 75 (not traced), that there is no MSVC build, and that the deterministic hashes change for 14 of 15 scenarios.

| check (worker host, head f8447aff; tree identical to the gated head) | result |
|---|---|
| Release build, test_rules, test_game_rules, test_monsters, test_monster_thinks, test_monster_ai, test_combat, test_game (MPQ folder set) | 7 of 7 exit 0 |
| speed oracle, seed 20261102, 122,646 cases | base 5df54372: 50,626 of 61,654 monster rows differ (RN 25,321, WL 25,305), merc/pet/player 0; head: 0 of everything |
| deterministic scenarios, 15, seed 3, one slot | passive identical; 14 differ, first differing line a monster position; base reruns repeat their hashes in all 15 |
| g++ and clang++, TU compile commands plus -Wall -Wextra -Wpedantic -Wshadow -Wrange-loop-construct -Wconversion -Wsign-conversion, ai.cpp and the two monster tests | rc 0; 29 (g++) and 33 (clang++) warnings, equal to the base |
| clang-tidy on the same three TUs | no finding on an added line |
| pr_guard, range and each of the 2 commits | 0 FAIL / 0 WARN |
| presubmit --quick | 0 FAIL, 3 WARN (unit kinds, test-gap, build skipped; answered in the body) |
| tooling unit tests (tools/*.py) | 31 ok |

Not re-run: an MSVC build, the full ctest, the full CI tidy selection. No comment, label or reviewer request was posted. The limit of four authorised PRs is used up (#166, #167, #170, #171).

## Shared-file lines for the merger (propenmonsterpaceb)

- BACKLOG: #171 (monster base velocity 75) is open; watch for the maintainer and CodeRabbit. The authorised four PRs are used up.
- INDEX, decomp-queue, research README: see merge-lines.txt.

## Conflict rebase of #159, #160, #163 (card prconflicts)

- rule (V, read fresh from upstream main 4bf911b3: AGENTS.md section 6, CONTRIBUTING.md, .github/pull_request_template.md, .claude/skills/contribute): "Keep the branch current with `git rebase origin/main` before asking for review and after main moves." Commits: short subject, at most three lines of body; the author is the contributor; the PR body follows the template and ends `Agent: <name>`; run `python3 tools/pr_guard.py` and `python3 tools/presubmit.py --quick` before pushing; ask before every push and reply. No merge-commit policy is stated, but the stated policy is rebase, which rewrites the PR branch (force-push). The maintainer merges with merge commits ("Merge pull request #171"), so no squash is expected.
- found: no human comment on #159, #160 or #163 (issue comments, review comments, reviews: only CodeRabbit and our own replies); the conflict is GitHub's own state (#159 and #160 CONFLICTING/DIRTY, #163 MERGEABLE/BEHIND).
- posted: nothing. No push to the PR branches, no comment.
- fetch upstream: main 4bf911b3 (the #171 merge). Rebased with `git rebase upstream/main` in a worktree on the worker host, numeric-ID noreply identity as author and committer, `Agent: Claude Code` unchanged. New local branches (also pushed to the private fork remote only): local/prconf-163 3b6dfc83 (3 commits, no conflict), local/prconf-160 b91e9c38 (3 commits), local/prconf-159 1a0a26d9 (3 commits).
- conflicts: #160 in `components/game/fight.hpp` and `fight/missiles.cpp`: upstream gave `apply_target_stats` an optional `CalcEnv*` (enemy auras carry their own env) and added `self_state`; resolved by keeping both: signature `(skill_row, lvl, target, base, aura_env = nullptr)`, the `base` snapshot taken before any curse, the fifth rule inside the stat cases, the enemy-aura loop passes `base` and `&aura.env`. #159 in `components/game/ai.hpp`: upstream added `shatter` and `no_heal_until` beside our `stunned_until`; kept all three. #163 had no conflict. ai.cpp, strike.cpp and states.cpp of #159 merged without conflict; no other writer of `stun_until` was added upstream (grep).
- check (worker host, Release build with `-ffp-contract=off`, `D2_MPQ_DIR` set):

| check | result |
|---|---|
| test_rules, test_game_rules, test_monsters, test_monster_thinks, test_monster_ai, test_combat, test_game | exit 0 on each of the three heads |
| #163: `test_sk_ground --blizz` seeds 1 and 7, 100,000 casts each, vs the stored game.exe offsets | 2,500,000 shards each, never-made offsets 0 (0.00%), MISMATCH 0; offsets -6..5 on both sides |
| #160: `test_sk_resist --resist` on the stored 147,628-case files for both seeds, `sk_resist.py cmp` | adj differ 0; effres differ 3,986 / 3,992 and dealt differ 1,958 / 1,977, all under -100 (the floor, outside the claim, same as recorded) |
| #159: `test_armor_hit --armor-hit`, 108,506 stored cases | mismatches 0, calc1 differs 0, RESFAIL 0 |
| #159: `test_freeze_resume --freeze-resume`, two stored 37,776-case files | frozen in game.exe 0 differ (12,548 / 12,516), aidel 1 0 differ; output byte-identical to the recorded head output; the 25k chilled cases differ as before (outside the claim) |
| pr_guard (`upstream/main` range) and each commit | 0 FAIL / 0 WARN |
| presubmit --quick | 0 FAIL; 2 WARN (#160, #163), 3 WARN (#159): build skipped and test kinds, as at open time |
| tooling unit tests | 31 ok |
| g++ and clang++ -fsyntax-only with the TU command plus -Wall -Wextra -Wpedantic -Wshadow -Wrange-loop-construct -Wconversion -Wsign-conversion, changed TUs | warning counts equal to upstream main on every changed .cpp (29 g++; clang++ 39 missiles, 66 ai, 45 states, 44 strike, 38 gamedata_load) |
| merge-tree of the three heads pairwise | clean |

- not re-run: clang-tidy over the changed TUs, ruff with the repo configuration, MSVC, new game.exe oracle runs (stored game-side files used; the game side does not depend on d2d).
- hold: publishing needs `git push --force-with-lease` of local/prconf-N to fix/frozen-armor-and-resume-final, fix/curse-immune-final and fix/blizzard-shard-offsets-v2 on the public fork. Not done (standing rule); the user decides. The alternative is a merge of upstream main into each PR branch (plain push), which the maintainer's rule does not ask for.

### Second rebase on upstream main 878ebe32 (card prconflicts)

- fetch upstream: main moved from 4bf911b3 to 878ebe32 (#170, #172, #173 and others; it touches ai.cpp, ai.hpp, gamedata_load.cpp, tests/monsters/test_monsters.cpp, tests/game/test_game.cpp). The three branches rebased again without any conflict, as new local branches: local/prconf2-159 59833089, local/prconf2-160 bb8b0fa5, local/prconf2-163 744afc28 (3 commits each; same identity and trailer).
- old remote heads on the public fork (git ls-remote, V): fix/frozen-armor-and-resume-final c4e9db97, fix/curse-immune-final 802dee68, fix/blizzard-shard-offsets-v2 3ac63fa0. Undo = force-push these back.
- check (worker host): 7 suites exit 0 on #163 and #160 and #159 builds (build rc 0); pr_guard 0 FAIL / 0 WARN (range and each commit); presubmit --quick 0 FAIL (1, 2, 3 WARN: build skipped and test kinds); g++ and clang++ warning counts on the changed .cpp unchanged from the first rebase. Not re-run: the per-PR differentials (unchanged code paths; the first rebase passed them), clang-tidy.
- hold: no push to the public fork by this card. Commands: `git push --force-with-lease=fix/frozen-armor-and-resume-final:c4e9db97431f99508e5c8195ae855b493fbaa0ee ghfork local/prconf2-159:fix/frozen-armor-and-resume-final` (likewise 802dee68fb5b26e6c315c9509010f93dd4782590 for fix/curse-immune-final from -160, 3ac63fa06c3b3e21ca53c6eee6d1fbfd4dff84f4 for fix/blizzard-shard-offsets-v2 from -163).
