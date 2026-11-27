# cain.chain Paladin seed 11: why it was called flaky (TASK-680 AC 3), diagnosis and a night experiment

> status: current, 2026-11-27. Docs only, no run made for this note. Tags: V read in the named file or code, S secondary (a doc quoting a run), U unverified or inference. Classic 1.14d lab d2d, never D2R. Every cain.chain setup is CHEATED and IMPOSSIBLE (clvl 32, free merc): mechanics evidence only. The results ledger holds no line yet (`docs/process/proving-ground-status.md` shows "(no runs recorded)", V), so the outcomes below come from probe docs, not from the ledger.

## 1. Recorded outcomes, cain.chain (all on the builder lab slots, `-parallel 1`, 80 virtual min)

| # | when / source | class seed | bot state | verdict | where it ended, wall |
|---|---|---|---|---|---|
| 1 | 2026-11-19 stamp 165941 (`provingground-run-2026-11-19.md`, results json V) | Pal 11 | before cainfix3 | RED | record 4 = 0x0008, `bkd` still packed; taps `start scroll`, `start stones`; chickens in Stony Field, "area banned after repeated deaths", 86 area changes; 2106 s |
| 2 | cainfix3 stamp 194905 (S `provingground-cainfix3-2026-11-19.md` line 16) | Pal 11 | cainfix3, crowd override NOT wired into the d2bot fighter adapter | GREEN | 815 s wall |
| 3 | cainfix3 stamp 201758 (V same doc, "Still red") | Pal 11 | cainfix3, override wired | RED | free_cain after a potion trip from Tristram: record 4 stays 0x0018; 3 counted failures park the chunk |
| 4 | job 20261119-232655-7217 (S cainchain) | Pal 11 | cainchain fixes | GREEN | 18 min |
| 5 | job 20261120-000256-0f09 | Pal 5 | same | GREEN | 24 min |
| 6 | job 20261120-002704-6dae | Sor 3 | same | GREEN | 34 min |
| 7 | job 20261120-010053-a896 | Pal 11 | same | GREEN | 14 min |
| 8 | pgfix4 A, job 20261120-141046-7941, digest 2ffabd0c3a935a59 (S) | Pal 11 | + portal retry | GREEN | 987 s |
| 9 | pgfix4 B, job 20261120-144848-1dc8 (S) | Pal 11 | same | GREEN (exit 0, results lost) | about 22 min |
| 10 | pgfix4 Sor 3 A and B (jobs 20261120-142938-d413, 150941-1d57) | Sor 3 | same | GREEN, GREEN | 19 min, 871 s |
| 11 | cain-tree-scroll 2026-11-21 (S `cain-tree-scroll-2026-11-21.md`) | Pal 11 | unfixed then fixed tree | GREEN, GREEN (one more run BLOCKED by the wall limit under host load 25, not judged) | |

Summary (S): Pal 11 is 2 RED then GREEN in the earlier bot states (rows 1 to 3), and 6 GREEN, 0 RED since the cainchain and pgfix4 fixes (rows 4, 7, 8, 9, 11 twice). The card text "FLAKY" (row 2 vs 3) compares two DIFFERENT bot builds: row 2 lacked the adapter wiring that row 3 had (the doc says so). It is not a same-binary flip. Same-binary pairs on record: rows 4/7 (GREEN, GREEN), 8/9 (GREEN, GREEN; but pgfix4 notes run A came before a wiring fix, S H6 in `proving-ground-gate.md`). No digest pair was ever compared (two result files lost, S H7). The generated status counts flips only per ledger line, so it cannot show this yet.

## 2. What failed in the red runs (V where a log was read)

