# Lab night plan, 2026-11-27 (23:00 to 07:00 EST)

> status: current (docs only; no game, no lab run made by this card). Tags: V = read in the repo at origin/main 47e04bea8 (file named), S = from a probe doc, U = unverified. Sources: `docs/probes/dark-queue-final-2026-11-27.md` (flip lines, gates G1 to G4), `tools/provingground/run.sh`, `tools/provingground/knobs3way.sh`, `tools/provingground/scenarios/*.json`, `docs/probes/act2-stepexec-drill-2026-11-27.md`, `docs/probes/feature-soak-2026-11-26.md`.

## Rules in force (ADR 0027 amended 2026-11-24, private ops rules)

- At most 3 heavy runs tonight, hidden instances only (lab slots of `tools/d2dslot.sh`, never i0, never a desktop window), `nice -n 19`, the sandbox containers untouched (no flip is made by these runs; flips are the coordinator's, after the morning read).
- Before each run start: `labrun orion -- top -bn2 -d1` must show idle plus nice at least 25 percent; if not, wait or run the next run serially. The capacity check `tools/capacity.sh --add <cores>` is the private-ops form; it hangs here, so use the `top` line above (standing rule).
- Every result record carries `writes after start: N`. N above 0 is CHEATED and never counts. Proving Ground heroes are SETUP-written throwaways (CHEATED provenance): the results prove mechanics and relative A/B, never live parity (S act2 drill doc).
- Every process starts through `tools/agentrun.sh <agent> --ttl SECS -- ...`; the TTL below is the hard stop.

## Facts that shape the plan

- V `run.sh` runs ONE class and ONE seed per call (`-class`, `-seed`); `-id` takes a comma list and prefixes ending in `.`; `-repeat N` repeats (GREEN needs 2 of 3). A bundle is therefore a shell chain of calls inside one labrun job, so one job is one "run" in the three-run budget.
- V `D2SCEN_KNOBS="k=v,k=v"` layers numbers over a scenario's own tuning (`cmd/d2scenario/run.go` envKnobs), so `town.plan=4` can be forced on the stage 3 scenario files.
- S measured speed: about 3 virtual minutes per wall minute for a Paladin field row on a loaded host, casters about 6, town scenarios faster (soak doc section 2). Wall times below use 3 for field work and are U.
- NOT in the tree: the six new `drill.act2.{radament,radament.handin,cube,shaft,amulet,summoner}.stepexec` exist only on the unmerged branch `agent/act2chunks` (tip 6992ea23f, V `git grep`); main has only `drill.act2.assemble.stepexec` and `drill.act2.duriel.stepexec`. They run only after that branch is merged (coordinator).
- NOT in the tree anywhere: `combat.hold-interruptible`, `combat.teleport-reposition`, `combat.corpse-explodes` (V grep of scenarios, 0 hits; only design cards task-728/730/731/732 name them). Candidate 8 is dropped.
- `cain.read` (candidate 7) is a fast scenario, but the Akara hand-in fix card is not built (diagnosis only, `akara-handin-click-2026-11-27.md`): the run only gives a baseline, nothing to confirm. Cheap rider in Run 1.

## Ranking by value

| rank | candidate | owner card or row | unlocks | cost (U) |
|---|---|---|---|---|
| 1 | G1 board restart + G2 town stage 4/5 | TASK-983 `quests.boardSM`; `town.plan` 4 and 5 | 2 flips (boardSM on Questy, town.plan 4 on Doofer) and closes stage 5 evidence | about 3 h |
| 2 | G3 nav A/B, 5 areas | `nav.fullGrid`, `nav.warpRead`, `quests.layoutRead` | 3 flips in one line on Doofer | about 2.5 h |
| 3 | Act 2 stepexec: Duriel x3 plus six drills | TASK-976 AC 4, `quests.stepExec` Act 2 | stepExec Act 2 evidence; Duriel past the Tyrael park | about 4 h (needs act2chunks merged for 6 of 8) |
| 4 | arbiter 255 Sorceress 201 rerun | TASK-1030/1048, `bot.arbiterFamilies` 255 | confirms "stay on" on Doofer | about 40 min |
| 5 | necro.act1-stage1 | TASK-1033 AC 5 | closes the last live box of necroteeth | about 50 min wall (budget) |
| 6 | points.bank-by-default, town.resupply, cain.read | regression and baseline | no flip | about 20 min |
| 7 | combat drills | none exist | nothing | dropped |

## The three runs

Run 1 and Run 2 start together at 23:00 if the capacity line allows (parallel 2 each, so about 4 slots); otherwise Run 2 starts when Run 1 ends. Run 3 starts when the first of them ends and the capacity line passes again. Latest start for Run 3 is 03:30 (its TTL ends 07:00).

Common env for all runs (set once in the job, paths from the previous drill, S act2 drill doc): `PG_DIR=<work dir> PG_D2D_BIN=<featuresoak lab d2d, devp29> PG_DATA=<MPQ dir>`; do not change the lab d2d between runs. Job form (one per run; `--detach --notify --pull` returns at once, `labrun jobs` shows it):

```
tools/agentrun.sh nightplan-N --ttl SECS -- labrun orion --lane long --detach --notify --pull provingground-out -- nice -n 19 sh -c '<chain>'
```

### Run 1: gates G1 + G2 (flips: `quests.boardSM`, `town.plan` 4; evidence for 5) TTL 15000 s (4 h 10 min)

What it proves: the board state machine survives a bot restart mid chain and parks cleanly (G1, TASK-983); the stage 4 sell and pickup rules and the stage 5 rules hold on the stage 3 files when forced to the higher stage (G2, townflip-p5 4b). Class and seed: Paladin 11 and Sorceress 3 (the matrices of these files, V). Success: each scenario GREEN, hard invariants 0, `writes after start: 0`; `board.park.persist-restart` may fail in its setup (S: not seen to build, a scenario fault is not a finding, record it as such).

```
sh -c 'R=tools/provingground/run.sh
for cs in "Paladin 11" "Sorceress 3"; do set -- $cs
  $R -id board.sm.restart-mid-chain,board.park.persist-restart,board.sm.stop-cascades -class $1 -seed $2 -parallel 2
  D2SCEN_KNOBS=town.plan=4 $R -id town.plan.belt-refill-from-pack,town.plan.sell-keeps-need,town.plan.scroll-hoard -class $1 -seed $2 -parallel 2
  D2SCEN_KNOBS=town.plan=5 $R -id town.plan.gold-short-priorities,town.plan.short-gold-handover,town.plan.one-visit -class $1 -seed $2 -parallel 2
  $R determinism-run -n 3 -id town.plan.determinism -class $1 -seed $2 -parallel 1
done
$R -id points.bank-by-default,town.resupply,cain.read -class Paladin -seed 11 -parallel 2'
```

Expected wall (U): board 3 x 30 virtual min, town 6 x 15 to 40 virtual min, determinism 3 x 20, per class pair about 1.3 h; the tail rider about 20 min. Owner look at stage 4 (ADR 0028) is still needed before the flip (S final doc G2): the run only supplies the scenarios.

### Run 2: gate G3 nav A/B, 5 areas (flips: `nav.fullGrid`, `nav.warpRead`, `quests.layoutRead`) TTL 12600 s (3 h 30 min)

First (needs no slot, 1 minute): check the lab d2d caps for the verbs the arms use, `debug warptriggers` and `debug levelmap` (U per final doc G3). If one is absent, drop that knob from the arms below and say so in the result.

What it proves: with the map-knowledge knobs ON no scenario is BLOCKED or RED that the `off` arm lacks. Scenarios: the five navexit ids of the final doc (7 exist, V `navexit.json`; the two further `nav.catacombs-hop-35-36` and `-36-37` are optional, add them if the night is quiet). Class Paladin, seeds 11 and 3. Arms: `off`, `grid` (nav.fullGrid), `gridwarp` (+ nav.warpRead) are the script defaults (V); `layoutRead` is not in the defaults, so pass a fourth arm.

```
sh -c 'for id in nav.bloodmoor-coldplains nav.barracks-return nav.inner-cloister-cathedral nav.catacombs-hop-33-34 nav.catacombs-hop-34-35; do
  tools/provingground/knobs3way.sh $id Paladin "11 3" off= grid=nav.fullGrid=1 gridwarp=nav.fullGrid=1,nav.warpRead=1 all=nav.fullGrid=1,nav.warpRead=1,quests.layoutRead=1
done'
```

Expected wall (U): 5 ids x 2 seeds x 4 arms x 10 virtual min at about 3 = about 2.7 h serial (knobs3way is serial by design, V). Success per id and seed: ON arms (`grid`, `gridwarp`, `all`) verdict equal or better than `off`; any ON-only BLOCKED or RED is a finding that keeps that knob dark. Output dirs `provingground-out/k3-<time>/<arm>-Paladin-<seed>`.

### Run 3: Act 2 stepexec, necro stage 1, arbiter rerun TTL 21600 s (6 h; start by 03:30)

Precondition: `agent/act2chunks` merged into the bot tree that the job syncs (coordinator, tonight before 23:00); else drop the six drills from the chain and keep Duriel plus assemble.

```
sh -c 'R=tools/provingground/run.sh
$R -id drill.act2.duriel.stepexec -class Paladin -seed 11 -repeat 3 -parallel 1
$R -id drill.act2.radament.stepexec,drill.act2.radament.handin.stepexec,drill.act2.cube.stepexec,drill.act2.shaft.stepexec,drill.act2.amulet.stepexec,drill.act2.summoner.stepexec -class Paladin -seed 11 -parallel 2
$R -id necro.act1-stage1 -class Necromancer -seed 11 -parallel 1'
python3 tools/provingground/featuresoak.py run <copy of tools/provingground/campaigns/features-2026-11-26.json> --features arbiter-control-families-255 --classes sorceress --seeds 201 --hours 1
```

The last line is a second command in the same job (same TTL); run `featuresoak.py plan` first for its wall cap (about 42 minutes at ratio 3, formula hours*60/R*1.6+10, S soak doc). Use the same lab d2d as the campaign file names. Expected wall (U): Duriel 3 x 120 virtual min is about 2 h, the six drills (345 virtual min, parallel 2) about 1 h, stage 1 up to 50 min (its own `wall_min`), arbiter about 40 min.

What each proves and success:
- Duriel x3: Tyrael is now opened in Duriel's Lair (bd66a419d). Success: the chunk no longer parks at `talk_tyrael`; the Staff placement outcome (record 10 and 14) is finally observed. Still BLOCKED at the same step means a different cause: copy the journal lines, do not rerun.
- Six drills: each GREEN with `step executor ... done` lines and the quest word expected by its design (`docs/design/stepexec-p9.md`, S); a drill that fails in setup is a scenario fault.
- necro.act1-stage1 (Necromancer 11): TASK-1033 AC 5, the hero at clvl 12 or above casts Teeth (a `cast` line for Teeth in the bot log; U exact wording, grep `Teeth`). Not a clvl 12 start: the fixture starts at 16 (V scenario), so the AC text "clvl 12" is met by the earlier part of the run only if it levels from a lower base; report what the log shows.
- arbiter 255 Sorceress 201: no `watchdog: runner loop made no progress` exit 4 (the earlier RED was the watchdog false positive, S TASK-1048); hard invariants 0.

## Morning checklist (read in this order)

1. `labrun jobs`: all three jobs ended on their own (not TTL 124). `tools/agentreap.sh --list` empty of nightplan entries; reap anything left with `tools/agentreap.sh nightplan-N` (never kill by pid).
2. Pull: `provingground-out/` is synced back by `--pull`. For each run dir read `results.json`: verdict per scenario, hard invariant count, and `writes after start`. Anything with N above 0 is CHEATED: discard.
3. Run 1: G1 needs restart-mid-chain, stop-cascades GREEN on both classes (persist-restart setup fault excepted). G2 needs the three stage 4 ids GREEN under `town.plan=4` on both classes; stage 5 ids and determinism (journals identical across 3, `d2ddet` output) close stage 5 evidence. Flip lines: final doc G1 and G2 (owner look first).
4. Run 2: build the table id x seed x arm; ON arms vs `off`. All-green flips the G3 line of the final doc; a failing knob is dropped from that line, the rest still flips.
5. Run 3: Duriel stop point; the six drills; Teeth cast line; arbiter exit code (`exit` column of the soak table).
6. Record results in a new probe doc (coordinator) and tick TASK-976 AC 4, TASK-1033 AC 5, TASK-983 as they qualify. Do not flip from the morning read without the coordinator.
7. Trouble signs: load above 28 on the worker host (rows slow, the stall rule fires falsely, S act2 doc), memory below 2500 MB (run.sh refuses, exit 75, rerun), every scenario BLOCKED at the same step (suspect the lab d2d, not the feature).

## Not run tonight, and why

- Candidate 8 combat drills: scenarios do not exist.
- G4 invdesk (`manage.*`, final doc): no slot; first in line for the next window.
- Fix verification of the Akara hand-in: the fix card is not built.
