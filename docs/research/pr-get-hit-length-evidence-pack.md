# Evidence pack: PR "get-hit length by AnimData speed x (50 + FHR)" (agent HITLEN, 2026-10-31)

> status: current (card hitlen). One page for the user's go / no-go. Full proof: docs/research/pr-get-hit-length-proof.md; body: docs/research/pr-get-hit-length-draft.md. Nothing pushed, posted or filed.

## PROOF: complete (with disclosed limits; verifier verdict below)

| Requirement (docs/process/upstream-pr-standard.md) | Status | Evidence |
|---|---|---|
| Differential vs game.exe's own code | done | `tools/emu/hit_len.py` runs `FUN_00623f50` (from 0x624020: the mode dispatch and the get-hit branch, `FUN_00621740` inside) and `FUN_005539b0` (the mode-end event) natively under unicorn; 10,836 boundary + 100,000 random cases, seed 7, about 1 min |
| Mismatches before the patch | **102,980 / 110,836** | d2d's old lines (fight.cpp:457..460, ai.cpp:172) in `tools/emu/hit_len_harness.cpp` |
| Mismatches after the patch | **0 / 110,836** | upstream's own `tests/test_combat --get-hit-length` on the patched clone |
| No U link in the causal chain | closed | the rate branch, the effective FHR, the mode start, the event scheduler, its two callers (player mode-4 handler, monster mode set), the event's end handlers (player NU, monster mode table) all read or run; DECOMPREC's U items (server monster mode end, +0x64, frozen / stunned) closed or moved to the decision claim |
| Upstream tests + guards | green | test_combat ok; test_game 23 OK rc 0; pr_guard 0 FAIL 0 WARN; presubmit 0 FAIL 2 WARN (answered in the draft) |
| Unchanged state apart from the claimed field | partial | the 23 scripted `test_game` scenarios and `test_combat` pass unchanged; `tools/d2ddet/pair.sh` NOT run (it runs the lab fork's binary, not before / after upstream trees) |
| Statistical corroboration | not run | `cmd/d2dcasts2` needs the lab fork's debug verbs on the patched tree; time box. Corroboration only, never the claim |
| Independent adversarial verification | see below | fresh seed, adversarial cases, hand re-read of `FUN_005539b0` / `FUN_005416b0` and the callers |
| One claim per PR | yes | the length only; the decision is PR #134; the block length is listed, not changed |
| Their rules | yes | fresh clone, branch off upstream/main 32768ddb, author / committer shatteredeagle, no Claude trailer, commit body 3 lines, no tools/ or CI file, template filled, `Agent: Claude Code` |
| Spacing | one candidate | #134 and #135 open; this is the only new one; push >= 1 h after the last |

## What the PR changes in play (V from the rule, not measured live)

A Sorceress hit by a Carver and staggered: 16 ticks (0.64 s) instead of 8 (0.32 s) at 0 FHR; with 20 FHR 12 (the table's 11 frames + 1). A Zombie flinching on a Charged Bolt: 17 ticks (0.68 s) instead of 9; a Fallen 14 instead of 7; a Fallen Shaman 10 instead of 5. Kiting gets the original's room: a d2d Zombie no longer closes in twice as fast between bolts; the player's own get-hit is twice as long too, so melee classes without FHR are hit-locked as in the original.

## Recommendation

Go for the push and the PR (commit `2648d0be`) once the verifier's findings are fixed or listed, and after #134's review settles if the maintainer prefers one combat PR at a time: the claim is narrow, the differential is against game.exe's own code with zero mismatches, the limits are in the body.

## Commands (NOT RUN; each needs the user's go)

```sh
ssh worker.example 'cd ~/d2d-lab/pr/hitlen && git fetch upstream && git rebase upstream/main && python3 tools/pr_guard.py upstream/main'   # if main moved
ssh worker.example 'cd ~/d2d-lab/pr/hitlen && git push origin fix/get-hit-length'
# save the body (the text inside the four-backtick fence of pr-get-hit-length-draft.md) to ~/d2d-lab/pr/hitlen/PR_DESCRIPTION.md, then:
ssh worker.example 'cd ~/d2d-lab/pr/hitlen && gh pr create --repo psi29a/d2decomped --base main --head shatteredeagle:fix/get-hit-length --title "Get-hit length by AnimData speed x (50 + FHR)" --body-file PR_DESCRIPTION.md'
# then attach tools/emu/hit_len.py to the PR conversation (the oracle; never game data)
```

Rebuild the proof after a rebase: the lines under "Differential" in pr-get-hit-length-proof.md (`/tmp/hitlen-build` on the worker host; `ninja test_combat test_game` first).
