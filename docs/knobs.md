# Knobs and settings reference

Everything a user can turn: tuning knobs (`internal/env/tuning_*.go`), handicap profiles, `d2bot` / `d2fleet` / `d2soak` / `d2cheat` flags and environment variables.
Tags: V = read from the code in this worktree (cross-checked by grep and by `tools/knobdump`, which prints the knob tables below), U = not verified live.
Written 2026-10-29 from master `6b920b0`. Regenerate the tables between the `knobdump` markers in section 2 with `go run ./tools/knobdump` (reads code only; `go test ./tools/knobdump` fails when they drift).

## 1. How knobs are set (V)

- Every knob has a dotted key `group.name` (groups: `nav combat town loot bot boss merc resource`) and a range. All writes are clamped to `[min, max]` and rounded for int fields (`bot.SetKnob`).
- Order of precedence at start: defaults (optimal) -> handicap profile -> `"tuning"` overrides in config.json. At run time: `PUT /api/v1/instances/{id}/knobs` with `{profile?, overrides?: {key: number|null}}` (null resets to the profile value; the edit is all-or-nothing). `GET` returns every knob with its current value. Doc: docs/api.md.
- config.json (`D2BOT_CONFIG`, default `<data>/config.json`, the `"bot"` object or the whole file; unknown keys are errors): `profile`, `custom` {Name, Desc, Level, Per}, `tuning` {"bot.clearRadius": 20}. Doc: docs/bot.md "Tuning and handicap profiles".
- The "worst" column is the end of the range a profile moves toward (the end the bot is least efficient at). Knobs where the default already sits on that end do not change under any profile (list in section 7).
- Verified live: the key -> field -> clamp plumbing is tested offline (`go test ./internal/env ./internal/bot -run EveryKnob`). The EFFECT of an individual knob on a live game is mostly NOT verified: only the behaviours the domain docs mark VERIFIED LIVE are (combat death/rest/potions: docs/combat.md; town/nav walking: docs/town.md, docs/nav.md; leveler points: docs/leveler.md; resting before packs: docs/resource.md). No knob has a recorded live A/B test of its range. Ranges are offline-simulation windows (Sorceress, Normal Act 1), tagged U for anything else.
- The doc each group is explained in: merc -> docs/merc.md; nav -> docs/nav.md; combat -> docs/combat.md; town and loot -> docs/town.md, docs/items.md; bot -> docs/bot.md, docs/leveler.md, docs/merc-farm.md; resource -> docs/resource.md.
- Counts: nav 20 knobs, combat 38, boss 1, merc 10, resource 28 (fixed in docs/nav.md, docs/combat.md, docs/resource.md, docs/bot.md on 2026-10-29).

## 2. Tuning knobs (V: defaults, ranges, readers from the code)

Column "readers" is the heuristic count of non-test uses outside `internal/env`: `accessor` = read through a `NavTuning` method (`Poll()`, `Replan()`, `Stuck()`, `WalkTimeout()`, `ExploreLimit()`, `LoadWait()`, `PanelWait()`, each with 1 to 10 callers); `local var` = read as a local copy (`rt.WaitMaxSec`, internal/combat/resource.go:293). Units: subtiles (5 per tile), ms, s, %, gold. Bool knobs are 0/1.
Notes on defaults that are switches: `combat.missileDodge` default 0 (U, off until verified live), `combat.kiteEnabled` 1, `town.keyTarget` 0 (never buy keys), `town.sellJunk` 1.


<!-- knobdump:begin -->
### nav.*

| key | default | range | unit | worst (rookie) | readers | what it changes |
|---|---|---|---|---|---|---|
| nav.pathQuality | 1 | 0.1..1 | 0..1 | 0.1 | 1 | 1 = shortest, smoothed paths. Lower values make paths longer and wobblier (inflated A* heuristic, no smoothing below 0.5, sideways waypoint jitter) but still reach the goal. |
| nav.sloppiness | 8 | 0..14 | subtiles | 14 | 1 | Largest sideways displacement of a waypoint at path quality 0; scaled by (1 - quality). |
| nav.waypointSpacing | 8 | 4..24 | subtiles | 4 | 3 | Distance between walking clicks. Smaller spacing follows the path more exactly but clicks more often. |
| nav.arriveTolerance | 2 | 1.5..6 | subtiles | 6 | 3 | A walk ends within this distance of the goal. |
| nav.pollMs | 100 | 50..400 | ms | 400 | accessor | Observation period while walking; larger values react later to stops and monsters. |
| nav.replanMs | 1500 | 500..8000 | ms | 8000 | accessor | Earliest time a changed map or monster layout triggers a new plan. |
| nav.stuckMs | 1200 | 600..4000 | ms | 4000 | accessor | No progress for this long starts the recovery ladder. |
| nav.maxRecoveries | 6 | 2..12 | count | 2 | 1 | Stuck-ladder rungs tried before the walk fails with ErrStuck. |
| nav.wallClearance | 3 | 0..6 | cells | 0 | 2 | Cells next to walls that cost extra, keeping paths off corners. |
| nav.wallWeight | 0.75 | 0..3 | cost/cell | 0 | 2 | Extra path cost per cell of missing wall clearance. |
| nav.monsterRadius | 8 | 0..16 | subtiles | 0 | 3 | Paths keep this distance from monsters and NPCs when a detour is cheap. |
| nav.monsterPenalty | 6 | 0..20 | cost | 0 | 2 | Extra cost of a cell at a monster, fading to zero at the avoid radius. |
| nav.exploreInformed | 1 | 0..1 | bool | 0 | 1 | 1 prefers large, forward frontiers; 0 always walks to the nearest frontier. |
| nav.exploreBucket | 10 | 5..30 | subtiles | 30 | 1 | A frontier within this distance of a place already visited counts as inspected. |
| nav.walkTimeoutSec | 120 | 30..600 | s | 30 | accessor | Hard cap for one GoTo. |
| nav.exploreTimeoutSec | 300 | 30..1800 | s | 30 | accessor | Hard cap for one Explore. |
| nav.exitPush | 8 | 3..20 | subtiles | 3 | 1 | How far past a border gap or exit tile the walk continues to trigger the level change. |
| nav.loadWaitSec | 20 | 5..90 | s | 5 | accessor | How long to wait through a loading screen. |
| nav.panelWaitMs | 2500 | 800..8000 | ms | 800 | accessor | How long to wait for a clicked panel (waypoint) to open. |
| nav.useWaypoints | 1 | 0..1 | bool | 1 | 4 | 1: travel, the corpse run and the way back from town take an activated waypoint instead of walking (saves minutes and walking deaths; the route guard still applies to every area on the way). 0 walks everything. |
| nav.wpMinDistance | 2 | 2..8 | levels | 2 | 2 | A waypoint trip is taken only when it skips at least this many level hops; shorter routes walk. 2 is the shortest trip that saves a level. |
| nav.wpCooldownSec | 10 | 10..20 | s | 10 | 2 | game.exe silently refuses a waypoint travel for 10 s after any level change; the bot waits this long since arriving before clicking a row. Lower is not useful. |
| nav.cautionRadius | 20 | 8..40 | subtiles | 8 | 1 | A corpse run stops and reports when a living monster comes this close; its paths avoid monsters at twice the usual radius. |

### combat.*

