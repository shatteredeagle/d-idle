# Tasks

```yaml
change: publicprep
status: archived
agent: publicprep
branch: agent/publicprep
worktree: ~/code/d2lod-wt/publicprep
owns:
  - docs/research/public-readiness-audit.md
  - docs/research/README-public-draft.md
  - openspec/changes/publicprep/**
exclusive: []
gates:
  - "docs/research/public-readiness-audit.md <= 200 lines with findings table, publish plan, go-live checklist"
  - "go run ./cmd/changecheck: 0 warnings"
  - "tools/openspec.sh validate --all --strict passes"
merge: 3948a3c
result: Docs only, no live game. Audit verified by the git grep / git log commands named in the doc; secrets 0 in tree and history; scanners gitleaks/trufflehog absent.
```

## 1. Work

- [x] 1.1 ADR 0010 addendum, decisions.md line, licensing.md open question answered (coordinator-authorised guard-file edits)
- [x] 1.2 Secrets scan: working tree and full history (`git log --all`), scanners if installed; report type and path:line only
- [x] 1.3 Private infrastructure and personal data scan; classify must scrub / fine / decide
- [x] 1.4 Legal scan: game data, decompiled code, fork patches, SPDX, THIRD_PARTY.md accuracy
- [x] 1.5 Repo hygiene: README draft, AGENTS.md public/private split, .claude/.forge, big objects
- [x] 1.6 Publish plan with options, effort, risks, recommendation, go-live checklist; export script design
