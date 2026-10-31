# Tasks

```yaml
change: prochygiene
status: archived
agent: prochygiene
branch: agent/prochygiene
worktree: ~/code/d2lod-wt/prochygiene
owns:
  - openspec/changes/prochygiene/**
  - tools/agentrun.sh
  - tools/agentreap.sh
  - tools/d2dslot.sh
  - tools/d2dview-demo.sh
  - tools/d2ddet/pair.sh
  - tools/d2ddet/build.sh
  - tools/d2dloop/**
  - tools/test/**
  - docs/process/dispatcher-brief.md
  - AGENTS.md
exclusive: []
gates:
  - "bash tools/test/agentrun_test.sh passes on the worker host"
  - "no pkill/killall in the files this card owns"
merge: f48d339
result: "shell tests pass on the worker host (bash tools/test/agentrun_test.sh); no live game used; docs/process/worker-compute.md ship line not edited (not owned)"
```

## 1. Work

- [x] 1.1 tools/agentrun.sh and tools/agentreap.sh
- [x] 1.2 d2dslot, d2dview-demo, pair.sh, d2dloop register through agentrun
- [x] 1.3 shell tests
- [x] 1.4 AGENTS.md cleanup rule, dispatcher janitor duty
