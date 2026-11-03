# Tasks

```yaml
change: verifyplayerroll
status: archived
agent: verifyplayerroll
branch: agent/verifyplayerroll
worktree: ~/code/d2lod-wt/verifyplayerroll
skip_specs: true
owns:
  - openspec/changes/verifyplayerroll/**
  - docs/research/pr-playerroll-verifier.md
  - docs/research/pr-playerroll-verifier/**
exclusive: []
gates:
  - "changecheck 0 warnings for this card; leakcheck -mode fail 0"
result: "docs and tools only; merged by the triage queue"
merge: c833e45
```

## 1. Work

- [x] 1.1 Diff and clean merge against current upstream main
- [x] 1.2 Own counts: physical roll and floor, Sorceress and Necromancer element rolls, lo == hi draws
- [x] 1.3 Caller audit
- [x] 1.4 Det scenarios base vs head (15, one hidden slot)
- [x] 1.5 Warnings-as-errors build (g++ and clang++), MSVC hazards by reading
- [x] 1.6 Suites and ctest base vs head
- [x] 1.7 Draft claims vs counts; verdict
