# Tasks

```yaml
change: smallfixes
status: in-progress
agent: smallfixes
branch: agent/smallfixes
worktree: ~/code/d2lod-wt/smallfixes
owns:
  - tools/d2dfarm.sh
  - tools/d2dfarm-test.sh
  - tools/env.sh
  - tools/env-test.sh
  - tools/d2dview-demo.sh
  - tools/soak/soak.sh
  - tools/hooks/test-pre-push.sh
  - internal/bot/task.go
  - internal/bot/gate_plan_test.go
  - docs/nas-container.md
  - docs/BACKLOG.md
  - openspec/changes/smallfixes/**
exclusive: []
gates:
  - "bash tools/d2dfarm-test.sh prints ALL OK; sh tools/env-test.sh prints ALL OK"
  - "go test ./internal/bot passes; the fresh-game reset change needs a G1/G2 check"
  - "go run ./cmd/changecheck -cli 0 warnings; go run ./cmd/leakcheck -mode fail 0"
merge:
result: "d2dfarm-test and env-test ALL OK, bot tests pass; whole-repo vet/test on the worker; not verified live; fresh-game reset needs a G1/G2 check"
```

## 1. Tasks

- [x] 1.1 d2dfarm-test: cause was capacity.sh/capacity.stub moving to the ops repo (probe rc 127, rest cascaded); probe resolves the real script, test uses a fake
- [x] 1.2 freshGameToLevel also resets sweepKills, needBuff, stepArea (soakfull hotfix) with its test
- [x] 1.3 d2_role_env in env.sh; d2dview-demo and soak start forward it; ops live.sh forwards it (ops repo)
- [x] 1.4 nas-container heading and BACKLOG lines
- [x] 1.5 test-pre-push.sh executable bit
