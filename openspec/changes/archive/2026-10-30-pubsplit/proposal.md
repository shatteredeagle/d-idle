# Proposal: pubsplit

## Why

The user decided (2026-10-30) to scrub once, cut a fresh history, and keep the working repo public-clean by construction, with private ops material in a separate private repo `d2lod-ops`. The mass edit is scheduled for a quiet master; this change adds only the rules, the scanner and the wiring.

## What Changes

- docs/process/public-split.md, docs/process/public-manifest.json, docs/process/leakcheck-allow.txt: the rules, the file-level manifest, the allow list.
- cmd/leakcheck: scanner (warn and fail modes, staged range, JSON), wired into the pre-push hook and the CI checks job as warn only.
- README.md at the root, from the PUBLICPREP draft.
