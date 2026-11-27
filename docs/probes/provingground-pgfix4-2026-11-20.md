# Proving Ground: chicken.recovery, loot ring, cancelled town-portal cast (card pgfix4, 2026-11-20)

> status: current. Tags: V verified (job or stamp named), S secondary, U unverified. Host: the builder main lane, lab d2d slots, bot of branch agent/pgfix4, one scenario per job (`tools/provingground/run.sh -id <id> -class <C> -seed <n> -parallel 1`). Every setup is CHEATED (scenario `provenance`): mechanics evidence only, never balance or play evidence. Follows `provingground-run-2026-11-19.md` (second run), `provingground-cainfix3-2026-11-19.md`, `provingground-cainchain-2026-11-20.md`.

> verdicts: current verdicts per scenario and class are in [proving-ground-status.md](../process/proving-ground-status.md) (generated from the results ledger) and on the board page /pg. The tables below record these runs only; new runs are cited by run id, not added here (design 2026-11-21 Q10).

## Verdicts

| scenario | before | after | what changed it |
|---|---|---|---|
| chicken.recovery Paladin 11 | RED (job 20261119 stamp 175645: no hp1 in the belt, 18 area changes) | GREEN (V job 20261120-132219-9d9e, 3 area changes, 74 s) | scenario only: purse 5000, farm goal |
| chicken.recovery Sorceress 3 | RED | GREEN (V job 20261120-132338-28f3, 7 area changes, digest b011e74da05842f4) | same |
| loot.rare-ring-classified Paladin 11 | GREEN in pgrun2 (U: it proved nothing, see below) | GREEN (V job 20261120-134754-2378: `Found ? Ring`, `sell: sold Ring gold 3069`) | scenario setup order, assertion, pickit rule |
| loot.rare-ring-classified Sorceress 3 | RED | GREEN (V job 20261120-140100-e537, purse plus stash 4804, digest a6584b86889d3d7a) | same |
| cancelled town-portal cast | not retried | retried once, unit-tested (two classes); no scenario (see below) | `internal/bot/portalretry.go` |

## 1. chicken.recovery: the scenario, not the bot (V)

The failing assertions were measuring the setup. In the run of 2026-11-19 (stamp 175645) and in the repro of this card (job 20261120-131852-8c93, same RED) the bot log shows: the first town trip at 800 gold buys two town portal scrolls and a Tome of Town Portal (650 of the 800) and only four hp1, so the purse is empty after the first restock (`recovery: start (gold 60 < kit 500)`), later restocks buy two potions each, and the end state has no hp1 left in the belt. The 18 area changes were the `reach:30` level plan walking Blood Moor, Cold Plains, Stony Field, Underground Passage after the heal: legitimate travel, not a loop.

Fix (data): purse 5000 (the purse must cover the kit AND what the bot buys first) and `farm:Blood Moor` as the goal, so the hero stays where the scenario starts and the count really is leave, heal, return. Bot behaviour unchanged. What stays untested here: the recovery start gate (`waiting for the recovery`) with a small purse; cainfix3 fixed it for noGold chunks and no scenario holds a poor hero on a started chunk (U).

## 2. loot.rare-ring-classified: two scenario bugs and one real loot defect (V)

