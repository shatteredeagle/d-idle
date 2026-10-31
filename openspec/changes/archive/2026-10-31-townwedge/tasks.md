# Tasks

```yaml
change: townwedge
status: archived
agent: townwedge
branch: agent/townwedge
worktree: ~/code/d2lod-wt/townwedge
owns:
  - internal/nav/route.go
  - internal/nav/exit.go
  - internal/nav/exit_test.go
  - internal/nav/nav.go
  - internal/nav/townwedge_test.go
  - internal/env/tuning_nav.go
  - docs/townwedge.md
  - openspec/changes/townwedge/**
exclusive:
  - d2d-lab-binary
gates:
  - "go test -short -count=1 ./... and go test -race ./internal/nav ./internal/env pass on the worker; changecheck -cli 0 warnings; leakcheck -mode fail 0; openspec validate --all --strict"
  - "live verify: one hidden slot re-runs the navsoak2 recipe (seed 5): at least 60 wall minutes of area entries, no 'no route from level 1' storm"
merge: 5be7cc7
result: "verified live on hidden d2d (seed 5, Navy3, 74 wall min): the navsoak2 wedge state recurred at the same cell and was left by the pocket unstick; 0 bad edges, 0 no-route lines, 1582 area entries, clvl 11, 0 deaths; offline tests and whole-repo vet, short tests and race clean on the worker host"
```

## 1. Work

- [x] 1.1 root cause from the navsoak and navsoak2 logs (docs/townwedge.md)
- [x] 1.2 bad edge and gate expiry, re-entry reset, shown-edge rule, pocket unstick, knob nav.badEdgeTTLSec
- [x] 1.3 offline tests, spec delta, knobs.md
- [x] 1.4 live verify on one hidden slot, numbers in docs/townwedge.md
