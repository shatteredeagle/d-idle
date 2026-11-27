# Proving Ground: fast tier on the builder, and what the expansion makes runnable (2026-11-19)

> status: current (card pgexpand). Tags: V verified (run named), S secondary, U unverified. Builder long lane job 20261119-135007-52f0, lab d2d slots, bot of branch agent/pgexpand, Paladin seed 11, one run each, `tools/provingground/run.sh -tier fast -parallel 2`; raw `results.json` in `docs/probes/provingground-run-2026-11-19/fast-tier-results.json`. Every setup is CHEATED or SETUP (see each scenario's `provenance`): mechanics evidence, never balance evidence. A RED here is a finding about the bot or the scenario, nothing was fixed in this card.

> verdicts: current verdicts per scenario and class are in [proving-ground-status.md](../process/proving-ground-status.md) (generated from the results ledger) and on the board page /pg. The tables below record these runs only; new runs are cited by run id, not added here (design 2026-11-21 Q10).

## Fast tier, first run (V)

| scenario | verdict | wall | what the state shows |
|---|---|---|---|
| belt.refill-after-fight | GREEN | 149 s | belt topped up (min 3 hp-class column potions; U that 3 is the policy floor) |
| cain.read | GREEN | 113 s | `bkd` in the pack, no `bks`. CAUTION: the driver tapped nothing (empty taps log); the deciphering happened at a normal Akara visit, so the board hand-in path was NOT exercised (the assertion is satisfiable without the board) |
| cain.scroll | RED | 373 s | no `bks`/`bkd`; same signature as pgcain's `cain.scroll` from the camp: the chunk starts, town trips and fights keep the hero off the tree. Trail 1,2,3,4,10,1,10,5,6,5,10,4,1,4,1,4,1,4,3,9,1 |
| flail.areachanges | RED | 387 s | 26 area changes (limit 12). The driver tapped `start scroll`. Trail ends 1,9,13,9,1,2,1: the hero returns to the camp (area 1) six times and never completes the tree step. A town-start scenario with no travel to speak of (town.resupply) already shows 14 changes, so 12 is tight for a 40 minute window that includes the walk to the Dark Wood; U whether the limit or the stall is the finding. Both are worth a look; the stall is the same one as cain.scroll |
| heal.town | RED | 18 s | life 56 percent at the end. The run reached its 4 virtual minutes (`virtual run length reached`), the hero was healed (`status` life 22 then 136 after the chores) and then walked out and fought, so the end state is life after a fight, not after the heal. Scenario artifact (U): needs an observable taken right after the heal, or a goal that keeps the hero in town |
| town.resupply | GREEN | 94 s | hp1 in the belt, `tbk` bought |
| town.stash-gold-withdraw | GREEN | 140 s | stash 128000 to 127250 (750 gold taken), hp1 in the belt, `tbk` bought |

Counts: GREEN 4, RED 3, BLOCKED 0, PENDING 0 (fast tier, 7 runs). `writes after start: 0` on every town-start run (V, the `no_writes` assertion passed where present).

## All 19 scenarios by status (what runs now)

| status | scenarios |
|---|---|
| GREEN (run above) | belt.refill-after-fight, cain.read, town.resupply, town.stash-gold-withdraw |
| RED (run above) | cain.scroll, flail.areachanges, heal.town |
| RED or GREEN by pgcain (`docs/probes/provingground-cain-2026-11-19.md`) | cain.scroll.field GREEN with cainfix2 and `bks` or `bkd`; cain.scroll.sandbox RED (same stall as the lab build) |
| RUNNABLE, not run yet (full tier or nightly, field starts need the attach path) | cain.stones, cain.cage, cain.report, cain.chain, chicken.recovery, loot.rare-ring-classified |
| BLOCKED by design | cain.stones.shipped: a field start needs the lab d2d (verbs on) handed to the bot and the shipped build refuses the warp; stone lit state is a lab fact |
| PENDING | town.repair-trip, merc.hire-act1, difficulty.nightmare-start (reasons in the files) |

Counts over 19: GREEN 4 (plus 1 by pgcain), RED 3 (plus 1 by pgcain), runnable-not-run 6, BLOCKED 1, PENDING 3.

## Decisions (the observables)

- Area changes: the number of `zone entry` lines after the first in the bot's own log (`state.area_changes`, `area_trail`). S: the bot's report of what the game told it. The d2d holds one control client and has no level counter.
- Cairn stones: the observable is `debug portals` listing a portal in level 4 (`portal_open`), V L1 on the lab build; it stays listed from Tristram. The per-stone `NU` to `ON` column is readable only while the hero is in Stony Field and is the bot's own check. Shipped build: BLOCKED (reason above); U that it lists portals.
- Board-driven scenarios use the `board-taps` driver; a scenario that must exercise the hand-in should assert something only the board path produces (open item for `cain.read`).

## Determinism (V, job 20261119-141800-a155 plus the fast-tier run)

`town.resupply` Paladin seed 11, three runs in total: GREEN, GREEN, GREEN, same verdict every time (`d2scenario determinism` exit 0, hard rule met). State digests: `40ec13d9` (fast-tier run), `4e9f6d77` (pair run 1), `40ec13d9` (pair run 2). The digest differs in one run (gold 700 against 712, stash 395 against 250; area changes 14 in all three): the documented wall-clock decisions of the bot loop, a warning not a failure (`docs/d2d-determinism.md`). The second scenario of the pair was not run (one scenario only, U for the rest).

## Coverage (`d2scenario coverage`)

91 behaviours: COVERED 22 (was 10), PENDING-ONLY 5, UNCOVERED 64. Newly covered: hook.flail-hold, hook.heal-in-town, hook.low-life-gate, hook.board, hook.task, hook.claim (by the chain, not yet run), resource.chicken, loot.pickup-classify, travel.portal-return, combat.fight, hook.belt-fill and town.resupply with more scenarios, and the quest chunks `stones`, `free_cain`, `report` (all five Search for Cain chunks now have a scenario). Still uncovered: every quest outside Search for Cain (42 chunks), the vault, gear, claim-trip, farm, recovery and potion-drink hooks.

# Second run (card pgrun2, 2026-11-19, evening)

> Tags: V verified (result file named), S secondary, U unverified. Builder, lab d2d slots, bot of branch agent/pgrun2, one scenario per job on the main lane (builder idle, queue empty), `tools/provingground/run.sh -id <id> -class <C> -seed <n> -parallel 1`. Raw `results.json` per job in `docs/probes/provingground-run-2026-11-19/run2/<hhmmss>-results.json` (stamp in the table). Every setup is CHEATED (mechanics evidence only). A RED is a finding; nothing in the bot was changed in this card.

## First pass: the scenarios as merged (Paladin seed 11 unless noted)

| scenario | verdict | wall | stamp | stall line (V, the bot log and the saved state of that run) |
|---|---|---|---|---|
| heal.town | GREEN (Paladin 11 and Sorceress 3) | 8 s | 165421, 173941 | after moving the assert point: `virtual_min` 4 to 1 (the run ends right after the free heal; the first run read life after the next fight) |
| cain.report | GREEN | 31 s | 165747 | quest report handed in |
| cain.stones | RED | 99 s | 165442 | `portal_open`: no portal in level 4. `quest board: start gate: waiting for the recovery (purse and kit) to finish`, then `queued: waiting for the fight to end`; the chunk never touches a stone. The hero is clvl 20 with 55 life (the banked points were never spent): chicken and potion trips, trail 4,1,2,3,4,10,1,10,... 24 area changes |
| cain.cage | RED | 84 s | 165622 | record 4 is 0x0018 (Tristram arrival bit set, cage bit 0x0002 not). Start in Tristram (38), `safety decision chicken` at life 10 of 55, `death autopsy`, the hero leaves Tristram for the camp and the level plan (trail 38,1,38,1,...,2,3,4,10) |
| cain.chain | RED | 2106 s | 165941 | 80 virtual min. The first two chunks WORK end to end on the board: taps `start scroll` 17:00:08 then `start stones` 17:06:30 (tree, Akara reads: `bkd` in the pack). Then `step Touch the cairn stones`, chickens in Stony Field (max life 55 at clvl 32), `quest board: search_for_cain/stones: needs_attention: ... not ready: area banned after repeated deaths`; 86 area changes (3,4,1,4,1 ...), never lit a stone |
| chicken.recovery | RED | 43 s | 165819 | `area_changes_le` 8 against 23. `recovery: start (gold 60 < kit 500)` (the 800 purse is gone within the first minutes), 55 life at clvl 10, `safety decision chicken` many times, Cave Level 1 <-> camp six times |
| loot.rare-ring-classified | RED | 32 s | 165902 | `item_absent rin ground` fails: the ring stays on the ground. The hero leaves Blood Moor within a minute (level goal) and the bot log has no pickup line; U whether the scenario's ring was ever in view |
| cain.scroll Sorceress seed 3 | RED | 189 s | 173448 | see "Why the Sorceress leaves" below |
| cain.stones second run (pair) | RED | 103 s | 173758 | determinism pair, see below |

## Why the Sorceress leaves (cain.scroll seed 3, V stamps 173448 and 174257)

Sequence in the log of 174257 (purse 5000, points spent): `quest board: starting chunk search_for_cain/scroll`, `step Activate waypoint`, then `quest board: waiting: context canceled` (a fight cancels the walk), `queued: waiting for the fight to end`, `start gate: waiting for the fight to end`, and 1 s later `level plan: clvl 17 -> area 19 Mausoleum (in window 11-18)`. The mover is the LEVEL goal (`reach:30`, level plan window), not campaign, farm or mana: from that line on the hero fights in the Crypt and Mausoleum (trail 17,19,17) and the chunk's start gate never opens again because the hero is always in a fight. Later `recovery: start (gold 40 < kit 500)` also holds the gate ("waiting for the recovery"). Two scenario defects sit under it: (1) the hero had 17 banked skill points and no bound attack (459 log lines `combat no bound attack skill, swinging the basic attack`, life 22 percent at the end, 117 area changes between camp and Cold Plains); with the points spent (the knob `resource.pointsDecisions` 1, a played clvl 17 has spent them) she fights with Ice Bolt and stays healthy (life 93 percent, vit 88) but still leaves for the Crypt. (2) the purse of 5000 is spent within minutes (gold 40 at the end), so the recovery gate shuts the board again. Owner: bot (the board preempt gate behind a continuous fight under a level goal, same family as the Paladin signature) plus scenario (setup).

## Second pass: banked points spent by the bot (scenario knob, kept)

The common root of the first pass is the setup: a hero made clvl 20 to 32 by stat verbs keeps all its points banked (the product default is that the player spends them), so it has 55 life or no attack skill and cannot survive the field the scenario asks it to stand in. The Cain, chicken and loot scenarios now set `resource.pointsDecisions` 1 (the bot spends them, as a played hero's player would). Result (stamps 174257 to 175822):

| scenario | verdict | what changed |
|---|---|---|
| loot.rare-ring-classified Paladin 11 | GREEN | with a hero that survives, the ring is picked up (stamp 175822) |
| cain.stones Paladin 11 | RED | life 100 percent, vit 87, still no portal: `start gate: waiting for the recovery`, then `waiting for the fight to end`, the hero spends the run in Burial Grounds and Black Marsh (35 area changes); gold 80 at the end |
| cain.stones Sorceress 3 | RED | portals open in levels 3 and 1 only (the stone step started but the Tristram portal did not open); `recovery: end (reason stalled)` |
| cain.cage Paladin 11 | RED | still 0x0018: arrives in Tristram, `crowd-retreat`, leaves for the camp at once (trail 38,1,2,3,4,10,...) |
| cain.cage Sorceress 3 | RED | trail 38,1,38,1,38,1: three Tristram entries and exits, bit 0x0002 never set; life 84 |
| chicken.recovery Paladin 11 | RED | `item_in_pack hp1 belt` fails (0 hp1 at the end, purse 24): the restock after the chicken did not happen; `recovery: gold -N` lines, `crowd-abandon` |
| cain.scroll Sorceress 3 | RED | see above |

## Findings (nothing fixed in this card)

1. Scenario setup (owner scenario): cheated levels leave banked points; with them unspent the hero is unfit (55 life, no attack skill). The knob now spends them; the REDs that remain are not from this.
2. Purse drain (owner bot or scenario, U which): a 5000 or 800 purse is gone within minutes (tome, potions, gear) and `recovery: start (gold < kit 500)` follows; the quest board's start gate waits behind the recovery (V, the new `start gate` lines of the cainstart card). Every step scenario then spends its budget in a recovery. A scenario purse must cover the kit AND what the bot buys, or the recovery gate needs its own scenario.
3. Board start gate behind a continuous fight under a level goal (owner bot): `waiting for the fight to end` never ends in a dense area (Crypt, Mausoleum, Stony Field); the chunk is queued for the whole run (cain.scroll Sorceress, cain.stones).
4. Tristram (owner bot): cain.cage heroes arrive, flee (`crowd-retreat` or chicken) and walk back to the camp and the level plan instead of clearing the three archers; the cage never opens (both classes).
5. `area banned after repeated deaths` blocks the stones chunk for good in the chain (owner bot, U that a ban without expiry inside a chunk is the intent).
6. cain.read GREEN of the first run needs the caution of the first record (the hand-in path was not exercised); in the chain the scroll and read chunks DO run on the board (taps 6 min apart, `bkd`), so the chain is the evidence for that path.

## Determinism (V)

`cain.stones` Paladin seed 11 twice (stamps 165442 and 173758, about 45 minutes apart on the same builder): verdict RED both, state digest `3ae39e6352888fdf` BOTH times (gold 5, clvl 20, 24 area changes, same items). `go run ./cmd/d2scenario determinism <results.json> <results.json>`: `deterministic: 1 runs, same verdicts (0 digest warnings)`, exit 0. (The tool takes the two `results.json` files, not the per-run files: with the per-run files it prints `0 runs`.) Together with the `town.resupply` pair of the first run: two scenarios deterministic in verdict; one digest identical, one with the known wall-clock gold difference.

## Status after the second run

Run live at least once: GREEN belt.refill-after-fight, cain.read, cain.report, heal.town (both classes), loot.rare-ring-classified (with the points knob), town.resupply, town.stash-gold-withdraw, cain.scroll.field (pgcain); RED (findings above): cain.scroll, flail.areachanges, cain.stones, cain.cage, cain.chain, chicken.recovery, cain.scroll.sandbox; BLOCKED by design: cain.stones.shipped; PENDING: 6 skeletons. Not re-run after the knob: cain.chain, cain.report, flail.areachanges, cain.scroll Paladin.
