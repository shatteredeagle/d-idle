# Roadmap: milestones, work breakdown, operating rhythm (agent ROADMAP, 2026-10-31)

> status: ADOPTED by the user on 2026-10-31: decisions D1 to D6 in section 8 are DECIDED. Card `roadmap`. Reviewed by d2-sme and spec-sme (section 6). Tags: V = checked in the repo or by a command named here, S = read in a doc, U = judgment. Numbers that have an owner (gates, knob defaults) are cited from their spec or docs/knobs.md, not copied. The cards are `openspec/changes/m1-*`, `m2-*`, `m3-*`.

## 1. Diagnosis: where the effort goes and why it feels like flailing

Counts: `git log --since=7.days --merges` in the old-history checkout (the project started 2026-10-28), merge subjects classified by branch-name prefix with a regex (approximate; classes in the ROADMAP report). 466 merges in three days (101, 218, 188 per day).

| theme | merges | share |
|---|---|---|
| bot behaviour (town, combat, nav, quests, merc, boss, classes) | 203 | 44 % |
| process, infra, CI, public split, docs | 96 | 21 % |
| d2d engine, harness, fork, determinism, replay, mesh | 51 | 11 % |
| soak and measurement tooling | 45 | 10 % |
| upstream PRs and game.exe traces | 26 | 6 % |
| container, fleet, wizard | 24 | 5 % |
| stream, viewer, dashboard | 21 | 4 % |

1. **The gated number is not converging, and no card owns it.** XP/h on d2d (fresh Sorceress, 2 virtual hours): 3013 (D2DLOOP), 3830 (KILLFIX), 3869 (ECONOMY), 4985 (soakfull), 5043 +- 210 (xpgap, 2026-10-31) against the early baseline 6600 (V: docs/d2d-soak-full-fix.md, docs/d2d-xp-gap.md on agent/xpgap); kills/h 197 against 500 (late baseline, not comparable). Five diagnoses agree (per-kill-time-split, soak-full-fix, xp-gap, basics-audit, xpgap A/B): the sinks are town trips (14.5 % of time, 4 min each, 62 % of gold on portal scrolls), the walk back to the field (14 %), roaming a cleared field and chicken restarts (28 % and 21 % of the long kill gaps). xpgap also showed that recovery farming is NOT the loss (same kills/h as the quest step) and that three knobs A/B-tested with 4 seeds per arm cannot beat the noise (SE 7 to 9 % of the mean). Each fix card moved the number 20 to 30 % or not at all; no card's gate was "the early fidelity gate passes".
2. **Seven streams, one merge queue.** Open at once on 2026-10-31: xpgap, basicsfix, specfix (five cards), soakwatch, d2dmesh, stream-grid, three container cards, two deferred Game.exe cards, three open upstream PRs and one drafted (V: STATUS generated block, `git branch`). Mesh, stream, container and upstream do not feed the user's stated milestone (BACKLOG section 0). One coordinator merges and archives all of it; the merge checklist (whole-repo test on the worker, archive, OWNERS, STATUS) is the throughput limit, not the four d2d slots (a 2 virtual hour run takes 8 to 9 wall minutes, V soakfull).
3. **The same bug gets fixed four times.** Fresh-game livelock: runab 1f4b62f, livelockfix 5f864c2, the soakfull hotfix 9b33985, botsmall forgetGame (V: docs/d2d-soak-full-fix.md anomaly 1). Poverty sell trips: economy, xpgap povertySellMin, basics-audit G2. TP-before-potions: shoptp, scrollfirst, xpgap tpMinGold, basicsfix potions-first. Cause: fixes land as knobs default off or one-line patches without a scenario test and a spec line that state the wanted end state, so the next soak finds the same symptom and the audit tests pin the old behaviour (V: `PinsCurrent` tests in internal/town/basicsaudit_test.go).
4. **Unverifiable obligations pile up.** Every shared-code change since ADR 0009 carries "needs a G1/G2 check" (champ, botsmall, navstall, xpgap), but Game.exe is frozen and no G1 can run (V: ADR 0009 addendum). A change to shared code is verified on d2d or marked U; the note is dead weight.
5. **Measurement debt sits in the gate path.** d2dsoak can exit 0 on INCONCLUSIVE and count empty seeds as clean exposure (review H1, S); the early baseline is one Game.exe run with XP/h and deaths/h only, no kills/h (V: tools/d2dsoak/baselines-early.json); and no A/B protocol says how many paired seeds a claim needs. A PASS cannot be claimed and a 10 % improvement cannot be seen.
6. **Merged on 2026-10-31 without a milestone** (12 merges in the public log, V): golivepre, golivepre2 (user goal: public), doctrim, streamfix, hitlen, lutproof (upstream), botsmall, champ, smallfixes, navstall, basicsaudit. Two of twelve are on the gate path (navstall fixed the seed-11 spin; basicsaudit is M1 preparation).

