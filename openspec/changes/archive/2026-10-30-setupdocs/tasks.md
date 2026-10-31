# Tasks

```yaml
change: setupdocs
status: archived
agent: setupdocs
branch: agent/setupdocs
worktree: ~/code/d2lod-wt/setupdocs
owns:
  - AGENTS.md
  - GLOSSARY.md
  - docs/agents/**
  - docs/process/plugin-pin.md
exclusive: []
gates:
  - "go run ./cmd/changecheck -cli prints 0 warnings"
  - "every relative link added in AGENTS.md, GLOSSARY.md and docs/agents/* resolves (checked by script)"
  - "GLOSSARY.md has at least 60 terms, each with a path, UNSURE marked where not sure"
merge: b41bddf
result: "GLOSSARY (77 terms), docs/agents/*, AGENTS.md skills block + guard rule; plugin pin needs the user: /plugin > Marketplaces > disable auto-update; whether skills find docs/agents/* unverified"
```

## 1. Work

- [x] 1.1 docs/agents/issue-tracker.md, domain.md, triage-labels.md hand-written in the layout the setup skill would produce (read-only look at the plugin; nothing run)
- [x] 1.2 AGENTS.md: `## Agent skills` pointer block and the guard rule (no ADR or GLOSSARY.md writes without the user's grill session or a coordinator brief)
- [x] 1.3 GLOSSARY.md at the root in the domain-modeling format
- [x] 1.4 plugin pin: docs/process/plugin-pin.md (settings change or user steps)
