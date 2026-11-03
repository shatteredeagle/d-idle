## Why

Work carried into the upstream decomp project must credit people the way its maintainer counts them (git author, `Traced-by`, `Reported-by`), without AI lines in commit messages, and nobody may get lost.

## What Changes

- Optional `credit:` block in the card header (openspec/README.md), `tools/credit/credit.py` (trailers, PR Credits section, check) with unit tests wired into tools/ci/scripts.sh.
- docs/process/credit.md, docs/community/credits-ledger.md, the upstream-use sentence in chunker.md and agent-work.md; `credit:` seeded on the palaura, paldur and palskill cards.

## Impact

Docs, tooling, cards. No Go, spec, ADR, glossary or decisions change. Our licence is GPL-3.0-or-later (LICENSE, README, ADR 0010), compatible with upstream.