The bottleneck is bot policy on the early game (trips, walk, roam) plus the absence of a milestone exit test that each card must move, with the measurement tooling one step behind. Engine fidelity is secondary (XP per monster and spawn placement equal game.exe, V docs/research/xp-award-trace.md, spawn-density-trace.md); compute is not the limit; process overhead is self-inflicted by the number of parallel streams.

## 2. Vision, as the repo states it

PLAN.md: "a self-contained, native-Linux app that turns Diablo II LoD 1.14d into an idle game": the user sets goals, the bot plays; "the idle-game layer is built by making the bot optimal first and then degrading it on purpose" (PLAN.md line 73). ADR 0009: v1 is the d2d container, Game.exe is frozen and retires at the container's 24 h soak; d2d only has to match the original on the four gated metrics within 20 %. So: **an idle-game layer on a faithful bot, on d2d.** d2d-as-oracle (upstream PRs, traces) is a means to the fidelity gate and a goodwill lane, not a product.

## 3. Milestones

Sizes are agent-hours, rough (U). Exit criteria cite the spec requirement that owns the number; the closing card of each milestone carries the exit as its `gates:` (spec-sme: no process spec in openspec/specs).

| id | goal (one sentence) | exit criterion (measurable) | evidence | size |
|---|---|---|---|---|
| M1 basics | Town and inventory basics are rock solid on d2d (BACKLOG section 0, user) | docs/basics-audit.md coverage table: 0 G rows; every S1/S2 row C with an offline test that asserts the SME end state (no `PinsCurrent`) and a live d2d proof line; the 2 unknowns measured V; capability `town-basics` holds the rules | docs/basics-audit.md, `go test -p 1 ./internal/town ./internal/bot -run Audit`, card m1-liveproof | 9 h, 5 cards |
| M2 early gate | The bot on d2d matches the original's early game within the fidelity gate, and we can measure a 10 % change | `d2dsoak compare -gate v2` against baselines-early (extended, fidelity-gate `Early baseline fields`) on >= 8 fresh seeds x 3 virtual h: PASS on every gated metric, exit 0 (`Verdict exit codes`), 0 excluded seeds; A/B protocol in force | card d2d-replicated-soak (= m2-gaterun) result, docs/probes | 14 h, 6 cards |
| M3 Act 1 clear | A fresh Sorceress clears Act 1 unattended on d2d: Den, free Rogue, Andariel, Warriv | act1-campaign `Unattended Act 1 clear`: 3 of 3 seeds, no cheats, Act 2 entered within 1.3 x 176,132 / (M2 XP/h) virtual hours (35 h at 6600), no death-rule FAIL code, 0 panics, 0 interventions, merc alive >= 70 % after the claim | docs/act1-run.md, card m3-act1run | 22 h, 5 cards |
| M4 Act 2 | The Act 2 chain runs on d2d through Duriel with the Desert merc | 1 seed: Duriel dead within 24 virtual hours after the Act 2 hop (U); G2 floors on the way (soak-gates `G2 meaning and floors`: clvl 16 at 8 h) | docs/act2-run.md | 25 h |
| M5 ship | The d2d container is the product: default backend, 24 h soak, Game.exe retired | licence decision recorded (ADR); cards d2d-default-backend and d2d-container-soak PASS (soak-gates `Verdict`: report.txt PASS, exit 0) | the two cards' results | 15 h + user |
| M6 idle layer | The idle-game layer the user parked | defined by a grill after M5; out of scope until then | ADR | ? |

Dependency graph (text): `xpgap, basicsfix (merging) -> M1 -> M2 -> M3 -> M4 -> M5 -> M6`; `m2-soakexit, m2-earlybase, m2-abdesign, m1-loot` have no upstream dependency (start now); `m3-andariel` depends only on a free slot and runs in parallel with M1 and M2 (d2-sme); grill 0003 gates `m3-merc`, `m3-routeguard`; the licence review gates M5 only.

