# Evidence pack: PR "Draw a get-hit once at its rate, held on the last frame" (agent RENDERGH, 2026-10-31)

> status: current (card rendergh). One page for the user's go / no-go. Full proof: docs/research/pr-get-hit-render-proof.md; body: docs/research/pr-get-hit-render-draft.md. Nothing pushed, posted or filed. Stacked on PR #136: it must wait for #136 to merge (then rebase onto main) or be retargeted at `fix/get-hit-length`.

## PROOF: complete for monsters and the merc (with disclosed limits; verifier verdict below)

The claim in one sentence: while a monster or the merc is in its get-hit, d2d now draws at each tick of the hold the frame game.exe's client draws (the position stepped by the get-hit rate while under the end, shown >> 8: once through, then held on the last stepped frame), instead of the base-rate loop.

| Requirement (docs/process/upstream-pr-standard.md) | Status | Evidence |
|---|---|---|
| Differential vs game.exe's own code | done | `tools/emu/hit_render.py`: `FUN_004b1280` (the step with the real mode-table row at 0x7252e0 + 36), `FUN_00621810` (the drawn frame), `FUN_00623f50` (the rate), `FUN_005539b0` (the hold) run natively under unicorn; 10,836 boundary + 100,000 random cases, seed 7, about 4 min |
| Mismatches before the patch | **1,488,667 ticks in 49,641 of 55,089 monster cases** | d2d's old lines (town.cpp:195..203, world_view.cpp:440, world_view.hpp:164..165) in `tools/emu/hit_render_harness.cpp` |
| Mismatches after the patch | **0 of 1,747,580 ticks** | upstream's `tests/test_combat --get-hit-frames` on the patched worktree |
| No U link in the causal chain | closed for the claim | the tick dispatcher, the row lookup and its data, the over test, the step, the frame getter and its draw caller, the client's mode start, the server's hold: read or run. U outside the claim: `FUN_00463720` (after the hold), the player's +0x154 end path |
| Upstream tests + guards | green | test_combat ok; test_game 23 OK rc 0; pr_guard 0 FAIL 0 WARN (against upstream/main and fix/get-hit-length); presubmit --quick 1 FAIL (`doc-paths`: #136's `tools/emu/hit_len.py` line, shipped by the base's new commit 9006a688) + 4 WARN answered in the draft |
| Unchanged state apart from the claimed field | by construction + tests | `get_hit_rate` is set only for mode "GH" (two lines); `get_hit_elapsed` returns the old value otherwise; the 23 `test_game` scenarios and `test_combat` pass; `tools/d2ddet` NOT run |
| Statistical corroboration | absent | none needed; the claim is per-tick exact |
| Independent adversarial verification | see below | general-purpose, Fable, read-only; given the code and the oracle, not the conclusion |
| One claim per PR | yes | the drawn frame of monsters and the merc; the player's frames are a listed gap, the hold is #136 |
| Their rules | yes | worktree of the user's fork clone, author / committer shatteredeagle, no trailer, body `Agent: Claude Code`, no tools/ or CI file in the commit, template filled, every known gap a `// ponytail:` |
| Spacing | one candidate | after #136 (and #134 / #135 if open); push >= 1 h after the last |

## What the PR changes in play (V from the rule, not measured live)

A Zombie flinching on a Charged Bolt: its 5-frame stagger plays once over the 17-tick hold (about 3 ticks a frame) and then stands, instead of replaying three and a half times at the base rate. A Fallen (7 at 256): 14 ticks, two a frame. The merc the same. Nothing moves for the player's own sprite, objects, other modes.

## Recommendation

Go for the PR after #136 lands (rebase onto main: the base's commit 9006a688 added an ai.cpp ponytail this patch supersedes and reworded combat.md's "Ported" bullet; resolve, delete the note, re-run the AFTER differential and the checks). Or, if the maintainer wants it sooner, push `fix/get-hit-render` and open the PR with base `fix/get-hit-length` (GitHub then retargets to main when #136 merges). The player's frames are a clean follow-up candidate (the oracle already measures them).

## Commands (NOT RUN; each needs the user's go)

```sh
# the worker host: rebase onto the moved base, re-check
ssh worker.example 'cd ~/d2d-lab/pr/render && git rebase 9006a688'            # conflicts: docs/research/re/combat.md (keep both bullets); then delete the 8-line ponytail 9006a688 added in components/game/ai.cpp set_mode
ssh worker.example 'cd ~/d2d-lab/pr/render && ninja -C /tmp/render-build test_combat test_game && /tmp/render-build/tests/test_combat && python3 tools/pr_guard.py upstream/main && python3 tools/presubmit.py --quick'
ssh worker.example 'cd /tmp/render-emu && /tmp/render-build/tests/test_combat --get-hit-frames < cases.tsv > after.tsv && source /tmp/casttrace-venv/bin/activate && PYTHONPATH=~/d2d-lab/pr/hitlen/tools/emu D2_MPQ_DIR=~/d2d-lab/data python hit_render.py compare game.tsv after.tsv cases.tsv 1'
ssh worker.example 'cd ~/d2d-lab/pr/render && git push origin fix/get-hit-render'
# save the body (the text inside the four-backtick fence of pr-get-hit-render-draft.md) to ~/d2d-lab/pr/render/PR_DESCRIPTION.md, then (after #136 merges: --base main after a rebase onto upstream/main):
ssh worker.example 'cd ~/d2d-lab/pr/render && gh pr create --repo psi29a/d2decomped --base fix/get-hit-length --head shatteredeagle:fix/get-hit-render --title "Draw a get-hit once at its rate, held on the last frame" --body-file PR_DESCRIPTION.md'
# then attach tools/emu/hit_render.py and hit_render_harness.cpp to the PR conversation (the oracle; never game data); log the post in this file
```

## Posted log

- nothing posted.
