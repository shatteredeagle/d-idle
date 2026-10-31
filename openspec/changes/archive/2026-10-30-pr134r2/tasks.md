# Tasks

```yaml
change: pr134r2
status: archived
agent: PR134ROUND2
branch: agent/pr134r2
worktree: ~/code/d2lod-wt/pr134r2
owns:
  - docs/research/pr-get-hit-rule-review2-draft.md
  - tools/emu/hit_rule*.py
  - openspec/changes/pr134r2/**
exclusive:
  - upstream-pr:get-hit-rule
gates:
  - "every CodeRabbit comment on PR #134 since d55973e1 has a verdict with evidence (V/S/U) in docs/research/pr-get-hit-rule-review2-draft.md"
  - "differential vs game.exe FUN_0057cee0 re-run on the clone at the PR head: mismatch counts before (summed model) and after (per blow) reported, after = 0 (tools/emu/hit_rule.py compare)"
  - "upstream tests green on the clone at the PR head: test_combat ok, test_game 23 OK, pr_guard 0 FAIL 0 WARN"
  - "one independent adversarial verifier report attached for the logic change"
  - "nothing pushed or posted to GitHub; replies and push commands drafted only"
merge: 0d1c848
result: 'PR #134 round 2: CodeRabbit comments were on the first commit and already addressed by e3c1e535; differential re-run seeds 7 and 1342: upstream main 13,935/14,118 differ, summed model 702/702, per-blow 0/0; independent verifier holds with limits; one inaccuracy in our reply (frame 2264 example) corrected in the thread by the user''s go; no maintainer activity'
```

## 1. Read the review (GET only)

- [x] 1.1 reviews, review comments, issue comments and the check runs of PR #134 read in full; the timeline of the review vs our pushes written down
- [x] 1.2 maintainer (psi29a) activity checked: reviews, comments, reactions, requested reviewers

## 2. Verdict per comment, with evidence

- [x] 2.1 fight.cpp:4800 / :4907 (Major, per-blow get-hit): the call pairs read in the dump (V), the differential re-run on the clone at e3c1e535 for seed 7 (the documented run) and a fresh seed: summed model > 0, per-blow 0
- [x] 2.2 test_combat.cpp:68 (Minor, loop index): the rename checked in the diff at the PR head

## 3. Fix, verifier, draft

- [x] 3.1 fix commit needed? decided from the re-run (none: e3c1e535 already on the branch and pushed)
- [x] 3.2 one independent verifier (general-purpose, Fable, read-only) on the per-blow change; verdict attached to the draft doc
- [x] 3.3 docs/research/pr-get-hit-rule-review2-draft.md: verdicts, numbers, draft replies (not posted), push and reply commands (not run)
- [x] 3.4 changecheck 0 warnings, tools/openspec.sh validate --all --strict, committed on agent/pr134r2
