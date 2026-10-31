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

## merged: upstream conflict round (agent REBASE134, 2026-10-31)

The maintainer commented on PR #134 (https://github.com/psi29a/d2decomped/pull/134#issuecomment-6544321745, GET-verified 13:58Z): another round of conflicts to resolve; `mergeable_state` was `dirty` (upstream main moved: #135 fine damage, #136 get-hit length, #143 renderer / merc FHR). Resolved LOCALLY by merging upstream/main into `fix/get-hit-rule` (no rewrite; backup ref `fix/get-hit-rule-pre-merge` = e3c1e535). Merge commit **6fc9451e** (local, NOT pushed), net diff vs upstream/main 7 files +163 -15.

Conflicts (5 files), both sides kept:
- `components/game/ai.cpp` / `ai.hpp`: `hurt(.., int fraction = 0, int hit_class = 0, bool marked = false)`: #135's 256ths carry (`life_frac`, dead-at-under-a-point) and our get-hit in one function. The decision takes THIS blow's own size in 256ths, `(damage << 8) + fraction` taken before the carried part joins (the old `damage << 8` would have dropped the fine part); the fine damage still reaches the monster's life as upstream wrote it. `heal` kept.
- `components/game/fight.cpp` (`Fight::land`): one call with `taken == blow.damage ? blow.frac : 0` (#135's soak rule, its ponytail kept) plus `blow.hit_class, blow.elemental || blow.deadly`. The player / merc side (`hurt_side`, `struck_big`, `Foe::blows`) merged without conflict and is untouched by #135 (it works on whole points); #143's merc `get_hit_ms` call coexists.
- `components/rules/combat.hpp`: `get_hit` / `get_hit_divisor` kept next to #136's `get_hit_rate` / `get_hit_ticks` / `get_hit_frame`; `Blow::frac` and `Blow::elemental` both; `missile_blow` keeps #135's `resisted_fine` and now sets `elemental` from the fine value (`fine > 0 && etype != 4`: the record's element is in 256ths, so an element under one point still marks the blow; before it needed a whole point). One semantic change, not covered by the differential (the oracle takes `marked` as an input).
- `docs/research/re/combat.md`: our section kept; the "FHR and FBR" and "monster hit recovery at an eighth" non-modelled lines dropped (both now modelled), #143's FBR line kept.

Re-run on the merged tree (worker host, `/tmp/hitrec-build`): `ninja test_combat test_game` ok; `test_combat` rc 0; `test_game` (D2_MPQ_DIR = the lab data folder, D2_PATCH_INSTALLER = the 1.14d installer) rc 0, 27 OK lines. Get-hit decision differential, oracle `tools/emu/hit_rule.py` vs `test_combat --get-hit` on the saved round-2 cases (cases2 `27625643`, game2 `e7e49d38`): **0 differ of 73,694 frames** (21,532 multi-record; player 0 of 36,904, monster 0 of 36,790), output md5 `4fdb5ced` identical to the pre-merge after2. The BEFORE numbers (13,935 upstream main, 702 for the summed version) are unchanged: they quote lines this merge does not touch. The decision harness calls `rules::get_hit`, so the plumbing (`blow_fine`, per-blow `struck_big`, two blows in one tick) is checked by reading: `struck_big` is unchanged and still loops `foe.blows` with each blow's class and mark; monsters take one `hurt` per landed blow. `pr_guard` upstream/main: 0 FAIL, 0 WARN. `presubmit --quick`: 0 FAIL, 3 WARN (the same sibling / skipped-build answers as before). The claim did not change.

## Mark derivation (merge-resolution check, agent MARKPROOF, 2026-10-31)

The question: the merge made `missile_blow` set `Blow::elemental` from the fine value (`fine > 0`), and the decision differential takes `marked` as an input, so how the mark is DERIVED from the damage record was never checked against game.exe 1.14d. Full detail: docs/research/pr-get-hit-rule-proof.md, section "Mark derivation".

- **game.exe's rule (V).** `FUN_0057c6c0` -> `FUN_0057c1e0` -> `FUN_0057bf80` resists each element field of the record IN PLACE (the resisted value is written back, a field at or under 0 becomes 0); then `FUN_0057cee0` -> `FUN_0057ce30` (0x57ce30) marks the class from the fields fire +0x10, lightning +0x1c, cold +0x24, poison +0x28 (table 0x6e16d8): any one over 0 (signed, 256ths, so one 256th counts; no 0x100 threshold, no whole-point test; magic +0x20 and physical +0x08 are not in it). Per record, i.e. per blow. With several fields it takes the first from a start index that rotates with a global counter (0x88cad0): the class bits vary, the mark does not.
- **Differential (V).** `tools/emu/hit_mark.py` + `hit_mark_harness.cpp` (the harness includes the real merged `combat.hpp` and calls `missile_blow`): 111,200 cases (11,200 boundary + 100,000 random, seed 7; every element, rolls 0, 1, 2, 127, 128, 255..257, 511..513, the 0x100000 MulDiv edge, 0x7fffffff, min < max rolls with a physical part, resistance -100..150, pierce 0..60 vs a non-immune). Merged tree as resolved (6fc9451e): fire, lightning, cold, magic, stun 0 differ over 81,000+ cases; **poison 3,923 differ**: the merge kept `blow.poison > 0`, the whole-point total `roll * length >> 8`, while the record's poison field (per tick, 256ths) is tested like the others. Fix, one line (`resisted_fine(roll, res) > 0`): **0 of 111,200 differ**. Also `multi` (20,000 records with several fields set, game.exe vs the any-of-four model): 0 differ.
- **Was the merge right.** The line quoted in the question (`fine > 0 && etype != 4` for fire, lightning, cold; magic excluded) is exactly game.exe's rule. The poison line was not; fixed.
- **Unit asserts (test_combat).** A fractional element (128/256), 1/256 and 0x10000 mark; an empty record and magic do not; resisted to nothing (immune; 1/256 at 50 %) does not, 2/256 at 50 % does; poison per tick of 1/256 marks although its total is under a point.
- **New local head** `5466cd17` (one commit on 6fc9451e, 2 files +16 -1, authored as the user, no trailer, `Agent: Claude Code`). `ninja test_combat test_game` ok; `test_combat` rc 0; `test_game` rc 0, 27 OK; decision differential on the saved round-2 cases: 0 differ of 73,694 frames, output md5 `4fdb5ced` identical to pre-merge; `pr_guard` upstream/main 0 FAIL 0 WARN; `presubmit --quick` 0 FAIL 3 WARN (same answers). The oracle is not shipped upstream: #134 does not ship `hit_rule.py` either (the clone keeps it untracked).
- **Limits.** UNVERIFIED: how the missile hit fills the poison field (taken as the roll per tick; `FUN_005a89a0` rolls it, the fill code is not traced). Domain as the #135 pack: a plain monster, magic damage reduction and absorb 0, physical resistance 0, pierce against a non-immune only (pierce through an immunity is the existing ponytail in `missile_blow`).

Push command NOT RUN (needs the user's go; plain, no force):

```sh
ssh worker.example 'cd ~/d2d-lab/pr/hitrec && git push origin fix/get-hit-rule'
```

Draft reply for the thread (not posted):

```text
@psi29a the conflicts are resolved by merging upstream main into the branch (no history rewrite); the decision runs on each blow's own size in 256ths, and the element mark is now checked against game.exe (111,200 cases, 0 differ) with one extra commit for the poison field.
test_combat, test_game and the get-hit differential (0 of 73,694 frames) re-run clean on the result.

Agent: Claude Code
```

## Posted log

- posted: 2026-10-31 PR #134 opened (head d55973e1 at 00:10Z, opened 00:16Z) https://github.com/psi29a/d2decomped/pull/134 go: user, ref not recorded
- posted: 2026-10-31 follow-up commit e3c1e535 pushed to PR #134 (00:43Z) https://github.com/psi29a/d2decomped/pull/134/commits/e3c1e535b9fa371b4327905035b27f78ca4afc9e go: user, ref not recorded
- posted: 2026-10-31 review replies on PR #134 (00:45Z, 02:08Z) https://github.com/psi29a/d2decomped/pull/134#discussion_r4429655638 , https://github.com/psi29a/d2decomped/pull/134#discussion_r4429655757 , https://github.com/psi29a/d2decomped/pull/134#discussion_r4429926697 go: user, ref not recorded
- Verified 2026-10-31 by read-only `gh api` GET (pulls/134, pulls/134/commits, pulls/134/comments).
