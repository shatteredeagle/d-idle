# Tasks

```yaml
change: equipreq
status: archived
agent: equipreq
branch: agent/equipreq
worktree: ~/code/d2lod-wt/equipreq
owns:
  - internal/town/rebuy.go
  - internal/town/beltfill.go
  - internal/town/require.go
  - internal/town/equipreq_test.go
  - internal/data/affixreq.go
  - internal/data/affixreq_test.go
  - docs/gear-gate.md
exclusive: []
gates:
  - "gofmt, go vet, go test -race of internal/town and internal/data pass; whole repo via labrun worker has no FAIL"
  - "no live game work (Game.exe path frozen)"
merge: 090af96
result: "RequirementSource: affix level reqs on d2d via item names, Game.exe uses base+margin 6 (U); refused clicks blacklisted; belt only into empty slots; unverified live"
```

## 1. Work

- [x] 1.1 RequirementSource seam with affix levelreq lookup and the magic margin
- [x] 1.2 equip skip logging, blacklist by id/clvl/str/dex, right-click fallback off
- [x] 1.3 belt room by occupied cells, empty-cell drag, count verified
- [ ] 1.4 live check of the d2d name path (needs d2denv ItemName)
