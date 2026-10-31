# Tasks

```yaml
change: hitlen
status: in-progress
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

- [ ] 1.1 the GH rate branch of FUN_00623f50 (player mode 4 / monster mode 3), the length formula
- [ ] 1.2 the mode end test: client tick (FUN_006217c0 / FUN_00463390) and the server's monster mode end
- [ ] 1.3 frozen / stunned / GH-mode exemptions and record +0x64 as they touch the length

## 2. Differential driver

- [ ] 2.1 tools/emu/hit_len.py: game.exe's code under unicorn, >= 100000 cases
- [ ] 2.2 before (d2d main) and after (patched) mismatch counts

## 3. Patch and regression

- [ ] 3.1 ~/d2d-lab/pr/hitlen fresh branch, author shatteredeagle, no trailer, pr_guard 0/0
- [ ] 3.2 upstream tests green; scripted-scenario hashes
- [ ] 3.3 corroboration on one hidden d2d slot (short)

## 4. Verification and docs

- [ ] 4.1 independent adversarial verifier report attached
- [ ] 4.2 docs/research/pr-get-hit-length-proof.md, -draft.md, -evidence-pack.md; changecheck clean
