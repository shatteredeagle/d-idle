# Tasks

```yaml
change: m2-earlybase
status: archived
agent: m2-earlybase
branch: agent/m2early
worktree: ~/code/d2lod-wt/m2early
owns:
  - tools/d2dsoak/baselines-early.json
  - docs/d2d-baselines-early.md
  - docs/probes/m2early/**
exclusive: []
gates:
  - "tools/d2dsoak/baselines-early.json has at least 2 runs, each with xp_per_hour, kills_per_hour, deaths_per_hour, deaths_per_1000_kills and a source line; rules gate kills_per_hour (rel 0.2) and deaths_per_1000_kills (rel 0.2, abs 1.0)"
  - "go test ./cmd/d2dsoak passes (the baseline file loads) and `d2dsoak compare -base tools/d2dsoak/baselines-early.json` on the soakfull runs in docs/probes/soakfull prints a verdict for all four metrics"
  - "go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes"
merge: 8bf3829
result: "Fresha V and Soaky S entries with the four metrics; 8 seeds x 3 h d2d set measured (4767 +- 230 XP/h, 192 +- 7.5 kills/h, 3 deaths, 0 exclusions), compare FAIL on XP and kills (-48 %, -52 %); verified live on hidden d2d for the d2d side, Game.exe side read-only from logs. Gate 2 note: docs/probes/soakfull holds only summaries (no bot logs), so compare ran on the new m2early logs, docs/probes/m2early."
```

## 1. Work

- [x] 1.1 List the original-game logs used by docs/research/per-kill-time-split.md (Inputs) and run `d2dsoak extract` on the clvl 1 to 7 windows; record the commands in the doc
- [x] 1.2 Write the run entries and the gated rules; keep the existing Soaky entry, add Fresha; note the kill undercount
- [x] 1.3 Re-run compare on docs/probes/soakfull; paste the verdict into the doc; changecheck, leakcheck, validate
