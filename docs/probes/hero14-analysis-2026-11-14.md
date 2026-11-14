# Hero Doofer (Paladin clvl 14 to 15): why it is poor and slow (2026-11-14)

> status: analysis, docs only (task hero14). Tags: V verified (how is said), S secondary, U unverified. No code, no sandbox change.

Data: the hero's container log from 2026-11-13T18:34Z to 2026-11-14T09:54Z (15.3 h, 131k JSON lines, build 0.0.261113174258 on the rebased d2d devp6), plus the last 40 minutes of the older pre-rebase log (17:01Z to 17:41Z). Method: streaming Python over the JSON lines (`minute`, `combat kill`, `killtime`, `area summary`, `task step`, `area skipped`, `action cancelled`, `state`, `chores done`, `sell:`, `shop:`), no whole-file load; every number below is a count over those lines (V). The user's windows (80, 23, 52, 83, 77, 53 kills per 10 minutes) are the few windows in which the hero fought at all; the rest of the time it did nothing.

## 0. The answer in five lines

1. The hero is not slow in combat, it is **parked**. Of 93 ten-minute windows, 10 have any kill; 632 kills in 15.3 h is 0.69 per minute overall; 0 kills in the last 14 hours of the log except one 1-hour burst at 01:00Z (V `minute` rows: kills 8,453 at 22:15Z, 8,628 at 01:17Z, 8,628 at 09:54Z).
2. The cause is one loop (V, section 4): a stale target area (Blood Moor, later Stony Field) is still set while the quest step waypoints the hero into Dark Wood; the guard re-plans every half second ("exploring crossed an area border"), the border-loop breaker bans the stale area for 48 minutes, and the quest route then refuses to start because a banned area is on the road. The refusal text says "banned after repeated deaths"; there were **0 deaths** (V: `deaths:0`; the ban reason in `area skipped` is `border-loop`).
3. 86 % of the run (789 of 918 minutes, V `area summary`) was spent in the Rogue Encampment; 2.7 % (24.7 minutes, V `combat fight` lines) was in-fight time.
4. While fighting, the Paladin is fine: 25 to 44 kills per minute in-fight (V), Zeal on 506 of 652 kills (78 %).
5. Poor: kills gave 86 gold in the whole run (V `area summary` goldPickup); everything else came from one sell of 8 items (1,121 gold, V `sell: sold`) after they sat in the pack for 6.6 hours.

## 1. Time budget

Per game state, minutes (V: one `minute` row per minute, state column; total 917 rows):

| state | minutes | share | what |
|---|---|---|---|
| quest travel | 498 | 54 % | standing at (134,124) in town, the quest step "Go to the Dark Wood" blocked 89,641 times (5 per second) |
| waiting (Paused) | 262 | 29 % | the failsafe ladder gave up ("no progress for 600s ... paused itself, retrying in 5m", step 5 in 8 cases, V) |
| other (Planning blocked: waiting in town) | 54 | 6 % | the planner found every target behind a banned area |
| town chores | 48 | 5 % | 158 trips; 94 of them for "3 healing potions" (nothing affordable: `shop ... broke:true`) |
| recovery (farming gold) | 33 | 3.6 % | Blood Moor at clvl 14, see section 2 |
| quest fight | 22 | 2.4 % | the one-minute visits to Dark Wood, cancelled by the re-plan loop |

Per area, minutes (V `area summary` + `minute` area column): Rogue Encampment 789 (86 %), Blood Moor 35, Dark Wood 35, Cold Plains 7, Stony Field 5, Underground Passage 1 5. Chickens 0, deaths 0, merc not alive (601 `merc: seen ... alive=false` lines).

