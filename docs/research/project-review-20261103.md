# Project review, 2026-11-03: goals, what was done, evaluation, rewritten goals and plan

> status: proposal for the owner (card `projreview`), docs only. Nothing here is decided until the owner approves appendix F. Read from the public repo, `git log` of main and the pending agent branches (`agent/xphshort`, `agent/gate3prep`, `agent/lockstepplan`, `agent/barbdiff`). No game, no worker host, no fork.

Tags: **V** = checked in this repo by a command or file named here; **D** = computed here from `git log` or committed minute archives (command given); **S** = session fact from the coordinator or another doc, not re-checked; **I** = my inference or judgement; **U** = unverified.

## 0. Bottom line (read this if nothing else)

1. **The project has two incompatible end goals on paper, and the gates measure neither of them directly.** ADR 0009 says the v1 product is the d2d container and "users never run Game.exe" (V, docs/adr/0009). The coordinator's framing of the owner's wish is "the bot works on game.exe" (S). Every gate now runs on d2d (ADR 0013, V), the container (the ADR 0009 product) has had no work in the last week (V: 9 commit subjects since 10-27 mention Andariel, Act 2 or the container at all), and the evidence that d2d results carry over to game.exe is INCONCLUSIVE (botab, V). **Decide which one is "done"** (appendix F, ADR 0015 draft). My recommendation: game.exe 1.14d is the acceptance target; d2d is the fast lab that gets us there; the d2d container is a later, optional distribution.
2. **The Gate 1 XP criterion cannot do the job it is being used for.** All gate runs are ONE seed (11) (V, every record), single-seed noise is 14 to 30 percent (V, baseline-audit cited in docs/soaks/gate1-20261102.md), five runs of the same seed spread 7060 to 10042 XP/h (sd 13.8 percent, V xphshort). A run at -23 percent is not evidence that the last fix cost XP, and xphshort shows it did not: the gap is where the bot farmed, not town time (D in that doc). Two spec deltas in one day (decisions.md 2026-11-03, V) bent the gate around single runs instead of fixing the sample. The fidelity-gate spec already says a bot-change claim needs 8 or more paired seeds (V). Gates must be multi-seed with holdouts (soak-gates "Holdout seeds" exists and was never used: V, no record lists seeds 16 to 23).
3. **We do not reliably know which class the gate character is.** Every Gate 1 record says "fresh Sorceress `Gate1b` (class 4, seed 11)" (V, docs/soaks/gate1-20261102.md and -20261103-0053.md); in our own code class 4 is the **Barbarian** (`internal/d2/player.go:22`, V); every committed gate minute archive ran plan table `fallback-lightning-v2`, the lightning Sorceress build (D, below). barbdiff (branch) asserts class 4 = Barbarian and spends its effort there. Either the record text is wrong (then barbdiff targets the wrong class), or a Barbarian played a Sorceress plan (then every gate number is a bug, and the AGENTS.md rule "never fall back silently to the Sorceress" was broken). One command on the worker host settles it (section E, step S1).
4. **Effort went mostly to d2d itself, not to the bot.** Of 435 merged agent branches from 2026-10-31 to 2026-11-03 (D), about 29 percent were rule coverage and verification, 23 percent upstream PRs and fork work, 28 percent bot behaviour and drills, 9 percent gate runs and judging, 2 percent lockstep, 10 percent process and infra. By lines added: `tools/emu` 39k and `docs/research` 25k against about 14k in all `internal/` bot packages (D). Bot milestones did not move: M1's own exit test is not met (13 `| C ` rows in docs/basics-audit.md, exit needs 15: V), M2 is not met (fidelity verdict INCONCLUSIVE), M3 (Act 1 clear, Andariel) has no live attempt (S).
5. **The plan documents are stale and too many.** STATUS.md's hand-written part is from 2026-10-30 (V, its header) and lists agents and open items that no longer exist; PLAN.md's status is from 2026-10-29 (V); 38 open cards (V, `ls openspec/changes`), 309 BACKLOG bullets (V), 17 specs, 14 ADRs in six days. The WIP limit of 3 (ADR 0012) is not what happened (435 merges in 4 days).

