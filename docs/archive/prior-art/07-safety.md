# 07 Safety: potions, chicken, death, hostiles, watchdogs, escalation

Sources: KB:threads/ToolsThread.js (main loop), KB:core/Common/Tools.js (drinkPotion 183-257, getPotion 148-181, checkPing 27-60, exit 134), KB:modules/workers/TownChicken.js, KB:threads/AntiIdle.js, KB:threads/AntiHostile.js (not read in depth), KB:core/Town.js getCorpse 1641-1695, KB:core/Misc.js errorReport 1047-1110, SP:Core/Globals.js (config 572-660), SP:Threads/ToolsThread.js, KO:internal/health/*.go, KO:config.yaml.dist, BT:src/health_manager.py, game_recovery.py, death_manager.py, TY:docs/features/emergency-run-recovery.md.

## 1. The watcher (separate thread/loop, ~20 ms tick in KB)
Each tick while in game, not in town, `gameReady`:
```
if UseHP>0 and hp% < UseHP:       drink healing potion
if UseRejuvHP>0 and hp% < UseRejuvHP: drink rejuv
if LifeChicken>0 and hp% <= LifeChicken: LOG; leave game (chicken)
if UseMP>0 and mp% < UseMP: drink mana
if UseRejuvMP>0 and mp% < UseRejuvMP: drink rejuv
if ManaChicken>0 and mp% <= ManaChicken: leave game
necro: iron golem hp <= 128*IronGolemChicken/100 -> leave game
merc (if UseMerc): mercHP>0 and merc alive:
     mercHP < MercChicken: leave game
     mercHP < UseMercHP: give healing potion; mercHP < UseMercRejuv: give rejuv
if ViperCheck every 250 ms: revived Tomb Viper not owned by me -> leave
if high ping for N seconds (PingQuit) -> leave
anti-idle: periodically send a harmless packet (every 1200..1500 s after 1 min in game)
exceptions inside the loop: error report, then leave game
```
Ordering matters: health actions come before chicken so that a potion can save the run; chicken is tested only after potions are tried each tick.
Potion cooldowns (KB `drinkPotion`): healing/mana: at least 1000 ms between drinks and not while the potion state (HealthPot/ManaPot state flag 'recovering') is active; rejuv: 300 ms; merc rejuv: 2000 ms; other potions 8000 ms. Wait for the cast action to end (mode `SkillActionSequence`) before drinking. Potion choice: highest classid first (better potion), inventory before belt for self, belt only for merc ("Drinking potion from inventory" vs belt). KO: healing interval 4 s, merc heal 6 s, mana 4 s, rejuv 2 s; belt key per column.
`getPotion`: list of items of potion type, sort by classid descending; choose first in inventory (self) or first in belt.
Thresholds (percent):
| source | UseHP | UseRejuvHP | UseMP | LifeChicken | TownHP | HealHP | merc UseMercHP | MercChicken |
|---|---|---|---|---|---|---|---|---|
| KB sorc template | 75 | 40 | 30 | 30 | 0 | 50 | 75 | 0 |
| SP softcore | 80 | 50 | 65 | 10 | 35 | 99 | 75 | 0 |
| SP hardcore | 90 | 65 | 75 | 45 | 0 | 99 | 75 | 0 |
| KO default | 75 | 50 | 10 (mana) | 30 | n/a | n/a | 80 (rejuv 30) | 10 |
| botty (D2R) | chicken at config, 6 s grace after start before chickening, also chicken when an inventory/quest/skill panel is found open |
Recommended defaults for us (our handicap `Tuning`): UseHP 75, rejuv 40, mana 30, chicken 30 (softcore), TownHP 35. Those four ranges are the "survives" range; lower chicken < 20 starts dying on Hell packs (reaction time of the polling loop matters: use <= 100 ms polling).

## 2. Town trips instead of leaving (TownChicken worker, KB)
Trigger: `hp% < TownHP` or `mp% < TownMP` or flag `townCheck` or (if `TownCheck`) belt needs potions or keys needed; must be able to TP (has tool, not in town/Arreat/Uber). Every 100 ms; belt check throttled to 300 ms.
Action: pause other threads, set `Attack.stopClear`, make portal, enter, do chores (heal, buy potions, repair, merc), move to portal spot, wait until >= 4.5 s total, take portal back to the original area; on failure retry using any portal; no TP and alone -> quit; resume. Guard against recursion with a flag.
Our design: `bot/safety` raises an "interrupt: go town" event to the run executor; executor performs `visitTown` and resumes at the same node.

## 3. Leaving the game (chicken exit)
KB calls the host `quit()`; then the out-of-game loop creates a new game after a delay. For us: Esc -> "Save and Exit Game" via UI clicks (08-oog), return to character select, re-enter. Because everything is offline single player the exit saves the char; the next game starts in town (the last town), unlike online where it stays at the town too. Mercenary and items state are preserved.
Important: chicken exit takes a couple of seconds of UI: if the char is about to die, a TP-to-town is often faster: KB only chickens out when TP is not usable; SP TownChicken first.
BT recovery: `go_to_hero_selection` runs up to 30 s: waits through loading black screens, clicks through menus; failing 3x restarts the whole D2R process (`restart_d2r_when_stuck`).

## 4. Death handling
- Detect: player mode == dead (mode id in memory doc; UNVERIFIED value) or life 0 (KO `HPPercent <= 0` -> run finished with event Death). Dead-body: `getCorpse` in the next game: find own corpse unit (player of my name, mode dead) within 20 (or in town), walk to it and click (repeat); retries with random 1-tile nudges after 3 s; give up after 30 s (stop the bot). Without equipment on char (first check `me.getItem(equipped)` empty -> likely died last game): poll corpse up to 2.5 s.
- After retrieving corpse: re-init skills, re-equip (items sit in the corpse = inventory 'cursor', corpse click picks gear back up), clear belt/inv ordering, heal, restock, resume.
- Track `deaths` stat: compare experience at game start vs saved (decrease => died).
- SP only counts deaths (stat update); no tactic change was found.
- Hardcore: never chicken too late: LifeChicken 45 / Town 0 / HP potion 90 (SP). Offline idle "hardcore" is out of scope.
- Softcore death consequences (corpse holds gear, experience penalty, gold loss) are game rules we only need to cope with, not model.

## 5. Hostility, other players, anti-PK
Not relevant offline (single player has no other players). Skip AntiHostile/Party/Rusher threads entirely. Realm-down and ping checks also not relevant (ping ~0).

## 6. Anti-idle and watchdog layers
| layer | KB | TY/KO/BT | ours |
|---|---|---|---|
| keep alive | AntiIdle packet every 20-25 min | n/a | offline single player has no idle kick: skip |
| script watchdog | `Guard` worker, thread memory dumps | TY progress watchdog per segment (reset on new point), 12 s "route_clear_no_progress", 2 s idle retarget/reposition/ exhausted stages | `bot/watchdog`: per-step timeouts (MoveTo 30 s KO), global per-run cap (`MaxGameTime` 2 h SP `MaxGameTime=7200`, `MinGameTime` 400 s), no-progress detectors |
| stuck movement | see 03 | | |
| attack no-progress | skip monster after 15 attacks / 300 count | after 2 s without sending an attack: retarget; +2 s: reposition (<= 4 tiles toward next route point); after <= 2 reposition steps and 2 s -> fail run | same stages |
| UI stuck | `me.cancelUIFlags` before every move | BT chickens when it finds an open panel | Esc until all UI flags clear |
| unexpected area | Loader checks area after transitions | TY fail-closed on wrong area | after each transition assert area id |
| process hang/crash | D2Bot restarts game | BT restarts D2R | `launch` crash detection (PLAN) |

## 7. Error escalation ladder (consensus)
1. Retry the action locally (3x typical; mouse clicks with flash/resync).
2. Reposition / clear nearby monsters / snap to nearest walkable.
3. Abort the current step and the current script; log + screenshot; continue with the next script (Kolbot `Misc.errorReport` appends to a log file with stack and takes a screenshot).
4. Go to town via TP and re-plan (KO: one local clear + second portal try).
5. Leave game (save & exit) and retry the same queue item in a new game (TY, budgeted: bounded retries per session).
6. Restart the client/process (BT after 30 s stuck in menus; D2Bot after "stuck at connecting" 4x).
7. Stop and alert the operator (death retrieval fails after 30 s; no TP and no gold; repeated crashes).
Budgets must be explicit (TY keeps `session budgets`): max retries per run, per game, per hour.

## 8. Specific bugs that shaped the code (why things exist)
- `getUnit` bug: unit queries sometimes return null for visible NPCs/objects (KB handles by trying another act, moving, re-querying). Ours: re-read memory; units may be absent from the hash until the room loads.
- Portals: clicking portal too soon after previous use fails (2.5 s guard); "Dummy" portal names; shared portal wrong owner.
- Item on cursor blocks all clicks -> always check/drop/return.
- UI panels open (inventory/stash/NPC) swallow world clicks -> cancel UI before moving.
- Mercenary dead after portal: `getMerc()` may return null for a moment after arrival (retry 3x with 100 ms sleeps before deciding the merc is dead).
- Lag/desync: character not where the client thinks (melee attackers re-sync with `Packet.flash` every 5..15 attacks). Offline desync is rare, but click-vs-position mismatch in our external design appears as "not moving" and is handled by stuck rules.
- Shapeshifted druid: cannot use town/precast skills (`wereFormCheck`).
- Ping spikes drive all `max(ms, 2*ping)` waits.

## 9. Mapping
| need | package | data |
|---|---|---|
| hp/mp/merc hp percent | `d2` | stats HP (6,7) shifted by 8 bits (mem 2.4), merc unit stats |
| potion use | `input` + `d2` | belt slot -> hotkey 1..4 mapping (hotkeys per belt column are fixed by D2: keys 1-4 drink column 1..4 front potion); inventory potion: right-click on item |
| chicken exit | `input` | Esc menu geometry; `launch` OOG state detection |
| death detect | `d2` | player mode / life == 0; corpse unit (player unit with dead mode) |
| watchdog | `bot` | monotonic clock; progress counters |
