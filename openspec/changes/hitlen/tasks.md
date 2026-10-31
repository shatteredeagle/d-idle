# Tasks

```yaml
change: hitlen
status: in-progress
result: 'get-hit length PR candidate: differential vs game.exe FUN_00623f50 + FUN_005539b0: 102,980/110,836 differ before, 0 after; verifier HOLDS WITH LIMITS; patch eb89646e in the clone, nothing pushed; not verified live'
agent: hitlen
branch: agent/hitlen
worktree: ~/code/d2lod-wt/hitlen
owns:
  - docs/research/pr-get-hit-length-*.md
  - tools/emu/hit_len*.py
exclusive:
  - upstream-pr:get-hit-length
gates:
  - "differential vs game.exe's GH length code: >= 100000 random + boundary cases, mismatches before > 0, 0 after the patch"
  - "upstream tests green on the patched clone; pr_guard 0 FAIL 0 WARN"
  - "one independent adversarial verifier report attached"
  - "nothing pushed or posted to GitHub"
```

## 1. Trace (game.exe 1.14d, addresses only)

- [x] 1.1 the GH rate branch of FUN_00623f50 (player mode 4 / monster mode 3), the length formula
- [x] 1.2 the mode end test: client tick (FUN_006217c0 / FUN_00463390) and the server's monster mode end
- [x] 1.3 frozen / stunned / GH-mode exemptions and record +0x64 as they touch the length

## 2. Differential driver

- [x] 2.1 tools/emu/hit_len.py: game.exe's code under unicorn, >= 100000 cases
- [x] 2.2 before (d2d main) and after (patched) mismatch counts

## 3. Patch and regression

- [x] 3.1 ~/d2d-lab/pr/hitlen fresh branch fix/get-hit-length eb89646e, author shatteredeagle, no trailer, pr_guard 0/0
- [x] 3.2 upstream tests green (test_combat ok, test_game 23 OK, pr_guard 0/0, presubmit 0 FAIL); scripted-scenario hashes NOT run (pair.sh runs the lab fork, not before/after upstream trees)
- [ ] 3.3 corroboration on one hidden d2d slot: NOT run (cmd/d2dcasts2 needs the lab fork's debug verbs on the patched tree; time box)

## 4. Verification and docs

- [x] 4.1 independent adversarial verifier report attached
- [x] 4.2 docs/research/pr-get-hit-length-proof.md, -draft.md, -evidence-pack.md; changecheck clean
