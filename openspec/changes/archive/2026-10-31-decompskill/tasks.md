# Tasks

```yaml
change: decompskill
status: archived
skip_specs: true
agent: decompskill
branch: agent/decompskill
worktree: ~/code/d2lod-wt/decompskill
owns:
  - docs/research/skill-damage-trace.md
  - docs/research/pr-skill-damage-proof.md
  - docs/research/pr-skill-damage-draft.md
  - docs/research/pr-skill-damage-evidence-pack.md
  - tools/emu/skill_*.py
exclusive:
  - fork-branch:fix/skill-damage
gates:
  - "every address cited is read in game.exe 1.14d or run in the oracle (V) or tagged S/U; no U in a claim's causal chain"
  - "each deviation claimed has a differential against game.exe's own code: mismatches before > 0, after = 0, over >= 100k cases or exhaustive"
  - "independent adversarial verifier report attached verbatim"
  - "nothing pushed, posted or filed on GitHub; our repo not pushed"
merge: b3e9d71
result: "two deviations proven (min-damage synergy gate FUN_00644d50; MulDiv shares), 116,000 cases 4,549 -> 0; Static Field three rules found by reading (candidate, not in the PR); branch fix/skill-damage c6be9247 ready pending the user's go; verified offline only (no game, no d2d slot)"
```

## 1. Work

- [x] 1.1 read the prior proofs (item1, PR #135 proof, cast trace, gap-casts) and upstream's rules
- [x] 1.2 trace FUN_0064b860 -> FUN_00644d50 / 00644e40 / 00644f20 / 00644c90 / 00646460 / 00646c00 / 006442a0, FUN_0064ac60 (the missile's stats), FUN_005c96a0 (Static Field)
- [x] 1.3 oracle `tools/emu/skill_damage.py` + `test_game --skill-damage`; differential 116,000 cases: before 4,549, gate only 344, final 0
- [x] 1.4 branch fix/skill-damage (3 commits, author the user, `Agent: Claude Code`), pr_guard 0/0, presubmit --quick 0 FAIL 4 WARN answered, test_combat ok, test_game rc 0
- [x] 1.5 independent adversarial verifier (Fable, read-only), report in the proof doc
- [x] 1.6 docs: trace, proof, draft, evidence pack; card; OWNERS; changecheck / leakcheck / openspec validate