- Row 1: stones chunk gated by the death ban plus a 55-life hero; fixed by cainfix3 F1 to F4 (S).
- Row 3: the chain is GREEN through scroll, read, stones and free_cain start; at free_cain the hero fought Tristram (crowd 14 to 24), needed potions 8 s in, the town-portal cast was cancelled by `crowd-retreat` ("cancelled, nothing clicked"), the walk fallback reached camp, and the return used portal id 805306369 left from the earlier Dark Wood trip: the hero came back to the Dark Wood, not Tristram (V cainfix3). Fixed in two parts by cainchain (`Town.portalFrom`, `ErrPortalStale`, cage step re-enters through the Stony Field portal; S) and pgfix4 (`bot.portalRetryWaitSec` (3), retries a cancelled cast once; unit-tested only, no live scenario, S).
- A fifth failure class was a race between the board accept and the level planner (cainchain finding 3; the doc itself calls it "the most likely run-to-run difference of the earlier flaky verdicts", U: no pair was run before the fix).

## 3. Ranked hypotheses for why seed 11 flipped

| rank | hypothesis | tag | support | status after today's merges |
|---|---|---|---|---|
| 1 | Different binaries, not randomness: rows 2 and 3 ran different bot builds (crowd override wired or not) and rows 1 to 3 predate the cainchain/pgfix4 fixes | S | the cainfix3 doc states it; six GREEN since | the flake label is probably stale; AC 3 was never tested on one binary |
| 2 | Crowd on arrival in Tristram (monster placement and density from the seed, lab d2d packs may be denser than game.exe, S l3 doc) forces a potion trip, a cancelled cast and a stale or wrong portal | S/U | row 3 log; Tristram crowd 14 to 24 | defended in code (stale-portal refusal, cage re-entry, one retry) but the defence is exercised only when the pack and timing coincide; Pal 11 greens may simply not have hit that case (U: no green log was checked for "cancelled, nothing clicked") |
| 3 | Board accept vs level planner race (the planner runs between chunks while the hero stands in Tristram; every target "no route") | S | cainchain finding 3; fixed by `LevelTo` wait | fixed; residual only if a new planner path is added (U) |
| 4 | Tap timing: the board tapper polls on the WALL clock (20 s, then 5 s), which is 25 virtual s to 5 virtual min depending on builder load, so Start and Hand in land at a different game tick every run (V by code, `cmd/d2scenario/tap.go`; S H1) | S | three Start taps per run; row 11 shows load 25 cut a run | unchanged; the tapper now holds off Start while a hand-in is pending (V `tap.go` `handInPending`) |
| 5 | Hand-in tap going stale or being refused ("already waiting") at the report step: the 5 minute `handInTimeout` (V `internal/questboard/desk.go:36`) and the one-slot mailbox | S | no recorded red ended at the report; all reds ended earlier | TASK-1051 (merged today) logs `hand-in queued/served/dropped`, drops a timed-out request and serves the tap from an idle goal in town: removes this risk; live drill still open (V card result) |
| 6 | Wall-limit cut judged as a verdict (H5): a loaded builder can end a run early; row 11 saw one such run | S | proving-ground-gate H5 | harness, unchanged; BLOCKED rather than RED is the only handled case |
| 7 | Navigation gate (Stony Field to Tristram nav hint absent, roaming 5 minutes) | S | cainfix3 "free_cain started outside Tristram" | worked around by the cage-step re-entry; nav itself untouched |
| 8 | RNG: d2d seed is set by the scenario (seed 11), but the observer d2d is started without one (H8) and the bot itself has wall-clock reads in its loop (gate doc 2.3) | S | digests never compared | unchanged |

## 4. Which of today's merges change the picture (all main 2026-11-27)

- akarafix (TASK-1051): makes hypothesis 5 observable and removes the stale request; it adds log lines usable as discriminators. Offline tests only (V card). It does not touch Tristram or portals.
- stepexec chunks (`quests.stepExec`, default 0, dark-launch; V `docs/knobs.md` line 562): when on, the Cain scroll, Akara read, staff transmute run on the generic executor with bounded retries. The cain.chain scenario does NOT set it, so the chain runs the old handlers. Knob on is a new variant, not the default: do not mix it into the discrimination run. A sibling scenario `cain.read.stepexec` exists (S, `cain.json`).
- tyrael (TASK-1053) and act2chunks: Act 2 only; do not touch the Act 1 chain (S, by task titles; U that nothing shares code with the hand-in path, and the reviewer should confirm).
- town stages (town7, townguard): short-gold priority and guard refusal wording; the chain hero holds 8000 gold, so these should not trigger (U).
- cainscroll (TASK-776, quest-word-based scroll step) is merged and cain.chain Pal 11 was GREEN with it (row 11).
- Nothing merged today changes the Tristram crowd, portal, or planner-race paths; hypotheses 2, 3, 4, 7 are as of rows 8 to 11.