Town chores: 158 trips, median 7 s, mean 14 s, total 37 min (V `town trip` to `trip budget: trip done`): chores themselves are cheap, 4 % of the run. Game restarts: 49 `Save and Exit` (V; the failsafe ladder, not chickens: `deaths` and chicken potions stayed 0); waypoint trips 1 to 5: 58 (`wp: used`), each a short Dark Wood visit that is cancelled (section 4). Portals: no town portal item until 01:13Z (`inv-tp-dump hastp:false`), 47 times "no town portal: taking the waypoint to town".

Ten-minute windows (V; kills, xp gained, area minutes, dominant states, gold at the end of the window). The first 14 windows and the last 3; windows 14 to 89 are the same pattern except 21:xx to 22:xx (planning blocked, 22:13 sale of a Chipped Emerald for 250), 00:41 onward (Paused) and the 01:00 burst:

| window (13th/14th, Z) | kills | xp | area:minutes | states | gold |
|---|---|---|---|---|---|
| 13 18:34 | 83 | 38 | 2:7, 1:1 | recovery 7, town chores 1 | 54 |
| 13 18:44 | 78 | 137 | 3:5, 4:3, 2:2 | town chores 5, quest travel 5 | 59 |
| 13 18:54 | 56 | 831 | 10:4, 3:2, 4:1 | town chores 5, quest travel 4, quest fight 1 | 29 |
| 13 19:04 | 0 | 0 | 1:9, 5:1 | quest travel 8, quest fight 1 | 29 |
| 13 19:14 | 0 | 0 | 1:9, 5:1 | quest travel 9 | 29 |
| 13 19:24 | 0 | 0 | 1:9, 5:1 | quest travel 9 | 29 |
| 13 19:34 | 0 | 0 | 1:10 | quest travel 9 | 29 |
| 13 19:44 | 80 | 33 | 2:7, 1:3 | recovery 8 (ban shortened to 3 min) | 30 |
| 13 19:54 | 16 | 1 | 1:5, 2:3, 5:2 | quest travel 6 | 1 |
| 13 20:04 .. 20:44 | 0 | 0 | 1:9..10 | quest travel 8..10 | 1 |
| 14 01:0x (the burst, 1 hour) | 175 | 3,232 | 1:47, 5:6, 4:3 | quest travel 50 | 800 |
| 14 05:54 to 09:54 (24 windows) | 0 | 0 | 1:9..10, 5:1 | waiting 28..48 per hour, quest travel 10..30 | 72 |

The last four hours, in short: 0 kills, 0 xp, 0 gold, 100 % in the Rogue Encampment except a 1-minute Dark Wood visit per hour (15 visits, each cancelled within a second of arrival).

## 2. Kills per minute, xp, and the area

In-fight rates (V, `combat fight` lines; kills over fight seconds) and per-kill cost (V `killtime`, excluding gaps over 20 s which are town or ban waits):

| area | kills | xp per kill | gold piles picked | in-fight kills/min | casts/s | median time per kill |
|---|---|---|---|---|---|---|
| Blood Moor (2) | 332 | 0.6 | 86 gold | 25.1 | 0.37 | 2.3 s |
| Cold Plains (3) | 63 | 5.4 | 0 | 43.3 | 1.01 | 2.7 s |
| Stony Field (4) | 66 | 1.8 | 0 | 43.8 | 0.98 | 1.8 s |
| Dark Wood (5) | 111 | 7.8 | 0 | 34.6 | 0.85 | 2.1 s |
| Underground Passage 1 (10) | 65 | 28.7 (Skeleton Archer 115, Vile Hunter 12) | 0 | 16.7 | 0.75 | 2.0 s |

The `combat kill xp` field is the xp delta at the kill (V as logged; the Archer figure includes one 250 pack kill). Blood Moor pays 0.6 xp per kill at clvl 14 and 0.26 gold per kill (86 gold over 334 kills; the xp-gold trace says 0.39, V run): it is a dead field for this hero, and it is where the "Recovery: farming gold" mode sends it (31 minutes; `farm: restricted pick empty, falling back`, 19 times). Per kill time is 2 s; an average Blood Moor kill took 36 s only because 18 kills carry gaps over 20 s (175 minutes of "wait" in `killtime`: the hero standing in town while the ban ran).