## A. Goals as they stand today

| # | goal (as written) | source | status today |
|---|---|---|---|
| A1 | Turn D2 LoD 1.14d into an idle game: the user sets goals ("reach level 30"), the bot plays offline single player, native Go | PLAN.md header (V) | Bot levels Act 1 on d2d for 2 h runs (V gate records); goal manager exists; idle layer PARKED (ADR 0012 M6) |
| A2 | v1 product is the d2d container; Game.exe frozen, acceptance reference, retires at the d2d container's 24 h soak | ADR 0009 (V), ADR 0001 container end goal | Container not built (S: cards d2d-container-* proposed/deferred, V STATUS block); licence blocker (ADR 0003/0010) |
| A3 | Only four gated metrics must match game.exe within 20 percent (XP/h, kills/h, deaths/h, deaths per 1000 kills) | ADR 0009 item 2, spec fidelity-gate (V) | NOT met and not measurable from existing records: every XP/h comparison INCONCLUSIVE, kills/min of d2d 0.73 to 0.78 of game.exe (botab, V) |
| A4 | Gate ladder on d2d: Gate 0 drills, Gate 1 2 h + fault, Gate 2 8 h, Gate 3 24 h | ADR 0013, spec soak-gates (V) | Gate 1 PASSED once (bot 050a509, V), then five re-runs NOT PASSED (V index + branches; 0053 re-judged PASSED under the 11-03 delta, V specdelta-rejudge); Gate 2 FAILED 2026-11-02 (S); Gate 3 not run |
| A5 | Milestones M1 basics, M2 early gate, M3 Act 1 clear, M4 Act 2, M5 ship, M6 idle; WIP limit 3 | ADR 0012, docs/roadmap.md (V) | M1 exit not met (13 of 15 C rows, V); M2 not met; M3 to M6 not started live |
| A6 | Early-game policy: Den first, no early vendor gear; free Rogue via Blood Raven; Act 2 Desert merc with Defiance; readiness = power vs danger | ADR 0004, 0006, 0011, specs early-game-policy, merc-policy (V); Defiance merc S | Implemented offline; merc claim/guard bugs found in Gate 2 and fixed offline (S) |
| A7 | Death rule by streak, recovery share and rate | ADR 0005, spec death-rule (V) | Implemented in d2dsoak (V spec); 0 to 1 deaths per gate run (S) |
| A8 | Build flexibility: builds and classes are data; no Sorceress fallback | decisions.md 2026-11-01 (V), AGENTS.md | Rule in force; finding 3 suggests it may be violated in the gate path (U) |
| A9 | Upstream contribution to d2d under the maintainer's rules (one subsystem per batch, one change per commit, one batch open), with game.exe differential proof | spec upstream-pr (V); maintainer's rules S | PR #177 (11 commits) merged (S); batch 2 ready, waiting for the maintainer's #181 (S) |
| A10 | d2d rule coverage against game.exe | docs/coverage/rule-coverage.md (V) | Open-ended coverage STOPPED 2026-11-02 in favour of "must be correct" rows (S); 4910 rows, 69 percent run-tested (S); Sorceress skills 15 deviations, 0 proven clean (V table) |
| A11 | Lockstep on game.exe (clock hook k=2), Tier B true stepping | tierb-design-20261102 (V), lockstep-plan-20261103 (branch, V) | k=2 works as a second regime (V there); Tier B approved but HELD; statistical A:B first (V there) |
| A12 | Memory writes allowed for setup and instrumentation, never a crutch | ADR 0014 (V) | In force; records carry `writes after start` (V spec) |
| A13 | Process: change cards, specs, changecheck, credit for contributors, hidden instances only | openspec/README.md, AGENTS.md (V); credit design S | In force; heavy (finding 5) |

