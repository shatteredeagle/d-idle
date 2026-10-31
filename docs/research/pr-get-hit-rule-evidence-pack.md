# Evidence pack: PR "get-hit by hit class and share of life" (agent HITREC, 2026-10-30)

> status: current (card hitrec). One page for the user's go / no-go. Full proof: docs/research/pr-get-hit-rule-proof.md; body: docs/research/pr-get-hit-rule-draft.md. PR #134 follow-up pushed and replied 2026-10-31 (commit e3c1e535).

## PROOF: complete (for the narrowed claim), with disclosed limits

| Requirement (memory d2-pr-proof-standard) | Status | Evidence |
|---|---|---|
| Differential vs game.exe's own code | done | `tools/emu/hit_rule.py` runs `FUN_0057cee0` (the whole chain: class fix-up `FUN_0057ce30`, stun skip, `FUN_0057cb00`, the mode call) natively under unicorn; 2264 boundary + 100,000 random cases, seed 7, 16 s |
| Mismatches before the patch | **19,343 / 102,264** | d2d's old lines (fight.cpp:4903, ai.cpp:2853) in `tools/emu/hit_rule_harness.cpp` |
| Mismatches after the patch | **0 / 102,264** (round 1); **0 / 73,694 frames** incl. 21,532 multi-record (round 2, PR #134 review) | upstream's own `tests/test_combat --get-hit` on the patched clone |
| No U link in the causal chain | closed | +0x28 = poison per tick; hit class = record +0x60 after `FUN_0057ce30` (elements / deadly / 13); flag 0x2000 = deadly strike; only caller `FUN_0057cee0`. Narrowed around: record +0x64 (treated as 0), the length claim (not in this PR) |
| Upstream tests + guards | green | test_combat ok; test_game 23 OK (one scenario's damage made sure for every class); pr_guard 0 FAIL 0 WARN; presubmit 0 FAIL 3 WARN (answered in the draft) |
| Unchanged state apart from the claimed field | partial | the 23 scripted `test_game` scenarios and `test_combat` pass unchanged except the one GH expectation; `tools/d2ddet/pair.sh` NOT run (it checks determinism of one binary, not before / after) |
| Statistical corroboration | not run | `cmd/d2dcasts2` needs the lab fork's debug verbs; time box. It would be corroboration only |
| Independent adversarial verification | HOLDS WITH LIMITS | fresh 102,264 + 34 adversarial cases 0 differ; three fixes applied (flag 0x2000 wording, magic not a mark, poison marks the player's blow), re-run 0 differ; limits 3 to 8 in the proof doc |
| Review round (PR #134, CodeRabbit) | fix e3c1e535 pushed, replied 2026-10-31 | Major finding valid: per-record test in game.exe (`0x56ba73` / `0x56ba7f` pairs); d55973e1 differed in 702 multi-record frames, e3c1e535 in 0; replies drafted in pr-get-hit-rule-review-draft.md |
| One claim per PR | yes | the decision only; the length (50 + FHR) is a second, separate claim, not prepared |
| Their rules | yes | fresh clone, branch off upstream/main, author/committer shatteredeagle, no Claude trailer, no tools/ or CI file, template filled, `Agent:` line |

## What the PR changes in play (V from the rule, not measured live)

Sorceress (60 life) hit by a Carver for 4: game.exe flinches 37.5 % (over a sixteenth), d2d never (under a twelfth) -> now 37.5 %. Rakanishu's 10 on 60: 75 % instead of always. A Zombie (7 to 12 life) hit by a 4-point Charged Bolt: sure on both (over a quarter). A sword (class 2) hit without elements on a 100-life monster: an eighth, 37.5 % from 12.5 instead of sure from 12.5; the same sword with a fire charm: a sixteenth.

## Recommendation

Go for the push and the PR (commit `d55973e1`): the claim is narrow, the differential is against game.exe's own code with zero mismatches, and the limits are written into the body. Keep the length claim (DECOMPREC section 3: 50 + FHR; the monster mode end U) for a second PR after its own differential (the client tick count is already oracled by `move_recovery.py rate`; the server-side monster mode end needs one more trace).

## Commands (NOT RUN; each needs the user's go)

```sh
ssh worker.example 'cd ~/d2d-lab/pr/hitrec && git fetch upstream && git rebase upstream/main && python3 tools/pr_guard.py upstream/main'   # if main moved
ssh worker.example 'cd ~/d2d-lab/pr/hitrec && git push origin fix/get-hit-rule'
# save the body (the text inside the four-backtick fence of pr-get-hit-rule-draft.md) to ~/d2d-lab/pr/hitrec/PR_DESCRIPTION.md, then:
ssh worker.example 'cd ~/d2d-lab/pr/hitrec && gh pr create --repo psi29a/d2decomped --base main --head shatteredeagle:fix/get-hit-rule --title "Get-hit by hit class and share of life" --body-file PR_DESCRIPTION.md'
```

Rebuild the proof after a rebase: the four `hit_rule.py` / `test_combat --get-hit` lines in pr-get-hit-rule-proof.md "Differential" (`/tmp/hitrec-build` on the worker host; `ninja test_combat test_game` first).

## Posted log

- posted: 2026-10-31 PR #134 opened (head d55973e1 at 00:10Z, opened 00:16Z) https://github.com/psi29a/d2decomped/pull/134 go: user, ref not recorded
- posted: 2026-10-31 follow-up commit e3c1e535 pushed to PR #134 (00:43Z) https://github.com/psi29a/d2decomped/pull/134/commits/e3c1e535b9fa371b4327905035b27f78ca4afc9e go: user, ref not recorded
- posted: 2026-10-31 review replies on PR #134 (00:45Z, 02:08Z) https://github.com/psi29a/d2decomped/pull/134#discussion_r4429655638 , https://github.com/psi29a/d2decomped/pull/134#discussion_r4429655757 , https://github.com/psi29a/d2decomped/pull/134#discussion_r4429926697 go: user, ref not recorded
- Verified 2026-10-31 by read-only `gh api` GET (pulls/134, pulls/134/commits, pulls/134/comments).