1. Setup order (scenario bug, V): `debug give rin 6 unid` ran BEFORE `debug goto 2`, so the ring was dropped beside the hero in the Rogue Encampment, not in the Blood Moor. The pgrun2 GREEN for the Paladin was therefore an accident of the end-state read, and the REDs of the Sorceress were real (the ring was never in the field). `BuildSetup` now emits ground items after the position verb (`TestGroundItemAfterPosition`, fails on the old order for Paladin and Sorceress).
2. The assertion (scenario bug, V): `item_absent rin ground` read the floor of the level the hero ends in, which is the town, and the observer counts vendor shelves as floor (a ring on a vendor's shelf made it RED, an empty shelf GREEN). New kind `wealth_between` (purse plus stash, the bank chore moves gold): the ring is judged by what it earned (picked up, identified, sold; the rolled rare sells for 2000 to 3000 gold). Without a pick-up the hero ends under 400 (U for a fresh no-fix run).
3. The defect (bot, V): in the Sorceress runs of this card (jobs 20261120-132609-237a, 132834-8254, 132956-4f38) a ring that a monster dropped in the Blood Moor was seen and skipped: `loot: 3 seen, picked 1, skipped 2 (1x filtered: drop [C] no rule matches: Ring)` while the loot shadow said `ground Ring today=ignore rules=pickup rule=g-jewelry-id`. On d2d the floor label of a never-normal base has quality 0 when `ground full` answers nothing (`d2dloot.peekGround` sets 0 on purpose), and the pickit rule "jewelry to id" only matches quality magic or rare. New pickit rule "jewelry unknown quality" (`match.qualityUnknown`, ring/amulet/circlet, unidentified) keeps it for the identify step, as the new rules engine already does (`onUnknown: match`). Test `TestUnknownQualityJewelryIsTaken` (Paladin and Sorceress), run on the builder with the game tables, fails without the rule (V, labrun job 20261120-133220-8c1e).

The other loot scenario changes (stocked belt, farm goal) keep the Sorceress from leaving for a kit trip and a level walk before the loot sweep; they are scenario design, not behaviour.

## 3. The cancelled town-portal cast (design, V unit test, U live)

Evidence: cainchain stamp 201758 and cainfix3 (potion trip out of Tristram, the cast cut by `crowd-retreat`), and again in the chicken runs of this card (`d2d portal: cancelled, nothing clicked (bot: safety drink-life / chicken / crowd-retreat)`, job 20261120-131852-8c93). The guard cancels the whole action context, so `Town.ToTown` cannot retry inside (its context is dead; its walk fallback then handled the trip, which is how the earlier runs still reached the town).

Design (data: `bot.portalRetryWaitSec`, default 3, 0 = off, `internal/env/tuning_bot.go`): when a cast is cancelled BEFORE the click (nothing spent) by a transient safety action (drink life, drink mana, drink rejuv, crowd retreat), `do` carries the safety action out as before, then waits that long and queues ONE town trip back from the same area, unless a trip is already pending (its next tick retries by itself). Chicken, town portal and crowd abandon carry their own way out and are not touched. A second cancelled cast in the same stretch is not retried again (the flag clears on the next successful action): the existing fallbacks (walk home from a neighbour area, waypoint, `healFails` wait) apply. The cancel is recognised by `town.ErrPortalCancelled` (wrapped by the d2d portal adapter) and by `Town.TakePortalCancelled()`, because the quest-step paths return only the guard's cause.

Tests: `TestPortalRetryDue` (table), `TestPortalCancelRetry` (Paladin and Sorceress, knob on and off, once only; fails with the call removed). `go test -race -count=20 -run 'PortalCancelRetry|PortalRetryDue' ./internal/bot` passes on the head box.

Scenario `town.portal-cancel`: NOT possible with the current format. A scenario has `pre` (setup verbs), `trigger` (goals, knobs, board taps) and asserts on the end state; there is no verb or driver that cancels a cast at a chosen moment, and the cancel comes from the guard's own safety decision under a crowd, which no verb produces on demand. The chicken scenarios exercise the path indirectly (the log shows the cancel). A scenario would need a fork verb that injects a hostile pack at the cast (U, candidate for the fork card).

## 4. Widening: cain.chain
Builder main lane, same host, `-id cain.chain -parallel 1`, 80 virtual min each, `writes after start: 0`, record 4 = 0x2019, no bks/bkd left (V where a results.json came back):

| class, seed | run | job | verdict | wall | digest |
|---|---|---|---|---|---|
| Paladin 11 | A | 20261120-141046-7941 | GREEN | 987 s | 2ffabd0c3a935a59 |
| Paladin 11 | B | 20261120-144848-1dc8 | GREEN (exit 0) | about 22 min | not recovered |
| Sorceress 3 | A | 20261120-142938-d413 | GREEN (exit 0) | about 19 min | not recovered |
| Sorceress 3 | B | 20261120-150941-1d57 | GREEN | 871 s | e7d21fd9c2a123b6 |

Verdicts are deterministic on this host: Paladin 11 GREEN twice, Sorceress 3 GREEN twice (with the three of cainchain: Paladin 11 three times, Sorceress 3 twice before). Digests: only two of the four `results.json` files survived the labrun pull (the builder's output directory lost the others, the exit code and the bot log stayed), so no digest pair is compared; the area trails differ between runs (wall-clock decisions, `docs/d2d-determinism.md`), so identical digests were not expected. Code state: run A of Paladin 11 ran before the cancelled-cast flag was forwarded through the d2d and merc town wrappers (the retry was silently off in it); the other three include it. The retry never fired in a chain run that is on record here (U: no log line was searched in the three runs without results).

## Not done / stays red
- The cancelled-cast retry has no live evidence of its own: the wiring and unit tests pass, the chain runs stayed GREEN with it, but no run is known to have hit the path after the flag fix (U). No Proving Ground scenario exists for it (format, section 3).
- chicken.recovery with a small purse (the recovery start gate) is not covered; the scenario now measures the heal and restock, not the poor-hero path.
- The labrun pull dropped some `results.json` files; evidence for those runs is the exit code and the job id.
- Tests: `go test -p 1` of internal/bot, internal/town (skips without the game tables on the head box; the loot test ran on the builder), internal/loot, internal/scenario, cmd/d2bot wiring; `-race -count=20` on internal/loot, internal/scenario and the portal-retry tests; leakcheck 0 hits, contracts and backlog checks clean. A whole-repo `go vet` and test through the builder gate was not run (the builder mirror was held by the chain jobs); the merge step runs it.
