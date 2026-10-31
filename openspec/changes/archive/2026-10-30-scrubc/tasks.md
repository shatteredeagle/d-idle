# Tasks

```yaml
change: scrubc
status: archived
agent: scrubc
branch: agent/scrubc
worktree: ~/code/d2lod-wt/scrubc
owns:
  - openspec/changes/scrubc/**
  - openspec/changes/archive/**
  - openspec/specs/**
  - openspec/README.md
  - CHANGELOG.md
  - GLOSSARY.md
  - PLAN.md
  - tools/ci/README.md
  - tools/soak/CHECKLIST.md
  - deploy/container/**
  - deploy/d2d-container/**
  - deploy/nas/**
  - docs/adr/**
  - docs/agents/**
  - docs/archive/**
  - docs/classes/**
  - docs/d2d-container/**
  - docs/grills/**
  - docs/milestones/**
  - docs/net/**
  - docs/plan/**
  - docs/process/**
  - docs/progression/**
  - docs/research/**
  - docs/sme/**
  - docs/stream/**
  - docs/d2-1.14d-memory.md
  - docs/d2d-backend.md
  - docs/d2d-bot.md
  - docs/d2d-combat.md
  - docs/d2d-container-bot.md
  - docs/d2d-determinism.md
  - docs/d2d-economy.md
  - docs/d2d-exit.md
  - docs/d2d-fidelity.md
  - docs/d2d-fork.md
  - docs/d2d-lab.md
  - docs/d2d-lab6.md
  - docs/d2d-lan-oracle.md
  - docs/d2d-loop.md
  - docs/d2d-pace.md
  - docs/d2d-probes.md
  - docs/d2d-replay.md
  - docs/d2d-run-ab.md
  - docs/d2d-soak-full-fix.md
  - docs/d2d-soak.md
  - docs/d2d-spike.md
  - docs/d2d-town.md
  - docs/d2d-view.md
  - docs/d2d-walk.md
  - docs/d2dctl.md
  - docs/data.md
  - docs/decisions.md
  - docs/difficulty.md
  - docs/sandbox.md
  - docs/saves.md
  - docs/sentinel.md
  - docs/setup.md
  - docs/soak-analysis.md
  - docs/soak.md
  - docs/soak24.md
  - docs/stallfatal.md
  - docs/streamctl-d2d.md
  - docs/streamctl.md
  - docs/summon.md
  - docs/OWNERS
  - docs/B*
  - docs/I*
  - docs/a*
  - docs/b*
  - docs/c*
  - docs/e*
  - docs/f*
  - docs/g*
  - docs/i*
  - docs/k*
  - docs/l*
  - docs/m*
  - docs/n*
  - docs/o*
  - docs/p*
  - docs/q*
  - docs/r*
  - docs/t*
  - docs/v*
  - docs/w*
exclusive: []
gates:
  - "go run ./cmd/leakcheck -mode warn reports 0 hits"
  - "go run ./cmd/changecheck -cli has 0 warnings"
  - "tools/openspec.sh validate --all --strict passes"
  - "go vet and go test -short pass on the worker host"
merge: 17cf55e
result: 'public scrub package C: docs scrub: leakcheck 1057 hits in 282 files -> 0 (machine 926, person 62, domain 41, homepath 9, ip 8, email 6, secret 5 (all false positives, allow-listed with reasons)); ~280 files, 10 commits by directory class; deploy/nas -> deploy/nas; glossary machine names -> Worker host/Head box; residue: archived card ids two archived cards keep machine-name ids (allow-listed; rename at the fresh-history cut), docs/nas-container.md stub (real doc in the ops repo names the NAS)'
```

## 1. Work

- [x] 1.1 scripted replacement by role word, reviewed by hand
- [x] 1.2 renames (deploy/nas, archived card ids) and link fixes
- [x] 1.3 allow-list reasons, manifest regenerated
- [x] 1.4 verification (leakcheck 0, changecheck 0, openspec strict, vet+test on the worker host)
