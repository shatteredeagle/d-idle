## Why

docs/soakskills.md lists the per-skill soak fields that `d2dsoak report` prints as n/a because the bot does not log them. The Gate 2/3 audit (docs/research/sorc-act-gate.md section 2) wants them.

## What Changes

- Emit the missing log lines from the bot's combat code (small, one place each) and teach internal/d2dsoakstat/skills.go to read them: a `cast result` line (skill, made effect, cut before the action frame), mana per cast, the last skill of a kill, an armor-state sample on the minute line, a `build` line with branch and commit, the Lightning cast ticks.
- No behaviour change to the bot's play: logging only. Skills are named from data, never by numeric id; test with two builds.

## Impact

Offline only, not verified live. Logging touches internal/combat; declared overlap with main-owned files, small edits.
