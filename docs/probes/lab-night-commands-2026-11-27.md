# Lab night commands, 2026-11-27 (23:00 to 07:00 EST): corrected, paste-ready

> status: current (docs only; card nightcheck, no game, no lab run). Corrects and extends `lab-night-plan-2026-11-27.md` (its ranking and success rules stand) and `live-queue-2026-11-27.md`. Tags: V = read or run offline here (`d2scenario list/plan`, `labrun --help`), S = from a probe doc, U = unverified. Preflight: `tools/provingground/nightcheck.sh` (below).

## 0. Before 23:00

1. `tools/provingground/nightcheck.sh` (about 20 s, starts nothing). Add `NC_WORKER=<builder>` to test the lab d2d and data paths there (`ls -d` through the light lane), `--include-deferred` to list the scenarios of section 5. Exit 1 prints `MISSING IDS`. V result on this tree: all ids of sections 1 to 4 exist, 258 scenarios validate.
2. `labrun <builder> -- top -bn2 -d1`: idle plus nice at least 25 percent (the rule of the plan). Never use `tools/capacity.sh`.
3. Set once in the shell: `BUILDER=<builder host name of local.env>`; `LANE="--lane long"`.

## 0b. Finding that changes the plan (V `labrun --help`)

The long lane has ONE job at a time, a hard limit of 150 minutes per job, and queues up to 4 hours. The plan's three runs were sized 3.5 to 6 hours and meant to start together. They cannot run together on the long lane, and a job is cut at 150 minutes. The `d2scenario plan` upper bounds (V) of the chains below: Job 1 about 260 min, Job 2 about 450, Job 3 about 400; typical wall is about half (S plan). Therefore: (a) every chain below is ordered by value, lowest value last, and a cut loses only the tail (finished scenarios keep their `results.json`; `--pull` copies them back when the job ends); (b) if the coordinator accepts the main lane for the night (not verified: it has no stated hard limit, but merge gates have priority there), set `LANE=""` and the three jobs can run in parallel; (c) `agentrun --ttl` only bounds the submit call (`--detach` returns at once); the stop of a job is the lane limit or `labrun cancel <job>`.

Submit form (one per job): `tools/agentrun.sh nightcheck-N --ttl 300 -- labrun $BUILDER $LANE --detach --notify --pull provingground-out -- nice -n 19 env PG_D2D_BIN=<lab d2d> PG_DATA=<MPQ dir> sh -c '<chain>'`. Replace `<lab d2d>` and `<MPQ dir>` by the paths of the previous drill (present on the builder, S act2 drill doc); run.sh's defaults apply when the `env` part is dropped.

## 1. Job 1: G1 + G2, both classes, plus riders (flips boardSM, town.plan 4; evidence for 5)

New against the plan: shell variables replaced by literal ids (the preflight can read them); the six act2 drills are not here; riders added: `cain.read.stepexec` (the Akara baseline with `quests.stepExec` on, S cain doc) and `town.plan.broke-after-bank`, `town.plan.malus-errand-gold` (TASK-757 AC 3, S live-queue). Determinism only for Paladin to save 48 min (U: Sorceress determinism is then not run).

```sh
tools/agentrun.sh nightcheck-1 --ttl 300 -- labrun $BUILDER $LANE --detach --notify --pull provingground-out -- nice -n 19 sh -c '
tools/provingground/run.sh -id board.sm.restart-mid-chain,board.park.persist-restart,board.sm.stop-cascades -class Paladin -seed 11 -parallel 2
D2SCEN_KNOBS=town.plan=4 tools/provingground/run.sh -id town.plan.belt-refill-from-pack,town.plan.sell-keeps-need,town.plan.scroll-hoard -class Paladin -seed 11 -parallel 2
D2SCEN_KNOBS=town.plan=5 tools/provingground/run.sh -id town.plan.gold-short-priorities,town.plan.short-gold-handover,town.plan.one-visit -class Paladin -seed 11 -parallel 2
tools/provingground/run.sh -id board.sm.restart-mid-chain,board.park.persist-restart,board.sm.stop-cascades -class Sorceress -seed 3 -parallel 2
D2SCEN_KNOBS=town.plan=4 tools/provingground/run.sh -id town.plan.belt-refill-from-pack,town.plan.sell-keeps-need,town.plan.scroll-hoard -class Sorceress -seed 3 -parallel 2
D2SCEN_KNOBS=town.plan=5 tools/provingground/run.sh -id town.plan.gold-short-priorities,town.plan.short-gold-handover,town.plan.one-visit -class Sorceress -seed 3 -parallel 2
tools/provingground/run.sh determinism-run -n 3 -id town.plan.determinism -class Paladin -seed 11 -parallel 1
tools/provingground/run.sh -id points.bank-by-default,town.resupply,cain.read,cain.read.stepexec -class Paladin -seed 11 -parallel 2
D2SCEN_KNOBS=town.plan=4 tools/provingground/run.sh -id town.plan.broke-after-bank,town.plan.malus-errand-gold -class Paladin -seed 11 -parallel 2'
```

