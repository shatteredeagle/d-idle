## ADDED Requirements

### Requirement: Every upstream post is logged
Every push or post to the upstream project (branch push, PR open, comment, reply) SHALL be recorded as a dated line `posted: <date> <what> <url or sha> <user go ref>` in that topic's evidence pack (`docs/research/pr-<topic>-evidence-pack.md`); a post made by the coordinator alone needs the log line, not a change card. Source: user go 2026-10-31 (spec-sme review); backfilled in the packs of PRs 134, 135 and 136.

#### Scenario: Review reply posted
- **WHEN** the coordinator replies to a review comment on an open PR
- **THEN** the topic's evidence pack gains a `posted:` line with the date, what was posted, its URL and the user's go reference

#### Scenario: Post without a line
- **WHEN** an upstream push or comment exists but its pack has no `posted:` line
- **THEN** the pack is incomplete and the line is added before the next post