## 5. The discriminating experiment (one night run, at most 40 min wall)

Cost of one run: 14 to 34 min wall, 871 to 987 s typical (S rows 4 to 10). Sequential `-parallel 1` fits only one run in 40 min, so the experiment needs parallel lab slots: run it only if `tools/capacity.sh` and the builder show at least 4 free slots (U: the number of slots was not read; if fewer, cut the matrix from the bottom of the table and say so in the report). Night window 23:00 to 07:00 EST, hidden instances only, per the private rules.

Command shape, one job per row, started together (`--lane long --detach`), from a worktree on current main:

`tools/provingground/run.sh -id cain.chain -class Paladin -seed <n> -parallel 1` (knob overrides via the scenario's `trigger.knobs`, a scratch copy of the scenario, not the committed file)

| job | class, seed | knobs | question |
|---|---|---|---|
| A1, A2 | Pal 11 | defaults | same-binary repeat pair on today's main: the verdict AND the digest compare; the real flake test |
| B | Pal 12 | defaults | is it a seed-11 property or general |
| C | Pal 13 | defaults | same, third seed |
| D | Pal 11 | `bot.portalRetryWaitSec` 0 | does the chain still pass without the one retry (H2: how much is the retry carrying) |

Optional if more slots: Sor 3 default; Pal 11 `combat.crowdAbandonQuestN` 0 (only to prove the crowd override matters; it is expected RED in Tristram).

Stop rules: wall limit per job 40 min; a job cut by the wall limit is BLOCKED, never counted; do not repeat on a load above the builder's normal (S: load 25 cut a run).

Read from each job's `*.bot.log`, `*.taps.log` and results json (names as in row 1, V):

| log evidence | separates |
|---|---|
| virtual time of each tap (`TapsAt` in results) for start scroll, start stones, hand in, across A1 and A2 | H4: a spread of more than 1 virtual min between A1 and A2 on the same tap, with the same verdict, shows the timing varies and the bot tolerates it |
| `cancelled, nothing clicked` count, then `portal retry` lines, then `ErrPortalStale` or a return to area 2 or 3 | H2: a green with zero cancels proves nothing about the defence; a green with cancels and a retry proves it. Job D with a cancel and no retry is the sharp test |
| `crowd` / `crowd-retreat` / `abandoning the walk` lines in area 38 (Tristram) with counts | H2 and crowd on arrival; compare density across seeds 11, 12, 13 |
| `no route`, `unreachable`, `LevelTo` waits between the stones chunk and free_cain | H3 |
| `quest board: hand-in queued`, `served`, `dropped` and the gap between queued and served | H5 (new in akarafix); any `dropped` or "already waiting" is a finding |
| `area banned after` (death ban text), `death autopsy` | the old row 1 cause |
| last line `virtual run length reached` present or absent | H6: a run without it was cut early |
| final `quest_bits` got value | 0x2019 good; 0x0018 means free_cain parked (row 3 shape); 0x0008 means the stones chunk never finished (row 1 shape) |

Verdict rule to write into the card: if A1 and A2 are GREEN with equal digests and B, C are GREEN, tick AC 3 and drop "flaky" (the old flake was two bot builds); if A1 differs from A2 on the same binary, keep it open and report the first divergent log line plus the tap time difference; if only D fails on cancel with no retry line, the retry is load bearing and the unit-tested path needs a scenario (the fork verb the pgfix4 doc names, U).

## 6. Not done

- No run was made here. No ledger line exists to pull outcomes from.
- The number of free lab slots and the load of the builder were not read.
- Whether the Tristram crowd on seeds 12 and 13 matches seed 11 is unknown; seeds 12 and 13 are not in the scenario matrix (`seeds: [11]`), the run.sh `-seed` flag is used as pgfix4 did with Pal 5 (S).
