## Why

The user asked ("is our repo overly documented", then "do the trim now", 2026-10-31) for the doc pile to be trimmed so a new reader finds things. docs/ holds about 440 files; docs/research alone has 50 with no index, finished PR drafts sit beside live ones, and docs/BACKLOG.md lists items that are merged.

## What Changes

- docs/research/README.md: one topical index with status per doc; docs/README.md: one-page entry point.
- Finished drafts (posted PRs, executed review replies) and superseded docs move to docs/archive/ with a stub and a status banner at the old path; nothing is deleted.
- docs/BACKLOG.md rewritten to the open items; the old text is kept in docs/archive/BACKLOG-2026-10-30.md.
