## Why

`d2dsoak report` still prints n/a for kills credited per skill and hits per cast (docs/soakskills.md, docs/logfields.md). The Gate 2/3 audit wants them.

## What Changes

- Log-side: credit each kill to the skill whose hit killed (the `last_skill` key exists on the kill line; make per-skill kill credit explicit) and log hits per cast where the combat code knows them (missile hits, nova targets). Logging only, no behaviour change.
- Report: print both fields; casts per kill becomes per-skill credited.

## Impact

Offline only, not verified live. Skills by name from data.
