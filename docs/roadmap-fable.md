# Fable roadmap: every recommendation, its state, and what flips it

> status: current (card fableroadmap, 2026-11-26; refreshed by card roadmap2 the same day and by card roadmap3 on 2026-11-27, against `git log origin/main` at 388dcc3da). A living tracker: the coordinator edits the STATE, evidence and next-action cells when a card merges or a knob flips; the dark-launch ledger (`data/darklaunch.json`, rule `docs/process/dark-launch.md`) is the machine-checked half for default-0 knobs. It holds no counts that change: tally by `grep -c '| built-dark |' docs/roadmap-fable.md` (and the other state words).
>
> Sources: the logic review of 2026-11-25 ([research/fable-logic-review-2026-11-25.md](research/fable-logic-review-2026-11-25.md), proposals P1 to P12, tests T1 to T7, cause classes A to K); the design review of 2026-11-19 ([design/fable-review-2026-11-19/](design/fable-review-2026-11-19/00-summary.md), roadmap R1 to R17, ADRs 0020 to 0024); the four reviews of 2026-11-21 (Proving Ground gate, combat engine, quest board, town planner; [design/fable-review-2026-11-21/](design/fable-review-2026-11-21/), ADRs 0025 to 0028); the map and UI reviews of 2026-11-22 ([design/fable-review-2026-11-22/](design/fable-review-2026-11-22/)).
>
> Evidence tags: **V** read in this tree at the commit named (grep, `git log`, `docs/knobs.md`, a task file); **S** another doc's claim; **U** not known from the repo. Card status in `backlog/tasks/` lags merges (several cards read Proposed or In Progress while their commits are on main); the evidence column therefore cites `git log origin/main --grep` and the knob table, not the card's status line.

## Next up (2026-11-27, card roadmap3)

Ranked by value toward finishing every recommendation, class order Necromancer first (classorder, 2026-11-26), among work that is unblocked now. Evidence tags as in the rows; no number here is hand-counted.

1. **Necromancer live drill with `combat.deliveryUpkeep` 2** (R10; cards epic-class-necromancer TASK-942, dlv-upkeep-state, dlv-upkeep-minions, dlv-mark, dlv-stance-army). V the upkeep, minion, mark and army-stance drivers, the summon-missing invariant, life floors and the summoner build are merged, offline only; mode 2 is on sandbox Doofer. Needs the lab prep fixed first (item 2). No live Necromancer run exists (U).
2. **necroteeth (TASK-1033)**: V card result says the soak prep banks every skill point, so no class skill is learned and every class kills with plain Attack; until fixed no soak or drill tells anything about the Necromancer. Small, unblocks item 1 and every class row.
3. **Flip `quests.stepExec` on one sandbox** (P9; flipwave2 TASK-998). V the Cain scroll and bark scroll scenarios exist and the lab soak read them green (lab only); ledger `criteria_met` no, review due 2026-12-03. Standard rule, first real container run. Final list: `docs/probes/dark-queue-final-2026-11-27.md` (flip on Questy, 2 h window; the Duriel/Tyrael step stays blocked and parks safely, V sixth drill run).
4. **Duriel kill step** (R8; epic-act2 TASK-937, Act 2 for every class). V `act2drill5`: lair entered and Duriel killed 3 of 3, blocked on the kill step; the hand-in, placement and dungeon rows are merged.
5. **Arbiter default per family** (P6, R4; arbmig TASK-973, arbloop TASK-1034, hostilespin follow-up). V families 255 are on the sandbox and the session verdict exists; the open cause is a 40 min quest-board slice outlasting the 10 min watchdog (V TASK-1048 result) and one recorded night (`bot.tickLogSec`) is still missing (U). Cause class A.
6. **`town.plan` stage 4 flip** (gated on the three stage 4 scenarios, final list G2) (P5, TOWN6; townflip TASK-972, town6 TASK-758, close town5 TASK-757). V stage 3 window closed clean 2026-11-27; stage 4 sell and pickup merged dark, offline tests only. Cause class C.
7. **bsmdef restart drill, then `quests.boardSM` flip** (BSM; TASK-983). V no board commits since the ledger row; needs one restart drill. Cause class H.
8. **ckptlive drill (TASK-990) and the 24 h records**: the checkpoint and dodge knobs are on the sandbox without their drills (V ledger). Record the drill and the 24 h watch so the rows close (U on every `criteria_met`).

Not ranked but waiting on a card close, not on work: inputl3 (TASK-986) and ticklog (TASK-981) are merged yet read Proposed, so plan.py shows inputl4 and arbmig as BLOCKED (V plan snapshot below).

### Flips ready under the standard enable rule

Standard rule: scenario, drill or replay green, flip on the sandboxes, 24 h watch, auto-revert on a hard breach. Ready means the evidence the rule asks for exists in the tree today; none is flipped by this page.

| knob | to | evidence that makes it ready | gap before it closes |
|---|---|---|---|
| `quests.stepExec` | 1 on one sandbox | V scenarios `cain.scroll.stepexec`, `cain.read.stepexec`; lab soak green on them (TASK-1024, lab only) | first sandbox run, 2 h window (U) |
| `town.plan` | 4 on Doofer | NOT ready: V stage 3 window closed clean (ledger note, 2026-11-27T14:25Z); stage 4 has offline tests on two builds only, its three scenarios are not run (`docs/design/townflip-p5.md` 4b) | the three stage 4 scenarios green, owner look (ADR 0028); then a 2 h window |
| `bot.arbiterShadow`, `bot.tickLogSec` | promote from trial to enabled | V both on the sandbox since 2026-11-27 06:00Z, no revert; ledger says enable after a night of replay evidence | one night replayed with `arbiter-replay -session` (U); then a 2 h re-flip closes the rows |
| `combat.deliveryUpkeep` | keep 2 on Doofer, widen to a second sandbox | V window closed clean; minions and mark shadow runs show zero disagreements on the old Necromancer loop (card result, offline) | live Necromancer drill; `criteria_met` no |
| `bot.checkpointEnable`, `combat.missileDodge`, `combat.threatDodge`, `combat.standOK`, `combat.deliveryExpect`, `bot.portalGuard`, `bot.arbiterControl` and families 255, `town.plan` 2 | stay on; close the ledger rows | V all on the sandbox since 2026-11-26 or 27, windows of 30 and 45 minutes, no revert | nothing to flip (already `enabled`; the 2 h rule needs no 24 h record); the checkpoint live drill is not recorded (U) |

Not ready: `quests.boardSM` (restart drill), `bot.zoneModel` (A/B sets per class), `agency.planner` mode 2 (shadow review and owner go), `items.ladderRunewords` and the other owner rows (owner go).

## STATE words

`not started` (no code, maybe a card) | `in flight` (code partly merged or an agent running) | `built-dark` (merged, behind a knob at its default 0 or a mode that does nothing; has a ledger row) | `trial` (switched on in the sandboxes, not default) | `enabled` (default on, or a tool that is simply in use) | `retired` (decided against or deleted).

## Standard enable rule (user decision 2026-11-26)

Flip, watch 24 h, auto-revert: a dark feature is switched on the sandboxes as soon as its scenario, drill or replay passes; the invariant monitor watches for 24 h; a HARD invariant breach (a death, a stall, a potion not drunk, a refusal-as-failure storm) reverts the knob automatically; otherwise it stays on and the ledger row closes. No criterion needs more than 24 h of live observation; longer evidence comes from scenarios and replays. The owner of every flip is the coordinator, due next session. The tool is card `autorevert1`; until it exists the coordinator reverts by hand when the monitor reports a hard breach.

## Operating rule

