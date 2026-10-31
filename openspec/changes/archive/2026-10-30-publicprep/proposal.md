# Proposal: publicprep

## Why

The user decided on 2026-10-30 that the d2lod-bot repo is going public soon (ADR 0010 addendum). Before anything leaves the forge we need a read-only audit of secrets, private infrastructure, personal data, legal exposure (game data, GPL patches, third-party text) and repo hygiene, plus a concrete publish plan and a draft README. Nothing is pushed or made public by this change.

## What Changes

- docs/research/public-readiness-audit.md: findings table, publish-plan options with a recommendation, go-live checklist, export/sanitise script design.
- docs/research/README-public-draft.md: draft public README (not yet at the root).
- docs/adr/0010 addendum, docs/decisions.md line, docs/licensing.md open question marked answered (coordinator-authorised).