## 2. Job 2: Cain chain experiment, then gate G3 nav A/B

The cain experiment (A1/A2/B/C/D of `cain-chain-flake-2026-11-27.md` section 5) goes first: it settles TASK-680 AC 3. A1 and A2 are one `determinism-run -n 2` (same binary, verdict and digest compared, V `run.sh` header). D uses the knob `bot.portalRetryWaitSec` 0 through `D2SCEN_KNOBS` (V in `docs/knobs.md`; the doc proposed a scratch scenario copy, the env form needs none). Stop rule of that doc: a job cut by the wall limit is BLOCKED, never counted. Nav: seed 11 for all five ids first, then seed 3 (so a cut keeps whole seeds). `knobs3way.sh` is serial (V). `nav.fullGrid` / `nav.warpRead` need the `debug warptriggers` and `debug levelmap` verbs: check the lab d2d caps first (plan Run 2, U).

```sh
tools/agentrun.sh nightcheck-2 --ttl 300 -- labrun $BUILDER $LANE --detach --notify --pull provingground-out -- nice -n 19 sh -c '
tools/provingground/run.sh determinism-run -n 2 -id cain.chain -class Paladin -seed 11 -parallel 1
tools/provingground/run.sh -id cain.chain -class Paladin -seed 12 -parallel 1
tools/provingground/run.sh -id cain.chain -class Paladin -seed 13 -parallel 1
D2SCEN_KNOBS=bot.portalRetryWaitSec=0 tools/provingground/run.sh -id cain.chain -class Paladin -seed 11 -parallel 1
for id in nav.bloodmoor-coldplains nav.barracks-return nav.inner-cloister-cathedral nav.catacombs-hop-33-34 nav.catacombs-hop-34-35; do
  tools/provingground/knobs3way.sh $id Paladin "11" off= grid=nav.fullGrid=1 gridwarp=nav.fullGrid=1,nav.warpRead=1 all=nav.fullGrid=1,nav.warpRead=1,quests.layoutRead=1
done
for id in nav.bloodmoor-coldplains nav.barracks-return nav.inner-cloister-cathedral nav.catacombs-hop-33-34 nav.catacombs-hop-34-35; do
  tools/provingground/knobs3way.sh $id Paladin "3" off= grid=nav.fullGrid=1 gridwarp=nav.fullGrid=1,nav.warpRead=1 all=nav.fullGrid=1,nav.warpRead=1,quests.layoutRead=1
done'
```

Read from the cain jobs the log evidence table of section 5 of the cain doc (tap times, `cancelled, nothing clicked`, `portal retry`, `hand-in queued/served/dropped`, final `quest_bits`) and apply its verdict rule.

## 3. Job 3: Act 2 stepexec, necro, arbiter, combat drills (all on main)

New against the plan: `agent/act2chunks` is merged (V: the six `drill.act2.*.stepexec` are in `stepexec-act2chunks.json` on main, `d2scenario list`), so the precondition is gone; the five combat drills that exist on main are added (one class per call: `d2scenario` takes one class and one seed); the Necromancer corpse scenarios seeds 11 to 13 and the `necro.drill.act2.*` set last. The `combat.*` and `necro.corpse-*` rows need the fork lab d2d with `lab-verbs-attach` and `read.minions` (V `d2scenario plan` needs line; U whether the lab d2d of the night has them: a BLOCKED row with that reason is a harness result, not a finding).