| key | default | range | unit | worst (rookie) | readers | what it changes |
|---|---|---|---|---|---|---|
| combat.drinkLifePct | 55 | 35..80 | % | 35 | 6 | Life percent that triggers a healing potion. |
| combat.drinkRejuvPct | 35 | 20..60 | % | 20 | 4 | Life percent that triggers a rejuvenation potion. |
| combat.drinkRejuvBossPct | 50 | 0..70 | % | 0 | 0 | Life percent that triggers a rejuvenation potion in a fight against champions, uniques or bosses (danger 1+); 0 = off. SME advice, EXP. |
| combat.championEarlyPct | 10 | 0..25 | pp | 0 | 0 | Percentage points added to the healing potion line and the rejuvenation (boss) line while champions (not bosses) are in range, in Normal below character level 15. Champions have about 3x life and +50% damage. 0 = off. SME rule of thumb, EXP. |
| combat.chickenPct | 25 | 15..45 | % | 15 | 4 | Life percent at or below which the bot leaves the game (Normal, Nightmare). |
| combat.chickenHellPct | 30 | 20..50 | % | 20 | 2 | Same as chicken, in Hell. |
| combat.drinkManaPct | 25 | 5..60 | % | 5 | 2 | Mana percent that triggers a mana potion. |
| combat.townPotionMin | 2 | 0..4 | count | 0 | 4 | Healing potions left in the belt at which the bot wants a town trip. |
| combat.crowdN | 8 | 0..20 | count | 20 | 6 | Hostiles within 8 subtiles that make a crowd: below it the normal drink and chicken rules apply alone. 0 turns the crowd rule off. |
| combat.crowdPortalPct | 70 | 40..90 | % | 40 | 3 | In a crowd, open a town portal at or below this life percent when a scroll exists and fewer than 3 monsters are adjacent (a 2 s cast fails when surrounded). |
| combat.crowdDrinkPct | 65 | 40..85 | % | 40 | 2 | In a crowd, drink a healing potion below this life percent (the portal cannot be cast when surrounded); the normal rule drinks at 55. |
| combat.crowdAbandonN | 15 | 0..30 | count | 30 | 2 | Hostiles within 8 subtiles on a travel or quest walk that make the bot abandon the walk at once, retreat 20+ subtiles and open a portal. 0 turns it off. |
| combat.potionGapMs | 1000 | 900..3000 | ms | 3000 | 4 | Minimum time between healing or mana potions. |
| combat.rejuvGapMs | 300 | 250..1500 | ms | 1500 | 1 | Minimum time between rejuvenation potions. |
| combat.reactionMs | 120 | 40..700 | ms | 700 | 1 | Delay before the bot acts on a survival event. |
| combat.pollMs | 60 | 30..400 | ms | 400 | 6 | How often the fight loop looks at the game. |
| combat.targetQuality | 100 | 20..100 | % | 20 | 3 | 100 picks the best target; lower picks the nearest more often and loses ticks. |
| combat.rangeMargin | 3 | 0..8 | subtiles | 0 | 1 | Distance kept inside a skill's maximum range. |
| combat.clearRadius | 40 | 15..80 | subtiles | 15 | 1 | Radius around the start point cleared when the caller gives none. |
| combat.fightRadius | 22 | 10..40 | subtiles | 10 | 11 | A hostile monster this close counts as a fight. |
| combat.maxTargetSec | 45 | 20..90 | s | 90 | 1 | Give up on one target after this long. |
| combat.noProgressSec | 12 | 5..30 | s | 30 | 1 | Give up on a target whose life did not drop for this long. |
| combat.clearMaxSec | 180 | 30..600 | s | 30 | 2 | Upper bound of one Clear call. |
| combat.castGapMs | 450 | 250..1200 | ms | 1200 | 10 | Minimum time between two casts. |
| combat.lowManaWaitMs | 700 | 300..3000 | ms | 3000 | 1 | Pause when out of mana. |
| combat.staticCasts | 4 | 0..8 | count | 0 | 1 | Static Field casts that open an elite or boss fight. |
| combat.staticCastsBR | 0 | 0..8 | count | 8 | 1 | Static Field casts that open the Blood Raven fight; 0 keeps the mana for the attack skill (Charged Bolt for the Sorceress). |
| combat.staticFloorNorm | 25 | 10..60 | % | 60 | 1 | Target life percent below which Static Field is not cast. |
| combat.staticFloorNM | 33 | 10..60 | % | 60 | 1 | Same in Nightmare. |
| combat.staticFloorHell | 50 | 10..70 | % | 70 | 1 | Same in Hell. |
| combat.novaManaFloor | 25 | 0..60 | mana | 60 | 1 | Frost Nova on a crowd of 3 or more is not cast with less mana than this, so the mana stays for the bolts. |
| combat.kiteEnabled | 1 | 0..1 | bool | 0 | 2 | 1 backs away from melee monsters that get close. |
| combat.kiteDistance | 10 | 4..20 | subtiles | 4 | 1 | Back away when a melee monster is closer than this. |
| combat.kiteLifePct | 70 | 40..100 | % | 40 | 0 | Kite normal monsters only below this life percent. |
| combat.kiteMaxSteps | 6 | 0..12 | count | 0 | 4 | Retreat steps per target before standing to fight. |
| combat.kiteRetreat | 12 | 6..20 | subtiles | 6 | 10 | Length of one retreat step. |
| combat.dodgeRadius | 4 | 0..8 | subtiles | 0 | 6 | Step away from hostile missiles closer than this. |
| combat.missileDodge | 0 | 0..1 | bool | 0 | 1 | 1 sidesteps fire walls, lightning, hydra and chaos missiles (unverified live, off by default). |
| combat.curseRetreatPct | 60 | 30..90 | % | 30 | 2 | With Amplify Damage or Iron Maiden on the player, retreat and drink below this life percent. |
| combat.maxApproaches | 12 | 4..30 | count | 4 | 1 | Approach steps per target before it is dropped as unreachable. |
| combat.reachSec | 15 | 6..40 | s | 40 | 1 | Give up on a target that was not shot at once in this long (unreachable, no line of sight). |
| combat.ignoreMemorySec | 90 | 20..300 | s | 20 | 1 | How long an ignored target is left alone (the memory doubles with every repeat). |
| combat.auraCheckMs | 1500 | 500..5000 | ms | 5000 | 1 | How often the fight loop re-reads the active aura and re-applies it. |
| combat.leftFailSec | 60 | 10..300 | s | 10 | 1 | After a left skill could not be selected, wait this long before trying it again. |
| combat.buffRecastSec | 150 | 60..300 | s | 300 | 1 | Recast armors and shields this long after the last cast. |

### town.*