Is it in the right area? No. The planner picks sensible areas for clvl 14 (V `planner`: Black Marsh, Cave 2, Cold Plains, Dark Wood, Forgotten Tower, Jail 1, Catacombs 2/3, matching docs/levelplan.md bands), but none ever starts: each is skipped with "the route crosses a banned area" (19 lines, 8 areas at once at 19:44) because Blood Moor, the first field out of town, is banned. Only Blood Moor, Cold Plains, Stony Field, Dark Wood and Underground Passage 1 saw any kills; the planner's fallback and the poverty "Recovery" mode kept it at Blood Moor. New-game resets are not the cause: 49 restarts, none changed the pattern.

xp: 172,338 at 18:35Z, 176,811 at 09:54Z: 4,473 xp in 15.3 h (292 per hour; 3,232 of it in the one burst hour: Underground Passage plus Dark Wood). Clvl 15 was reached at 01:1xZ (threshold 176,132, V doc xp-gold-trace). The first-hour doc's xp model is for clvl 1 to 10; at clvl 14 to 15 the observed per-kill values (V above) are Dark Wood 7.8, Cold Plains 5.4, Underground Passage Vile Hunter 12. Realistic target (U, arithmetic from those values): 14 kills per minute in Dark Wood class ground is 6,500 xp per hour at 7.8 xp per kill and 10,000 at 12; the next level needs far more than that, so one level per several hours is the honest pace at clvl 15 in Act 1 Normal; higher-xp ground (Forgotten Tower, Catacombs, monster level 10 to 12, xp-gold-trace section 6: 35 to 97 per plain kill at low clvl, level gap applies) is what pays and is behind the same ban.

## 3. Gold

Income over the run (V): kills 86 gold (piles); sales 1,371 (Chipped Emerald 250 at 22:13Z; 8 items 1,121 at 01:12Z: Hard Leather Armor 463, Leather Armor 198, two Skull Caps 180 each, Targe 36, Sash 32 x3). Spend 1,330 (V `shop: bought`): Tome of Town Portal 400, 6 portal scrolls 600, 9 Minor Healing Potions 270, 1 Minor Mana Potion 60. Repair 0, belt buys 0, merc revive 0 (needed 650, price 600; gold never above 800 and spent at once on the kit; 120 `merc: revive skipped` lines).

Why low (V + S): gold per kill is 0.39 (Blood Moor) to 2.5 (Dark Wood/Black Marsh), xp-gold-trace section 3: at the first-hour target of 14 kills per minute that is 5 gold per minute in Blood Moor and 30 in Dark Wood. A normal Act 1 hour at clvl 14 is therefore a few hundred to about 1,800 gold from piles (S arithmetic) plus the sell value of drops (0.375 drops per kill, xp-gold-trace Q9); the hero's piles are small because it fought 87 minutes in 15 hours, mostly in Blood Moor. Gold priority works (7 `gold grab: took a pile mid-fight`, pile-left lines are rare: 8 `recovery: loot` lines with 1 pile left for lack of room, V), the piles are just tiny.

The real income leak is the sell chore: the "[9 items to sell]" trip fired 22 times from 18:38Z, but `chores done` lists `sell` done in 1 of 158 trips (skipped list in the other 157, V). The 9 sellable items (pack contents `lbl x4, skp x2, hla, lea, pa1`) were worth about 1,150 gold and stayed in the pack for 6.6 hours until 01:12Z, during which the hero had 1 to 59 gold, no portal, 1 healing potion and could not revive the merc. Cause of the skip: U (`sell` is enabled by `plan.Sell && SellJunk != 0 || marked > 0` and its need check is `t.needSell`, internal/town/chores.go; the bot's own count (`sellable:9`) and the town's `sellable()` disagree; which gate said no is not in the log).

## 4. The loop (root cause, V from the log)

