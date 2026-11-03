## Why

Three Gate 1 runs ended on two dates; the generator named record, branch and card by date only, so records overwrote each other (the later commit replaced the earlier file) and the generated merge-lines.txt used a format tools/merge.sh rejects. This change restores the three records under unique run-id names.

## What Changes

- Docs only: docs/soaks/gate1-20261102-1939.md, gate1-20261102-2210.md, gate1-20261103-0053.md with their minutes archives, plus docs/soaks/gate1-index-20261103.md. Evidence and verdicts are unchanged; only the title (run id added) and the minutes link were fixed.
- The old PASSED record docs/soaks/gate1-20261102.md is untouched.
- The generator fix lives in the ops repo (not part of this change).

## Overlap (declared)

The stale card gate1run-20261102 owned the glob `docs/probes/soak-logs/gate1-20261102-*`, which covers the new run-id archives. This change narrows that glob (card tasks.md and the docs/OWNERS line) to its one real file, gate1-20261102-minutes.jsonl.gz, so the globs no longer overlap.
