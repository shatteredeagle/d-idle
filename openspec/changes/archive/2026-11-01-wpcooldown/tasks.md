# Tasks

```yaml
change: wpcooldown
status: archived
merge: ddf6213d
agent: wpcooldown
branch: agent/wpcooldown
worktree: ~/code/d2lod-wt/wpcooldown
skip_specs: true
owns:
  - internal/nav/waypoint.go
  - internal/nav/wpclick_test.go
  - internal/env/tuning_nav.go
  - docs/wpcooldown.md
  - docs/waypoints.md
  - docs/fixwp.md
  - openspec/changes/wpcooldown/**
exclusive: []
gates:
  - "go test -p 1 ./internal/nav ./internal/env; changecheck -cli 0 warnings; leakcheck -mode fail 0"
result: "Cooldown wait removed by default (knob kept, default 0, deprecated); fake-clock tests pass."
live: "not verified (decomp and upstream differential only); drill recipe in docs/wpcooldown.md"
```

- [x] default 0 and no-wait path
- [x] tests
- [x] docs and banners
