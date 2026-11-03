## Why

Merge-lines.txt grammar errors (an OWNERS line, a "- docs/BACKLOG.md:" line from the gate record generator) were only found at merge preflight and cost two failed merges of 30+ minutes each.

## What Changes

- tools/merge_lines_check.py (stdlib, tests in tools/ci/tests/test_merge_lines_check.py): lints merge-lines.txt against the grammar the ops merge script applies (V: copied from the ops repo tools/merge_lines.py parse()).
- tools/ci/check.sh: the hygiene part runs `merge_lines_check.py --changed` (only files the branch adds or changes against origin/main, so cards already on main with old bad lines do not fail the gate).
- The ops-repo side (tools/mergeq.sh, docs-only fast path in merge.sh) is committed in the ops repo, not here.

## Impact

Tooling only. No spec, policy, ADR or glossary change.
