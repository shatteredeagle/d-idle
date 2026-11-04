# Stale status check: docs/BACKLOG.md and docs/decomp-queue.md (2026-11-04)

> status: check only. No shared file was edited. The replacement lines in the last section are proposals for the merger.

Card: `backlogcheck` (docs only, `skip_specs: true`).

## Method

1. Scanned `docs/BACKLOG.md` (577 lines) for status words (`ready`, `proposed`, `pending`, `in progress`, `open`, `deferred`, `READY`). 117 lines matched. `docs/decomp-queue.md` (69 lines) was scanned the same way.
2. Named 40 candidate items (the cap) and checked each with:
   - `git log origin/main -F -i --grep=<name>` (commit subjects; V, the local `origin/main` ref at `9bc9a8b3`, not fetched)
   - `ls openspec/changes/archive | grep -i <name>`
   - `ls openspec/changes | grep -i <name>`
   - `status:` of any live card, and the `result:` and `live:` lines of archived cards (V, read from the files)
   - `git merge-base --is-ancestor <hash> origin/main` for every hash cited below (all true, V)
3. Verdicts:
   - STALE: the item's work is merged and archived, or its stated open state is no longer true per the archived result.
   - CURRENT: no merge or archive, or the archived card itself still says the item is open.
   - UNSURE: the evidence does not settle the stated state (for example an upstream PR state, which needs GitHub access that this check did not use).

Limits: upstream PR state was not checked (no network use). "Archived" is taken as merged, following AGENTS (the coordinator archives at merge). Line numbers are for `docs/BACKLOG.md` unless the file is named.

## Table