Out of scope until M3 is done (DECIDED D2, 2026-10-31): d2dmesh and stream-grid are PARKED (cards set `deferred`, OWNERS lines removed, text kept; the livestream keeps running as is), container work beyond the licence decision, new upstream PR candidates (replies to #134 to #136 and the Lut Gholein post stay: one PR lane), Game.exe anything, Nightmare and Hell, the idle layer.

## 4. Work breakdown: the cards

Launch order; one agent, <= 2 h each. `proposed` = launchable now (OWNERS lines present); `deferred` = queued behind the dependency named in its proposal (its `owns:` and `exclusive:` are intended, not claimed; the dispatcher sets `proposed`, adds the OWNERS lines and re-runs changecheck when the dependency merges).

| # | card | status | after | what |
|---|---|---|---|---|
| 1 | m2-soakexit | proposed | - | d2dsoak exit codes and excluded seeds (review H1) |
| 2 | m2-earlybase | proposed | - | kills/h and deaths per 1000 kills for the early baseline from the Game.exe logs |
| 3 | m2-abdesign | proposed | - | paired same-seed A/B protocol and tool; re-analysis of xpgap's 24 runs |
| 4 | m1-loot | proposed | - | charms and runes kept, chipped and flawed gems sold before the Cube (D4, audit G7) |
| 5 | m3-andariel | deferred | soak-slot:3 free | the Andariel fight on a cheat-levelled throwaway (the biggest unknown; parallel to M1, M2) |
| 6 | m1-audittests | deferred | xpgap, basicsfix | flip the pinned audit tests; spec backfill for basicsfix (ID scrolls, purchase order, bank first, repair latch); follow-up of the merged scroll rule (D3) |
| 7 | m1-stashcap | deferred | basicsfix | stash gold cap (E4) |
| 8 | m1-trips | deferred | xpgap, basicsfix | second sell door, gold trip above the keep line, pack-full in free cells |
| 9 | m2-tripwp | deferred | basicsfix, m2-abdesign | leave town by waypoint, portal return, chicken by portal (14 % + 21 % of the loss) |
| 10 | m2-roam | deferred | m1-trips | empty-area exit on, no border ping-pong (walk 77 % of kill time) |
| 11 | m1-liveproof | deferred | m1-audittests, m1-stashcap, m1-trips | the 10 live checks and 2 unknowns; closes the coverage table (M1 exit) |
| 11a | m2-latebase | deferred | m1-liveproof | late baseline for XP/h and kills/h from a cheat-levelled clvl 12 throwaway with the M1 kit (D5) |
| 12 | d2d-replicated-soak (= m2-gaterun) | proposed (exists) | m2-roam, m2-tripwp, m2-latebase | the closing run of M2: 8 seeds x 3 h, PASS, exit 0 |
| 13 | m3-merc | deferred | grill 0003 (waiting, prebrief section 10) | free Rogue claimed, kept alive, revived on d2d for hours; merc on by default on d2d |
| 14 | m3-routeguard | deferred, WAITING ON GRILL 0003 (no delta until then) | grill 0003, xpgap | Monastery, Forgotten Tower and Andariel floors set to the decided numbers; MODIFIED deltas written at launch |
| 15 | m3-act1chain | deferred | m3-merc, m3-routeguard | Act 1 quest order and quest evidence on d2d up to the Andariel approach |
| 16 | m3-act1run | deferred | m3-andariel, m3-act1chain, M2 | the closing run of M3: 3 fresh seeds (M3 exit) |

## 5. Operating rhythm (proposal; text in docs/process/dispatcher-brief-proposed.md)

- **WIP limit 3 implementation agents** (2 on the current milestone's critical path, 1 side lane: upstream replies, stream, process), plus read-only agents (SME, observer, reviewer) as needed. Why 3: the merge checklist is the throughput limit (one coordinator, whole-repo test per merge); 4 d2d slots give each live agent one slot and keep one for the livestream soak.
- **Every job maps to a milestone card or is a bug fix with a finding** (a log line, a failing test or a soak anomaly cited in the proposal, and a scenario test that asserts the end state). No card, no agent.
- **Tick** (every 2 hours or at a merge): read the current milestone's exit criterion and its number, pick the next card in the launch order, promote deferred cards whose dependency merged, janitor duty (process-hygiene spec), `changecheck status -write`, nothing else.
- **Weekly review** (the user, 20 minutes): the milestone table with the current number per exit criterion; cards archived per milestone; the decisions list; what to stop.
- **Stop doing:** research docs without a card gate or PR consequence; knobs that default off "for a later check"; "needs a G1/G2 check" notes (Game.exe is frozen: verify on d2d or say U); a second fix of the same symptom without a scenario test and a spec line; more than one new upstream PR in flight; proposed cards that hold a slot claim while blocked (d2d-container-soak holds `soak-slot:1` behind the licence review: set it deferred).
- **Upstream cadence:** one lane, one PR in flight at a time, replies within a day, new candidates only from a gap a milestone run found.

## 6. Alignment with the SMEs (2026-10-31)

| point | d2-sme / spec-sme | ROADMAP | resolution |
|---|---|---|---|
| Milestone order M1 -> M2 -> M3 | d2-sme: sensible; DISAGREE mildly with serializing the Andariel fight behind M2 | agreed | m3-andariel decoupled: only a free slot, runs in parallel (card 5) |
| Route guard 20 / Andariel floor | d2-sme DISAGREE with 20 (537k XP; Act 1 is under-level from clvl 17 to 18): Monastery 18 no merc (14 with), tower 16 (12 with), Andariel 15 with the merc (+2 dead merc) | plan-v2 says 20 | UNRESOLVED: D6, the user runs grill 0003 (prebrief section 10 lists the numbers and questions); card m3-routeguard waits, no delta until then |
| 24 virtual hour cap for M3 | d2-sme: not realistic at today's rate (clvl 15 = 176,132 XP; 35 h at 5k/h, 18 h at 10k/h); derive it | fixed 24 h | changed: cap = 1.3 x 176,132 / M2 XP/h (act1-campaign `Unattended Act 1 clear`, U) |
| Gems on a sell visit | d2-sme: sell chipped and flawed gems before the Cube; the user's checklist row 6 says gem kept | followed the checklist | DECIDED D4: sell chipped and flawed gems before the Cube, keep better gems; m1-loot implements it |
| Cheat-levelled clvl 12 for the late baseline | d2-sme: fair for XP/h and kills/h with XP set to the table threshold, points spent by the leveler, the baseline kit; not for deaths or gold | wanted a decision | DECIDED D5: yes, ledgered, card m2-latebase |
| A `milestones` capability spec | spec-sme DISAGREE: process state, not behaviour; MODIFIED on every re-plan | wanted one | changed: exit criteria live here and in the closing cards' gates; behaviour in `town-basics`, `act1-campaign`, deltas on fidelity-gate, early-game-policy, merc-policy |
| `deferred` as the queue status | spec-sme AGREE, with a README line (proposed in the report; README is owned by specfix) | same | agreed; convention stated in section 4 |
| Purpose in a new capability's delta | spec-sme: required, else archive writes a TBD Purpose that changecheck flags | same | every new-capability delta carries title and Purpose |
| m3-routeguard without a delta at propose time | spec-sme: deltas belong at propose time; MODIFIED needs the final numbers | numbers undecided | skip_specs true with a blocking task: the deltas are written at launch from the decision; flagged |

## 7. Risks

1. The M3 cap is derived from M2's XP/h: if M2 stalls at 6600 the first Act 1 run costs 35 virtual hours per seed (about 1.5 wall hours at lockstep speed), fine for compute but slow to iterate.
2. The route guard and the Andariel floor are in dispute (SME 18/14/15 vs plan 20); M3's policy cards cannot launch before grill 0003.
3. basicsfix's scope is only known from the coordinator's note; m1-audittests re-reads the merged tree and strikes closed gaps in the launch brief.
4. The early baseline stays weak (2 runs) until 5 Game.exe runs exist, which cannot happen (frozen); the late baseline is built by m2-latebase (D5 decided yes).
5. Deferred cards can silently overlap a running one: the dispatcher must run changecheck right after the flip.

## 8. Decisions (the user, 2026-10-31)

1. **D1 DECIDED (2026-10-31): ADOPTED.** Milestones M1 to M6 and the WIP limit of 3 implementation agents (read-only agents unlimited); every job maps to a milestone card or is a bug fix with a finding.
2. **D2 DECIDED (2026-10-31): PARKED.** d2dmesh and stream-grid wait until M3: both cards are `deferred`, their docs/OWNERS lines removed, their text kept. The livestream keeps running as is.
3. **D3 DECIDED (2026-10-31): purchase order at gold 100 to 190 is the SME order**: healing potions first up to 3, then a Town Portal scroll only when gold >= scroll price plus 2 potions. basicsfix merged "scroll at clvl >= 4 with >= 100 gold left" (`town.tpFirstLevel` 4): changing that rule is work of card m1-audittests (code, not in this change).
4. **D4 DECIDED (2026-10-31): gems.** Sell chipped and flawed gems before the Horadric Cube recipe is available; keep better gems (and every rune). Card m1-loot.
5. **D5 (late baseline) DECIDED (2026-10-31): YES.** XP/h and kills/h late baseline from a cheat-levelled clvl 12 throwaway with the M1 kit on hidden d2d, every cheat recorded in the ledger (internal/cheat, cmd/d2cheat, docs/cheats.md); card m2-latebase. Not used for deaths or gold.
6. **D6 (grill 0003) DECIDED (2026-10-31): run the grill now**, on the route guard numbers (SME 18/14 vs plan 20), the Andariel floor and the revive budget. The user starts it; docs/grills/0003-merc-and-act2/prebrief.md section 10 holds the numbers and questions. m3-routeguard and m3-merc wait.
   *Settled by ADR 0011 (grill 0003, accepted 2026-10-31): readiness is power against danger, interim floors road/tower 16 (14 with a merc), Andariel 17 (15 with a merc); cards m3-routeguard and m3-merc proposed, m3-readiness-model deferred.*