Before the coordinator approves a new patch card (a `bug` or `bot-fix` card for the bot), the coordinator names its cause class (A to K of review 2026-11-25 section 2.2: A two deciders undo each other, B refusal counted as failure, C supply and economy, D navigation geometry, E stale state after portal, death or new game, F a fact the backend lacks, G threshold moved by one incident, H per-quest special case, I class-specific combat, J tests and flakes, K not visible) and checks that class against this tracker and `data/darklaunch.json`. If a row covers the class (see the "kills" column of section "Cause class to row" below), the fix card is attached to that row (its id goes into the row's card cell and the card says so in its description) and the question asked is "does the row's flip remove this cause?", not "what number do we move?". A fix card that repeats a class whose row is `built-dark` needs the coordinator to state why the row has not been flipped, or it waits. A fix card may not add a knob without removing one or the owner's word (P10). Class D (navigation geometry) is the only class this rule does not redirect: it is the game's own complexity and keeps being fixed case by case with a fixture each time (review 7.4).

### Cause class to row

| class | what | rows that kill it |
|---|---|---|
| A | deciders undo each other | ARB, P6, P2, TEMP (below) |
| B | refusal counted as a failure | P2, P4, BSM |
| C | supply and economy | TOWN, P5 |
| D | navigation geometry | MAP rows (partial), none structural |
| E | stale state after portal, death, new game | P6, P2 |
| F | a fact the backend lacks | ADR22 rows, P9, INPUT rows |
| G | threshold moved by one incident | P4, P10, ZONE |
| H | per-quest special case | P9, BSM, EPIC rows |
| I | class-specific combat | DLV rows |
| J | tests, races, flakes | PG rows, T3 |
| K | not visible | P1, P7, T4 |

## Built but dark

Updated 2026-11-27 (card roadmap3). The ledger rows (`data/darklaunch.json`, V `python3 -I tools/darklaunch/report.py`: the report lists the non-enabled rows, 0 overdue, next review due 2026-12-03 for `quests.stepExec`, then 2026-12-09). Dates are EST unless a `Z` is shown, from the ledger. A ledger status `enabled` here means switched on a sandbox by `tools/flip/flip.py`; the code default in `docs/knobs.md` is still 0 (V for the knobs grepped there), so this page calls those rows `trial`. Flip rule: scenario, drill or replay green, then `python3 tools/flip/flip.py <instance> <knob>=<value> --why ... --window 24h` (V `Merge agent/autorevert1` to `autorevert4`, db6b83314 to c04df4f8a). The tool has run: nine knobs were flipped on the sandbox on 2026-11-26 and 2026-11-27 with windows of 30 and 45 minutes, no revert (V `45d0d99b5`, `61feb4de9`), then three more rows went to `trial` (`21f455168`, `6497a9d1b`). The windows were shorter than the 24 h of the standard rule (V ledger notes), so `criteria_met` stays U on every flipped row until the 24 h watch is recorded. Default-on since 2026-11-25: P1 (`bot.invariants` 1), P2 (`bot.typedOutcome` 1), P3 (`bot.reflexDrink` 1), see the review table.

| order | feature | knob (default) | why this position | criteria (measurable) | blocker | cards |
|---|---|---|---|---|---|---|
| 0 | invariant monitor, now default on | `bot.invariants` 1 | it is the measuring stick for every flip below; its hard counters are the revert trigger | none: it is the watcher | none | invmon (Done) |
| 1 | `bot.checkpointEnable` | 0 (sandbox 1, ledger `enabled` since 2026-11-26) | smallest scope, an independent safety add; the Dark Wood deaths are its target | live drill green (portal cast healthy, retreat at low life, return after death) and the 24 h watch recorded | the drill has not run: the flip happened without it (U) | ckptlive (TASK-990, Proposed) |
| 2 | `town.plan` stages 2 to 4 | 0 (sandbox stage 3; stage 2 on since 2026-11-27T00:11Z) | largest cause class (C); stages 1 to 3 on the sandbox, stage 4 (sell, pickup) built | stage 3 24 h watch recorded, stage 4 drill green, flip stage 4, 24 h watch; stage 5 after, owner looks first (ADR 0028) | stage 4 not run on any sandbox (U) | townflip (TASK-972), town5, town6 |
| 3 | `quests.boardSM` | 0 | removes the per-quest patch treadmill for the board | restart drill green with boardSM 1 (a park survives), flip, 24 h watch | restart drill (U); bsmchk (rules 12 to 14) is not a flip blocker | bsmdef (TASK-983), bsmchk |
| 4 | `bot.arbiterControl` and `bot.arbiterFamilies` with `bot.arbiterShadow` | 0, 3 and 0 (sandbox 1, 255 and 1) | the core; P2 and P8 now exist, stage 4 session verdict and families chores, claim, goldfarm, recovery, quest and level are merged | `arbiter-replay -session` verdict green on a recorded sandbox night (`bot.tickLogSec`), quest-board slice no longer outlasts the watchdog, 24 h watch recorded; owner go per family | watchdog hang on a quest-board slice (arbloop, hostilespin); no recorded night yet (U) | arbmig (TASK-973), arbfam4 (Done), flipwave2 (TASK-998) |
| 5 | `quests.stepExec` | 0 | P9; the Cain scroll and bark scroll chunks and the Act 2 Staff chunk are built; ledger `criteria_met` no | scenarios `cain.scroll.stepexec` and `cain.read.stepexec` green 2 of 3 runs on a sandbox (the lab soak read them green, lab only), flip on one sandbox, 24 h watch | first sandbox run (U); Duriel kill step | stepexec1, stepexec2 (merged), flipwave2 (TASK-998) |
| 6 | `bot.zoneModel` | 0 | one margin for zone choice | A/B seed sets per class (offline), flip, 24 h watch | A/B sets per class; P4 step 3 | admit1, thrgov (merged) |
| 7 | `agency.planner` mode 2, `agency.autoAlloc` | 0 | the owner's central ask (path D); lower risk because the player decides by default | shadow lines replayed and reviewed, owner go for mode 2, flip, 24 h watch | review of shadow lines (U) | agencyp1b (TASK-984) |
| 8 | combat dodge and stand knobs (`combat.missileDodge`, `combat.threatDodge`, `combat.standOK`, `combat.deliveryExpect`) | 0 (sandbox 1, ledger `enabled` since 2026-11-26) | each has a live drill in its ledger row | per row in the ledger; the 24 h watch | live drills (U) | none new; ledger rows own them |
| 9 | `combat.deliveryUpkeep` mode 2 (upkeep, minions, mark, stance) | 0 (sandbox mode 2, trial since 2026-11-27) | the Necromancer summoner path (R10) runs on it | Necromancer live drill on 3 seeds, mode 1 disagreements reviewed, 24 h watch | no live Necromancer run (U) | dlv-upkeep-* cards, necro cards |
| 10 | `bot.tickLogSec` and `bot.arbiterShadow` promotion | 0 (sandbox 1, trial) | feeds row 4; the ledger says enable after a night of replay evidence | one recorded night replayed with `arbiter-replay -session` | the night (U) | ticklog (TASK-981), arbmig |

Ledger rows added since the previous version of this page: `bot.arbiterFamilies` (autorevert3), `bot.tickLogSec` (ticklog), `combat.deliveryUpkeep` (upkeep, with the Necromancer cards). Each card that adds a knob adds its ledger row in the same PR (rule 1 of `docs/process/dark-launch.md`); admit1 still owes one row per provider when a provider gets a knob.

## Review of 2026-11-25 (logic review, card fablereview2)

| id | recommendation | review | card | STATE | knob (default) | evidence | blocker | next action | owner | enable criteria |
|---|---|---|---|---|---|---|---|---|---|---|
| P1 | invariant monitor, log only (T1) | 11-25 | invmon (TASK-955, agent/invariants) | enabled | `bot.invariants` 1 | V `Merge agent/invariants` (e201e0878); `docs/knobs.md:348` default 1; `fa0dee197` (mode-no-exit fires on no progress); `263895530` (d2scenario: hard hits turn a run red). U whether both sandboxes carry the counts | none | read the `inv=` counts as the 24 h watch | coordinator | n/a (it is the watcher) |
| P2 | typed step outcome, Refused{need,wake} | 11-25 | outcome1 (TASK-956 to 961, agent/outcome1) | enabled | `bot.typedOutcome` 1 | V `docs/knobs.md:349` default 1; `git log origin/main --grep "^outcome"`: outcome1 stage 1 (`7b1b5e5af`) and the conversions outcomes2a to 2g (`b71406ea9`, `caa9c27ce`, `5c2d3919b`, `51d1dcebf`, `4cf936afd`, `e4f4b63d8`, `363a0d3cf`). U: the four partial exemptions of review 3.1 | none | read refusal counts; the exemptions need one card if they still matter | coordinator | scenario green, flip, 24 h monitor, auto-revert on a hard breach |
| P3 | reflex lane: latency, cancellation test, in-town check, drink by the safety check | 11-25 | reflex1 (TASK-962 is step a and b, agent/reflex1) | enabled | `bot.reflexDrink` 1, `bot.reflexDeadlineMs` 1000 | V `docs/knobs.md:279-280`; `git log origin/main --grep "^reflex1"`: 1a latency and T6 (`75b982d7e`), 1b in-town check (`31d35189e`), 1c reflex drink (`9ed05287c`), 1d timing fix (`333a3ae43`) | none | watch the safety-latency line; no drink-not-sent hits | coordinator | scenario green, flip, 24 h monitor, auto-revert on a hard breach (safety latency over 2 s counts as one) |
| P4 | admission function MayEnter / MayLeaveTown | 11-25 | admit1 | in flight | floors in `data/admission.json`, per provider default 0 (no ledger row yet, U) | V `5dcf5c205` (internal/admit MayEnter, MayLeaveTown) and `6c4e529c6` (old judges read floors from admission.json), `Merge agent/admit1` (f32da7a4f); no admit commit since 2026-11-26 (V `git log 57fca96d9..origin/main --grep admit`); card TASK-970 still Proposed | step 3 not started; a ledger row per provider that adds a knob | finish step 3, then one provider in shadow | coordinator | per provider: shadow replay equals the old judge or is right in a reviewed sample, then flip, 24 h monitor, auto-revert on a hard breach |
| P5 | finish and flip town.plan, delete old counters | 11-25 | townflip, town5 to town8, town11 | trial | `town.plan` 0 (stage 0..5; sandbox value 3) | V ledger row `trial` (`data/darklaunch.json`): stage 2 on since 2026-11-27 00:11Z, stage 3 on sandbox Doofer, 45 min window closed clean 14:25Z, kept on trial (`6497a9d1b`, `99ded96b1`); `docs/knobs.md:149` default 0; stage 4 sell side (`a98254d8b`, `Merge agent/townflip` add24a8d9) and stage 4 pickup side (`Merge agent/town6` 4626dd212) merged dark; town5 card In Progress, townflip and town6 cards still Proposed; town7 to town12 have no merged commit | stage 4 drills not run (U) | watch stage 3, then flip stage 4 (sell and pickup together) with `tools/flip/flip.py`; close town5, townflip, town6 cards | coordinator | stage drill green, flip, 24 h monitor, auto-revert; see Built but dark row 2 |
| P6 | arbiter migration with the outcome exit rule, TEMPORARY intents, lost-control hook | 11-25 | arbmig, arbfam | trial | `bot.arbiterShadow` 0 (sandbox 1, trial); `bot.arbiterControl` 0 (sandbox 1, ledger `enabled`); `bot.arbiterFamilies` default 3 (sandbox 255) | V stages 1 to 3 (`5e70c6326`, `2d51ef663`, `c14bea280`); stage 4, the session outcome verdict over a recorded tick log (`32a203128`, `b635246c6`, `Merge agent/arbmig2`); families chores, claim, goldfarm, recovery, quest and level (`0a739f825`, `17863b5f7`, `Merge agent/arbfam4`; card TASK-997 Done); `docs/knobs.md:351-353`. The lab soak (TASK-1024) read families 255 RED on one watchdog hang and families 3 weak RED on Necromancer xp; `hostilespin` (TASK-1048) found no spin, one 40 min quest-board slice outlasted the 10 min watchdog (V card result); `arbloop` (TASK-1034) fixes a chicken loop (card In Progress); card TASK-973 still Proposed and BLOCKED on TASK-981, which is merged (`Merge agent/ticklog`) | board slice versus watchdog (U whether fixed); owner go per family | run `arbiter-replay -session` on the first sandbox night; then decide default per family | owner for each family go | Built but dark row 4 |
| P7 | explain: decision ring, /api/explain, operator status --why | 11-25 | explain1 (TASK-965, agent/explain1) | enabled | none | V `62159cb76` (decision ring, /api/explain, operator why hold home go park), `Merge agent/explain1` (80c9b2ea1); `outcomes2g` puts refusals in the ring; ADR 0030 amendment in `docs/decisions.md:15`. U: endpoint answers on both sandboxes | none | read /api/explain after the next death | coordinator | endpoint answers on both sandboxes |
| P8 | incident to scenario (T2) plus report-only scenario tier (T7) | 11-25 | incident1 (TASK-971), incident2 (TASK-1004) | in flight | none | V `55b752d86` (incident files at death, `d2scenario incident`, three Questy rows, report-only), `Merge agent/incident1` (98c2bdfb3); `Merge agent/incident2` (744eff15f): an invariant hit writes a rate-limited pre-state file and `d2scenario incident` replays it as a report-only nightly row (unit tests only, no lockstep run); `docs/probes/incident-scenarios.md`. U: the report-only tier in the merge worker (ops repo) and its 7 clean days | ops half (the report-only tier) | count clean days; first lockstep run of an invariant row | coordinator | report-only tier 7 clean days, then owner decides (ADR 0027 item 2) |
| P9 | generic step executor with postconditions | 11-25 | stepexec1, stepexec2 | built-dark | `quests.stepExec` 0 | V `b1f391fff` (RunStep engine), `3404afe5f` (bot adapter, knob), `8a5b104e9` (ledger row, scenario cain.scroll.stepexec); `Merge agent/stepexec2` (9565fe882, 848edfa61): TALK and USE adapters, Akara bark scroll chunk, scenario cain.read.stepexec; Act 2 chunk (Horadric Staff transmute and place in the orifice, `act2drill5` `e53929dfb`: lair entered and Duriel killed 3 of 3 in the drill, blocked on the kill step); `docs/knobs.md:561`; ledger row `dark`, `criteria_met` no; the lab soak read the step executor green on the Cain scenarios (V TASK-1024 result, lab only) | Duriel kill step (U); scenario run on a sandbox (U) | flip on one sandbox through flipwave2 (TASK-998); Duriel kill chunk next | coordinator | chunk scenario green 2 of 3 runs with it on |
| P10 | threshold governance | 11-25 | thrgov | enabled | none (report-only CI check) | V `443572a0a` (rule `docs/process/threshold-governance.md`, `tools/ci/knobbudget.py` report-only, seeded evidence rows), `Merge agent/thrgov` (9ac8f343b) | none | after one session of report-only, make the check failing | coordinator | check in CI report-only for one session, then failing (the 24 h rule: evidence from scenarios, not weeks) |
| P11 | operator verbs hold, home, go, park | 11-25 | explain1 (TASK-965; hold, home, go, park verbs in the same branch) | enabled | none | V `62159cb76` and `Merge agent/explain1`; ADR 0030 amended (`docs/decisions.md:15`). U: audited and rate-limited verbs run on a sandbox | none | use the hold, home, go and park verbs once on a sandbox | coordinator | audited and rate-limited verbs on a sandbox |
| P12 | fault-injection soak (T5) | 11-25 | faultsoak, faultstall | in flight | none | V `73ee56c13` (fault proxy, fault sets, soak driver, soak.fault rows), `Merge agent/faultsoak` (b89f8cde8); `Merge agent/faultstall` (3835f3995): fault rates in time, a hold ends when the bot hangs up (TASK-1035 Done); `docs/probes/faultsoak-design-2026-11-26.md`; the fork verb is only proposed. U: the first nightly report-only run | night window ADR 0027 | run one night, turn each hit into an incident scenario | coordinator | nightly report-only; each hit becomes an incident scenario |
| T3 | property tests on the pure deciders | 11-25 | proptest1 (TASK-980, Done) | enabled | none | V `Merge agent/proptest1` (ff87b2ee9): combat Decide, supply Solve, arbiter Decide, two builds each, fixed seeds; no bug found (card result) | none | none; extend to new deciders as they land (upkeep driver, admit) | coordinator | tests in CI |
| T4 | recorded sessions, tick line, `d2arbiter -follow` | 11-25 | ticklog (TASK-981) | built-dark | `bot.tickLogSec` 0 (sandbox 1, trial on two sandboxes since 2026-11-27 06:00Z) | V `Merge agent/ticklog` (ee716b023), `docs/knobs.md:354`, ledger row `trial`; `arbiter-replay -session` reads the tick lines (`32a203128`); card result: no sandbox night recorded yet (U) | one recorded night (U) | replay the first night, then promote the row | coordinator | one real night replayed |
| T6 | cancellation-latency test | 11-25 | reflex1 (step b) | enabled | none | V `75b982d7e` "T6 cancel test for the fighter" (reflex1a), `Merge agent/reflex1` | none | none | coordinator | in CI |
| DL | no structural work parked at 0 without a flip date and a measure (7.4) | 11-25 | darklaunch, this tracker | enabled | ledger check in `tools/ci/check.sh` | V `tools/darklaunch/check.py` on main; `report.py` run 2026-11-27 lists the non-enabled rows, 0 overdue, next review due 2026-12-03 (stepExec) | none | weekly `report.py` decisions | coordinator | every row past due gets a decision |
| DEC | four decision changes (7.5): ADR 0030 allowlist (why, hold, home, go, park); `docs/bot.md` safe drink by the safety check; arbiter exit rule by outcome; ADR 0020 incident-to-scenario report-only tier | 11-25 | explain1, reflex1, arbmig, incident1 | enabled | n/a | V all four APPROVED by the owner 2026-11-26; text landed: `Merge agent/decisions1` (33f5b6247), `docs/decisions.md:15`, ADR 0030 amended | none | none | coordinator | n/a |

## Design review of 2026-11-19 (card fablereview, ADRs 0020 to 0024)

| id | recommendation | review | card | STATE | knob (default) | evidence | blocker | next action | owner | enable criteria |
|---|---|---|---|---|---|---|---|---|---|---|
| R1 | Proving Ground scenario runner as the merge gate | 11-19, ADR 0020 item 1 | provingground (done), pgexpand, pgcain | trial | none | V `internal/scenario`, `tools/provingground/scenarios`; review 2.5 says the merge worker runs no scenario tier | merge-gate wiring (mgonejob; incident1 for the report-only tier) | see PG rows below | coordinator | T0 and T1 block every merge; T2 report-only, owner decides when to block (ADR 0027) |
| R2 | Search for Cain end to end as the first scenario set | 11-19 | pgcain, cainfix2, cainfix3, cainchain, cainscroll | enabled | none | V pgcain, cainfix2, cainchain, cainscroll Done in `backlog/tasks/`; cainfix3 Proposed | live verdict of a fresh hero (U) | none | coordinator | scenarios green 2 of 3 |
| R3 | agency P0: points by verb, sheet, autopilot switch, legality | 11-19, ADR 0020 item 2 | agencyp0 | enabled | `resource.pointsDecisions` 0 by design (the player decides) | V agencyp0 Done; ledger row says stays 0 by design | none | none | owner (product) | owner go only |
| R4 | intent arbiter | 11-19, ADR 0020 item 3 | arbitershadow (done), arbmig | trial | `bot.arbiterShadow` 0 (sandbox 1) | V `docs/knobs.md:351`; ledger rows `trial` (shadow, since 2026-11-27 06:00Z, enable after a night of replay evidence) and `enabled` (control, sandbox only); see P6 | see P6 | see P6 | coordinator | Built but dark row 4 |
| R5 | quest chains as checkable data (questspec) | 11-19 | questspec | enabled | none | V `internal/questspec`, `TestShippedChains`; questspec Done | rules 12 to 14 not built (bsmchk) | bsmchk | coordinator | rules in CI |
| R6 | survival model as the one judge | 11-19 | survsd, survab8, admit1 | built-dark | `bot.zoneModel` 0 | V `docs/knobs.md:254`; ledger row | A/B sets per class; P4 | see Built but dark row 5 | coordinator | A/B per class, then default per archetype |
| R7 | agency P1: capability table, planner in shadow, then driving, viability line | 11-19 | agencyp1a (done), agencyp1b | built-dark | `agency.planner` 0 (mode 1 shadow, 2 drives) | V `docs/knobs.md:582`; ledger row `dark`; `Merge agent/prereq` (947a4a255): the planner buys missing prerequisites first (offline) | shadow review; U | agencyp1b | coordinator, owner go | see Built but dark row 7 |
| R8 | Act 2 through the quest board | 11-19 | epic-act2 (TASK-937) | in flight | none | V card Proposed but work merged: `act2plan` (Act 2 plan data, dungeon road rows), `act2temple` (life floors, arrival crowd row), `act2handin` and `bossdone` (Seven Tombs quest bits, boss kill read from game state), `placefix` to `placefix3`, `lairentry` (the way down reads as object class 100), `act2drill` to `act2drill5` (V `Merge agent/act2drill5`: lair entered and Duriel killed 3 of 3, blocked on the kill step); `act345map` (Acts 3 and 5 data, S) | Duriel kill step; Proving Ground rows for Act 2 | read the act2drill5 note, cut the kill-step chunk | coordinator | epic-act2 criteria |
| R9 | difficulty transition (Nightmare, Hell) | 11-19 | epic-normal, epic-nightmare, epic-hell | not started | none | V cards Proposed (`backlog/tasks/`) | Acts 2 to 5 | none now | coordinator | epic cards |
| R10 | Necromancer Summoner (upkeep, minions, curses) | 11-19 | epic-class-necromancer (TASK-942), necro1, necro2a to necro2e, necroteeth, dlv-upkeep-state, dlv-upkeep-minions, dlv-mark, dlv-stance-army | in flight | `combat.minionRead` 0 (ledger `dark`); `combat.deliveryUpkeep` 0 (sandbox mode 2, trial) | V merged: `necro1` (summoner guide build, default table binds Teeth), `necro2a` summon-missing invariant, `necro2b` keys (no change needed on d2d), `necro2c` life floors, `necro2d` merc policy data (not wired: no Act 1 hire step), `Merge agent/upkeep`, `minions`, `mark`, `stancearmy`, `upkeepcmp` (all behind `combat.deliveryUpkeep`, offline tests, shadow plays the old loop with zero disagreements per the minions card); open: necro2e (Corpse Explosion points, needs a scenario pair), necroteeth (TASK-1033: lab prep banks skill points, so soaks never learn a class skill). The four dlv cards still read Proposed. No live Necromancer run (U) | live drill; the lab soak did not run Necromancer defaults (V TASK-1024 result) | first live Necromancer drill with mode 2 on the sandbox; fix the lab prep | coordinator | class drill green on 3 seeds |
| R11 | Hammerdin and second Sorceress build | 11-19 | epic-class-paladin, epic-class-sorceress, dlv-stance-pack, aura2a to aura2d, auralive | in flight | none | V Paladin aura work merged: `auralive` verified live on a lab slot (a Paladin aura picked through the right skill is active at once; character load not verified), `aura2a` state line, `aura2b` Sacrifice dropped, `aura2c` Prayer rule, `aura2d` merc aura; `aimpoint` rows for Sorceress Blizzard (radii U); epic cards and dlv-stance-pack Proposed | dlv-stance-pack | next after the Necromancer drill | coordinator | per the epic cards |
| R12 | Barbarian | 11-19 | epic-class-barbarian | not started | none | V card Proposed | R10 or R11 | none now | coordinator | per the epic card |
| R13 | Acts 3 to 5 | 11-19 | epic-act3, epic-act4, epic-act5 | not started | none | V cards Proposed | R8 | none now | coordinator | per the epic cards |
| R14 | runeword and cube advisor, shrines, status reflexes | 11-19 | advisor1 | not started | none | V no card existed before this branch | agencyp1b | build after agencyp1b | coordinator | advisor lines on the Return screen reviewed by the SME agent |
| R15 | agency P3: share, import, NIP importer | 11-19 | agencyp3 | not started | none | V no card existed before this branch | agencyp1b | none now | coordinator | round trip of the eight shipped tables stays exact |
| R16 | Blizzard, Druid, Assassin, Amazon (ground target, traps, bow) | 11-19 | epic-class-druid, epic-class-assassin, epic-class-amazon, dlv-aim-point, dlv-aim-corpse | in flight | none | V `Merge agent/aimpoint` (d2ce0e875): point aim rows for Blizzard and Eruption and the Assassin trap lay behind `combat.deliveryUpkeep`; `Merge agent/castverbs` (1e7ac4599): d2d cast verbs and a corpse unit; `Merge agent/casthold`, `skillhand` (fork hold verb, hand and unit numbers); epic cards Proposed | fork-side proof of the cast verbs (U); fidelity A5, A7 to A9 | none now | coordinator | per the epic cards |
| R17 | input layer I1 to I7 | 11-19, ADR 0021 | inputlayer (I1, I2 done), inputl3, inputl4, inputl5 | in flight | `input.planner` 0, `input.hotkeyMap` 0 | V `docs/knobs.md:574-575`; inputlayer Done; `Merge agent/inputl3` (a49fdac79): town portal by the use-item verb behind `input.planner`, fork differential run 2026-11-27, not verified live; `Merge agent/forkcaps` (67d74d66e): caps set of the fork build | I4 to I7 blocked in plan.py on inputl3 (card still Proposed) | close the inputl3 card, then inputl4 | coordinator | drill passes per verb; CI pointer-call count 0 on the d2d path |
| F | fidelity lane re-pointed by class and act | 11-19 | fid-* cards, fid-matrix-refresh | in flight | none | V cards fid-matrix-refresh (task-100) and the fid-* series exist | none | continuous | coordinator | per rule: a zero-difference differential or a listed shim (ADR 0023) |
| Q4 | ladder-only runewords enabled in d.idle mode (ADR 0024) | 11-19 | q4runes (merged) | built-dark | `items.ladderRunewords` 0 | V `docs/knobs.md:588`; `Merge agent/q4runes` on main | owner go to turn the rule on | owner go | owner | gates stay vanilla; flag shown to the player |
| Q5 | engine facts as reads never make a run CHEATED; level layout read (ADR 0022) | 11-19 | k1layout, knownedge, warptrigger | built-dark | `quests.layoutRead` 0, `nav.fullGrid` 0, `nav.warpRead` 0 | V `docs/knobs.md:51-52, 555`; commits `Merge agent/k1layout`, knownedge, warptrigger on main | live drills (U) | drills per ledger rows | coordinator | quest layout drill: objects found by read in 3 areas; no stuck regressions on 5 areas; warp drill across 5 areas |
| Q7 | fidelity acceptance per rule (ADR 0023) | 11-19 | fid-* | enabled | none | V ADR 0023 accepted; header amends 0015 | none | none | coordinator | n/a |

Class order of care (user decision 2026-11-26, after the live Paladin). Dated lines, no boxes ticked:

- 2026-11-26 R10 (necromancer): priority: 1 (necromancer first in the class order)
- 2026-11-26 R11 (sorceress): priority: 2b (druid 2a and sorceress 2b are either; sorceress is the default build and already has five drill rows)
- 2026-11-26 R12 (barbarian): priority: 3 (effort: 3 plan rows, 2 profiles, melee path, no blocker beyond drill rows)
- 2026-11-26 R16 (druid): priority: 2a (either with sorceress 2b; 3 plan rows, 1 profile, no drill rows)
- 2026-11-26 R16 (amazon): priority: 4 (effort: 3 plan rows, 1 profile, smoke only, 16 unlock lines)
- 2026-11-26 R16 (assassin): priority: 5 (effort: 2 plan rows, 1 profile, needs a trap planner first, docs/classes/assassin.md)

## Reviews of 2026-11-21

### Proving Ground gate (card fablepg, ADR 0027; 16 cards)

| id | recommendation | review | card | STATE | knob (default) | evidence | blocker | next action | owner | enable criteria |
|---|---|---|---|---|---|---|---|---|---|---|
| PG1 | pgprov: result records bot commit, d2d hash, host role, load | 11-21 | pgprov, pgprov2 | enabled | none | V card TASK-733 Done; `Merge agent/pgprov2` (989af6273): slot load (parallel) in the record, stability needs two parallel runs | none | none | coordinator | records present in a run |
| PG2 | pgdigest: assertion digest D1, determinism compare | 11-21 | pgdigest | enabled | none | V card TASK-735 Done | none | none | coordinator | determinism pair run |
| PG3 | pgjournal: journal variable, `determinism-run -n N` | 11-21 | pgjournal | enabled | none | V card TASK-737 Done | none | none | coordinator | pair proof written up |
| PG4 | pgtaps: scripted taps on the virtual clock | 11-21 | pgtaps | in flight | none | V one commit on main | none | finish | coordinator | scenario run id in the result |
| PG5 | pgclock: goals clock, farm timers on the env clock | 11-21 | pgclock | in flight | none | V `Merge agent/pgc5` on main | none | finish | coordinator | two-build test |
| PG6 | pgstability: stable, quarantined, tier retag | 11-21 | pgstability | enabled | none | V card TASK-740 Done; `Merge agent/pgstability` (4ef40348b): state from the ledger, auto-quarantine on a same-binary flip, tier retag | none | none | coordinator | auto-quarantine on a verdict flip |
| PG7 | pghero: hero library with played and banked profiles | 11-21 | pghero | enabled | none | V card TASK-741 Done; `Merge agent/pghero` (c99e25214), prep defaults to the played profile | none | none | coordinator | survival prep uses it |
| PG8 | pgselfcheck | 11-21 | pgselfcheck | enabled | none | V card TASK-742 Done; `Merge agent/pgselfcheck` (0b82c29ec), not verified live | live look (U) | none | coordinator | self-check passes on the prep d2d |
| PG9 | pgledger: JSONL ledger and ingest | 11-21 | pgledger, pgledgeron | in flight | none | V `pgledgeron` commit on main | none | finish | coordinator | results written on the host |
| PG10 | pgpage: board page and status doc | 11-21 | pgpage | in flight | none | V `Merge agent/pgpage` (670290dff): page-json and generated status doc; card TASK-744 still Proposed, no ledger lines exist yet (card result) | first ledger lines | close the card after the first run | coordinator | page shows the tiers |
| PG11 | mgonejob: merge gate as one locked builder job | 11-21 | mgonejob | not started | none | V card Proposed, no commit | ops repo | build in the ops repo | coordinator | positive ran-marker |
| PG12 | mgcache: cached whole-repo gate, nightly uncached race | 11-21 | mgcache | in flight | none | V card TASK-747 Done, two commits on main | 20-merge measurement (U) | measure | coordinator | measured over 20 merges, then default |
| PG13 | mglanes | 11-21 | mglanes | enabled | none | V card Done, `Merge agent/pgc1` | none | none | coordinator | n/a |
| PG14 | mgwatchdog | 11-21 | mgwatchdog | enabled | none | V card Done, `Merge agent/pgc2` | none | none | coordinator | n/a |
| PG15 | pgnightly: one nightly job (T2, T3, T4) | 11-21 | pgnightly | not started | none | V card Proposed | night window ADR 0027 item 4 | build | coordinator | one night run, at most 3 runs |
| PG16 | pgearned: earned save chain, `pre.from_save` | 11-21 | pgearned | not started | none | V card Proposed | save portability U | last | coordinator | one earned act run |
| PGT | tiers T0 to T4; T0 and T1 block every merge, T2 report-only then blocking after 7 clean days | 11-21, ADR 0027 | incident1 (TASK-971, report-only tier), mgonejob | in flight | none | V review 2.5: no scenario tier in the merge worker | PG11 | wire T0 and T1; T2 report-only | owner (after 7 clean days) | owner decides when to block |

### Combat engine (card fablecombat, ADR 0026; 16 cards plus one parked)

| id | recommendation | review | card | STATE | knob (default) | evidence | blocker | next action | owner | enable criteria |
|---|---|---|---|---|---|---|---|---|---|---|
| DLV0a | one capability table, generated views, tiers | 11-21 | dlv-data | enabled | none | V `data/capabilities/capabilities.json`, `internal/delivery/gen.go`; commits `80c5939bf`, `0bbdc37d3` | none | none | coordinator | generated views equal checked-in files |
| DLV0b | Rotation type, FromProfile adapter, golden parity | 11-21 | dlv-rotation | enabled | none | V card TASK-719 Done; commit `dd605af3b` | live look (U) | none | coordinator | golden parity over the corpus |
| DLV0c | expectation ledger | 11-21 | dlv-expect | trial | `combat.deliveryExpect` 0 (mode 2 max; sandbox 1) | V `docs/knobs.md:131`; ledger row `enabled` on sandbox d2play-20261108 since 2026-11-26 (`45d0d99b5`); card TASK-720 Done | live look (U) | read the mode 1 failure log from the sandbox | coordinator | mode 1 failure log replayed, no false failures; flip, 24 h monitor, auto-revert on a hard breach |
| DLV0d | read verbs minions, skillstate, combat events | 11-21 | dlv-reads | enabled | none | V card TASK-721 Done; fork branch named in the card | live (U) | none | coordinator | scenario per assertion kind |
| DLV1 | upkeep (state, minions), mark, stance behindArmy | 11-21 | dlv-upkeep-state, dlv-upkeep-minions, dlv-mark, dlv-stance-army | trial | `combat.deliveryUpkeep` 0 (0 off, 1 shadow, 2 acts; sandbox mode 2) | V `Merge agent/upkeep` (d3a579824), `minions` (acf7e202c), `mark` (b7f493e78), `stancearmy` (685f85220), `auraupkeep`, `upkeepcmp` (comparison counts on the minute line); ledger row `trial`, mode 2 on sandbox Doofer, 45 min window closed clean 2026-11-27T14:25Z, `criteria_met` no; four cards still Proposed, results say offline only | live look; scenario run per card (U) | decide default after the Necromancer drill | coordinator | per card scenario; mode 1 shadow disagreements reviewed, then flip |
| DLV2 | cast verbs, point and corpse aim, stance insidePack | 11-21 | dlv-cast-verbs, dlv-aim-point, dlv-aim-corpse, dlv-stance-pack | in flight | none | V `Merge agent/castverbs` and `aimpoint` on main (cards still Proposed); aim-corpse and stance-pack have no merged commit | fork-side proof (U); R6 survival judge before `insidePack` acts | stance-pack, then aim-corpse | coordinator | per card scenario |
| DLV3 | press, Teleport driver, Charge and Leap (Whirlwind parked) | 11-21 | dlv-press, dlv-move-tele, dlv-move-melee, dlv-whirl | not started | none | V cards Proposed | wave 2; fidelity A6 for Whirlwind | after wave 2 | coordinator | per card scenario |
| DLVc | cleanup: delete skills.json, skillKinds, class switches | 11-21 | dlv-cleanup | not started | none | V card Proposed | every driver knob at 2 for the agreed number of nights | last | coordinator | CI literal check at 0 |

### Quest board state machine (card fableboard, ADR 0025; 10 cards)

| id | recommendation | review | card | STATE | knob (default) | evidence | blocker | next action | owner | enable criteria |
|---|---|---|---|---|---|---|---|---|---|---|
| BSM | stored runs and latches in the plan file (cards 1 to 7 of the review) | 11-21 | bsm1 to bsm9 | built-dark | `quests.boardSM` 0 | V `docs/knobs.md:548`; `Merge agent/bsm1` to `bsm9` on main (bsm7 card still reads Proposed); `docs/probes/boardsm-progress-2026-11-21.md`; ledger row | restart drill (U); the P1 counts exist now | bsmdef (TASK-983) | coordinator | Built but dark row 3 |
| BSMc | questspec rules 12 to 14 in the merge gate | 11-21 | bsmchk | not started | none | V no such rules in `internal/questspec` | none | build | coordinator | rules in CI |
| BSMa | the board as an arbiter intent source | 11-21 | arbmig (family quest) | retired | none | V no card of its own; it is one family of P6 (TASK-973), a separate row only duplicates it (2026-11-26) | none | tracked in P6 | coordinator | see P6 |
| BSMd | default 1 after the scenario set is green; later remove old paths and dual-write | 11-21, ADR 0025 item 11 | bsmdef | not started | `quests.boardSM` 0 | V ledger row | restart drill (U) | build (TASK-983) | coordinator | Built but dark row 3 |

### Town planner (card fabletown, ADR 0028; 11 cards)

| id | recommendation | review | card | STATE | knob (default) | evidence | blocker | next action | owner | enable criteria |
|---|---|---|---|---|---|---|---|---|---|---|
| TOWN1 | planner scenarios | 11-21 | town1 | enabled | none | V Done, `Merge agent/town1` | none | none | coordinator | n/a |
| TOWN2 | one counter (`supply.CountOf`) | 11-21 | town2 | enabled | none | V Done; `internal/supply/count.go`; at stage 0 it returns the old numbers | none | none | coordinator | n/a |
| TOWN3 | NeedFor, Solve, profiles, stage 1 shadow trace | 11-21 | town3 | trial | `town.plan` 0 (sandbox 3) | V `docs/knobs.md:149`; `Merge agent/town3`; covered by the sandbox value, see P5 | none | see P5 | coordinator | stage 1: shadow lines present |
| TOWN4 | stage 2: guards, chunk preconditions and shop read the plan | 11-21 | town4 | trial | `town.plan` 0 (sandbox 3) | V `Merge agent/town4`; stage 2 on the sandbox since 2026-11-27T00:11Z (ledger); `docs/probes/town-plan-stage2-2026-11-21.md` | none | see P5 | coordinator | Built but dark row 2 |
| TOWN5 | stage 3: purse settled once per visit, recovery reads wealth | 11-21 | town5 | trial | `town.plan` 0 (sandbox 3) | V `a559c72fe`, `Merge agent/townstage3`; stage 3 on sandbox Doofer, window closed clean 2026-11-27T14:25Z (ledger note); `docs/probes/town-plan-stage3-2026-11-22.md` (CHEATED setups); card TASK-757 still In Progress | none | watch, then stage 4 | coordinator | stage drill green, flip, 24 h monitor, auto-revert on a hard breach |
| TOWN6 | stage 4: sell, keep and pickup | 11-21 | town6, townflip | built-dark | `town.plan` 0 (stage 4 not on any sandbox) | V `a98254d8b` (sell side) and `Merge agent/town6` (pickup side); card result: offline tests on Sorceress and Paladin, not verified live; cards still Proposed | stage 3 watch | flip stage 4 after stage 3 (P5) | coordinator | per stage look |
| TOWN7 | short-gold priority table as data | 11-21 | town7 | not started | none | V card Proposed | none | build | coordinator | table reviewed |
| TOWN8 | stage 5: route stops by nav cost | 11-21 | town8 | not started | none | V card Proposed | stage 4 | build | coordinator | per stage look |
| TOWN9 | viewer GET /api/townplan and a card | 11-21 | town9 | not started | none | V card Proposed | none | build | coordinator | endpoint answers |
| TOWN10 | town.plan as arbiter source and visitor | 11-21 | town10 | not started | none | V card Proposed | P6 | with P6 | coordinator | see P6 |
| TOWN11 | cleanup: delete stage 0 and retired knobs | 11-21 | town11 | not started | none | V card Proposed | stage 5 stable | last | coordinator | old counters deleted, knob rows removed |
| TOWN12 | SME review of the supply profile numbers | 11-21 | town12 | not started | none | V card Proposed | none | run the SME agent once | coordinator | advice recorded |

## Reviews of 2026-11-22 (map knowledge, UI)

| id | recommendation | review | card | STATE | knob (default) | evidence | blocker | next action | owner | enable criteria |
|---|---|---|---|---|---|---|---|---|---|---|
| MAP1 | mapverify: Countess three ways with the map knobs, Monastery Gate border strip memo | 11-22 | mapverify | enabled | none | V Done; 23 commits matching the card on main | none | none | coordinator | n/a |
| MAP2 | FullyKnown measured; a whole grid is planned on alone | 11-22 | knownedge | built-dark | `nav.fullGrid` 0 | V `docs/knobs.md:51`; ledger row | A/B nav drill (U) | drill | coordinator | no stuck regressions on 5 areas |
| MAP3 | read warp trigger areas from the engine fork | 11-22 | warptrigger | built-dark | `nav.warpRead` 0 | V `docs/knobs.md:52`; fork branch pushed per the card result | drill (U) | drill | coordinator | warp drill across 5 areas |
| MAP4 | engine link table against the bot's exit data | 11-22 | exitsread | enabled | none | V one commit on main; card result: verified on the lab d2d (CHEATED, link facts only) | fix card for the findings (U) | read the result | coordinator | n/a |
| UI | UI unification (navigation, sheets, toasts, build cards, pre-cockpit deletion E11) | 11-22 | uinav, uisheet, uitoast, uisettings, uifit1 to uifit4 and others | in flight | none | V most listed ui* cards Done in `backlog/tasks/`; uifit1 Proposed | E11 deletion after the Settings sheet | see `docs/design/fable-review-2026-11-22/e11-deletion-plan.md` | coordinator | per the plan |

## Other built-dark rows of the ledger (not from a Fable recommendation)

These are in `data/darklaunch.json` and owned by the ledger, not by this tracker: `resource.gearDecisions`, `resource.skillCapPerLevel` (retired 2026-11-27, TASK-1055, V ledger row retired), `agency.autoAlloc`, `combat.minionRead`, `quests.layoutRead`, `loot.vault`, `loot.rulesEngine`, `bot.farmReset`, `bot.mercReviveUnverified`, `bot.cubeJobs`, `town.identifyTome`, `ladder.enabled`, `invdesk.verbs`, `invdesk.instant`, `ui.managementFast`, `watch.paletteFeed`, `input.planner`, `input.hotkeyMap`, `items.ladderRunewords`. Flipped on the sandbox since 2026-11-26 (ledger `enabled`, default still 0): `combat.missileDodge`, `combat.threatDodge`, `combat.standOK`, `bot.portalGuard`. The weekly `python3 -I tools/darklaunch/report.py` is their review; do not repeat them here.

## Order of work (the coordinator's reading, updated 2026-11-27)

1. Done: P1, P2, P3, P7/P11, P8 (writer and invariant rows), P10, T3 property tests, the flip tool with auto-revert (autorevert1 to autorevert4), the first nine sandbox flips, PG1 to PG3 and PG6 to PG8 (cards Done), the Necromancer data and driver cards (offline).
2. Watch: stage 3 of `town.plan`, `combat.deliveryUpkeep` mode 2, `bot.tickLogSec`, `bot.arbiterShadow` (the sandbox windows closed clean, none has the 24 h record yet).
3. Flips next (see "Next up"): `quests.stepExec` through flipwave2, `town.plan` stage 4, the `bsmdef` restart drill, then the arbiter default per family once the watchdog slice is fixed.
4. In flight to finish: admit1 step 3 (P4), arbloop (TASK-1034), pgtaps, pgclock, pgledgeron, pgpage card close, town5 card close.
5. Everything else (dlv waves, classes, acts, input I4 to I7) follows the epic cards; the class order is decided (classorder, 2026-11-26): Necromancer first.

## Triage of the rows still `not started` (card roadmap2, 2026-11-26)

Written 2026-11-26; many of these rows have since moved (see the rows above, the "Next up" section and `grep -c '| not started |' docs/roadmap-fable.md`). Each row gets one decision: **card** (a card exists, linked), **new card** (created by this branch), **retired**, or **owner** (lane waiting-owner). No row needed a design decision from the coordinator; the card sizes are the Size line of each card (V `backlog/tasks/`).

### Review of 2026-11-25 (2 rows)

| row | size | decision |
|---|---|---|
| T3 proptest1 | S | card TASK-980 (lane next, low). Do after the first flips; property tests on combat.Decide, supply.Solve, the arbiter |
| T4 ticklog | S | card TASK-981 (next). Pick: gives one real night to replay, and arbmig TASK-973 still lists it as a dependency (shows BLOCKED in plan.py although arbmig stages 1 to 3 merged; U whether the dependency is still true, the coordinator should drop or keep it) |

### Overhaul review of 2026-11-19 (9 rows)

| row | size | decision |
|---|---|---|
| R8 Act 2 | L | card epic-act2 TASK-937 (later); blocker is further P9 chunks (Duriel place_staff): no chunk card yet, to be cut when `stepexec1` chunk one is flipped |
| R9 difficulty transition | L x3 | cards epic-normal, epic-nightmare, epic-hell (later); needs Acts 2 to 5 |
| R10 Necromancer | L | cards epic-class-necromancer TASK-942 and dlv-upkeep-* (next); class order is the owner's: new card classorder1 (TASK-999, waiting-owner) |
| R11 Hammerdin, 2nd Sorceress | L | cards epic-class-paladin, epic-class-sorceress, dlv-stance-pack; order via classorder1 |
| R12 Barbarian | L | card epic-class-barbarian TASK-943 (later); order via classorder1 |
| R13 Acts 3 to 5 | L x3 | cards epic-act3, epic-act4, epic-act5 (later) |
| R14 runeword and cube advisor | M | card advisor1 TASK-991 (blocked by agencyp1b TASK-984, correct) |
| R15 agency P3 share, import | M | card agencyp3 TASK-985 (blocked by TASK-984, correct) |
| R16 ground-target classes | L x3 | cards epic-class-druid, assassin, amazon, dlv-aim-point, dlv-aim-corpse; order via classorder1 |

### Reviews of 2026-11-21: Proving Ground gate, combat engine, quest board, town (20 rows)

| row | size | decision |
|---|---|---|
| PG6 pgstability | M | card TASK-740 (next); needs PG2, PG3 merged first |
| PG7 pghero | L | card TASK-741 (next) |
| PG8 pgselfcheck | M | card TASK-742 (next), after PG7 |
| PG10 pgpage | M | card TASK-744 (next), after PG9 |
| PG11 mgonejob | M | card TASK-745 (next); ops repo work, the merge-worker half is live there (U) |
| PG15 pgnightly | M | card TASK-754 (next); night window ADR 0027 |
| PG16 pgearned | L | card TASK-756 (next); last, save portability U |
| DLV1 | L, L, M, M | cards dlv-upkeep-state, dlv-upkeep-minions, dlv-mark, dlv-stance-army (next); wave 0 cards merged with results, live look U |
| DLV2 | L, L, S, L | cards dlv-cast-verbs, dlv-aim-point, dlv-aim-corpse, dlv-stance-pack (next) |
| DLV3 | M, L, M, M | cards dlv-press, dlv-move-tele, dlv-move-melee, dlv-whirl (whirl parked per ADR 0026) |
| DLVc dlv-cleanup | M | card TASK-736 (next); last |
| BSMc bsmchk | S | card TASK-982 (next); not a flip blocker |
| BSMd bsmdef | M | card TASK-983 (next); the boardSM flip, Built but dark row 3 |
| BSMa | none | RETIRED: one family of P6, tracked there |
| TOWN6 stage 4 | M | card town6 TASK-758 (next); after stage 3 is on |
| TOWN7 short-gold table | M | card town7 TASK-759 (next) |
| TOWN8 stage 5 | M | card town8 TASK-761 (next) |
| TOWN9 /api/townplan | M | card town9 TASK-762 (next) |
| TOWN10 town as arbiter source | M | card town10 TASK-763 (next); also a family of P6 (TASK-973): decide there whether TASK-763 stays |
| TOWN11 cleanup | M | card town11 TASK-764 (next); last |
| TOWN12 SME review | S | card town12 TASK-765 (next); one SME run |

New cards from this pass (no earlier card covered them): `flipwave2` (TASK-998, lane next, epic fableroadmap): the flips of `bot.arbiterControl` (survive, player families) and `quests.stepExec`, which have ledger rows but no card of their own; `classorder1` (TASK-999, lane waiting-owner): the owner's class order. Rows with U cannot be ticked from this tree.

## First five picks (2026-11-26, superseded)

Superseded by the "Next up" section at the top of this page (2026-11-27). Of the five: ckptlive was flipped on the sandbox without its drill, townflip stage 2 and 3 are on the sandbox, flipwave2 partly (arbiter control on the sandbox, stepExec not), bsmdef not started, ticklog merged.

## Plan snapshot 2026-11-27 (`python3 -I tools/backlog/plan.py`, top of the output)

```
PLAN 2026-11-27
NOW (7)
  TASK-337 campaign1              hig credit-runner Proposed    
  TASK-183 recoverypace           med live-agent    Proposed    
  TASK-472 uib9                   med credit-runner Proposed    
  TASK-738 pgtaps                 med credit-runner In Progress  6d
  TASK-739 pgclock                med credit-runner Proposed    
  TASK-757 town5                  med credit-runner In Progress  6d
  TASK-777 pgledgeron             med credit-runner Proposed    
NEXT (93)
  TASK-27  testerreport           hig credit-runner Proposed    
  TASK-39  earlydeath13           hig pro-runner    Proposed    
  TASK-245 firsthour              hig live-agent    Proposed    
  TASK-246 sessions               hig live-agent    Proposed    
  TASK-303 repairtrip             hig credit-runner Proposed    
  TASK-609 survrecover            hig credit-runner Proposed    
  TASK-972 townflip               hig credit-runner Proposed    
  TASK-981 ticklog                hig credit-runner Proposed    
  TASK-983 bsmdef                 hig credit-runner Proposed    
  TASK-998 flipwave2              hig live-agent    Proposed    
  TASK-1033necroteeth             hig credit-runner Proposed    
  TASK-40  tpreserve-live         med live-agent    Proposed    
  ... 81 more (plan.py --lane next)
WAITING ON OWNER (3)
  TASK-107 h1baseline             hig user          Proposed      assignee user: needs the owner decision
  TASK-112 h3verdict              med user          Proposed      assignee user: needs the owner decision
  TASK-999 classorder1            low user          Proposed      owner decides which class the epics follow first (R10 to R
BLOCKED (10)
  TASK-29  testerpack             hig credit-runner Proposed      waits for TASK-28
  TASK-108 qdensity               hig credit-runner Proposed      waits for TASK-19
  TASK-973 arbmig                 hig credit-runner Proposed      waits for TASK-981
  TASK-114 designbrief            med credit-runner Proposed      waits for TASK-112
  TASK-115 qqueue                 med credit-runner Proposed      waits for TASK-19
  TASK-988 inputl4                med credit-runner Proposed      waits for TASK-986
  TASK-103 fid-poison-res-diag    low credit-runner Proposed      waits for TASK-72
  TASK-985 agencyp3               low credit-runner Proposed      waits for TASK-984
  TASK-989 inputl5                low credit-runner Proposed      waits for TASK-988
  TASK-991 advisor1               low credit-runner Proposed      waits for TASK-984
STALE (no commit on owns) (1)
  TASK-28  testerlicence          hig credit-runner In Progress  20d  lane parked
LATER 16, PARKED 53 (plan.py --lane later|parked)
DARK-LAUNCH
  age(d)  name                         status  met  owner        due
      28  bot.mercReviveUnverified     dark    U    coordinator  2026-12-09 (12d)
      28  bot.cubeJobs                 dark    U    coordinator  2026-12-09 (12d)
      22  ladder.enabled               dark    U    coordinator  2026-12-09 (12d)
      17  resource.gearDecisions       dark    U    owner        2026-12-09 (12d)
      15  invdesk.verbs                dark    U    coordinator  2026-12-09 (12d)
      13  loot.rulesEngine             dark    U    coordinator  2026-12-09 (12d)
      13  invdesk.instant              dark    U    coordinator  2026-12-09 (12d)
      13  bot.zoneModel                dark    U    coordinator  2026-12-09 (12d)
      12  town.identifyTome            dark    U    coordinator  2026-12-09 (12d)
      12  bot.farmReset                dark    U    coordinator  2026-12-09 (12d)
      11  watch.paletteFeed            dark    U    owner        2026-12-09 (12d)
```

## How to keep this file true

- On each merge that touches a row's card: update STATE, evidence (commit or doc) and next action in the same PR as the coordinator's merge bookkeeping, not in the feature PR (the feature agent does not own this file).
- A row moves to `trial` when the knob is set on a sandbox; to `enabled` only when the default changes and the ledger row says `default-on`.
- A row with `built-dark` past its `review_by` appears in `report.py` as due; the decision is enable, extend (with a changed why) or retire.

## Cards created by the fableroadmap branch

Cards for work that has no other card. The pieces already carded by other agents are NOT recreated: P1 invmon is TASK-955, P2 is TASK-956 to 961, P3 is TASK-962 (agent/reflex1), P7 and P11 are TASK-965 (agent/explain1), P8 is TASK-971 (agent/incident1), the ledger is TASK-954. All status Proposed, size and order in each card; none ticks a box. Task ids are assigned on this branch and can shift at merge (the card id in the first column is the stable name).

| card | task | tracker row |
|---|---|---|
| admit1 | TASK-970 | P4 |
| townflip | TASK-972 | P5 |
| arbmig | TASK-973 | P6 (also BSMa, TOWN10 as sources) |
| stepexec1 | TASK-976 | P9 |
| thrgov | TASK-977 | P10 |
| faultsoak | TASK-979 | P12, T5 |
| proptest1 | TASK-980 | T3 |
| ticklog | TASK-981 | T4 |
| bsmchk | TASK-982 | BSMc |
| bsmdef | TASK-983 | BSMd, BSM |
| agencyp1b | TASK-984 | R7 |
| agencyp3 | TASK-985 | R15 |
| advisor1 | TASK-991 | R14 |
| inputl3 | TASK-986 | R17 (I3) |
| inputl4 | TASK-988 | R17 (I4) |
| inputl5 | TASK-989 | R17 (I5 to I7) |
| autorevert1 | TASK-994 | standard enable rule (flip, 24 h watch, auto-revert) |
| ckptlive | TASK-990 | checkpoint row of Built but dark |

Cards created by roadmap2 (2026-11-26): flipwave2 (TASK-998), classorder1 (TASK-999). Existing cards that already cover a recommendation (dlv-*, pg*, mg*, town1 to town12, bsm1 to bsm9, epic-act*, epic-class*) are not recreated; only the epic blockers that map to a row are linked in the rows above.
