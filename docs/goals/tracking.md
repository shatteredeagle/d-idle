# Tracking: how goal progress is measured and shown

Scope: DOCS ONLY. Tags: V = verified (LIVE per docs/mem.md, docs/town.md, docs/bot.md, or a fetched source), S = secondary, U = unverified.
Sources for game facts: https://classic.battle.net/diablo2exp/quests/act1.shtml..act5.shtml (V), https://www.purediablo.com/?p=3457 (S), D2BS/kolbot
structures as quoted in docs/d2-1.14d-memory.md (V as source code, U as 1.14d behaviour until a live check).

## 1. What the bot can read today (state inventory)

| field | where | status |
|---|---|---|
| player unit, class, name, position, mode, act, area id | `d2.Player` (`UnitAny`, `PlayerData+0`, Path) | V LIVE (mem.md) |
| level, experience | stat 12 / 13 in the merged stat vector | level V (level 2 reached live, bot.md), exp V in practice (xp events), scaling of exp/gold U |
| life, mana, stamina, max | stats 6-11, 24.8 fixed | V LIVE |
| str/dex/vit/energy, unspent stat points (4), unspent skill points (5) | stat list | ids V (D2BS), values U; HUD "+" buttons seen as UI flags 6 and 7 V (bot.md) |
| resistances | stats 36, 37, 39, 41, 43, 45 per combat.md | ids V, live values U |
| gold (14), stash gold (15) | stat list | location V, value scaling U (character had none, mem.md Q4) |
| skill list, levels, left/right skill | `Info` +4/+8/+0xC | V LIVE (Sorc start skills) |
| inventory, belt, stash, cube grid, equipment, cursor | `r.Inventory()` with containers | V LIVE (stash and belt moves verified, town.md); cube container U |
| item quality, identified flag, ilvl, body location, page | `ItemData` | V LIVE; ethereal bit, socket count, durability stat, runeword name U |
| item name/type/code | data layer (`internal/data` from MPQ) | V (class index = position in `data.DB.Items`, town.md) |
| UI flags (inventory 0x01, char 0x02, skills 0x04, NPC dialog 0x08, shop 0x0C, quest 0x0F, waypoint 0x14, stash 0x19) | UI array RVA 0x3A27C0 | V LIVE for 1, 2, 4, 0x0F, 0x08, 0x0C, 0x19; 0x14 U |
| difficulty | byte at RVA 0x3A060C | V LIVE location, Normal = 0 only; 1/2 for NM/H U |
| monsters/objects/missiles/ground items, hovered unit | unit hash tables | V LIVE; monster flags/enchants/unique number U (all zero on town NPCs) |
| quest bits per difficulty | `PlayerData` quest pointers (D2BS layout: pNormalQuest/pNightmareQuest/pHellQuest after the 16-byte name, U) | **U, needs spike** |
| waypoint bits per difficulty | PlayerData waypoint pointers (after the quest ones, U); `CheckWaypoint` RVA 0x260E50, table RVA 0x3BF081 (suspect, memory doc Q7) | **U, needs spike** |
| kill counts | not stored in game memory | derive from events only (V: bot emits Kill from combat) |
| area visited / cleared | not stored; automap reveal data (U) | derive from `area_change` events (V) and nav `ErrExplored` (V, per game) |
| merc state (alive, level, aura) | NPC unit owned by player | U |
| act unlocked, "can go east" flags | quest bits 7/15/23/28 in D2BS ordering (U) | U |
| town NPC, waypoint object ids | unit scan (class ids per act table) | V for Gheed/Akara/Charsi; others U |

## 2. Per goal family: read vs derive

| family | measured from | metric `current` / `target` | notes |
|---|---|---|---|
| level.reach | memory: level, exp | level / level; sub: exp / xpNext | exists in the manager; add ETA from xp/h |
| quest.* | memory: quest bits (U); fallback derived: step events | steps done / steps total, or 0/1 | see section 3 for sub-steps and the fallback when bits are unknown |
| boss.* / kill_boss | events: `kill` with boss flag (V) | kills / count | exists; per-run time and drops are derivable |
| area.clear | nav explore state; events kills | explored % (U) or kills / untilKills | cleared = `ErrExplored` (per game); persistence across games: derive from `area` events |
| area.explore (visited set) | `area_change` events or automap | areas visited / list size | store the visited set per character (NEW) |
| wp.activate | memory: waypoint bits (U); fallback: `TakeWaypoint` panel rows | active / expected (9 per act, U) | fallback: after a successful `ActivateWaypoint`, record locally; wrong after respawn in another save |
| res.gold | memory: gold + stash gold (location V, scaling U) | gold / target | trivial progress bar; rate from time buckets |
| res.potions/scrolls | inventory: belt contents (V), scroll count | count / target | `town.belt` model; event `potion` already exists |
| item.have / collect_item | events `drop` (exists) and inventory diff | qty / count | drops vs "have" is different: needs inventory scan (V) |
| item.equip | inventory equipment (V) + tier score | slots upgraded | no progress bar, boolean |
| build.* | memory: unspent points (stat 4/5, U), skill levels (V list) | points spent / table entry | level-up events (V) trigger |
| rune.* | inventory scan by item code (V) | runes owned / needed | cube result: new item appears |
| safe.* | events: deaths (V), potions (V) | none (rule, not a bar) | already in settings and `pauseDeathsPerHour` |
| diff.* | memory: difficulty byte (V Normal), quest bits | step in act 1..5 | state `act of difficulty` |

