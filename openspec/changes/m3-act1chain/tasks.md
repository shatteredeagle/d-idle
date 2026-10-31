# Tasks

```yaml
change: m3-act1chain
status: deferred
agent: m3-act1chain
branch: agent/m3-act1chain
worktree: ~/code/d2lod-wt/m3-act1chain
owns:
  - internal/task/act1.go
  - internal/task/campaign.go
  - internal/task/act1_d2d_test.go
  - docs/act1-d2d.md
  - docs/probes/act1chain/**
exclusive:
  - soak-slot:3
gates:
  - "1 seed from a cheat-levelled clvl 12 throwaway (ledgered) on hidden d2d: Den, Blood Raven, the Rogue claim and the Andariel approach (Catacombs 4 entered) reached unattended, every quest's done state read from the save or memory and matching docs/progression/quests.md (raw in docs/probes/act1chain)"
  - "go test -p 1 ./internal/task passes including TestAct1OrderOnD2D (Cain optional, Countess skipped, Andariel required for Warriv)"
  - "go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes; tools/agentreap.sh m3-act1chain leaves nothing"
merge:
result:
```

## 1. Work

- [ ] 1.1 Read docs/progression/quests.md, docs/progression/verification.md, internal/task/act1.go; the order test first
- [ ] 1.2 Quest evidence reads on d2d (save words after a save point, memory if wired); no T3-only skip
- [ ] 1.3 The run; docs/act1-d2d.md; changecheck, leakcheck, validate; reap
