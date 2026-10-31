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
result:
```

## 1. Work

- [ ] 1.1 measure docs/ and write the numbers in the report
- [ ] 1.2 docs/research/README.md topical index, status banners
- [ ] 1.3 posted-PR draft docs and executed review drafts to docs/archive/ with stubs
- [ ] 1.4 finished/superseded docs to docs/archive/ with stubs
- [ ] 1.5 BACKLOG.md rewritten short; old text in docs/archive/BACKLOG-2026-10-30.md
- [ ] 1.6 docs/README.md
- [ ] 1.7 verify: changecheck, leakcheck, openspec validate, link check