| key | default | range | unit | worst (rookie) | readers | what it changes |
|---|---|---|---|---|---|---|
| town.healBelowLifePct | 70 | 10..100 | % | 10 | 1 | Visit the healer in town when life is below this. |
| town.healBelowManaPct | 30 | 0..100 | % | 0 | 1 | Visit the healer when mana is below this; 0 disables. |
| town.tripMinHPPotions | 2 | 0..8 | count | 0 | 1 | Go to town when fewer healing potions remain (belt and pack). |
| town.tripMinMPPotions | 1 | 0..8 | count | 0 | 2 | Go to town when fewer mana potions remain (only if the belt plan has a mana column). |
| town.tripBeltColumnPct | 50 | 0..100 | % | 0 | 2 | Go to town when a planned belt column is filled below this. |
| town.tripPackUsedPct | 80 | 40..100 | % | 100 | 4 | Go to town when the backpack is this full. |
| town.tripRepairPct | 20 | 0..60 | % | 0 | 1 | Go to town when an equipped item is below this durability. |
| town.beltFillPct | 100 | 25..100 | % | 25 | 2 | Refill each potion type to this share of its planned columns. |
| town.tpScrollTarget | 2 | 1..20 | count | 1 | 3 | Scrolls to hold when no tome is owned. |
| town.tpTomeMinGold | 450 | 0..5000 | gold | 5000 | 2 | Buy a Tome of Town Portal when gold is at least this. |
| town.idScrollTarget | 2 | 0..20 | count | 0 | 4 | Scrolls to hold when no tome is owned. |
| town.minGoldReserve | 100 | 0..1e+04 | gold | 1e+04 | 4 | Optional purchases never spend below this. |
| town.keyTarget | 0 | 0..12 | count | 0 | 6 | Keys to hold; 0 never buys keys. |
| town.repairBelowPct | 30 | 0..90 | % | 0 | 1 | Repair when any equipped item is below this durability. |
| town.stashGoldBase | 300 | 0..2e+04 | gold | 2e+04 | 3 | Deposit inventory gold above this plus level times stashGoldPerLvl. Death takes ALL carried gold (LIVE G1 attempt 12: 1125 to 0), so keep little. |
| town.stashGoldPerLvl | 50 | 0..2e+04 | gold/lvl | 2e+04 | 3 | Deposit inventory gold above stashGoldBase plus level times this. |
| town.beltBuyLevel | 10 | 0..40 | clvl | 0 | 4 | From this level on a town visit buys the cheapest belt with more potion slots than the worn one (after the potion kit, never below the gold reserve plus a minimal kit). 0 = never. docs/sme/potion-policy.md. |
| town.sellJunk | 1 | 0..1 | bool | 0 | 1 | 1 sells items the pickit rates as sell; 0 keeps them. |
| town.rebuyGearTier | 1 | 0..3 | tier | 0 | 1 | After a death: 0 potions only, 1 weapon and body armor, 2 also helm/boots/gloves/belt, 3 also shield. |
| town.rebuyPotionPct | 40 | 0..100 | % | 0 | 1 | Share of the rebuy budget potions may use before gear. |
| town.rebuyItemMaxPct | 60 | 10..100 | % | 10 | 1 | One rebuy item may cost at most this share of the budget. |
| town.npcStopDist | 4 | 2..10 | subtiles | 10 | 1 | Distance from an NPC at which the walk ends and the click starts. |
| town.npcOpenTries | 3 | 1..6 | count | 1 | 2 | Attempts to open an NPC dialog before the chore is skipped. |
| town.uiTimeoutMs | 2500 | 800..8000 | ms | 800 | 2 | Wait for a panel flag after a click. |
| town.clickSettleMs | 150 | 40..800 | ms | 800 | 7 | Pause after a UI click before memory is re-read. |
| town.moveTries | 3 | 1..6 | count | 1 | 3 | Attempts to move one item between grids (each verified in memory). |
| town.portalSettle | 4500 | 1000..8000 | ms | 8000 | 1 | Wait after arriving in an area before using a portal. |
| town.portalReuseDist | 40 | 0..100 | subtiles | 0 | 1 | ToTown walks into an open portal this close instead of reading a scroll. |
| town.greetingWaitS | 90 | 0..180 | s | 180 | 1 | Wait for an NPC menu after a click (first contact plays a long speech). |
| town.greetingSkip | 0 | 0..3 | mode | 3 | 2 | How the first-contact greeting is skipped when the menu is not open 1 s after the click: 0 wait it out (greetingWaitS), 1 click the NPC again, 2 Esc then click again, 3 left click on the NPC tile without hover. docs/npc-skip.md. |
| town.futileGoldStep | 100 | 0..5000 | gold | 5000 | 3 | After a town visit could not fix a trip reason, do not ask again until gold grew by this. |
| town.futileMinutes | 10 | 0..120 | min | 120 | 2 | A trip reason a visit left unfixed is not raised again for this long. |
| town.spareKeep | 4 | 0..30 | count | 0 | 1 | A full pack sells spare potions and identify scrolls beyond this many per kind (they fill the pack but are never belted). |
| town.sellMaxS | 60 | 15..300 | s | 15 | 2 | Time the sell chore alone may take before it is cut off. |
| town.choreBudgetS | 150 | 30..600 | s | 30 | 1 | Time one Chores call may take before it gives up. |

### loot.*

| key | default | range | unit | worst (rookie) | readers | what it changes |
|---|---|---|---|---|---|---|
| loot.pickRadius | 30 | 5..60 | subtiles | 5 | 1 | Ground items within this distance are considered. |
| loot.pickStrictness | 1 | 0..3 | level | 3 | 2 | 0 loose (also items worth selling), 1 normal, 2 keepers only, 3 gold/runes/gems/uniques only. |
| loot.pickTries | 3 | 1..6 | count | 1 | 1 | Click attempts per item. |
| loot.walkDist | 4 | 2..8 | subtiles | 8 | 1 | Walk to within this distance before clicking an item. |
| loot.confirmMs | 1500 | 400..4000 | ms | 400 | 1 | Wait after a click for the item to leave the ground. |
| loot.gapMs | 150 | 0..1000 | ms | 1000 | 1 | Pause between two pickups. |
| loot.goldMinStack | 10 | 0..2000 | gold | 2000 | 1 | Ignore gold piles smaller than this. |
| loot.freeCellsKeep | 2 | 0..12 | cells | 0 | 1 | Inventory cells that must stay free after a pickup. |
| loot.potionBuffer | 0 | 0..10 | count | 0 | 1 | Extra needed-type potions carried in the pack beyond the belt. |
| loot.rejuvSpare | 3 | 0..6 | count | 0 | 4 | Rejuvenation potions carried in total (belt plus pack); more are left on the ground or sold, Full Rejuvenation preferred over regular. Pack ones are moved to a free belt slot in town so the belt hotkey can drink them. |
| loot.lowGoldBelow | 2000 | 0..1e+05 | gold | 1e+05 | 1 | Below this gold, also pick items worth selling. |
| loot.minValuePerCell | 100 | 0..5000 | gold | 5000 | 1 | Sell value per cell an item needs to be picked in low-gold mode. |
| loot.valuableMinGold | 1500 | 100..1e+05 | gold | 1e+05 | 2 | Unequipped items worth at least this are valuables (stash before dangerous trips). |
| loot.valuableMinPrio | 70 | 1..100 | prio | 100 | 4 | Items whose pickit rule prio is at least this are valuables. |
| loot.carryValuables | 0 | 0..20 | count | 0 | 1 | Valuables allowed in the pack outside town (insurance: the rest stay in the stash). |
| loot.poorGoldBelow | 300 | 0..1e+04 | gold | 1e+04 | 1 | Below this gold every gold pile and junk weapon or armor worth selling is picked. |
| loot.poorMinSell | 6 | 0..500 | gold | 500 | 2 | Estimated sell value a junk item needs to be picked in poor mode. |
| loot.maxPerCall | 30 | 1..100 | count | 1 | 1 | Upper bound on pickups in one PickUp call. |

### bot.*