Not a goal anywhere, but implied by the owner (S): "the bot plays game.exe unattended". No spec, ADR or milestone says this today; ADR 0009 says the opposite for v1.

## B. What we have done, by goal

| goal | done | evidence | verified vs claimed |
|---|---|---|---|
| A1/A4 bot loop | Full bot loop on d2d lockstep: navigation, combat, loot, town, merc, recovery, kit gate, stash withdraw | docs/soaks/*.md, gate1-index-20261103.md | V: 2 h unattended runs on d2d, 0 to 1 deaths (records). Never on game.exe since the freeze; never past Act 1 |
| A4 Gate 1 | 1 pass (050a509); 5 failed re-runs after Gate 2 fixes; 4 root causes fixed (kit trip pricing, walk-home, recovery switching potions off, no stash withdraw) | gate1-index, beltfix*-analysis, beltfix4/5, specdelta-rejudge | V the runs; fixes verified offline only (each merge says "not verified live") |
| A4 Gate 2 | One 8 h run, FAILED (merc claim/guard, kit trip timing, tome stat collision, judge artefacts) | gate3-prep (branch) cites the Gate 2 record | S (record not on main); fixes offline only |
| A3 fidelity | botab: existing records cannot answer; seeddist/seedspawn: Act 1 layouts, spawn lists and dice equal; k=2 pilot -2.9 percent XP/h inside noise | botab-20261102, seeddist-trace, seedspawn-trace, tierb-design section 6 | V as analysis; no verdict exists |
| A9 upstream | #177 merged (11 commits); batch 2 (mana floor, Teleport, monster damage roll) built and independently checked | openpr-log.md, pr-batch2-body.md | Merge S; batch 2 checks V in openpr-log |
| A10 coverage | 4910 rows, coverage tool, many deviations; Barbarian, Druid, Necro, Amazon skills mostly clean, Sorceress not | docs/coverage/rule-coverage.md | V table; note the table says Barbarian 30/30 clean while barbdiff claims six deviations (both V read, conflicting, barbdiff has no run) |
| A11 lockstep | clock hook P0 to P4, k=2 regime, Tier B design, lockstep plan | lockstep-p*.md, tierb-design, lockstep-plan (branch) | V runs in those docs; hook private |
| A13 process | specs, changecheck, leakcheck, credit, board, Discord feeds, public split | docs/process/* | V tools exist |

### Effort vs result, last four days (D)

Command: `git log --since=2026-10-27 --merges --format=%s | grep -o 'Merge agent/[a-z0-9-]*'` (435 branches, 10-31: 158, 11-01: 200, 11-02: 234, 11-03: 18 so far), each branch name sorted into one area by name pattern (approximate, I):

| area | merges | share | result that moved a goal |
|---|---:|---:|---|
| d2d rule coverage, verify*, sk*, traces | 124 | 29 % | coverage table; deviations found; no bot metric moved |
| bot behaviour and drills | 121 | 28 % | Gate 1 pass once; Gate 2 failure causes fixed offline |
| upstream PR, fork, CodeRabbit follow-ups, credit | 98 | 23 % | #177 merged; batch 2 ready |
| process, docs, board, Discord, archive | 45 | 10 % | none on a goal |
| gates, soaks, judging, baselines | 38 | 9 % | 6 Gate 1 records, 1 Gate 2 record, 2 spec deltas |
| lockstep | 9 | 2 % | k=2 regime proven, Tier B designed |

Lines added since 10-31 by path (`git log --since=2026-10-31 --no-merges --numstat`, D): docs/probes 128.7k (mostly run logs), tools/emu 39.3k, docs/research 25.4k, openspec/changes 20.1k, cmd/d2drill + internal/drill 15.1k, all other internal/* bot packages about 14k. Git cannot show hours; merges are the proxy.

## C. Evaluation

**C1. Is "d2d first" still right?** Partly. As a development lab it is clearly right: 6 to 30 times faster than real time, deterministic, hidden, scriptable (V, lockstep and d2d docs). As the **proof** that the bot works on game.exe it is weak today: the rules that drive the gated metrics are exactly where d2d is not yet game.exe (Sorceress skills 15 of 30 deviate, skill element damage 100 of 100 rows deviate, monster stats 734 of 734 rows marked deviation, V rule-coverage; I: the monster row count is one deviation class, the damage roll shape, applied to every row). And the seed-for-seed bridge is impossible (tierb-design, V). So "done on d2d" does not imply "done on game.exe"; only a game.exe run does. Keep d2d first for iteration, but add a game.exe acceptance rung at the end of the ladder and a statistical A:B in the middle.

**C2. Is the Gate 1/2/3 d2d ladder the right gate?** For stability (no panics, restarts, saves, fault recovery, invariants) yes: those are backend-independent enough and binary. For progress it is the wrong instrument as written: one seed, one character, XP/h compared against a single baseline number with a tolerance that is smaller than the run-to-run noise. Consequence seen this week: five re-runs, five root-cause hunts, two spec deltas, and the last "regression" turned out to be farming-area luck (xphshort, branch). The belt-window rule is a reasonable symptom check but it became the target; the right question is "does the bot buy and drink potions when it can afford them", which the delta now nearly says. Fix: judge stability per run, judge XP/h only on a seed set (5 seeds, 2 of them holdout), and judge bot changes by the paired A/B the fidelity spec already defines.

**C3. Is XP/h vs baseline the right progress metric?** It measures efficiency, not the owner's goal ("reach level 30", "farm Mephisto"). A bot at 80 percent of the baseline XP/h that clears Act 1 and Act 2 unattended is worth more than one at 110 percent stuck in Act 1. Replace it as the headline with **milestone time**: virtual hours from a fresh character to Andariel dead, to Act 2, to Duriel dead, per seed, with deaths. XP/h stays as a diagnostic.

**C4. Conflicts between goals.**
- Upstream contribution vs bot progress: 23 percent of merges. It pays where it fixes a rule the bot depends on (monster damage roll, mana, Teleport: yes); it does not where it fixes rules the bot never meets (other classes' skills, items the bot never uses). The maintainer's one-batch-open rule now caps throughput anyway, so extra upstream prep beyond the next batch is idle inventory.
- Coverage vs bot: open-ended coverage was correctly stopped on 11-02 (S); barbdiff continued on a premise (Barbarian gate character) that the records contradict (finding 3).
- A:B vs gates: both need the worker host; the lockstep plan schedules A:B after Gate 2, gate3prep schedules Gate 3 after the A:B pilot (both V, branches). That order is right; keep it.
- Product (d2d container) vs acceptance (game.exe): unresolved, see finding 1.

**C5. Dead or stale goals.** PLAN.md status (10-29) and its Wine-only architecture; STATUS.md hand-written part (10-30); ADR 0009's "retires at the d2d container's 24 h soak" (the container has no owner and no activity); fallback-lightning-v3 (approved pending a clvl 19-30 soak that no plan schedules); M6 idle layer (correctly parked); stream-grid and d2dmesh (parked); Tier B (held; correct).

**C6. Goals that cost more than they give.** Process surface (38 open cards, 17 specs, Discord/board/brand work: 10 percent of merges) for a single-owner project; upstream prep beyond the open batch; coverage rows for classes the bot does not play; reruns of Gate 1 on seed 11.

**C7. What would "done" look like?** A fresh character, never cheated, started by `d2bot` with a goal, plays unattended on **game.exe 1.14d** (hidden lane on the worker host, k=1 or the k=2 hook) for 24 h, clears Act 1 and reaches Act 2, with no manual intervention, saves intact, deaths within the death rule. Everything else (d2d gates, A:B, upstream fixes) is a means to that.

## D. Rewritten goals (6)

| id | goal | definition of done (owner can check) | metric | vs today |
|---|---|---|---|---|
| G-A | **Acceptance on game.exe**: the bot plays D2 LoD 1.14d (game.exe) unattended | one 24 h hidden game.exe run from a fresh, uncheated character: Gate 1 stability criteria hold over 24 h, Andariel dead, Act 2 entered; record in docs/soaks with `writes after start: 0` | pass/fail + virtual hours to Andariel | REWRITE (replaces ADR 0009's "users never run game.exe" and adds a final game.exe rung to ADR 0013's ladder) |
| G-B | **Act 1 clear on d2d** (old M3) | 3 of 3 seeds (one holdout), fresh uncheated character, Den, free Rogue, Andariel, Warriv to Act 2, unattended, inside the death rule | virtual hours to Andariel per seed; deaths per seed | KEEP (M3), promoted to the headline progress goal |
| G-C | **Stability on d2d** (gate ladder, judged correctly) | Gate 2 (8 h) PASS on 3 seeds incl. 1 holdout, then Gate 3 (24 h) PASS on 1 seed; per-run criteria = stability, invariants, belt-when-funded; XP/h is NOT a per-run pass/fail | count of seeds passing; INVARIANTS: PASS | REWRITE (soak-gates: XP/h moves to a seed-set verdict; holdouts required) |
| G-D | **Transfer is measured, not assumed** (fidelity) | statistical A:B fresh clvl 1-6 cell: 6 game.exe + 6 d2d lanes (lockstep-plan round 2), verdict EQUIVALENT or DIFFERENT at 20 percent on XP/h and kills/h; if DIFFERENT, the top cause named | 90 % interval of the d2d/game.exe ratio | REWRITE (fidelity-gate kept; sample defined; "Early baseline" from a single seed dropped) |
| G-E | **Upstream: fix what the bot depends on, one batch at a time** | each batch merged upstream fixes a deviation in a rule on the bot's path (current class's skills, monster damage/AI in Acts 1-2, mana, drops, prices); no new upstream prep while a batch is open beyond the next batch | Sorceress (or gate class) skill rows "run 0 diff" count; batches merged | REWRITE (A9 + A10 merged and scoped; open-ended coverage DROP formalised) |
| G-F | **One page of truth** | STATUS.md hand part dated within 3 days, at most 12 open cards, every card mapped to G-A..G-E; WIP limit 3 implementation agents honoured | open cards; age of STATUS | REWRITE (ADR 0012 enforced instead of restated) |

DROP or DEFER vs today: d2d container as v1 product (DEFER until G-A passes and the licence route is decided); fallback-lightning-v3 default switch (DEFER, no run planned); Barbarian deviation work (DROP unless S1 shows a Barbarian gate character); Tier B (stays HELD, trigger as in tierb-design); open-ended coverage (DROP, already stopped); idle layer, stream-grid, d2dmesh (stay parked); single-seed Gate 1 reruns on seed 11 (DROP).

## E. Plan

### Short (next 3 days)

| # | work package | trigger / exit evidence | tier | resources |
|---|---|---|---|---|
| S1 | **Class check**: read the class byte of `Gate1b`'s save and the bot log's class line on the worker host; fix the "Sorceress (class 4)" text in the records | if Barbarian: STOP all gate work, it is a bot bug (Sorceress table on a Barbarian), fix plan selection, rerun; if Sorceress: fix the record template, close barbdiff as low priority | Sonnet, main account | worker host, read only, 5 minutes |
| S2 | **Owner decision** on the product/acceptance target (appendix F, ADR 0015) | owner yes/no | owner | none |
| S3 | **Multi-seed Gate 1** on current main: seeds 11, 12, 13 + holdout 16, 17 in parallel, judged per run on stability and per set on XP/h | 5 records + one set verdict; this replaces any further seed-11 rerun | Sonnet, main account | worker host, about 3.5 wall h (xphshort estimate, U) |
| S4 | **Area choice fix** from xphshort (rank fallback areas by XP per kill; activate Dark Wood/Black Marsh waypoints) as one card with a paired A/B of 8 seeds per arm | A/B shows an effect or `no effect shown`; merge either way if it is not worse | Sonnet (code), Opus review | worker host after S3 |
| S5 | **Doc truth pass**: STATUS.md hand part rewritten to one page, PLAN.md status block (appendix F), close or defer cards not mapped to G-A..G-E | at most 12 open cards | Sonnet (Pro plan, public repo) | none |
| S6 | Upstream: answer the maintainer on #181/batch 2 only; nothing new prepared | batch 2 opened or still waiting | Opus for review replies | the Windows build PC only if asked |

### Medium (2 to 4 weeks)

1. Gate 2 on 3 seeds (one holdout) after S3/S4 pass. Exit: 3 PASS. Sonnet, main account, worker host 8 wall h per run (parallel slots).
2. A:B pilot, fresh clvl 1-6 cell (lockstep-plan round 1: 3 game.exe + 3 d2d lanes, about 2 h real), then round 2 to 6 + 6. Exit: verdict at 20 percent. Trigger for Tier B only if DIFFERENT and unexplained (tierb-design rule). Opus design check, Sonnet runs, worker host (3 hidden game.exe instances).
3. Andariel and Act 1 clear on d2d (G-B): first on a cheat-levelled throwaway to find the fight's problems (setup writes only, never counted), then fresh seeds. Exit: 3 of 3 seeds. Sonnet + Opus review, worker host.
4. Upstream batch 3 scoped to the gate class's skill rows and the Act 1 monster rules the A:B or Andariel attempts show to matter. Credit runner for emulator differentials; the Windows build PC for MSVC parity in its quiet window.
5. Gate 3 (24 h d2d) after 1 and 2. Exit: PASS with memory curve.

### Long (2 to 3 months)

1. G-A: the game.exe acceptance run: first 2 h, then 8 h, then 24 h on a hidden lane (k=2 hook allowed if the A:B shows k=2 inside noise; k=1 otherwise). Requires lifting the game.exe freeze for bug fixes found there (ADR 0009 change, appendix F). Main account, worker host.
2. Act 2 (old M4) with the Desert merc on d2d, then on game.exe.
3. Licence route and container (old M5) only after G-A; then the idle layer grill (M6).
4. A second class through Act 1 (build flexibility proven by a run, not by tests).

### Main risks
- S1 shows a Barbarian playing a Sorceress plan: every gate number since 11-01 is void (I: likely contributes to the XP gap).
- game.exe lanes load the worker host that the d2d gates also need: schedule, do not overlap (gate3prep, lockstep-plan agree, V).
- The A:B says DIFFERENT and the cause is a d2d engine gap: then G-A on game.exe becomes the only proof that counts, which is the recommendation anyway.
- The maintainer's pace caps upstream; do not let our plan wait on it.

### What to cut first if time runs short
Process and infra work (board, Discord, brand), further coverage rows, Tier B, the container, Gate 3 on d2d (Gate 2 on 3 seeds + a game.exe 8 h run says more), the k=2 vs k=1 study beyond the pilot.

## Session facts checked here

| fact (S from the coordinator) | check | tag |
|---|---|---|
| d2d primary, G1/G2 retired for a d2d ladder | ADR 0009, 0013 | V |
| two soak-gates deltas 11-03 | decisions.md, specdelta-rejudge.md | V |
| Gate 1 history and XP/h 10042 .. 7060 vs 9172 | gate1-index (3 runs on main), specdelta-rejudge, xphshort (branch) | V |
| only seed 11 ever run | all records and botab | V |
| botab cannot answer 15 percent; lane counts | botab-20261102 section 0 | V |
| seed-for-seed A:B impossible | tierb-design section 0, lockstep-plan section 1 | V |
| Sorceress no row proven clean | rule-coverage (15 deviation + 15 read-only) | V |
| gate runs play a Barbarian | records say "Sorceress (class 4)"; class 4 = Barbarian in code; plan table lightning-v2 in all archives | CONFLICT, U |
| Barbarian has six d2d deviations | barbdiff (branch) has no run; rule-coverage says 30/30 clean | U |
| PR #177 merged, batch 2 waiting for #181, maintainer rules | openpr-log | S |
| Pro / credit / main account / Windows build PC split | runner notes | S |

Plan table check (D): `python3 -I -c 'import gzip,json,collections,glob; [print(f, collections.Counter(str(json.loads(l).get("table")) for l in gzip.open(f)).most_common(1)) for f in sorted(glob.glob("docs/probes/soak-logs/gate1-2026110*-minutes.jsonl.gz"))]'` gives `fallback-lightning-v2` for all five archives on main.

## Appendix F: proposed text for guard-rule files (for the owner to approve; not applied)

### F1. docs/decisions.md (index lines)
- `- [ADR 0015](adr/0015-acceptance-on-game-exe.md) (2026-11-0X, proposed): Acceptance is game.exe 1.14d: the bot is done when a fresh character plays 24 h unattended on game.exe and clears Act 1; d2d stays the development lab and gate ladder; the d2d container is deferred until then; amends ADR 0009 item 1 and 5 and ADR 0013`
- `- 2026-11-0X (user, no ADR): gate verdicts on XP/h come from a seed set (at least 5 seeds, 2 of them holdout 16-23), never from one seed; per-run gates judge stability, invariants and funded belt windows only`
- `- 2026-11-0X (user, no ADR): the headline progress metric is virtual hours from a fresh character to Andariel dead and to Act 2, per seed; XP/h is a diagnostic`
- `- 2026-11-0X (user, no ADR): open cards are capped at 12, each mapped to one of goals G-A..G-E of research/project-review-20261103.md; the WIP limit of ADR 0012 is enforced by changecheck`

### F2. New ADR (title and text)
`docs/adr/0015-acceptance-on-game-exe.md`

> # 0015: Acceptance is game.exe; d2d is the lab
> Context: ADR 0009 made the d2d container the v1 product and said users never run game.exe; ADR 0013 moved all gates to d2d. Since then the container has had no work, d2d's rules still deviate where the bot's metrics come from (rule-coverage), seed-for-seed comparison with game.exe is impossible (tierb-design), and the owner's goal is a bot that plays the original game.
> Decision: (1) The project is done when a fresh, uncheated character plays 24 h unattended on game.exe 1.14d on a hidden lane and clears Act 1 (Andariel dead, Act 2 entered). (2) d2d remains the primary development backend and runs the gate ladder (Gate 0 to 3) for stability. (3) A final rung, Gate A, runs on game.exe: 2 h, then 8 h, then 24 h. (4) The game.exe freeze is lifted for fixes found by Gate A only (no new features that d2d does not get). (5) The d2d container is deferred until Gate A passes and the licence route is decided. (6) Transfer is measured by the statistical A:B of the fidelity-gate spec, not assumed.
> Consequences: game.exe memory reading stays maintained; lanes on the worker host are scheduled between d2d gates; ADR 0009 items 1 and 5 and ADR 0013 "Not changed" are amended.

### F3. PLAN.md status block (replace the 2026-10-29 status paragraph)
> ## Status (2026-11-03, project review)
> Current truth: docs/plan/STATUS.md and docs/research/project-review-20261103.md. The bot plays Act 1 Normal unattended for 2 h on d2d (lockstep fork), one Gate 1 pass, Gate 2 not yet passed; nothing past Act 1 live; game.exe frozen since 2026-10-30. Goals G-A..G-F of the review replace the milestone ladder below when the owner approves ADR 0015.

### F4. GLOSSARY terms
- **Gate A**: the game.exe acceptance rung after the d2d gate ladder (2 h, 8 h, 24 h on a hidden game.exe lane, fresh uncheated character).
- **Seed set**: the seeds of one gate verdict, at least 5, of which at least 2 are holdout seeds (16 to 23); XP/h is judged on the set, never on one seed.
- **Milestone time**: virtual hours from a fresh character to a named milestone (Andariel dead, Act 2 entered, Duriel dead), per seed.
