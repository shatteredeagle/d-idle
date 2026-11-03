# Tasks

```yaml
change: forknext
status: archived
merge: 25274844
agent: forknext
branch: agent/forknext
worktree: ~/code/d2lod-wt/forknext
skip_specs: true
owns:
  - docs/forknext.md
  - openspec/changes/forknext/**
exclusive:
  - fork-branch:lab-nexthit
  - d2d-lab-binary
gates:
  - "changecheck status -write then -cli 0 warnings; leakcheck -mode fail 0; tools/ci/check.sh; openspec validate --all --strict"
result: "fork branch lab-nexthit pushed (989517079642); differential on the fork build 0 of 100,198 differ (before 84,779); a Frost Nova and a Nova cast at an adjacent monster through debug cast hit once (live, one hidden slot); docs/forknext.md"
live: "verified live on one hidden slot with a --no-save throwaway Sorceress (no game.exe)"
```

## 1. Work

- [x] 1.1 fork branch lab-nexthit from lab-debug-skills, claim and test mode cherry-picked clean
- [x] 1.2 vet: build, differential before and after on the fork build, live one-cast smoke
- [x] 1.3 push lab-nexthit to the private remote
- [x] 1.4 docs/forknext.md, card, OWNERS
