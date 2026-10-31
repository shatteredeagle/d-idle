# PR #128 review: CodeRabbit findings, analysis and draft replies

> status: executed 2026-10-30: fix commits 725033ea and d88c8c3e pushed to PR #128 (fast-forward c6efb68b..d88c8c3e); replies posted in both CodeRabbit threads; the Druid reply says 18, not 15. Original draft text below. Tags: V = read in code or run on the worker host, S = secondary, U = unverified.

Findings read in full (GET only, `gh api repos/psi29a/d2decomped/pulls/128/{comments,reviews}`, review at 21:34Z, head c6efb68b, base 2a04c18e). The PR head is a merge of main into our branch, so the worker host clone `~/d2d-lab/pr/cast-ticks` (1edc5f20) was stale; the fix is built on a fresh clone of the PR head: `~/d2d-lab/pr/pr128-fix/repo` (branch `fix128`, two commits on c6efb68b). Patches (kept on the worker host only, GPL code): `~/d2d-lab/pr/pr128-fix/patches/`.

## Finding 1: fight.cpp:456, Major, "Scale the SC action time with the cast rate": VALID

- V: `Fight::player_modes` (SC branch) tests `now_ms >= mode_ms + player_anim(kModeSC).action_ms()`, the unscaled action time, then clears the mode when `now_ms >= pmode_until`. The missile and spot branches (fight.cpp ~976, ~980, ~1004) divide by `actor->prate`. SC was the odd one out.
- V (arithmetic): Sorceress SC, 14 frames, speed 256, FCR 105: EFCR = floor(120*105/225) = 56, speed = 256*156/100 = 399, `cast_ticks` = ceil(3584/399) - 1 = 8, so `pmode_until` = 320 ms. An action frame of 9 is due at 360 ms unscaled. The mode clears first and the cast (portal, summon, state, missile) never fires. CodeRabbit's numbers hold (the action frame 9 itself is S: its comment, not re-read from the data; test_game has no d2data.mpq here).
- It predates the PR (master used the same unscaled test with `speed_frames`), but the PR shortens the mode by one more tick and is about cast timing, so it belongs here.
- Original game: the event fires when the animation position (+0x44, +256 per frame step scaled by the speed +0x4c) passes the action frame, so it scales with the cast rate (V for the rate/end test from our trace: `00623f50`, `006217c0`, `00463390`; U for the event byte path `00623e00`, not traced here).
- Fix: divide by `prate` as the other branches do. `prate = length_ms / len` so the scaled time is at most `len`, and the action check runs before the clear, so it always fires.
- Not tested by a unit test: the path needs game data (test_game SKIPs without d2data.mpq). `test_combat` and `test_game` pass with the fix (V, the worker host).

## Finding 2: skills.md:1183, Minor, "Qualify the cast-tick comparison": VALID

- V: the table counts `frames_shown` of the older formula, which assumes AnimData speed 256. `rules::cast_ticks(15, 208, 0)`: ceil(256*15/208) - 1 = 19 - 1 = 18. The Druid row says 15. So "one more than the ticks a cast takes" is true at speed 256 only (Sorc 14 -> 13 ticks) and false for the Druid (DZSCHTH speed 208).
- The separate PR `fix/skills-doc-cast` (held) already rewrites the Druid row as `0→19 ...` (ticks + 1). Here the minimal change is to qualify the sentence and name the Druid, not to edit the row (that is the other PR's claim). Possible trivial merge conflict between the two in skills.md (neighbouring lines).
- Fix: sentence qualified, with the 18-tick example.

## Fix commits (author and committer shatteredeagle, no trailer)

1. `A self cast fires on its action frame at the cast rate, not at the unscaled one` (fight.cpp, one line)
2. `skills.md: the cast table's one-more-than-ticks holds at AnimData speed 256 only`

Checks (V, the worker host, fresh build in `~/d2d-lab/pr/pr128-fix/b`, `cmake -G Ninja -DD2D_WERROR=OFF`, `ninja test_combat test_game`): `tests/test_combat` exit 0, `tests/test_game` exit 0 (data checks SKIP: no d2data.mpq), `python3 tools/pr_guard.py up/main`: 0 FAIL, 0 WARN. The upstream's contributor rules were followed: agent disclosure goes in the PR body (`Agent:` line), commits are user-authored.

## Draft replies (not posted)

Reply to finding 1:

> Right, thanks. `player_modes` compared the unscaled `action_ms()` for SC, while the missile and spot branches divide by `prate`. At 105 FCR a 14-frame Sorc cast is 8 ticks (320 ms) but the unscaled action frame 9 is 360 ms, so the mode cleared first and the cast never fired. It was already like that before this PR, but this PR shortens the mode by a tick, so I fixed it here: the SC branch now divides by `prate` like the others (<hash>). `test_combat` and `test_game` pass; there is no unit test for it because the path needs the game data (test_game skips without d2data.mpq).

Reply to finding 2:

> Right. The table counts `frames_shown` of the older formula, which assumes AnimData speed 256; the one-more-than-ticks statement holds there only. The Druid's SC is speed 208, so `cast_ticks(15, 208, 0)` is 18, not 14. Qualified the sentence and named the Druid in <hash>. The row itself is rewritten in the separate skills-doc PR, so I left it alone here.

## Commands once the user says go (do NOT run before)

```sh
ssh worker
cd ~/d2d-lab/pr/pr128-fix/repo && git log --format='%h %an %s' -3      # check author and the two commits
git push https://github.com/shatteredeagle/d2decomped.git fix128:fix/cast-ticks   # fast-forward on top of c6efb68b, no force
# then reply in each thread (fill in the pushed hashes):
gh api repos/psi29a/d2decomped/pulls/128/comments/<comment_id>/replies -f body='...'
```
Comment ids: `gh api repos/psi29a/d2decomped/pulls/128/comments --jq '.[]|.id,.path'`. The worker host clone `~/d2d-lab/pr/cast-ticks` is behind the PR head; sync it with `git fetch ../pr128-fix/repo fix128 && git merge --ff-only FETCH_HEAD` only after the push if needed.
