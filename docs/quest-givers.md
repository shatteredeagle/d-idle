# Quest givers in town (TASK-243)

> status: decision logic, table and fixtures built and tested offline; the live part is U (checklist below). Tags: SRC read in this tree, U unverified live. Builds on [quest-hints.md](quest-hints.md) (TASK-197), which only speaks.

## What it does

In the original game a quest is given by talking to an NPC. When the hero is in the Rogue Encampment on Normal and a giver has a quest for it, the campaign now makes that visit first, through the existing town talk path (`task.StepTalk` -> `talk.Adapter`), then goes on with its usual plan:

- accept: the quest is not given yet (stage none), it is available, and the hero's level reached the class family's level. Available and level come from the hint table (`data/quests/act1-hints.json`), so each number has one owner. Today: Kashya (Sisters' Burial Grounds; melee 7, caster 8, summoner 9; Cold Plains reached) and Charsi (Tools of the Trade; 12/14/15; Outer Cloister reached). Den of Evil is not in the table on purpose: its own handler opens with the Akara talk.
- turn in: the objective is met and the reward is not claimed (stage cleared): Kashya, pick `reward` (already live-used in the Burial handler).
- not yet available or too weak: nothing is done; the hint desk speaks the "I cannot help yet" line as before (quest-hints.md). Nothing was changed there.

Where it shows: the plan's step title and reason go to the whyline (`quest_step`, "Ask Kashya what troubles the rogues"), the quest word rising shows in the quest log, Chronicle and diary exactly as for any quest event. No new surface.

## Data and code

- `data/quests/act1-givers.json` (schema `quests.givers/1`): per quest the NPC, an `accept` and/or `turnin` action (pick, step title, reason, tag). Joined by quest id to the hint table in `task.ParseGivers`; a quest without a hint row, a giver that disagrees with the hint table, or an action without pick/title/why is an error naming the quest.
- `internal/task/givers.go`: `GiverTable.Errands(world, class)` (pure), `Campaign.WithQuestGivers` (opt-in, like `WithRogueClaim`), `Campaign.giverPlan` at the top of `Plan`. A visit is one optional talk step in its own group `givers.<quest>.<kind>`: it never blocks the chain, never repeats in a session (executor `Finished`), is skipped when blocked, when the campaign does not include the quest (e.g. `SkipTools`) or skips it (`skipWhy`), and outside Normal or the Rogue Encampment.
- Class generic: the class family comes from the hint table's `families` (data); an unknown class is an error naming it, reported once in the log, and makes no visit (V `TestGiverUnknownClassIsReported`). Tests run Paladin (melee) and Sorceress (caster) at the two levels (V `TestGiverVisitByClassFamily`).
- `internal/talk/tables.go`: one row, `{Kashya, "quest"}: Talk` (U live).
- The guided journey (docs/campaign.md) runs after the giver visits: `Campaign.Plan` asks `giverPlan` first, then the spine.
- Switch: `internal/bot/givers.go`. On for the player's game; off in a named run (`D2BOT_RUN`) and with `D2BOT_QUEST_GIVERS=0`.

## Limits (U)

- "Area visited" is approximated by waypoint, cleared or swept areas in the task world (it has no visited-areas set). A hero that walked Cold Plains without touching the waypoint is not visited yet.
- Without a quest reader the stage stays none; the session `Finished` mark stops a second visit. With a save-only reader the stage lags until Save and Exit; same mark.
- Not done: Cain's chain (Akara finishes it, auto on area), Andariel (Warriv, auto on kill), the Tower (no NPC). They stay with their handlers.

## Live checklist (not run; needs a Normal character in the Rogue Encampment, hidden instance)

1. Hero at clvl 7 (melee) or 8 (caster) with Cold Plains waypoint, Burial not given: does the log show `npc: talk npc=Kashya pick=quest path=[Talk]` and does the quest word of `sisters_burial` rise to started? If Kashya's first Talk opens a topic list, the row label goes into `internal/talk/tables.go`.
2. After Blood Raven dies and the hero is home: one `pick=reward` visit, quest word done.
3. Charsi at clvl 12 with Outer Cloister reached (needs the Tools entry in the campaign): quest word started, no Imbue item pick at this step.
4. A hero below the level: no visit, the hint line speaks.
5. `D2BOT_QUEST_GIVERS=0`: no visit.
