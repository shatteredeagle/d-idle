# Tasks

```yaml
change: softguard
status: archived
agent: softguard
branch: agent/softguard
worktree: ~/code/d2lod-wt/softguard
owns:
  - docs/process/dispatcher-brief.md
  - docs/process/worker-capacity.md
  - openspec/changes/softguard/**
  - tools/capacity.sh
  - tools/capacity.stub
exclusive: []
gates:
  - "idle+nice and iowait are judged on a 5 minute average, never one top sample"
  - "non-big work and offline work are never gated on the user"
merge: 5d2a999
result: "soft capacity rule: predicted idle+nice >= 25% and iowait <= 40% on a 5-min average, per-item core costs, big-run 5-minute warn-then-go; tools/capacity.sh; the worker host run OK; deltas applied at archive"
```

## 1. Work

- [x] 1.1 tools/capacity.sh (POSIX sh, read-only) with stub mode
- [x] 1.2 spec delta for lab-rules and soak-gates
- [x] 1.3 dispatcher-brief health guard rewritten
- [x] 1.4 docs/process/worker-capacity.md
- [x] 1.5 validate: openspec strict, changecheck, sh -n (see result)

## Follow-ups for the coordinator (not owned here)

- AGENTS.md "the worker host is shared and loaded" and docs/review-strategy.md: point at docs/process/worker-capacity.md instead of the 2 Game.exe cap
- docs/soak24.md section 2 and the soak24.sh `D2_SOAK_MAXLOAD` guard: read `tools/capacity.sh --add 1.0` (exit 2 refuses)
- dispatch agent briefs (`.claude/agents/*`): replace the cap sentence by the pointer
- record `warned: <time>` in the header of the card of any big run (a free-form header line)
