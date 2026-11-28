# Lab night results, 2026-11-28: triage of the three Proving Ground jobs

> status: current (card labtriage, TASK-1101; docs and scenario data only, no game run by this card). Consumer: townflip (TASK-972), bsmnight, navbarracks, drillskills. Command plan: `lab-night-commands-2026-11-27.md`.

Tags: V = read in a run's `results.json`, bot log or journal under `provingground-out/` of the lab checkout, or in the code named; S = from a scenario note or an older doc; U = unverified.

All rows ran on bot commit `2f2e092c3` (V: `"version":"2f2e092-dirty"` in every bot log, `bot_commit` in the ledgers), that is before the fixes merged the same night (fix1, fix2, fix3, fix6, fix7). Every setup is CHEATED (position, level, items by debug verbs): the rows test mechanics, never balance or parity. `writes_after_start` is 0 in the ledgers read (V).

Classes of a non-GREEN row: (a) product bug, (b) stale or wrong scenario expectation, (c) harness or setup, (d) expected RED.

## Corrections to the first reading of the logs

1. Job 2 was read as 20 GREEN, 0 RED, 1 BLOCKED. V: it has 23 GREEN, 5 RED, 1 BLOCKED. `knobs3way.sh` prints only the last three lines of each run, so the four RED rows of `nav.barracks-return` show only as `GREEN 0  RED 1` count lines, and the first `cain.chain` repetition of seed 11 was RED (the determinism pair differed hard).
2. Job 2 was cut: the seed 3 half of the nav A/B has one area of five (`nav.bloodmoor-coldplains`); the log ends inside it.
3. Job 3 never reached its last line (the eight `necro.drill.act2.*` rows did not run).
4. Job 1 has 10 GREEN when the three determinism repetitions are counted, 13 RED, 2 BLOCKED, 2 PENDING.

## Job 1: board SM, town.plan stages 4 and 5, riders

| scenario | class/seed | status | class | evidence (V unless tagged) | fix |
|---|---|---|---|---|---|
| board.sm.restart-mid-chain | Paladin 11, Sorceress 3 | RED | b | `quest board: step Take the Inifuss bark` is logged each time the chunk is entered again after a town trip: Paladin 23:00:17 and 23:01:18 (state `resupplying` between them, one process), Sorceress 5 lines. The assertion `max 1` reads that as a repeated step. Sorceress also ended with the chunk `in_progress` (mana 0 to 6 of 35 in the minute lines). U: whether the restart itself happened (one `d2bot start` in the kept log) | card bsmnight (TASK-1106): assert a once-per-step line |
| board.sm.stop-cascades | Paladin 11, Sorceress 3 | RED | b | the api trigger `after_s` 60 fired after the scroll chunk was done (start 22:59:56, `awaiting hand-in of search_for_cain/read` 23:00:15, stones started 23:00:16): reply 409 code `done`, nothing to cascade | card bsmnight: tie the cancel to a board state |
| board.park.persist-restart | both | PENDING | c | not run by design (scenario marked pending by bsmprep) | none here |
| town.plan.belt-refill-from-pack | both | BLOCKED | c | `setup: debug give cap pack: d2d: no room` | fixed by fix6 (pack room); rerun |
| town.plan.sell-keeps-need | Paladin 11 (3 left), Sorceress 3 (6 left) | RED | b | the 16 potions were drunk, not sold: `potionsUsed` 23 in the last minute line, `packHP` 16, 13, 12 ... in `shop: potion plan`; the only `sell: sold` lines are a Short Staff and Leather Gloves | FIXED here: the end count is replaced by `log_line ^sell: sold .*item=[^=]*Healing Potion` max 0; rerun |
| town.plan.gold-short-priorities | both | RED | d | with 228 gold the bot bought a Scroll of Town Portal (100) before healing potions; no `verdict=short-gold` line; gold 325 at the end. The scenario note says the priority table is stage 7 (S, card town7), not built | none until town7; card townnight (TASK-1108) records it |
| town.plan.one-visit | both | RED | a | the bot logged `town plan: route kept the list (no preset position for cain, chore identify)` (internal/town/route.go:274) instead of `town plan: route stops=` | card townnight |
| town.plan.short-gold-handover | both | RED | b | the one failing assertion matched attention lines of the tapped quest: `too weak for Go to the Barracks; 3 failed attempts` (Paladin), `boss not found in the arena: needs quests.layoutRead 1` (Sorceress). The other assertions passed | FIXED here: pattern narrowed to `needs_attention: .*\b(gold|purse|afford)` (U the gold wording); rerun |
| town.plan.malus-errand-gold | Paladin 11 | RED | a | `quest.tools_of_the_trade/talk-charsi/home ran but is still needed (evidence 2)` repeats; quest record 3 stays 0. U: fork or bot | card townnight |
| town.plan.determinism | Paladin 11, 3 repetitions | GREEN | - | same verdict three times; the state digest drifts and the first divergence is BOT-SIDE at tick 822 and 861 (a different read order on the same state). Drift is reported, not a failure | U: worth a look by the determinism owner |

