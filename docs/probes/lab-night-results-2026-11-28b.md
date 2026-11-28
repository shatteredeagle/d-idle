# Lab night results 2026-11-28, part b: reruns after the fixes

> status: current (lab operator labops2, no task card; docs and scenario data, lab slots only, no live game). Continues `lab-night-results-2026-11-28.md` (the triage of the first three jobs). Consumer: the coordinator (flip lines), cards bsmnight (TASK-1106), navbarracks (TASK-1102), drillskills (TASK-1105), act2night (TASK-1107), townnight (TASK-1108), cainfix3 (TASK-680), boardsidestart (TASK-1120, opened here).

Tags: V = read in a run's `results.json`, bot log, journal or ledger under `provingground-out/` of the labops2 lab checkout (run directory named in each row), or in the code named; S = from a scenario note or an older doc; U = unverified.

Every setup is CHEATED (level, position, items, quest words by debug verbs): the rows test mechanics, never balance or parity. `writes_after_start` is 0 in every ledger row read (V). All runs used hidden lab slots through `tools/provingground/run.sh` under `tools/agentrun.sh labops2`.

Classes of a non-GREEN row: (a) product bug, (b) stale or wrong scenario expectation, (c) harness or setup, (d) expected RED.

## Bot under test

| batch | wall start (EST) | bot | how known |
|---|---|---|---|
| 1 (town.plan, cain pair and seeds, nav A/B, board, necro pair, act2 first pass) | 00:30 to 01:25 | main `f30206f0d` (fix1, fix2, fix3, fix6, fix7 merged) | V: one `bot_sha256` `1e6281b49a...` in all ledger rows; some rows name `ed547f5f1`, a local scenario-only commit that was rewritten as `76e8bade2` (same bot sources) |
| 2 (second cain pair, park restart x3, and later) | from 01:23 | main `93b181848` (adds fix8) merged as `b1d54bf20` | V ledger |

