# Tasks

```yaml
change: d2dspeed
status: archived
merge: 2102dce
agent: d2dspeed
branch: agent/d2dspeed
worktree: ~/code/d2lod-wt/d2dspeed
owns:
  - internal/d2dsrc/**
  - internal/d2dctl/**
  - internal/d2denv/**
  - cmd/d2dprobe/**
  - docs/d2d-pace.md
  - docs/d2d-fork.md
exclusive:
  - d2d-lab-lock
  - fork-branch:lab4
  - d2d-lab-binary-lab4
gates:
  - "bot attached under lockstep >= 20 virtual seconds per wall second over 10 wall minutes"
result: "bot attached under lockstep 22-33x real time (27.4x over 10 wall min seed 7) with fork lab5 (no render in lockstep, immediate verbs) + Prefetch + virtual-time sleeps; gate 20x met; determinism of lab5 not re-run (U: D2DLOOP checks it); remaining ceiling: bot CPU/GC"
```

- [x] profile (trace, baseline run)
- [x] pipelined snapshot read
- [x] fork lab4 patch
- [x] inventory flag, d2d.out rotation, socket diagnosis
- [x] 20x gate met: 27.4x over 10 wall minutes, boosted clvl 25 character
