# Tasks

```yaml
change: mapitem
status: archived
merge: on main (see result)
agent: mapitem
branch: agent/mapitem
worktree: ~/code/d2lod-wt/mapitem
owns:
  - docs/stream/map-contiguity.md
  - cmd/d2dview/stitch.go
  - cmd/d2dview/stitch_test.go
  - cmd/d2dview/feed.go
  - cmd/d2dview/pub.go
  - cmd/d2dview/main.go
  - cmd/d2dview/view_test.go
  - cmd/d2dview/web/stream.html
  - cmd/d2dview/web/index.html
exclusive: []
gates:
  - "go test -p 1 ./cmd/d2dview passes including TestStitchTransitionBloodMoorColdPlains, TestStitchFieldAndTown, TestStitchFieldToCaveIsNotStitched"
  - "go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes"
result: "closed: merged; the live check is a nicety (triage 2026-11-04)"
live: "not verified live: see the last section of docs/stream/map-contiguity.md"
```

- [x] game rule found and written (docs/stream/map-contiguity.md)
- [x] composite map in the act frame, swap-in pages, trail rule
- [x] scenario tests (stitched crossing, field to cave, field and town)
- [x] spec delta (dashboard-pub)
- [ ] live check on a hidden slot
