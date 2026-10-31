# PR #134 review, round 2: the CodeRabbit comments re-checked against game.exe (agent PR134ROUND2, 2026-10-30)

> status: done; correction reply posted 2026-10-31 (thread 4429578410). (card pr134r2). Read with GET only (`gh api repos/psi29a/d2decomped/pulls/134`, `/reviews`, `/comments`, `/issues/134/comments`, `/commits/e3c1e535/check-runs`); nothing posted, pushed or edited on GitHub. Round 1 (the fix and the posted replies): docs/research/pr-get-hit-rule-review-draft.md; the proof: docs/research/pr-get-hit-rule-proof.md. Tags: V = run in game.exe's own code under unicorn (`tools/emu/hit_rule.py`) or read in the capstone dump (`/tmp/casttrace/all.txt` on the worker host); V-d2d = read or run in the clone `~/d2d-lab/pr/hitrec` at the PR head; S = secondary; U = unverified.

## What the PR looks like now (read 2026-10-31 ~01:10Z)

| When (Z) | What |
|---|---|
| 00:10 | `d55973e1` committed and pushed (PR opened, head d55973e1 on base 32768ddb) |
| 00:16 | CodeRabbit summary comment 6536351767 (`head_sha=d55973e1`), two findings listed |
| 00:24 | CodeRabbit review 5720082484 on `d55973e1`: two actionable inline comments, 4429578410 (fight.cpp:4800, also :4907, Major) and 4429578414 (test_combat.cpp:68, Minor) |
| 00:43 | `e3c1e535` "Get-hit once per landed blow, not on the frame's sum" committed and pushed (head now e3c1e535) |
| 00:45 | our replies 4429655638 and 4429655757 posted (the round-1 drafts, word for word); CodeRabbit's auto-notes "Addressed in commit e3c1e53" / "Confirmed as addressed", its reply 4429657511 ("addresses this finding for both the player and the mercenary ... I have not rerun test_combat or test_game"), both threads resolved |
| 00:45..01:00 | checks on e3c1e535: guard, Linux (GCC), macOS (Clang), Windows (MSVC), Flatpak x86_64 and aarch64 all `success`; CodeRabbit `neutral` (its incremental review of e3c1e535 posted nothing new); "Publish release assets" skipped (release workflow, not a PR check) |

So the review with the two actionable comments is on `d55973e1` and predates our second push; there is NO review or comment on `e3c1e535`. The two threads are answered and resolved. `mergeable_state: blocked` is the branch protection waiting for the maintainer's review. Maintainer activity (psi29a): none (0 reviews, 0 issue comments, 0 review comments, no reactions, no requested reviewers, no labels, no assignee).

## Finding 1: fight.cpp:4800 / :4907, Major, "Evaluate get-hit for each landed blow": VALID, already fixed in e3c1e535 (V)

What CodeRabbit said: `Foe::take` adds every blow of the frame, but the get-hit is tested once on the total with one hit class, so two class-2 blows of 6 on 64 max life (each under the eighth) could trigger a get-hit together.

- V (game.exe, dump): the get-hit is decided once per damage record. `FUN_0057c6c0` (apply one record) and `FUN_0057cee0` (the takes-damage event, the `FUN_0057cb00` size test on result flag 4) are called as a pair per hit at `0x56ba73` / `0x56ba7f`, `0x57d5c7` / `0x57d658` and `0x5ad8c6` / `0x5ad8d3` (`grep -n "^0056ba7\|^0057d5c\|^0057d65\|^005ad8[cd]" /tmp/casttrace/all.txt`: the six `call` lines, re-read by this agent). Two hits in a tick are two records, two tests, two seed draws.
- V-d2d: on `d55973e1`, `hurt_side` tested `side[1].damage` (merc, :4800) and `foe.damage` (player, :4903) once with the last blow's class. `e3c1e535` keeps every landed blow (`Foe::Blow { damage, hit_class, marked }`, `Foe::blows`, pushed in `Foe::take`; `struck_by` sets the class at the six sites; the aura pulse and the burning-ground missile push their own blow) and `struck_big` in `hurt_side` tests `rules::get_hit` once per blow for the player (d2d's `rng`) and the merc (`merc->seed`), the absorb pool taken off the blows in landing order. The loop runs every blow (a later blow still rolls after an earlier get-hit, as game.exe re-tests each record).
- V (differential, re-run by this agent on the clone at `e3c1e535`, `/tmp/pr134r2/` on the worker host; `test_combat` and `test_game` were up to date: `ninja: no work to do`):

