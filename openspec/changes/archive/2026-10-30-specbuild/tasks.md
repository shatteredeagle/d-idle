# Tasks

```yaml
change: specbuild
status: archived
agent: specbuild
branch: agent/specbuild
worktree: ~/code/d2lod-wt/specbuild
owns:
  - openspec/**
  - cmd/changecheck/**
  - docs/OWNERS
  - docs/adr/**
  - docs/process/**
  - docs/decisions.md
  - docs/INDEX.md
  - docs/BACKLOG.md
  - docs/boss.md
  - docs/plan/plan-v2.md
  - docs/plan/STATUS.md
  - PLAN.md
  - AGENTS.md
  - .claude/agents/spec-sme.md
  - docs/archive/**
  - docs/deathautopsy.md
  - docs/liveruns-briefs.md
  - docs/restloop.md
  - docs/prior-art/**
  - docs/container.md
  - docs/nas-container.md
  - docs/quickstart.md
  - docs/troubleshooting.md
  - docs/soak-hosts.md
  - docs/sandbox.md
  - docs/act3.md
  - docs/act4.md
  - docs/act5.md
  - docs/nightmare-campaign.md
  - docs/review-strategy.md
  - docs/soak-analysis.md
  - docs/merc-farm.md
exclusive: []
gates:
  - "go test -race ./cmd/changecheck passes; gofmt and go vet clean"
  - "go vet and go test -short on the whole repo via labrun worker show no FAIL"
  - "changecheck on the repo prints its findings and the proposal Why of every card is 50 to 1000 characters"
merge: 9a4c06f
result: "change-card layer, 7 specs, changecheck, ADRs, doc fixes, spec-sme"
```

## 1. Work

- [x] 1.1 Layout and README
- [x] 1.2 Seven specs
- [x] 1.3 cmd/changecheck with tests
- [ ] 1.4 Cards and archive entries
- [ ] 1.5 Doc fixes, banners, ADRs, AGENTS.md
- [ ] 1.6 spec-sme agent and trial-notes