GREEN: town.plan.scroll-hoard (both classes), town.plan.broke-after-bank, cain.read, cain.read.stepexec, points.bank-by-default, town.resupply.

## Job 2: Cain chain experiment, nav A/B

| scenario | class/seed | status | class | evidence | fix |
|---|---|---|---|---|---|
| cain.chain, determinism pair | Paladin 11 | rep1 RED, rep2 GREEN | a | rep1: `quest_bits record 4 set 0x0001 clear 0x0002, got 0x0008`; first divergence BOT-SIDE at tick 3974 (rep1 reads `debug presets` once more on identical state). The flake of TASK-680 is reproduced, not settled | TASK-680 stays open; card cainspin (TASK-1103) cites it |
| cain.chain with `bot.portalRetryWaitSec` (0) | Paladin 11 | BLOCKED | a | 2108 s wall, bot log 2.7 million lines; the last 200000 lines are a cycle of `roam: done`, `area cleared`, `run finished`, `gold ledger` (28569 each); `d2d: leave: re-entered the game` from line 1485. The wall cut is the consequence, not the cause | card cainspin |
| nav.barracks-return | Paladin 11, arms off, grid, gridwarp, all | RED x4 | a | identical in all arms: `d2dworld: arrived in level 28: 0 adjacent entries`, 44 x `no route from Barracks (28) to Outer Cloister (27): nav: no exit to that level`, failsafe rung 4, pause after 15 s wall. With `nav.warpRead` 1 the warptriggers verb was read 18 times and the route was still missing. The off arm is RED too, so this is not a knob regression | card navbarracks (TASK-1102) |

GREEN: cain.chain seeds 12 and 13; nav.bloodmoor-coldplains, nav.inner-cloister-cathedral, nav.catacombs-hop-33-34, nav.catacombs-hop-34-35 in all four arms for seed 11; nav.bloodmoor-coldplains in all four arms for seed 3.

### Do the A/B arms really differ? (V, journals)

| area (seed) | `debug warptriggers` reads off/grid/gridwarp/all | `debug levelmap` reads | journal lines off/grid/gridwarp/all | `nav stuck` lines off/grid/gridwarp/all |
|---|---|---|---|---|
| bloodmoor-coldplains (11) | 0/0/33/33 | 0/0/0/33 | 73327/73316/73476/73530 | 4/4/4/4 |
| barracks-return (11) | 0/0/18/18 | 0/0/0/18 | 60611/60611/59825/59896 | 0/0/0/0 (all RED) |
| inner-cloister-cathedral (11) | 0/0/34/34 | 0/0/0/34 | 24802/24940/24872/25055 | 4/4/4/4 |
| catacombs-hop-33-34 (11) | 0/0/13/13 | 0/0/0/13 | 84373/42142/42254/42202 | 32/36/36/36 |
| catacombs-hop-34-35 (11) | 0/0/23/23 | 0/0/0/23 | 76287/68947/67478/67593 | 9/9/12/12 |
| bloodmoor-coldplains (3) | 0/0/37/37 | 0/0/0/37 | 72187/71659/71411/71770 | 8/8/8/8 |

`nav.warpRead` and `quests.layoutRead` are proven on by their verbs. `nav.fullGrid` has no verb of its own: it shows as a different journal in four of six rows (half the journal on hop 33-34) and as an identical line count on barracks-return (the run never moved). So the arms differ (V for the two reads, S for the grid).

## Job 3: Act 2 stepexec, necro, arbiter soak, combat drills

