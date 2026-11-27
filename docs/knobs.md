# Knobs and settings reference

Everything a user can turn: tuning knobs (`internal/env/tuning_*.go`), handicap profiles, `d2bot` / `d2fleet` / `d2soak` / `d2cheat` flags and environment variables.
Tags: V = read from the code in this worktree (cross-checked by grep and by `tools/knobdump`, which prints the knob tables below), U = not verified live.
Written 2026-10-29 from master `6b920b0`. Regenerate the tables between the `knobdump` markers in section 2 with `go run ./tools/knobdump` (reads code only; `go test ./tools/knobdump` fails when they drift).

## 1. How knobs are set (V)

- Every knob has a dotted key `group.name` (groups: `nav combat town loot bot boss merc resource ladder senses watch`) and a range. All writes are clamped to `[min, max]` and rounded for int fields (`bot.SetKnob`).
- Order of precedence at start: defaults (optimal) -> handicap profile -> `"tuning"` overrides in config.json. At run time: `PUT /api/v1/instances/{id}/knobs` with `{profile?, overrides?: {key: number|null}}` (null resets to the profile value; the edit is all-or-nothing). `GET` returns every knob with its current value. Doc: docs/api.md.
- config.json (`D2BOT_CONFIG`, default `<data>/config.json`, the `"bot"` object or the whole file; unknown keys are errors): `profile`, `custom` {Name, Desc, Level, Per}, `tuning` {"bot.clearRadius": 20}. Doc: docs/bot.md "Tuning and handicap profiles".
- The "worst" column is the end of the range a profile moves toward (the end the bot is least efficient at). Knobs where the default already sits on that end do not change under any profile (list in section 7).
- Verified live: the key -> field -> clamp plumbing is tested offline (`go test ./internal/env ./internal/bot -run EveryKnob`). The EFFECT of an individual knob on a live game is mostly NOT verified: only the behaviours the domain docs mark VERIFIED LIVE are (combat death/rest/potions: docs/combat.md; town/nav walking: docs/town.md, docs/nav.md; leveler points: docs/leveler.md; resting before packs: docs/resource.md). No knob has a recorded live A/B test of its range. Ranges are offline-simulation windows (Sorceress, Normal Act 1), tagged U for anything else.
- The doc each group is explained in: merc -> docs/merc.md; nav -> docs/nav.md; combat -> docs/combat.md; town and loot -> docs/town.md, docs/items.md; bot -> docs/bot.md, docs/leveler.md, docs/merc-farm.md; resource -> docs/resource.md.
- Counts: nav 20 knobs, combat 38, boss 1, merc 13 (was 10; +3 in mercrevive2), resource 28 (fixed in docs/nav.md, docs/combat.md, docs/resource.md, docs/bot.md on 2026-10-29).

## 2. Tuning knobs (V: defaults, ranges, readers from the code)

Column "readers" is the heuristic count of non-test uses outside `internal/env`: `accessor` = read through a `NavTuning` method (`Poll()`, `Replan()`, `Stuck()`, `WalkTimeout()`, `ExploreLimit()`, `LoadWait()`, `PanelWait()`, each with 1 to 10 callers); `local var` = read as a local copy (`rt.WaitMaxSec`, internal/combat/resource.go:293). Units: subtiles (5 per tile), ms, s, %, gold. Bool knobs are 0/1.
Notes on defaults that are switches: `combat.missileDodge` default 0 (U, off until verified live), `combat.kiteEnabled` 1, `town.keyTarget` 0 (never buy keys), `town.sellJunk` 1.


<!-- knobdump:begin -->

### nav.*

