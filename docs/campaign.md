# Campaign: the guided journey

> status: current (TASK-335 stage 0: spine, loader, bands, decision, state; stage 1 planner: TASK-337; stage 2 API and Journey strip: TASK-338). Nothing here is verified live.

User direction 2026-11-11 ("Guided journey"): "How do we guide the player through the campaign? Currently I am just letting the bot level and farm aimlessly. ... New Character -> Den of Evil -> Blood Raven -> etc. We shouldn't FORCE the players to do anything, but progression should generally follow the game." Default path: all Act 1 quests, each optional-flagged so the player can skip.

## 1. Why the hero ground "level up to 16" (diagnosis, 2026-11-11)

The live Paladin (clvl about 12) showed `QUEST: FIGHTING (LEVEL UP TO 16)`: a `level.16` Repeat step (`task.StepLevel`, 30 min budget) planned by `task.Campaign.Plan` (internal/task/campaign.go). Read from the code (V by reading, the hero's quest flags U: not read live):

- The template is `normal_act1` with `Cain` off (`goals.Params.Cain` defaults to false), so the Search for Cain (floor 10) is not in the list at all.
- Den of Evil and Blood Raven are done (they have floors 3 and 8 and would have been chosen first).
- Tools of the Trade is optional and skipped by `ImbueSkipWhy`: `campaign.imbueMinLevel` is 20 and `campaign.imbueNeedsMerc` 1.
- The Forgotten Tower is optional and skipped by `TowerSkipWhy` (clvl < `route.towerMinLevel` 15 without a living merc). Skipped entries never feed the "lowest floor", so they vanish instead of becoming the next level target.
- Andariel (and the Act 2 hop) remain: the Paladin no-merc floor is 16 (`plandata.BossLevels`, docs/act-plan.md). With no living merc the planner's only remaining target is `level.16`. With the Rogue alive it would be 14.
- So the hero is not aimless by bug; the old planner is a list of walls. The level step does not say which chapter it unlocks, the skipped optionals are invisible, and a clvl 12 Paladin who could rescue Cain, run the Countess (mlvl 7 area) or the Barracks farms instead. Probably also no living merc (Blood Raven done but the free Rogue not hired or dead): a live check (section 9).

In Guided mode the same hero goes to Find Cain (comfortable at 11), then Charsi's hammer and the Countess as their recommended levels come, and only then Andariel (test `TestPaladinClvl12GoesToCain`).

## 2. Terms (proposed glossary text, the coordinator adds it)

- **Chapter**: one step of the campaign a human player recognises (Den of Evil, Blood Raven, Find Cain, ...), with a place, a reward, a why line, a level row and a completion rule.
- **Spine**: the ordered list of chapters of one difficulty (data/campaign/<difficulty>.json). The bot follows it in Guided mode; nothing on it is a lock.
- **Readiness band**: too early (clvl under the floor), ready (at the floor, under the recommended level, or a readiness check fails), comfortable (at or over the recommended level, no failing check). Bands are advice; the player can always Go now.
- **Journey**: the player-facing view of the spine for one hero (the Journey strip, the Up next card, the Why line).
- **Intent**: the player's wish for one chapter: Auto, Ask, Skip, Hold.

## 3. The spine as data (stage 0)

`data/campaign/normal.json`, schema `campaign/1`, embedded by `data/campaign` (package campaigndata), loaded by `internal/campaign` (`campaign.Shipped(Validator)`, `campaign.Parse`). Fields per chapter:

| field | meaning |
|---|---|
| id, title, act, area, areaId, kind | kind is quest, boss, chore or unlock |
| quest | the task-layer quest id the bot runs (internal/task handlers); validated against `task.Handler` |
| requires | earlier chapter ids; a prerequisite counts when done or skipped |
| optional | the chapter may be skipped; the UI offers Skip |
| done | exactly one of quest (task quest done), waypoint (area id known), level |
| reward, why | player text (Up next card, Why line, Chronicle) |
| levels | `default` row plus class rows: floor, recommended, floorMerc, recommendedMerc |
| levelsFrom | `plandata:<gate>`: the class row of `plandata.BossLevels` (Andariel); an unknown class is an error naming it |
| checks | `internal/readiness` check ids that cap the band (a fail caps comfortable at ready) |
| intent, classIntent | default intent; per-class default (casters skip Charsi's imbue) |

Farm zones (`farm`): area, areaId, mlvl, requires (a chapter). `FarmFor` picks the lowest open zone with mlvl >= clvl-5, else the highest open one (pace doc section 1 rule, EXP).

Chapters, Act 1 Normal (levels EXP from docs/sme/pace-and-formulas.md beats unless noted): start (Blood Moor, done at clvl 2), Cold Plains waypoint (2/3), Den of Evil (3/4, casters 5/6; matches `campaign.denMinLevel*`), Blood Raven (8/9, casters 9/10; matches `boss.br*`; intent ask), Find Cain (optional, 9/11), Countess (optional, 11/14, merc 11/13; intent ask), Charsi's hammer (optional, 10/13, merc 9/12; casters skip by default), Andariel (`plandata:andariel`; intent ask). Validation errors name the chapter and the field (`TestParseErrorsNameTheChapter`). Nightmare and Hell are later files with the same schema.

## 4. Bands and the decision (stage 0 engine, stage 1 wiring)

`campaign.Rate(chapter, hero, checks)` gives the band; `Spine.Next(progress, heroState, opts)` the decision; `Spine.Observe` updates the state after it (farm clock, question time, chapters first seen done for the Chronicle and the unlock ladder). Pure functions, tested on Paladin and Sorceress (`internal/campaign/campaign_test.go`).

Decision rules, Guided mode:
1. Walk the chapters in order; skip done, Skip and Hold; skip a chapter whose prerequisite is not done or skipped (unless it is the Go now target).
2. Go now (the player's tap) goes at once, whatever the band; below comfortable it carries a warning, never a refusal.
3. A chapter goes when its band is enough for the policy row `goAt` (recommended by default: comfortable; floor: ready or better). Intent Ask at the floor queues a question instead (veto window: default yes after `AskWait` with no answer; a no holds until the next level-up).
4. After `FarmBudget` in a level step for a ready (floor) chapter the hero offers to go at the floor (the same question).
5. Nothing goes: farm the best open zone for the lowest needed level; the Why line says "Farming Cold Plains for level 10 to unlock Blood Raven" (or "until X is ready (reason)" when a readiness check, not the level, holds it).
6. Every chapter done or skipped: done (the Act 2 hop stays the task layer's).

Modes: Guided (default) follows the spine; Free returns `free` and the caller runs today's planner (the task campaign floors); Orders returns `orders`: only the player's orders, otherwise the hero farms its current safe zone. In every mode the player's manual orders come first (docs/inventory-actions.md priority ladder: safety, then player commands, then the plan); the plan never undoes an order.

Safety walls stay: the Monastery and Tower route guards, the Andariel unattended hold, the kit gates and the chicken rules live in internal/task and still apply under a Go. A band is advice; a wall is safety.

## 5. Stage 1: planner integration (TASK-337)

- A `task.Planner` adapter (Guided) maps the decision to plans: go = the chapter's quest handler nodes; farm = a level step whose title names the chapter ("Level up to 13 for Charsi's hammer", visible budget) and whose area is the farm zone; ask = the same level step plus a return-screen question (the question queue); free = the old `Campaign.Plan`.
- `state/campaign.json` per hero (`campaign.Store`), knobs for `AskWait` and `FarmBudget`.
- Readiness checks for the chapter area come from the bot's readiness evaluator.

## 6. Stage 2: API, Journey strip, nudges (TASK-338)

- `GET /api/campaign` (schema campaign/1 view: chapters with band, intent, done/skipped, the decision, ETA) and `POST /api/campaign/intent` (intent, go now, goAt, mode; same-origin and session gate). Goldens added, none changed.
- One new panel module `journey` (panel.json, versioned, tokens only): chapters as a horizontal path with done, current, ready, too early, skipped markers; tap for reward, readiness, Go now, Skip, Hold, Ask/Auto. The Up next card and the Why line read the same state; a compact form in the top band tooltip.
- Nudges, never locks: the Up next card shows the next chapter, its band, an ETA from the level pace and Go now; a progress meter for the level step; the floor offer after the budget; completed chapters write Chronicle entries and drive ladder unlocks (Cain rescued: the "Identify all (Cain)" command already reads the quest state).

## 7. Build-specific parts

Only data: the class rows of `levels`, `classIntent` and the Andariel row in plandata. No class literal in Go; an unknown class with `levelsFrom` is an error.

## 8. Dead ends and open points

- The old `Act1Options.Cain` default false hides Cain from the planner; Guided mode includes it (optional). The goals param stays for Free mode.
- Mode names: the brief lists Guided, Free and Orders; Free is read as "today's planner plus orders", Orders as "orders only". Ask the user if Free should mean orders only.

## 9. Live checks (none run)

1. On the play hero: read the quest flags and merc state; confirm Den and Blood Raven done and whether the Rogue is alive (explains floor 16 vs 14).
2. Guided mode on a fresh Paladin: chapter order start, Cold Plains WP, Den, Blood Raven, Cain, Charsi, Countess, Andariel; Why lines name the chapter; writes after start 0.
3. Ask flow: Blood Raven at the floor queues a question; no answer for `AskWait` goes; a no farms until the next level.
4. Go now on a too-early chapter goes with a warning line; Skip and Hold survive a bot restart (state/campaign.json).
