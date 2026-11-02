## Why

Community PRs (#19, #22, #23) lost git authorship: AGENTS.md told every agent to commit as `Claude <noreply@anthropic.com>`, and contributors followed it literally. Also, V run numbers in outside docs were not reproducible from the PR.

## What Changes

- AGENTS.md: the Claude identity and trailer apply only to the owner's own agent runs; new rule for contributors (commit as yourself, `Agent:` trailer, `Author:` header line, squash-merge).
- docs/contributing/agent-work.md and the PR template: REPRODUCIBILITY rule (script or exact command, seed, case count; "self-check, not compared with d2d") and two checkboxes.
- docs/process/chunker.md has no Credit section on origin/main yet (branch agent/credits not merged); not touched.

## Impact

Docs and template only. No spec, ADR, glossary or decisions change.