| key | default | range | unit | worst (rookie) | readers | what it changes |
|---|---|---|---|---|---|---|
| nav.pathQuality | 1 | 0.1..1 | 0..1 | 0.1 | 1 | 1 = shortest, smoothed paths. Lower values make paths longer and wobblier (inflated A* heuristic, no smoothing below 0.5, sideways waypoint jitter) but still reach the goal. |
| nav.sloppiness | 8 | 0..14 | subtiles | 14 | 1 | Largest sideways displacement of a waypoint at path quality 0; scaled by (1 - quality). |
| nav.waypointSpacing | 8 | 4..24 | subtiles | 4 | 5 | Distance between walking clicks. Smaller spacing follows the path more exactly but clicks more often. |
| nav.arriveTolerance | 2 | 1.5..6 | subtiles | 6 | 4 | A walk ends within this distance of the goal. |
| nav.pollMs | 100 | 50..400 | ms | 400 | accessor | Observation period while walking; larger values react later to stops and monsters. |
| nav.replanMs | 1500 | 500..8000 | ms | 8000 | accessor | Earliest time a changed map or monster layout triggers a new plan. |
| nav.stuckMs | 1200 | 600..4000 | ms | 4000 | accessor | No progress for this long starts the recovery ladder. |
| nav.maxRecoveries | 6 | 2..12 | count | 2 | 1 | Stuck-ladder rungs tried before the walk fails with ErrStuck. |
| nav.wallClearance | 3 | 0..6 | cells | 0 | 3 | Cells next to walls that cost extra, keeping paths off corners. |
| nav.wallWeight | 0.75 | 0..3 | cost/cell | 0 | 3 | Extra path cost per cell of missing wall clearance. |
| nav.monsterRadius | 8 | 0..16 | subtiles | 0 | 3 | Paths keep this distance from monsters and NPCs when a detour is cheap. |
| nav.monsterPenalty | 6 | 0..20 | cost | 0 | 2 | Extra cost of a cell at a monster, fading to zero at the avoid radius. |
| nav.exploreInformed | 1 | 0..1 | bool | 0 | 1 | 1 prefers large, forward frontiers; 0 always walks to the nearest frontier. |
| nav.exploreBucket | 10 | 5..30 | subtiles | 30 | 1 | A frontier within this distance of a place already visited counts as inspected. |
| nav.walkTimeoutSec | 120 | 30..600 | s | 30 | accessor | Hard cap for one GoTo. |
| nav.exploreTimeoutSec | 300 | 30..1800 | s | 30 | accessor | Hard cap for one Explore. |
| nav.exitPush | 8 | 3..20 | subtiles | 3 | 1 | How far past a border gap or exit tile the walk continues to trigger the level change. |
| nav.loadWaitSec | 20 | 5..90 | s | 5 | accessor | How long to wait through a loading screen. |
| nav.panelWaitMs | 2500 | 800..8000 | ms | 800 | accessor | How long to wait for a clicked panel (waypoint) to open. |
| nav.useWaypoints | 1 | 0..1 | bool | 1 | 8 | 1: travel, the corpse run and the way back from town take an activated waypoint instead of walking (saves minutes and walking deaths; the route guard still applies to every area on the way). 0 walks everything. |
| nav.wpMinDistance | 2 | 2..8 | levels | 2 | 4 | A waypoint trip is taken only when it skips at least this many level hops; shorter routes walk. 2 is the shortest trip that saves a level. |
| nav.wpCooldownSec | 0 | 0..20 | s | 0 | 2 | Deprecated, default 0 (no wait): game.exe has no arrival cooldown in single player (docs/research/waypoint-trace.md) and d2d dropped its own in #153. A value above 0 makes the bot wait this long (plus 1 s) after arriving before clicking a row, for an old d2d binary. |
| nav.badEdgeTTLSec | 300 | 0..3600 | s | 300 | 2 | A level link or gate that a walk failed on is routed around for this long (game time), and forgotten at once when the game is re-entered; a link the level map itself shows is never marked. 0 restores the old behaviour: marks last for the rest of the run, and a character cut off from a gate it can see is not walked out of its pocket (townwedge). |
| nav.fullGrid | 0 | 0..1 | bool | 0 | 3 | 1: a level whose grid is measured whole (every cell of its rectangle known, as d2d's debug grid is on arrival) is planned on that grid alone and treated as fully known only then; a goal outside it is refused (no path) instead of walked toward as a 'known edge'. A partial grid keeps the optimistic walk and the explore-to-reveal path. 0: the old behaviour (d2d always fully known, optimistic fallback). Default 0 until the Countess scenarios are green (docs/design/fable-review-2026-11-22/map-knowledge.md card 2). |
| nav.warpRead | 0 | 0..1 | bool | 0 | 2 | 1: the d2d backend reads each warp's trigger area from the fork's read-only debug warptriggers (the cells whose click takes the warp, and the cells from which that click is taken at once) and the mouth gate refuses exactly the clicks that land on it (plus a margin of 2 subtiles) instead of every click within the guessed 15 subtile bound; an exit without a read area keeps the guess. Needs a fork with the verb (stock d2d answers err and the guess stays). 0 (default): the guessed bounds, until the Countess scenario is green with it (docs/design/fable-review-2026-11-22/map-knowledge.md card 3). |
| nav.cautionRadius | 20 | 8..40 | subtiles | 8 | 1 | A corpse run stops and reports when a living monster comes this close; its paths avoid monsters at twice the usual radius. |
| nav.roamSpacing | 16 | 5..30 | subtiles | 16 | 1 | Distance between the coverage points a fully known level is roamed by (d2d). Fewer, farther points mean shorter clears; the bot fights whatever comes within its clear radius on the way. Read when a level is first roamed. docs/roamspacing.md. |
| nav.roamCover | 10 | 3..20 | subtiles | 10 | 1 | A coverage point counts as visited when the hero comes within this radius (Chebyshev); clamped to nav.roamSpacing so a visit can never skip a whole grid cell. docs/roamspacing.md. |
| nav.teleReservePct | 40 | 0..90 | % | 40 | 2 | A Teleport hop is skipped (the walk goes on, no waiting for mana) when the mana after it would be below this share of max mana, so fights are not starved. Default: about eight casts of a 5 to 8 mana attack on a 150 mana pool (UNSURE, docs/teletune.md). |
| nav.teleMinLeg | 13 | 0..120 | subtiles | 13 | 6 | A hop is skipped unless it skips more than this much of the planned walking path: running speed times the measured time of a hop (0.5625 x 18 ticks = 10, DRILLS24) plus a margin of 3 (docs/teletune2.md). 0 hops whenever mana allows. |
| nav.teleEscapeReservePct | 15 | 0..90 | % | 15 | 1 | After a walk stalled at the character's spot, a hop ignores the minimum leg and may spend mana down to this share of max mana (never above nav.teleReservePct): leaving a pocket beats keeping mana for fights. Default: about three casts of a 5 to 8 mana attack on a 150 pool (UNSURE). |
| nav.stuckFightLifePct | 60 | 0..100 | % | 60 | 1 | A walk stalled for the second rung of the stuck ladder or later, with hostiles inside the fight radius and life below this share, hands control to the fight (or the town portal once the fights are used up) instead of side-stepping on (Gate 2 g2c seed 13 died walking the ladder through a boss pack). 0 switches it off. |
| nav.pursuitRadius | 10 | 0..20 | subtiles | 10 | 2 | An explore, roam or exit walk with a living monster this close for nav.pursuitPolls observations in a row stops and hands over to the fight (the Yield hook) instead of walking on with it behind (d2play 2026-11-10: the Paladin walked away from two followers at full life). Class generic; the fight decides attack or retreat. 0 switches it off. |
| nav.pursuitPolls | 4 | 1..20 | count | 4 | 1 | How many observations in a row (nav.pollMs apart) a monster must stay within nav.pursuitRadius before the walk turns to fight; a monster the hero outruns or passes does not count. |
| nav.pursuitMinLifePct | 50 | 0..100 | % | 50 | 1 | The pursuit rule only turns the walk to fight at or above this life share; below it the safety guard, potions and the stuck ladder decide. |

### combat.*

| key | default | range | unit | worst (rookie) | readers | what it changes |
|---|---|---|---|---|---|---|
| combat.drinkLifePct | 55 | 35..80 | % | 35 | 6 | Life percent that triggers a healing potion. |
| combat.drinkRejuvPct | 35 | 20..60 | % | 20 | 5 | Life percent that triggers a rejuvenation potion. |
| combat.drinkRejuvBossPct | 50 | 0..70 | % | 0 | 0 | Life percent that triggers a rejuvenation potion in a fight against champions, uniques or bosses (danger 1+); 0 = off. SME advice, EXP. |
| combat.championEarlyPct | 10 | 0..25 | pp | 0 | 0 | Percentage points added to the healing potion line and the rejuvenation (boss) line while champions (not bosses) are in range, in Normal below character level 15. Champions have about 3x life and +50% damage. 0 = off. SME rule of thumb, EXP. |
| combat.chickenPct | 25 | 15..45 | % | 15 | 3 | Life percent at or below which the bot leaves the game (Normal, Nightmare). |
| combat.chickenHellPct | 30 | 20..50 | % | 20 | 2 | Same as chicken, in Hell. |
| combat.drinkManaPct | 25 | 5..60 | % | 5 | 2 | Mana percent that triggers a mana potion. |
| combat.townPotionMin | 2 | 0..4 | count | 0 | 2 | Healing potions left in the belt at which the bot wants a town trip. |
| combat.crowdN | 8 | 0..20 | count | 20 | 9 | Hostiles within 8 subtiles that make a crowd: below it the normal drink and chicken rules apply alone. 0 turns the crowd rule off. |
| combat.crowdPortalPct | 70 | 40..90 | % | 40 | 3 | In a crowd, open a town portal at or below this life percent when a scroll exists and fewer than 3 monsters are adjacent (a 2 s cast fails when surrounded). |
| combat.crowdDrinkPct | 65 | 40..85 | % | 40 | 2 | In a crowd, drink a healing potion below this life percent (the portal cannot be cast when surrounded); the normal rule drinks at 55. |
| combat.crowdAdjacent | 8 | 3..12 | subtiles | 12 | 7 | A hostile within this distance (centre to centre) is in melee reach and counts as adjacent: three or more make a town portal cast a non-escape (potion, else leave the game) and a crowd 'surrounded'. Default 8 from the live zombie reads (drills11), UNVERIFIED for other monsters. |
| combat.crowdAbandonN | 15 | 0..30 | count | 30 | 1 | Hostiles within 8 subtiles on a travel or quest walk that make the bot abandon the walk at once, retreat 20+ subtiles and open a portal. 0 turns it off. |
| combat.crowdFarN | 5 | 0..20 | count | 20 | 2 | Hostiles within crowdFarRadius subtiles (archers and casters stand outside the 8 of crowdN) that make a far crowd: with life at or below crowdFarPct the hero opens a town portal, else drinks. 0 turns the far crowd rule off. |
| combat.crowdFarPct | 55 | 30..90 | % | 30 | 1 | In a far crowd, leave by portal (else drink a healing potion) at or below this life percent. |
| combat.crowdFarRadius | 14 | 8..30 | subtiles | 30 | 2 | Radius that counts hostiles for the far crowd rule (crowdN counts within 8). |
| combat.crowdAbandonQuestN | 40 | 0..60 | count | 60 | 0 | Like crowdAbandonN, but while a quest chunk the player started runs and the hero's readiness for it is not too hard: the hero fights the crowd instead of leaving. 0 = use crowdAbandonN. The drink, portal and chicken lines still apply. |
| combat.potionGapMs | 1000 | 900..3000 | ms | 3000 | 5 | Minimum time between healing or mana potions. |
| combat.potionEmergencyPct | 30 | 10..50 | % | 10 | 2 | Below this life percent a healing potion is drunk even while an earlier one is still working. Above it the bot drinks again only for a higher tier: the game adds a same-tier drink to the running effect (longer, not faster; fid-potions.md Q5). |
| combat.rejuvGapMs | 300 | 250..1500 | ms | 1500 | 2 | Minimum time between rejuvenation potions. |
| combat.reactionMs | 120 | 40..700 | ms | 700 | 1 | Delay before the bot acts on a survival event. |
| combat.pollMs | 60 | 30..400 | ms | 400 | 14 | How often the fight loop looks at the game. |
| combat.targetQuality | 100 | 20..100 | % | 20 | 3 | 100 picks the best target; lower picks the nearest more often and loses ticks. |
| combat.rangeMargin | 3 | 0..8 | subtiles | 0 | 1 | Distance kept inside a skill's maximum range. |
| combat.clearRadius | 40 | 15..80 | subtiles | 15 | 1 | Radius around the start point cleared when the caller gives none. |
| combat.fightRadius | 22 | 10..40 | subtiles | 10 | 22 | A hostile monster this close counts as a fight. |
| combat.maxTargetSec | 45 | 20..90 | s | 90 | 1 | Give up on one target after this long. |
| combat.noProgressSec | 12 | 5..30 | s | 30 | 1 | Give up on a target whose life did not drop for this long. |
| combat.clearMaxSec | 180 | 30..600 | s | 30 | 2 | Upper bound of one Clear call. |
| combat.castGapMs | 650 | 250..1200 | ms | 1200 | 12 | Minimum time between two casts. |
| combat.castPerSkill | 1 | 0..1 | bool | 0 | 1 | 1 = the gap after a cast is at least that skill's cast length (Lightning and Chain Lightning take 760 ms in Game.exe, longer than the flat gap; d2d casts them as plain SC and keeps 650); 0 = the flat cast gap for every skill. |
| combat.castMarginMs | 90 | 0..400 | ms | 400 | 1 | Added to the cast length (in 40 ms ticks, plus one dispatch tick) for the per-skill gap; 90 ms keeps the early Sorceress bolts at the old 650. |
| combat.castFCR | 0 | 0..200 | % | 0 | 3 | Faster cast rate the per-skill gap assumes; 0 is safe for any gear, a real value shortens the gap of the slow skills. |
| combat.lowManaWaitMs | 700 | 300..3000 | ms | 3000 | 1 | Pause when out of mana. |
| combat.staticCasts | 4 | 0..8 | count | 0 | 1 | Static Field casts that open an elite or boss fight. |
| combat.staticCastsBR | 0 | 0..8 | count | 8 | 1 | Static Field casts that open the Blood Raven fight; 0 keeps the mana for the attack skill (Charged Bolt for the Sorceress). |
| combat.staticFloorNorm | 25 | 10..60 | % | 60 | 1 | Target life percent below which Static Field is not cast. |
| combat.staticFloorNM | 33 | 10..60 | % | 60 | 1 | Same in Nightmare. |
| combat.staticFloorHell | 50 | 10..70 | % | 70 | 1 | Same in Hell. |
| combat.novaManaFloor | 25 | 0..60 | mana | 60 | 1 | Frost Nova on a crowd of 3 or more is not cast with less mana than this, so the mana stays for the bolts. |
| combat.kiteEnabled | 1 | 0..1 | bool | 0 | 2 | 1 backs away from melee monsters that get close. |
| combat.kiteDistance | 10 | 4..20 | subtiles | 4 | 3 | Back away when a melee monster is closer than this. |
| combat.kiteLifePct | 60 | 40..100 | % | 40 | 2 | Step back from a single melee monster only below this life percent (and falling). |
| combat.kiteMaxSteps | 6 | 0..12 | count | 0 | 4 | Retreat steps per target before standing to fight. |
| combat.kiteRetreat | 12 | 6..20 | subtiles | 6 | 10 | Length of one retreat step. |
| combat.dodgeRadius | 4 | 0..8 | subtiles | 0 | 7 | Step away from hostile missiles closer than this. |
| combat.missileDodge | 0 | 0..1 | bool | 0 | 1 | 1 sidesteps fire walls, lightning, hydra and chaos missiles (unverified live, off by default). |
| combat.minionRead | 0 | 0..1 | bool | 0 | 6 | 1 counts the hero's summons from the fork read `debug minions` (needs --read-combat); a failed read falls back to the snapshot census. Off by default (U until a live drill). |
| combat.threatDodge | 0 | 0..1 | bool | 0 | 2 | 1 dodges only hostile missiles the missile threat table says can be dodged in dodgeReactMs and fights the owners of the fast ones first (card botthreat, off until a live drill). |
| combat.dodgeReactMs | 400 | 100..800 | ms | 100 | 2 | Poll plus click time the threat model assumes before the character moves; a missile reaching us sooner is not dodged (I). |
| combat.shooterKillMin | 0 | 0..200 | pts | 0 | 4 | Ranged monsters with an elemental-only attack and a peak hit of at least this are fought first; 0 = off (derived from monstats, U). |
| combat.runnerGuard | 1 | 0..1 | bool | 0 | 1 | 1 skips the kite, step-back and curse retreat while a monster that runs at the player (Zombie, Brute, Blood Hawk, Corrupt Rogue) is close; walking away cannot break contact. |
| combat.standOK | 0 | 0..1 | bool | 0 | 1 | 1 does not drop a Fallen that stands still as never in range (S, off until a live drill). |
| combat.curseRetreatPct | 60 | 30..90 | % | 30 | 2 | With Amplify Damage or Iron Maiden on the player, retreat and drink below this life percent. |
| combat.maxApproaches | 12 | 4..30 | count | 4 | 1 | Approach steps per target before it is dropped as unreachable. |
| combat.reachSec | 15 | 6..40 | s | 40 | 1 | Give up on a target that was not shot at once in this long (unreachable, no line of sight). |
| combat.ignoreMemorySec | 90 | 20..300 | s | 20 | 1 | How long an ignored target is left alone (the memory doubles with every repeat). |
| combat.auraCheckMs | 1500 | 500..5000 | ms | 5000 | 1 | How often the fight loop re-reads the active aura and re-applies it. |
| combat.leftFailSec | 60 | 10..300 | s | 10 | 1 | After a left skill could not be selected, wait this long before trying it again. |
| combat.sacrifice | 1 | 0..1 | bool | 0 | 1 | 1 lets a profile attack with Sacrifice (about x1.9 damage over Attack, costs 8 percent of the damage dealt in life, docs/palprofile.md); 0 keeps plain Attack. |
| combat.sacrificeMinLifePct | 70 | 40..100 | % | 40 | 2 | Sacrifice only at or above this life percent; below it plain Attack until life is 10 points above the floor again. |
| combat.sacrificeMinPotions | 1 | 0..8 | count | 0 | 1 | Sacrifice only with at least this many healing or rejuvenation potions in the belt (and no dead merc). |
| combat.deliveryExpect | 0 | 0..2 | mode | 2 | 1 | 0 = off; 1 = record what each paid cast should change, settle it from later snapshots and log failures (decisions unchanged); 2 = also skip a skill after three failed casts in a row for a growing back-off (combat plan section 3.4). |
| combat.deliveryUpkeep | 0 | 0..2 | mode | 2 | 1 | [dark-launch] 0 = old buff pass; 1 = shadow: the old pass acts and a line is logged wherever the driver would decide otherwise; 2 = the driver acts: buff state ids from the game tables (Skills.txt aurastate), curses never kept up as self buffs, inside a fight non-vital upkeep limited to one cast in the profile's share and above its mana reserve, and a pass after each clear (combat plan 3.2, Q7). |
| combat.holyFireMinMonsters | 3 | 1..10 | count | 10 | 1 | A needsCrowd aura (Holy Fire) is wanted only with at least this many hostiles within 8 subtiles; fewer and Might is used (Might wins below about 3 monsters, docs/palprofile.md). |
| combat.buffRecastSec | 150 | 60..300 | s | 300 | 1 | Upper bound and fallback: recast armors and shields this long after the last cast. An armor whose real duration is known (Frozen Armor 120 s at level 1) is recast earlier, see buffMarginSec. |
| combat.buffMarginSec | 6 | 0..30 | s | 30 | 1 | Recast an armor or shield this many seconds before its real duration (Skills.txt auralencalc at the skill's level) runs out. |

### town.*

| key | default | range | unit | worst (rookie) | readers | what it changes |
|---|---|---|---|---|---|---|
| town.healBelowLifePct | 70 | 10..100 | % | 10 | 1 | Visit the healer in town when life is below this. |
| town.healBelowManaPct | 30 | 0..100 | % | 0 | 1 | Visit the healer when mana is below this; 0 disables. |
| town.tripMinHPPotions | 2 | 0..8 | count | 0 | 1 | Go to town when fewer healing potions remain (belt and pack). |
| town.tripMinMPPotions | 1 | 0..8 | count | 0 | 1 | Go to town when fewer mana potions remain (only if the belt plan has a mana column). |
| town.tripBeltColumnPct | 50 | 0..100 | % | 0 | 2 | Go to town when a planned belt column is filled below this. |
| town.tripPackUsedPct | 80 | 40..100 | % | 100 | 6 | Go to town when the backpack is this full. |
| town.tripRepairPct | 20 | 0..60 | % | 0 | 1 | Go to town when an equipped item is below this durability. |
| town.beltFillPct | 100 | 25..100 | % | 25 | 1 | Refill each potion type to this share of its planned columns. |
| town.plan | 0 | 0..5 | stage | 5 | 16 | 0 = old town code (default). 1 = shadow: every chores visit logs 'town plan (shadow)' with what the supply plan needs, has and would buy; no behaviour change. 2 = guards, chunk preconditions and the shop read the plan (belt plus pack counted; a guard blocks only on a line the plan cannot close). 3 = the purse is settled once per visit to the plan's spend plus the carry line. 4 = the sell chore never sells a pack potion the plan line needs (keeps max(target, min) belt plus pack, best tier first; the rest is surplus under pack pressure); pickup still reads the old buffer. 5 = route (not built). Measured run records carry the stage. |
| town.equipKindFails | 2 | 1..10 | count | 10 | 1 | Failed equips of one item kind (base class) into one slot before that kind is skipped until the character gains a level. The memory is by kind, so a fresh game (new item ids) does not repeat the same refused swap (g2g E1: Cap to head failed in every game). |
| town.tpScrollTarget | 2 | 1..20 | count | 1 | 5 | Scrolls to hold when no tome is owned. |
| town.looseScrollKeep | 2 | 0..10 | count | 0 | 3 | Town portal and identify scrolls kept loose in the pack while a tome of the same kind is owned. The pickup rule takes no more, and the sell chore sells the surplus (sandbox hero: 22 loose scrolls beside a tome of 10 filled the pack to 82%). |
| town.tpReserve | 2 | 1..5 | count | 1 | 3 | Town portal scrolls a poor character buys once it carries its minimal healing kit (a dive then does not start with the only scroll spent). 1 = one scroll, the old cap. |
| town.tpMinGold | 0 | 0..1000 | gold | 1000 | 1 | Below this gold the first town portal scroll is not bought and a missing scroll is no trip reason, so a poor character puts its gold into potions first (XPGAP: 62% of the gold of a fresh d2d character went to scrolls). 0 = always buy at the scroll price. |
| town.tpTomeMinGold | 450 | 0..5000 | gold | 5000 | 4 | Buy a Tome of Town Portal when gold is at least this. |
| town.idScrollTarget | 2 | 0..20 | count | 0 | 4 | Scrolls to hold when no tome is owned. |
| town.identifyTome | 0 | 0..1 | bool | 0 | 8 | 1 lets the town chore buy a Tome of Identify and fill it to 20 (before identify and sell, only from gold above the repair, kit and merc reserves; a logged reason when short). 0 (default) = the bot never buys it by itself; the watch UI Buy scrolls errand always works and ignores this knob. |
| town.identifyTopUpBelow | 10 | 1..20 | count | 1 | 1 | With identifyTome on, refill the Tome of Identify to full when it holds fewer scrolls than this. |
| town.minGoldReserve | 100 | 0..1e+04 | gold | 1e+04 | 5 | Optional purchases never spend below this. |
| town.keyTarget | 0 | 0..12 | count | 0 | 6 | Keys to hold; 0 never buys keys. |
| town.repairBelowPct | 30 | 0..90 | % | 0 | 3 | Repair when any equipped item is below this durability. |
| town.stashGoldBase | 300 | 0..2e+04 | gold | 2e+04 | 3 | Deposit inventory gold above this plus level times stashGoldPerLvl. Death takes ALL carried gold (LIVE G1 attempt 12: 1125 to 0), so keep little. |
| town.stashGoldPerLvl | 50 | 0..2e+04 | gold/lvl | 2e+04 | 3 | Deposit inventory gold above stashGoldBase plus level times this. |
| town.beltBuyLevel | 10 | 0..40 | clvl | 0 | 4 | From this level on a town visit buys the cheapest belt with more potion slots than the worn one (after the potion kit, never below the gold reserve plus a minimal kit). 0 = never. docs/sme/potion-policy.md. |
| town.sellJunk | 1 | 0..1 | bool | 0 | 2 | 1 sells items the pickit rates as sell; 0 keeps them. |
| town.rebuyGearTier | 1 | 0..3 | tier | 0 | 1 | After a death: 0 potions only, 1 weapon and body armor, 2 also helm/boots/gloves/belt, 3 also shield. |
| town.rebuyPotionPct | 40 | 0..100 | % | 0 | 1 | Share of the rebuy budget potions may use before gear. |
| town.rebuyItemMaxPct | 60 | 10..100 | % | 10 | 1 | One rebuy item may cost at most this share of the budget. |
| town.npcStopDist | 4 | 2..10 | subtiles | 10 | 1 | Distance from an NPC at which the walk ends and the click starts. |
| town.npcOpenTries | 3 | 1..6 | count | 1 | 2 | Attempts to open an NPC dialog before the chore is skipped. |
| town.uiTimeoutMs | 2500 | 800..8000 | ms | 800 | 2 | Wait for a panel flag after a click. |
| town.clickSettleMs | 150 | 40..800 | ms | 800 | 9 | Pause after a UI click before memory is re-read. |
| town.dropTries | 6 | 1..8 | count | 1 | 1 | Ground clicks tried, each on another cell beside the character, to drop a held item (then a free pack cell, then the failsafe). |
| town.cursorStuckMax | 3 | 1..8 | count | 1 | 2 | Pack spots tried to put a held item away (a refused spot is skipped); after this many failed clears in a row loot drops the item on the ground without trying the pack. |
| town.moveTries | 3 | 1..6 | count | 1 | 4 | Attempts to move one item between grids (each verified in memory). |
| town.portalSettle | 4500 | 1000..8000 | ms | 8000 | 1 | Wait after arriving in an area before using a portal. |
| town.portalReuseDist | 40 | 0..100 | subtiles | 0 | 1 | ToTown walks into an open portal this close instead of reading a scroll. |
| town.portalClickFails | 2 | 1..6 | count | 6 | 1 | BackToField drops a town portal id after this many failed entries (click without level change) and falls back to a fresh portal, then the waypoint, then the walk. |
| town.greetingWaitS | 90 | 0..180 | s | 180 | 1 | Wait for an NPC menu after a click (first contact plays a long speech). |
| town.greetingSkip | 0 | 0..3 | mode | 3 | 2 | How the first-contact greeting is skipped when the menu is not open 1 s after the click: 0 wait it out (greetingWaitS), 1 click the NPC again, 2 Esc then click again, 3 left click on the NPC tile without hover. docs/npc-skip.md. |
| town.futileGoldStep | 100 | 0..5000 | gold | 5000 | 4 | After a town visit could not fix a trip reason, do not ask again until gold grew by this. |
| town.futileMinutes | 10 | 0..120 | min | 120 | 3 | A trip reason a visit left unfixed is not raised again for this long. |
| town.tidyEnabled | 1 | 0..1 | 0/1 | 1 | 2 | 1: the Tidy chore stashes by the tidy policy table and drops pure junk from a full pack; it never touches equipment or marked items (docs/tidy.md). |
| town.tidyMaxActions | 12 | 1..40 | count | 40 | 2 | Most stash and drop actions of one tidy pass (safety valve). |
| town.tidyMinPackPct | 60 | 20..100 | % | 100 | 1 | A tidy pass drops junk only when the backpack is at least this full. |
| town.mercReserve | 1 | 0..1 | 0/1 | 0 | 2 | 1: while the merc is dead the shop keeps its revive price back from the tome, spare town portal scrolls and mana potions (hero14: 950 gold went to a tome and scrolls, then 600 for the revive was missing). Healing potions and the first portal scroll still come first. 0 = off. |
| town.spareKeep | 4 | 0..30 | count | 0 | 1 | A full pack sells spare potions and identify scrolls beyond this many per kind (they fill the pack but are never belted). |
| town.sellMaxS | 60 | 15..300 | s | 15 | 2 | Time the sell chore alone may take before it is cut off. |
| town.potionsFirst | 1 | 0..1 | bool | 0 | 2 | 1 buys the kit potions before the first town portal scroll, which then needs gold for the scroll plus tpScrollPotions potions (d2-sme order, decision D3). 0 = scroll first (old). |
| town.sellLowGems | 1 | 0..1 | bool | 0 | 1 | 1 sells chipped and flawed gems until the Horadric Cube is in the pack or stash (decision D4, SME: the Cube recipe comes in Act 2); better gems, runes and charms stay. 0 keeps low gems for cube upgrades (old). |
| town.tpFirstLevel | 0 | 0..30 | clvl | 30 | 1 | potionsFirst: extra floor, no first town portal scroll below this level. 0 = none (default, the SME gold rule decides); 4 with tpScrollPotions 0 restores the old BASICSFIX rule. |
| town.tpScrollPotions | 2 | 0..10 | potions | 10 | 2 | potionsFirst: the first town portal scroll is bought only when the gold left after the kit potions covers the scroll plus this many healing potions (d2-sme order, decision D3). 0 = no gold margin. |
| town.idScrollNeed | 1 | 0..1 | bool | 0 | 2 | 1 buys identify scrolls only while an unidentified set, rare, unique or crafted item is in the pack; 0 = always idScrollTarget (old). |
| town.withdrawShop | 1 | 0..1 | bool | 0 | 2 | 1 takes gold out of the stash up to the shopping list (portal scrolls, tome, potions) when the pack cannot pay it; 0 = only when no healing potion is carried. |
| town.bankFirst | 1 | 0..1 | bool | 0 | 2 | 1 deposits gold above the keep line (never below the minimal kit) before the shop; 0 = the shop spends first (old). |
| town.repairLatch | 1 | 0..1 | bool | 0 | 1 | 1 retries a repair that changed nothing only after durability changed or gold grew by futileGoldStep; 0 = every visit (old). |
| town.tripEmptyHPBelt | 1 | 0..1 | on/off | 0 | 1 | Go to town when the belt and pack hold no healing potion (Rejuvenation does not count, as in the BELT gate) and the gold buys one. 0 = off. |
| town.tripMinSpend | 0 | 0..500 | gold | 500 | 2 | A pure restock trip (healing potions below the resupply line, mana, belt columns, portal scroll) with a healing potion still in the belt starts only when the planned purchase is worth this much; a character with gold under the kit line and under 2 healing potions also buys no portal scroll (potions first). docs/goldfloor.md: 83% of NAVSOAK trips bought under 30 gold. 0 = off (old), the default: levers A and B are off after the failed A/B of 2026-10-31. |
| town.mpCapEarly | 0 | 0..8 | count | 0 | 2 | Mana potions one visit buys below mpCapLevel; only used while mpCapLevel > 0, so set mpCapEarly 1 together with mpCapLevel 10 for the goldfloor lever (OFF by default after the failed A/B, docs/goldfloor.md finding 3: mana was 74 to 305 gold/h at clvl 1-10). |
| town.mpCapLevel | 0 | 0..40 | clvl | 40 | 1 | Below this level mpCapEarly caps the mana purchase and a low mana belt raises no town trip. 0 = off (old). |
| town.choreBudgetS | 150 | 30..600 | s | 30 | 2 | Time one Chores call may take before it gives up. |

### loot.*

| key | default | range | unit | worst (rookie) | readers | what it changes |
|---|---|---|---|---|---|---|
| loot.vault | 0 | 0..1 | 0/1 | 1 | 7 | 1: items the Vault preset (docs/vault.md) selects are peeked and recorded in the vault instead of being picked up; the bot walks to them as for a pickup, sends no pickup click and uses no pack space. Gold, potions, scrolls, keys, gems and stacks keep their own rules. Needs d2d started with --vault-verbs; without it the knob does nothing and a Needs-you info line says why. Forced off in measured runs (gates, soaks, baselines: D2BOT_RUN). 0 (default): today's pickit, byte-identical. |
| loot.vaultKeep | 1 | 0..1 | 0/1 | 1 | 1 | 1 (default, with loot.vault 1): the Keep button of the Ethereal Vault queues the item and the bot creates it in the hero's stash at the next safe moment in town, exactly as dropped (docs/vault.md, Keep). 0: the queue stops creating at once (kill switch); queued items stay queued. Needs d2d with --vault-verbs; forced off in measured runs (D2BOT_RUN). |
| loot.vaultEquip | 0 | 0..1 | 0/1 | 1 | 1 | 1 (with loot.vault 1 and loot.vaultKeep 1): the item sheet of the Ethereal Vault offers Equip: the bot takes the item out and wears it in the same town visit, vaults the item it replaced and pins the slot (docs/vault.md, Equip). 0 (default): the sheet offers Take out only. Forced off in measured runs (D2BOT_RUN). |
| loot.vaultKeepPerVisit | 20 | 1..100 | items | 100 | 1 | Most items the Keep queue creates in one town visit, so a long queue does not stall the trip; the rest follows at the next safe moment. |
| loot.vaultSellPerVisit | 10 | 1..50 | items | 50 | 1 | Most vault items the Sell errand creates in the pack and sells at the vendor in one town visit (docs/vault.md, Sell); the rest follows at the next safe moment. |
| loot.vaultSellPrompt | 0 | 0..1 | 0/1 | 1 | 1 | 1 (with loot.vault 1): when the town budget (repair, potion floor, mercenary reserve) is short of gold, a Needs-you row offers to sell the lowest-value unpinned vault items (never rare or better); Yes queues them for the Sell errand. 0 (default): no prompt. Nothing is ever sold without the answer. |
| loot.vaultAutoTrim | 0 | 0..1 | 0/1 | 1 | 1 | 0 (default, user decision 2026-11-21): a full vault never removes, sells or replaces an item; new drops are not recorded (they are counted as missed drops on the Vault page) and only the player takes items out. 1: an unlock for later, the old behaviour: a worthwhile drop trims the least valuable unpinned item, and items marked for sale are sold at the next visit without a confirm (docs/vault.md, Decision 2026-11-21). |
| loot.vaultPoorPickup | 1 | 0..1 | 0/1 | 1 | 1 | 1 (default, with loot.vault 1): below loot.poorGoldBelow gold the looter picks up the items worth loot.poorMinSell gold or more instead of only recording them, so a broke hero earns gold (docs/vault.md). 0: it records them. |
| loot.vaultRevealGold | 0 | 0..1 | 0/1 | 1 | 1 | 1 (with loot.vault 1 and loot.vaultKeep 1): the Reveal bar of the Vault page offers Gold as a payment. A gold reveal is an order the player gives: the price (the vendor price of one Scroll of Identify per item) is subtracted from the STASH gold at once through the d2d fork's `debug paygold` (needs d2d --vault-pay; ledgered with the reason 'vault gold reveal order <id>'; ADR 0029) and the items whose affixes the vault holds are revealed in the same moment: no town visit, no scroll, no tome charge. Never in a measured run (D2BOT_RUN), never above the stash gold, never the purse. 0 (default): off. |
| loot.vaultDirectEquip | 1 | 0..1 | 0/1 | 1 | 1 | 1 (default, with loot.vault 1): Equip and Wear set on the Vault page put the items on the hero at once through the fork's direct equip verbs (no town trip, no waiting; the hero may be fighting; the pieces that come off go to the Vault as records; ledgered with a reason, player-ordered only, never in a measured run, the bot's own play never uses it). Needs a fork build with the verbs (probed at start); an older build, or 0, keeps the order flow of loot.vaultEquip (docs/vault.md, Equip v2). |
| loot.vaultSellInstant | 1 | 0..1 | 0/1 | 1 | 1 | 1 (default, with loot.vault 1): Sell removes the records from the Vault at once and adds what the engine's own sell function pays for them to the STASH gold (the fork's debug creditgold and item price verbs, ledgered, once per order id, never in a measured run). A fork without those verbs keeps the old queue. 0: the old behaviour, the hero sells them at the vendor at the next town visit. |
| loot.vaultSellConfirm | 2000 | 0..2.5e+06 | gold | 2.5e+06 | 1 | A sale worth this much gold or more asks for a second confirming press on the page (Sell N items for G gold? Stash after: S). |
| loot.vaultRevealGoldConfirm | 2000 | 0..1e+05 | gold | 1e+05 | 1 | A gold reveal that costs this much or more (paid from the stash gold) asks for a second confirming press on the page. |
| loot.vaultRevealMethods | 4 | 1..7 | mask | 7 | 1 | Which ways to reveal an unidentified vault item the Vault page offers, a bit mask: 1 Free (Cain, scrolls, tome charge), 2 Ether, 4 Gold. Default 4: only Gold (hidden for now: user decision 2026-11-21). The other routes and the keeper still work; this only decides what the page draws and which payments the Reveal bar quotes. |
| loot.vaultAll | 0 | 0..1 | 0/1 | 1 | 1 | 1 (with loot.vault 1): every ground item that is not gold, a gem or a stack and that no other rule takes is recorded too, plain and low items included. 0 (default): the Vault preset only. |
| loot.pickRadius | 30 | 5..60 | subtiles | 5 | 2 | Ground items within this distance are considered. |
| loot.pickStrictness | 1 | 0..3 | level | 3 | 4 | 0 loose (also items worth selling), 1 normal, 2 keepers only, 3 gold/runes/gems/uniques only. |
| loot.pickTries | 3 | 1..6 | count | 1 | 1 | Click attempts per item. |
| loot.walkDist | 4 | 2..8 | subtiles | 8 | 2 | Walk to within this distance before clicking an item. |
| loot.confirmMs | 1500 | 400..4000 | ms | 400 | 1 | Wait after a click for the item to leave the ground. |
| loot.gapMs | 150 | 0..1000 | ms | 1000 | 1 | Pause between two pickups. |
| loot.goldMinStack | 10 | 0..2000 | gold | 2000 | 4 | Ignore gold piles smaller than this (a ceiling: see goldPileShare). |
| loot.goldPileShare | 50 | 0..100 | percent | 100 | 2 | The min-pile bar never skips more than this share of the area's piles: pile size is ilvl to 6 ilvl - 1 coins (treasure-trace section 4), so Blood Moor piles (1..5) would all fall under 10. 0 keeps the flat bar. |
| loot.freeCellsKeep | 2 | 0..12 | cells | 0 | 2 | Inventory cells that must stay free after a pickup. |
| loot.potionBuffer | 4 | 0..10 | count | 0 | 1 | Extra healing potions carried in the pack beyond the belt (SME EXP, docs/levelplan.md: 4). |
| loot.potionBufferMana | 2 | 0..10 | count | 0 | 1 | Extra mana potions carried in the pack beyond the belt. |
| loot.rejuvSpare | 3 | 0..6 | count | 0 | 5 | Rejuvenation potions carried in total (belt plus pack); more are left on the ground or sold, Full Rejuvenation preferred over regular. Pack ones are moved to a free belt slot in town so the belt hotkey can drink them. |
| loot.lowGoldBelow | 2000 | 0..1e+05 | gold | 1e+05 | 3 | Below this gold, also pick items worth selling. |
| loot.minValuePerCell | 100 | 0..5000 | gold | 5000 | 1 | Sell value per cell an item needs to be picked in low-gold mode. |
| loot.valuableMinGold | 1500 | 100..1e+05 | gold | 1e+05 | 4 | Unequipped items worth at least this are valuables (stash before dangerous trips). |
| loot.valuableMinPrio | 70 | 1..100 | prio | 100 | 8 | Items whose pickit rule prio is at least this are valuables. |
| loot.carryValuables | 0 | 0..20 | count | 0 | 1 | Valuables allowed in the pack outside town (insurance: the rest stay in the stash). |
| loot.poorGoldBelow | 300 | 0..1e+04 | gold | 1e+04 | 1 | Below this gold every gold pile and junk weapon or armor worth selling is picked. |
| loot.poorMinSell | 20 | 0..500 | gold | 500 | 3 | Estimated sell value a junk item needs to be picked in poor mode. |
| loot.maxPerCall | 30 | 1..100 | count | 1 | 1 | Upper bound on pickups in one PickUp call. |
| loot.memEnabled | 1 | 0..1 | 0/1 | 0 | 2 | 1: remember wanted drops beyond the pickup radius and walk back for them at quiet moments. |
| loot.memMinGold | 20 | 0..5000 | gold | 5000 | 1 | Smallest remembered gold pile worth a walk-back at clvl 1; the bar grows by 10 percent per character level. |
| loot.memMaxDist | 120 | 20..400 | subtiles | 20 | 1 | Remembered drops farther away than this are not fetched. |
| loot.memMaxTrips | 6 | 0..30 | count | 0 | 2 | Walk-back trips per visit of an area. |
| loot.goldPriority | 1 | 0..1 | 0/1 | 0 | 8 | 1: gold piles are fetched right below survival: mid-fight when safe, before the next roam leg, and before an area is left (walk-back also for small piles, no trip cap). 0: old behaviour. |
| loot.goldGrabDist | 20 | 5..40 | subtiles | 5 | 0 | A pile this close is picked up during a fight when no hostile is near. |
| loot.goldSafeDist | 8 | 2..30 | subtiles | 2 | 0 | A mid-fight grab needs every hostile at least this far from the hero. |
| loot.goldMaxDanger | 0 | 0..3 | level | 0 | 0 | Highest pack danger (0 none, 1 champions or shamans, 2 elite, 3 deadly) at which a mid-fight grab is still made. |
| loot.goldMinLifePct | 60 | 0..100 | percent | 100 | 0 | Below this life percent gold is ignored (survival comes first). |
| loot.rulesEngine | 0 | 0..3 | stage | 3 | 1 | 0: today's pickit and tidy table decide (default). 1: the loot rules engine (docs/lootrules.md) decides what to pick up from the ground, with today's strictness, minimum priority, per-cell and small-gold filters still applied on top. 2: ground and the tidy chore's stash and drop decisions. 3: plus the sell chore (the five sell decisions of docs/lootrules.md section 13: marks first, then the sell rules, then the ground rules as the pickit judges today). If the engine errors or panics, that one decision falls back to today's code and is logged once. |
| loot.goldRetries | 4 | 1..10 | count | 1 | 4 | Failed pickups of one pile, over all calls, before it is given up. |
| loot.memCap | 64 | 4..512 | count | 4 | 2 | Remembered drops kept at most; the lowest value is evicted. |

