## Why

docs/basics-audit.md lists 10 live checks and 2 unknowns (starting gold of a new offline 1.14d character, unidentified-item sell prices) that no offline test can close, and the coverage table needs a live d2d proof line per S1 and S2 row before milestone M1 can be called done. This card runs them on one hidden d2d slot with ledgered cheats on throwaway characters and closes the table.

## What Changes

- docs/basics-live.md: one section per live check (setup, cheat ledger ids, log lines seen, verdict V or U), the two unknowns measured on 2 classes.
- docs/basics-audit.md: the coverage table updated (rows flipped to C with the test name and the live proof line); gaps list closed or pointed at a decision.
- docs/probes/basics/: the raw log excerpts.
- No production code; cheats only through cmd/d2cheat with a reason; each run under 30 virtual minutes, started through tools/d2dslot.sh under tools/agentrun.sh.

## Dependencies

After m1-audittests, m1-stashcap and m1-trips are archived (it proves their fixes) and after xpgap releases soak-slot:3.
