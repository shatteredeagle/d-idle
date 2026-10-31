# Roadmap: milestones, work breakdown, operating rhythm (agent ROADMAP, 2026-10-31)

> status: DRAFT v1 for SME review (d2-sme, spec-sme). Proposal for the user; nothing here is decided until the user says so. Tags: V = checked in the repo or by a command named here, S = read in a doc, U = judgment. Numbers that have an owner (gates, knob defaults) are cited from their spec or docs/knobs.md, not copied.

## 1. Diagnosis: where the effort goes and why it feels like flailing

Counts: `git log --since=7.days --merges` in the old-history checkout (the project started 2026-10-28), merge subjects classified by branch-name prefix with a regex (approximate; `tools/` has no script for it, the classes are in this agent's report). 466 merges in three days (101, 218, 188 per day).

| theme | merges | share |
|---|---|---|
| bot behaviour (town, combat, nav, quests, merc, boss, classes) | 203 | 44 % |
| process, infra, CI, public split, docs | 96 | 21 % |
| d2d engine, harness, fork, determinism, replay, mesh | 51 | 11 % |
| soak and measurement tooling | 45 | 10 % |
| upstream PRs and game.exe traces | 26 | 6 % |
| container, fleet, wizard | 24 | 5 % |
| stream, viewer, dashboard | 21 | 4 % |

1. **The gated number is not converging, and nobody owns it.** XP/h on d2d (fresh Sorceress, 2 virtual hours): 3013 (D2DLOOP), 3830 (KILLFIX), 3869 (ECONOMY), 4985 (soakfull) against the early baseline 6600 (V: docs/d2d-soak-full-fix.md, docs/d2d-loop.md); kills/h 164 to 195 against 500 (late baseline, not comparable). Four diagnoses said the same thing (docs/research/per-kill-time-split.md, docs/d2d-soak-full-fix.md, docs/d2d-xp-gap.md on agent/xpgap, docs/basics-audit.md): walk is 77 % of the seconds per kill and the character sits in the poverty loop 54 % of the time. Each fix card moved the number 20 to 30 % and stopped; no card's gate was "the early fidelity gate passes".
2. **Seven streams, one merge queue.** Running or open at once on 2026-10-31: xpgap, basicsfix, specfix (five cards), soakwatch, d2dmesh, stream-grid, three container cards, two deferred Game.exe cards, three open upstream PRs and one drafted (V: STATUS generated block, `git branch`). Mesh, stream, container and upstream do not feed the user's stated milestone (BACKLOG section 0). One coordinator merges and archives all of it; the merge checklist (whole-repo test on the worker, archive, OWNERS, STATUS) is the real throughput limit, not the four d2d slots (a 2 virtual hour run takes 8 to 9 wall minutes, V soakfull).
3. **The same bug gets fixed four times.** Fresh-game livelock: runab 1f4b62f, livelockfix 5f864c2, the soakfull hotfix 9b33985, botsmall forgetGame (V: docs/d2d-soak-full-fix.md anomaly 1, git log). Poverty sell trips: economy, xpgap povertySellMin, basics-audit G2 "second door". TP-before-potions: shoptp, scrollfirst, xpgap tpMinGold, basics-audit G5 (still a policy question). Cause: fixes land as knobs default off or as one-line patches without a scenario test that states the wanted end state, so the next soak finds the same symptom.
4. **Unverifiable obligations pile up.** Every shared-code change since ADR 0009 carries "needs a G1/G2 check" (champ, botsmall, navstall, xpgap), but Game.exe is frozen and no G1 can run (V: ADR 0009 addendum). These notes are dead weight: a change to shared code is verified on d2d or not at all.
5. **Measurement debt sits in the gate path.** d2dsoak exits 0 on INCONCLUSIVE and counts empty seeds as clean exposure (review H1, S: docs/research/code-review-2026-10-30.md); the early baseline has one Game.exe run with XP/h and deaths/h only, no kills/h (V: tools/d2dsoak/baselines-early.json). A PASS cannot be claimed against it today.
6. **What was merged on 2026-10-31 without a milestone** (12 merges in the public log, V): golivepre, golivepre2 (user goal: public), doctrim, streamfix, hitlen, lutproof (upstream), botsmall, champ, smallfixes, navstall, basicsaudit. Two of twelve are on the gate path (navstall fixed the seed-11 spin; basicsaudit is M1 preparation).

The real bottleneck is bot policy on the early game plus the absence of a per-milestone exit test that each card must move. Engine fidelity is secondary (XP per monster and spawn placement equal game.exe, V docs/research/xp-award-trace.md, spawn-density-trace.md); compute is not the limit; process overhead is high but self-inflicted by the number of parallel streams.

## 2. Vision, as the repo states it

PLAN.md: "a self-contained, native-Linux app that turns Diablo II LoD 1.14d into an idle game": the user sets goals, the bot plays; "the idle-game layer is built by making the bot optimal first and then degrading it on purpose" (PLAN.md line 73). ADR 0009: v1 is the d2d container, Game.exe is frozen and retires at the container's 24 h soak; d2d only has to match the original on the four gated metrics within 20 %. So: **an idle-game layer on a faithful bot, on d2d.** d2d-as-oracle (upstream PRs, traces) is a means to the fidelity gate and a goodwill lane, not a product.

## 3. Milestones

Sizes are agent-hours, rough (U). Exit criteria cite the spec that owns the number; the `milestones` capability spec (openspec/changes/roadmap/specs/milestones/spec.md) states each as a requirement with scenarios so the dispatcher can tick it.

| id | goal (one sentence) | exit criterion (measurable) | evidence artifact | size |
|---|---|---|---|---|
| M1 basics | Town and inventory basics are rock solid on d2d (BACKLOG section 0, user) | docs/basics-audit.md coverage table: 0 G rows; every S1/S2 row C with an offline test that asserts the SME end state (no "pins current") and a live d2d proof line; the 2 unknowns measured | docs/basics-audit.md table, `go test -p 1 ./internal/town ./internal/bot -run Audit` | 10 h, 5 cards |
| M2 early gate | The bot on d2d matches the original's early game within the fidelity gate | `d2dsoak compare -gate v2` vs baselines-early (extended with kills/h) on >= 8 fresh seeds x 3 virtual h: PASS on every gated metric, exit 0, 0 hung seeds (fidelity-gate: 20 % tolerance, 5 seeds minimum) | docs/d2d-replicated-soak.md, docs/probes/ raw | 12 h, 5 cards |
| M3 Act 1 clear | A fresh Sorceress clears Act 1 unattended on d2d: Den, free Rogue, Andariel, Warriv | 3 of 3 seeds: Andariel dead and Act 2 entered within 24 virtual hours (U cap), death-rule PASS, 0 panics, 0 interventions, merc alive >= 70 % of Act 1 time after the claim (plan-v2 section 5) | docs/act1-run.md, docs/probes/act1/ | 20 h, 5 cards |
| M4 Act 2 | The Act 2 chain runs on d2d through Duriel with the Desert merc | 1 seed: Duriel dead within 24 virtual hours after the Act 2 hop; G2 floors (soak-gates: clvl 16 at 8 h) met on the way | docs/act2-run.md | 25 h |
| M5 ship | The d2d container is the product: default backend, 24 h soak, Game.exe retired | licence decision recorded (ADR); cards d2d-default-backend and d2d-container-soak PASS (soak-gates: report.txt PASS, exit 0) | the two cards' results | 15 h + user |
| M6 idle layer | The idle-game layer the user parked | defined by a grill after M5; out of scope until then | ADR | ? |

Dependency graph (text): `xpgap, basicsfix (running) -> M1 -> M2 -> M3 -> M4 -> M5 -> M6`. M2's measurement cards (m2-earlybase, m2-soakexit) and M1's loot card have no upstream dependency and start now. Grill 0003 (merc, route guard, Andariel floor) gates M3's policy cards. Licence review gates M5 only.

Out of scope until M3 is done: d2dmesh (park, nothing needs more than the 4-slot pool), stream-grid beyond what is already merged, container work beyond the licence decision, new upstream PR candidates (replies to #134 to #136 and the Lut Gholein post are fine: one PR lane), Game.exe anything, Nightmare and Hell, idle layer.

## 4. Work breakdown: the cards (openspec/changes/m1-*, m2-*, m3-*)

Launch order (one agent, <= 2 h each; status `proposed` = launchable now, `deferred` = queued behind the dependency named in its proposal; the dispatcher flips it to `proposed` and adds its OWNERS lines when the dependency merges):

1. m2-soakexit (now): d2dsoak exit codes and empty-seed exposure (review H1).
2. m2-earlybase (now): kills/h and deaths per 1000 kills for the early baseline from the existing Game.exe minute lines.
3. m1-loot (now): charms, gems and runes kept on a sell visit (audit G7) after the SME's charm rule.
4. m1-shop (after xpgap, basicsfix): ID scrolls gated on a pack item that needs identifying, bank before shop, TP-vs-potion order per the user's decision (G1, G5, G6).
5. m1-chores (after m1-shop): repair latch, stash gold cap (G4, G10).
6. m1-trips (after xpgap, basicsfix): the second sell-trip door, a gold trip trigger above the keep line, one pack-full knob (G2, G8, G9).
7. m1-liveproof (after m1-chores, m1-trips; one hidden d2d slot): the 10 live checks of docs/basics-audit.md, the 2 unknowns, coverage table closed.
8. m2-roam (after m1-trips): walk share of kill time: border ping-pong and empty-area roaming on the first fields.
9. m2-gaterun = the existing card d2d-replicated-soak (after m2-roam): the closing run of M2.
10. m3-merc (after grill 0003): the free Rogue claimed, kept alive and revived on d2d for hours.
11. m3-routeguard (after grill 0003): Monastery and tower guards and the Andariel floor set to the decided numbers.
12. m3-act1chain (after m3-merc): Act 1 quest chain on d2d: quest-state reads, Cain, Andariel approach.
13. m3-andariel (after m3-act1chain): the Andariel fight script on d2d (antidotes, poison, merc gate, kill detect) on a cheat-levelled throwaway.
14. m3-act1run (after m3-andariel): the closing run of M3, 3 seeds.

## 5. Operating rhythm (proposal; full text in docs/process/dispatcher-brief-proposed.md)

- **WIP limit 3 implementation agents** (2 on the current milestone's critical path, 1 on a side lane: upstream replies, stream, process), plus read-only agents (SME, observer) as needed. Why 3: the merge checklist is the throughput limit (one coordinator, whole-repo test per merge); 4 d2d slots give each live agent one slot and keep one for the livestream soak.
- **Every job maps to a milestone card or is a bug fix with a finding** (a log line, a failing test or a soak anomaly cited in the proposal). No card, no agent.
- **Tick** (every 2 hours or at a merge): read the milestone's exit criterion, pick the next card in the launch order, promote deferred cards whose dependency merged, run the janitor duty (process-hygiene spec), update STATUS by `changecheck status -write`, nothing else.
- **Weekly review** (the user, 20 minutes): milestone table with the current number per exit criterion; cards archived per milestone; the decisions list; what to stop.
- **Stop doing:** research docs without a card gate or PR consequence; knobs that default off "for a later G1/G2 check"; "needs a G1/G2 check" notes (Game.exe is frozen: verify on d2d or say U); second fixes of the same symptom without a scenario test; more than one new upstream PR in flight.
- **Upstream cadence:** one lane, one PR in flight at a time, reply to review comments within a day, new candidates only from a gap a milestone run found.

## 6. Alignment with the SMEs

(filled after the consultation)

## 7. Risks

1. M3's 24 virtual hour cap is a guess: at 5 to 10k XP/h, clvl 15 needs 176k XP (Arreat XP table, S), that is 18 to 35 virtual hours; M2 must land first or the cap moves.
2. The route guard (Monastery at clvl 20 without a merc) and the Andariel floor are still in dispute (SME 14 to 18 vs plan 20); M3 cannot start its policy cards until grill 0003.
3. basicsfix's scope is not visible on main yet; M1 cards name audit gap numbers so the launch brief can strike what basicsfix closed.
4. The early baseline is one Game.exe run; extending it with kills/h from the same logs keeps it weak (S quality). A clean PASS against the late baseline needs a clvl 11 to 14 character (decision 3).

## 8. Decisions needed from the user

1. Adopt this milestone order (M1 to M6) and the WIP limit of 3.
2. Park d2dmesh and stream-grid until M3 (or keep one side lane).
3. May the late-baseline fidelity run use a cheat-levelled clvl 12 Sorceress with the m1 kit (ledgered), or must the character get there honestly (after M3)?
4. Grill 0003 now: route guard clvl (20 vs 14 with a merc), Andariel floor, merc revive budget.
5. TP-first or potions-first at gold 100 to 190 (basics-audit G5).
