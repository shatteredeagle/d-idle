# Quest log (TASK-156, unlockquestlog)

Third of the ten unlockables (`docs/research/unlockables.md`); sits on the ladder (`docs/unlock-ladder.md`). Tags: V verified by the offline tests named, S secondary source, U unverified (needs a live run).

## What it is

An Act 1 quest list: for each quest the hero holds, its state (started, cleared, done), the current objective, why the hero is going there, and the reward where known. Ladder row `questlog`: trigger `quest_stage` `den_of_evil` stage 1 or more (Akara hands out the Den of Evil quest). A locked log shows as a silhouette under "Still locked" with the teaser "Take a quest from Akara". Quests the hero has not been given are not listed, only counted ("N more in Act 1 not given yet"), so the log spoils nothing.

## Where the states come from

- V (code read): the bot already reads quest state: `cmd/d2bot/task.go` `saveQuestReader.Read` (memory words first, the save as prior knowledge) and `internal/bot/observe.go` `pollQuests` emit `goals.EvQuest{Quest, Stage, Done, Diff}` whenever a stage rises, every 15 s. The log uses those events and adds no reader and no offset.
- Stage numbers are the task layer's (`task.StageStarted` 1, `StageCleared` 2, `StageDone` 3). Cleared means the objective is met and the reward is not claimed (Den cleared: back to Akara).
- U live: that the reader reports Started right when Akara hands out the Den quest (the quest words are V live per `docs/saves.md`; a drill that talks to Akara and reads the stage was not run here). The `Started && Pending` reading as Cleared is the reader's; see `docs/tasklayer.md`.
- Missing reader: none for Act 1. Quest events of other acts are ignored by the log (the table is Act 1 only).

## Den of Evil, checked against the data

Task layer (`internal/task/act1.go` `denNodes`): talk to Akara, clear the Den (area 8, every monster, V), talk to Akara again for the reward. `docs/plan/data/quest-points.json`: reward "1 skill point", difficulty normal, tag V (Maxroll). The nightmare and hell reward is not in that data, so the table lists the reward for normal only and shows no reward line in other difficulties (U). The other five quests' text and rewards come from general knowledge of the game (S).

## Pieces

- `data/quests/act1.json`: the table, six quests (ids are the task layer's: `den_of_evil`, `sisters_burial`, `tools_of_the_trade`, `search_for_cain`, `forgotten_tower`, `sisters_to_slaughter`), each with an objective and a reason per stage and a reward per difficulty. `ParseQuestTable` rejects a duplicate id, a missing stage, an unknown stage or difficulty, naming the quest.
- `internal/watch/unlockquestlog.go`: `QuestLog`, `GET /api/questlog` (schema `watch.questlog/1`; locked: teaser only). A stage only rises (a replay or a late lower stage changes nothing); a higher difficulty replaces the lower one.
- Engine: trigger kind `quest_stage` (args `quest`, `min` 1 to 3), event `EvQuestStage`; `Unlocks.OnGoalEvent` sends it for every quest event with a stage. The older `quest` kind (Done) is unchanged.
- Page: a "Quests" tab (`web/unlockui.js`), hidden until open.

## Verified

V offline: `go test ./internal/watch ./internal/unlock` (golden payloads for Sorceress and Paladin, locked and open, other acts ignored, no step back, difficulty and reward, unknown row and table errors, the reveal line, trigger validation, table ids are task layer ids). U: the live read at Akara; the page in a browser.