## 3. Existing model versus what must change

Current model (`internal/goals`): `Goal{ID, Type, Title, Params, Priority, Schedule, State, Endless, Progress, Note, ...}`; `Progress{Current, Target,
Percent, StartLevel, Kills, BossKills, Drops, Deaths, ActiveSeconds}`; counting by the manager from events only (api.md: `reach_level` from level,
`farm_area` kills/drops, `kill_boss` boss kills, `collect_item` drop qty, `idle` minutes). Status gives a `Character` card (name, class, level, xp, life,
area, difficulty). It cannot express: sub-steps, state read from memory (quests, waypoints, gold), "already true" goals, dependencies, ETAs, campaigns.

Proposed additions (names are suggestions, additive, JSON-compatible with version 1 files):

```go
// New goal types
const (
    Quest     Type = "quest"      // params.quest = catalog id, params.difficulty
    Waypoints Type = "waypoints"  // params.act 1..5 or "all"
    Resource  Type = "resource"   // params.resource gold|potions|scrolls|keys, params.count
    Build     Type = "build"      // params.what spend_points|skill|stat, params.skill, params.level
    Campaign  Type = "campaign"   // params.template name, children
    Difficulty Type = "difficulty" // params.complete or params.enter
    Explore   Type = "explore"    // params.area, params.mode visit|clear
    HaveItem  Type = "have_item"  // inventory state, not drops: item filter + count + where (pack|stash|equipped)
)

type Params struct { // new fields
    Quest   string `json:"quest,omitempty"`
    Act     int    `json:"act,omitempty"`
    Resource string `json:"resource,omitempty"`
    Skill   int    `json:"skill,omitempty"`
    Template string `json:"template,omitempty"`
    Requires []string `json:"requires,omitempty"` // goal ids or predicate names (planner.md)
}

type Step struct {                      // sub-steps of one goal
    ID, Title string
    State     State                     // queued active done failed skipped
    Current, Target int                 // e.g. 3 of 5 seals
}

type Progress struct { // new fields
    Steps    []Step  `json:"steps,omitempty"`
    Phase    string  `json:"phase,omitempty"`      // "travel", "fight", "turn-in", "cube"
    Source   string  `json:"source,omitempty"`     // "memory" | "events" | "mixed": how Current was obtained
    ETASeconds int64 `json:"etaSeconds,omitempty"` // 0 unknown
    Satisfied bool   `json:"satisfied,omitempty"`  // true at start: nothing to do
    Verified  bool   `json:"verified,omitempty"`   // true when confirmed by a memory read, false when inferred from events
}
```

The Runner interface must also supply read-only **observations**, because quests, waypoints and gold are not events. Add one optional interface so
old runners keep working:

```go
type Observer interface { // implemented by bot.Runner
    Observe() Observation
}
type Observation struct {
    Difficulty string
    Quests     map[string]QuestState // "den_of_evil" -> {Done, Started, RewardTaken, Failed}
    Waypoints  map[int]bool          // area id -> active, current difficulty
    Gold, StashGold int
    Potions    map[string]int        // healing, mana, rejuvenation in belt
    Unspent    struct{ Stat, Skill int }
    Resist     [4]int
    Visited    []int                 // area ids seen this character
    Inventory  []Item                // for have_item and rune goals
    Merc       struct{ Alive bool; Level int }
}
```

The manager evaluates "state goals" (`quest`, `waypoints`, `resource`, `have_item`, `build`) from `Observation` on each tick (cheap, every 2-5 s) and
sets `Satisfied` immediately (a goal already true completes at once, which prevents wasted work). Event-counted goals stay as they are. The runner
still just works on the goal and emits events; the manager still owns completion (api.md contract unchanged).

New events (type, fields) in addition to `kill drop death level_up area_change potion error`:

| event | fields | used by |
|---|---|---|
| `quest_step` | quest, step, state | quest progress and sub-steps |
| `quest_done` | quest, difficulty | quest goals, campaign |
| `waypoint_activated` | areaId | wp goals, area visited |
| `difficulty_change` | difficulty | diff goals |
| `act_change` | act | quest act_complete |
| `item_acquired` | item (even quest items), where | have_item, rune |
| `item_equipped` | item, slot | item.equip |
| `points_spent` | kind stat|skill, name, level | build |
| `gold` | delta, reason (loot, sell, buy, repair) | res.gold rate |
| `town_trip` | reason | statistics |
| `chicken`, `corpse_run` | | safety stats |

