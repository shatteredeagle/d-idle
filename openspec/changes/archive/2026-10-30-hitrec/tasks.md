# Tasks

```yaml
change: hitrec
status: archived
agent: hitrec
branch: agent/hitrec
worktree: ~/code/d2lod-wt/hitrec
owns:
  - docs/research/pr-get-hit-rule-*.md
  - tools/emu/hit_*
exclusive:
  - upstream-pr:get-hit-rule
gates:
  - "differential vs game.exe FUN_0057cee0: >= 100000 random + boundary cases, 0 differ after the patch (tools/emu/hit_rule.py compare)"
  - "upstream tests green on the patched clone: test_combat ok, test_game 23 OK; pr_guard 0 FAIL 0 WARN"
  - "one independent adversarial verifier report attached"
  - "nothing pushed or posted to GitHub"
merge: 08a2044
result: 'get-hit rule PR #134: differential vs game.exe FUN_0057cee0: 19,343/102,264 differ before, 0 after; CodeRabbit round: per-blow get-hit fix e3c1e535 (702/73,694 multi-record frames differed with the summed model, 0 with per-blow); verifier holds with limits; not verified live'
```

## 1. Trace (game.exe 1.14d, addresses only)

- [x] 1.1 close U: damage record +0x28 = poison per tick (record int 10, FUN_0057c6c0 -> FUN_0057ac50); a hit that is all poison is no get-hit (0x57cb22)
- [x] 1.2 close U: the hit class at the call is the record's +0x60 after FUN_0057ce30 (0x57cf54): 13 for none, an element's bits from the table at 0x6e16d8, 0x10 for a deadly strike (flag 0x2000, 0x57b929); so an elemental blow divides by 16
- [x] 1.3 close U: the two callers both pass that class (0x57d0c2, 0x57d374); the only caller of FUN_0057cb00 is FUN_0057cee0 (upstream game-calls.tsv)
- [x] 1.4 narrow: the server-side monster GH mode end (-1 tick) stays U; the length claim (rate 50 + FHR) is NOT in this PR

## 2. Differential driver

- [x] 2.1 tools/emu/hit_rule.py: runs game.exe's FUN_0057cee0 natively (mode calls recorded), 2264 boundary + 100000 random cases, 16 s
- [x] 2.2 before: 19343 of 102264 differ (tools/emu/hit_rule_harness.cpp quotes the old d2d lines); after: 0 differ (upstream's own test_combat --get-hit)

## 3. Patch and regression

- [x] 3.1 ~/d2d-lab/pr/hitrec branch fix/get-hit-rule d55973e1 off upstream/main 32768ddb, author shatteredeagle, no trailer, pr_guard 0/0
- [x] 3.2 test_combat ok, test_game 23 OK (one scenario's damage max/4 -> max/2: a sure band for any class)
- [ ] 3.3 scripted-scenario state hashes (tools/d2ddet pair.sh): NOT run (it compares two runs of one binary, not before/after; time box)
- [ ] 3.4 corroboration on a hidden d2d slot: NOT run (needs the lab fork's debug verbs on the patched tree; time box)

## 4. Verification and docs

- [x] 4.0 PR #134 review round: CodeRabbit Major valid (per-record test, V); fix e3c1e535 on the branch, 702 -> 0 differ on 73,694 frames; replies drafted; not pushed

- [x] 4.1 independent adversarial verifier (general-purpose, fable) report attached in pr-get-hit-rule-proof.md
- [x] 4.2 docs/research/pr-get-hit-rule-proof.md, -draft.md, -evidence-pack.md; changecheck 0 warnings