Sequence repeated 1,906 times (all `action cancelled: re-plan` lines are in Dark Wood, area 5; 1,059 with "target area 2", 847 with "target area 4"):

1. The quest step "Go to the Dark Wood" runs (`task step ... outcome ran`), the hero waypoints 1 to 5 (`wp: used 1 -> 5`), state becomes "Quest: fighting (Take the Inifuss bark)".
2. `ob.targetArea` still holds the previous farm target (Blood Moor, later Stony Field). In `replanCause` (internal/bot/loop.go, StateClearing, `snap.Player.Area != ta`), every guard tick returns "exploring crossed an area border" and calls `noteCrossing(ta)`.
3. Five such "crossings" within a minute trip the border-loop breaker (`area skipped ... border-loop`, 997 times Blood Moor, 788 Stony Field, ban 2,880 s, V) although the hero did not cross anything; the trip repeats every 0.5 s because nothing deduplicates it.
4. The run is cancelled, the hero returns to town; the next quest-step attempt fails in the gate: `area not allowed now: Blood Moor (2): banned after repeated deaths`, 47,075 times, then Stony Field 42,572 times (89,647 `task step blocked/danger` lines, 99.9 % of all task steps). `banned()` does not carry the reason, so a border-loop ban prints as a death ban (internal/bot/geargate.go `routeAllowed`). With a waypoint to Dark Wood known (used 58 times) the walk would not cross those fields at all.
5. The quest step is "parked until it expires or the character levels up" (attention line), but the executor keeps retrying at 5 Hz; with every other target also skipped by the ban, the character idles in town, the no-progress failsafe ladder runs (restart game 10, leave and re-enter 13), then pauses itself for 5 minutes at a time (262 minutes Paused).

The older log (pre-rebase, 17:01Z to 17:41Z) shows the same pattern from 17:30Z (V `minute` rows: "Quest: travelling (Go to the Dark Wood)" and "Paused" with kills fixed at 7,706): it is not a product of the rebase.

## 5. Potions and belt (V)

Healing potions used: 9 in 15 h (`potionsUsed`); `no healing potion in the belt` 2 times; 15 `combat potion`; potion buffer skips 26 (`loot: potion skip ... buffer=2`); shop buys 9 Minor Healing (270 gold) and 1 Minor Mana (60), nothing wasted. The belt is not the problem: it holds 4 boxes at the end. The 94 "3 healing potions" town trips (of 158) were useless at 1 to 70 gold (`shop: visit ... broke:true`) but cost only 7 s each (about 15 minutes in total).

## 6. Combat (V)

Zeal 506 of 652 `combat kill` lines (78 %), plain Attack 146 (ranged or unreachable targets: "Fallen Shaman out of melee reach, attacking nearest Fallen", 11), Sacrifice, Holy Fire, Prayer, Vigor, Smite and Charge appear 0 times in the log (the build uses Zeal and Might only: 21 `combat aura up skill=Might`). In-fight casts per second 0.75 to 1.0 in the fields (Zeal swings, the 0.37 in Blood Moor is waiting for monsters). Life floor: lowest observed 48 % (one guard drink at 01:05Z in Underground Passage). The merc is dead from the start (601 `merc: seen ... alive=false`; every one of 193 `zone entry` lines says merc none; `merc=alive` 0 times), so the route guards also block Monastery, Tower and Andariel floors 14 to 16 (66 of the 83 `route guard:` lines name merc=dead or none). The Tower needs "no merc, clvl 15", met at 01:1xZ, so the merc is a cap on later ground, not the cause of today's stall. Skills per level: the log has no skill-page line (U: ask `/api/skills`).

## 7. Headline: realistic rates and the ranked list