| key | default | range | unit | worst (rookie) | readers | what it changes |
|---|---|---|---|---|---|---|
| bot.tickMs | 100 | 100..1000 | ms | 1000 | 3 | How often the bot re-plans between actions. Slower means more idle gaps. |
| bot.watchMs | 50 | 50..300 | ms | 300 | 5 | How often events, hazards and death are checked while an action runs. Above 300 the bot dies on Hell packs. |
| bot.reactionMs | 0 | 0..300 | ms | 300 | 1 | Extra delay before reacting to low life or a chicken condition. |
| bot.resupplyPotions | 2 | 2..4 | potions | 4 | 2 | Healing potions left in the belt that trigger a town trip. Higher means more trips. |
| bot.invFullPct | 85 | 40..85 | % | 40 | 2 | Share of occupied inventory cells that triggers a town trip. Lower means more trips. |
| bot.levelMargin | 0 | 0..8 | levels | 8 | 7 | Levels above a step's enter level the planner requires. Higher is safer but slower. |
| bot.leaveSlack | 0 | 0..6 | levels | 6 | 5 | Levels past a step's leave level before moving to the next area. Higher wastes time in easy areas. |
| bot.corpseRetries | 1 | 0..1 | count | 0 | 1 | Max attempts to recover the corpse after a softcore death. 0 abandons the gear. |
| bot.portalSettleMs | 600 | 600..4500 | ms | 4500 | 1 | Pause in town after chores before going back out. |
| bot.clearRadius | 30 | 12..30 | subtiles | 12 | 13 | Reach of one clear sweep. Smaller means more walking between sweeps. |
| bot.corpseValueMin | 1000 | 1..2e+04 | gold | 2e+04 | 4 | Estimated value of the gear on the corpse below which it is abandoned. Higher abandons more. |
| bot.spiralDeaths | 4 | 3..8 | deaths | 8 | 4 | Deaths in one area within the spiral window that make the bot avoid that area (one fewer when the area is clearly too high for the level or the character died with no potions and no gold). Dying is normal: the default allows three deaths on the same goal. Higher tolerates more. |
| bot.dangerBanMin | 20 | 5..120 | min | 120 | 1 | How long the spiral breaker keeps the character out of an area. Longer stalls the goal more. |
| bot.starterBanMin | 3 | 1..30 | min | 30 | 1 | A ban of the first field of an act is cut to this once the character stands in town because of it (it is the only way out). |
| bot.repeatSiteBanMin | 30 | 0..120 | min | 120 | 2 | A second death in one area within the repeat-site window, or a death within the repeat-site gap of the previous one, bans the area this long and writes a planner dump. 0 turns the rule off. Dying is normal: this changes where the bot farms, not how careful it is. |
| bot.repeatSiteEscalate | 1 | 0..1 | bool | 1 | 1 | 1: the second repeat-site ban of the same area within 12 hours lasts 4 times the ban length, the third and later 12 times (capped at 6 hours). It stops a bot from walking back into the same death spot every half hour. 0 keeps every ban at the base length. |
| bot.repeatSiteWindowMin | 30 | 5..120 | min | 120 | 2 | Two deaths in one area within this many minutes make it a repeat site. |
| bot.repeatSiteGapSec | 180 | 30..600 | s | 600 | 1 | A death within this many seconds of the previous death (any area) makes the area a repeat site. |
| bot.corpseMinValue | 50 | 0..5000 | gold | 5000 | 3 | Below this estimated gold value the in-field corpse run is skipped entirely (the stricter bot.corpseValueMin still applies above it). |
| bot.navSkipSec | 150 | 30..900 | s | 900 | 1 | How long a navigation fault (stuck, no exit to the area) keeps the planner out of an area. Short on purpose: it is a bot fault, not danger. |
| bot.areaIdleSec | 60 | 15..600 | s | 600 | 1 | When every candidate area is skipped or banned, the bot waits this long in town at most, then farms the least bad area anyway (a death streak waits at least the starter ban cap). |
| bot.spiralMinutes | 15 | 5..15 | min | 5 | 5 | Window in which the deaths are counted. Shorter forgets sooner. |
| bot.pauseDeathsPerHour | 8 | 8..20 | deaths | 20 | 2 | Deaths in the last hour after which the bot pauses itself and asks the user. |
| bot.povertyTier | 0 | 0..2 | tier | 2 | 3 | Gear tier bought after an abandoned corpse: 0 is the cheapest viable weapon and armor, potions first. |
| bot.goldFloor | 2000 | 0..2000 | gold | 0 | 6 | Gold town visits must not spend, kept to rebuy gear after a death. |
| bot.pointsRetrySec | 30 | 30..300 | s | 300 | 2 | Wait before spending stat and skill points again after an attempt that did not finish. Longer leaves points unspent. |
| bot.gearCheckSec | 60 | 60..600 | s | 600 | 1 | Minimum time between gear passes in town (equip better items, shop). Longer wears worse gear for longer. |
| bot.gearMinGold | 150 | 150..1500 | gold | 1500 | 3 | Gold needed before the bot shops for weapons and armor. Higher shops rarely. |
| bot.gearBudgetPct | 70 | 20..70 | % | 20 | 1 | Share of the gold above the potion reserve one shopping pass may spend on gear. |
| bot.gearReserve | 100 | 0..100 | gold | 0 | 1 | Gold kept for potions when shopping for gear. Lower risks starving after a shopping spree. |
| bot.sellMinItems | 4 | 4..12 | items | 12 | 3 | Sellable pack items that justify a town trip for a character without gold. Higher carries loot longer. |
| bot.difficultyBanMin | 60 | 60..600 | min | 600 | 3 | Minutes the bot stays in its difficulty after a switch to the next one did not happen. |
| bot.savePointMin | 20 | 0..120 | min | 20 | 1 | Minutes in one game before the bot, in town and idle, saves and re-enters it (a crash loses what came after the last save); 0 is off. |
| bot.heapLimitMB | 4096 | 0..8192 | MB | 4096 | 1 | Go heap in use above which d2bot writes a heap profile and exits for a supervisor restart (a runaway allocation would otherwise end in the OOM killer); the GC gets a soft limit 25 percent above it. 0 is off. Normal runs use under 100 MB. |
| bot.mercRetryMin | 10 | 1..60 | min | 10 | 1 | Minutes between merc chore attempts; a failed hire or revive is not retried sooner. |
| bot.mercHireMinGold | 400 | 0..2000 | gold | 400 | 1 | Gold in Act 2 town below which no hire is attempted. |
| bot.mercReviveUnverified | 0 | 0..1 | bool | 0 | 3 | 1 lets the town visit click the revive dialog, which is UNVERIFIED live (docs/merc-farm.md). Off until a live probe. |
| bot.cubeJobs | 0 | 0..1 | bool | 0 | 3 | 1 runs the gold-free gem and rune upgrade chore in town. UNVERIFIED live (docs/items.md), so off. |
| bot.cubeMaxJobs | 6 | 1..12 | count | 6 | 1 | Transmutes per town visit when the cube chore is on. |
| bot.cubeRuneCap | 10 | 1..10 | rune | 10 | 1 | Highest rune number the chore produces (10 = Thul; Dol to Hel and above are ladder-flagged, docs/items.md Q1). |
| bot.cubeGemCap | 2 | 0..4 | grade | 2 | 1 | Highest gem grade the chore produces: 0 chipped, 1 flawed, 2 normal, 3 flawless, 4 perfect. |

### boss.*

