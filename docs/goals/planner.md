# Planner: composing goals into campaigns

Scope: DOCS ONLY. Builds on catalog.md (ids), execution.md (how), tracking.md (state we can read). Tags as elsewhere: (V) verified, (S) secondary, (U) unverified.
Sources for the quest graph: https://classic.battle.net/diablo2exp/quests/act1.shtml..act5.shtml (V names/triggers), https://www.purediablo.com/?p=3457 (S rewards),
docs/archive/prior-art/01-leveling.md section 4 (SoloPlay order and difficulty rule, S/U), docs/plan/leveling.md, docs/plan/data/progression-*.json.

## 1. Dependency graph

Edges mean "needs". Hard = game-enforced (V where the quest text says so), soft = safety/efficiency (clvl floors, gear).

```
Act 1:  den_of_evil -(soft)-> sisters_burial -> search_for_cain -> rescue_cain_tristram (soft)
        forgotten_tower(Countess) -(soft: clvl 14)    tools_of_trade (Smith, clvl 8 V)
        sisters_to_slaughter (Andariel, soft clvl 22)  -hard-> act1_complete -> Act 2
Act 2:  radament -(hard: scroll V)-> horadric_staff pieces: cube(HotD3) + viper_amulet(Tainted Sun) + staff_of_kings(Maggot Lair)
        -> horadric_staff -(hard)-> duriel -> act2_complete -> Act 3     [arcane_sanctuary -> summoner: optional (U)]
Act 3:  golden_bird -> blade_old_religion -> lam_esen -> blackened_temple (soft order, V triggers)
        khalims_relics (eye+heart+brain) + flail (Council) -> khalims_will -(hard)-> mephisto -> act3_complete -> Act 4
Act 4:  izual (soft) ; hellforge (needs Mephisto's Soulstone from Act 3 drop, V) ; diablo -(hard)-> act4_complete -> Act 5
Act 5:  siege_shenk -> rescue_barbarians (soft) ; prison_of_ice -> nihlathak -(hard)-> ancients (clvl 20/40/60 V) -> baal -> diff complete
Between difficulties:  Baal dead -> diff.enter:next (new game) -(gate: clvl floor, resist after penalty > 0, gold, not broke; prior-art 01)
```

Cross-act soft ties: `cube` needed for Wirt's Leg (cows), Khalim's Will, runes; `tristram` needs Cain quest done (prior-art 02); `hellforge`
needs the Soulstone dropped by Mephisto (pick up in Act 3, V item). Act changes and talk-to-NPC "go east" steps are part of each act_complete goal.

Machine form (for the planner and the dashboard graph):
```json
{"quest.sisters_to_slaughter": {"requires": ["level>=22"], "unlocks": ["quest.act1_complete"]},
 "quest.horadric_staff": {"requires": ["quest.horadric_cube","quest.viper_amulet","quest.staff_of_kings"]},
 "quest.duriel": {"requires": ["quest.horadric_staff","level>=24"]},
 "quest.khalims_will": {"requires": ["quest.khalims_relics","boss.council_flail"]},
 "quest.mephisto": {"requires": ["quest.khalims_will","level>=28"]},
 "diff.enter:nightmare": {"requires": ["diff.complete:normal","level>=40","res.all>40-after-penalty"]}}
```
Predicates (`level>=N`, `res.all>N`, `gold>=N`, `quest.X`, `have(item)`) are evaluated against `Observation` (tracking.md). Level floors are the
`min_clvl` of docs/plan/data/runs.json plus the progression files (U, recollection).

## 2. Templates and campaigns

A campaign is an ordered list of goals with `requires`, created by one POST (`type: campaign, params.template`). The manager expands it into
normal goals (so queue, scheduling and persistence need no change) and keeps a `campaignId` and `Steps` for the dashboard card.

| template | content | effort (U) |
|---|---|---|
| `normal_playthrough` | acts 1-5 critical path (below) | 25-35 h |
| `normal_full_clear` | adds every optional quest and `wp.activate:all` | 40-50 h |
| `nightmare_playthrough` | same critical path, NM floors, skips optional quests that give nothing (Tristram, Den) | 45-70 h |
| `hell_playthrough` | critical path only, strict resist/level gate | 100-200 h |
| `hell_baal_runs` | `boss.baal` x N with chores, loop until a stop condition (drops, kills, time) | endless |
| `rune_farm_countess` | `boss.countess` loop with `collect_item rune` stop at Tal+Eth (runewords.json) | 2-6 h |
| `mf_pindle_eldritch_shenk` | `boss.act5_trio` loop | endless |
| `gold_and_gear_recovery` | after repeated deaths: `res.gold`, `res.potions`, `item.equip` and a safe leveling area | 1-3 h |
| `waypoint_sweep` | `wp.activate:act1..5` | 4 h |

