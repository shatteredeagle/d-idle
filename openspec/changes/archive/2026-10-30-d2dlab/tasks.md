# Tasks

```yaml
change: d2dlab
status: archived
agent: d2dlab
branch: agent/d2dlab
worktree: ~/code/d2lod-wt/d2dlab
owns:
  - docs/d2d-lab.md
exclusive:
  - d2d-lab-binary
  - fork-branch:lab
  - fork-branch:d2dtown-vendor
  - fork-branch:p3-layout-verb
  - fork-branch:lab2
  - d2d-lab2-binary
  - d2d-lab-lock
gates:
  - "bin/d2d.lab is read-only with sha256 and provenance next to it (bin/d2d.lab.txt)"
  - "verify-lab.sh passes for debug layout, debug nearby, dump, dump vendor, clock, step"
  - "every fork branch is pushed to the forge <owner>/d2d-fork the day it changes"
merge: 902e5fa
result: "lab (P1+P2+P3+vendor on e36922e) and lab2 (rebased on 3306e60 plus upstream-style price and cast patches) built, verified, pushed to the forge; start.sh defaults to d2d.lab2"
```

## 1. Work

- [x] 1.1 Build the lab binary from the lab branch (Ninja, Release, WERROR off)
- [x] 1.2 Document branches and rebuild in docs/d2d-lab.md
- [x] 1.3 Push all fork branches to the forge <owner>/d2d-fork (AGENTS.md rule)
- [x] 1.4 Final verify-lab.sh run and report