| # | item | line (file) | stated status | evidence (commit or archive dir) | verdict |
|---|---|---|---|---|---|
| 1 | FIXMANAMODEL | 194 | ready, S, offline | merged `0101ebe5` (on origin/main); card `changes/fixmanamodel` still in-progress | STALE |
| 2 | SOAKSKILLS | 195 | ready, S, offline | archive `2026-11-03-soakskills`, `b6ad2ab7` | STALE |
| 3 | SKSMOKE-A1 | 192 | ready, M, live slot | archive `2026-11-03-sksmoke-a1`, `5204e9e8` | STALE |
| 4 | FORKSTATS | 21, 203 | ready candidate | archive `2026-11-01-forkstats`, `f500c24b`; merge `120e5341` | STALE |
| 5 | FIXSKILLLVL | 21, 197 | card open, awaits a live proof | merged `eced68bf`; card `changes/fixskilllvl` in-progress; no live result found | UNSURE |
| 6 | NEXTHIT | 189-190 | proposed, L, proof only | merges `458ecd4f` (evidence pack, PR not ready), `d8e48e04`; card `changes/nexthit` in-progress | UNSURE |
| 7 | LIGHTSEQ | 191 | proposed, S to M | no card, no commit with that name; SORCDIFF merged `013512a3` covers the lightning sequence | UNSURE |
| 8 | ARMORHIT | 193, 532-533, 543 | proposed, M; line 543 READY but not opened | merged `4fbfbf24` (docs, card kept open); archive `2026-11-01-verifyarmorhit`; card `changes/armorhit` in-progress | UNSURE |
| 9 | FIXLAIR | 490 | ready fix card | archive `2026-11-01-fixlair`, `68e97db3` | STALE |
| 10 | sknova B1, B2, B5, B6 | 510, 511, 514 | ready fix card, record only | archive `2026-11-01-sknova`, `d856abae` | STALE |
| 11 | ROAMLEG | 531 | ready fix card | archive `2026-11-01-roamleg`, `a1d85cc5` | STALE |
| 12 | FIXMOUTH, FIXMOUTH2, FIXDENBOUNCE | 48, 501, 520 | live proof pending; implementation in progress | archives `2026-11-01-fixmouth` (`d83ceb0`), `2026-11-01-fixmouth2` (`b2c88b8`); merge `6344c73b`; archived results say den bounce closed live (DRILLS19) | STALE |
| 13 | NEXTHITFIX | 528 | READY | archive `2026-11-01-nexthitfix`, merge `d8e48e04` | STALE |
| 14 | PALHOLD, PALHAND | 537, 540 | proposed, from palrecon | no card, no commit with either name | CURRENT |
| 15 | SKMELEE+ | 538 | proposed | merge `1fccb880` (skmeleewire); archives `2026-11-01-skmelee`, `2026-11-02-prbuildskmelee3`; the exact row 20 change is not traced | UNSURE |
| 16 | PALAURA | 539 (also done line 286) | proposed | archive `2026-11-03-palauratrace` (`83fa8d2`); its result says live unverified | UNSURE |
| 17 | PALDUR | 541 | proposed, decomp plus drill | archive `2026-11-02-paldur` (`b415aef`, docs only), `2026-11-02-verifypaldur`; card `changes/paldurtrace` in-progress | UNSURE |
| 18 | FREEZERESUME | 548 | open: verifier edits, the user's go | archive `2026-11-01-freezeresume` (`bc86cec`); its result says PR body edits pending, not opened upstream; later state unknown | UNSURE |
| 19 | MISSILEFIX3 | 249 | both drafts ready for the user's go | archive `2026-11-02-missilefix3`, `39d3e57a` | STALE |
| 20 | VERIFY-NMLEVEL / nmlevel-v5 | 260, 321 | READY WITH CHANGES; waits for the go | archive `2026-11-02-nmlevel`, `a6e28fcf` (result: docs and tools only) | STALE |
| 21 | BLIZZARDFIX | 208 | READY, held for the user's go | archive `2026-11-01-blizzardfix`, `32fd9ff6` | STALE |
| 22 | fixmisc F4 | 443 | live proof pending | archive `2026-10-31-fixmisc`, `749a0134`; result: D-sync PASS x3 (DRILLS6) | STALE |
| 23 | fixstall3, fixstall4 | 447, 448 | live proof pending | archives `2026-10-31-fixstall3`, `2026-10-31-fixstall4`, `0b0bdb25`; results: PASS (DRILLS8, DRILLS9) | STALE |
| 24 | fixmeter | 446 | live proof pending | archive `2026-10-31-fixmeter`, `9eda4b06`; result: PASS (DRILLS10) | STALE |
| 25 | fixf17 | 464 | live proof pending | archive `2026-10-31-fixf17`, `9eda4b06`; result: PASS (DRILLS10) | STALE |
| 26 | fixtalk | 466 | live proof pending | archive `2026-10-31-fixtalk`, `9eda4b06`; result: PASS (DRILLS10) | STALE |
| 27 | fixportalmelee | 30 | card in progress; live check open | archive `2026-11-01-fixportalmelee`, `b4752a9d`; result: decision proven on a snapshot, leave-game and hard hit still unverified | STALE |
| 28 | fixbuff | 32 | card in progress; live proof open | archive `2026-11-01-fixbuff`, `b689235c`; result: proven live on d2d | STALE |
| 29 | fixrecast | 25 | card in progress | archive `2026-11-01-fixrecast`, `8e783b17`; clock path proven live, state path untested | STALE |
| 30 | modsprep | 23 | card in progress | archive `2026-11-01-modsprep`, `ec345825` | STALE |
| 31 | gen-invariants | 16 | awaiting the user's approval to launch | archive `2026-10-31-gen-invariants`, `a83b28d9` | STALE |
| 32 | wpcooldown | 198 | live drill pending | archive `2026-11-01-wpcooldown`, merge `ddf6213d`; the archived card still says live not verified | CURRENT |
| 33 | snaplib | 231 | first live restore pending | archive `2026-11-01-snaplib`, `fd335257`; result: game load of a renamed character untested | CURRENT |
| 34 | tomestat | 346 | live check pending | archive `2026-11-02-tomestat`, `ba434dde`; result: live check pending | CURRENT |
| 35 | VERIFY-MONSTERPACE | 419 | READY WITH CHANGES | archives `2026-11-02-monsterpace` (`7b7c0f90`), `2026-11-03-verifymonsterpace` | STALE |
| 36 | VERIFYPRICES | 299 | NOT READY as is | archive `2026-11-02-verifyprices`, `583c1e38` | STALE |
| 37 | VERIFY-PLAYERROLL | 328 | READY WITH CHANGES | archive `2026-11-02-verifyplayerroll`, `7cc1008b` | STALE |
| 38 | goldfloor | 186 | deferred | card `changes/goldfloor` status deferred; levers merged `a4035859` (goldfloor-impl) | UNSURE |
| 39 | stream-grid, d2dmesh | 155 | in progress | no merge, no archive; both cards say deferred (label mismatch only) | CURRENT |
| 40 | wppr upstream candidate | decomp-queue.md 24 | READY (local), pending the user's go | archive `2026-10-31-wppr`, merge `aa947216`; the same row says the upstream PR merged as #153 | UNSURE |

