## Why

The user decided on 2026-10-30 (docs/process/matt-pocock-skills-fit.md) to adopt the Matt Pocock skills plugin for dedicated grill sessions, but by hand: the files its setup skill would create are written here without running the skill, so the skills find our openspec layout, a GLOSSARY.md exists, and agents cannot write ADRs or the glossary on their own.

## What Changes

- docs/agents/issue-tracker.md, domain.md, triage-labels.md describing our layout (openspec change cards, no gh, no .scratch, single context)
- AGENTS.md gets the `## Agent skills` pointer block and a guard rule
- GLOSSARY.md at the repo root
- docs/process/plugin-pin.md records the pinned plugin version and how it is held
