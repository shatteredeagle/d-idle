# Tasks

```yaml
change: kitstall
status: archived
agent: kitstall
branch: agent/kitstall
worktree: ../d2lod-wt/kitstall
skip_specs: true
owns:
  - docs/kitstall.md
  - openspec/changes/kitstall/**
exclusive: []
gates:
  - "go test -p 1 ./internal/nav ./internal/gear ./internal/town; changecheck -cli 0 warnings; leakcheck -mode fail 0"
result: "go tests nav gear town pass; not verified live"
live: "not verified"
merge: 49cead9
```

- [x] timeline from the bot log
- [x] dead-gate memory in nav with test
- [x] gear and town ammo fixes with tests
- [x] docs/kitstall.md with the open U items and the read-only probe
