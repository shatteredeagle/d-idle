# Tasks

```yaml
change: doctrim
status: in-progress
agent: doctrim
branch: agent/doctrim
worktree: ~/code/d2lod-wt/doctrim
owns:
  - docs/README.md
  - docs/BACKLOG.md
  - docs/research/**
  - docs/archive/**
  - docs/merc-farm.md
  - docs/sandbox.md
  - openspec/changes/doctrim/**
exclusive: []
gates:
  - "changecheck -cli 0 warnings; leakcheck fail 0; openspec validate --all --strict; broken relative md links after <= before"
  - "nothing deleted: every moved doc leaves a stub with a status banner"
merge:
result: 'docs trim: 10 docs archived with stubs, research and docs indexes, BACKLOG 61 -> 32 lines (old text archived); offline docs only, not verified live'
```

## 1. Work

- [x] 1.1 measure docs/ and write the numbers in the report
- [x] 1.2 docs/research/README.md topical index, status banners
- [x] 1.3 posted-PR draft docs and executed review drafts to docs/archive/ with stubs
- [x] 1.4 finished/superseded docs to docs/archive/ with stubs
- [x] 1.5 BACKLOG.md rewritten short; old text in docs/archive/BACKLOG-2026-10-30.md
- [x] 1.6 docs/README.md
- [x] 1.7 verify: changecheck, leakcheck, openspec validate, link check
