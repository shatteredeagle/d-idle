# Backlog stale rows applied (2026-11-04)

> status: archived 2026-11-05 (cleanup B): finished or superseded, kept for the record. Old path: `docs/research/backlog-apply-20261104.md` (now a stub). Earlier status: done, docs only. Not verified live. Source: docs/research/backlog-stale-20261104.md (25 STALE rows; its table has 31 STALE rows by item number, rows 1-4, 9-13, 19-31, 35-37 are the 25 STALE groups).

Convention (V, read tools/merge_lines_check.py and the BACKLOG: grammar): a merge-lines `BACKLOG:` line is only appended; there is no in-place replace. Each confirmed line is therefore a dated line that says "supersedes" the old one. The old lines stay until the coordinator edits them.

Re-verification (V): archive result:/live: lines read from `openspec/changes/archive/*/tasks.md`; current BACKLOG text grepped in this worktree (line numbers have moved since the check).

## Confirmed (17 groups, lines in openspec/changes/backlogapply/merge-lines.txt)

FIXMANAMODEL (merged 0101ebe5, but live drill still open: worded partial), SOAKSKILLS (offline only), SKSMOKE-A1 (PARTIAL: Fire Bolt only, game.exe blocked), FIXLAIR (offline, not live), ROAMLEG (live DRILLS23), FIXMOUTH/FIXMOUTH2/FIXDENBOUNCE (fixmouth2 failed DRILLS17, closed DRILLS19), NEXTHITFIX (offline, PR not opened), fixmisc F4, fixstall3/4, fixmeter, fixf17, fixtalk (all live PASS), fixportalmelee (partial), fixbuff, fixrecast (caveat kept), modsprep (not live), gen-invariants (offline).

## Dropped (8 groups)

| item | reason |
|---|---|
| FORKSTATS (row 4) | Line 21 "ready candidate FORKSTATS" is about the skill-bonus stats (127/83/188/97/126/107); the archived card (lab2-stats) did gear stats and weapon base damage, and line 203 already states it. Different scope, not provably done. |
| sknova B1, B2, B5, B6 (row 10) | The archived card is a trace only ("no PR candidate", bot findings B1-B6); no fix is merged, so "ready fix card, record only" is still true. |
| MISSILEFIX3 (row 19) | Archive result: docs only, PRs wait for the owner's go. The line says the same; still true. |
| VERIFY-NMLEVEL / nmlevel-v5 (row 20) | Result: docs and tools only, not live; still waits for the go. Still true. |
| BLIZZARDFIX (row 21) | Result: PR draft READY, nothing upstream; "held for the user's go" is still true. |
| VERIFY-MONSTERPACE (row 35) | The line is a verifier verdict (READY WITH CHANGES), not an open state; the archive confirms it. |
| VERIFYPRICES (row 36) | Verdict line (NOT READY as is), still the archived verdict. |
| VERIFY-PLAYERROLL (row 37) | Verdict line (READY WITH CHANGES), still the archived verdict. |

UNSURE and CURRENT rows: untouched.