Targets, docs/first-hour.md Targets (S/U, written for a fresh character): kills per minute at least 14 in the field, deaths 0, chickens 0, town share 5 to 8 %. This hero: in-field kill rate 25 to 44 per minute (V, above the target), wall-clock 0.69 per minute (V) because 97 % of the time is not field time, town share 86 % (V; the target is under 12 %), deaths 0, chickens 0. Realistic for this hero once unblocked (U): 14 to 20 kills per minute wall-clock in Dark Wood, Underground Passage and Black Marsh class ground, about 6,500 to 10,000 xp per hour at today's observed 7.8 to 12 xp per kill, 30 to 60 gold per minute from piles plus about 1,000 gold per hour from sales.

Ranked time sinks and lost income, per hour of the last 14 hours (V counts above):

| rank | item | lost per hour | proposed fix | owner |
|---|---|---|---|---|
| 1 | Stale `ob.targetArea` while a waypoint or quest step lands the hero elsewhere; `replanCause` fires every tick and `noteCrossing` trips the border-loop breaker (48 min bans) | about 55 minutes (all of it since 19:04Z) | code: clear or rebind `ob.targetArea` when the run starts or the area changes by waypoint; rate-limit `noteCrossing` (one stamp per area and 10 s) so 5 crossings need 5 real crossings; do not trip when `Player.Area` is the area of the current run | d2-worker, new branch `agent/staletarget` (internal/bot/loop.go, areaskip.go) |
| 2 | Misleading ban reason and route check: a border-loop ban shows as "banned after repeated deaths" and blocks a walk that uses a waypoint | enables rank 1 to stay hidden; 89,647 log lines (56 MB) | code: `routeAllowed` / `banned()` use `avoidWhy` (death vs fault) and ignore non-death bans for hops a waypoint jumps over (extend the existing `routeBanned` nav-fault rule to border-loop); log the real reason | same branch |
| 3 | Idle in town instead of farming: planner and failsafe have no fallback when the quest step is blocked; Paused 262 minutes, "Planning blocked" 54 | about 25 minutes | knob or code: a quest step blocked for N minutes yields to the level-plan farm area by waypoint (Dark Wood, Cold Plains, Black Marsh); failsafe "no progress" while in town should retry the plan, not pause; park the blocked step with a real backoff (5 Hz now) | campaign/planner agent, branch `agent/questpark` |
| 4 | Sell chore skipped 157 of 158 trips; 9 items (about 1,150 gold) sat 6.6 h; no portals, potions, merc revive | about 175 gold per hour at the observed price of the sale, and the whole kit | investigate `plan.Sell`, `SellJunk`, `needSell` vs the bot's `sellable` count; test with the pack contents `lbl x4, skp x2, hla, lea, pa1`; make the "[N items to sell]" trip force the sell chore | town agent, branch `agent/sellskip` (internal/town/chores.go) |
| 5 | "Recovery: farming gold" at Blood Moor (0.6 xp, 0.26 gold per kill at clvl 14) and kit spend before the merc revive | about 3 minutes per hour (31 minutes in total) and an unfunded merc | data/knob: poverty recovery picks the best open field for the level (Dark Wood 2.2 gold per kill) or sells first; shop order: merc revive (600) before the tome (400) once clvl 14 or more | goldprio agent, branch `agent/goldprio2` |

Not a lever here (V): potions, belt, combat skill choice, chickens; the Paladin fights well when it is allowed to fight.

## 8. Checklist and open items

- U: why `sell` was skipped (section 3), why the stale `targetArea` is 2 then 4 (it is the farm target of the last run; read the writer of `ob.targetArea`), why `noteCrossing` is not rate-limited (loop.go:1289).
- Live check (not run, no game): after the fix, a clvl 15 Paladin in the Rogue Encampment with Blood Moor and Stony Field banned by `border-loop` takes the waypoint to Dark Wood and stays in the run for 10 minutes: kills above 100, `action cancelled` 0, `area skipped` 0.
- Replay check offline: a unit test with a Runner whose `ob.targetArea` is 4, snapshot area 5, state clearing, guard ticked 10 times: `replanCause` returns "" and `crossLog` stays empty.
