## Why

The change-card checks are warn-only, so nothing stops a bad master push, and strict mode had no real date (trial-notes.md plans card checks fatal about day 4 and merge checks about day 7). Pending Needs-you items are only seen when the coordinator remembers to look. Hooks make the checks bind and the brief automatic, without any CI runner (the forge CI job waits for runners).

## What Changes

- openspec/.strict.yaml: real fatal-from dates (2026-11-03 cards, 2026-11-06 merges) and an allowlist of the two old grill merges; changecheck -cli exits non-zero for groups past their date
- tools/hooks/pre-push plus tools/install-hooks.sh (core.hooksPath), bypass only with D2_SKIP_HOOKS=1
- tools/session-brief.sh wired as the SessionStart hook in .claude/settings.json
- docs/process/hooks.md and docs/process/ci-checks.md (the command lines a CI job would run)
