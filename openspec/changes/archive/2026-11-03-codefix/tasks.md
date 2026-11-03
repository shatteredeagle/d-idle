# Tasks

```yaml
change: codefix
status: archived
agent: codefix
branch: agent/codefix
worktree: ~/code/d2lod-wt/codefix
skip_specs: true
owns:
  - docs/research/codefix.md
  - openspec/changes/codefix/**
exclusive: []
gates:
  - "go test -p 1 on the touched packages; whole repo vet+test on the worker host; changecheck -cli 0 new warnings; leakcheck -mode fail 0; tools/ci/check.sh"
result: "whole repo vet and test pass on the worker host"
credit:
  author: owner
  agents: [Claude Code]
merge: 9c9b547
```

- [x] 1 F1 walk-home list split from NearTown (ChainHome, no-healing emergency only)
- [x] 2 F2 trip end records the policy's gold (stash included)
- [x] 3 F3 mana reserve only for the missing part of the kit
- [x] 4 F4 chain walk limited to the route fields
- [x] 5 F5 recovery potion floor lifted after a withdrawal in the same round
- [x] 6 F8 cut-cast memory kept on a dead context; F6, F7, F9, F10 open (docs/research/codefix.md)
- [ ] 7 live check (Gate 1 re-run, section 4 of the doc)