### bot.*

| key | default | range | unit | worst (rookie) | readers | what it changes |
|---|---|---|---|---|---|---|
| bot.zoneModel | 0 | 0..1 | mode | 0 | 1 | 0 (default until the A/B of docs/survival/ab-harness.md passes): the plan tables choose the leveling zone. 1: the survival controller chooses among the plan's zones by power-vs-danger margin and expected value (docs/survival/zonemodel.md); any controller error falls back to the plan tables with a log line. Also lets the margin replace the Andariel level floor when boss margin data exists, may leave a dead merc dead, and may ask for a fresh game when the best zone is cleared. |
| bot.zoneDial | 1 | 0..2 | mode | 1 | 1 | Risk setting of the margin model (bot.zoneModel 1): 0 cautious, 1 balanced, 2 bold. Moves the enter and stay lines of the margin; read only by the model. |
| bot.portalGuard | 0 | 0..1 | mode | 0 | 1 | 0 (default until drill R3, docs/fixportalmelee.md): three or more adjacent attackers block a town portal cast (the old count rule). 1: block it when the expected damage of the adjacent attackers over the cast and the walk reaches bot.portalLifePct of current life; with any damage term missing the count rule applies. A hit does not interrupt the cast (V run, decomp28), so the risk is damage taken, not a cut. |
| bot.portalCastMs | 3000 | 500..5000 | ms | 5000 | 1 | Time of the town portal scroll or tome read used by the guard (generous upper bound, U, drill R3 measures it). Longer blocks more casts. |
| bot.portalWalkMs | 3000 | 0..5000 | ms | 5000 | 1 | Walk from the cast spot into the portal used by the guard (generous upper bound, U, drill R3 measures it). Longer blocks more casts. |
| bot.portalHitMs | 700 | 300..3000 | ms | 300 | 1 | Assumed time between two hits of one adjacent monster in the guard (its strongest attack lands every time: an upper bound, U). Shorter blocks more casts. |
| bot.portalLifePct | 50 | 10..100 | % | 10 | 1 | The guard blocks a portal cast when the expected damage over the exposure reaches this share of current life. Lower blocks more casts. |
| bot.tickMs | 100 | 100..1000 | ms | 1000 | 4 | How often the bot re-plans between actions. Slower means more idle gaps. |
| bot.watchMs | 50 | 50..300 | ms | 300 | 5 | How often events, hazards and death are checked while an action runs. Above 300 the bot dies on Hell packs. |
| bot.reactionMs | 0 | 0..300 | ms | 300 | 1 | Extra delay before reacting to low life or a chicken condition. |
| bot.resupplyPotions | 2 | 2..4 | potions | 4 | 4 | Healing potions left in the belt that trigger a town trip. Higher means more trips. |
| bot.invFullPct | 85 | 40..85 | % | 40 | 2 | Share of occupied inventory cells that triggers a town trip. Lower means more trips. |
| bot.levelMargin | 0 | 0..8 | levels | 8 | 12 | Levels above a step's enter level the planner requires. Higher is safer but slower. |
| bot.leaveSlack | 0 | 0..6 | levels | 6 | 9 | Levels past a step's leave level before moving to the next area. Higher wastes time in easy areas. |
| bot.corpseRetries | 3 | 0..3 | count | 0 | 3 | Attempts to recover the corpse in the field after a softcore death, whatever it is worth. After the last failed try the game is reset: the corpse moves to town and is taken there. 0 abandons the gear. |
| bot.portalSettleMs | 600 | 600..4500 | ms | 4500 | 1 | Pause in town after chores before going back out. |
| bot.clearRadius | 30 | 12..45 | subtiles | 12 | 35 | Reach of one clear sweep: how far away a monster is noticed and fought. Smaller means more walking between sweeps; above 30 is the Aggressive stance (UNVERIFIED survival). |
| bot.exploredPct | 100 | 30..100 | % | 100 | 2 | Share of a zone's roam coverage points the hero must have visited before a clear run calls the zone explored and moves on (the planner then takes the next area or a fresh game). 100 visits every point (the old behaviour). Quest and boss runs always explore everything. Needs a roaming (fully known map) backend; others ignore it. docs/exploredpct.md. |
| bot.farmReset | 0 | 0..1 | bool | 0 | 3 | 1: in a leveling zone, once its coverage reaches bot.farmResetPct and a fresh game pays (the zone's current XP per minute is below what a reset yields after the 15 s restart and the walk back, measured in this run), the hero goes home, takes a fresh game (monsters respawn) and returns to the same zone. Never during a fight, a quest step, a boss run, a named run, a recovery or within 180 s of one. docs/farmloop.md. |
| bot.farmResetPct | 85 | 50..100 | % | 85 | 2 | Share of a zone's roam coverage points that must be visited before the farm loop may reset it (bot.farmReset). Keep it at or above bot.exploredPct, which would end the clear first. A zone below this share is never reset. |
| bot.farmResetCooldownMin | 5 | 1..60 | min | 5 | 2 | Least minutes between two farm resets: a reset that did not pay cannot repeat at once. |
| bot.reflexDrink | 1 | 0..1 | bool | 1 | 3 | 1: when the safety check wants a life or rejuvenation potion from the belt, the guard sends the key itself through the input controller and does not cancel the running action (a drink needs no stop). 0: the old way, the guard cancels the action and the loop drinks after it returns. A drink that needs the pack, a portal, a chicken and a crowd retreat always cancel. Measured by safetyLatAvgMs, safetyLatMaxMs and reflexDrinks on the minute line. |
| bot.reflexDeadlineMs | 1000 | 0..5000 | ms | 1000 | 3 | A portal, chicken or crowd cancel gives the running action this long to return; after that the guard carries the safety action out itself, so a stuck action cannot hold a flee back. 0 turns the deadline off. |
| bot.lootShadow | 1 | 0..1 | bool | 1 | 1 | 1: the loot rules engine (docs/lootrules.md) judges every ground pickup, tidy and sell decision next to today's code and logs the differences as `loot shadow: ...` lines (read-only report at /api/lootrules/shadow). Log only: it never changes what the bot does. 0 turns it off. |
| bot.corpseValueMin | 1000 | 1..2e+04 | gold | 2e+04 | 1 | Only the older-body search reads it now: a body worth less than this does not count as the one holding the gear. The corpse run itself no longer depends on value (bot.corpseRetries tries, then a fresh game moves the corpse to town). |
| bot.spiralDeaths | 4 | 3..8 | deaths | 8 | 4 | Deaths in one area within the spiral window that make the bot avoid that area (one fewer when the area is clearly too high for the level or the character died with no potions and no gold). Dying is normal: the default allows three deaths on the same goal. Higher tolerates more. |
| bot.dangerBanMin | 20 | 5..120 | min | 120 | 1 | How long the spiral breaker keeps the character out of an area. Longer stalls the goal more. |
| bot.starterBanMin | 3 | 1..30 | min | 30 | 1 | A ban of the first field of an act is cut to this once the character stands in town because of it (it is the only way out). |
| bot.repeatSiteBanMin | 30 | 0..120 | min | 120 | 2 | A second death in one area within the repeat-site window, or a death within the repeat-site gap of the previous one, bans the area this long and writes a planner dump. 0 turns the rule off. Dying is normal: this changes where the bot farms, not how careful it is. |
| bot.repeatSiteEscalate | 1 | 0..1 | bool | 1 | 1 | 1: the second repeat-site ban of the same area within 12 hours lasts 4 times the ban length, the third and later 12 times (capped at 6 hours). It stops a bot from walking back into the same death spot every half hour. 0 keeps every ban at the base length. |
| bot.repeatSiteWindowMin | 30 | 5..120 | min | 120 | 2 | Two deaths in one area within this many minutes make it a repeat site. |
| bot.repeatSiteGapSec | 180 | 30..600 | s | 600 | 1 | A death within this many seconds of the previous death (any area) makes the area a repeat site. |
| bot.corpseMinValue | 50 | 0..5000 | gold | 5000 | 0 | No longer read: a corpse with anything on it is chased (user 2026-11-25); one worth 0 is skipped. |
| bot.navSkipSec | 150 | 30..900 | s | 900 | 1 | How long a navigation fault (stuck, no exit to the area) keeps the planner out of an area. Short on purpose: it is a bot fault, not danger. |
| bot.areaIdleSec | 60 | 15..600 | s | 600 | 1 | When every candidate area is skipped or banned, the bot waits this long in town at most, then farms the least bad area anyway (a death streak waits at least the starter ban cap). |
| bot.questBlockedFarmMin | 10 | 1..120 | min | 120 | 1 | A quest step the guards refuse is waited out this long; then the hero farms the best leveling area of the level plan and retries the step after this wait, doubling each time (cap x64). Shown as journeyBlocked in the status. |
| bot.parkAfter | 2 | 0..10 | blocks | 10 | 2 | A quest step that fails with 'not found' this many times (planning blocked, the same step) is parked: the hero farms the level plan instead of waiting in town. 0 turns the rule off. |
| bot.parkMinutes | 15 | 1..240 | min | 240 | 1 | How long a parked quest step stays parked before the planner offers it again. The reason shows as journeyBlocked in the status. |
| bot.stallWarnMin | 30 | 0..240 | min | 240 | 1 | Virtual minutes without campaign progress (no kill or xp, no non-town area entered, no purchase, no quest step) before the bot logs 'campaign stalled' and starts a fresh game. 0 turns the stall ladder off. Longest legitimate stretch in the navsoak logs: 19 min. |
| bot.stallExitMin | 30 | 5..240 | min | 240 | 1 | Virtual minutes still without campaign progress after the fresh game before the bot exits (status 5) so the supervisor restarts the whole process. |
| bot.spiralMinutes | 15 | 5..15 | min | 5 | 5 | Window in which the deaths are counted. Shorter forgets sooner. |
| bot.pauseDeathsPerHour | 8 | 8..20 | deaths | 20 | 2 | Deaths in the last hour after which the bot pauses itself and asks the user. |
| bot.pauseCapMin | 60 | 0..720 | min | 720 | 2 | A Pause the player asked for (the frozen game) ends by itself after this many wall minutes and a Needs you row says how long it was. 0 = no cap. |
| bot.stalePackHostiles | 4 | 0..12 | count | 12 | 3 | Hostiles near the hero when it left a field by chicken, town portal or crowd exit that make the area a stale pack: no return by portal and no planner pick of that area until bot.stalePackMinutes pass or the kit is rebuilt. 0 turns the rule off. Seed 13 of Gate 2 g2f died 2 s after re-entering a 12-hostile pack at clvl 4. |
| bot.stalePackMinutes | 10 | 1..60 | min | 60 | 1 | How long a stale pack keeps its area closed. Monsters of a live game stay where they are (U), so the hold is a guess at the time the kit needs to be rebuilt. |
| bot.stalePackMinHP | 5 | 1..8 | count | 8 | 1 | Healing potions (belt and pack) that lift the stale-pack hold early, together with bot.stalePackMinTier. |
| bot.stalePackMinTier | 2 | 1..3 | tier | 3 | 1 | Best healing potion tier in the belt that lifts the stale-pack hold early (1 Minor Healing, 2 Light Healing). |
| bot.potionBurstN | 5 | 0..12 | count | 12 | 2 | Life drinks within bot.potionBurstSec that turn the next drink into a town portal while bot.potionBurstHostiles or more hostiles are near and a portal exists. 0 turns the rule off. Seed 13 drank 11 potions in about 17 s and then died. |
| bot.potionBurstSec | 20 | 5..60 | s | 60 | 2 | Window in which the life drinks are counted. |
| bot.potionBurstHostiles | 3 | 1..8 | count | 8 | 1 | Hostiles within the fight radius the potion burst rule needs before it retreats. |
| bot.povertyTier | 0 | 0..2 | tier | 2 | 3 | Gear tier bought after an abandoned corpse: 0 is the cheapest viable weapon and armor, potions first. |
| bot.goldFloor | 2000 | 0..2000 | gold | 0 | 7 | Gold town visits must not spend, kept to rebuy gear after a death. |
| bot.pointsRetrySec | 30 | 30..300 | s | 300 | 2 | Wait before spending stat and skill points again after an attempt that did not finish. Longer leaves points unspent. |
| bot.gearCheckSec | 60 | 60..600 | s | 600 | 1 | Minimum time between gear passes in town (equip better items, shop). Longer wears worse gear for longer. |
| bot.gearMinGold | 150 | 150..1500 | gold | 1500 | 4 | Gold needed before the bot shops for weapons and armor. Higher shops rarely. |
| bot.gearBudgetPct | 70 | 20..70 | % | 20 | 2 | Share of the gold above the potion reserve one shopping pass may spend on gear. |
| bot.gearReserve | 100 | 0..100 | gold | 0 | 3 | Gold kept for potions when shopping for gear. Lower risks starving after a shopping spree. |
| bot.sellMinItems | 4 | 4..12 | items | 12 | 3 | Sellable pack items that justify a town trip for a character without gold. Higher carries loot longer. |
| bot.difficultyBanMin | 60 | 60..600 | min | 600 | 3 | Minutes the bot stays in its difficulty after a switch to the next one did not happen. |
| bot.savePointMin | 20 | 0..120 | min | 20 | 1 | Minutes in one game before the bot, in town and idle, saves and re-enters it (a crash loses what came after the last save); 0 is off. |
| bot.heapLimitMB | 4096 | 0..8192 | MB | 4096 | 1 | Go heap in use above which d2bot writes a heap profile and exits for a supervisor restart (a runaway allocation would otherwise end in the OOM killer); the GC gets a soft limit 25 percent above it. 0 is off. Normal runs use under 100 MB. |
| bot.failsafeRung3Max | 6 | 0..50 | count | 6 | 1 | Leave-and-re-enter cycles of failsafe ladder rung 3 allowed without real progress (XP, kills, levels) before the ladder skips to the pause rung. 0 is unbounded. |
| bot.rung3WindowMax | 3 | 0..20 | count | 3 | 1 | Failsafe leave-and-re-enter cycles allowed inside the burst window even when kills come between them. One more ends in a field reset (skip the area, no new game). 0 is off. |
| bot.rung3WindowMin | 20 | 1..240 | min | 20 | 2 | Virtual minutes the re-entry burst cap counts over. |
| bot.droughtMin | 8 | 0..60 | min | 8 | 1 | Virtual minutes without a kill or XP gain in a field area at healthy life before the bot leaves the area (second time in a row: a fresh game). Keep below the Gate 2 no-kill limit of 10. 0 is off. |
| bot.droughtStage2Min | 2 | 0..30 | min | 2 | 1 | Virtual minutes after the first drought action (leave the area) before the second (a fresh game). The old rule waited a full bot.droughtMin, which can never help under the Gate 2 limit of 10. 0 uses bot.droughtMin. |
| bot.droughtSlowMin | 10 | 0..60 | min | 10 | 1 | Early game: after this many minutes in one field area with fewer than bot.droughtSlowKills kills in them (a depleted Blood Moor gives a straggler every few minutes, which keeps bot.droughtMin quiet; V, tester realtime drill: 45 kills in 25 min, then one per 1 to 5 min) the bot starts a fresh game, which refills the field. 0 is off. |
| bot.droughtSlowKills | 3 | 1..30 | kills | 3 | 1 | Kills the slow-field window must hold to go on in the same game. |
| bot.droughtSlowMaxLevel | 10 | 0..40 | level | 10 | 2 | The slow-field rule applies below this clvl only (small early fields deplete; later levels clear on purpose). 0 is off. |
| bot.recoveryDryMin | 5 | 0..30 | min | 5 | 1 | While a recovery farms gold, after this many minutes in one field area with fewer than bot.recoveryDryKills kills in them (a depleted Blood Moor: 141 s since the last kill, V realtime Paladin drill) the hero moves on to the next area the recovery rules allow, or starts a fresh game when there is none. The safe-range rules (route guard, kit gate, bans) still decide which areas qualify. 0 is off. |
| bot.recoveryDryKills | 5 | 1..60 | kills | 5 | 1 | Kills the dry-recovery window must hold to stay in the same area. |
| bot.recoveryDrySkipMin | 15 | 1..60 | min | 15 | 1 | Minutes the recovery stays out of an area it left as dry (a new game or a respawn refills it). |
| bot.noPortalWaitSec | 20 | 0..240 | s | 20 | 1 | Low life, no potion and the way to town fails because there is no town portal (and no walk back for this area): seconds after the first such failure before the hero leaves the game, which restarts it in town. 240 keeps the old two minute waits. |
| bot.portalRetryWaitSec | 3 | 0..30 | s | 3 | 3 | A town portal cast cancelled before the click (a drink or a retreat cut it, nothing spent) is retried once as a town trip after this many seconds. 0 is off: the next planning tick decides. |
| bot.checkpointEnable | 0 | 0..1 | bool | 0 | 4 | 1 casts a town portal on purpose at a danger area while the hero is healthy and nothing is near, retreats to it at low life and returns through it after a death. 0 is off (docs/design/danger-protocol-2026-11-25.md). |
| bot.checkpointMinLifePct | 80 | 50..100 | % | 80 | 1 | A checkpoint portal is cast only at or above this life percent. |
| bot.checkpointRetreatPct | 40 | 10..70 | % | 40 | 1 | At or below this life percent a live checkpoint within reach is walked to and taken (potions and the crowd rule act first). |
| bot.checkpointGapSec | 60 | 10..600 | s | 60 | 1 | At least this many seconds between two checkpoint casts. |
| bot.checkpointMinScrolls | 2 | 0..10 | count | 2 | 1 | A checkpoint is cast only with more portal casts than this in the pack, so the last ones stay for escapes. |
| bot.checkpointSafeRadius | 30 | 10..60 | subtiles | 30 | 1 | The checkpoint spot must have no hostile this close. |
| bot.checkpointEntryDist | 40 | 5..120 | subtiles | 40 | 1 | The cast happens at most this far from where the hero entered the area (the entrance of a cleared stretch). |
| bot.checkpointReach | 60 | 10..200 | subtiles | 60 | 1 | A retreat to the checkpoint is tried only when the portal is at most this far away. |
| bot.checkpointArrivalHostiles | 4 | 1..20 | count | 4 | 1 | After a death, arriving through the checkpoint portal into this many hostiles near the exit sends the hero straight back to town. |
| bot.freshAfterRecoverySec | 180 | 0..900 | s | 180 | 1 | No unplanned fresh game (no target left, drought) this long after a lost-game recovery, unless life is below the resume line. A second restart right after the recovery fails the Gate 2 fault criterion. 0 is off. |
| bot.emptyFieldSec | 150 | 0..900 | s | 150 | 1 | A quest-run area with no hostile in sight and no kill for this long counts as cleared, so the route moves on. Other areas follow the sweep setting. 0 is off. |
| bot.clearIdleMin | 3 | 0..30 | min | 3 | 1 | A leveling area where the hero has killed nothing for this long counts as cleared, so the planner moves on to the next area (a human leaves an emptied field). 0 is off. |
| bot.failsafePauseMaxSec | 3600 | 300..1.44e+04 | s | 3600 | 1 | Longest failsafe self pause. The pause starts at 5 minutes and doubles with every consecutive pause without real progress. |
| bot.flailMoves | 6 | 0..100 | moves | 6 | 1 | This many area changes inside the flail window without a kill or XP count as a loop (any cause: the quest board, area done and waypoint fallbacks undoing each other). The bot then holds in place. 0 is off. |
| bot.flailWindowMin | 10 | 1..120 | min | 10 | 1 | Length of the window in which the area changes of the flail guard are counted. |
| bot.flailHoldMin | 5 | 1..120 | min | 5 | 2 | How long the bot stays in the current area (fights in the field, rests in town) after a flail was found. Doubles for each repeat within an hour, at most eight times. |
| bot.invariants | 1 | 0..1 | bool | 1 | 4 | 1 checks about twelve named invariants once per tick (a potion not drunk under the drink line, a field entry without the kit, a refusal counted as a failure, ...) and logs one rate-limited `invariant` line per hit plus per-name counts in the minute line. Logging only, nothing changes for the hero. 0 is off (docs/invariants.md). |
| bot.typedOutcome | 1 | 0..1 | bool | 1 | 9 | 1 makes a refusal of the quest board step (early road, route guard, gate: not ready) a typed Refused outcome with a need and a wake fact: it is handled in one place, never counted as a failure, and logged once per change as `step refused`. 0 is the legacy path (the refusal goes through the failure counter). Other paths are not converted yet (docs/outcomes.md). |
| bot.arbiterShadow | 0 | 0..1 | bool | 0 | 3 | 1 runs the intent arbiter beside the goal chain and logs one line per divergence (chain chose A, arbiter would choose B because ...) plus per-family counters in the minute line. It never acts: nothing changes for the hero. 0 is off (docs/arbiter.md). |
| bot.arbiterControl | 0 | 0..1 | bool | 0 | 2 | [dark-launch] 1 lets the intent arbiter act for the families in bot.arbiterFamilies: the goal chain's hooks of such a family defer to its decision, and the old chain is the fallback when the arbiter chooses nothing or fails. Needs bot.arbiterShadow 1. 0 is off: nothing changes (docs/arbiter.md, data/darklaunch.json). |
| bot.arbiterFamilies | 3 | 0..255 | mask | 3 | 2 | Arbiter control: which families the arbiter controls, one bit each in the order survive 1, player 2, chores 4, claim 8, goldfarm 16, recovery 32, quest 64, level 128. 3 is survive and player, the first family. Every bit is wired (arbfam entries 1 to 4); with bot.arbiterControl 0 none acts (docs/arbiter.md). |
| bot.tickLogSec | 0 | 0..60 | s | 0 | 2 | [dark-launch] Above 0 writes one `tick` line at most every this many seconds: area, kills, xp, the chain's hook family and the arbiter's world facts, so `d2arbiter -follow` can replay a recorded session. Log only, independent of bot.arbiterShadow. 0 is off (docs/design/ticklog-t4.md, data/darklaunch.json). |
| bot.arbiterDwellSec | 120 | 10..1800 | s | 120 | 1 | Arbiter shadow: how long a chosen intent holds its lease; only a higher priority class preempts it inside this time. |
| bot.arbiterTravelSec | 600 | 30..7200 | s | 600 | 1 | Arbiter shadow: an area change that would undo the previous one inside this window is refused unless a kill, XP, gold or belt change happened in between. |
| bot.arbiterLogGapSec | 60 | 1..3600 | s | 60 | 1 | Arbiter shadow: minimum seconds between two divergence lines of the same (chain, arbiter) pair. |
| bot.mercRetryMin | 10 | 1..60 | min | 10 | 1 | Minutes between merc chore attempts; a failed hire or revive is not retried sooner. |
| bot.mercHireMinGold | 400 | 0..2000 | gold | 400 | 1 | Gold in Act 2 town below which no hire is attempted. |
| bot.mercReviveUnverified | 0 | 0..1 | bool | 0 | 3 | 1 lets the town visit click the revive dialog, which is UNVERIFIED live (docs/merc-farm.md). Off until a live probe. |
| bot.mercReviveLoopN | 3 | 0..10 | count | 3 | 3 | Only with bot.zoneModel 1. After this many revives that each ended in the merc's death within bot.mercReviveLoopMin minutes in the same zone, the model stops reviving him while that zone is its plan (it stays closed for the survival noReviveMinutes, 60). A death after a longer life resets the count. 0 turns the breaker off. Every revive and every death logs the merc's time to death either way. |
| bot.mercReviveLoopMin | 10 | 1..120 | min | 10 | 2 | A merc who dies within this many minutes of a revive counts toward bot.mercReviveLoopN (seed 103 of the screen: 4 revives, each lived about a minute). |
| bot.cubeJobs | 0 | 0..1 | bool | 0 | 3 | 1 runs the gold-free gem and rune upgrade chore in town. UNVERIFIED live (docs/items.md), so off. |
| bot.cubeMaxJobs | 6 | 1..12 | count | 6 | 1 | Transmutes per town visit when the cube chore is on. |
| bot.cubeRuneCap | 10 | 1..10 | rune | 10 | 1 | Highest rune number the chore produces (10 = Thul; Dol to Hel and above are ladder-flagged, docs/items.md Q1). |
| bot.cubeGemCap | 2 | 0..4 | grade | 2 | 1 | Highest gem grade the chore produces: 0 chipped, 1 flawed, 2 normal, 3 flawless, 4 perfect. |

### boss.*

| key | default | range | unit | worst (rookie) | readers | what it changes |
|---|---|---|---|---|---|---|
| boss.andarielMinLevel | 0 | 0..22 | levels | 0 | 5 | Override of the per-class Andariel level floor without a living merc (a dead or unknown merc counts as none); 0 = the class row (internal/plandata/boss_levels.go; Paladin 16, recommended 17, and a warning below the recommended level instead of a wall). The kit gate (potions, town portal, at least 2 antidotes) applies on top. 22 is the old hold. |
| boss.andarielMinLevelMerc | 0 | 0..22 | levels | 0 | 1 | The same override with a living merc; 0 = the class row (Paladin 14, recommended 15). |
| boss.andarielHold | 1 | 0..1 | bool | 1 | 3 | 1: the campaign does not enter Andariel unattended (log: boss: held for andariel (unattended hold ...)) until the throwaway fight of card m3-andariel has won once; the level farms instead. Set 0 after that win. Attended runs (bossrun) are not affected. ADR 0011 gate 3. |
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
| merc.reviveGoldMult | 1 | 1..10 | x | 1 | 3 | Revive only with gold of at least this many times the quote plus the revive reserve (merc.reviveReserveLow/High). |
| merc.stopDeathsPerHour | 3 | 1..12 | count | 3 | 3 | More merc deaths than this in an hour stop reviving until gold reaches the stop multiple. |
| merc.stopGoldMult | 5 | 1..20 | x | 5 | 2 | While reviving is stopped by deaths, revive only with gold of at least this many times the price. |
| merc.hireGoldMult | 1 | 1..10 | x | 1 | 2 | Hire or replace only when gold is at least this many times the quote plus merc.kitReservePct of the full kit refill. |
| merc.freeRogueOnly | 0 | 0..1 | bool | 0 | 2 | 1 never pays for the Act 1 Rogue, even once Blood Raven is skipped for good; 0 allows that fallback. |
| merc.replaceAtAct2 | 1 | 0..1 | bool | 1 | 2 | 1 swaps the Rogue for a Desert Mercenary with the class aura at the first Lut Gholein visit per difficulty. |
| merc.swapGainPct | 10 | 0..100 | % | 10 | 4 | A merc item is swapped only when its score is this much better. Lower swaps more often. |
| merc.reqStrDex | 1 | 0..1 | bool | 1 | 2 | 1 checks the merc's own Strength and Dexterity against gear (d2decomped finding, UNVERIFIED live); 0 checks only level. |
| merc.potionLifePct | 50 | 10..80 | % | 50 | 3 | A healing potion goes to the merc only below this life percent in a non-trivial fight. |
| merc.kitReservePct | 100 | 0..500 | % | 100 | 2 | A hire also needs this percent of the full kit refill cost (town portal, resource.kitMinHP healing, resource.kitMinMP mana) in gold on top of the price: potions first, then the merc. |
| merc.kitReservePctFullKit | 50 | 0..500 | % | 50 | 2 | The kit reserve used instead of merc.kitReservePct while the kit is already full (potions and portal just bought, so the next refill is not due soon). |
| merc.reviveReserveLow | 0 | 0..2000 | gold | 0 | 2 | Gold a revive keeps back on top of the real quote while clvl is below merc.reviveReserveClvl. A dead merc is revived as soon as the purse (pack plus stash) covers the quote plus this, before potions and repair (SME rule, user 2026-11-06). merc.kitReservePct applies to hires only. |
| merc.reviveReserveHigh | 50 | 0..2000 | gold | 50 | 2 | The same reserve from clvl merc.reviveReserveClvl on: enough for a few potions, never a full kit refill. |
| merc.reviveReserveClvl | 13 | 1..99 | levels | 13 | 2 | First clvl that uses merc.reviveReserveHigh instead of merc.reviveReserveLow. |
| merc.goldFarmBudgetMin | 40 | 0..600 | min | 40 | 4 | One gold farm arming lasts at most this many virtual minutes, and ends early when gold reaches the revive bar; then merc.goldFarmCooldownMin applies. 0 = no time limit. |
| merc.goldFarmCooldownMin | 60 | 0..600 | min | 60 | 3 | Minutes after a budget end before the gold farm may arm again. |
| merc.goldFarmMaxLevel | 17 | 0..40 | levels | 17 | 3 | While the merc is dead or missing and gold cannot pay for a revive or hire, a Normal Act 1 character below this clvl levels only in Blood Moor, Cold Plains, Stony Field, Den, Caves, Burial Grounds and Crypt. 17 equals boss.andarielMinLevel, the no-merc Andariel floor: at that level farming buys nothing. 0 turns it off. |
| merc.quoteMaxAgeHours | 4 | 0..48 | h | 4 | 1 | The revive price read from the Kashya menu is used for the gold farm trigger and the revive bar for this many game hours while the merc level is unchanged; after that, or at a new level, the next visit reads it again. The level formula is only the fallback before any quote was read. 0 = never stale. |
| merc.enabled | 1 | 0..1 | bool | 1 | 2 | 1 runs the mercenary policy in the bot (revive, Act 2 hire, merc state for the route guards, status line); 0 leaves the bot exactly as it was. On by default since the 2026-11-02 merc decision (docs/merc-wire.md). |

### campaign.*

| key | default | range | unit | worst (rookie) | readers | what it changes |
|---|---|---|---|---|---|---|
| campaign.imbueMinLevel | 20 | 8..40 | levels | 20 | 4 | Below this character level the optional Tools of the Trade quest (kill Smith in the Barracks, Charsi imbues an item) is skipped; Sorceress, Necromancer, Druid and Amazon never do it. G1 attempts 11 to 15 died on its walk three times at clvl 12. |
| campaign.imbueNeedsMerc | 1 | 0..1 | bool | 1 | 2 | 1: the Imbue quest is also skipped while no mercenary is alive (an unknown merc counts as none). 0: the level floor alone decides. |
| campaign.denMinLevelCaster | 5 | 1..12 | levels | 5 | 1 | Sorceress, Necromancer, Druid and a bow-less Amazon enter the Den of Evil (always the first quest) only from this clvl; below it they level in Blood Moor (log: den: waiting for clvl N in Blood Moor). SME: a ~40 life caster dies to Fallen packs on floor 1; 5 is the safe end of clvl 4-5. |
| campaign.denMinLevelMelee | 3 | 1..12 | levels | 3 | 1 | Same for Barbarian, Paladin, Assassin and a bowed Amazon. |
| campaign.journeyAskWaitMin | 10 | 0..240 | minutes | 10 | 0 | RETIRED, no effect (the Journey was removed with the quest board, docs/questboard.md S9; the key stays so saved overrides still load). Guided journey (docs/campaign.md): a 'go to the next chapter at the floor?' question with no answer counts as yes after this many minutes; 0 = never by default (the hero keeps farming until answered). U: judgment. |
| campaign.journeyFarmBudgetMin | 30 | 0..600 | minutes | 30 | 0 | RETIRED, no effect (the Journey was removed with the quest board, docs/questboard.md S9; the key stays so saved overrides still load). After this many minutes in a level step for a chapter that is ready at its floor (but under its recommended level) the hero asks whether to go at the floor; 0 = never asks. U: judgment, one level step budget. |
| campaign.journeyGoAtFloor | 0 | 0..1 | bool | 0 | 0 | RETIRED, no effect (the Journey was removed with the quest board, docs/questboard.md S9; the key stays so saved overrides still load). 0: Auto chapters go at the recommended level (comfortable); 1: at the floor (ready). The default for a hero that never set the policy row on the Journey panel; the hero's own choice wins. |

### route.*

| key | default | range | unit | worst (rookie) | readers | what it changes |
|---|---|---|---|---|---|---|
| route.monasteryMinLevel | 10 | 10..40 | levels | 10 | 4 | The Monastery road (Tamoe Highland, Monastery Gate, Cloisters, Barracks, Jail, Cathedral, Catacombs, so Andariel too) is entered without a living merc only from this clvl, and always with the supply kit (4 healing, 2 portals, mana for casters). Otherwise the bot levels in Cold Plains to Black Marsh and the caves. Interim (ADR 0011, grill 0003; merc decision 2026-11-02): 14 with a full kit, was 16 and 20, so a dead or missing merc does not block the road (the living-merc floors rate a 140-HP Rogue too high). If road deaths fail the death rule at 14 in the act1run soak raise it to 16 or 18 and re-grill. 20 restores the old number. |
| route.monasteryMinLevelMerc | 10 | 0..40 | levels | 10 | 2 | The same floor with a living merc (unknown counts as none). Was 14 (ADR 0011 interim); 10 since the user's recalibration of 2026-11-24 (docs/probes/andariel-gate-recalibration-2026-11-24.md). 0 restores the old rule: a living merc opens the road at any level. |
| route.towerMinLevel | 15 | 0..40 | levels | 15 | 7 | The Forgotten Tower and Tower Cellar 1 to 5 (areas 20-25, the Countess, a no-reward quest in Normal) are entered without a living merc only from this clvl AND with route.towerMinLife max life, and with the supply kit (4 healing, a portal with charges, route.towerKitMana mana potions for a class that uses mana). Below it the quest is skipped as optional (log: route guard: blocked tower (no merc, clvl N < 15) or (no merc, life N < 150)), corpses there are abandoned and a character standing there walks out first. G1 attempt 17: a clvl 14 caster died twice in the cellars. SME 2026-11-04 (user-approved, numbers U): 15 plus life 150, was 16 (ADR 0011 interim) and 20. 0 turns the tower guard off. |
| route.towerMinLevelMerc | 14 | 0..40 | levels | 14 | 4 | The Forgotten Tower floor with a living merc while route.towerNeedsMerc is 0. Interim (ADR 0011): 14. 0 means the no-merc floor applies to a merc too. |
| route.towerMinLife | 150 | 0..1000 | life | 150 | 4 | Without a living merc the Forgotten Tower also needs this max life (log: route guard: blocked tower (no merc, life N < 150)). U: SME number, tuned by the death loop. 0 turns the life rule off. While max life cannot be read the fallback floor route.towerMinLevelNoLife applies and the log says so. |
| route.towerMinLevelNoLife | 16 | 0..40 | levels | 16 | 1 | The no-merc clvl floor while max life is not readable (the pre-life rule, ADR 0011 interim). Log: route guard: blocked tower (no merc, clvl N < 16 (max life unreadable, fallback floor)). |
| route.towerKitMana | 3 | 0..12 | potions | 3 | 2 | Mana potions the Forgotten Tower kit wants for a class that uses mana every attack (Sorceress, Necromancer, Druid; resource.ManaNeed 2). A melee build such as the Barbarian is not asked. U: SME number. |
| route.towerNeedsMerc | 0 | 0..1 | bool | 0 | 3 | 0: a living mercenary only lowers the Forgotten Tower floor to route.towerMinLevelMerc (an unknown merc counts as none). 1: a living merc opens the tower at any level (the old rule). |
| route.barracksMinLevel | 11 | 0..60 | levels | 11 | 7 | The Tools of the Trade chunk (the Smith in the Barracks) is not started below this clvl for a melee class; the board shows 'level N needed' and the hero levels instead. Live and lab 2026-11-23: a clvl 20 Paladin with 279 life lost 150 to 250 life in the first seconds of the arrival (Plague Fester, Ice Fang, Sin Kill, four Champion Black Rogues, 19 to 35 more) and never reached the Smith. d2-sme ballpark (EXP, not looked up): clvl 24 to 26 with 400 life. 0 = no floor. |
| route.barracksMinLevelCaster | 11 | 0..60 | levels | 11 | 0 | The same floor for a caster class (Sorceress, Necromancer, Druid): was 22 (d2-sme ballpark, EXP); 11 since the user's recalibration of 2026-11-24 (Barracks monsters are level 9). 0 = no floor. |
| route.barracksMinLife | 0 | 0..3000 | life | 0 | 5 | Max life a melee class needs besides the level (unspent stat points do not count: the allocator or the player must spend them). While max life is unreadable (0) only the level counts. 0 = no life rule. |
| route.barracksMinLifeCaster | 0 | 0..3000 | life | 0 | 0 | The same max life floor for a caster class. 0 = no life rule. |
| route.barracksLifeWaiver | 4 | 0..40 | levels | 4 | 2 | The max life floor of the Barracks stops applying once the character is this many levels above the level floor (the log says 'life floor waived'), so a build whose life grows slowly (a Paladin with its points mostly in strength, a caster with little vitality) cannot train forever. 0 = never waived. |
| route.freshGapSec | 600 | 0..7200 | s | 600 | 0 | When every leveling area is cleared or blocked the bot starts a fresh game. If the previous fresh game gained under route.freshMinXP the next restart waits until this many seconds have passed (in town, log: leveling: fresh game held back). Gate 2 rerun seed 13 (clvl 15) restarted 75 times in 4 h with no XP. 0 = no gap. |
| route.freshMaxNoGain | 3 | 0..50 | restarts | 3 | 0 | After this many fresh games in a row that each gained under route.freshMinXP, the bot logs 'leveling: blocked: needs input' and only probes one more restart every route.freshBlockedSec, waiting in town in between. A cycle with enough XP resets the count. 0 turns the bound off (the old endless restart loop). |
| route.freshMinXP | 200 | 0..1e+05 | xp | 200 | 0 | Experience gained between two fresh-game restarts that resets the no-gain count (log field xpGained). U: a starting number, tune from the Gate logs. |
| route.freshBlockedSec | 1800 | 0..1.44e+04 | s | 1800 | 0 | Once the no-gain bound is reached the bot waits this long in town between restart probes, so the loop never spins; bans expire and the picker is asked again after each wait. |
| route.earlyRoute | 1 | 0..1 | bool | 1 | 5 | 1 (default): the Underground Passage level 1 and the Dark Wood, the only way to the Tree of Inifuss, stay closed to a hero under the class level in plandata/earlyroutes.json (melee 7, caster 8, U) or short of 2 healing potions and a town portal source; the board says 'level N needed' and the hero levels in the first fields instead. The owner's try anyway opens it. 0: no guard (a clvl 4 Paladin with no potions died there, Questy bed 2026-11-25). |

### resource.*

| key | default | range | unit | worst (rookie) | readers | what it changes |
|---|---|---|---|---|---|---|
| resource.restStartManaPct | 60 | 20..90 | % | 20 | 2 | Casters rest in place until mana reaches this percent (or the pack's cast budget) before engaging. |
| resource.castsPerMonster | 3 | 1..6 | count | 1 | 3 | Casts budgeted per monster of the next pack when deciding how much mana is enough. |
| resource.restMaxSec | 45 | 10..120 | s | 10 | 5 | Longest rest before a pack; beyond it the policy drinks, falls back to melee or goes on. |
| resource.engageLifePct | 65 | 40..90 | % | 40 | 4 | Life percent below which the bot fixes its life before a pack. |
| resource.minEngageLifePct | 40 | 25..70 | % | 25 | 7 | Life percent below which a pack is not engaged at all. |
| resource.retreatDanger | 2 | 1..3 | level | 3 | 1 | Pack danger at which resting happens out of sight. |
| resource.hpReserve | 1 | 0..4 | count | 0 | 1 | Healing potions kept for emergencies, not spent on engaging. |
| resource.economyManaPct | 40 | 10..70 | % | 10 | 1 | Below this mana percent the cheapest attack skill is used. |
| resource.meleeMinLifePct | 50 | 30..80 | % | 30 | 3 | Life percent needed for the melee fallback when out of mana. |
| resource.meleeMaxPack | 3 | 1..6 | count | 6 | 3 | Largest hostile count the melee fallback accepts. |
| resource.lowManaWaitMs | 700 | 300..2000 | ms | 2000 | 2 | Wait when out of mana and nothing else is possible. |
| resource.waitMaxSec | 8 | 2..30 | s | 30 | local var | Longest stretch of waiting for mana inside a fight before retreating. |
| resource.backoffMaxSteps | 8 | 0..30 | steps | 30 | 2 | Consecutive out-of-mana back-off steps before a weak caster swings its weapon instead of running on (any pack size, danger below 3, life above the melee line). 0 = never. |
| resource.recoveryManaFloor | 2 | 0..4 | count | 0 | 1 | While recovering, a caster may also buy mana potions up to this many (belt plus pack), after the healing floor. 0 = none. |
| resource.poorGold | 150 | 50..500 | gold | 50 | 1 | Below this gold life potions come before mana potions. |
| resource.reserveGold | 30 | 0..200 | gold | 0 | 2 | Never spend below this on potions. |
| resource.potionGoldPct | 80 | 30..100 | % | 30 | 1 | Share of spendable gold potions may take when not poor. |
| resource.hpFirst | 3 | 1..6 | count | 1 | 12 | Healing potions bought before any mana potion. |
| resource.mpFirst | 2 | 0..6 | count | 0 | 3 | Mana potions bought for casters after the first healing potions. |
| resource.tripHPMin | 1 | 0..4 | count | 0 | 2 | Town trip when healing potions are at or below this. |
| resource.tripMPMin | 0 | 0..4 | count | 0 | 1 | Town trip (casters) when mana potions are at or below this. |
| resource.tripMinGold | 60 | 30..300 | gold | 30 | 2 | Below this gold a potion trip is futile. |
| resource.futileStep | 100 | 30..500 | gold | 30 | 1 | After a trip that could not fix a need, gold must grow this much before the next. |
| resource.tripHealPct | 50 | 25..80 | % | 25 | 1 | Life percent below which a character that cannot regenerate goes to town for free healing. |
| resource.mercDeadPack | 2 | 1..6 | count | 6 | 1 | Hostiles at which a fight is left once the merc is dead. |
| resource.fleeLifePct | 50 | 30..80 | % | 30 | 1 | Life percent below which a character with no healing potion and no portal breaks away from hostiles instead of trading blows. |
| resource.kitTPGold | 100 | 50..200 | gold | 50 | 13 | Price of one town portal scroll, for the minimal-kit cost. |
| resource.kitPotionGold | 30 | 20..100 | gold | 20 | 16 | Price of the cheapest healing potion, for the minimal-kit cost. |
| resource.povertyMargin | 60 | 0..300 | gold | 0 | 1 | Gold above the minimal kit cost at which poverty mode ends. |
| resource.retreatBelt | 0 | 0..3 | count | 0 | 2 | In a fight with life under the drink line and at most this many healing potions left, leave (portal, or walk home near town) instead of drinking. 0 = off. |
| resource.retreatCountPack | 1 | 0..1 | 0/1 | 0 | 1 | 1 = spare healing/rejuvenation potions in the backpack (up to the free belt slots) count towards retreatBelt, so a full pack does not trigger a retreat; 0 = belt only. |
| resource.manaPaysCasts | 3 | 0..6 | casts | 6 | 2 | Below the mana drink line a caster skips the potion while its mana still pays this many casts of the cheapest spell. 0 = drink at the line regardless (veteran play: drink at 30 percent before the skill swap). |
| resource.minFieldBeltPct | 0 | 0..100 | % | 0 | 2 | Leave town for the field only with at least this percent of the planned healing potions (unless poor, then Blood Moor only). 0 = off. |
| resource.gearGateMode | 1 | 0..1 | 0/1 | 0 | 3 | 0 = hard gate (no weapon, no body armor or no potions and no gold closes the field), 1 = graded risk model (docs/gear-gate.md). |
| resource.gearRiskMedium | 3 | 1..10 | points | 10 | 0 | Risk points at which gear risk is medium (logged, rebuy priority). |
| resource.gearRiskHigh | 6 | 2..20 | points | 20 | 0 | Risk points at which gear risk is high and only the first field is open. |
| resource.gearDeathsHigh | 4 | 2..10 | deaths | 2 | 0 | Deaths in the hour or in one area from which deaths add gear risk. Dying is normal: a few change nothing. |
| resource.gearDecisions | 0 | 0..1 | 0/1 | 1 | 0 | 0 = the player: the bot never equips, swaps or buys equipment and never fills an empty slot (it only advises); 1 = the bot (legacy: drills, soaks, measured runs set it through D2BOT_RUN, and the Gear Sets unlock). Potions, scrolls, keys, ammo, repair, selling and pickup are not equipment and do not change. docs/gear-policy.md. |
| resource.pointsApi | 1 | 0..1 | 0/1 | 0 | 0 | 1 = serve the read-only stat and skill points sheet (GET /api/hero/points) and show the Points tile; spending is always an explicit player action, never automatic. docs/points.md. |
| resource.pointsDecisions | 0 | 0..1 | 0/1 | 1 | 0 | 0 = the player: stat and skill points are banked and the hero asks (Needs you, return screen); 1 = the bot from its plan table (legacy: drills, soaks and measured runs set it through D2BOT_RUN; a hero's autopilot switch does the same for that hero). docs/points.md. |
| resource.skillCapPerLevel | 0 | 0..1 | 0/1 | 1 | 0 | 1 = a skill takes no more hard points than character level minus its required level plus one (the kolbot rule). d2d's can_learn and game.exe's icon check do not enforce it, so it is off; the table checker reports rows that would break it. docs/points.md. |
| resource.earlyGearMax | 15 | 0..40 | level | 0 | 3 | Below this level armor is ignored (worn only if it drops and is better, never bought, never a trip or gate reason) and a weapon matters only for Amazon, Paladin, Barbarian, Assassin. 0 = off (old rebuy and gate). docs/sme/early-gear-policy.md. |
| resource.tripBudget | 3 | 0..10 | trips | 0 | 3 | Finished town trips allowed per window before a thin belt no longer sends the bot home (it drinks and fights on). 0 = no budget. |
| resource.tripWindowMin | 10 | 1..60 | min | 1 | 3 | Minutes the trip budget looks back. |
| resource.tripGoldFloor | 300 | 0..1000 | gold | 0 | 4 | Below this gold a thin belt does not trigger a retreat and town buys only one portal scroll. 0 = off. |
| resource.recoveryOn | 1 | 0..1 | 0/1 | 0 | 2 | 1 = a broke character pauses its quest step, farms gold and sellable loot in a safe area and returns at the target (docs/recovery.md). 0 = old behaviour. |
| resource.recoveryKitGold | 500 | 200..1000 | gold | 1000 | 7 | Recovery starts when gold is below this and the missing kit (portal, potions) costs more than the gold. SME: about 400-600. |
| resource.recoveryTargetGold | 900 | 400..2000 | gold | 400 | 2 | Recovery ends at this gold once the loot is sold. SME: 800-1000. |
| resource.recoveryMaxMin | 45 | 10..120 | min | 10 | 3 | Longest recovery cycle; it ends at the cap even when the target is not reached. SME: 30-60. |
| resource.recoveryCooldownMin | 15 | 0..120 | min | 0 | 1 | Minutes after a recovery before the next one may start. |
| resource.recoveryStallMin | 20 | 0..120 | min | 0 | 2 | A recovery that gained under recoveryStallGold after this many minutes ends as stalled: at clvl 1 to 4 a plain kill drops a few gold, so the bar is out of reach for hours (V, tester realtime hour: 28 gold in 11 min) and the campaign never resumed. 0 = off. |
| resource.recoveryStallGold | 100 | 20..500 | gold | 20 | 1 | Gold a recovery must have gained after recoveryStallMin minutes to go on. |
| resource.recoveryStallHoldMin | 60 | 0..240 | min | 0 | 1 | Minutes after a stalled recovery during which a low kit does not start another one (a death streak still does). |
| resource.recoveryPotionFloor | 2 | 0..6 | count | 6 | 4 | While recovering, town buys healing potions only up to this many in the belt (no mana, no rejuvenation) so the last gold stays for the kit. 0 = normal belt plan. |
| resource.recoveryXPPick | 1 | 0..1 | 0/1 | 0 | 1 | 1 = a recovering Normal act 1 character farms the safe area with the best XP per kill for its level (gold per kill is the same everywhere, d2d); 0 = the plan area of a character povertyLevels lower. |
| resource.recoveryMaxAreaAbove | 1 | 0..4 | levels | 4 | 1 | XP pick: the recovery area's monster level may be at most this many above the character level. |
| resource.recoveryXPEarly | 0 | 0..3 | levels | 0 | 1 | XP pick: a recovering character may enter an area this many levels below its plan enter and kit level (XPGAP: Cold Plains pays 1.5x the XP per kill of Blood Moor at the same kill rate). 0 = the plan levels. |
| resource.recoveryEndOnKit | 0 | 0..1 | 0/1 | 0 | 1 | 1 = a running recovery ends as soon as the gold covers the missing kit (the same test that stops one from starting), instead of going on to recoveryTargetGold or the stall check (TASK-183: at clvl 1 to 5 the target is for gear the character does not need yet). 0 = the old end rules only. |
| resource.recoveryKitGate | 1 | 0..1 | 0/1 | 0 | 1 | 1 = a recovering character with no healing potion in the belt or no town portal scroll farms only Blood Moor and Cold Plains and its walks avoid dungeon areas (Den, caves, Burial Grounds, Crypt, Mausoleum; KITGATE). 0 = the old area choice. |
| resource.kitGateLevel | 8 | 0..20 | level | 20 | 3 | KITGATELEVEL: below this character level an incomplete kit holds a recovering character in Blood Moor and Cold Plains; from it the gate holds only a character with no portal and no healing potion, any other incomplete kit makes a town trip and the area progression goes on. 0 = the gate holds at every level. |
| resource.kitMinHP | 4 | 1..6 | count | 1 | 3 | Minimum kit from kitGateLevel: healing potions (belt, pack, rejuvenation). Advisor range 4-6. |
| resource.kitMinMP | 3 | 0..4 | count | 0 | 0 | Minimum kit from kitGateLevel: mana potions in the belt. Advisor range 3-4. Casters only (a build that needs no mana is not asked). |
| resource.kitTripCap | 2 | 1..5 | trips | 5 | 2 | Kit town trips with the kit still incomplete before the bot proceeds anyway and logs kitgate-skip. |
| resource.povertySellMin | 0 | 0..16 | items | 0 | 1 | A poor character walks to town to sell junk only with this many sellable items (XPGAP: 4-item trips netted +23 gold for 3.5 minutes). 0 = built-in 4 on foot, 6 with a portal scroll. |
| resource.brAttempts | 3 | 0..6 | count | 6 | 1 | Blood Raven attempts before the bot waits for more levels (or skips her). 0 = unlimited. SME: 2-3. |
| resource.brLevelGap | 2 | 1..6 | levels | 1 | 0 | Levels to gain after the last attempt before the Blood Raven budget renews. SME: clvl+2. |
| resource.beltPlan | 0 | 0..6 | template | 6 | 2 | What the four belt columns hold. 0 = the build's own plan, 1 balanced (healing, healing, mana, rejuvenation), 2 all healing, 3 all mana (risky: no healing potion is bought), 4 three healing to one mana, 5 one to one, 6 rejuvenation only. Tier follows the level and the potion tier option. docs/belt-plan.md. |
| resource.beltCol1 | 0 | 0..4 | kind | 4 | 2 | Override the first belt column: 0 = plan, 1 healing, 2 mana, 3 rejuvenation, 4 empty (nothing bought for it). |
| resource.beltCol2 | 0 | 0..4 | kind | 4 | 2 | Override the second belt column (values as beltCol1). |
| resource.beltCol3 | 0 | 0..4 | kind | 4 | 2 | Override the third belt column (values as beltCol1). |
| resource.beltCol4 | 0 | 0..4 | kind | 4 | 2 | Override the fourth belt column (values as beltCol1). |
| resource.povertyLevels | 4 | 1..10 | levels | 1 | 2 | Poverty mode farms the plan area for a character this many levels lower. |

### ladder.*

| key | default | range | unit | worst (rookie) | readers | what it changes |
|---|---|---|---|---|---|---|
| ladder.enabled | 0 | 0..1 | bool | 0 | 3 | 1: the wired decision points (t1.repair, t1.tp.lost; docs/t0-slice.md) build a context key and ask the decision-point store for the action: a learned rule, else the untrained default. The hero never waits for a question. 0: the legacy code paths, unchanged. Default off until the live drill passes. |
| ladder.repeatCooldownMin | 30 | 0..600 | min | 30 | 2 | The same question (same decision point and context key) is not put again for this long after it was asked or answered; if the player answered it, that answer acts meanwhile (no re-ask). 0: ask every time it comes up (the old behaviour, which re-asked about every minute while the situation lasted). TASK-137. |
| ladder.quietAfter | 3 | 0..20 | questions | 3 | 2 | After this many questions for one key faded unanswered in a row the bot uses the default, stops asking about it and writes one journal line. An answer resets the count. 0: never stop. Needs ladder.repeatCooldownMin above 0. TASK-137. |
| ladder.awayMin | 20 | 1..1440 | min | 20 | 2 | Minutes since the player last looked at the watch screen after which the player counts as away. The hero keeps farming where it is; a zone proposal waits for the return and is never accepted by itself (docs/proposezone.md). TASK-131. |

### senses.*

| key | default | range | unit | worst (rookie) | readers | what it changes |
|---|---|---|---|---|---|---|
| senses.tier | 1 | 0..1 | tier | 1 | 4 | Highest senses tier the watch screen shows beside the video (needs --watch). 1: the fogged map of the area and the hero (docs/senses-map.md). 0: no panel and no /api/senses. The panel only reads what the bot already knows; bot inputs are identical either way. TASK-148. |

### invdesk.*

| key | default | range | unit | worst (rookie) | readers | what it changes |
|---|---|---|---|---|---|---|
| invdesk.verbs | 0 | 0..1 | bool | 0 | 1 | 1: a player's equip command goes through the private fork's item verbs (the server's own item handlers) when the connected d2d advertises them in caps; the click path stays the fallback and the oracle. 0 (default): always the click path. Off until the user has seen it work live (docs/design/item-verbs.md). |
| invdesk.instant | 0 | 0..1 | bool | 0 | 1 | 1: with invdesk.verbs on and a backend that advertises the verbs, an item action the player gives (equip, unequip, drop, identify, and stash, take and gold at the chest) runs at once through the verbs instead of waiting for the bot's next safe moment. 0 (default): every action queues. Off until the user has seen the fast lane work live (docs/inventory-actions.md, Instant actions). |
| invdesk.maxSets | 8 | 1..20 | sets | 8 | 1 | How many named gear sets the player may save per hero on the inventory page (docs/gearsets.md). A set is applied only when the player clicks Apply; the bot never chooses one. |

### ui.*

| key | default | range | unit | worst (rookie) | readers | what it changes |
|---|---|---|---|---|---|---|
| ui.managementFast | 0 | 0..1 | bool | 0 | 0 | 1: the viewer's management actions (equip, unequip, drop, identify, stash and take, gold) run at once through the backend's item verbs, not behind the bot's goal loop, and Recall now (POST /api/recall) takes the hero to town at the next safe moment. Needs a backend that advertises the item verbs; without them, in a named run, or when the lane refuses, the action queues as before and the page names the reason. Driving orders (farm, quest, pause) stay queued. 0 (default): everything queues. Off until the user has seen it work live (docs/process/management-fast-path.md). |

### watch.*

| key | default | range | unit | worst (rookie) | readers | what it changes |
|---|---|---|---|---|---|---|
| watch.paletteFeed | 0 | 0..1 | bool | 0 | env.PaletteFeedOn | 1: the viewer (d2dview, and the watch screen through the tap relay) also serves the lossless RGBA tile-delta feed of the game picture (/stream.pfeed; measured 0.4 to 0.7 MB/s against 1.8 MB/s of JPEG q72, docs/probes/video-feed-2026-11-15.md) and the page draws it on a canvas when the glance reply lists it; any decode trouble falls back to MJPEG. Needs the d2d framefeed ring. d2dview and the friends pack read the environment switch D2_PALETTE_FEED=1 (same meaning; the d2dview process has no tuning file). 0 (default): MJPEG only, nothing changes. Off until the live A/B has run (docs/video-feed.md section 9). |

### quests.*

| key | default | range | unit | worst (rookie) | readers | what it changes |
|---|---|---|---|---|---|---|
| quests.board | 1 | 0..1 | bool | 1 | 1 | 1 (default since the Journey was retired, docs/questboard.md S9): a quest runs only after the player accepts its current chunk (Start); the bot does that chunk, walks to the NPC and stops, and the player taps Hand in. Nothing else starts quests: between chunks the hero acts on the player's mode (Train levels toward the next quest, Idle parks in town, Farm farms loot and gold). Quests the bot cannot do yet show 'not supported yet' and the hero trains instead. 0: the old floor planner plays the quests as before (used by measured runs and gates, no Journey); a hero on 0 has no Start button. |
| quests.boardSM | 0 | 0..1 | bool | 0 | 4 | 1: the quest board stores each accepted chunk as a run with a status and latches completion in the hero's plan file (a park survives a restart, accept implies the earlier parts, Stop implies the parts that wait for it, a chunk seen done stays done), design docs/design/fable-review-2026-11-21/quest-board.md. 0 (default): the board derives its state from the evidence every tick, as before. The plan file keeps its old accepted records either way. |
| quests.stepExec | 0 | 0..1 | bool | 0 | 3 | [dark-launch] 1: quest steps that carry an exec row in data/questspec run on the generic step executor of internal/questspec (approach, act, wait for the declared fact read from the game, bounded retries with backoff, a typed outcome): the Cain scroll (take_scroll), Akara reading it (read_scroll), the Horadric Staff transmute and placing the Staff in the orifice; 0 (default): the old handlers. Card stepexec1, ledger row in data/darklaunch.json. |
| quests.autoDelaySec | 20 | 0..300 | s | 20 | 1 | Seconds an opted-in quest counts down (shown on the page, with Cancel) before it starts by itself; 0 starts at once. Only a SAFE offer ever auto-starts. |
| quests.awayMin | 10 | 1..240 | min | 10 | 1 | Minutes without any watch client asking for the board before the player counts as away: then a chunk the board started by itself is also handed in by the board. A present player taps Hand in. |
| quests.autoHoldMin | 15 | 0..240 | min | 15 | 1 | No quest starts by itself this many minutes after the player paused or reset the bot. |
| quests.autoGapMin | 30 | 0..600 | min | 30 | 1 | At least this long between two auto-started chunks (a bug can then not chain starts). |
| quests.autoDayCap | 6 | 0..100 | count | 6 | 1 | At most this many auto-started chunks in 24 hours; 0 turns auto-start off. |
| quests.legacyWall | 2 | 0..10 | levels | 2 | 1 | A chunk judged only by the old level guide (no boss margin) reads too hard, and is not startable, when the hero is more than this many levels under its floor: the board said risky for a clvl 4 Paladin and the Cain scroll walk through the Underground Passage killed it (Questy bed 2026-11-25, floor 9). 0: warn only, as ADR 0011 had it (the owner's Start anyway still starts it). |
| quests.layoutRead | 0 | 0..1 | bool | 0 | 2 | 1: ask the private d2d fork for the current level's room layout and quest-object positions (read-only verb `debug levelmap`, ADR 0022: a read never makes a run CHEATED) and walk straight to the Tree of Inifuss, the cairn stones or Cain's cage instead of exploring for them; a second read (`debug levelmonsters`) gives a superunique's spawn point (the Countess in Tower Cellar 5) from the level's build; stock d2d or an error falls back to exploring. 0 (default): explore as before. |

### identify.*

| key | default | range | unit | worst (rookie) | readers | what it changes |
|---|---|---|---|---|---|---|
| identify.policy | 1 | 0..3 | code | 1 | 0 | 0 manual (only the player's commands), 1 cain-free (default: Deckard Cain identifies on a town visit that happens anyway, free; nothing is bought; manual while Cain is not rescued), 2 scrolls-above-value (Cain when open, else scrolls or the tome for items worth identify.minValue), 3 never. Forced to manual in measured runs. |
| identify.minValue | 0 | 0..1e+05 | gold | 0 | 0 | Policy 2: items whose estimated sell value is below this are not identified with a scroll. 0 = no bar. |
| identify.maxScrolls | 10 | 0..100 | count | 10 | 0 | Identify scrolls or tome charges one town visit may spend. |
| identify.goldFloor | 600 | 0..1e+05 | gold | 600 | 0 | Gold that must stay in the purse after an identify purchase; a purchase that would break it is skipped. |
| identify.tripCap | 40 | 1..200 | count | 40 | 0 | Most items one Cain visit identifies; the rest follows at the next visit. |
| identify.chunk | 10 | 1..40 | count | 10 | 0 | Items taken out of the stash per batch before Cain; each batch is verified before the next. |
| identify.autoSellRest | 0 | 0..1 | bool | 0 | 0 | 1: the sell chore may sell identified items the loot rules rate sell. 0 (default): identified items are kept for the player to see first. |
| identify.etherPerReveal | 1 | 0..1e+05 | ether | 1 | 1 | Ether one vault reveal costs when the player picks Ether instead of a scroll; 0 = Ether is not used for reveals. Default 1: a deliberate small spend (a quest hand-in pays 2 to 5 Ether, the first vault tier costs 20). Replaces the scroll: no scroll or tome charge is used; free after Cain is rescued and here. |

### input.*

| key | default | range | unit | worst (rookie) | readers | what it changes |
|---|---|---|---|---|---|---|
| input.planner | 0 | 0..1 | bool | 0 | 8 | 1: skill selection (right hand and aura, left hand) runs through the input planner: the fork's select verb when the connected d2d advertises input.select in caps, else the bound key, else the popup click, and every path is verified by reading the hand back; the path used is counted in the minute line as input=verb/key/click. 0 (default): the legacy popup path, unchanged. Also the vendor buy and sell, the waypoint take, the town portal enter and the ground pickup use the fork's input.buy, input.sell, input.waypoint, input.portal and input.pickup verbs when the connected d2d lists them, with the click paths as the fallback and no click after a named refusal (docs/process/input-verbs2.md). Off until the Proving Ground scenario and a live check pass on the devp14 image (docs/process/sandbox-devp14.md; question Q13 of the Fable review is open). |
| input.hotkeyMap | 0 | 0..1 | bool | 0 | 1 | 1: when a game starts the bot derives the hero's skill hotkey map from the planner's slots (main attack, second element, movement, buffs in upkeep order, aura, summons, town portal), writes it into the game's own hotkey table through the skill popup (F1 to F8) and reads the table back from the game state, logging any slot that does not match (docs/design/fable-review-2026-11-19/07-input-layer.md 4.3, I7). Never writes a save file. 0 (default): the hotkeys are left alone. U: not verified live. |

### agency.*

| key | default | range | unit | worst (rookie) | readers | what it changes |
|---|---|---|---|---|---|---|
| agency.autoAlloc | 0 | 0..1 | 0/1 | 0 | 1 | 0 (default): off, points are banked for the player. 1: a hero whose build is locked (chosen on the Points page) puts every new stat and skill point where that build's table says, at a calm moment, by verb. A skill the hero cannot use yet is skipped with a visible reason and the point stays banked (docs/autoalloc.md). |
| agency.planner | 0 | 0..1 | mode | 0 | 0 | 0 (default): off. 1: shadow. Once a minute the hero logs a `planner shadow:` line with the attack and aura the planner would pick from the skills the player actually bought, next to what the combat profile picks, and GET /api/hero/plan and the Points page show the derived rotation, the warnings and the viability line. Shadow only: it never changes what the hero does (docs/agency.md). |

### items.*

| key | default | range | unit | worst (rookie) | readers | what it changes |
|---|---|---|---|---|---|---|
| items.ladderRunewords | 0 | 0..1 | bool | 0 | 3 | 1: the planner may socket the ladder-only runewords (Spirit, Insight and the rest of the Runes.txt server rows) and the Return screen shows 'ladder runewords: on'. Set it only together with the fork flag --ladder-runewords (ADR 0024, d.idle mode); read at start. 0 (default, vanilla): they stay unavailable, as in every gate and fidelity run. |

### respec.*

| key | default | range | unit | worst (rookie) | readers | what it changes |
|---|---|---|---|---|---|---|
| respec.baseGold | 500 | 0..1e+06 | gold | 500 | 0 | Fixed part of the price of a respec (all skill and stat points refunded). price = clamp(minGold, maxGold, baseGold + perLevel x level) x (1 + perRespecPct/100 x paid respecs). PLACEHOLDER number: the model matters, the value is not decided (docs/autoalloc.md). |
| respec.perLevel | 250 | 0..1e+05 | gold | 250 | 0 | Gold added per character level before the escalation. PLACEHOLDER number: the model matters, the value is not decided (docs/autoalloc.md). |
| respec.perRespecPct | 50 | 0..1000 | % | 50 | 0 | Percent added to the price for every respec this hero already paid for. PLACEHOLDER number: the model matters, the value is not decided (docs/autoalloc.md). |
| respec.firstFree | 0 | 0..10 | count | 0 | 0 | How many of a hero's first respecs cost nothing (they do not count for the escalation). 0 = every respec is paid. PLACEHOLDER number: the model matters, the value is not decided (docs/autoalloc.md). |
| respec.minGold | 1000 | 0..1e+06 | gold | 1000 | 0 | The price before the escalation is at least this. PLACEHOLDER number: the model matters, the value is not decided (docs/autoalloc.md). |
| respec.maxGold | 1e+05 | 0..1e+07 | gold | 1e+05 | 0 | The price before the escalation is at most this (0 = no cap). PLACEHOLDER number: the model matters, the value is not decided (docs/autoalloc.md). |
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
| - | `D2BOT_PPROF_HTTP` | off | `1` mounts the on-demand CPU profile routes on the answer listener (docs/probes/bot-cpu-fix-2026-11-16.md) |
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
| `--xp-advisory-per-hour` (advisory, never fails) | 0 (off) | 15000 | 0 | 1000 | 0 |
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

## 9. Game picture on the watch page (card viewerfps, 2026-11-15)

Not tuning knobs; settings of the watch screen's video (docs/probes/viewer-fps-2026-11-15.md).
- `GET /stream?fps=N` (1 to 30): this viewer gets at most N parts a second, newest frame wins; other viewers and the capture are untouched. The page sets `fps=8` in "Low rate" mode (button in the video card, remembered per browser in local storage key `d2videolowrate`; the "Performance" hint offers it after 6 s under 15 fps). Absent: every frame.
- Identical consecutive frames (a still scene) are withheld for up to 1 s, then one is sent anyway (the page's stall watchdog is 3 s).
- `/api/glance` field `stream` (only with `-watch-video`): `viewers`, `fpsIn` (capture), `fpsOut` (all viewers summed), `kBps`, `skippedSame`, `missed` (frames a slow viewer never got). The page shows its own `stream: N fps, K KB/s` in the video card head when under 15 fps or with `?debug`; `window.d2video.stats()` returns it.
- Existing: `--watch-video-fps` (x11: 1 to 30, default 15; tap: cap 1 to 30), `--watch-video-quality` (ffmpeg -q:v, default 5), 3 viewers, 512 KiB frame cap.