Host load during batch 1 was 15 to 45 (other agents' merge gates), so the wall times are long and the pace is 4.5 to 13 virtual minutes per wall minute (V `prov.pace`). Lockstep keeps the verdicts independent of the load, except where a scenario trigger is in WALL seconds (`trigger.api[].after_s`, `trigger.restart_at_s`; V `cmd/d2scenario/tap.go`, `run.go`).

## Batch 1 table

| scenario | class/seed | result | vs earlier night | class | evidence (V unless tagged), run dir |
|---|---|---|---|---|---|
| town.plan.belt-refill-from-pack (`town.plan` 4) | Paladin 11 | GREEN | was BLOCKED (no pack room) | - | fix6 pack room works; `20261128-003041` |
| town.plan.belt-refill-from-pack (`town.plan` 4) | Sorceress 3 | RED | was BLOCKED | b | `belt_at_least >= 3 hp` cannot hold for this class: the bot fills the belt 2 healing + 2 mana (`belt: moved 3 potions from pack (hp 1->2 mp 0->2)`, plan line `needs healing 2/4`), then drinks in the field; `item_in_pack >= 6 x hp1` fails for the same reason (9, 7, 5, 2 left). The fill from the pack itself works three times in the log; `20261128-004424` |
| town.plan.sell-keeps-need (`town.plan` 4) | Paladin 11, Sorceress 3 | GREEN, GREEN | was RED (b), assertion fixed by labtriage | - | `20261128-003041`, `20261128-004424` |
| town.plan.malus-errand-gold (`town.plan` 4) | Paladin 11 | RED | same | a | `quest_bits record 3 ... got 0x0000`; card townnight, waits for nf4; `20261128-003041` |
| town.plan.short-gold-handover (`town.plan` 5) | Paladin 11, Sorceress 3 | GREEN, GREEN | was RED (b), pattern fixed by labtriage | - | `20261128-003527`, `20261128-004802` |
| town.plan.one-visit (`town.plan` 5) | Paladin 11, Sorceress 3 | RED, RED | same | a | no `town plan: ... stops=` line; card townnight, waits for nf4 |
| town.plan.gold-short-priorities (`town.plan` 5) | Paladin 11, Sorceress 3 | RED, RED | same | d | priority table is stage 7 (S), not built |
| cain.chain determinism pair | Paladin 11 | GREEN + GREEN | was RED + GREEN | - | both end `quest_bits` 0x2019; details below; `20261128-003057` |
| cain.chain | Paladin 12, 13 | GREEN, GREEN | same | - | `20261128-010529` (503 s wall), `20261128-011353` (411 s) |
| nav A/B, 5 areas, 4 arms, seeds 11 and 3 | Paladin | 32 GREEN, 8 RED | seed 11 identical to the first night; seed 3 now complete | a (the 8 RED) | all 8 RED are `nav.barracks-return` in every arm; table below |
| board.sm.stop-cascades | Paladin 11, Sorceress 3 | GREEN, GREEN | was RED (b) | - | `after_s` 60 to 8 (scenario fix here): the cancel now lands while the scroll chunk runs and the reply lists the dependants; `20261128-010108`, `20261128-011304` |
| board.sm.restart-mid-chain | Paladin 11 | RED | was RED (b) | a | new cause: the chunk parks `gave_up` one second after its start, see below; card boardsidestart (TASK-1120); `20261128-010108` |
| board.sm.restart-mid-chain | Sorceress 3 | RED | was RED | c | the hero has no mana (1325 x `out of mana`), the scroll is not taken in 150 wall seconds, so the restart resumes the chunk (second `starting chunk` line) and finishes it 60 s later. The resume works; the scenario's precondition (scroll read before the restart) is not met by this setup; `20261128-011304` |
| board.park.persist-restart (first run, old assertions) | Paladin 11 | RED | was PENDING | c | no hold and no park: the recovery ends, the hero goes to the Countess, `auto-finish forgotten_tower/countess: the quest reads done`; `20261128-010108` |
| board.park.persist-restart (first run, old assertions) | Sorceress 3 | RED | was PENDING | b | the chunk DID park and the park DID survive the restart, but as `gave_up` after three tower guard refusals (`0 mana potions < 2 (caster)`), not as `hold_timeout`; assertions rewritten here, rerun in batch 2; `20261128-011304` |
| necro.corpse-ce1, necro.corpse-ce5 | Necromancer 11, 12, 13 | RED x6 | same | c | `minions 0`; the build is now right (`combatBuild=summoner`, was not logged so before) and the hero casts Teeth (15 of 76 kills), but there is no `skills: allocated`, `leveler: plan` or `combat minion` line: no summon skill is learned. Waits for drillskills (nf1); `20261128-010438`, `-010526`, `-010629` |
| drill.act2.amulet, radament, radament.handin (.stepexec) | Paladin 11 | GREEN x3 | same | - | `20261128-011030` |
| drill.act2.cube.stepexec | Paladin 11 | RED | same | a | invariant `potion-not-drunk` 2 hits; card act2night, waits for nf3 |
| drill.act2.shaft.stepexec | Paladin 11 | RED | same | c | `item_in_pack >= 1 x msf, got 0`; card act2night, waits for nf3 |

### cain.chain: the TASK-680 AC 3 question

| | rep1 | rep2 |
|---|---|---|
| verdict, end word | GREEN, 0x2019 | GREEN, 0x2019 |
| state digest | `6393df446af3de71` | `1e64366ee7cdc9e1` |
| assert digest | `fba0b25d82093bbe` | `fba0b25d82093bbe` |
| taps at virtual s (start scroll, start read, hand-in read, start stones, start free_cain) | 90, 572, 603, 630, 918 | 63, 570, 601, 630, 920 |
| `cancelled, nothing clicked` / `portal retry` | 2 / 0 | 2 / 0 |
| hand-in queued / served / dropped | 1 / 1 / 0 | 1 / 1 / 0 |
| death ban, death autopsy | 0, 0 | 0, 0 |
| `virtual run length reached` | 1 | 1 |
| wall s, load1 at start | 1077, 38 | 993, 22 |

First divergence (V `determinism.txt`): BOT-SIDE at journal record 1396, tick 138: rep1 sends `debug missiles` and steps, rep2 sends `debug players` on identical state. It is a read-order difference in the first seconds, not a game difference.

Verdict by the rule of `cain-chain-flake-2026-11-27.md` section 5: A1 and A2 are GREEN, B and C are GREEN, but the digests of A1 and A2 are NOT equal, so the rule's first branch ("GREEN with equal digests") is not met to the letter and its second branch ("A1 differs from A2") is met only for the digest, not for the verdict. Proposed reading for the card owner: the old flake (one RED of two on `2f2e092c3`) did not come back in 2 of 2 on `f30206f0d`; the first start tap moved by 27 virtual seconds between the repetitions and the bot tolerated it; no run needed the portal retry (0 `portal retry` lines with 2 cancels each, so these greens say nothing about the retry). A second pair on fix8 is in batch 2. Stop rule note: the pair ran above the builder's normal load (22 to 38) and was not cut (under the 40 min limit).

### nav A/B (knobs3way, full result rows read, not the last three lines)

| area | seed | status off/grid/gridwarp/all | `nav stuck` lines | warptriggers reads | levelmap reads | journal lines | run dir |
|---|---|---|---|---|---|---|---|
| nav.barracks-return | 11 | RED/RED/RED/RED | 0/0/0/0 | 0/0/18/18 | 0/0/0/18 | 59894/60075/59958/60486 | k3-003115 |
| nav.bloodmoor-coldplains | 11 | GREEN x4 | 4/4/4/4 | 0/0/33/33 | 0/0/0/33 | 73251/73271/73305/73106 | k3-003308 |
| nav.inner-cloister-cathedral | 11 | GREEN x4 | 4/4/4/4 | 0/0/34/34 | 0/0/0/34 | 25184/25376/25375/25595 | k3-003631 |
| nav.catacombs-hop-33-34 | 11 | GREEN x4 | 32/36/36/36 | 0/0/13/13 | 0/0/0/13 | 83932/41369/41402/41715 | k3-003722 |
| nav.catacombs-hop-34-35 | 11 | GREEN x4 | 9/9/12/12 | 0/0/23/23 | 0/0/0/23 | 76659/68527/67951/67507 | k3-004017 |
| nav.barracks-return | 3 | RED/RED/RED/RED | 0/0/0/0 | 0/0/18/18 | 0/0/0/18 | 59938/60097/60653/59920 | k3-004553 |
| nav.bloodmoor-coldplains | 3 | GREEN x4 | 8/8/8/8 | 0/0/37/37 | 0/0/0/37 | 71875/71335/71403/71382 | k3-004722 |
| nav.inner-cloister-cathedral | 3 | GREEN x4 | 5/5/5/5 | 0/0/33/33 | 0/0/0/33 | 26683/26819/26988/26870 | k3-005315 |
| nav.catacombs-hop-33-34 | 3 | GREEN x4 | 14/20/20/20 | 0/0/30/30 | 0/0/0/30 | 79210/79686/80084/79886 | k3-005414 |
| nav.catacombs-hop-34-35 | 3 | GREEN x4 | 14/9/9/9 | 0/0/30/30 | 0/0/0/30 | 80616/89168/89284/89336 | k3-005947 |

Reading:

1. `nav.barracks-return` is RED in all four arms on both seeds with the knobs OFF too: `d2dworld: arrived in level 28: 0 adjacent entries`, 62 x `nav: no exit to that level` (V both seeds). Not a knob regression; card navbarracks (TASK-1102) is not built yet.
2. `nav.fullGrid` (off against grid): the `nav stuck` line count is equal on 5 of 8 green pairs, higher on hop 33-34 (32 to 36 on seed 11, 14 to 20 on seed 3) and lower on hop 34-35 seed 3 (14 to 9). On hop 33-34 seed 11 the journal is half as long with the grid on (the walk is shorter). Whether four to six more `nav stuck` lines on a shorter walk are a "stuck regression" is the coordinator's reading; no arm failed.
3. `nav.warpRead` (grid against gridwarp): equal everywhere except hop 34-35 seed 11 (9 to 12). The read is proven on by its verb in every area.
4. `quests.layoutRead` (gridwarp against all): no change in any count; the level map is read, no quest object is looked for in these drills.

### board.sm.restart-mid-chain, Paladin 11: a product bug (TASK-1120)

V log `20261128-010108`: the levelling plan (`level plan: clvl 17 -> area 23 Tower Cellar 3`) walks the hero through the Dark Wood and the Black Marsh into the Forgotten Tower in the 20 wall seconds before the tap (`d2dworld: arrived in level 20: 0 adjacent entries`). The tapped chunk `search_for_cain/scroll` starts at 01:01:32; its step `Go to the Dark Wood` fails three times between 01:01:33.0 and 01:01:33.9 with `exit to Black Marsh: nav: no exit to that level: level 6 from 20`; the board logs `parked search_for_cain/scroll` and `gave up after 3 tries`. After the restart at 150 s the bot logs `search_for_cain/scroll parked: The Search for Cain is stuck: gave up after 3 tries` and does not start the chunk again. So the park survives the restart (the property the row is for), and the row is RED because the chunk never ran. The first night's Paladin passed this step because the tap found the hero outside the tower.

Two faults: the missing way out of a level entered with 0 adjacent entries (same as TASK-1102) and the board counting three failures inside one second as three tries with no road through town. The second is the new card.

## Scenario changes made here (`tools/provingground/scenarios/board.json`)

| scenario | change | why |
|---|---|---|
| board.sm.stop-cascades | `trigger.api[0].after_s` 60 to 8 | the trigger clock is wall seconds from the bot start; the chunk was done 23 to 57 s after the start. Now GREEN on both classes. A state-tied trigger stays the real fix (TASK-1106 AC 2) |
| board.sm.restart-mid-chain | `restart_at_s` 300 to 150, budget 30 to 60 virtual min, the assertion counts `quest board: starting chunk search_for_cain/scroll` (max 1) instead of the step line | the 30 virtual minutes ended after about 150 wall seconds, so the restart came after the budget; the step line repeats on a retry inside one process |
| board.park.persist-restart | pending to active, matrix Sorceress 3 only, `restart_at_s` 1500 to 420, assertions: parked once, `blocked` at the end with any reason, started at most once | the setup parks by `gave_up` (guard refusals), never by `hold_timeout`; the Paladin does not park at all |

These touch the file card bsmnight (TASK-1106) owns; that card is Proposed and unassigned work, and the brief of this night allowed scenario fixes. TASK-1106 AC 1 and AC 2 are covered in the sense above; AC 3 (both GREEN on both classes) is not: restart-mid-chain needs TASK-1120 (Paladin) and a Sorceress setup with mana.

## Batch 2 table (01:23 to 02:05 EST)

Bot: main `93b181848` (fix8) for the cain pair and the park drill (V `bot_sha256` `4bae6a802e...`), main `702b6cfd5` (adds nf1, the navbarracks fix) for the rest (V `2bb15e1b13...`).

| scenario | class/seed | result | vs earlier | class | evidence (V unless tagged), run dir |
|---|---|---|---|---|---|
| cain.chain determinism pair, second | Paladin 11 | GREEN + GREEN | batch 1: GREEN + GREEN | - | 0x2019 both; digests `cfd482943b853b60` vs `e8711c5d81f6c144`, same assert digest; first divergence BOT-SIDE at record 49152, tick 10376 (41 equal checkpoints before it); cancels 2 and 3, `portal retry` 0; wall 413 and 655 s; `20261128-012318` |
| board.park.persist-restart (new assertions) | Sorceress 3, 3 repetitions | GREEN 3 of 3 | first run RED (b) | - | each repetition: `quest board: parked forgotten_tower/countess` 272 to 321 wall seconds after the start, restart at 420 s, then `forgotten_tower/countess parked: The Forgotten Tower is stuck` once and no second `starting chunk`; `20261128-012548` |
| nav.barracks-return on the nf1 build, 4 arms | Paladin 11, 3 | RED x8 | same | a | nf1 did not change the lab result: `nav no usable gate from=28 to=27 exits=1 adjacent=0 borderCells=0 unproven=131 pending=1 cands=0 known=63000 fullyKnown=true badTTL=true`, `nav no exit known yet, exploring`, then 20 x `exit to Outer Cloister: nav: no exit to that level: level 27 from 28 (nothing new since the last search)` inside one second, `d2dworld: link wait over after 0s, 0 polls ... linked [27]`, `town portal did not appear` x5. TASK-1102 AC 1 stays open; `k3-014356` |
| cain.scroll.stepexec | Paladin 11, 3 repetitions | GREEN 3 of 3 | not run in the first night | - | `20261128-014714` |
| cain.read.stepexec | Paladin 11, 3 repetitions | GREEN 3 of 3 | 1 of 1 | - | same dir |
| drill.act2.assemble.stepexec | Paladin 11, 3 repetitions | GREEN 3 of 3 | not run | - | same dir |
| drill.act2.duriel.stepexec | Paladin 11, 3 repetitions | BLOCKED 0 of 3 | BLOCKED 0 of 3, but further | a | the chunk now starts and runs to the boss: `Entered Duriel's Lair`, `boss not found but dead: the kill step is done`, `Quest progress: duriel`, then 11 x `step talk_tyrael: 2 tries used: open Tyrael: town: npc not found or not reachable`, `parked seven_tombs/duriel`, campaign stall, exit. Identical in all three. New card durieltyrael (TASK-1126); `20261128-015020` |
| drill.act2.summoner.stepexec | Paladin 11 | BLOCKED (stopped by the operator after 35 min wall) | BLOCKED (watchdog) | a | new shape: `step touch_journal: 2 tries used: interact done, but post [quest_state(summoner,done)] not observed after 4 looks`, `town: no town portal available`, `parked summoner/kill_summoner ... all steps done but the quest does not read as finished`, and from 01:18:52 a busy loop of `roam: done` and `gold ledger` in the Canyon of the Magi, 79000 of each in 31 min with no `status` line and no clock step in the journal (the last journal records are `debug players` reads at tick 74199). Same loop as cainspin (TASK-1103); the summoner cause is card summonerstall (TASK-1104, fix in nf3, not merged); `20261128-011030` |
| town.plan.scroll-hoard, town.plan.broke-after-bank (`town.plan` 4) | Paladin 11, Sorceress 3 | GREEN x4 | GREEN (old bot; broke-after-bank was Paladin only) | - | `20261128-015337`, `20261128-015436` |
| town.plan.belt-refill-from-pack (`town.plan` 4, class-neutral assertions) | Sorceress 3, Paladin 11 | GREEN, GREEN | Sorceress RED (b) in batch 1 | - | scenario fix here (fill line and no potion sold, in place of two end counts); `20261128-015616`, `20261128-015803` |
| board.sm.restart-mid-chain (tap version) | Paladin 3 | RED | new seed | a | the same park in one second as seed 11 (`level 6 from 20`), so TASK-1120 is not a seed 11 accident; `20261128-015546` |

## Batch 3 table (02:00 to 02:30 EST)

Bot: main `702b6cfd5` (nf1) for the rows started before 02:12, main `41fb5b723` (adds nf3) for the act2 rows (V `bot_commit` `42d84163a`), main `28216bd08` (adds nf1b, nf2, nf4; merge `5a431443e`) for the combat and necro rows.

| scenario | class/seed | result | vs earlier | class | evidence (V unless tagged), run dir |
|---|---|---|---|---|---|
| board.sm.restart-mid-chain (chain accepted in the setup) | Paladin 11, 13, Sorceress 3 | GREEN x3 | Paladin 11 RED (a), Sorceress 3 RED (c) in batch 1 | - | Paladin 11: chunk start 4 s after the bot, `awaiting hand-in of search_for_cain/read` at 27 s, restart at 150 s (38 virtual min done), no second `starting chunk`, scroll `done`; `20261128-020322`, `-020438`, `-020728` |
| board.sm.restart-mid-chain (tap version) | Paladin 12 | GREEN | new seed | - | the tap found this hero outside the tower; `20261128-020025` |
| board.sm.stop-idempotent | Paladin 11 | RED then GREEN | not run before | b | RED with the triggers at 60 and 75 wall seconds (409 `done`, the chunk was over); GREEN after the scenario fix (8 and 14 s); `20261128-020016`, `-020911` |
| board.sm.busy-second-quest | Paladin 11 | RED then GREEN | not run before | b | RED with the trigger at 30 s (the second quest simply started, HTTP 200); GREEN at 8 s (409 `busy`); same dirs |
| board.sm.stop-idempotent, busy-second-quest | Sorceress 3 | GREEN x2 with the old and x2 with the new trigger times | not run before | - | `20261128-020443`, `-021352` |
| board.sm.start-idempotent, board.sm.one-head | Paladin 11, Sorceress 3 | GREEN x4 | not run before | - | `20261128-020016`, `-020443` |
| board.handin.field-tap, board.handin.without-run, board.hold.broke | Paladin 11 | GREEN x3 | not run before | - | `20261128-021638` |
| board.hold.restock | Paladin 11 | RED | not run before | b, U | no `hold needs_potions` line: with the three potions of the setup the chunk starts at once and the Countess run is auto-finished. The hold wording or the kit minimum moved since the scenario was written (U which); same dir |
| board.hold.restock-stash-gold | Paladin 11 | RED | not run before | U | `stash 6641`, wanted at most 4999: the stash grew instead of paying the restock; the start gate did wait (`waiting for the recovery (purse and kit) to finish`). Not triaged further; same dir |
| board.retry.runs-a-step, board.stale-accept-unblocks, board.view.reason-always | Paladin 11 | BLOCKED x3 | not run before | c | operator fault: `exit status 1` at wall 0 while the merge of nf4 left `board.json` with conflict markers for about a minute; rerun below |
| drill.act2.shaft.stepexec (nf3) | Paladin 11 | GREEN | was RED | - | the early road fix works: `msf` in the pack; `20261128-021402` |
| drill.act2.radament, radament.handin, amulet (.stepexec, nf3) | Paladin 11 | GREEN x3 | same | - | same dir |
| drill.act2.cube.stepexec (nf3) | Paladin 11 | RED | same | a | `invariant potion-not-drunk` twice: `life=35% line=50% beltHP=4 drank=0 for=18s hostiles=4`, then `life=18% ... drank=0 for=93s hostiles=0`, step `Quest: fighting (Level up to 23)`. TASK-1107 names it, its result says not addressed |
| drill.act2.duriel.stepexec (nf3) | Paladin 11, 3 repetitions | BLOCKED 0 of 3 | same as batch 2 | a | identical to batch 2 in all three (lair 1, boss dead 1, 11 x `talk_tyrael`, parked); TASK-1126; `20261128-021448` |
| drill.act2.summoner.stepexec (nf3, without nf2) | Paladin 11 | BLOCKED | batch 2: busy loop | a | no watchdog and no busy loop any more: `boss: left the route or arena`, `Kill The Summoner` twice, `all steps done but the quest does not read as finished`, `parked summoner/kill_summoner`, then campaign stall and exit after 141 s wall. The journal touch is still not credited; `20261128-021806` |
| combat.whirlwind-clears.control | Barbarian 11 | GREEN | same | - | `20261128-022149` |
| combat.whirlwind-clears (nf1b) | Barbarian 11 | RED | same status, new cause | a, U | the setup holds now (Whirlwind bound right, prep self-check passed) and the driver still logs no `combat move` and no `move shadow` line; 98 kills by Attack 42, Double Swing 47, Bash 9; card drilldrivers (TASK-1127) |
| combat.hold-interruptible (nf1b) | Paladin 11 | RED | same status, one assertion fewer | b | Zeal is used now (25 kills `last_skill=Zeal`); the only failing assertion is `^combat potion`: minlife 100 in every fight; `20261128-022229` |
| combat.teleport-reposition (nf1b) | Sorceress 11 | RED | same | b | no `use=escape` leg: minlife 100, no surround; 6 x `out of mana`; `20261128-022258` |
| combat.corpse-explodes (nf1b) | Necromancer 11 | RED | same | a, U | Corpse Explosion bound right, the Clay Golem of the setup stands (`combat minion skill=Clay Golem`), kills by Attack 69 and Teeth 13, no `combat corpse-explosion` line; TASK-1127; `20261128-022322` |
| necro.corpse-ce1, ce5 (nf1b) | Necromancer 11, 12, 13 | RED x6 | was `minions 0`, now `minions 1` | a, U | the one minion is the setup's golem; Raise Skeleton has 8 points and is never cast (casts: Teeth 28 to 31, Amplify Damage 2, Bone Armor 1, Clay Golem 1) with 85 corpses made; TASK-1127; `20261128-022352`, `-022436`, `-022528` |

## Batch 4 table (02:24 to 02:50 EST)

Bot: main `28216bd08` (fix1 to fix3, fix6 to fix8, nf1, nf1b, nf2, nf3, nf4; V `bot_commit` `5a431443e` or `3a674388b`, both merges of that main).

| scenario | class/seed | result | vs earlier | class | evidence (V unless tagged), run dir |
|---|---|---|---|---|---|
| cain.chain with `bot.portalRetryWaitSec` (0) | Paladin 11 | GREEN | was BLOCKED (busy loop, 2108 s wall) | - | nf2 works in the lab: 552 s wall, 23590 log lines (was 2.7 million), `virtual run length reached`, end word 0x2019, 2 x `cancelled, nothing clicked`, 0 x `portal retry`: the chain passes without the retry; `20261128-023323` |
| town.plan.one-visit (`town.plan` 5, nf4) | Paladin 11, Sorceress 3 | RED, RED | same status, new cause | c, U | nf4 removed the old cause; now every stop lacks a position: `town plan: route leaves out chore shop (no preset position for shop)`, the same for stash, repair, bank, withdraw, then `route kept the list (no stop of this visit has a preset position)`, 6 visits. So the lab hero's town has no preset positions at all for the route (U: the lab d2d does not serve them, or the bot does not read them in this start); `20261128-022421`, `-022808` |
| town.plan.short-gold-handover (`town.plan` 5) | Paladin 11, Sorceress 3 | GREEN, GREEN | GREEN | - | third and fourth green; same dirs |
| town.plan.gold-short-priorities | both | PENDING | was RED (d) | d | nf4 marked it pending for card town7 |
| town.plan.malus-errand-gold (`town.plan` 4, nf4) | Paladin 11 | RED | same | c | `tools_of_the_trade/malus: refused (route_closed): ... route guard (life 180 needed for the underground_passage road (max life 55))`, parked after 3 tries: the setup hero is too weak for the road, as the nf4 card result says; `20261128-023115` |
| drill.act2.summoner.stepexec (full build) | Paladin 11 | BLOCKED | no busy loop any more | a | nf2 and nf3 hold (0 `roam: done` loops, no watchdog). What is left: `step touch_journal: 2 tries used: interact done, but post [quest_state(summoner,done)] not observed after 4 looks`, `town: area -1 not reached within 15s`, `quest.summoner/home ran but is still needed`, park, campaign stall, exit after 147 s wall. TASK-1104 stays open on the journal credit; `20261128-024236` |
| board.retry.runs-a-step | Paladin 11, Sorceress 3 | RED, RED | not run before | b, U | `^quest board: step` wanted after virtual minute 3, all step lines fall before it (the chunk is over in the first minutes); `20261128-022643`, `-022313` |
| board.stale-accept-unblocks, board.view.reason-always | Paladin 11 (rerun), Sorceress 3 | GREEN x4 | Paladin BLOCKED by the operator's merge | - | same dirs |
| board.handin.field-tap, board.handin.without-run, board.hold.broke | Sorceress 3 | GREEN x3 | not run before | - | `20261128-022313` |
| board.hold.restock | Sorceress 3 | RED | as Paladin | b, U | no `hold needs_potions` line |
| board.hold.restock-stash-gold | Sorceress 3 | RED | Paladin RED on the stash count | U | `parked forgotten_tower/countess ... gave up after 3 tries` twice (the caster's mana potion guard, as in the park drill) |
| board.sm.restart-mid-chain, board.sm.stop-cascades (confirm on the final main) | Paladin 11 | GREEN, GREEN | GREEN | - | `20261128-023336` |
| board.sm.stop-cascades (confirm) | Sorceress 3 | GREEN | GREEN | - | `20261128-023818` |
| board.sm.restart-mid-chain (confirm, restart at 150 s) | Sorceress 3 | RED | GREEN at 02:07 | c | the scroll was not taken at 150 wall seconds (1095 x `out of mana`), the restart resumed the chunk and finished it 60 s later; the 02:07 green had made it with 5 s to spare. The drill is marginal for the banked Sorceress; restart moved to 240 s in a 90 min budget, 3 repetitions per class in batch 5 |
| combat.point-spell-hits, combat.point-spell-hits.druid | Sorceress 11, Druid 11 | GREEN, GREEN | first run | - | `20261128-023932`, `-024007` |
| combat.hammer-in-pack and control | Paladin 11 | BLOCKED x2 | first run | c | setup: `build paladin_hammerdin_guide at clvl 24: main attack Blessed Hammer gets no point from the plan (banked: 0 skill points)`; `20261128-023920` |
| combat.traps-stand | Assassin 11 | RED | first run | a, U | 28 casts of Fire Trauma, no `combat trap` line; TASK-1127; `20261128-024044` |
| combat.army-holds.control | Necromancer 11 | GREEN | first run | - | `20261128-024117` |
| combat.army-holds, combat.pets-hold | Necromancer 11, Druid 11 | RED, RED | first run | b, U | the stance works (`combat stance summary kind=behindArmy casts=12 in_place=9`); only `^combat minion` is missing, and these scenarios do not set `combat.minionRead` (U whether the line needs it); `20261128-024117`, `-024158` |
| necro.upkeep-pack | Necromancer 11 | GREEN | not run in the first night | - | golem and skeletons raised, minions standing, with `combat.minionRead` (1); `20261128-024225` |
| necro.revive-after-death | Necromancer 11 | RED | first run | U | 5 minions stand (Raise Skeleton 3, Skeletal Mage 1, golem 1), no Revive cast; the run lasted 9 wall seconds; same dir |
| act1.wp.dark-wood, act1.wp.dark-wood.layout | Paladin 11 | GREEN, GREEN | first run | - | the waypoint step ran with and without `quests.layoutRead`; `20261128-024251` |
| act1.wp.dark-wood, act1.wp.dark-wood.layout | Sorceress 3 | RED, RED | first run | c | both arms: `refused (route_closed): area not allowed now: Dark Wood (5): route guard`, 735 x `out of mana`: the banked Sorceress is refused by the road guard; `20261128-024405` |

## Batch 2 and later

See the sections below; they are appended as the runs finish.
