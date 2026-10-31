# Prior art: how existing bots implement the logic

Purpose: give wave-2 engineers (NAV, COMBAT/SURVIVAL, TOWN+LOOT, RUNS+LEVELING, OOG) a behavior spec so they do not rediscover
what Kolbot and friends already learned. Nothing here is code from those projects. Algorithms are paraphrased, numbers are facts.

## Documents
| file | topic |
|---|---|
| [01-leveling.md](01-leveling.md) | per-class leveling, stat/skill order, quest and area order, difficulty change, merc |
| [02-runs.md](02-runs.md) | boss and area runs: routes, standing spots, detection, run-ending, what kills runs |
| [03-pathing.md](03-pathing.md) | walk vs teleport, node spacing, doors, stuck recovery, waypoints, portals, exploring, recorded routes |
| [04-combat.md](04-combat.md) | target choice, immunities, skill switching, positioning, precast, merc, dangerous monsters |
| [05-town.md](05-town.md) | NPC order, heal/buy/repair/identify/sell/stash/gamble, inventory, TP scrolls, stash layout |
| [06-loot.md](06-loot.md) | NIP evaluation, pickup order, ground-item detection, cube recipes |
| [07-safety.md](07-safety.md) | potions, chicken, death, hostile/portal use, watchdogs, escalation |
| [08-oog.md](08-oog.md) | out-of-game flow offline: menus, character select, difficulty, game start/exit |
| [lessons.md](lessons.md) | top ideas to adopt, top failure causes, where our design differs |

## Sources and provenance tags
Source material was read only, never run or built. Cite style: `repo:path:lines`.
| tag | repo | license | note |
|---|---|---|---|
| KB | blizzhackers Kolbot clone (scratchpad `kb/kolbot`), root `d2bs/kolbot/libs/` | NO license file | paraphrase only. Classic D2 1.14d via D2BS. |
| SP | blizzhackers/Kolbot-SoloPlay (`sp/repo`) | GPL-3.0 | paraphrase only, numbers recorded as facts |
| D2BS | noah-/d2bs (`nd/repo`) | see repo | C++ host; path reducers, collision flags, `getPath` semantics |
| KO | dulingzhi_koolo (fork of hectorgimenez/koolo) | MIT | D2R, Go, external memory read + input. Closest to our architecture. |
| TY | Tyniann d2r-offline-farming-bot | source-available, learning allowed | D2R, Go, recorded routes with layout fingerprints |
| BT | aeon0/botty (+ jagarop memread fork) | see repo | D2R pixel bot; only used for health/recovery ideas |
| WEB | Blizzhackers forum / GitHub search | n/a | thin: D2NT is an old AutoIt-era bot with no retrievable source; MMBot, KukBot, d2info (memory reader, supports 1.14d) mentioned. Nothing actionable beyond Kolbot. |

`hectorgimenez_koolo` and `cdennison_*` clones in the scratchpad were empty; the upstream URL for hectorgimenez/koolo returned "not found", so KO (a fork) is the only Koolo source used.

## Reading conventions
- "world coords" = absolute level coordinates as D2 stores them (e.g. Andariel arena x=22549,y=9520). Our `internal/d2` Position is the same space (docs/d2-1.14d-memory.md 2.3).
- Distances are Euclidean in those units unless noted. 1 "tile" in D2 pathing = 1 coordinate unit (a subtile); Kolbot calls them yards.
- Tag **UNVERIFIED** marks anything not read directly from source (my own game knowledge, or inference).
- Package mapping uses PLAN.md names: `internal/{mem,data,d2,d2map,input,nav,bot}`.
- Kolbot and KO are D2R-in-process / D2R-external for different reasons; 1.14d differences are noted where I know them.
