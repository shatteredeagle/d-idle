## Why

Work package A of docs/process/public-split.md: split the private ops material out of the working repo before the repo goes public. User decision 2026-10-30: one scrub, one fresh-history cut, private ops in the private repo d2lod-ops.

## What Changes

- Private-class files (lab docs, lab scripts, board, publisher, stream tooling, agent definitions) leave this tree; docs and scripts that others call stay as pointer stubs.
- tools/ops.sh, tools/env.sh, local.env.example; AGENTS.md split (public rules here, lab rules in d2lod-ops AGENTS.private.md); CLAUDE.md imports AGENTS.md.
- tools/ops-cutover.sh prints and verifies the quiet-master cutover steps (no destructive action).
- Branch only: the coordinator merges on a quiet master and runs the checklist.