| Side | seed 7 (the documented run) | seed 1342 (fresh) |
|---|---|---|
| frames (of which 2 or 3 records on one unit) | 73,694 (21,532) | 73,979 (21,260) |
| upstream main 32768ddb (a twelfth / eighth on the frame's sum, `hit_rule_harness`) | 13,935 (player 6,324, monster 7,611) | 14,118 (player 6,255, monster 7,863) |
| `d55973e1` model (`rules::get_hit` on the sum, `hit_rule_harness --sum`) | **702** (all player, all multi-record: 0 of 52,162 single-record frames; 546 get-hits game.exe does not give, 156 it gives and the sum did not) | **702** (all player, all multi-record; 545 / 157) |
| `e3c1e535` (per blow, upstream's `test_combat --get-hit`) | **0** | **0** |

The same 702 on both seeds is a coincidence (seed 7: 5 boundary frames + 697 random; seed 1342: 2 + 700; different frame ids). Checksums (md5 prefixes) of the seed-7 files equal the round-1 ones: cases `27625643`, game `e7e49d38`, after `4fdb5ced`, main `0afd13c4`, sum `f3712806`; seed 1342: cases `627ae6ce`, game `d7025c61`, after `b2ec8b07`, main `e71a8933`, sum `8197efd7`.

- One detail of the posted reply is not exact. It says "your two-sixes-on-64 case among them". In the seed-7 run the 64-life frame of that case (frame 2264) gives no get-hit on all three sides: the sum (12 of 64, class 2: the 75 % band) rolled a no with that frame's unit seed. The same frame with another seed (seed 1342, frame 2264: game 0, sum 1) is a mismatch, and the boundary frames next to it are (seed 7: 2266, 2283, 2294, 2302, 2311, e.g. two class-3 blows of 2 on 64 and three of 6 on 333: game 0, sum 1). The substance (the sum reaches a roll game.exe never makes) holds; the specific frame did not differ in that run. A one-line correction is drafted below; posting it is the user's call (the thread is resolved and the reader is a bot).

```sh
# the worker host, re-run by PR134ROUND2 (the clone at e3c1e535, the oracle = the copy on master tools/emu/hit_rule.py, md5 4eb70bbd)
cd ~/d2d-lab/pr/hitrec/tools/emu && source /tmp/casttrace-venv/bin/activate
python hit_rule.py gen 100000 7 > /tmp/pr134r2/cases-7.tsv                                               # 73,694 frames
D2_MPQ_DIR=~/d2d-lab/data python hit_rule.py --exe /tmp/casttrace/game.exe game /tmp/pr134r2/cases-7.tsv > /tmp/pr134r2/game-7.tsv   # 15 s
g++ -O2 -std=c++20 -o /tmp/pr134r2/harness /tmp/hit_rule_harness.cpp   # = master tools/emu/hit_rule_harness.cpp
/tmp/pr134r2/harness < /tmp/pr134r2/cases-7.tsv > /tmp/pr134r2/main-7.tsv
/tmp/pr134r2/harness --sum < /tmp/pr134r2/cases-7.tsv > /tmp/pr134r2/sum-7.tsv
/tmp/hitrec-build/tests/test_combat --get-hit < /tmp/pr134r2/cases-7.tsv > /tmp/pr134r2/after-7.tsv
python hit_rule.py compare /tmp/pr134r2/game-7.tsv /tmp/pr134r2/main-7.tsv /tmp/pr134r2/cases-7.tsv    # 13935 differ
python hit_rule.py compare /tmp/pr134r2/game-7.tsv /tmp/pr134r2/sum-7.tsv /tmp/pr134r2/cases-7.tsv     # 702 differ
python hit_rule.py compare /tmp/pr134r2/game-7.tsv /tmp/pr134r2/after-7.tsv /tmp/pr134r2/cases-7.tsv   # 0 differ
# the same with `gen 100000 1342`: 14118 / 702 / 0
```

Regression on the clone at `e3c1e535` (V-d2d, this agent): `ninja test_combat test_game` no work to do; `./tests/test_combat` -> `ok`; `D2_MPQ_DIR=~/d2d-lab/data D2_PATCH_INSTALLER=~/Downloads/LODPatch_114d.exe ./tests/test_game` -> 23 `OK`, rc 0; `python3 tools/pr_guard.py upstream/main` -> 0 FAIL, 0 WARN; `git status` clean apart from the untracked `PR_DESCRIPTION.md` and `tools/emu/hit_rule.py` (the oracle, not part of the PR).

Verdict: VALID; fixed by `e3c1e535`, which is already the PR head; no further fix commit.

## Finding 2: tests/test_combat.cpp:68, Minor, one-letter loop index: VALID, fixed in e3c1e535 (V-d2d)

`git show e3c1e535 -- tests/test_combat.cpp`: `for (int i = 0; i < 4000; ++i)` -> `for (int sample = 0; sample < 4000; ++sample)` (the band-count loop). No other one- or two-letter name is introduced by `e3c1e535` (its new names: `Blow`, `blows`, `struck_by`, `struck_big`, `cut`, `big`, `struck`, `absorbed`, `carry`, `frame_recovers`, `frame_id`, `next`). Verdict: VALID; done; no further action.

## Fix commit

None needed: both findings are addressed by `e3c1e535`, which is the PR head, pushed, with all checks green. No new commit was made on the clone; its branch is unchanged (`e3c1e535` on `d55973e1`, author and committer shatteredeagle, no trailer).

## Verifier (independent adversarial agent, general-purpose, Fable, read-only)

Brief: break the claim that e3c1e535 decides the get-hit once per landed blow as game.exe does per record; 15 min; read-only on the clone, GET only on GitHub. Its report, verbatim:

> VERDICT: HOLDS WITH LIMITS. The per-blow decision in e3c1e535 matches game.exe's FUN_0057cee0 on every frame I threw at it; the limits are modelling gaps already noted as ponytails, not errors in the claim.
>
> What I ran (all on worker, outputs under /tmp/pr134r2-verif/):
> - Fresh seed: `hit_rule.py gen 100000 909090 > cases.tsv`, `game`, `test_combat --get-hit`, `hit_rule_harness --sum`, `compare`: 73606 frames (21518 multi-record). e3c1e535: **0 differ**. Summed model: **743 differ** (player 743/36856; monster 0 because the harness models d2d's monster hurt() per blow, not a sum).
> - Hand-written hand-cases.tsv (5440 frames, 13440 records, 40 seeds x 17 scenarios x 4 max-life x 2 kinds): mixed classes 2/3, three and four records, sure-GH first then roll bands, stunned (incl. 0 and 0x80 damage), frozen, poison-only first/second beside physical, rec64 on one of two class-5 blows, f2000 on one, exotic classes 0x7f/14/0, 0xff/0x100 records, mgh=0. e3c1e535: **0 differ**; summed: **196 differ**.
> - GH-already-set probe: copy hit_rule_gh.py presets unit+0x10 to 4 (player) / 3 (monster) before each record: identical output to the normal run (0 differ). game.exe re-tests and re-rolls a unit already in GH; only mode 0/0x11 (player, 0x57d2f0) or 0/0xc (monster, 0x57cfb3) skip. So struck_big rolling on after a success is correct.
>
> Code findings:
> - foes is a per-frame local (fight.cpp:4581), so blows need no clear: non-issue. monster_auras (4588) runs before hurt_side (4749): its blows are tested the same frame.
> - Every take() with a landed hit is followed by struck_by except corpse_blast fight.cpp:1793, fire_blast :1812, corpse-explode ai.cpp:2241: blow hit_class stays 0 -> divisor 16, and game.exe marks those (fire) -> 16 too; outcome identical, pre-existing cosmetic.
> - Absorb pool taken off blows in landing order (fight.cpp struck_big), merc passes absorbed 0, player rolls on d2d's rng not the unit seed: documented limits, unverified.
> - Pre-existing, outside this commit: d2d's blow damage excludes poison; game.exe's record total (+0x4c) includes it for a mixed phys+poison record (FUN_0057cb00 0x57cb22-0x57cb2c), so a mixed blow is tested on a smaller number in d2d. Limit.
> - test_combat --get-hit re-implements hurt_side's rule; the real merc/player hurt_side code is only exercised by test_game's single-blow arrow scenario. Limit.
>
> Not checked: live game, Energy Shield ordering, FUN_0057c6c0's state event, d2d's monster hurt() path.

What follows from it: no fix. The three `take()` sites without `struck_by` (corpse blast, fire blast, corpse explode) give the same divisor game.exe would (16, an elemental record), so the outcome is the same; the poison-in-the-total and shared-rng points are limits 3 and 8 of the proof doc (already disclosed), and "test_combat --get-hit re-implements the rule" is a known shape of the differential (the real `hurt_side` is covered by `test_game`'s arrow scenario only), worth a sentence if the maintainer asks.

## Draft replies (NOT posted; both threads are already answered and resolved)

Nothing is owed on either thread. One optional correction to our reply on 4429578410, if the user wants the record exact:

> One correction to my count above: in that run the 64-life frame of your example itself rolled no get-hit on the sum (12 of 64 for class 2 is the 75 % band, and that frame's seed rolled the 25 %), so it is not among the 702; the frames beside it are (two class-3 blows of 2 on 64, three of 6 on 333: game.exe none, the sum a get-hit), and the same frame with another unit seed is. Re-run on a second seed (73,979 frames): 702 / 0 again.

## Commands once the user says go (NOT RUN)

Nothing to push: the branch and the PR head are the same commit. If the correction is wanted:

```sh
ssh worker.lan.<domain> 'cd ~/d2d-lab/pr/hitrec && git log --format="%h %an %s" -2 && git status --short'   # e3c1e535 on d55973e1, nothing to push
# save the blockquote above (without the "> ") to ~/d2d-lab/pr/hitrec/reply3.txt, then:
ssh worker.lan.<domain> "cd ~/d2d-lab/pr/hitrec && gh api repos/psi29a/d2decomped/pulls/134/comments/4429578410/replies -f body=\"\$(cat reply3.txt)\""
```

Otherwise the next step is to wait for psi29a (`gh pr view 134 --repo psi29a/d2decomped --json reviews,mergeStateStatus`). Cleanup: none left (no game started; `/tmp/pr134r2/` on the worker host holds TSV files only, 42 MB, delete when the PR is merged).
