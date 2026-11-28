# Fable review 6 of 8: the Proving Ground as a safety net and as evidence

> status: review, 2026-11-28, read on main 415b11f25. Docs only: no code, no lab run, no game. Tags: V = read in the cited file or produced by a command on that commit (counts are a snapshot of that commit, made with a throwaway script over `tools/provingground/scenarios/*.json` and `data/darklaunch.json`); S = secondary (a doc says so); U = unverified. Severity: blocker, high, medium, low.

## 0. Verdict

The harness failed CLOSED on 2026-11-27 (BLOCKED, nothing went green that should not have), which is the one thing it did right. Everything around it failed open: nothing ran the scenarios, the refusals were printed to a stream nobody reads, and the page that is meant to show verdicts says "(no runs recorded)". As a source of evidence for flips it is weaker than the flip texts assume: 254 of 266 scenarios are CHEATED setups, 207 play a hero with every point unspent, 60 assert only on the bot's own log lines, more than half count "writes after start" from the bot's own journal, and the flip tool checks none of it. A GREEN today proves "the code path runs when the hero is put on rails in the fork's game", not "the feature works for a played hero".

## 1. Why the break was not caught, and the minimum net

### 1.1 The chain (all V unless tagged)

| # | link | where |
|---|---|---|
| 1 | necroteeth changed a harness-wide default (`played`) for every scenario that did not name a profile. No scenario file changed, so the diff showed 9 Go and doc files and not the 207 scenarios whose setup changed. | commit 2d79c46b6; `internal/scenario/scenario.go:496-501` (now banked again) |
| 2 | The card closed Done with result "Not verified live" and criterion 5 (the live row) unticked. | `backlog/tasks/task-1033 - necroteeth-...md` |
| 3 | The live `attack` row needs the skills on the hands. `prepSelfCheck` never filled `Observed.LeftSkill/RightSkill` (it read level, gold, potions and the point dump only), so `bound` was false for EVERY played hero above level 1. The unit tests passed because they build `hero.Observed` by hand with the hands set. | `cmd/d2scenario/validate.go:185-219`, `internal/scenario/hero/selfcheck.go:55-61`; fixed by f48dc5943 |
| 4 | `prepSelfCheck` and `execOne` have no test at all (`grep -rn "prepSelfCheck\|execOne(" --include=*_test.go` is empty). The prep path only ever runs against a real lab d2d. | `cmd/d2scenario/run.go:275-413` |
| 5 | `validate` (T0, it does run at merge through `TestCoverageAndValidateOnShippedData`) uses `hero.NameTables()` by default: every skill name resolves, no learn rule, and its own comment says "a pass here can still fail live; U". The static half checks that a main attack NAME exists, never that the prep can bind it. | `cmd/d2scenario/validate.go:29`, `internal/scenario/hero/selfcheck.go:117-134` |
| 6 | A run blocked before the setup object existed carried no tags; `NewLedgerRecord` refuses a line without a setup tag; `shipLedger` prints the refusal to stderr, writes nothing to the spool and leaves the exit code alone. | `internal/scenario/ledger.go:123-133`, `cmd/d2scenario/ledger.go:29-34` |
| 7 | Nothing runs scenarios on a schedule. The nightly script exists; its timer does not (the ops unit directory holds a ledger-pull timer and the nightly race timer, no Proving Ground job; the ledger-pull timer is not active on the head box, U whether the board host is another machine). The after-push T2 stage is "ops-side", unbuilt. `mgonejob` (TASK-745) is Proposed. | `docs/process/pg-ops-pgnightly.md` ("no night has run"); TASK-754 is Done with criterion 1 "Nightly job runs ..." ticked |
| 8 | BLOCKED is not evidence for the stability rule, so GREEN turning BLOCKED never flips, quarantines or alarms. | `internal/scenario/stability.go:89-91` |
| 9 | The generated status page is empty. Its filter is branch `main`; every hand run is stamped `agent/<worktree>` by `pgenv.sh`. | `docs/process/proving-ground-status.md`; `tools/provingground/pgenv.sh:5-6`; `cmd/d2scenario/ledger.go:75` |

