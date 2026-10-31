# Tasks

```yaml
change: m3-act1run
status: deferred
agent: m3-act1run
branch: agent/m3-act1run
worktree: ~/code/d2lod-wt/m3-act1run
owns:
  - docs/act1-run.md
  - docs/probes/act1run/**
exclusive:
  - soak-slot:3
gates:
  - "3 of 3 fresh Sorceress seeds, no cheats: Andariel dead and Act 2 entered (area 40 or the Act 2 quest word) before the time cap of act1-campaign `Unattended Act 1 clear`; d2dsoak reports no death-rule FAIL code (death_streak, death_recovery_stuck, death_rate); 0 panics; 0 manual interventions"
  - "merc alive at least 70 percent of field time after the claim on every seed (minute lines)"
  - "tools/agentreap.sh m3-act1run leaves nothing; changecheck -cli 0 warnings; leakcheck 0; validate --strict passes"
merge:
result:
```

## 1. Work

- [ ] 1.1 Compute the cap from the M2 result (1.3 x 176,132 / the M2 XP/h) and write it in the delta with its source
- [ ] 1.2 3 seeds, one slot, each wall-limited; collect reports
- [ ] 1.3 docs/act1-run.md; a finding per failure for the next card; changecheck, leakcheck, validate; reap