| scenario | class/seed | status | class | evidence | fix |
|---|---|---|---|---|---|
| drill.act2.duriel.stepexec | Paladin 11, 0 of 3 | BLOCKED | a | `quest board: starting chunk seven_tombs/duriel`, then `Waiting for a fresh game: no eligible area is left in this game`, `campaign stalled: fresh game failed: attached d2d cannot be restarted`, exit after 60 virtual min. The harness cannot give a fresh game, but the bot found no way to the chunk first | card act2night (TASK-1107) |
| drill.act2.cube.stepexec | Paladin 11 | RED | a | invariant `potion-not-drunk` 2 hits: `life=35% line=50% beltHP=4 drank=0 for=18s hostiles=4` | card act2night |
| drill.act2.shaft.stepexec | Paladin 11 | RED | c | `action failed to town: no town portal available`, `horadric_staff/shaft: not ready: early road (no town portal for the act2_maggot_lair road)`; the Shaft (`msf`) was never reached. U: the setup lacks a scroll, or the hero should have bought one | card act2night: give the setup a portal scroll |
| drill.act2.summoner.stepexec | Paladin 11 | BLOCKED | a | after 647 s wall: `nav: stuck recovered method=replan rung=3 repeat=true`, then `watchdog: runner loop made no progress, exiting for a restart age=614s limit=600s` and a goroutine dump. U: the watchdog age is wall clock; the slot ran at load 7 | card summonerstall (TASK-1104) |
| combat.whirlwind-clears | Barbarian 11 | RED | b + c | marker wrong and no whirl leg at all, see below | marker FIXED here; card drillskills (TASK-1105) |
| combat.hold-interruptible | Paladin 11 | RED | b + c | the asserted lines `combat cast ... skill=Zeal` and `combat attack ... Zeal` do not exist; every `combat kill` carries `last_skill=Attack` | marker FIXED here (`^combat kill .*\blast_skill=Zeal\b`); card drillskills |
| combat.teleport-reposition | Sorceress 11 | RED | b + c | marker wrong; no `combat move` line; `combat out of mana: backing off`, `watchdog cannot cast`, minlife 100 in every fight | marker FIXED here; card drillskills |
| combat.corpse-explodes | Necromancer 11 | RED | c | the marker `^combat corpse-explosion ` is right (aimcorpse.go:260); the line was never logged | card drillskills |
| necro.corpse-ce1, necro.corpse-ce5 | Necromancer 11, 12, 13 | RED x6 | c | `minions 0` is a real read (`debug minions` is in the journal) and no `combat minion` line exists: the hero never summoned | card drillskills |

GREEN: drill.act2.amulet.stepexec, drill.act2.radament.stepexec, drill.act2.radament.handin.stepexec, necro.act1-stage1, combat.whirlwind-clears.control.

### The combat drill markers (V, code and logs)

`log_line` matches `<msg> k=v ...` with the attributes SORTED by key (internal/scenario/logline.go), and every bot line carries `inst`. `whirl.go` logs `f.info("move", "use", MoveWhirl, "skill", ..., "to", ..., "why", ...)`, which renders as `combat move inst=... skill=... to=... use=whirl why=...`. The patterns `^combat move use=whirl ` and `^combat move use=escape ` could never match, and `^combat trap skill=` neither (`dist` and `inst` sort first). Fixed in `combatdrills.json`:

| old | new |
|---|---|
| `^combat move use=whirl ` (and the control's `max 0` form) | `^combat move [a-z_]+=.*\buse=whirl\b` (the `[a-z_]+=` keeps `combat move shadow` and `combat move failed` out) |
| `^combat move use=whirl .*life (...)%` | `^combat move [a-z_]+=.*\buse=whirl\b.*life (...)%` |
| `^combat move use=escape ` | `^combat move [a-z_]+=.*\buse=escape\b` |
| `^combat cast .*skill=Zeal\|^combat attack .*Zeal` | `^combat kill .*\blast_skill=Zeal\b` |
| `^combat trap skill=` | `^combat trap .*\bskill=` |

`d2scenario validate`: ok, 266 scenarios. The markers alone do not turn the rows GREEN: in all ten RED rows of this block the bot log has no `leveler: plan`, `stats: allocated` or `skills: allocated` line (the nav runs with `hero_profile` banked have them), `class-aware start` reports `combatBuild=blizzard` for the Barbarian and the Paladin, and the heroes fight with plain Attack. U: whether the setup spends the points and leaves the bot nothing to learn, or nobody does (`cmd/d2scenario/smokegame.go` notes that `debug points` may change nothing). Second fault of the encounter: ten to fourteen Fallen do not hurt a clvl 24 to 30 hero (minlife 100), so the escape, potion and life-floor triggers cannot fire.

### Arbiter soak (`arbiter-control-families-255`, Sorceress 201, 1 h)

Both arms ended at clvl 12 with 0 deaths. Invariant lines: base 16 (field-entry-no-kit 9, guard-stall 7), arbiter arm 12 (guard-stall 7, field-entry-no-kit 3, stuck-in-pack 1, arbiter-starved 1). Kills 340 base against 240 arbiter. Fewer hits, but two kinds the base does not have (V counts; one seed, so S as a comparison).

## Counts

37 non-GREEN rows: (a) 12, (b) 12, (c) 11, (d) 2. Rows fixed on the scenario side here: 8 (sell-keeps-need x2, short-gold-handover x2, four combat markers), each needs a rerun; four of them still wait for card drillskills.

## Dark-ledger verdicts (criteria text from `data/darklaunch.json`)

No row's criteria are fully met by this night, so there is no closing `flip.py` line to run.

| row | criteria | night evidence | verdict |
|---|---|---|---|
| `nav.fullGrid` | A/B nav drill: no stuck regressions on 5 areas | 4 areas GREEN in both arms (seed 11); `nav stuck` 32 to 36 on hop 33-34 with the grid on; the fifth area RED in both arms; seed 3 has one area | NOT met (4 of 5 areas, one small regression on one seed) |
| `nav.warpRead` | Warp drill across 5 areas with the read on | read proven on in 5 areas, 4 GREEN; `nav stuck` 9 to 12 on hop 34-35; barracks-return RED with the read on | NOT met (4 of 5) |
| `quests.layoutRead` | Quest layout drill: quest objects found by read in 3 areas | the nav drills read the level map but find no quest object; no quest layout drill ran | NOT met (no evidence of the right kind) |
| `quests.stepExec` | four scenarios green 2 of 3 each | cain.read.stepexec 1 GREEN of 1; drill.act2.duriel.stepexec 0 of 3 | NOT met |
| `quests.boardSM` | restart drill green with boardSM 1 (a park survives) | restart-mid-chain RED on both classes (class b, so no verdict about the product), persist-restart PENDING | NOT met, U |
| `town.plan` (stages 4, 5) | stage scenario set green | stage 4: scroll-hoard and broke-after-bank GREEN, sell-keeps-need and belt-refill need a rerun, malus RED; stage 5: all three RED | NOT met |
| `combat.minionRead` | Necromancer live drill: census matches the fork read | the read works (verb in the journal) but the hero had no minions | NOT met, U |
| `combat.deliveryUpkeep` | combat.buffs-stay-up green for two classes at mode 2 | not run; the mode 2 drivers never fired (drillskills) | NOT met |
| `bot.arbiterControl`, `bot.arbiterFamilies`, `bot.arbiterShadow` | fewer monitor hits than the chain and no new ones; incident rows green | 12 against 16 hits, two new kinds | NOT met |

Closing lines to use AFTER a rerun shows 5 of 5 areas GREEN with no added `nav stuck` (not now; instance and wording are the coordinator's):

```sh
python3 tools/flip/flip.py <instance> nav.fullGrid=1 --reflip --why "A/B nav drill: 5 areas, no stuck regression (lab night rerun)"
python3 tools/flip/flip.py <instance> nav.fullGrid=1 nav.warpRead=1 --reflip --why "warp drill: 5 areas green with the read on (lab night rerun)"
```

## Rerun list (ready to paste after cards navbarracks and drillskills, and with the fixes of this night on the lab checkout)

```sh
D2SCEN_KNOBS=town.plan=4 tools/provingground/run.sh -id town.plan.belt-refill-from-pack,town.plan.sell-keeps-need -class Paladin -seed 11 -parallel 2
D2SCEN_KNOBS=town.plan=5 tools/provingground/run.sh -id town.plan.short-gold-handover -class Paladin -seed 11 -parallel 1
tools/provingground/knobs3way.sh nav.barracks-return Paladin "11 3" off= grid=nav.fullGrid=1 gridwarp=nav.fullGrid=1,nav.warpRead=1 all=nav.fullGrid=1,nav.warpRead=1,quests.layoutRead=1
```

Tooling note for the owner of `knobs3way.sh` (not owned by this card): print the result rows, not `tail -3`, so a RED arm is visible in the job log.
