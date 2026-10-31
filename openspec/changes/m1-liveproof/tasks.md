# Tasks

```yaml
change: m1-liveproof
status: deferred
agent: m1-liveproof
branch: agent/m1-liveproof
worktree: ~/code/d2lod-wt/m1-liveproof
owns:
  - docs/basics-audit.md
  - docs/basics-live.md
  - docs/probes/basics/**
exclusive:
  - soak-slot:3
gates:
  - "docs/basics-audit.md coverage table has 0 G rows and every S1 or S2 row is C with a test name in the offline-test column and a live proof line (checked by the coordinator by eye; `grep -c '| C ' docs/basics-audit.md` is at least 15)"
  - "starting gold and unidentified sell prices recorded V in docs/basics-live.md with cheat ledger ids"
  - "tools/agentreap.sh m1-liveproof leaves nothing; changecheck -cli 0 warnings; leakcheck 0"
merge:
result:
```

## 1. Work

- [ ] 1.1 Measure the two unknowns first (new throwaway on 2 classes, `state` after the first town spawn; one unidentified and one identified ring sold)
- [ ] 1.2 Run live checks 1 to 10 of docs/basics-audit.md (Proposed live checks) on the merged tree; record each
- [ ] 1.3 Update the coverage table and the gaps list; changecheck, leakcheck; reap