### Critical path of `normal_playthrough` (goal ids from catalog.md, in order)
1. `level.reach:5` via Blood Moor and Den of Evil (`area.clear:den_of_evil`, `quest.den_of_evil`) then Cold Plains, Burial Grounds (`quest.sisters_burial`).
2. `quest.search_for_cain` is skipped in the critical path unless the cube/Tristram is wanted (U; Cain is needed for Act 2 scroll dialog, check on a spike).
3. `level.reach:10`, `quest.forgotten_tower` is skipped until clvl 14 (Countess drops runes), `quest.tools_of_the_trade` after clvl 8 (Smith).
4. `level.reach:22`, `quest.sisters_to_slaughter`, `quest.act1_complete`.
5. Act 2: `quest.radament`, cube + amulet + shaft, `quest.horadric_staff`, `level.reach:24`, `quest.duriel`, `quest.act2_complete`. Optional `quest.summoner` when Teleport exists.
6. Act 3: relics, Flail, `quest.khalims_will`, `quest.blackened_temple`, `level.reach:28`, `quest.mephisto`. Optional quests (bird, blade, Lam Esen for +5 stat points, S) when clvl is high enough.
7. Act 4: `quest.izual` (2 skill points, V), `quest.hellforge` optional, `quest.diablo`.
8. Act 5: `quest.siege_shenk`, `quest.prison_of_ice`, `quest.nihlathak`, `quest.ancients` (clvl 20, V), `quest.baal`.
9. Gate, then `diff.enter:nightmare`.

## 3. Suggested default campaign for a brand-new character
(Matches the primary build, docs/plan/leveling.md: Blizzard Sorceress; a different build changes only the level floors.)
1. `idle` (first hire/stock) is not needed; first chores: buy potions and TP tome (`res.scrolls:tp_tome`, `res.potions:belt_healing`).
2. `build.hotkeys:bound` (exists), then `level.reach:5` (Blood Moor, Den).
3. `quest.den_of_evil` (skill point to Charged Bolt per leveling.md, V repo), `quest.sisters_burial` + `quest.hire_rogue`.
4. `level.reach:12` (Cold Plains, Stony Field, Burial Grounds) with `wp.activate:act1` on the way.
5. `quest.sisters_to_slaughter` at clvl 20-22 (runs.json 22, U), `quest.act1_complete`.
6. Continue `normal_playthrough` from Act 2.
Then the user can switch to `nightmare_playthrough` or an endless farming template. All goals are normal queue entries, so the dashboard shows
the order and the user can reorder or remove them (existing API).

## 4. Auto-planner: choose the next goal from character state

Input: `Observation` (tracking.md), settings (difficulty preference, chicken), history (deaths per area, goals done). Output: the next goal to enqueue.
Runs when the queue is empty, after a goal finishes, after a death spiral, or on a schedule. Pure function, unit-testable with fake observations.

```
func next(o Observation, h History) Goal:
  if o.Gold < floor(level) or no potions or no TP: return res.* chore goal           # not broke gate (prior-art 01 Check.broke)
  if stat or skill points unspent: return build.spend_all                              # immediate
  if recent deaths in the leveling area >= spiralDeaths: return level.reach with area banned (existing bot behaviour)
  d := o.Difficulty
  for q in criticalPath(d) in order:                                                    # first unsatisfied quest
      if satisfied(q, o): continue
      if requiresMet(q, o) and levelOK(q, o) and gearOK(q, o): return quest goal q
      else: return level.reach(minLevel(q))                                             # level up toward the gate
  # critical path done:
  if d is hardest or no gate passed: return endless farm (template per level: mf runs or xp farm)
  if gate(d -> d+1): return diff.enter(d+1) else: return level.reach(floor(d+1)) or gear/resist farm
```
Rules: (1) never choose an act the config does not allow (`acts` in bot config, bot.md), (2) prefer cheap goals when already near (same area), (3) interleave
chores before long goals, (4) after 3 failures of one goal, mark it skipped for the session and pick the next independent goal, (5) escalate to the user
(a `needs_attention` note on a goal) for: quest handler step failing 3 times, softlock suspected, stuck at a gate for 2 h, (6) stop condition always present
(`untilMinutes`) so no goal can hang forever (the manager already has time limits; watchdog ladder in bot.md).

### How a user sees it
- Auto mode on: planner appends goals to the queue with a note "auto: why" (e.g. "need clvl 22 for Andariel (S)"). Off: nothing appended; empty queue = idle.
- Manual goals always outrank auto goals (priority 0 vs -1).

## 5. Ordering and priority policy
- Safety and chore goals (potions, TP, repair, merc revive) are not queue items but preconditions the bot already handles (`town.NeedsTrip`).
- Quest goals within an act run in the dependency order above; different acts never interleave except for chores.
- Level-up goals are inserted by the planner just in front of a gated goal and removed when the gate is met.
- A schedule window on a campaign applies to all its children (manager `schedule` field already on goals).
- Concurrency: one active goal at a time (existing manager). Campaign expansion preserves it.

## 6. Open design choices
- Whether to treat campaigns as a parent goal with children or to expand them into plain goals only (the doc recommends plain goals plus a `campaignId`).
- Whether `satisfied` checks run on every tick (cheap) or only on goal start (simpler); recommended: on start and every 5 s while active.
- Hardcore: not designed here; the death policy and gates (corpse retrieval off, higher level floors) need a separate pass.
