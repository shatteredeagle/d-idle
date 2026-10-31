# Tasks

```yaml
change: m2-latebase
status: deferred
agent: m2-latebase
branch: agent/m2-latebase
worktree: ~/code/d2lod-wt/m2-latebase
owns:
  - tools/d2dsoak/baselines-late.json
  - docs/d2d-baselines-late.md
exclusive:
  - soak-slot:3
gates:
  - "tools/d2dsoak/baselines-late.json has at least 5 runs from a cheat-levelled clvl 12 throwaway with the M1 kit, each with xp_per_hour, kills_per_hour, a source line and the ledger entry ids; only xp_per_hour and kills_per_hour are gated (rel 0.2), deaths and gold are not"
  - "every cheat used is in the cheat ledger with a reason (cmd/d2cheat ledger listing pasted into docs/d2d-baselines-late.md); no cheat inside a measured window"
  - "go test ./cmd/d2dsoak passes (the baseline file loads); go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes; tools/agentreap.sh m2-latebase leaves nothing"
merge:
result:
```

## 1. Work

- [ ] 1.1 Read docs/cheats.md and the fresh-character policy; name the throwaway character and the ledger actor; write the protocol (XP set to the clvl 12 table threshold, points spent by the leveler, M1 kit) in docs/d2d-baselines-late.md
- [ ] 1.2 Run the seeds on hidden d2d with `tools/d2dsoak` through `tools/agentrun.sh`; extract xp_per_hour and kills_per_hour per run
- [ ] 1.3 Write baselines-late.json with source lines and ledger ids; paste the ledger listing; changecheck, leakcheck, validate; agentreap
