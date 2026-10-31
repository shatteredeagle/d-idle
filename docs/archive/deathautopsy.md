# Death autopsy: G1 attempts 11 to 13 (2026-10-29)

## Findings (top)

1. ROOT CAUSE of the quest-walk deaths (V, code plus logs): `internal/bot/task.go` returned nil for a step that ended
   `task.OutInterrupted`. The guard cancels a step with `safetyInterrupt{drink|portal|chicken}`; `Runner.do` only reacts
   to the cancel cause when the action returned an error, so for a quest step the safety action was never carried out.
   Evidence: in attempts 6, 8, 11, 12 every `combat potion` / `combat town portal cast` line happened in state
   `clearing` (the fighter's own loop); all 10 deaths in state `traveling` have no potion, portal or chicken line, although
   the log has `nav exit` repeated 28 times in 11 s (the executor re-ticking the interrupted step) and the 22:56 death shows the
   character at 7% then 1% life for 8 s with 2 healing potions and 5 portal scrolls in the kit (deaths.json lifeCurve).
   The guard samples life every 55 ms (lifeCurve), so it saw it. Fix: the task closure returns the cancel cause when it is a
   safetyInterrupt (test `TestCampaignWalkCarriesOutSafetyInterrupt`, fails without the fix).
2. Not a level-floor problem: deaths came 2-5 s after entering the Tamoe Highland / Monastery Gate border (x 15141-15186,
   y 5080-5108, same spot every time), life 97% to 0 in about 2-5 s (a pack: hostileNear 15-37). The bot also flapped across
   the 7/26 border (`left level 26 for 27 but ended in 7`) in the middle of that pack. With safety working it now drinks at 55%,
   opens a portal when it has no potion (and a scroll), and leaves the game at 25%.
3. Kiting instead of attacking (attempt 13, user report "ran in circles, never attacked, full mana"; V from the log):
   `combat.kiteEnabled` kited on EVERY tick with life <= `kiteLifePct` (70%) and any melee monster within 10 subtiles. Soaky sat
   at 52-67% life (no regen, potions at 55%), so a Cold Plains pack of 15 produced `combat kiting` every 1-5 s: 23:25:03 to
   23:25:57 zero kills; the moment life reached 85% it killed 11 in 2 s. `nav stalled by monsters, yielding` handed over to
   Clear, which then kited again. Fix: kite only ONE melee monster that is hurting us (life dropped in the last 2 s), never a
   pack. Test `TestPackIsNotKited`. U: why individual casts did not kill during kite gaps (no per-cast log existed); the new
   `casts= kites= approaches=` fields on the `combat fight` line (parser-compatible, appended) and a debug `combat decision`
   line (cast/kite, with reason) will show it next run.
4. Dark Stalker (class 45, also the "Steel Grumble" boss pack with mana burn) is a melee monster at the death distance
   (dist 2-3 in `death diff=`); `monstats rangedtype` is now carried to the post-mortem as `killerType`. The killer name is the
   nearest hostile at the last living snapshot, best effort (U).
5. Not changed (judgment, U): the quest step "Go to the Barracks" at clvl 11 with 86 gold, 2 minor potions and no portal
   scroll (attempt 11) is a kit problem the gear/trip gates should catch; the corpse sits at the border, retrieval was
   abandoned by policy.

## Reconstructed deaths (from ~/d2lod-live/soak24/runs and instances/i4/deaths.json)

| time | area | killer | life curve | kit / notes |
|---|---|---|---|---|
| 22:02:23 | Tamoe 7 | Dark Stalker | 59% to 6% in 5 s | no portal scroll, naked after death 1 |
| 22:05:02 | Monastery Gate | Steel Grumble (boss, mana burn) | 85% to 12% in 5 s | no safety action |
| 22:07:02 | Monastery Gate | Steel Grumble | 81% to 21% in 5 s | |
| 22:08:24 | Tamoe 7 | Dark Stalker | 64% to 8% | 5 portal scrolls bought at 22:03, none used |
| 22:09:38 | Tamoe 7 | Dark Stalker | 97% to 14% in 2.2 s | 5 deaths in 10 min, bans |
| 22:37:39 | Monastery Gate | Dark Lancer | 29% to 9% | arrived at 29% |
| 22:56:47 | Monastery Gate | Dark Stalker | 7% for 5 s, then 1% | 2 hp potions, 5 TPs, 1125 gold lost |
| 22:59:44 | Tamoe 7 | Dark Stalker | 80% to 11% | |

## Death post-mortem schema

`DeathRecord.autopsy` (deaths.json, `/deaths` API `records[].autopsy`) and one JSON object per death in `deaths.jsonl` next
to deaths.json (full DeathRecord, trimmed to the newest 200 lines when above 256 KB). Activity log: a WARN line
`death autopsy: <summary>` before `character died`.

- `class, build`, `killerType` (melee|ranged, monstats), `killerLevel` (monstats level in the difficulty), `killerAffixes[]`
- `hostilesNear` (within `combat.fightRadius` at the last living snapshot), `state` (traveling|clearing|...), `activity`, `goal`
- `lastSafety` (last survival decision in the minute: none means the bot never reacted), `potionsDrunk` (last minute)
- `trail[]`: `{secBefore, life%, mana%, hostiles, state}` about once per second for 60 s; `damagePer5s[]` life percent lost
- `kit`: belt hp/mp/rv, pack hp potions, town portal scrolls, gold carried, equipped items
- `lostXP` (xp lost to restarts), `reaction` (ban or pause set by a rule), `summary` (plain words)
- Corpse outcome stays on the record (`retrieved`, `abandoned`, `why`, `costGold`, `costItems`).

### M0 fields (autopsy, additive; `/deaths` API `records[].autopsy` carries them too)

- `beltHP`, `beltMP`: belt potions of the last living snapshot (memory read through the item catalog, not inferred from drinks); `kit.BeltHP` is the same number. `potionsDrunk60s`: potions drunk in the 60 s trail window (equals `potionsDrunk`).
- `goldCarried`, `stashGold`: stat 14 and 15 of the last living snapshot. `mercAlive`: true/false when a merc unit was listed, null when none was (U whether a dead merc stays listed).
- `routeId`: area ids of the last 6 distinct areas entered, `"2>3>8"`; a trace of where the bot walked, not the planned route (no route id exists in the bot).
- `bug`: `beltHP >= 1 && potionsDrunk60s == 0`.
- Known only afterwards, null until then: `goldNextMinute` (gold on the first minute line after the death), `backInFieldTs` and `secondsToField` (first living snapshot outside town, minus the death time). They are written onto the record in deaths.json and, since deaths.jsonl is append-only, as one extra line `{"followUp":true,"time":<death time>,"goldNextMinute":..,"backInFieldTs":..,"secondsToField":..}` joined to the death by `time`. Readers of deaths.jsonl must skip or merge lines with `followUp`. A death still unresolved when 4 are pending is flushed with nulls.

The dashboard card (web/panels.js `deathCard`) shows summary, kit, damage per 5 s and the reaction.
`goldRecovered` (GOLDSWEEP, additive): gold the post-death gold sweep picked up (`autopsy.goldRecovered`, null when no sweep ran), also appended as a follow-up line `{"followUp":true,"time":<death time>,"goldRecovered":N}`. Readers that merge follow-ups by `time` ignore it safely (cmd/d2soak does); unknown fields are skipped.

Not done: skills cast per death (needs the new per-fight cast counters wired into the record), and the
reaction is not re-written to deaths.jsonl after the line was appended (it is in deaths.json). Dashboard not viewed in a browser.

## Verification

Offline only: `go test -race -p 1` per package (bot, combat, task), `labrun worker` whole repo. No live run; the swallowed-interrupt
fix and the kite rule are U until the next soak shows `combat potion`/`town portal cast` lines in state `traveling` and
`casts=` on fight lines.

## Route guard fields (agent ROUTEGUARD, 2026-10-30; U live)
New `autopsy` fields (all additive): `nearest[]` (5 nearest hostiles: class id, name, distance in subtiles), `portalCast` (a Town Portal cast of the last 30 s: seconds before, duration ms, ok; null when none), `lastLife[]` (last 6 life samples at least 100 ms apart, newest last), `casts5s` and `castSkills` (skill casts and ids in the 5 s before), `mode` (moving or fighting from the bot state), `mercState` (none, alive, dead) and `routeGuard` (the verdict line, e.g. `route guard: blocked monastery (no merc, clvl 12 < 20)`). The same facts are appended to the `death autopsy:` WARN summary so one grep shows a repeat.

FIXED `potionsDrunk60s` (V by code reading, not re-run live): live runs set `DisableObserve = [kill drop potion]` (cmd/d2bot/live.go) because combat emits the potion events, so the observer's belt-diff counter never moved and every death logged `BUG: N healing potions in the belt and none drunk` (02:30 death, `combat potion` at 02:30:15 and :18). The counter is now `Runner.drank`, bumped in `Runner.emit` for every `EvPotion` (combat or observer), and feeds the trail, the autopsy and the `potionsUsed` field of the minute line. Test `TestAutopsyCountsCombatPotionsAndRouteFacts`.
