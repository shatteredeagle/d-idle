## Why

The public D.idle page showed generic status ("paused right now"). The user approved (2026-10-30) a page with live status and the d2d direction.

## What Changes

- live runs section from the per-minute bot log lines (runs.py, runs.json), "between runs" with last-run summary
- "Now building on d2d" block (ADR 0009), upstream credit and open PR links
- bare agent merges only appear in the shipped list through a hand-written override
- sanitizer rule private-name (machine and person names), milestone text scrub
