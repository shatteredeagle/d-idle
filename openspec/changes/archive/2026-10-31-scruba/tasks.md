# Tasks

```yaml
change: scruba
status: archived
agent: scruba
branch: agent/scruba
worktree: ~/code/d2lod-wt/scruba
owns:
  - openspec/changes/scruba/**
  - AGENTS.md
  - CLAUDE.md
  - local.env.example
  - tools/env.sh
  - tools/ops.sh
  - tools/ops-cutover.sh
  - tools/live.sh
  - tools/game.sh
  - tools/jobs.sh
  - tools/helper.sh
  - tools/shot.sh
  - tools/mute-d2.sh
  - tools/mute-d2-watch.sh
  - tools/capacity.sh
  - tools/devhealth.sh
  - tools/sentinel.sh
  - tools/watch.sh
  - tools/d2-prefix.sh
  - tools/grill.sh
  - tools/grill-guard.sh
  - tools/container-game-copy.sh
  - tools/agentstatus/**
  - docs/process/public-split.md
  - docs/process/public-manifest.json
exclusive: []
gates:
  - "go test -p 1 ./cmd/leakcheck passes; changecheck 0 warnings; openspec validate --all --strict"
merge: archived-by-cutover
result: |-
  public scrub package A: 105 private-class files moved to the private ops repo <owner>/d2lod-ops (with history), stubs left, AGENTS.md split public/private, tools/env.sh + local.env.example + tools/ops.sh, CLAUDE.md = @AGENTS.md; cutover executed 2026-10-31 (master changes since 57dd902 to moved paths replayed into ops). Not live-run (no game needed).
```

- [x] 1.1 Extract the private-class files with history into ~/code/d2lod-ops (filter-repo on a scratch clone)
- [x] 1.2 git rm in this branch, stubs, link fixes
- [x] 1.3 AGENTS split, CLAUDE.md, tools/ops.sh, tools/env.sh, local.env.example
- [x] 1.4 Manifest regenerated, public-split.md package A section, tools/ops-cutover.sh