Summary: 40 rows (40 candidate items, one row per group where several items share a card): 25 STALE (rows 1-4, 9-13, 19-31, 35-37), 5 CURRENT (rows 14, 32-34, 39), 10 UNSURE (rows 5-8, 15-18, 38, 40).

## Notes for the coordinator

- Rows 5, 6, 8, 15-18, 38 and 40 need a second look with the upstream PR state or the card owner, not a text edit.
- Rows 22-29 are archived with live proof recorded in their results, so the open wording is stale even though the archived `live:` lines are not in the BACKLOG line.
- Row 14: palrecon proposals with no card. They are the only items here with no trace at all, which is why they are CURRENT.
- Row 39: the label "in progress" does not match the cards' `status: deferred`. Changing the label is a coordinator call.

## Shared-file lines for the merger

These are proposals only. None is in any `merge-lines.txt`.

BACKLOG: FIXMANAMODEL: merged to main (0101ebe5); card changes/fixmanamodel still in-progress, archive pending.
BACKLOG: SOAKSKILLS: merged and archived (2026-11-03-soakskills, b6ad2ab7).
BACKLOG: SKSMOKE-A1: merged and archived (2026-11-03-sksmoke-a1, 5204e9e8).
BACKLOG: FORKSTATS: merged and archived (2026-11-01-forkstats, f500c24b).
BACKLOG: FIXLAIR: merged and archived (2026-11-01-fixlair, 68e97db3).
BACKLOG: sknova B1, B2, B5, B6: merged and archived (2026-11-01-sknova, d856abae).
BACKLOG: ROAMLEG: merged and archived (2026-11-01-roamleg, a1d85cc5).
BACKLOG: FIXMOUTH, FIXMOUTH2, FIXDENBOUNCE: merged and archived (2026-11-01-fixmouth, 2026-11-01-fixmouth2, merge 6344c73b); den bounce closed live in DRILLS19 per the archived results.
BACKLOG: NEXTHITFIX: merged and archived (2026-11-01-nexthitfix, d8e48e04).
BACKLOG: MISSILEFIX3: merged and archived (2026-11-02-missilefix3, 39d3e57a).
BACKLOG: VERIFY-NMLEVEL and nmlevel-v5: merged and archived (2026-11-02-nmlevel, a6e28fcf).
BACKLOG: BLIZZARDFIX: merged and archived (2026-11-01-blizzardfix, 32fd9ff6).
BACKLOG: fixmisc F4: merged and archived (2026-10-31-fixmisc, 749a0134); live D-sync PASS x3 in DRILLS6 per the archived result.
BACKLOG: fixstall3, fixstall4: merged and archived (2026-10-31-fixstall3, 2026-10-31-fixstall4, 0b0bdb25); live PASS in DRILLS8 and DRILLS9.
BACKLOG: fixmeter: merged and archived (2026-10-31-fixmeter, 9eda4b06); live PASS in DRILLS10.
BACKLOG: fixf17: merged and archived (2026-10-31-fixf17, 9eda4b06); live PASS in DRILLS10.
BACKLOG: fixtalk: merged and archived (2026-10-31-fixtalk, 9eda4b06); live PASS in DRILLS10.
BACKLOG: fixportalmelee: merged and archived (2026-11-01-fixportalmelee, b4752a9d); leave-game action and hard hit stay unverified live.
BACKLOG: fixbuff: merged and archived (2026-11-01-fixbuff, b689235c); live proof on d2d per the archived result.
BACKLOG: fixrecast: merged and archived (2026-11-01-fixrecast, 8e783b17); clock path proven live.
BACKLOG: modsprep: merged and archived (2026-11-01-modsprep, ec345825).
BACKLOG: gen-invariants: merged and archived (2026-10-31-gen-invariants, a83b28d9).
BACKLOG: VERIFY-MONSTERPACE: merged and archived (2026-11-02-monsterpace, 7b7c0f90; 2026-11-03-verifymonsterpace).
BACKLOG: VERIFYPRICES: merged and archived (2026-11-02-verifyprices, 583c1e38).
BACKLOG: VERIFY-PLAYERROLL: merged and archived (2026-11-02-verifyplayerroll, 7cc1008b).