| key | default | range | unit | worst (rookie) | readers | what it changes |
|---|---|---|---|---|---|---|
| boss.andarielMinLevel | 15 | 12..22 | levels | 15 | 9 | Character level at which the Normal campaign attempts Andariel; the kit gate (potions, town portal, antidotes) applies on top. Lower hits the XP wall of Act 1 later, higher is safer for the fight. 22 is the old hold. |
| boss.brCasterLevel | 9 | 6..14 | levels | 9 | 0 | Character level a Sorceress, Druid or Amazon needs before the campaign attempts Blood Raven (the free Rogue). SME: below 8 it is a death trap, 9 with the kit is the first clean try. |
| boss.brMeleeLevel | 8 | 6..14 | levels | 8 | 0 | Same floor for the Necromancer (skeletons and Clay Golem do physical damage, her 0 percent resist) and the melee classes. |
| boss.brPotions | 4 | 2..8 | count | 4 | 0 | Healing potions (belt and pack) the kit needs before each Blood Raven attempt. |
| boss.brManaPotions | 2 | 1..6 | count | 2 | 0 | Mana potions a caster needs before each Blood Raven attempt (she resists every element by half: a mana race). |
| boss.brTPCharges | 2 | 1..5 | count | 2 | 0 | Town portal tome charges or scrolls needed before each attempt. The bot cannot read charges yet (U), then any portal counts. |
| boss.brGold | 250 | 0..1500 | gold | 250 | 0 | Gold needed at each attempt: a retreat trip refills potions and a portal, and an empty purse made G1 attempt 8 drain its gold. Below it the character levels and farms first. |
| boss.brMaxRetreats | 2 | 1..4 | count | 2 | 2 | Retreats to the entry (drink, regroup) one attempt may use. The last one ends the attempt with a town portal; the next attempt waits for more levels. |

### merc.*

| key | default | range | unit | worst (rookie) | readers | what it changes |
|---|---|---|---|---|---|---|
| merc.reviveMaxPerHour | 2 | 0..12 | count | 2 | 2 | Most merc revives in one hour; a merc that keeps dying costs more than it saves. |
| merc.reviveGoldMult | 2 | 1..10 | x | 2 | 2 | Revive only with gold of at least this many times the price (and the town portal and potions kit). |
| merc.stopDeathsPerHour | 3 | 1..12 | count | 3 | 2 | More merc deaths than this in an hour stop reviving until gold reaches the stop multiple. |
| merc.stopGoldMult | 5 | 1..20 | x | 5 | 2 | While reviving is stopped by deaths, revive only with gold of at least this many times the price. |
| merc.hireGoldMult | 3 | 1..10 | x | 3 | 2 | Hire or replace only when gold after the kit is at least this many times the quote. |
| merc.freeRogueOnly | 0 | 0..1 | bool | 0 | 2 | 1 never pays for the Act 1 Rogue, even once Blood Raven is skipped for good; 0 allows that fallback. |
| merc.replaceAtAct2 | 1 | 0..1 | bool | 1 | 2 | 1 swaps the Rogue for a Desert Mercenary with the class aura at the first Lut Gholein visit per difficulty. |
| merc.swapGainPct | 10 | 0..100 | % | 10 | 4 | A merc item is swapped only when its score is this much better. Lower swaps more often. |
| merc.reqStrDex | 1 | 0..1 | bool | 1 | 2 | 1 checks the merc's own Strength and Dexterity against gear (d2decomped finding, UNVERIFIED live); 0 checks only level. |
| merc.potionLifePct | 40 | 10..80 | % | 40 | 3 | A healing potion goes to the merc only below this life percent in a non-trivial fight. |
| merc.enabled | 0 | 0..1 | bool | 0 | 1 | 1 runs the mercenary policy in the bot (revive decision, merc state for the route guards, status line); 0 leaves the bot exactly as it was. Off until the merc reader is verified live (docs/merc-wire.md). |

### campaign.*

| key | default | range | unit | worst (rookie) | readers | what it changes |
|---|---|---|---|---|---|---|
| campaign.imbueMinLevel | 20 | 8..40 | levels | 20 | 4 | Below this character level the optional Tools of the Trade quest (kill Smith in the Barracks, Charsi imbues an item) is skipped; Sorceress, Necromancer, Druid and Amazon never do it. G1 attempts 11 to 15 died on its walk three times at clvl 12. |
| campaign.imbueNeedsMerc | 1 | 0..1 | bool | 1 | 2 | 1: the Imbue quest is also skipped while no mercenary is alive (an unknown merc counts as none). 0: the level floor alone decides. |
| campaign.denMinLevelCaster | 5 | 1..12 | levels | 5 | 1 | Sorceress, Necromancer, Druid and a bow-less Amazon enter the Den of Evil (always the first quest) only from this clvl; below it they level in Blood Moor (log: den: waiting for clvl N in Blood Moor). SME: a ~40 life caster dies to Fallen packs on floor 1; 5 is the safe end of clvl 4-5. |
| campaign.denMinLevelMelee | 3 | 1..12 | levels | 3 | 1 | Same for Barbarian, Paladin, Assassin and a bowed Amazon. |

### route.*

| key | default | range | unit | worst (rookie) | readers | what it changes |
|---|---|---|---|---|---|---|
| route.monasteryMinLevel | 20 | 10..40 | levels | 20 | 2 | The Monastery road (Tamoe Highland, Monastery Gate, Cloisters, Barracks, Jail, Cathedral, Catacombs, so Andariel too) is entered only with a living merc or from this clvl, and with the supply kit (4 healing, 2 portals, mana for casters). Otherwise the bot levels in Cold Plains to Black Marsh and the caves. |
| route.towerMinLevel | 20 | 0..40 | levels | 20 | 7 | The Forgotten Tower and Tower Cellar 1 to 5 (areas 20-25, the Countess, a no-reward quest in Normal) are entered only with a living merc or from this clvl, and with the supply kit (4 healing, a portal, mana for casters). Below it the quest is skipped as optional (log: route guard: blocked tower (no merc, clvl N < 20)), corpses there are abandoned and a character standing there walks out first. G1 attempt 17: a clvl 14 caster died twice in the cellars. 0 turns the tower guard off. |
| route.towerNeedsMerc | 1 | 0..1 | bool | 1 | 3 | 1: a living mercenary opens the Forgotten Tower below the level floor (an unknown merc counts as none). 0: only the level floor opens it. |

### resource.*

