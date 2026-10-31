# Tasks

```yaml
change: prbuild
status: archived
agent: prbuild
branch: agent/prbuild
worktree: ~/code/d2lod-wt/prbuild
owns:
  - docs/research/pr-missile-damage-fine-draft.md
exclusive: []
gates:
  - "pr_guard 0 FAIL 0 WARN, test_combat and test_game pass on the worker host"
merge: ebe78c8
result: "PR-ready branch fix/missile-damage-fine (clone ~/d2d-lab/pr/gaps 91a7cc92): element spells keep 256ths damage fraction; pr_guard 0/0; test_combat mean 3.0; NOT pushed: user chose measure first (prmeasure). Merged f081ec1-series (ebe78c8)"
```

## 1. Work

- [x] change, tests, pr_guard in the clone
- [ ] before/after casts per kill (not run: lab lock busy, command in the draft)
- [x] PR body draft
