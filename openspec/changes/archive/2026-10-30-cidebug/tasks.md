# Tasks

```yaml
change: cidebug
status: archived
agent: cidebug
branch: agent/cidebug
owns:
  - docs/ci.md
  - .forgejo/workflows/ci.yml
  - tools/ci/**
exclusive: []
gates:
  - "checks and test pass on the NAS runner"
merge: d9e1e10
result: "the NAS CI test failure was d2fleet admission refusing on host load (load 5.8 on 4 cores): container-checklist.sh runs with D2FLEET_MAX_LOAD_FACTOR=-1; verified on the NAS run via workflow_dispatch API"
```

- [x] read the failing task log on Tower (the forge actions_log), reproduce in the runner container
- [x] fix: tools/ci/scripts.sh exports D2FLEET_MAX_LOAD_FACTOR=-1
- [x] verified: workflow_dispatch on ci-debug, tasks 1187 (checks) and 1188 (test) succeeded on the NAS; branch deleted