| key | default | range | unit | worst (rookie) | readers | what it changes |
|---|---|---|---|---|---|---|
| resource.restStartManaPct | 60 | 20..90 | % | 20 | 2 | Casters rest in place until mana reaches this percent (or the pack's cast budget) before engaging. |
| resource.castsPerMonster | 3 | 1..6 | count | 1 | 3 | Casts budgeted per monster of the next pack when deciding how much mana is enough. |
| resource.restMaxSec | 45 | 10..120 | s | 10 | 5 | Longest rest before a pack; beyond it the policy drinks, falls back to melee or goes on. |
| resource.engageLifePct | 65 | 40..90 | % | 40 | 3 | Life percent below which the bot fixes its life before a pack. |
| resource.minEngageLifePct | 40 | 25..70 | % | 25 | 6 | Life percent below which a pack is not engaged at all. |
| resource.retreatDanger | 2 | 1..3 | level | 3 | 1 | Pack danger at which resting happens out of sight. |
| resource.hpReserve | 1 | 0..4 | count | 0 | 1 | Healing potions kept for emergencies, not spent on engaging. |
| resource.economyManaPct | 40 | 10..70 | % | 10 | 1 | Below this mana percent the cheapest attack skill is used. |
| resource.meleeMinLifePct | 50 | 30..80 | % | 30 | 2 | Life percent needed for the melee fallback when out of mana. |
| resource.meleeMaxPack | 3 | 1..6 | count | 6 | 2 | Largest hostile count the melee fallback accepts. |
| resource.lowManaWaitMs | 700 | 300..2000 | ms | 2000 | 2 | Wait when out of mana and nothing else is possible. |
| resource.waitMaxSec | 8 | 2..30 | s | 30 | local var | Longest stretch of waiting for mana inside a fight before retreating. |
| resource.backoffMaxSteps | 8 | 0..30 | steps | 30 | 2 | Consecutive out-of-mana back-off steps before a weak caster swings its weapon instead of running on (any pack size, danger below 3, life above the melee line). 0 = never. |
| resource.recoveryManaFloor | 2 | 0..4 | count | 0 | 1 | While recovering, a caster may also buy mana potions up to this many (belt plus pack), after the healing floor. 0 = none. |
| resource.poorGold | 150 | 50..500 | gold | 50 | 1 | Below this gold life potions come before mana potions. |
| resource.reserveGold | 30 | 0..200 | gold | 0 | 2 | Never spend below this on potions. |
| resource.potionGoldPct | 80 | 30..100 | % | 30 | 1 | Share of spendable gold potions may take when not poor. |
| resource.hpFirst | 3 | 1..6 | count | 1 | 7 | Healing potions bought before any mana potion. |
| resource.mpFirst | 2 | 0..6 | count | 0 | 2 | Mana potions bought for casters after the first healing potions. |
| resource.tripHPMin | 1 | 0..4 | count | 0 | 2 | Town trip when healing potions are at or below this. |
| resource.tripMPMin | 0 | 0..4 | count | 0 | 1 | Town trip (casters) when mana potions are at or below this. |
| resource.tripMinGold | 60 | 30..300 | gold | 30 | 2 | Below this gold a potion trip is futile. |
| resource.futileStep | 100 | 30..500 | gold | 30 | 1 | After a trip that could not fix a need, gold must grow this much before the next. |
| resource.tripHealPct | 50 | 25..80 | % | 25 | 1 | Life percent below which a character that cannot regenerate goes to town for free healing. |
| resource.mercDeadPack | 2 | 1..6 | count | 6 | 1 | Hostiles at which a fight is left once the merc is dead. |
| resource.fleeLifePct | 50 | 30..80 | % | 30 | 1 | Life percent below which a character with no healing potion and no portal breaks away from hostiles instead of trading blows. |
| resource.kitTPGold | 100 | 50..200 | gold | 50 | 7 | Price of one town portal scroll, for the minimal-kit cost. |
| resource.kitPotionGold | 30 | 20..100 | gold | 20 | 7 | Price of the cheapest healing potion, for the minimal-kit cost. |
| resource.povertyMargin | 60 | 0..300 | gold | 0 | 1 | Gold above the minimal kit cost at which poverty mode ends. |
| resource.retreatBelt | 0 | 0..3 | count | 0 | 2 | In a fight with life under the drink line and at most this many healing potions left, leave (portal, or walk home near town) instead of drinking. 0 = off. |
| resource.retreatCountPack | 1 | 0..1 | 0/1 | 0 | 1 | 1 = spare healing/rejuvenation potions in the backpack (up to the free belt slots) count towards retreatBelt, so a full pack does not trigger a retreat; 0 = belt only. |
| resource.manaPaysCasts | 3 | 0..6 | casts | 6 | 2 | Below the mana drink line a caster skips the potion while its mana still pays this many casts of the cheapest spell. 0 = drink at the line regardless (veteran play: drink at 30 percent before the skill swap). |
| resource.minFieldBeltPct | 0 | 0..100 | % | 0 | 2 | Leave town for the field only with at least this percent of the planned healing potions (unless poor, then Blood Moor only). 0 = off. |
| resource.gearGateMode | 1 | 0..1 | 0/1 | 0 | 2 | 0 = hard gate (no weapon, no body armor or no potions and no gold closes the field), 1 = graded risk model (docs/gear-gate.md). |
| resource.gearRiskMedium | 3 | 1..10 | points | 10 | 0 | Risk points at which gear risk is medium (logged, rebuy priority). |
| resource.gearRiskHigh | 6 | 2..20 | points | 20 | 0 | Risk points at which gear risk is high and only the first field is open. |
| resource.gearDeathsHigh | 4 | 2..10 | deaths | 2 | 0 | Deaths in the hour or in one area from which deaths add gear risk. Dying is normal: a few change nothing. |
| resource.earlyGearMax | 15 | 0..40 | level | 0 | 3 | Below this level armor is ignored (worn only if it drops and is better, never bought, never a trip or gate reason) and a weapon matters only for Amazon, Paladin, Barbarian, Assassin. 0 = off (old rebuy and gate). docs/sme/early-gear-policy.md. |
| resource.tripBudget | 3 | 0..10 | trips | 0 | 3 | Finished town trips allowed per window before a thin belt no longer sends the bot home (it drinks and fights on). 0 = no budget. |
| resource.tripWindowMin | 10 | 1..60 | min | 1 | 3 | Minutes the trip budget looks back. |
| resource.tripGoldFloor | 300 | 0..1000 | gold | 0 | 4 | Below this gold a thin belt does not trigger a retreat and town buys only one portal scroll. 0 = off. |
| resource.recoveryOn | 1 | 0..1 | 0/1 | 0 | 1 | 1 = a broke character pauses its quest step, farms gold and sellable loot in a safe area and returns at the target (docs/recovery.md). 0 = old behaviour. |
| resource.recoveryKitGold | 500 | 200..1000 | gold | 1000 | 5 | Recovery starts when gold is below this and the missing kit (portal, potions) costs more than the gold. SME: about 400-600. |
| resource.recoveryTargetGold | 900 | 400..2000 | gold | 400 | 2 | Recovery ends at this gold once the loot is sold. SME: 800-1000. |
| resource.recoveryMaxMin | 45 | 10..120 | min | 10 | 3 | Longest recovery cycle; it ends at the cap even when the target is not reached. SME: 30-60. |
| resource.recoveryCooldownMin | 15 | 0..120 | min | 0 | 1 | Minutes after a recovery before the next one may start. |
| resource.recoveryPotionFloor | 2 | 0..6 | count | 6 | 1 | While recovering, town buys healing potions only up to this many in the belt (no mana, no rejuvenation) so the last gold stays for the kit. 0 = normal belt plan. |
| resource.recoveryXPPick | 1 | 0..1 | 0/1 | 0 | 1 | 1 = a recovering Normal act 1 character farms the safe area with the best XP per kill for its level (gold per kill is the same everywhere, d2d); 0 = the plan area of a character povertyLevels lower. |
| resource.recoveryMaxAreaAbove | 1 | 0..4 | levels | 4 | 1 | XP pick: the recovery area's monster level may be at most this many above the character level. |
| resource.brAttempts | 3 | 0..6 | count | 6 | 1 | Blood Raven attempts before the bot waits for more levels (or skips her). 0 = unlimited. SME: 2-3. |
| resource.brLevelGap | 2 | 1..6 | levels | 1 | 0 | Levels to gain after the last attempt before the Blood Raven budget renews. SME: clvl+2. |
| resource.povertyLevels | 4 | 1..10 | levels | 1 | 2 | Poverty mode farms the plan area for a character this many levels lower. |
<!-- knobdump:end -->

### Fields without a Knob entry (V)

None. Every `Tuning` field has a knob. The seven merc and cube fields (`bot.mercRetryMin`, `mercHireMinGold`, `mercReviveUnverified`, `cubeJobs`, `cubeMaxJobs`, `cubeRuneCap`, `cubeGemCap`) became knobs flagged `noHandicap`: no profile moves them, and the two UNVERIFIED switches (`mercReviveUnverified`, `cubeJobs`) default to 0. `tools/knobdump` test `TestKnobsDocMatchesTables` fails when a field loses its knob.

## 3. Handicap profiles (V: internal/bot/profile.go)

Select by name in config.json `"profile"`, `d2fleet add --profile`, or the PUT /knobs `profile`. `""` is optimal; names are case-insensitive; an unknown name is an error. Each knob becomes `default + strength * (worst - default)`, clamped, where worst is the Min end for HigherIsBetter knobs and the Max end otherwise.

| name | strength (Level) | description in code |
|---|---|---|
| optimal | 0 (nothing applied) | every knob at its default |
| relaxed | 0.2 | slightly slower reactions, a few extra town trips |
| casual | 0.4 | noticeably human pace, still safe |
| sloppy | 0.7 | slow decisions, long detours, over-careful levelling |
| rookie | 1.0 | the worst setting every knob allows |

`custom` in config.json defines your own `{Name, Desc, Level, Per}`; `Per` overrides the strength for one key or a whole group by prefix with a trailing dot (`"nav.": 0.9`, longest prefix wins). Custom wins over a named profile.
Effect on every knob = the "worst (rookie)" column above scaled by the strength: relaxed 20 %, casual 40 %, sloppy 70 % of the way. Example: `combat.reactionMs` default 0 -> 700 worst, so casual is 280 ms.
WARNING (V docs/combat.md lines 57-59, offline): each combat knob at its worst alone is survivable, all at their worst together die against a 5-monster pack. Rookie is therefore not guaranteed to survive; prefer `sloppy` or a `custom` with `Per` for the survival knobs (`combat.`, `resource.`). Not verified live (U).

## 4. d2bot (cmd/d2bot/main.go) (V)

Exactly one of `--sim`, `--attach`, `--managed` is required. Flags with an env fallback take the env value as their default; an explicit flag wins.

| flag | env | default | meaning |
|---|---|---|---|
| `--data` | `D2BOT_DATA` | `$XDG_DATA_HOME/d2bot` or `~/.local/share/d2bot` | data dir: prefix, tables, per-instance state |
| `--config` | `D2BOT_CONFIG` | `<data>/config.json` (optional) | config JSON |
| `--addr` | `D2BOT_ADDR` | `127.0.0.1:8420` | HTTP listen; non-loopback needs a token |
| `--token` | `D2BOT_TOKEN` | empty | API token |
| `--helper` | `D2BOT_HELPER` | derived from instance (127.0.0.1:47012+N) | d2input helper address |
| `--gamedir` | `D2BOT_GAMEDIR` | `<data>/proton/pfx/.../Diablo II` | game dir for the tables |
| `--instance` | `D2BOT_INSTANCE` | `main` | instance id (logs, paths, API routes; also the marker that selects the game process and helper) |
| `--log-level` | `D2BOT_LOG_LEVEL` | `info` | debug, info, warn, error |
| `--allow-host` | `D2BOT_ALLOW_HOST` | empty | extra Host accepted on a loopback listener (proxy, ssh tunnel) |
| `--sim` | `D2BOT_SIM` | off | simulated runner, no game |
| `--attach` | `D2BOT_ATTACH` | off | dev: attach to a running game |
| `--managed` | `D2BOT_MANAGED` | off | d2bot launches game and helper (docs/launch.md) |
| `--dry` | `D2BOT_DRY` | off | read-only observe, no input, no goals; needs attach or managed |
| `--once` / `--dry-for D` | - | off / 0 | with --dry: one observation / stop after D |
| `--char` | `D2BOT_CHAR` | name read from the game | character for re-entering via the menus |
| `--goals` | `D2BOT_GOALS` | empty | e.g. `farm:Blood Moor;reach:3;idle`; starts playing |
| `--autostart` | `D2BOT_AUTOSTART` | off | play saved goals at startup (config key `autostart`) |
| `--run-for D` | - | 0 (until signal) | graceful stop after D |
| `--stub-behaviors` | - | off | failing combat/town stubs, fake input |
| `--stop-game` | - | off | managed: also stop the game on shutdown |
| `--sim-speed N` / `--sim-seed N` | - | 4 / 1 | with --sim: virtual s per real s / seed |
| `--probe-states` | `D2BOT_PROBE_STATES` | off | debug: log StatList changes (docs/boss.md) |
| `--soak-dir` | `D2BOT_SOAK_DIR` | off | dashboard shows the newest soak run read-only |
| `--supervise` / `--supervise-max N` | `D2BOT_SUPERVISE` | off / 10 | restart as a child with backoff; give up after N restarts per hour |
| `--clean-start` | - | off | managed: stop a leftover game first (set by --supervise on restarts) |
| `--version` | - | - | print version |

Boolean env values are true only for `1`, `true`, `yes`, `on` (`envBool`). Other env vars read elsewhere in the code (V):

| env | where | meaning |
|---|---|---|
| `D2BOT_REAP` | internal/env/reap.go | default on; `0/off/false/no` disables killing orphaned winedevice.exe after a managed stop (docs/soak.md). Not applied for i0 |
| `D2BOT_RECYCLE_CYCLES` / `_FDS` / `_MIN_INTERVAL` | internal/launch/recycle.go | game recycle policy. Defaults 500 leave/re-enter cycles, 4000 fds of Game.exe, 30 min between recycles; ints, 0 = trigger off; interval is a Go duration |
| `D2BOT_HWND` | internal/input/client.go | pin the helper to one window handle |
| `D2INPUT_ADDR`, `D2INPUT_EXE`, `D2GAME_BIN` | input / launch | helper address override, test helper exe, d2game binary (default `~/d2lod-live/d2game/d2game`) |
| `D2_ROOT`, `PROTON` | internal/launch/config.go | i0 game root (same as tools/d2-prefix.sh), Proton path |
| `D2BOT_SETUP_PROBE` | internal/setup/probe.go | internal: the setup wizard's re-exec probe |
| `D2BOT_AGENT` | internal/cheat | actor name recorded in the cheat ledger |

config.json `bot` section scalars beyond profile/tuning (V internal/bot/config.go; zero = default): `acts` [1]; `autostart` false; `hardcore` false (disables corpse retrieval); `stuckAfterSec` 120; `travelTimeoutSec` 300; `sweepTimeoutSec` 240; `townTimeoutSec` 180; `stopTimeoutSec` 60; `recoverTimeoutSec` 60; `menuGraceSec` 8; `resupplyCooldownSec` 240; `errorRetrySec` 300; `globalStuckSec` 600; `clearIdleSec` 60 (docs/stallfatal.md); `maxFailsafeRestarts` 3; `stateStuckSec` {recovering 180, resupplying 240, traveling 240, in_town 300}; `maxStepFailures` 5; `dangerBanSec` 3600; `deathLogPath`; `ignoreTownAdvice`; `levelerBuild` "fallback" or "primary"; `noLeveler`; `disableObserve`; `planDir`. Live effect U unless docs/bot.md says otherwise.

## 5. d2fleet (cmd/d2fleet/main.go) (V)

`d2fleet serve [flags]` and client commands `add|remove|list|status|restart`. Client env: `D2FLEET_URL` (default `http://127.0.0.1:8431/api/v1/fleet-admin`), `D2FLEET_TOKEN`.

| flag | applies to | default | meaning |
|---|---|---|---|
| `--url` | client | env or default URL | admin URL |
| `--class`, `--role`, `--build`, `--difficulty`, `--hc` | add | empty / false | class (when the character is created), party role, build key, normal/nightmare/hell, hardcore |
| `--profile` | add | `""` = optimal | handicap profile name (section 3) |
| `--goals` | add | empty | `campaign:T,levelto:N,farm:AREA` |
| `--force` | remove | false | remove even when leased |
| `--addr` | serve | `127.0.0.1:8431` | listen address (loopback) |
| `--data` | serve | paths.FromEnv | data dir |
| `--token-file` | serve | none | bearer token file (none = loopback only) |
| `--setup` (env `D2FLEET_SETUP=1`) | serve | off | first-run wizard at /setup.html |
| `--setup-display` | serve | `$DISPLAY` | X display for installer windows |
| `--no-bots` | serve | off | start only game and helper per slot |
| `--stop-grace` | serve | `fleet.DefaultStopGrace` | time d2bot may take to save on SIGTERM |
| `--soak-dir` (env `D2BOT_SOAK_DIR`) | serve | off | show newest soak run |
| `--fake` | serve | off | TEST ONLY: fake wizard and runtime |

Slot ids `i1..i9`; helper port 47012+N, display :10N (docs/multi.md, docs/fleet.md). Each bot child gets `D2BOT_INSTANCE=<id>` (internal/fleet/bot.go:324). The spec `profile` and goals are written into the child's config section (internal/fleet/spec.go).

## 6. d2soak and its profiles (cmd/d2soak, V) and d2cheat (V)

`d2soak run|report|status|alerts|procs`; exit 0 pass, 1 fail, 2 usage, 3 no data. Docs: docs/soak.md, docs/soak24.md.
`run` flags: `--target` (http://127.0.0.1:8420), `--instance`, `--token` (env `D2BOT_TOKEN`), `--for` (24h), `--every` (30s), `--out` (soak-YYYYmmdd-HHMMSS), `--metrics-url` (TARGET/metrics, `off`), `--sentinel-log`, `--fail-fast`, `--proc name=PID|self|match:SUBSTR` (repeatable), `--bot-log`, `--wait-target` (2m). `report`: `--in`, `--json`, `--bot-log` plus the limit flags. `status`/`alerts`: `--in` (file or run dir), `status`: `--write`.

Limit defaults (DefaultLimits) and the explicit-flag-wins profiles:

| flag | default | m1 (2 h gate) | m3 (24 h) | m24 (alpha 24 h) | v1 (72 h) |
|---|---|---|---|---|---|
| `--max-deaths-per-hour` | 6 | 1 | 1 | 1 | 1 |
| `--max-no-progress` | 10m | 5m | 10m | 10m | 10m |
| `--max-sentinel-per-hour` | 4 | 2 | 4 | 2 | 4 |
| `--max-ladder-per-hour` | 4 | 1 | 4 | 2 | 4 |
| `--min-xp-per-hour` | 0 (off) | 15000 | 0 | 1000 | 0 |
| `--max-rss-mb-per-hour` | 50 | 60 | 40 | 40 | 30 |
| `--max-fds-per-hour` | 5 | 15 | 8 | 5 | 5 |
| `--max-threads-per-hour` | 10 | 10 | 5 | 5 | 3 |
| `--max-goroutines-per-hour` | 10 | 10 | 5 | 5 | 3 |

Unprofiled limit flags (defaults): `--max-unhealthy` 3m, `--max-zombie-samples` 3, `--warmup` 10m, `--area-step-fds` 8, `--wine-min-span` 2h, `--wine-max-fds-per-hour` 40, `--wine-max-growth-24h` 2000, `--min-slope-span` 30m. Alert flags: `--alert-stall` 10m, `--alert-restarts-per-hour` 8, `--alert-fd-window` 3h (0 = off); the sampler-stale alert is fixed at 5m. The profile numbers are U until a full 24 h run calibrates them (stated in the code and docs/soak24.md).

d2cheat (cmd/d2cheat, docs/cheats.md): `d2cheat [--reason TEXT] [--actor NAME] [--ledger FILE] <cmd>`; commands `heal`, `god on|off [--for 60s]`, `restore [--maxlife N]`, `gold N`, `stashgold N`, `level N`, `xp N`, `stats`, `setstat`, `overlay`, `save FILE k=v`, `quest`, `waypoints`, `unlock-difficulty`, `act`, `show`, `ledger`, `probe`. Every change needs a reason and is logged. Save edits need the game stopped and make a backup. Cheat env knobs (V internal/cheat): `D2CHEAT_ALLOW` (comma list of extra allowed character names), `D2CHEAT_ALLOWFILE` (default `~/.config/d2lod/cheat-allow.txt`, one name per line, `#` comments), `D2CHEAT_LEDGER` (default `<data>/ledger/cheats.jsonl`), `D2CHEAT_BACKUPS` (default `<data>/ledger/backups`), `D2_ROOT` (game root for table lookup), `D2BOT_AGENT` (ledger actor), `D2BOT_INSTANCE` (instance). Built-in allow list: Testy, Botty, Navi, Combi, Towni, Booti, Multi, Palli, Palli2, Amazi, Necki; the allow list is re-read on every check. The allow list is a safety rail: only name throwaway characters.

## 7. Dead and no-effect knobs (V by grep, 2026-10-29)

- REMOVED (2026-10-29): `bot.starveGold` and `bot.starveRecoverGold` had no reader (the starving logic uses `resource.poorGold`, `resource.povertyMargin`, `resource.kit*`). The struct fields and knobs are gone; the keys are listed in `env.RetiredKnobs`, so an old config.json `tuning` or saved tuner state naming them still loads and the entries are ignored. Defaults of all other knobs are unchanged.
- Default already at the worst end, so no profile changes them (V from the tables, worst == default): `combat.missileDodge`, `town.keyTarget`, `loot.potionBuffer`, `loot.carryValuables`, `resource.tripMPMin`. For these a handicap needs a `tuning` override or a different default.
- The seven merc and cube knobs are `noHandicap`: no profile moves them (rookie leaves them at default), an override still can.
- `rookie` sets every other knob to its worst end at once. docs/combat.md says the all-worst combat stack dies in the offline sim (`TestWorstAllowedSettingsDoNotDie` does not promise it); rookie has never been run live, so its survival is U.
- `nav.*` time knobs show 0 direct readers but are live through accessors (see the readers column).
- Unverified-live defaults: `combat.missileDodge` (off), `bot.cubeJobs` (off), `bot.mercReviveUnverified` (off) are intentionally disabled until a live check.
- Ranges are windows from offline simulations; resource.* ranges are tagged U in `internal/resource/tuning.go`.

## 8. Save point (bot.savePointMin)
The game writes the .d2s only on Save and Exit (V: SAVECHECK, docs/soak-analysis.md 6c). `internal/bot/savepoint.go`: in town, no trip, no panel, life >= 50%, no death in the last 2 minutes, no per-game quest memory pending, and `bot.savePointMin` minutes since the last game entry, the bot calls `Cycler.NewGame` (Save and Exit, re-enter the same character and difficulty) under `oogRun`. Every game entry (first start, recycle, farm NewGame, fresh-game leveling, crash re-entry) restarts the timer and resets the xpRisk/XPLOSS baseline. Planned entries are not counted as `restarts` in the minute line, do not feed `noteReentry` or the failsafe ladder; a failure backs off 5 minutes. 0 = off. Offline tests only (U live).
