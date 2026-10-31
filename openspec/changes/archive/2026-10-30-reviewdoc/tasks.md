# Tasks

```yaml
change: reviewdoc
status: archived
agent: reviewdoc
branch: agent/reviewdoc
worktree: ~/code/d2lod-wt/reviewdoc
owns:
  - openspec/changes/reviewdoc/**
  - docs/research/code-review-2026-10-30.md
  - docs/research/proposed-glossary-terms.md
  - docs/INDEX.md
  - docs/plan/STATUS.md
  - docs/BACKLOG.md
  - docs/soak24.md
  - docs/soak.md
  - docs/fleet.md
  - docs/setup.md
  - docs/launch.md
  - docs/d2d-backend.md
  - docs/d2d-spike.md
  - docs/plan/plan-v2.md
  - docs/liveruns.md
  - docs/liveruns-checks.md
  - docs/d2d-casts.md
  - docs/d2d-lab.md
  - docs/d2d-gaps.md
  - docs/d2d-bot.md
  - docs/container.md
  - docs/nas-container.md
  - docs/soak-hosts.md
  - docs/quickstart.md
exclusive: []
gates:
  - "docs/research/code-review-2026-10-30.md <= 220 lines, every finding has file:line, scenario and fix, read in code"
  - "docs/plan/STATUS.md hand-written part <= 60 lines, generated block untouched"
  - "go run ./cmd/changecheck prints 0 warnings"
merge: a59ffe9
result: "code review of the session merges: 4 H, 8 M, many L findings; fix cards soakfix, replayfix, pubfix opened; docs hygiene: STATUS rewritten, 18 banners, INDEX, BACKLOG, proposed glossary terms"
```

## 1. Code review (findings only, no code changes)

- [x] 1.1 killtime.go, observe.go hooks, hpfields.go (KILLGAP)
- [x] 1.2 internal/d2dsoakstat gatev2/minute/killtime + cmd/d2dsoak (Poisson bounds, edge cases)
- [x] 1.3 cmd/d2dview tap, read-verb whitelist, SSE, /stream, /api/pub sanitizer
- [x] 1.4 cmd/d2watch changes
- [x] 1.5 tools/publish/*.py sanitizer and runs cache
- [x] 1.6 tools/d2dslot.sh, tools/capacity.sh
- [x] 1.7 Pacer/LateClock (d2dinput, d2denv), DETLOOP guard (unmerged 62ce4b3)
- [x] 1.8 agent/replay 47a76e4, agent/killgap d27781e
- [x] 1.9 docs/research/code-review-2026-10-30.md written, ranked by severity

## 2. Doc hygiene

- [x] 2.1 docs/plan/STATUS.md hand-written part rewritten (<= 60 lines)
- [x] 2.2 `> status:` banners on superseded docs (listed in the report)
- [x] 2.3 docs/BACKLOG.md deduped and ordered
- [x] 2.4 docs/INDEX.md: one-page map by area
- [x] 2.5 docs/research/proposed-glossary-terms.md (guard files untouched)
- [x] 2.6 changecheck 0 warnings
