# Bot generality: framework or stencil? (agent GENERALITY, 2026-10-31)

> status: design study, read-only, decides nothing. Card `generality`. Answers the user's question of 2026-10-31 ("a flexible framework, or a stencil?") with counts, terms, a target architecture, ranked experiments and a card plan. Tags: V = counted in this worktree (command or file:line given), S = read in a doc, U = judgment. Numbers with an owner (knob defaults, gates) are cited, not copied. SME notes in section 7.

## 1. Headline

**Wide in code, narrow in evidence, fitted in policy.** The stencil is not where it first looks. The bot's *code* already reaches every class, act and difficulty: class profiles, builds, five act campaigns, bosses, Nightmare and Hell tables. Almost none of that has run. What has run, and been tuned, is one path: a fresh Sorceress in Normal Act 1, through clvl 13. The rules that make that path work (recovery, kit, safe areas, floors) are constants picked to fix what the soaks showed. Per layer:

| layer | verdict | why (short) |
|---|---|---|
| engine and backend (d2d adapter, contract, determinism, replay, pacer) | framework | 12.5k lines in `internal/d2d*` behind 26 backend-contract requirements; only 8 non-comment d2d references in the policy packages (V, grep in section 2) |
| game knowledge (tables) | framework | 33 game tables loaded from the MPQ (`internal/data`: monstats, levels, skills, items, treasureclassex, hireling, ...) (V) |
| mechanisms (nav, cast, buy, sell, stash, belt, corpse) | mostly framework | nav works from map data on any level (never walked past Act 1 on d2d); town's NPC table is act-keyed (`town/acts.go:42-50`) but only the Rogue Encampment is proven live (S d2-sme) |
| plan data (builds, progression, runs) | broad, unproven | 20 build files for 7 classes, progression for 3 difficulties (Normal: 44 area rows), runs.json, farm-runs.json (V `internal/plandata`) |
| campaign code | broad, unproven, partly duplicated | `internal/task` act1 to act5, nightmare.go, hell.go; `internal/boss` act1 to act5. Floors are copied from runs.json into code (`boss/act3.go:65`, `task/act5.go:58`) (V) |
| class model | half-built | data-driven `combat/classdata/profiles.json` (8 profiles, 7 classes), added after the smoke tests. But class if-chains remain in 11 files, e.g. `town/rebuy.go:126` (Amazon), and `leveler/prefs.go:34` silently falls back to the Sorceress (V) |
| policy (recovery, economy, safety, readiness) | **stencil** | five hand-written Act 1 area-id sets, fixed clvl floors, knobs with no provenance (section 2) |
| evidence | **stencil past clvl 5** | 12 seeds x 2 virtual h at clvl 1 to 5; anything longer is n=1 on seed 5; one class, one engine (section 2) |

## 2. Measuring the stencil (V unless tagged)

