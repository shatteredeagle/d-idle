# Tasks

```yaml
change: boardcards
status: archived
agent: boardcards
branch: agent/boardcards
worktree: ~/code/d2lod-wt/boardcards
owns:
  - tools/agentstatus/**
  - cmd/changecheck/**
  - docs/process/board-cards.md
exclusive: []
gates:
  - "go vet ./cmd/changecheck && go test -race ./cmd/changecheck passes"
  - "tools/agentstatus/tests/test_changes.sh passes (page data renders from fixture, HTML structure valid)"
  - "changecheck reports 0 warnings"
merge: 95edbdf
result: "/changes page, changecheck json mode, publish-cards.sh"
```

## 1. Data

- [x] 1.1 `changecheck json` prints cards, specs, gates, merges, collisions and warnings (go test)
- [ ] 1.2 board server stores and serves changes.json (PUT/GET /api/changes) and tools/agentstatus/publish-cards.sh uploads it

## 2. Page

- [ ] 2.1 changes.html at /changes, linked from the nav of every board page and from /hub (fixture test)

## 3. Docs

- [ ] 3.1 docs/process/board-cards.md has the deploy steps for the NAS container