## 4. Dashboard presentation
- Goal card: headline bar (Current / Target), phase text, **sub-step list** with check marks for `Steps`, ETA ("about 40 min at 12k XP/h") and a
  "verified by game" or "estimated" badge from `Verified`.
- Quest and campaign view: act columns with a row per quest and per difficulty (done, in progress, skipped, failed/locked), tick state read from
  `Observation.Quests`. The same grid can show the waypoint matrix (act x waypoint) with the visited area.
- Level goals: XP bar to next level, level plan marker (planner's current area and why).
- Resource goals: sparkline of the value over 24 h (reuse `/stats/series`, add `gold` and `potions` metrics).
- Campaign: ordered list with the current child highlighted, "next up" and a "blocked by X" reason from the planner.
- Failures: show `Note` with the last reason and a "retry" and "skip" action (existing endpoints `reset`, `skip`).
- ETA: Level: remaining XP / smoothed xp/h (existing `xpPerHour` in `/status`). Boss/farm: remaining count x average seconds per kill (from the goal's own history).
  Quest: sum of expected effort of remaining steps from a table (catalog.md effort, all U) corrected by recorded runtimes per step (learning: median of the last 5).

## 5. Spike checklist for the main agent (live, read-only unless noted; 24 items)
Run on a throwaway character (Testy/Botty) with the cheat ledger (docs/cheats.md) only where a state change is needed.
1. Find `PlayerData` field offsets after `szName[16]`: read +0x10..+0x30 of `PlayerData`; expect 3 quest pointers then 3 waypoint pointers (D2BS layout, U).
2. Quest buffer: for each pointer, dump the buffer (96 WORDs per difficulty, U) on a fresh character and diff after each action: talk to Warriv, talk to Akara, open quest log (`q`, UI 0x0F verified), kill Blood Raven, finish Den.
3. Map quest ids to indices: confirm the D2BS list (SpokeToWarriv 0, DenOfEvil 1, SistersBurialGrounds 2, ToolsOfTheTrade 3, SearchForCain 4, ForgottenTower 5, SistersToTheSlaughter 6, AbleToGotoActII 7, ... U).
4. Quest bit meaning: which bits mean started, completed, reward taken, "closed" (U); record per quest while playing a save at known states.
5. Check `CheckWaypoint(table, id)` (RVA 0x260E50) is replaceable by direct bit reads; verify RVA 0x3BF081 or the PlayerData pointer (memory doc Q7 suspects the RVA).
6. Waypoint bit order = the 39-waypoint list (U): activate one and diff.
7. Waypoint flags per difficulty (3 tables, U): check NM/H start at only the town.
8. Difficulty byte at RVA 0x3A060C: confirm 1 for Nightmare, 2 for Hell (only Normal seen).
9. Gold and stash gold scaling and where stash gold is (stat 15, U): earn or cheat gold and read.
10. Experience stat (13) value versus `levels.txt`/expTable: confirm `XPNext` computation.
11. Unspent stat/skill point stats (4, 5) after a level up (HUD buttons were seen at UI flags 6, 7).
12. Resistance stats (36..45 mapping) and the difficulty penalty on a Nightmare character.
13. Item fields: socket count, ethereal bit (ItemData flags 0x400000, U), durability and max durability stats (72/73, U), runeword flag, personalised flag.
14. Cube container (Page/location 3?) item read.
15. Quest items: item class ids of Horadric Malus, Scroll of Inifuss, Moldy Tome, Horadric Scroll, Cube, Shaft, Amulet, Khalim's parts, Soulstone, Hell Forge Hammer, Jade Figurine, Wirt's Leg: confirm as `data.DB.Items` classes.
16. NPC dialog menu: for Akara, Kashya, Charsi, Cain, Warriv, Atma, Jerhyn, Meshif, Tyrael, Larzuk, Qual-Kehk, Anya: list the option rows (the box floats above the NPC, town.md).
17. NPC class ids and positions per act (Act 2-5 town tables are from prior-art 05, U).
18. Object ids: quest objects (Tree of Inifuss, Cairn Stones, Horadric Orifice, Compelling Orb, Seals, Hellforge, Altars/chests): class numbers and `Mode` flags for "used".
19. Cairn stone order: random per game or fixed? Read object states while touching in sequence.
20. Superunique detection (`MonsterData` unique number, flags +0x16/+0x1C/+0x26, U) for Shenk, Nihlathak, Summoner, Radament, Izual.
21. Kill detection for bosses: `Mode` 12 or `LifePct` 0 on unit class ids (Pindleskin 786, Nihlathak 794 from kolbot, V as source).
22. Area transitions: area ids in an actual game versus recollection (levels.txt dump compare, docs/plan/data/runs.json).
23. Level data "visited": automap reveal flags or `Level` `pRoom2` "visited" flag (U); otherwise events.
24. Merc: find merc unit (owner = player, class id), alive state and level, hire dialog.
