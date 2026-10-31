# Tasks

```yaml
change: navsoak
status: archived
agent: navsoak
branch: agent/navsoak
worktree: ~/code/d2lod-wt/navsoak
owns:
  - docs/probes/navsoak/**
exclusive:
  - soak-slot:0
gates:
  - "soak started from origin/main 279a163 (contains 85b429d), healthy for 20 minutes, registry entry navsoak present"
  - "go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes"
merge: 8f4cf7b
result: docs only, merged; see docs/probes/navsoak/README.md (verified live by the soak itself)
```

## 1. Work

- [x] 1.1 Prove the build has the NAVSTALL fix, build d2bot.navsoak, ship to the worker host
- [x] 1.2 Start the soak under d2dslot + agentrun with the restart supervisor
- [x] 1.3 Verify 20 minutes of health; write docs/probes/navsoak/README.md
