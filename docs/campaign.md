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

Safety walls stay: the Monastery road guard and the tower guard, the Andariel unattended hold, the kit gates and the chicken rules live in internal/task and still apply under a Go. A band is advice; a wall is safety.

## 5. Stage 1: planner integration (TASK-337)

- `task.Campaign.WithGuide(*task.Guide)` (internal/task/guided.go): in Normal Act 1, after the town quest givers, the spine picks the plan. Go = the chapter's quest handler (`quest.<id>`), or a waypoint or level step for chapters without a quest; farm and ask = a `level.N` step titled "Level up to N for <chapter>" (the status line and the Why line show it); Orders mode = a level step at the current level plus one; Free mode, Act 2 and later, Nightmare and Hell, and a finished spine hand over to the floor planner (the Act 2 hop stays there).
- The floor planner's per-quest floors (`Entry.floor`: Den kit floor, Blood Raven class and wait level, route guard, tower guard, Andariel class row and unattended hold) are the chapter's safety wall (`Opts.Wall`): a wall above the data floor raises floor and recommended, so a band never sends the hero into a walk the guards refuse.
- Floor-planner entries without a chapter (the free Rogue claim) still run first when due.
- A quest the executor blocked this session is held for the session only (a copy of the state; the persisted intents are unchanged).
- `campaign.Controller` (internal/campaign/controller.go): per-hero state in `<state>/campaign.json`, `Decided` runs Observe and saves, `Apply` takes the player's requests (intent, go now, answer, goAt, mode), `View` is the API body. `cmd/d2bot/journeywire.go` builds it (veto window 10 min, floor offer after 30 min of a level step; U, constants until they become knobs) and `bot.Runner.SetJourney` installs it. A named run (`D2BOT_RUN`: gates, soaks, drills) never gets the guide, so measured runs keep the floor planner.
- The level step's area is still the bot's own level planner (progression table, danger, bans); the spine's farm table only names a zone when the bot has none. The floor question is shown on the Journey strip (stage 2); routing it through the return-screen question queue (decide catalogue point) is a follow-up card because the catalogue is a shared registry.

## 6. Stage 2: API, Journey strip, nudges (TASK-338)

- `GET /api/campaign` (`internal/watch/campaign.go`, body `campaign.View`, schema `campaign/1`): mode, goAt, level, the decision, the level meter (`from`, `to`, `pct` 0..1, `etaSeconds`), the veto window, and per chapter id, title, area, kind, optional, reward, why, state (done, skipped, held, current, next, open, waiting), band, floor, recommended, reasons, intent, asked. Goldens: `internal/watch/testdata/campaign/view-*.json` (Paladin clvl 12, Sorceress clvl 7).
- `POST /api/campaign/intent` on the answer listener (the policy gate: same origin, session token, JSON, unknown fields refused): one of `{chapter, intent}`, `{chapter, goNow}`, `{chapter, answer}`, `{goAt}`, `{mode}`; the reply carries the new view. 503 `off` until the bot bound the controller.
- Panel module `journey` (`internal/watch/web/panels/journey/`, manifest version 1, tokens only, fit, side band, placed at the end of `web/layout.json` from `tools/uishots/layout-rows.json`): the chapters as a numbered path (`data-state`, `data-band`, `data-key` per step), the Up next block (next chapter, band, the decision's Why, the level meter with ETA, buttons), and a detail block for a tapped chapter (reward, band with reasons, Go now, Skip for optional chapters, Hold or Resume, Ask first or Auto; Go now and Not yet when the floor question is open). DOM golden: `testdata/campaign/journey_dom.txt`.
- The Why line reads the same state: the level step title names the chapter (stage 1). The meter comes from the experience table (`telemetry.DefaultExpForLevel`) and the rate since the target was set (`cmd/d2bot/journeywire.go`).
- Not built (follow-up cards): the floor question in the return-screen question queue (a decide catalogue point), the compact top band tooltip (an edit of the hero-status panel: a panel family change), Chronicle lines and unlock-ladder triggers on a chapter done (`Controller.OnDone` only logs today; the ladder has a `quest_stage` trigger kind, so Cain rescued can open "Identify all" through data), and knobs for the veto window and farm budget. Wiring them needs owners of `data/unlocks/ladder.json`, `internal/decide` and the chronicle.
- Layout: adding the panel re-packs the tablet placement of the `alerts` card (the generator pairs half-width cards); the other breakpoints only gain the new panel.

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
5. The Journey panel on the watch screen: the path matches the hero's quests, Skip/Hold/Go now buttons post and the strip updates, the meter moves with XP, the panel fits at 390 and 1440 px (uishots on the worker host).
