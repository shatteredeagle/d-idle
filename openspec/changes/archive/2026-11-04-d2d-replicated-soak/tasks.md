# Tasks

```yaml
change: d2d-replicated-soak
status: archived
agent: d2d-replicated-soak
branch: agent/d2d-replicated-soak
worktree: ~/code/d2lod-wt/d2d-replicated-soak
owns:
  - docs/d2d-replicated-soak.md
exclusive:
  - soak-slot:2
gates:
  - "d2dsoak verdict printed for all four gated metrics, 5 or more seeds"
  - "gaps listed as explained or as fidelity bugs"
merge: none (dropped)
result: "dropped: stale (evidence: replaced by the seed set and the statistical A:B, ADR 0015 items 6 and 8; card triage 2026-11-04)"
```

## 1. Work

- [ ] 1.1 run 8 seeds by 2 virtual hours on the lab fork
- [ ] 1.2 compare the four gated metrics with the recorded baselines
- [ ] 1.3 list gaps with attribution per the new rule (engine vs bot)