```sh
tools/agentrun.sh nightcheck-3 --ttl 300 -- labrun $BUILDER $LANE --detach --notify --pull provingground-out -- nice -n 19 sh -c '
tools/provingground/run.sh -id drill.act2.duriel.stepexec -class Paladin -seed 11 -repeat 3 -parallel 1
tools/provingground/run.sh -id drill.act2.radament.stepexec,drill.act2.radament.handin.stepexec,drill.act2.cube.stepexec,drill.act2.shaft.stepexec,drill.act2.amulet.stepexec,drill.act2.summoner.stepexec -class Paladin -seed 11 -parallel 2
tools/provingground/run.sh -id necro.act1-stage1 -class Necromancer -seed 11 -parallel 1
python3 tools/provingground/featuresoak.py run tools/provingground/campaigns/features-2026-11-26.json --features arbiter-control-families-255 --classes sorceress --seeds 201 --hours 1 --out provingground-out/arbiter255
tools/provingground/run.sh -id combat.whirlwind-clears,combat.whirlwind-clears.control -class Barbarian -seed 11 -parallel 2
tools/provingground/run.sh -id combat.hold-interruptible -class Paladin -seed 11 -parallel 1
tools/provingground/run.sh -id combat.teleport-reposition -class Sorceress -seed 11 -parallel 1
tools/provingground/run.sh -id combat.corpse-explodes -class Necromancer -seed 11 -parallel 1
tools/provingground/run.sh -id necro.corpse-ce5,necro.corpse-ce1 -class Necromancer -seed 11 -parallel 2
tools/provingground/run.sh -id necro.corpse-ce5,necro.corpse-ce1 -class Necromancer -seed 12 -parallel 2
tools/provingground/run.sh -id necro.corpse-ce5,necro.corpse-ce1 -class Necromancer -seed 13 -parallel 2
tools/provingground/run.sh -id necro.drill.act2.radament,necro.drill.act2.cube,necro.drill.act2.shaft,necro.drill.act2.amulet,necro.drill.act2.assemble,necro.drill.act2.summoner,necro.drill.act2.duriel,necro.drill.act2.radament.handin -class Necromancer -seed 11 -parallel 2'
```

Run `python3 tools/provingground/featuresoak.py plan tools/provingground/campaigns/features-2026-11-26.json --features arbiter-control-families-255 --classes sorceress --seeds 201 --hours 1` first for its wall cap (S plan: about 42 min). If time is short, drop the `necro.drill.act2.*` and `necro.corpse-*` lines (lowest value).

## 4. Not on main yet (V `d2scenario list`; branches named from `git branch -a`)

| scenario ids | where | note |
|---|---|---|
| `combat.hammer-in-pack`, `.control`, `combat.point-spell-hits`, `.druid`, `combat.traps-stand`, `combat.army-holds`, `.control`, `combat.pets-hold` | branch `agent/combatscen2` (tip 247ec38fd, edits `combatdrills.json` and `tiers.json`) | classes: Paladin, Sorceress, Druid, Assassin, Necromancer, Druid; one seed 11, 8 virtual min each |
| the same file also changed by | branch `agent/combattelem` (assertions, doc) | both branches change `combatdrills.json`: the merge order matters, the coordinator resolves; after the merge run the block below |

Deferred block (the preflight skips it unless `--include-deferred`; run it after `combatscen2` is merged and the bot tree synced, as a rider of Job 3 or in the morning):

```sh deferred
tools/provingground/run.sh -id combat.hammer-in-pack,combat.hammer-in-pack.control -class Paladin -seed 11 -parallel 2
tools/provingground/run.sh -id combat.point-spell-hits -class Sorceress -seed 11 -parallel 1
tools/provingground/run.sh -id combat.point-spell-hits.druid -class Druid -seed 11 -parallel 1
tools/provingground/run.sh -id combat.traps-stand -class Assassin -seed 11 -parallel 1
tools/provingground/run.sh -id combat.army-holds,combat.army-holds.control -class Necromancer -seed 11 -parallel 2
tools/provingground/run.sh -id combat.pets-hold -class Druid -seed 11 -parallel 1
```

## 5. Not covered tonight

`manage.*` (G4), the Akara hand-in fix (not built: `cain.read` is a baseline), TASK-1034 AC 2 (needs `--classes necromancer --seeds 201` in the arbiter line: add it to the featuresoak call only if the night is quiet, U). Morning checklist and success rules: the plan, sections "Morning checklist" and each run's "What it proves", with Jobs renumbered (plan Run 2 is Job 2 and now carries the cain experiment).