| axis | what is fitted to it | count / file:line |
|---|---|---|
| class (Sorceress) | non-test files that mention the Sorceress | 49 files; most in `leveler/build.go` (12), `combat/classdata.go` (9), `leveler/prefs.go` (6) |
| class | class `switch`/`==` branches in policy code | 11 files (`town/panel/panel.go` 7, `combat/druid.go` 6, `bot/routeguard.go` 4, `town/rebuy.go` 2, ...) |
| class | smoke tests of other classes (Game.exe era, cheat-levelled, report-only) | Paladin 3 BLOCK / 9 KNOB, Necromancer 3 / 11, Amazon 5 / 9 (docs/classes/smoke-*.md). No other class has ever played unattended on any engine |
| Act 1 / areas | hand-written area-id sets in policy | `bot/broke.go:36` safeTier, `bot/kitgate.go:15` kitSafeTier, `bot/kitgate.go:19` dungeonMouths, `bot/recoveryxp.go:13` recoveryXPTier, `resource/town.go:18` walkableHome, and `combat/resource.go:22` otherActFields (S, never walked) |
| Act 1 | Act 1 area or NPC names in non-test code | Blood Moor 33 files, Den of Evil 27, Rogue Encampment 22, Andariel 21 (many are comments; the sets above are the live ones) |
| Act 1 / clvl | spec requirements naming Act 1 places / a clvl | early-game-policy 5 and 9 of 17; merc-policy 6 and 3 of 7; soak-gates 1 and 3 of 10 |
| clvl band | literal clvl comparisons in non-test code | 67 (bot 6 files, resource 4, town 3, task 3, ...) plus clvl floors as knobs (`boss.brCasterLevel`, `boss.andarielMinLevel`, `town.beltBuyLevel`, `combat.championEarlyPct` "below clvl 15") |
| difficulty | branches on difficulty | 45 references (`d2.Hell` 21, `d2.Normal` 18, `d2.Nightmare` 6). Nightmare shares the Hell potion policy (`combat/fighter.go:254`) |
| engine | d2d-specific code in policy packages | 8 non-comment lines (bot/goal.go, loop.go, town/trip.go, nav/*): the backend split holds |
| knobs | tuning knobs (docs/knobs.md knobdump block) | 254: resource 55, combat 45, town 43, bot 43, nav 24, loot 18, merc 11, boss 8, campaign 4, route 3. 10 are 0/1 switches that default to off |
| knob provenance | knob descriptions that name any source | about 39 of 254 (15 %): SME 19, spec/ADR 7, measured 8, derived 5 (keyword grep, overlaps). `env.Knob` has no provenance field (`env/tuning.go:40`). docs/knobs.md: "No knob has a recorded live A/B test of its range"; ranges are "Sorceress, Normal Act 1" simulation windows |
| tests that pin today's behaviour | `PinsCurrent` audit tests | present in `internal/town/basicsaudit_test.go` (roadmap section 1.3) |
| seeds | soakfull 2 virtual h, clvl 1 to 5 | 12 seeds (s1-s8, s12-s15): 0 deaths, 150-228 kills/h, 3628-5606 XP/h (docs/probes/soakfull/runs.csv). xpgap: 8 seeds, SE 7-9 % of the mean (S docs/d2d-xp-gap.md) |
| seeds | long runs (clvl 9 to 13, hours) | navsoak and navsoak2: seed 5 only, n=1 paired; both ended in the same town wedge (S docs/probes/navsoak-compare.md) |
| replay fixtures | recorded sessions under testdata | 2 (`task/testdata/*lab5-seed5*`, `*seed3*`), plus combat death fixtures |

Surprises: (a) the code is far broader than the evidence: act 2 to 5, Nightmare and Hell campaigns exist as code; (b) the early game is already multi-seed, so "mostly seed 5" holds only for the long runs; (c) the special-case rules sit in one place, the recovery/kit/safe-area policy of `internal/bot`, and are keyed by raw Act 1 area ids; (d) 85 % of knobs carry no stated source.

## 3. A better vocabulary

The user's question is a machine-learning one: **does the policy transfer, or is it overfit?** We tuned on a training set (seed 5, Sorceress, Act 1, clvl 1 to 13) and never tested on held-out cases. Six terms, each with a test:

| term | definition | how we would know |
|---|---|---|
| **coverage** (claim x axes) | Each claim names the axes it was proven on: seeds (count, holdout or not), class/build, act, difficulty, engine, start state (fresh, cheat-levelled, kit), horizon (virtual h). Outside those axes a claim is UNPROVEN, not PASS | every closing card's `result:` and every gate line says "proven on: ..."; a coverage matrix (claims x axes) can be filled from the archive alone |
| **transfer** (and transfer gap) | Running on a new axis value with no policy code change; the gap is the metric drop against the trained case | a holdout seed, a second class or a new start state runs with data-only changes; gap reported per metric |
| **mechanism vs policy** | Mechanism = how (walk, cast, buy, read memory). Policy = what and when (where to farm, when to go to town, when ready) | a new act or class needs no mechanism edit; policy diffs are data or one decision function |
| **knowledge vs behaviour** (data vs code) | Facts about the game (areas, monsters, builds, floors) live in tables; code only combines them | count of raw area ids, class branches and copied floors in policy code (section 2) trends to 0 |
| **constant provenance** | Each number is *measured* (from our logs, with doc), *derived* (formula from game data), *judged* (SME rule of thumb), *fitted* (picked so that one observed run passes) or *arbitrary* | a provenance column in docs/knobs.md; share of fitted + arbitrary falls; fitted constants re-checked on holdout seeds |
| **invariants vs competence** | Invariants hold on every run, whatever the axes: no wedge, no hang longer than the stall ladder, every death recovered, bounded deaths per 1000 kills, the character ends in town. Competence is how well: XP/h, kills/h, time to Andariel | invariant violations per 100 virtual h = 0 across the whole sweep, reported apart from the competence metrics |

The plain-language answer for the user: **"We have a good engine and a fitted policy. The goal is a policy that transfers."** Do not say "framework vs stencil". Say what a claim covers, how it transfers, and what produced each constant. The first is about effort; the second is checkable.

## 4. Target architecture (incremental, no rewrite)

| layer | exists today | gap | smallest first step | do NOT build yet |
|---|---|---|---|---|
| world/character model (perceive -> state) | `internal/mem`, `d2dworld`, `bot/observe.go` (969 lines), snapshots | no derived features (power, zone danger, distance to town, area kind); policies read raw snapshot fields | derived area attributes from levels.txt and lvlwarp: area level per difficulty, outdoor or dungeon, hops from town, waypoint | a full world model or a belief state |
| knowledge (data) | 33 game tables, `internal/plandata` (builds, progression, runs, quest points) | area sets and boss floors copied into code | replace the 5 area sets with one derived, act-keyed tiering; read the boss floors from runs.json only | new schemas for Nightmare and Hell |
| goal layer | `goals/manager.go`, `task/campaign.go`, the act campaigns, `bot/plan.go` (progression tables) | campaigns are Go code per act; fine while each act is new work | none now; keep campaigns as code until a second act runs on d2d | a generic quest DSL |
| readiness / risk | none (ADR 0011; card m3-readiness-model deferred) | fixed clvl floors decide areas and bosses | the m3-readiness-model card as planned, calibrated on the logs and scenario corpus (below) | a learned model; per-class danger weights before a second class runs |
| resource economy | `resource.KitShortfall`, `bot/broke.go`, `kitgate.go`, `goldsweep.go`, poverty-sell knobs | special-case rules (each a fix for one soak symptom); gold starvation unresolved (median gold about 100, S navsoak-compare) | write the economy as a budget: income per virtual h (kills, sells) against spend (potions, scrolls, repair, revive), and a *measured* kit line per clvl from the logs | an optimizer over the budget |
| build/class | leveler build-*.json, `combat/classdata/profiles.json`, belt plan per class | class if-chains (11 files); silent Sorceress fallback; gear weights caster-shaped (S smoke-paladin #9) | fail loud: a class without profile, build, belt plan or gear weights gets a startup WARN, never the Sorceress table | the other 5 classes |
| invariants / watchdog | stall ladder (`bot/stall.go`), death rule (`bot/deathrule.go`), supervisor exit 5, townwedge fix, panicsafe | no list of invariants and no per-run counter of violations | one invariant counter block per run in the d2dsoak report (wedge signature, stall exits, supervisor restarts, unrecovered deaths, deaths/1000 kills) | new watchdogs before a counter shows a violation |
| evaluation | `cmd/d2dsoak` (gate v2), `tools/ab/paired.py`, docs/ab-design.md, replay (2 fixtures) | no holdout seeds, no coverage matrix, no corpus of stuck situations | holdout seed list; a "proven on" line in results; 4 recorded scenarios as offline tests | a continuous-integration soak farm |

Self-calibration: each *fitted* constant gets one *measured* replacement from the soak logs. The kit line comes from gold-per-hour at each clvl; the stall windows from the distribution of honest gaps between kills; the champion thresholds from life lost per champion fight. Only where a log exists, and each replacement checked on holdout seeds.

## 5. Proving generality cheaply (ranked by information per cost)

| # | experiment | cost | reveals |
|---|---|---|---|
| 1 | **holdout sweep inside the M2 gate run**: d2d-replicated-soak's 8 seeds x 3 virtual h use seeds never used in a diagnosis (16 to 23), with the invariant counter block | ~0 extra lab (the run happens anyway); 0.5 agent-day for the counters | whether the long-run behaviour beyond seed 5 holds: wedges and stalls per run, spread of deaths per 1000 kills |
| 2 | **scenario corpus** offline: the town wedge pocket (navsoak (145,68)), the clvl-7 Burial Grounds pack death, the seed-8 poverty loop, the kit-less recovery | 1 agent-day, no lab | turns four fixed symptoms into end-state regression tests (roadmap 1.3: same bug fixed four times) |
| 3 | **area ablation** offline: derive the 5 area sets from levels.txt and lvlwarp, diff against the hand-written sets on Act 1 (must match or be explained), then print what the rule picks for Act 2 | 0.5 agent-day, no lab | whether the safety knowledge is derivable; the first Act 2 transfer answer without a run |
| 4 | **start-state transfer**: m2-latebase's cheat-levelled clvl 12 (decided D5) plus one Act 2 start at clvl 18 (act cheat V-03), 2 seeds x 1 virtual h, ledgered | 0.5 agent-day + 2 slot-hours | what breaks in Lut Gholein (town table, field choice, merc); the start-state axis |
| 5 | **second class: Paladin** on d2d, data only, fresh, 2 seeds x 2 virtual h | 1 agent-day + 4 slot-hours | the class axis: leveler, left/right binding, kite on melee, gear weights; first check d2d melee fidelity on one fight |
| 6 | Nightmare entry (cheat-levelled) | later | out of scope until M3 (roadmap D2) |

## 6. Sequenced plan against the roadmap (cards listed here, not created)

Rule of two: generalize a piece only when a second axis value actually runs through it. Generality work rides inside milestones where it can.

| card (proposed) | milestone | one line | owns / exclusive / gates (sketch) |
|---|---|---|---|
| **gen-invariants** (FIRST) | M2, before d2d-replicated-soak | per-seed invariant counters in the d2dsoak report + holdout seed convention; delta on soak-gates `Invariants` | owns `internal/d2dsoakstat/**`, `cmd/d2dsoak/**`; no slot; gates: counters on the soakfull logs, navsoak wedge detected, validate --strict |
| gen-corpus | M1/M2 | 4 recorded scenarios as offline end-state tests (section 5 #2) | owns `internal/bot/scenario_test.go`, `internal/bot/testdata/scenarios/**`; gates: tests fail on the pre-fix commits where available |
| gen-areatiers | M2 (feeds m2-roam, m2-tripwp) | one derived act-keyed area tiering replaces the 5 sets; Act 1 output identical | owns `internal/bot/{broke,kitgate,recoveryxp}.go`, `internal/resource/town.go`, new `internal/data/areatier.go`; gate: Act 1 sets equal, Act 2 list printed |
| gen-knobprov | M2 (after m3-routeguard frees docs/knobs.md) | `Provenance` field on `env.Knob`, knobdump column, warn-only lint | owns `internal/env/tuning*.go`, `tools/knobdump/**`, `docs/knobs.md` |
| gen-claims | M2 | README line "a gate names its axes; outside them UNPROVEN"; `result:` carries "proven on:"; coverage matrix doc seeded from the archive | owns `openspec/README.md` line, `docs/coverage-matrix.md` |
| gen-classguard | M3 | startup WARN when a class lacks profile/build/belt plan/gear weights; drop the silent Sorceress fallback; move `rebuy.go:126` into profile data | owns `internal/leveler/prefs.go`, `internal/town/rebuy.go`; delta: bot-loop log line |
| gen-actstart | M3, beside m2-latebase | Act 2 start transfer probe, findings only | exclusive `soak-slot:n`; gate: report with invariant counters |
| gen-paladin | new milestone **MT transfer** between M3 and M4 | Paladin on the Act 1 loop with data-only changes | exclusive `soak-slot:n`; exit of MT: the Act 1 claim proven on >= 8 holdout seeds and Den + Blood Raven done by a second class without policy code edits |

m3-readiness-model stays as planned; its calibration should use gen-corpus and gen-areatiers outputs (power/danger inputs). Proposed roadmap edits (coordinator): add MT to section 3, gen-invariants before card 12, gen-areatiers before m2-roam.

## 7. SME alignment (2026-10-31)

| point | SME | this study | resolution |
|---|---|---|---|
| shared vs class-specific | d2-sme: potions, TP/ID, vendors, waypoints, quests, XP penalty, resist penalties, merc tables are shared data; attack model, mana economy, kite vs tank, stats, gear priorities are class-specific | same split = knowledge vs behaviour | agree |
| Nightmare | d2-sme: weakest-modelled difficulty (Normal chicken, Hell potion policy, `fighter.go:254`) | counted 6 Nightmare refs vs 21 Hell | agree; listed, not fixed (D2 parks Nightmare) |
| second class | d2-sme: Paladin (most different play model; Necro would "pass for the wrong reason"); classes/ranking.md also ranks Paladin second | Paladin, after a one-fight d2d melee fidelity check (riskmap: units/damage paths mostly unverified upstream) | agree, with the fidelity caveat added |
| readiness | d2-sme: ADR 0011 is right; needs a per-class danger view; keep clvl as a sanity floor | same; per-class weights only when the second class runs | agree (ADR 0011 already keeps floors as the safety net) |
| merc code | d2-sme cited smoke-paladin #11 "no hire code" | stale: `internal/merc` exists (866 lines) and m3-merc is active | disagree with that line only |
| coverage as a capability | spec-sme: DISAGREE with a new `coverage` spec (it would copy owned numbers and go stale); use a "proven axes" clause in existing gate requirements, a README convention and a generated matrix doc | adopted: gen-claims, no new capability | agree |
| invariants in a spec | spec-sme: yes once X has a source | X = the stall ladder knobs (`bot.stallExitMin`) and death-rule numbers already own it | agree; gen-invariants carries the delta |
| knob provenance | spec-sme: generator column fed by a field in code, lint later; not a spec | gen-knobprov | agree |
| cards as folders | spec-sme: list in the doc; folders only with concrete gates and owners | listed here only | agree |

## 8. Risks and what we should NOT do

- Risk: generality becomes a detour from M2. Mitigation: gen-invariants and gen-areatiers ride on M2 cards; MT comes only after M3.
- Risk: a Paladin failure is d2d, not bot (melee and summon paths are less verified upstream than casts). One-fight fidelity check first.
- Risk: holdout seeds lower the M2 numbers. That is information, not regression: the seed-5 numbers were optimistic.
- Do NOT rewrite into a planner, a behaviour tree or a learned policy. Do NOT add a `Policy` interface with one implementation. Do NOT make all 7 classes or Nightmare/Hell work now. Do NOT add knobs as a fix (each fitted knob widens the overfit surface: prefer a derived or measured rule). Do NOT tune on the seed that showed the symptom. Do NOT add a `coverage` capability spec.