The incident data itself (V, the eight pulled run directories of 2026-11-27 21:57 EST in the coordinator's `provingground-out/`): 24 results, 21 BLOCKED `selfcheck: attack`, 2 BLOCKED `setup: debug give cap pack: d2d: no room` (a second, separate defect nobody has named), 1 ledger line. The blocked ids were the flip-gate drills G1 and G2 of `docs/probes/dark-queue-final-2026-11-27.md` (`board.sm.*`, `town.plan.*`) and `town.resupply`, a T2 smoke scenario. The row detail is in a slot directory on the worker host, not in the result.

What pgfix left open (V): (a) `run.go:238-242` adds tags only when `BuildSetup` succeeds, and a BLOCKED run with no d2d binary (`d2d_sha256` empty) or host role unknown is still refused, so exactly the infrastructure failures stay out of the ledger; (b) card TASK-1093 is merged with status Proposed, description TODO and no `result:` line, so there is no record of a live rerun (U); (c) 59 scenarios are `played` and now depend on `player.lskill/rskill` being in the point dump, verified by a doc citation only (`selfcheck.go:162`), not by a green run (U); (d) `validate.go:152` still carries the comment of the reverted default; (e) the default is now the hero design 6.2 said a Proving Ground hero must not be.

### 1.2 The minimum net: `d2scenario smoke` (no game, no slot, under a minute)

Blocking from day one: it has no game, so it has no flake source. Ship it as a Go test in `cmd/d2scenario` (so the scoped gate picks it up by reverse dependency) AND as a subcommand that prints one positive marker line the merge worker requires: `smoke: static N scenarios, prep 4, ledger 4, blocked 0`.

Stage A, static, every scenario and class (extends `staticCheck`, `cmd/d2scenario/validate.go:131-162`):

| check | rejects | exists today |
|---|---|---|
| A1 profile is explicit | a scenario with `pre.level` above 1 and no `pre.hero_profile`. No default at all: the migration writes `banked` into 207 files, which is the visible diff necroteeth lacked | no (`Scenario.HeroProfile`, `scenario.go:496-501` defaults) |
| A2 setup golden | `setup-golden.tsv`: scenario, class, sha256 of (verbs, profile, tags, knobs) from `BuildSetup` plus the hero library verbs. A change that moves rows it did not regenerate fails; a regenerated file shows the blast radius in review | no |
| A3 played hero is checkable | `hero.Resolve` gives an attack with a skill index for that class, build and level using REAL learn rules (a small checked-in table of class, skill, required level, class index, cut from the game tables with their hash; U whether the table may be checked in, else a fixture cut from one lab dump per build) instead of `NameTables`. A class with no main attack at that level must be declared `banked` (the unit test exempts the Paladin by hand, `herospend_test.go:31`; whether a played Paladin passes the live row is U) | partly: `hero.Static` (`selfcheck.go:95-115`) with name tables |
| A4 every exit is recordable | for each scenario and class, a BLOCKED result built from `BuildSetup` tags and a fixed provenance passes `LedgerRecord.Validate` | no (one test for two levels, `herospend_test.go:73`) |
| A5 tier data agrees | scenario `tier` (fast, full, nightly) and `tiers.json` (T2, T3, T4) do not contradict; every T2 id is in the stage B candidates | no: two vocabularies, 35 `fast` against 7 T2 |

Stage B, prep on a fake d2d, 4 scenarios: run the REAL prep half of `execOne` (caps, `applySetup`, `spendPoints`, `prepSelfCheck`, tags, ledger record) against `internal/d2dctl/fake` (it already serves a unix socket with fixtures, `fake.go:101`; `cmd/d2scenario/seed_test.go:25` has a second `fakeGame`). Needed seam: `execOne` takes the game process as a five-method interface (start, wait in game, controller, save and exit, stop) instead of `d2dinput.Process`. The fake keeps a small point model (`debug stat`, `cmd stat|skillpt|select`) and answers `dump player,stats,skills`, `state` and `items` from RECORDED replies of the lab d2d (file names carry the d2d hash), never hand-written JSON: a hand-written fake repeats link 3.

| scenario, class, seed | path it covers |
|---|---|
| `town.resupply` Paladin 11 | legacy banked, town start, T2; it was BLOCKED in the incident |
| `points.bank-by-default` Sorceress 3 | banked points scenario, raw point expectations |
| `combat.point-spell-hits` Sorceress 11 | played: spend, bind, the `attack` row |
| `cain.scroll.field` Paladin 11 | field start, journal write count, board record |

Then for each: `eval` on a recorded GREEN state must be GREEN and on the same state with the asserted fact removed must be RED (a must-fail arm, so an assertion that cannot fail is caught too). Expected for all four: prep not BLOCKED, one valid ledger line each.

Gate wiring (ops, part of TASK-745): run when the diff touches `cmd/d2scenario/`, `internal/scenario/`, `tools/provingground/`, and the prep's dependencies `internal/d2dctl/`, `internal/d2dinput/`, `internal/leveler/`, `internal/plandata/`. Note two holes in the scoped gate (V `tools/gate_scope.py:18` in the ops repo): `.sh` and `.py` are ignored, so `run.sh`, `pgenv.sh`, `nightly.sh` and `featuresoak.py` change untested, and `tools/ci/check.sh` does not call `nightcheck_test.sh` or `pgenv_test.sh`. Add both tests to `check.sh`.

Smoke does not replace a real run: it cannot see the fork changing a reply. So the second half of the minimum is the after-push canary ADR 0027 item 2 already decided: the 7 T2 scenarios, first class and seed, on the builder long lane, report-only for verdicts but ALARMING (board, push) on any BLOCKED and on "results N, ledger lines M, N != M".

## 2. Soundness as evidence

| # | sev | finding | evidence |
|---|---|---|---|
| E1 | high | Almost all evidence is CHEATED. 254 of 266 scenarios say CHEATED in their provenance; 140 start in the field by warp; no scenario uses `pre.from_save`, and `BuildSetup` lists it as Unsupported. The EARNED tier is a format, not a capability. | V script; `proving-ground.md` section 11 last bullet |
| E2 | high | Banked heroes. 207 scenarios take the default and play a hero raised by stat verbs with no skill learned (the process doc's own lesson: "55 life at clvl 32, no attack skill"). Any combat, survival or pace reading from them is about a hero no player has. pgfix restored this as the default. | V script; `proving-ground.md` section 13 |
| E3 | high | 42 percent of assertions are `log_line` (373 of 881) and 60 scenarios assert on nothing but log lines and `no_writes`, among them the T4 `act1.andariel.slaughter`. Section 1 of the process doc says the verdict is "never from log lines". A log-only GREEN proves the bot SAID it did it. | V script; `proving-ground.md:7` |
| E4 | high | "Writes after start" is self-reported on field starts. The lab d2d keeps its verbs on and the count is read from the bot's own command journal; 69 of the 140 field scenarios do not assert it at all. Only town starts have an independent count (a d2d without verbs that refuses). | `cmd/d2scenario/run.go:738-778`, `fieldledger.go:26-49` |
| E5 | high | A hang is not a failure. A run that never prints the run-length line is BLOCKED "not judged"; cut and BLOCKED runs are not counted. A bot that deadlocks the lockstep clock, the worst outcome, lands in the same bucket as a slow host and can never turn a scenario red or quarantine it. | `cmd/d2scenario/prov.go:19-25`, `run.go:386-388`, `stability.go:89-91` |
| E6 | high | Nothing can become stable, so by ADR 0027 nothing may gate. Stable needs 5 equal runs on ONE bot commit and d2d hash, 2 of them in parallel; any commit change resets the streak. With many merges a day and no nightly, no key gets there. Every ledger line also hard-codes `stability: candidate`. | `stability.go:20-26,147-150`, `ledger.go:97` |
| E7 | medium | Three flake rules that contradict each other: the process doc ("a flaky scenario is a failed scenario; do not retry until green"), the stability rule (any same-binary flip quarantines), and the flip criteria ("green 2 of 3 runs"). A 2-of-3 pass IS a same-binary flip: the flip rule accepts exactly what the gate rule quarantines. A scenario that passes half the time clears 2 of 3 in half the attempts, and attempts are not limited or recorded. | `proving-ground.md:62`; `data/darklaunch.json` row `quests.stepExec` |
| E8 | medium | Seeds and classes are thin and fixed. 111 scenarios run seed 11 only, 129 run 11 and 3; two scenarios use 11, 12, 13. 119 have one class. Class appearances: Paladin 198, Sorceress 148, Necromancer 60, Barbarian 3, Druid 3, Assassin 1, Amazon 0. The seeds a fix is developed against are the seeds it is judged on: there is no held-out seed. | V script |
| E9 | medium | Lockstep. The game waits for the bot, so reaction-time behaviour (dodge, chicken, potion timing) is judged in a world with no latency; the sandbox does not wait (S). The fork is also not game.exe: parity is ADR 0023's question and a GREEN says nothing about it. | `run.go:309,397`; design docs |
| E10 | medium | Wall limits against the lane. Default wall is virtual/5*1.6+10. `necro.earned-act2` (the only fresh-hero act run, the one the release rule needs) comes to 500 wall minutes against a 150-minute lane limit and a 240-minute nightly e2e cap; `necro.level-1-30` to 240. Cut means BLOCKED means "not judged" (E5), so the release rule of ADR 0027 item 8 has no run that can satisfy it at the slow end (U at the typical pace). | V script; `scenario.go:341-342`; `nightly.json`; `docs/probes/lab-night-commands-2026-11-27.md:13` |
| E11 | medium | Ledger completeness: refused lines vanish (1.1 link 6), there is no reconciliation of results against lines, the main view is empty (link 9), the ledger has no `knobs` field so a GREEN cannot be tied to the knob value it ran with, and `log_line` windows map wall time linearly onto virtual time (9 assertions, the doc tags it U). | `ledger.go:29-66` |
| E12 | low | Good and worth keeping: unknown JSON fields refused, BLOCKED never passes, UNOBSERVED is not a pass, `.control` twins exist for four scenarios (three combat, one step), the EARNED-with-writes refusal, the determinism pair. | V |

What a GREEN proves: on that d2d build, that class and that seed, from that declared (usually impossible-by-play) state, the bot's code reached the asserted fact inside the virtual budget. What it does not prove: that a played hero gets to that state, that the hero survives getting there, that it holds on another layout or class, that it holds in real time, that no write helped (field starts), or, for log-only scenarios, that anything happened in the game.

## 3. The rules that turn results into flips

| # | sev | finding | evidence |
|---|---|---|---|
| F1 | high | The flip tool checks no evidence. `flip.py` needs a `dark` or `trial` row and a `--why` string; `criteria_met` is typed by hand. Of the 10 `enabled` rows: 7 `U`, 2 `no`, 1 `yes`. | `tools/flip/flip.py:379-433`; `data/darklaunch.json` |
| F2 | high | The enable step is "no hard invariant counter rose on one sandbox hero in the window" (2 h, in practice 30 and 45 minutes for several rows). That is absence of a breach, not presence of the feature. `quests.stepExec` (four ported steps, two in Act 2) was enabled on a level 12 Act 1 hero; the flip doc says so itself ("U: Questy is not in Act 2 now"). `agency.planner` 1: "no recorded run; mode 1 changes no decision, so the 2 h watch is the check": a watch cannot fail for a knob that changes no decision. Six rows: "no drill (U), the clean window is the evidence". | `docs/probes/dark-queue-final-2026-11-27.md` section 1; `docs/process/dark-launch.md:37-45` |
| F3 | high | Circular in three places. (a) The scenario that gates a knob is written by the card that wrote the feature, often asserts the feature's own log line (E3) from a CHEATED start beside the objective: it proves the path executes on rails. (b) The watch reads the invariants from the bot under test (`ops status --why`); only "ten unreadable polls" is independent. (c) There is no knob-off arm: nothing shows the scenario is RED without the knob, so a GREEN may be the old code path. | as cited; `.control` twins exist for four scenarios only |
| F4 | medium | Criteria are advisory. "The ledger text asks for all four scenarios green 2 of 3: the fourth is not, so the flip is a knowing departure." 9 of the 14 enabled or trial rows still say "24 h" while the rule in force is 2 h. The written criterion and the practice disagree, so `criteria_met` means nothing. | same doc, row `quests.stepExec`; `data/darklaunch.json` |
| F5 | medium | Flips cite prose ("soak doc 3.3", "drill doc"), not run ids, against ADR 0027 item 10; with the status page empty no flip can cite a ledger row on main. | dark-queue-final section 1 |
| F6 | low | Same-day reversal: the flip doc lists `bot.farmReset` under RETIRE ("never fired" in 45 min, modelled gain 2.5 percent "not clear"); hours later it is always-on with `criteria_met: yes` on 70 minutes of two heroes. The criterion was "beats baseline XP/min". Not a Proving Ground defect, but it shows how elastic the rule is. | dark-queue-final section 5; ledger row note |

Not circular, to be fair: the incident drills (G1, G2) came back BLOCKED and, as far as the ledger rows show, no flip was made on them (U for anything done by hand since).

## 4. Fix list, in priority order

| # | fix | effort | closes |
|---|---|---|---|
| 1 | Record every run. Tags computed at the top of `execOne`; a BLOCKED line is accepted with `d2d_sha256: missing`; a refusal goes to `ledger-refused.jsonl`; `run` prints "results N, ledger M" and exits non-zero when they differ. | S | link 6, E11 |
| 2 | `pre.hero_profile` mandatory above level 1, no default; plus the setup golden file (A1, A2). | S | links 1, 3; E2 becomes visible per scenario |
| 3 | `d2scenario smoke` stages A and B as test and subcommand; the merge worker requires its marker line; the two shell tests into `tools/ci/check.sh`. | M | links 4, 5 |
| 4 | Ops: install the nightly timer and the after-push T2 canary with an alarm on any BLOCKED or unrecorded run; reopen criterion 1 of TASK-754 until a night has run. | M | link 7 |
| 5 | Stability: GREEN to BLOCKED on new binaries is a state (`broken`) that alarms; a cut with a stalled virtual clock is RED, a cut with a moving clock is BLOCKED; an equal verdict and D1 across a bot commit continues the streak. | M | link 8, E5, E6 |
| 6 | `flip.py --evidence <run id,...>`: each id must resolve in the ledger to GREEN on an ancestor commit with the knob at the flip value (new ledger field `knobs`), at least 2 classes and one held-out seed; `criteria_met` is then derived, never typed. | M | F1, F4, F5 |
| 7 | Must-fail control arm: a flip-gating scenario runs with the knob off and must be RED or differ on the asserted fact; and the watch needs an exercise counter above zero for the knob (else the window ends `inconclusive`, not `enabled`). | M | F2, F3 |
| 8 | Field starts: write count from the game side (a fork read verb, or verbs dropped at hand-over), not the bot's journal; until then tag the fact S and keep field runs out of gate and release evidence. | M (fork) | E4 |
| 9 | Mark log-only scenarios `evidence: log`; they may not gate, flip or release until they carry one state assertion. Convert the T4 and flip-gating ones first. | S rule, L conversion | E3 |
| 10 | Matrix floor for evidence: 2 classes, 3 seeds, one seed derived from the date and recorded; one flake rule (drop "2 of 3", cite the stability state); one tier vocabulary; fix the `necro.earned-act2` budget against the lane limit or split it into saved milestones. | M | E7, E8, E10, A5 |

Order rationale: 1 and 2 are a day of work and would each have made 2026-11-27 loud within one run; 3 and 4 make it loud before or right after the merge; 5 to 10 are about whether a GREEN means anything.

## 5. Not verified in this review

- The board-host ledger itself (not on this box): counts of real runs, flips and stable keys are U; E6 is argued from the code and the empty generated page.
- Whether `combat.point-spell-hits` and `cain.scroll.field` are the best stage B picks (chosen by path, not by run history).
- Whether any played scenario has run GREEN since f48dc5943.
- Pace on the night host for the two long T4 scenarios.
