## Why

The spec-sme review (2026-10-31) found 69 of about 125 archived cards of 10-30 and 10-31 carrying skip_specs, several of them changing observable output, because the README never says when skipping is allowed. Upstream posts (PR pushes, comments, replies) were also made without a dated record in the evidence packs.

## What Changes

- openspec/README.md: when skip_specs is allowed, commits without a card, archive prefix rule, deltas from the propose step.
- upstream-pr delta: every push or post to the upstream project is logged as a `posted:` line in the topic's evidence pack; backfilled for PRs 134, 135 and 136.
- Missing .openspec.yaml for the archived hitlen card.
