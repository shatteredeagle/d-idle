## Why

Four rule sets live only in prose (AGENTS.md, docs/process/*.md, ADR 0010, a memory note), so no check covers them and agents drift. The user decided on 2026-10-30 to turn them into OpenSpec capability specs so changecheck and validate cover the form and a named tool evidences each rule.

## What Changes

- New capability spec `upstream-pr`: proof standard and etiquette for PRs to the upstream decompilation project.
- New capability spec `process-hygiene`: every long process registered, reaped by record, orphans never killed.
- New capability spec `public-split`: public and private classes, leak-free by construction, the go-live gate flip and the one-time history cut.
- New capability spec `licence-policy`: GPL-3.0-or-later, SPDX headers, third-party code, distribution.
- ADDED deltas only; the archive step applies them to openspec/specs. No ADR, glossary or decisions.md edits, no code.
- Evidence docs stay where they are: docs/process/upstream-pr-standard.md, docs/process/public-split.md, docs/licensing.md, docs/process/spdx.md, AGENTS.md cleanup section.
