# Tasks

```yaml
change: prproof
status: archived
agent: prproof
branch: agent/prproof
worktree: ~/code/d2lod-wt/prproof
owns:
  - docs/process/upstream-pr-standard.md
  - docs/research/pr-missile-damage-fine-proof.md
  - docs/research/pr-missile-damage-fine-draft-v2.md
  - docs/research/pr-missile-damage-fine-evidence-pack.md
  - openspec/changes/prproof/**
exclusive:
  - upstream-pr:missile-damage-fine
gates:
  - "docs/process/upstream-pr-standard.md <= 90 lines"
  - "differential oracle vs d2d: >= 100k random cases + exhaustive boundaries, 0 mismatches after the patch"
  - "pr_guard 0 FAIL 0 WARN; ninja test_combat test_game green on the v2 branch"
  - "independent verifier report attached to the proof doc"
  - "go run ./cmd/changecheck: 0 warnings"
merge: 80f699e
result: not live; differential proof (oracle vs real d2d) zero mismatches over 57k boundary lines + 100k sequences; verifier holds with limits; v2 branch on the worker host, nothing pushed
```

## 1. Standard

- [x] 1.1 docs/process/upstream-pr-standard.md: differential proof, V/S/U chain, statistics as corroboration, scope, adversarial verification, limits, checklist + evidence pack format, spacing and disclosure (wc -l <= 90)

## 2. Proof for fix/missile-damage-fine

- [x] 2.1 Trace the apply path in game.exe (FUN_0057c6c0, FUN_0057c1e0, FUN_0057bf80, FUN_0057be00, FUN_00483360, FUN_0057bf10, FUN_0057a980) with disasm.py; identify the hooks that are not the claim
- [x] 2.2 Oracle tools/emu/life_apply.py (real game.exe code) and `test_game --life-apply` (real d2d code), same xorshift32 cases, diffed
- [x] 2.3 Improve the patch where not exact: resistance on the fine roll by MulDiv's branches (`resisted_fine`), dead = life 0, `heal()` clearing the fraction only when the cap cuts (v2 branch fix/missile-damage-fine-v2, author the user, no trailer)
- [x] 2.4 Their checks: pr_guard 0/0, presubmit 0 FAIL 3 WARN (answered in the body), test_combat ok, test_game rc 0 with D2_MPQ_DIR / D2_PATCH_INSTALLER
- [x] 2.5 Independent adversarial verifier (second agent), report in the proof doc
- [x] 2.6 docs/research/pr-missile-damage-fine-proof.md, -draft-v2.md, -evidence-pack.md
