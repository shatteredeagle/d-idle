# Quest board model (slice S1), executor (S3), API (S4) and chunked quests (S7)

> status: S9 DONE offline 2026-11-17 (card retirej, TASK-629): the Journey is retired and `quests.board` defaults to 1, see 'Journey retired'. S7 chunked quests (Acts 1 to 5 data, per-chunk handlers) built and tested offline, NOT verified live (see 'Chunked quests'); S1 model built and tested offline (card quests1, TASK-577); S3 executor built and tested offline behind the knob `quests.board` (default 0; card quests3, TASK-585), NOT verified live (the drill below has not run). Design: [docs/design/quest-board-2026-11-16.md](design/quest-board-2026-11-16.md). Tags: V verified by the named test or file, S secondary, U unverified live, I design choice.

## What is here

| piece | where | notes |
|---|---|---|
| definitions | `data/questboard/act1.json`, `act2.json`, `families.json`; embedded by `data/questboard/embed.go` | schema `quests.def/1`; strict decode (an unknown field is an error) |
| loader, validation | `internal/questboard/def.go`, `load.go` | unknown class, act, quest, chunk, difficulty, kind, risk: an error naming it (V `TestParseErrors`, `TestLookupsNameTheUnknown`); no silent fallback |
| states, derive | `internal/questboard/state.go` | nine states, `Derive`, `CanStart`, the `Transitions` table with the evidence tier each needs |
| plan records | `internal/questboard/plan.go` | per hero `<dir>/<hero>.quests.json` through `internal/store.File` (tmp, fsync, `.bak`, quarantine, future version refused) |

Quest ids are the save ids (`internal/saves` questTable; V `TestQuestIDsAreSaveIDs` reads that file). `taskId` carries the task-layer id where it differs (`sisters_burial`, `sisters_to_slaughter`).

## Rules the code enforces

- Derive is a pure function of its input: derive twice, same result (V `view` helper in every test). Handler progress is never stored; the only store is the player's decisions (accepted, skipped, auto-start opt-ins, seen toasts).
- Evidence: a quest prerequisite unlocks only on T1 (save) or T2 (memory); T3 (proxy) never unlocks (V `TestTransitionTableEvidence`, `TestChunkOrderAnyThenAssemble`). T3 may stop the hero at the NPC and show "done, confirming"; it pays nothing until T1/T2 confirms.
- Evidence conflict (I): for a single-chunk quest a fresh T2 "not done" beats a T3 "done"; a stale T1 "not done" does not beat a proxy (shown as confirming). A part of a multi-chunk quest has no bit of its own, so its proxy stands.
- Chunk order (I): a chunk's `after` is satisfied by any evidence of the earlier chunk (an offer skips no work, the handler re-derives); a chunk's or quest's `requires` need T1/T2; a quest's `requiresCleared` needs only the named quest's objective at any tier (a proxy such as the made item counts), for a prerequisite the quest itself completes (the Seven Tombs completes the Horadric Staff).
- Start tap: `CanStart` refuses anything that is not `ready` (locked, out of order, available, done); `Store.Accept` refuses a second accepted chunk in the same difficulty.
- Readiness: S2 is not built. Without a readiness word a chunk stays `available` ("readiness not known"); the tests stand in for S2 with the legacy floor per class family (clvl by melee, caster, summoner) on Paladin and Sorceress.
- Meta currency (design section 7): `Award.Key` is `quest/<quest>/<chunk or ->/<difficulty>`, never the hero (once per account per quest per difficulty). A quest the board never started (`byHand`) pays nothing: no backfill (user decision 2026-11-16). T3-only completion pays nothing yet. The amounts are data placeholders (U) in the `meta` columns, about 5 for a first Normal story quest and 2 for an optional chore.
- `autoStart: never` rows (hazard risk) refuse the opt-in (V `TestStoreSkipAndAutoStart`); the loader requires `autoStart never` on every `hazard` quest.
- Build flexibility: no class or skill literal in Go; the class-to-family table is `families.json`, the floors are the `legacyFloor` column per quest.

## The data and where it came from

Act 1 merges `data/quests/act1-hints.json` (giver, clvl floors, EXP/UNSURE tags), `act1-givers.json` (accept and hand-in picks) and `docs/goals/execution.md` section 3 (steps, boss, rewards, Req); the per-field `tags` map names the source of each field. Act 2 has no hint or giver table, so it comes from execution.md section 3 and docs/progression/quests.md alone and every row is U until a live drill. Chunks: Cain's chain is four (scroll, read, free Cain, report), the Horadric Staff is three (cube, shaft in any order, assemble after both and the Tainted Sun amulet). The old hint and giver files stay until S9 retires them.

Known gaps (U): the hand-in picks of Akara, Atma, Cain, Charsi (`imbue`), Warriv (`Go East`) and Meshif (`Sail East`) are not in `internal/talk/tables.go` as read here; `minutes` are my estimates (I); the Andariel `softAfter` the Burial quest is the hint table's UNSURE availability, kept soft on purpose so it cannot lock a player out; whether Radament gates the Staff (execution.md says V, kept as a hard `requires`).

## For S2 to S4

S3 builds `Facts` from `quest.Reader` (`Info.Status`, `Pending`, `Evidence` map to `Fact` and `Tier` by value: `EvProxy`=1, `EvSave`=2, `EvMemory`=3) and `ChunkFact` from the handler checks; S4 (done, section "API and stores") calls `CanStart` then `Store.Accept` through `Desk`. No knobs were added (the design's `quests.*` knobs belong to S3).

## Executor (slice S3)

Knob `quests.board` (default 0, docs/knobs.md). With 0 nothing here runs: `Exec` answers `ErrOff`, `boardStep` is never called, the planner plays quests as before (the only change on that path is the executor-creation block of `taskStep` moved into `ensureTaskExecutor`, same code; V by the unchanged `internal/bot` and `internal/task` suites, `TestQuestBoardKnobOffKeepsThePlanner`).

| piece | where |
|---|---|
| API-free interface for S4 | `questboard.Exec`: `Offers()`, `Accept(q, chunk)`, `HandIn(ctx, q, chunk)`, `State()`, plus `Cancel` (the player's Stop), `Reset`, `Pause`, `Unblock` (Try again), `SetEnabled`, `SetMode` |
| loop interface | `Next()` says idle, run, await, blocked, paused or busy; `Report(decision, StepResult)` feeds a slice back; `Tick(ctx, stepper)` is both for drivers without a guard |
| seams | `Env` (difficulty, facts, per-chunk handler checks, readiness) and `Actions` (hand-in dialogue, stop); `questboard.Fake` implements both over a scripted world for tests and the S4 API |
| bot adapter | `internal/bot/questboard.go`: `Runner.NewQuestBoard(board, store)` builds the Exec, `boardStep` is the loop tick, hooked in `goal.go` before `taskStep` |
| plan store | `Store.AcceptIf` lets a finished chunk's record stay (proof the board ran it, awards) without blocking the next Start |

Flow (tests: `internal/questboard/exec_test.go` on a Paladin and a Sorceress with the Fake, `internal/bot/questboard_test.go` on the real runner with the campaign rig):

1. `Offers` lists the Ready chunks (`Startable` false while a chunk is accepted). Readiness is `Board.LegacyReadiness` (the S1 legacy floor per class family, safe at the floor, risky within 2 levels): the S2 model (`readiness.go`, a `Word` per chunk) is merged, but the bot adapter does not build its survival inputs yet, so wiring it into `Env.Ready` is a follow-up (U).
2. `Accept` records the tap for the CURRENT chunk only. It refuses a chunk that is not Ready and a second accepted chunk, naming the one that runs (V `TestAcceptRunAwaitHandIn`, `TestTwoAcceptedRecordsNeverRun`, `TestStartCoversOnlyTheCurrentChunk`). The same tap twice is a no-op.
3. The loop calls `Next`. Accepted or in progress: one slice of the quest's task-layer handler (`task.Handler`) minus its hand-in step, run by the shared task executor inside the runner's guard (same `r.do` reactions, death policy and watchdog as a campaign step). The chunk is awaiting when the quest stage reads Cleared or better and no needed handler step is left (`ChunkFact.Objective`), or on the pending bit with T1/T2.
4. Awaiting: the hero walks to the giver (one open-and-close of the dialog, once per chunk and process) and rests in town. It never talks the reward by itself (V `TestQuestBoardAcceptRunStopHandIn`).
5. `HandIn` (the player's tap) needs the chunk Awaiting. The dialogue runs on the loop goroutine (a one-slot mailbox, so the guard applies), then the board re-derives: Done with T1/T2 is `done` and returns the meta awards; a proxy-only Done (the dialogue ran, the save has not caught up) is `confirming` and pays nothing until T1/T2 agree; a quest word that still reads not done retries (at most `MaxHandIn` 3 per tap), then parks the chunk with a reason. A repeated tap after success returns `Already` and runs no dialogue. A fresh T2 "not done" beats a proxy "done" (S1 rule, V `TestHandInReadBack`).

Retry, park, resume:

| situation | behaviour | test |
|---|---|---|
| a slice fails | counted at most once per 15 s in the bot (`boardFailGap`, so a gear gate does not park in 3 s); 3 in a row park the chunk with `gave up after 3 tries: <why>` | `TestFailureDeathParkResume` |
| nothing left to run but the quest does not read finished | counted as a failure (no silent loop) | bot adapter, U |
| the hero dies | the bot's own recovery runs; the chunk resumes from the evidence; the second death on one chunk parks it (`died 2 times`) | same |
| parked | state Blocked with the reason (Needs-you row in S5); the hero trains meanwhile; `Unblock` or a restart tries again | same |
| restart | no handler progress is stored; counters and parks are in memory; the chunk is re-derived from the quest word, the handler checks and the plan record (InProgress shows from any satisfied step) | `TestRestartAtEveryStep` |
| Reset, knob off | `Reset` clears counters and parks and calls `Actions.Stop`; the record stays accepted | `TestResetPauseCancel`, `TestSwitchOffKeepsRecords` |
| Pause | a running chunk holds; an awaiting one stays awaiting | `TestResetPauseCancel` |
| Stop (`Cancel`) | the record goes, the chunk is Ready again | same |

Between chunks (nothing accepted, parked or paused-free) the hero trains: one `task.StepLevel(clvl+1)` slice per tick, the same path the campaign's level step uses. The modes (train, idle, farm) are slice S6 (section Modes below).

Limits and unknowns:

- Only single-chunk quests with a task-layer handler run (U for the rest): a quest without a handler or with several chunks (Cain, the Staff, Khalim's pieces) has readiness unknown, so it is never offered; per-chunk handlers are slice S7. Act 1 quests have handlers; Act 2 to 5 handlers exist in `internal/task/act2..5.go` and run the same way once their data rows are offered, none drilled.
- The hand-in step of a handler is the Talk step whose id ends in `/reward` (V by grep, ten handlers). Quests whose hand-in is another talk (Alkor, Tyrael, Jerhyn rows) are S7.
- `ChunkFact.Steps` marks the share of satisfied handler steps over the board's named steps (I: the names do not match the handler ids); it drives the strip text only, never a decision.
- A mid-walk cancel is not implemented: Stop and Reset take effect at the next slice boundary (U).
- The reader is the save by default (`QuestsLive` false), which only changes on Save and Exit: a hand-in then reads `confirming` until the next save, and a restart before it shows the chunk awaiting again (the second tap talks the NPC once more; harmless for a done quest, U).
- Wiring: `cmd/d2bot` does not construct the board yet; S4 calls `Runner.NewQuestBoard` next to the API. Until then `boardStep` with the knob on and no board only trains (logged once).
- Not in S3: auto-start (S8), the meta ledger (S10; `HandInResult.Awards` carries the keys), modes (S6), the API (S4).

### Which goals run the board, and what the player sees while a chunk waits (card questrun, 2026-11-17)

Live bug (sandbox build 8, hero on goal `farm_area`, Dark Wood): the Start tap was recorded in the plan file and the hero went on farming for 13 minutes. Cause (V by the log: `goal":"farm_area"` on every minute line, no board line; and by code): `boardStep` was only called from the `c.spec.camp != nil` branch of `runGoal`, so under any non-campaign goal (farm area, reach level, kill boss) `Exec.Next` was never asked. The accepted chunk, and a hand-in tap, had no consumer.

Now `boardPreempts` (internal/bot/questboard.go) sends a non-campaign `farm_area`, `reach_level` or `kill_boss` goal into `boardStep` while `Next` says run, await or busy (knob 1 only; knob 0 returns false first). A chunk starts only when `safeToPause` holds (in town, or nothing hostile in the clear radius), so a fight is never cut; the hand-in is never delayed. The Farm gold order, a recovery that must refill the purse and a `runs` goal stay above it. When the chunk ends the hero goes back to the standing goal. A parked chunk (`Blocked`) or a Pause does not take over: the hero keeps its goal and the reason is shown. The slice walks or portals to the quest area by the handler's own steps (U live: the walk from another area of the act; an act change stays a campaign matter).

Never silent: `Strip.queued` and `Running.queued` in GET /api/quests carry the reason while a chunk is accepted and not running ("Starts when the fight is over.", "Parked: ...", "Paused: ...", else "Queued: starts at the hero's next free moment."; `DeskConfig.Queued` = `Runner.QuestQueued`). Log lines (INFO, all start with `quest board:`): `accepted <q>/<c>`, `starting chunk <q>/<c>`, `<q>/<c> queued: waiting for the fight to end`, `step <name>`, `waiting: <why>`, `parked <q>/<c>`, `awaiting hand-in of <q>/<c>`, `handed in <q>/<c>`, `hand-in failed <q>/<c>`, `<q>/<c>: <failure>`. Tests: `internal/bot/questboard_goal_test.go` (two classes, farm and level goals, fight delay with reason, knob 0), `TestDeskQueuedReason`. NOT verified live.

### A started chunk and the hero's own rules (card cainfix3, 2026-11-19; V by the Proving Ground runs named in the card result)

Findings of the Proving Ground chain runs and what the bot does now:

- Fight gate: a fight in the clear radius delays only the START of a chunk. After `boardQueueGrace` (20 s on the virtual clock) the start wins over the level goal's area choice ("starts although hostiles are near" in the log), and a chunk already under way is never gated again (`boardRT.startedKey`). Before this the level sweep kept a hostile in the radius at nearly every decision and `cain.scroll` (Sorceress) and `cain.stones` never started.
- Purse recovery: a chunk with `noGold` in `data/questboard/*.json` (scroll, stones and free_cain of Search for Cain buy nothing) is not held by the purse and kit recovery, on the preempt path and on the campaign path. Default false: every other chunk still waits.
- Area bans: a death ban (`bot.dangerBanMin`) on a field area no longer stops a chunk the player started, unless the readiness word of that chunk is `too_hard` (`Runner.banWaive`, set in `boardRun`; `questWins` and `taskEnv.World` honour it; nav-fault and border-loop skips are not touched). A ban that does apply is named in the refusal: "area banned after N deaths (until HH:MM)" (`task.World.BanNote`), which the board shows as the parked reason.
- Crowd: while such a chunk runs, `combat.crowdAbandonQuestN` (40) replaces `combat.crowdAbandonN` (15) when it is larger (`Fighter.SetQuestCrowd`, `SafetyInput.Quest`). The drink, portal and chicken lines are unchanged. 0 = no override.

## Live drill, Act 1 (checklist, NOT run)

Throwaway character (test-characters rule: create it normally, no setup writes if possible; log it in docs/probes/test-characters.md), one hidden instance, `tools/live.sh`, `quests.board` = 1, a board wired through S4 or a drill command that calls `Offers`, `Accept`, `HandIn` on the runner. Writes after start must be 0 (`d2cheat ledger --since-run <run>`).

1. Fresh Normal character, clvl 1 to 3, Rogue Encampment. Knob 1, campaign goal. Expect: the hero trains, no Akara talk, no quest travel, `Offers` lists Den of Evil as Ready (Paladin from clvl 1, Sorceress from clvl 2).
2. Den of Evil: `Accept`. Expect the log `quest accepted`, the walk to the Den, the clear, the return to camp, a single Akara dialog open and close, `Next` = await. Expect NO reward talk. Stand 60 s: nothing happens. Then `HandIn`: expect one Akara reward dialog, the skill point appears (check the unspent skill points), outcome `done` if the memory reader is live, `confirming` with the save reader. Tap again: `already`, no dialog.
3. Restart the bot while the Den clear runs and again while awaiting: it must resume or stand awaiting without redoing the clear.
4. Death: let the hero die once in the Den (or note the first natural death): after recovery the chunk resumes; a second death parks it with a reason.
5. Blood Raven: same from clvl 6 to 8 (Paladin floor 7, Sorceress 8; below it the offer must be absent or risky). `Accept`, Burial Grounds, Blood Raven, Kashya awaiting (Kashya row U: the approach open-and-close may fail; record it), `HandIn`, the Rogue claim is a separate step and must not run by itself.
6. One chunk at a time: with the Den running, `Accept` Blood Raven must be refused naming the Den.
7. Evidence: note which tier (T1 save, T2 memory) answered at each step in the log, and whether the hero stood next to the NPC.
8. Record in docs/probes: tiers, durations per chunk, whether the approach dialog opened, any parked chunk and its reason. Leave the hero in town, no panels open; stop the instance with `tools/game.sh stop`.

## Readiness (slice S2)

> Built and tested offline (card quests2, TASK-586). Pure query: no bot code imports it, `bot.zoneModel` stays 0 and nothing here changes zone choice. Tags as above.

| piece | where |
|---|---|
| thresholds (data) | `data/questboard/readiness.json` (`quests.readiness/1`): dial lines `enter`/`stay` per cautious/balanced/bold, kit minimums (`minPotions`, `minResist` per difficulty, U), default dial |
| core | `internal/questboard/readiness.go`: `Thresholds.Assess(AssessInput) Readiness`, reason codes, the sentence |
| power adapter | `internal/questboard/readiness_power.go`: `HeroFrom(power.Hero)`, `EncounterFrom(name, kind, power.Result, k)` (k = survival ledger `KZoneFor * KHeroFor`, the same division the controller does) |

- Word rule (V, same as `survival.word` and `power.ZoneMargin.Word`; a test compares them): `safe` when the Low margin is at or over the dial's `enter`, `risky` when Best is at or over `stay`, else `too hard`. The word of a chunk is the worst of its encounters (the boss, and the zone on the way when the caller passes it). The dial lines mirror `internal/power/profiles.json` (V `TestDialsMirrorThePowerProfiles`).
- `Readiness{Word, Reasons, LegacyFloor, Unknown, FromLegacy, Sentence, Debug}`; the margin is only in `Debug` (tools), never in the sentence. The S1 string type `Readiness` is now `Word` (`ReadySafe`, `ReadyRisky`, `ReadyTooHard`, `ReadyUnknown`); `Input.Ready` maps chunk keys to `Word`.
- Reason codes (closed): `level_under`, `no_merc`, `kit_potions`, `kit_resist`, `margin_low`, `margin_wall`, `offence_unread`, `legacy_only`, `no_data`. Order: fixable first (merc, kit), then level, then the margin, then offence. The level number appears only in `level_under` and is the legacy floor (user asked for it beside the margin).
- Legacy floor fallback (I): where no encounter has a margin (boss missing from the tables, hero unread), the old clvl floor of the quest's family decides: at or over the floor `safe`, under it `risky`, and `too hard` when the hero is more than `quests.legacyWall` (default 2) levels under the floor (2026-11-25: a clvl 4 Paladin started the Cain scroll at floor 9 and died on the Underground Passage; `quests.legacyWall` 0 restores warn, never wall). `FromLegacy` is true so a later auto-start can refuse to trust that `safe`.
- Unknown: no margin and no floor gives `Unknown`, an empty word, reason `no_data`; S1 keeps the chunk `available`, never offered (V `TestUnknownReadinessKeepsQuestAvailable`).
- Tests (V, `internal/questboard/readiness_test.go`): Paladin and Sorceress on made-up zones sized to the logged outcomes (Cold Plains clvl 4 and 5 safe, Black Marsh clvl 4 too hard, clvl 12 with a merc safe, without a merc risky; docs/survival/README.md). The Sorceress rows are regression rows, not logged outcomes. A synthetic `survival.Ledger` with two merc deaths and a death in Black Marsh turns a safe word into a harder one. U: real zone and boss tables are not used (the install is on the worker host); no live run.
- Not done (for S3): choosing the encounters per chunk. Quest rows carry only a `boss` id, not an area; S3 resolves `boss` through `danger.Table.FindBoss`/`power.BossPack` and passes the zone of the route, plus the dial from the hero's setting.

## API and stores (slice S4)

> Built and tested offline (card quests4). Behind the knob `quests.board` (default 0): with 0 every action answers `off` and the GET shows `enabled:false` with the reason; nothing else changes. NOT verified live; the curl list below has not run. No new knob (the design's `quests.*` knobs arrive with S6 to S8). Tags: tests are V (they ran), the hero's real margin words and the live behaviour are U.

| piece | where |
|---|---|
| desk (the API face of `Exec`) | `internal/questboard/desk.go`: `Desk.View()`, `Start`, `HandIn`, `Cancel`, `Retry`, `SetMode`; every action returns a `Reply` |
| routes | `internal/watch/questsapi.go`: GET on the viewer mux, POSTs on the answer mux (same `policyGate` as repair, pause and farm gold: origin list, token, JSON only, small body, unknown field refused) |
| wiring | `cmd/d2bot/questwire.go` (`wireQuestBoard`, `bindQuests`, glance rows), called next to `wireJourney` in both backends; `bot.Runner.SetQuestBoardMaker`, `QuestBoard`, `QuestReadiness`, `SetQuestMargin` |
| readiness | `bot.Deps.QuestMargin` (survival model, read only, `survivalShadow.questMargin`) feeds `questboard.Thresholds.Assess`; `bot.zoneModel` is untouched |
| stores | `<state>/quests/<hero>.quests.json` + `.bak` (`internal/store.File`); mode stored in the same file (`Plan.Mode`, default `train`) |

Routes (V `internal/watch/questsapi_test.go`, goldens in `internal/watch/testdata/quests/`):

| method, path | body | reply |
|---|---|---|
| GET `/api/quests` | none | the view, `Cache-Control: no-store`, derived at most once per second (cached) |
| POST `/api/quests/<id>/start` | `{}` or `{"chunk":"<id>"}` | accepts the CURRENT chunk (the offered one, never the next part). 200 |
| POST `/api/quests/<id>/handin` | `{}` or `{"chunk":".."}` | 202 `pending:true`: the dialogue runs on the bot loop (can wait for a town trip or a recovery, bounded to 5 min); the outcome is `handIn` in the next GET |
| POST `/api/quests/<id>/cancel` | same | stops an accepted chunk (record removed, chunk ready again) |
| POST `/api/quests/<id>/retry` | same | clears the park of a blocked chunk ("Try again") |
| POST `/api/quests/mode` | `{"mode":"train|idle|farm"}` | stored per hero; slice S6 gives it behaviour (`modeActs` is true while the board is on (S6)) |

Reply (every action): `{ok, already?, pending?, code?, reason?, quest, chunk, state, mode?}`. `ok:true` with `already:true` is a repeated press (idempotent). A refusal is `ok:false` with a `code` and a `reason` in words, never "could not do that" (the Pause bug of 2026-11-16: the page now reads state back from the GET and uses `ok`). Codes and HTTP status: `off` 409 (quests.board is 0), `no_hero` 503 (no snapshot yet), `paused` 409 (the player's pause is on), `busy` 409 (another chunk runs, named in the reason), `not_ready` 409 (not offered; the reason carries the evidence or readiness text), `not_awaiting` 409 (hand-in before the hero stands at the NPC), `done` 409, `unknown` 404, `bad_mode` 400; gate failures are 401 (token), 403 (origin), 405, 415, 400 (body), 503 (desk not bound).

GET fields: `schema watch.quests/1`, `enabled`, `reason` (when off), `difficulty`, `paused`, `mode`, `modeActs`, `meter{mode, applied, line, target, targetName, word, levelsToGo, offered}` (S6), `autoStart{available:false}` (stub until S8; the per-quest `auto` flag is read from the plan store), `offers[]` (quest, chunk, title, act, risk, `readiness` word, `reasons[]`, `legacyFloor`, `sentence`, `startable`, `why`, `auto`; `pays` is `[{kind:"ether", amount, key}]`, omitted without a wallet, when `loot.vault` is 0 or when nothing is left to earn, S10), `quests[]` (every quest of the difficulty with its chunks and states), `running`, `awaiting`, `blocked`, `handIn` (running, done, confirming, blocked or failed), `history[]` (newest 20 chunks the board ran), `needs[]`, `strip`, `extra`. Words only: no numeric margin and no chapter counter anywhere (V `TestDeskViewJSONShape`).

Glance (`/api/glance`): additive `quest{quest,title,chunk,part,state,step,npc}` for the video-card strip and Needs-you rows `quest_handin` ("Akara is waiting: hand in Den of Evil", button Hand in -> the handin route) and `quest_blocked` (Try again, Stop). The glance reads the same cached view.

Decisions and gaps (I unless tagged):
- The board is built per hero by the runner (the name is only known from the first snapshot): `SetQuestBoardMaker` opens `<hero>.quests.json`; a hero switch rebuilds it. The plan files carry the hero name, so `internal/charsel/perchar.json` does NOT list them (parking would hide the other hero's plan); V `TestQuestPlanFilesAreNotParkedBySwitch`.
- Readiness: `Assess` replaces the legacy word where the margin query has data (boss from the danger table, hero readable). Where `Assess` answers Unknown the legacy word stands (a quest without a floor stays safe) so no quest vanishes from the board because of missing data. The boss margin is divided by the ledger's hero correction only (a boss has no zone key; U). Queries are throttled to once per 30 s per level and difficulty. The potion count is `unread` -> no "short of potions" reason.
- "bot paused" is the player's pause (`PauseRunning` is the only running state), including the walk-home phase.
- A POST needs a body (`{}`); an empty body is a 400 like the other control routes.
- Auto-start flags and modes are built (S6, S8); the Ether award is built in S10 (section Ether below).
- Not done: the strip's step names are the board's descriptive names, not the handler's (S3 note); restart persistence of `accepted` is V offline (`TestDeskModeAndRestartPersistence`), not live.

### Reason codes and `changed` (card bsm6, `quests.boardSM`; design appendix B.1 and B.2)

- Every chunk row of `quests[].chunks[]` carries `reason: {code, text, remedy}` while the part is live and does not move (`internal/questboard/desk_reason.go`): a queued part `waits_for`, the head under a guard hold its hold code, an awaiting part `handin_wanted` (`handin_in_town` while the dialogue is in flight, an approximation: it also covers the talk itself, S; `handin_waiting` with the holder in words while the tap is queued and the bot loop has not picked it up: out in the field, recovery, another step; the same reason is `handIn.why` of the view; akarafix), a parked part its stored park code, a done part still unconfirmed `confirming`. The head that moves has no reason. `running`, `awaiting` and `blocked` carry the same object. The sentence comes from the one table (`sm_reason.go`); a placeholder the producer cannot fill is replaced by the producer's own words.
- A queued part shows the wire state `accepted` (not `locked`) while it has a run; the page and the scenarios read `accepted` + `waits_for`.
- `handin_ready` (row reason, no run): the game already counts the field work done (pending bit exact, no record). Start creates the run (it no longer answers "already started"); with the knob off nothing changes.
- Replies of start, cancel and retry carry `changed: [{quest, chunk, state, reason}]`: every run the verb touched (the chain a Start accepted, the dependants a Stop closed), the state and reason read back after the verb. A repeated verb has `already: true` and no `changed`. V offline by `TestDeskChangedAndReasons`, `TestDeskHandInReady` (Paladin and Sorceress).

### Live check list for the API (NOT run; needs `quests.board` = 1 and a hidden instance, `tools/live.sh`)

Replace `$H` with the watch address, `$T` with the answer token (never paste it into a log); POST bodies are JSON.

1. `curl -s $H/api/quests | jq '{enabled,reason,mode,offers:[.offers[]|{quest,readiness,reasons,legacyFloor,startable}]}'` with the knob 0: `enabled:false`, reason names the knob. Set the knob to 1: offers appear; Den of Evil is `safe`; no number for a margin anywhere.
2. `curl -s -X POST -H "Authorization: Bearer $T" -H 'Content-Type: application/json' -d '{}' $H/api/quests/den_of_evil/start`: `ok:true`, `state:accepted`. Repeat: `ok:true, already:true`. Start Blood Raven meanwhile: 409 `busy`, the reason names the Den.
3. Poll `GET /api/quests` every 5 s: `running.step` moves, `strip.npc` is Akara, glance `quest` matches. Press Pause in the UI: `paused:true`; a Start then answers `paused`.
4. When `awaiting` appears (and the Needs row `quest_handin`): POST `handin`: 202 `pending:true`; poll until `handIn.state` is `done` or `confirming`; a second POST answers `ok:true, already:true`; the dialogue ran once (log `talk Akara/reward` once).
5. `POST /api/quests/mode {"mode":"farm"}` then restart the bot: GET shows `mode:farm`. Wrong token: 401; foreign `Origin`: 403.
6. Restart the bot while a chunk runs and while awaiting: `running`/`awaiting` come back from the plan file and the evidence, the chunk is not redone. Writes after start must be 0.

## Audit and the stopped note (card questaudit)

Every action that reaches the quest POST routes (start, handin, cancel, retry, auto, mode, dismiss) writes one INFO log line
`quests: action=.. quest=.. chunk=.. path=.. remote=.. agent=.. confirm=.. ok=.. already=.. status=.. code=.. reason=..` and, when
the desk ran it, one JSON line to `<state dir>/quests/<hero>.audit.jsonl`, a ring of the last 500 lines (V: tests
`TestAuditRingIsBounded`, `TestCancelLeavesAuditRecordWithSource`). The source is a label only: `remote` is loopback, lan or other
(never an address; behind a proxy on the same host every client reads loopback), `agent` is the user-agent family (firefox, chrome,
curl, ...), `confirm` is the `{"confirm": true}` flag the panels send with the second tap of Stop (recorded, never required). Quest
ids and chunks are reduced to `[a-z0-9_-]`. No header, token or body is logged. A call the gate refuses is logged but not put in
the ring. The file name is not on the support-bundle allowlist (`internal/support.Allowed`), so no bundle carries it; to show
"last actions" in a bundle add a redacted `quest-audit.json` there on purpose.

`GET /api/quests` carries `stopped` (quest, questTitle, chunk, at) for a cancel that really stopped a chunk in the last 24 hours,
unless the quest runs again or is done; the Quests sheet shows "Stopped by you at HH:MM" (the viewer's local time).

## Watch UI (slice S5)

> Built offline (card quests5), NOT verified live. Tags: node tests and the fit run are V (they ran); the look on a real hero and a real phone is U. Everything below is dormant while `quests.board` = 0: the quest tile hides itself, the Journey tile stays and the sheet says how to turn the board on.

| piece | where |
|---|---|
| `quest-strip` tile (strip slot, after `journey-strip`) | `internal/watch/web/panels/quest-strip`: one line of state in words, one primary button, "All quests", the mode switch (Train, Idle, Farm) |
| `quest-board` sheet | the same-named drawer panel (menu entry "Quests", or "All quests" on the tile); registered with `overlay.js` like the other drawers: one overlay at a time, Escape closes it |
| `quest-list` | the phone's Quests tab (Orders, Map, Quests, Progress), the same list without a drawer; its CSS is the sheet's with the id swapped (`TestQuestListCSSIsTheBoardCSS`) |
| shared logic | `quest-board/panel.js`: `stripModel` (state to words and button), `boardModel` (acts, parts, groups), `settle` (the payoff), `act` (the one place a button reaches the API), `captionOf` |
| video card line | `quest-strip` publishes `questCaption`, `video` shows it top-left of the picture (never over the HUD); it has no endpoint of its own (the v1 manifest stays) |
| Needs you | quest rows ride the existing glance; `sev: "event"` has an accent (not an alarm) and a refusal shows `reason`, not "could not do that" |
| Journey | glance gets `questBoard: true` while the board is on (`GlanceSources.QuestOn`, wired in `cmd/d2bot/questwire.go`); `journey-strip` then hides. The old tile, chip and panel stay until S9 |

States of the tile (words, from GET /api/quests only): offer ("Offer: Den of Evil, safe" [Start]), training (no button), running ("Running: Den of Evil", the step in words, [Stop] with a second press to confirm), waiting at the NPC ("Akara is waiting: hand in Den of Evil" [Hand in]), handing in ("Handing in Den of Evil..." as soon as the button is pressed, no button), confirming ("Checking that the game agrees"), failed (the API's reason, [Try again]), stuck ([Try again] [Stop]), idle (the mode line). A Start tap covers the current chunk only. Hand in: the page decides success from the GET (`handIn.state` done, or the chunk `done`), then shows the payoff toast once ("Done: Den of Evil", plus "+N meta" only when the API sends `pays`) and the tile shows the next offer. A refused POST shows the reply's `reason` (or its code when the reason is empty), never a generic line. No margin number and no chapter counter anywhere: the fit run fails a page that has one (`quest words`).

Stubs: the per-quest auto-start cell is a live switch since S8 (below); the mode switch is stored but acts in S6 (tooltip says so); `pays` is absent until S10.

Gaps (I): the quest-board sheet is the 440 px drawer, not a rail-width sheet. The S5 gaps "step 2 of 4", "levels to go", the offer Needs-you row and the 360x640 Needs row overflow are closed in slice S8 (below).

Checks: `go test ./internal/watch/ -run Quest` (node: `testdata/questui/quest_ui.mjs`, fake API fed by the S4 goldens, both builds); `go run ./tools/uilint`; fit pages `quest-*` and `quest-sheet-*` in `tools/uishots.sh --fit` (all eight sizes plus 412x780, 360x640 for sheets), must-fail `questdefect-clip|small|words|two` (`tools/uishots/fixtures_quests.go`: the real Desk over the Fake world, variants by the `quests=` query), pictures with `tools/uishots.sh --pics --only 'quest-*'`.

## Modes (slice S6)

> Built and tested offline (card quests6), behind `quests.board` (default 0: nothing here runs, V `TestBoardModeKnobOffIsTheOldPlanner`). NOT run live; the checklist is at the end of this section. No new knob. Tags: tests V, live behaviour U.

Between chunks the hero is in the mode the player picked (`POST /api/quests/mode`, stored per hero, default `train`). Code: `internal/bot/boardmode.go` (the loop), `internal/questboard/mode.go` (target and meter, pure).

| mode | what the loop does | notes |
|---|---|---|
| train | the leveling slice (one `StepLevel(clvl+1)` per tick, so the zone is the plan table's pick for the level, or the zone model's pick when `bot.zoneModel` is 1) with the pinned target: `NextTarget` = the lowest-order open story quest (then optional, then unlock) whose chunk is Available or Ready | the pin names the goal in the log and the meter; it never starts the quest. It does not bias the zone model (U: the model has no target term yet, so with `bot.zoneModel` 1 the model's pick stands and the meter is the readiness line) |
| idle | `rest`: home when in the field, then the normal town chores; no leveling slice, no farm loop | parked on purpose: `Runner.parkedBoard` keeps the state watchdog (`stuckCheck`), the campaign stall ladder (`campaignStall`) and the kill drought (`droughtStage`) quiet, the same idea as Pause. Cleared at the start of every other board tick |
| farm | the leveling slice with the farm-gold area choice (`goldFarmTarget`, `Plan.FarmRung`: one rung below the zone the level is ready for, the same safe list, bans, guards and gear gate as the player's Farm gold order, so never above the readiness line) | `Runner.boardFarming` is true only inside a farm slice, so a quest step that levels is not bent. `bot.farmReset` keeps its own rules in Farm (only the player's gold order suppresses it). Loot rules are unchanged |

Rules (V by `internal/bot/boardmode_test.go`, two classes: Sorceress, Paladin):
- A mode change is adopted at the next tick where it is safe (`safeToPause`: in town, or nothing hostile in the clear radius; V `TestBoardModeChangeWaitsForASafeMoment`). A running field slice ends at its own bound first (U for the live length of a slice); the board shows `meter.applied` next to `meter.mode` until then.
- An accepted chunk (running, awaiting the hand-in) comes first whatever the mode; after the hand-in the hero returns to it (V `TestBoardModeChunkPrecedenceThenBack`). A parked or failed chunk falls back to the mode.
- Player orders, the Farm gold order, recovery and the gear gate sit above `boardStep` in `goal.go` and win over the mode. Pause and Reset stop the loop before it; the mode is stored and survives them.
- Switching mid-run: V `TestBoardModeSwitchMidRun` (train to idle: no more field work; idle to farm: leaves again).
- The readiness meter (`meter.line`, words only, no margin number): "Training toward Blood Raven: 2 levels to go." (levels by the legacy clvl floor of the class family), "Training. Blood Raven is ready; tap Start when you want it.", "Training toward X: level is not the gap, the margin still says too hard ...", "Waiting in town. ...", "Farming loot and gold ...", "On a quest: X. After it the hero returns to idle mode." (V `TestNextTargetAndMeterTwoBuilds`, `TestMeterWordsEveryMode`, `TestQuestMeterWords`).

Live checklist (NOT run; hidden instance, `quests.board` 1, a throwaway hero at clvl 3 to 6):
1. Train, 30 min: the log has `quest board: Training toward ...`; the hero levels in the plan's zone; `GET /api/quests` `meter.line` falls from "N levels to go" to "ready; tap Start" as the clvl passes the floor; nothing starts by itself.
2. Idle, 30 min: set the mode with the hero in the field; it finishes the fight, goes home, does the chores and stands in town; no area change, no `kill drought`, `no progress` or `campaign stalled` line in the log; `meter.applied` reads idle.
3. Farm, 30 min: the log has `goldfarm: farming for the quest board farm mode`; the zone is one rung below the level's pick; gold and drops rise; `farm reset:` lines when a zone passes `bot.farmResetPct`.
4. Switch train, idle, farm and back while a fight is on; tap Start in idle (the chunk runs, the hero waits at the NPC, Hand in, back to parked).
5. `d2cheat ledger` shows no writes (the modes only read).

## Chunked quests (slice S7)

Built and tested offline (card quests7), NOT verified live: no chunk of Acts 2 to 5 has run on a hero; every row of act3.json to act5.json is tagged U until a drill. Acts 3 to 5 are new data; the Act 5 rows are stubs over handlers that exist in `internal/task/act5.go` (nothing ran).

How a chunk maps to the task layer (`data/questboard/*.json`, `internal/questboard/def.go`, `internal/bot/questboard.go`):

| field | meaning |
|---|---|
| `handler` | the task group the chunk runs; default the quest's `taskId`, else its id. A piece with a group of its own (`horadric_cube`, `khalims_eye`) names it |
| `nodes` | the step suffixes (the part after `quest.<group>/`) a chunk of a multi-chunk quest runs in the field; a chunk that owns its group runs all steps but the hand-in |
| `handInStep` | the hand-in step of a chunk that ends in a hand-in (`reward` by default; `read`, `warriv`, `alkor`) |
| `unsupported` | why no handler can run it; Derive shows `Available` with `not supported yet: <reason>`, never `Ready`, so there is no Start (V `TestStaffPiecesAndUnsupportedAssembly`) |

Rules (V by the named tests):
- Any order: a chunk with no `after` is offered at once, several in parallel, and the executor accepts whichever the player taps (V `TestKhalimPiecesInEveryOrder`, 6 orders x 2 builds). An assembling chunk lists the pieces in `after`: it needs them all.
- One Start covers one chunk; the next chunk raises its own offer as soon as the previous one reads Done (V the same test). Hand-in stays the player's tap.
- Evidence of a part (bot, `nodesDone`): its field steps' own done checks (items in the pack, areas), its hand-in step when it has one, and every chunk it comes after done the same way. A hand-in step reads done before its item exists, so a part never counts before its predecessors (V `TestCainChunkEvidenceFollowsTheChain`). This is T3. A quest prerequisite unlocks only on T1/T2 (`blackened_temple` for the Will: V `TestChunkEvidenceConflictsAndDifficulty`); the quest word on T1 marks every part done; chunk order accepts any tier.
- Nothing is persisted but the player's records: a restart between or inside pieces re-derives (V `TestKhalimRestartBetweenPieces`). A new difficulty starts the quest again (the same test).
- Readiness: a boss chunk reads the margin of its `boss` (legacy `legacyFloor` where the survival table has no such boss: `legacy_only`); a fetch piece has no boss and takes the quest's legacy floor only: a per-zone margin word for the fetch areas is not wired yet (U). Floors for Acts 2 to 5 are the task layer's clvl gates (U), caster +2 and summoner +3.
- Validation (V `TestWholeBoardGraphIsAcyclic`, `TestEveryChunkIsWellFormed`, bot `TestEveryChunkHasAHandlerOrIsUnsupported`): the graph of quests, chunk `requires` and chunk `after` is acyclic; every chunk has steps and a handler whose steps exist, or an unsupported reason; every hand-in NPC is in `talk.ByAct` (the giver table: Cain is `Deckard Cain`, which fixed the old `Cain` of the Act 2 rows); no supported chunk waits behind an unsupported one in its own quest.

### Chunk tables (multi-part and explicit-handler quests; generated from the data files)

| act | quest | chunk | ends | order | handler |
|---|---|---|---|---|---|
| 1 | search_for_cain | scroll | item | any | search_for_cain: wp-dark-wood to-wood tree |
| 1 | search_for_cain | read | handin | after scroll | search_for_cain (hand-in read) |
| 1 | search_for_cain | stones | auto | after read | search_for_cain: to-stony stones to-tristram |
| 1 | search_for_cain | free_cain | auto | after stones | search_for_cain: cage home (CAIN-5b, supported; live drill open) |
| 1 | search_for_cain | report | handin | after free_cain | search_for_cain_board (hand-in report: Akara; supported) |
| 1 | act1_done | sail_east | handin | any | act1_complete (hand-in warriv) |
| 2 | tainted_sun | amulet | item | any | viper_amulet |
| 2 | horadric_staff | cube | item | any | horadric_cube |
| 2 | horadric_staff | shaft | item | any | staff_of_kings |
| 2 | horadric_staff | assemble | handin | after cube, shaft; needs quest tainted_sun | horadric_staff_board: assemble (hand-in cain) |
| 2 | seven_tombs | duriel | auto | any | duriel |
| 2 | act2_done | sail_east | handin | any | act2_hop (hand-in meshif) |
| 3 | golden_bird | bird | handin | any | golden_bird_board: meshif (hand-in alkor) |
| 3 | khalims_will | eye | item | any | khalims_eye |
| 3 | khalims_will | brain | item | any | khalims_brain |
| 3 | khalims_will | heart | item | any | khalims_heart |
| 3 | khalims_will | assemble | item | after eye, brain, heart; needs quest blackened_temple | khalims_will: assemble |
| 3 | khalims_will | orb | auto | after assemble | khalims_will: wp to orb |
| 3 | act3_done | portal | auto | any | act3_hop |
| 4 | hellforge | hammer | item | any | hellforge: wp to kill hammer |
| 4 | hellforge | smash | auto | after hammer | hellforge: place home |
| 4 | terrors_end | seals | auto | any | UNSUPPORTED |
| 4 | terrors_end | diablo | handin | any | terrors_end: wp to kill home (hand-in reward) |
| 4 | act4_done | portal | handin | any | act4_hop (hand-in tyrael) |
| 5 | prison_of_ice | find | auto | any | prison_of_ice: wp to touch home1 |
| 5 | prison_of_ice | potion | handin | after find | prison_of_ice (hand-in potion) |
| 5 | prison_of_ice | thaw | auto | after potion | prison_of_ice: to2 thaw home2 |
| 5 | prison_of_ice | scroll | handin | after thaw | prison_of_ice (hand-in scroll) |
| 5 | prison_of_ice | read | auto | after scroll | prison_of_ice: read |

### Gaps (each is `unsupported` or documented, none is a dead button)

| gap | why | what closes it |
|---|---|---|
| Cain: report to Akara | SUPPORTED in S7b: group `search_for_cain_board`, one turn-in step `report` (Akara, pick `cain report`, the plain Talk, U). It is needed only on the reader's word Cleared (Cain free, reward open); T3 never starts it. `cainNodes` now treats Cleared as free (before: Done), so the cage is not clicked again while the reward is open (U: whether the game sets Cleared at the cage or Done at once; Done at once skips the step with the same chunk result) | the drill |
| Staff: assemble and give to Cain | SUPPORTED in S7b: group `horadric_staff_board` = the Cube step plus a `cain` turn-in (Deckard Cain, pick `horadric staff`, U), needed with the Staff in the pack and the quest word not Done. The campaign group `horadric_staff` is unchanged. NOT built: a chunk that places the Staff in the Canyon (`place`, to-canyon steps stay in the campaign group only); Cain's talk does not set the done bit (V lab 2026-11-27: 0x0800 to 0x0c78); only the placement does (S d2d), so `seven_tombs` opens on `requiresCleared: horadric_staff` (the Staff made: record 10 bit 11, which survives the join, or hst in the pack), not on `requires` (placefix, placefix2) | the drill; a `place` chunk if the word needs the placement |
| Golden Bird | SUPPORTED in S7b as a hand-in only: group `golden_bird_board`, steps `meshif` (Jade Figurine -> Golden Bird) then `alkor`. The figurine is a random drop (U) and is never fetched: with neither item in the pack the chunk is listed with the reason "no Jade Figurine in the pack"; both steps are done only on the quest word (T1/T2) or, for Meshif, the bird in the pack | the drill; the dropper if it is ever known |
| Terror's End: seals | stays UNSUPPORTED (sharper reason in `act4.json`): seal progress has no durable evidence (no quest word, no save field; the seals and seal bosses are per game, U), so a seals-only chunk cannot be re-derived after a restart; `boss.OpenSeals` resumes only inside one game and ends by waking Diablo. The `diablo` chunk runs the seals itself | spike plan below |
| Boss ids | `council`, `mephisto`, `izual`, `hephasto`, `diablo`, `shenk`, `nihlathak`, `ancients`, `baal` go to the survival table's `FindBoss`; an id it does not know gives the legacy word only (U) | table rows |

### Terror's End seals: spike plan (NOT built, needs a lab hero in Act 4)

1. Open the five seals on a lab hero (CHEATED/IMPOSSIBLE tag), leave the game with the seals open, re-enter and read the seal objects' state with `mem` (a read only): do they stay open (U)? Do the seal bosses stay dead (U)? Does Diablo spawn only on the last seal?
2. If they reset, a seals-only chunk is impossible by design (no evidence): keep the single `diablo` chunk. If they persist, the seal states are T3 evidence read from the objects in the Chaos Sanctuary: add `boss.OpenSeals` with a `StopBeforeDiablo` option (it now waits for Diablo as its last step), a task step `seals` in `diablo` terms and a `nodes` list in the chunk.
3. Record the answer in `docs/act4.md`; build only after 1 shows persistence.

### Live drill list per chain (checklist, NOT run; every hero below is lab-created, tag CHEATED/IMPOSSIBLE in `docs/probes/test-characters.md` and the cheat ledger, so none of it is evidence about game behaviour, only about the board)

1. Act 1 Cain: hero at Normal clvl 12 in town, `quests.board` 1. Start `scroll` (Dark Wood tree), see it stop, Start `read` (Akara, hand in tap), Start `stones` (stones and the portal), then `free_cain` (the guards at the arrival are fought, then the cage; hazard: never auto; CAIN-5b, supported, live drill open), then Start `report` (the Akara hand-in: what does the quest word read after the cage, Cleared or Done? the word decides whether `report` is skipped). Kill the process after `scroll` and after `read`: the same offer returns, nothing is redone.
2. Act 2 Staff pieces: lab hero in Act 2 with Radament done. Start `cube` then `shaft`, then the other order on a second hero; check the offers while one runs (not startable) and after (the other offered); `assemble` (Cube step, then the Cain hand-in tap): does the quest word of `horadric_staff` change after Cain's talk, and does `seven_tombs` unlock (U)?
2b. Act 3 Golden Bird: a hero with a Jade Figurine in the pack (a lab-granted item, CHEATED): `bird` lists the missing item without it; Start with it makes the Golden Bird at Meshif, the tap hands it to Alkor; kill the process after Meshif and check the offer comes back.
3. Act 2 Tainted Sun, Arcane, Summoner, Duriel: one chunk each, boss margin words checked against the survival log; Duriel never auto.
4. Act 3 Will: hero with the Council done (Flail in the pack). Start Eye, Brain, Heart in two different orders on two heroes; Start `assemble` (cube in town), then `orb`; check each quest item is kept by the loot rules between pieces.
5. Act 3 Blackened Temple, Blade, Lam Esen, Mephisto and the portal: one chunk each; the portal chunk fights Mephisto again if the game was restarted (U).
6. Act 4 Izual, Hellforge (`hammer` then `smash`, Soulstone in the pack), Terror's End (`diablo` runs the seals inside; note whether a half-done seal run resets after a restart: U, record it), hop.
7. Act 5: one chunk each; Prison of Ice in five parts with a restart after each; Rescue may fail (record the word).
Every record carries `writes after start: N` (`d2cheat ledger --since-run <run>`); N > 0 is CHEATED and does not count.

## Auto-start and away (slice S8)

> Built and tested offline (card quests8, TASK-601), behind `quests.board` (default 0: nothing here runs, V `TestAutoKnobOffChangesNothing`, `TestQuestAutoNeedsTheOptInAndTheKnob`). NOT run live; the checklist is at the end of this section. Tags: tests V, live behaviour U. User decisions of 2026-11-16 (design section 2, memory d2-quest-design-2026-11-16).

Rules (code `internal/questboard/auto.go`, bot side `internal/bot/questboard.go` `boardStep` and `questauto.go`):

- **Opt-in per quest and difficulty, default off.** `POST /api/quests/<id>/auto {"on": true|false}` (real route; the page posts exactly this). A quest whose data says `autoStart: never` (the hazard quests) refuses with code `auto_never` and the words "always needs you". The flag is a plan record (`autoStart`), so it survives a restart (V `TestPlanRoundTripsAutoFields`, `TestAutoRestartPersistence`).
- **Starts only on a SAFE offer.** The readiness word must be exactly `safe`: never `risky`, never `too hard`, never unknown (V `TestAutoNeverOnRiskyOrTooHard`, `TestAutoNeverOnUnknownReadiness`, on a Paladin and a Sorceress). A risky opted-in offer waits for the player's tap and `autoStart.why` says so. An "also when risky" setting is NOT built.
- **Calm state.** The tick starts nothing while the bot is paused, a fight is on (`safeToPause` false) or the mode is `idle` (the player parked the hero on purpose). The mode `train` and `farm` allow it. `NoteHold` (the player's pause, resume or reset; `Runner.PlayerPauseMode`, `Runner.PlayerReset`) blocks starts for `quests.autoHoldMin` minutes.
- **Countdown.** An eligible offer first counts down `quests.autoDelaySec` seconds (default 20; 0 starts at once). The page shows it ("Den of Evil starts by itself in 12 seconds", `autoStart.countdown`, a Needs-you row `quest_auto` and the strip tile with Cancel). `POST /api/quests/<id>/cancel` on a quest that counts down cancels the countdown: that offer does not auto-start again until it has left the offer list or the process restarts; the opt-in flag stays (I: "clears auto for that offer once"). A countdown is dropped when anything makes the offer ineligible.
- **One chunk at a time.** A start covers the current chunk only (`Record.Auto` marks it). The next chunk raises its own offer, which auto-starts only if its quest is still opted in and the offer is still safe (same rules, same countdown).
- **Limits, counted from the plan's event log so a restart cannot reset them (V `TestAutoRateLimits`).** At least `quests.autoGapMin` minutes between two auto-started chunks (default 30) and at most `quests.autoDayCap` in any 24 hours (default 6; 0 turns auto-start off). A bug can therefore start at most that many chunks a day.
- **Away.** The player is away when no watch client asked for the board for `quests.awayMin` minutes (default 10): every `GET /api/quests` and every glance request that reads the board counts as presence (`Exec.Touch`, called by `Desk.View`). With no client since the process started, away counts from the start. U: a non-human poller of the glance (a dashboard on a wall) keeps the player "present".
- **Hand-in.** The player's tap while present. The only case the board hands in by itself: the chunk was started by the board (`Record.Auto`), it waits at the NPC, and the player is away, the bot calm and not paused. It runs `Exec.HandIn` once per chunk and process (the dialogue's own three attempts, then the chunk parks with a Needs-you row as before); the payoff is the same (awards by the same keys). A chunk the player started is never handed in by the board (V `TestAutoNeverHandsInAHandStart`). If the player comes back first, the tap is theirs (V `TestAutoHandInOnlyWhenAwayAndOnlyForAutoStarts`).
- **While you were away.** Every controller event is logged in the plan file (`autoLog`, 100 entries; kinds `autoStart`, `autoHandIn`, `autoHandInFailed`, with `away`). `GET /api/quests` has `whileAway.events[]` (text in words, `at`): the events with `away` since the player last dismissed. The page shows a card on the board sheet and the Quests tab ("While you were away: 14:02 Started Den of Evil by itself, 14:20 Handed in Den of Evil", button Got it) and a Needs-you row `quest_away`; `POST /api/quests/away/dismiss` clears both (V `TestDeskAwayCardAndDismiss`).

Knobs (data, `docs/knobs.md`): `quests.autoDelaySec` (20), `quests.awayMin` (10), `quests.autoHoldMin` (15), `quests.autoGapMin` (30), `quests.autoDayCap` (6). Nothing about a class, build or skill is in the controller; the safe word comes from the readiness model and the per-quest `autoStart` from the quest data.

API additions (all additive): `autoStart` {`available`, `note`, `delaySec`, `why`, `countdown`}, quest rows and offers `autoOk`, offers `levelsToGo`, running and strip `stepNo` / `stepCount` / `steps[]` (`name`, `state` done, now or todo) and `auto`, `whileAway`, need rows `quest_offer`, `quest_auto`, `quest_away` (need rows carry `sub`, the readiness word as the chip).

S5 gaps closed here: the tile says "step 2 of 4" (from the chunk's step list); `levelsToGo` (the legacy clvl floor minus the level, offers that are not safe; `meter.levelsToGo` for the training target) is rendered in words only ("two levels to go"; the fit run still fails a digit-count page); an offer raises the Needs-you row "Den of Evil is ready: Start" with its readiness chip while it stands (it goes when the chunk starts, the offer changes or the bot pauses; at most one row: the first safe offer); the 360x640 Needs row overflow is fixed (below).

Checks: `go test -p 1 ./internal/questboard/ ./internal/bot/ -run 'Auto|Quest'`, `go test ./internal/watch/ -run Quest` (node), fit pages `quest-countdown*`, `quest-away*`, `quest-sheet-countdown`, `quest-sheet-away*`, `quest-phone-tab-away`, and every `quest-*` page now also at 360x640.

Live checklist (NOT run; hidden instance, `quests.board` 1, throwaway hero at clvl 6 to 8, no cheat writes; `writes after start: 0` from `d2cheat ledger --since-run <run>`):
1. Opt in Den of Evil (switch on), keep the page open: a countdown of 20 s shows (tile, Needs row); Cancel stops it and nothing starts for it; toggle the page away and back, it counts down again; let it run: the chunk starts, the hero works it and stops at Akara; with the page open Hand in waits for the tap; tap it.
2. Opt in again, close every page (and the phone) for 12 minutes: the next offer counts down, starts, the hero stops at the NPC, hands in by itself ("autoHandIn" in `quests/<hero>.quests.json`), reward claimed once, XP and the Den skill point as by hand.
3. Reopen the page: the "While you were away" card lists both events; Got it clears it and the Needs row.
4. A risky opt-in quest (Blood Raven one or two levels under the floor): nothing starts and the board says it waits for you. Pause the bot, resume: no start for 15 minutes. Restart the bot during a run: the chunk resumes, the flag is still on, the day cap is not reset.
5. `autoDayCap` 1: a second start the same day is refused with the reason in `autoStart.why`.

## Ether (slice S10, built offline 2026-11-17, card quests10)

User decisions 2026-11-16: Ether is earned ONCE per account per quest per difficulty; no backfill; a hand-played quest pays nothing; skipping never awards; an auto hand-in while away pays at the same payoff.

- Data: the `meta` column of `data/questboard/act*.json` (per quest and per chunk, per difficulty; the word `meta` is the column name, the currency is Ether). All values are U placeholders from the user's first guess: story and boss quests 5 / 8 / 12 (normal / nm / hell), optional chores 2 / 3 / 5, act 3 and 4 optionals 3 / 5 / 8, chunk columns 1 / 2 / 3 and boss chunks 2 / 3 / 5. Nightmare is about 1.6x and Hell about 2.4x of Normal. Tuning is data only.
- Key: `quest/<quest>/<chunk or ->/<difficulty>`, account-wide (never the hero); it is the wallet's own key for source `quest` (`internal/ether`, `Earn("quest", "<quest>/<chunk or ->", difficulty, amount)`), so the wallet refuses a second booking.
- Where: `Exec.HandIn` (after the read-back confirms done, so the tap and the auto hand-in both pay), plus `Exec.Settle` (called when the desk builds a view and when a repeated hand-in finds the chunk done). The award is derived from T1/T2 evidence (`Award`, state.go) every time; a restart or crash between the evidence read and the ledger write only means the next settle pays, never twice (V `TestEtherCrashBetweenEvidenceAndLedger`). T3-only completion (`confirming`) pays nothing until T1/T2 confirm it.
- No backfill: the first time the board derives a difficulty for a hero it stores a baseline in the plan file (`based`, `baseline`): every award key that already reads Done. A baseline key never pays, so existing saves and heroes upgraded from before S10 earn nothing for old quests. Quests the board never started (`touched`) pay nothing either, and a skipped quest is never accepted.
- Wiring: `questboard.Options.Ledger` (a function, nil result = no wallet = inert), `Runner.SetQuestLedger`, `cmd/d2bot/questwire.go` (`questLedger`, over `etherFor(...).Wallet()`, which is nil while `loot.vault` is 0). `ether.Wallet.Paid(key)` reads what a key paid.
- API: offers carry `pays: [{kind:"ether", amount, key}]` (what is still unpaid; the quest column counts on its last chunk, U); `handIn.awarded`, the `handin` reply `awarded` (the sync repeat path) and `already: true` when nothing new was booked; `history[].paid`, `quests[].paid`, `whileAway.events[].awarded`. All omitted without a wallet.
- UI: the payoff toast "Done: <quest>   +N Ether", a "pays N Ether" chip on the offer line, "+N Ether" under a done quest row and in Recent, the away card text. The wallet chip is unchanged. Fit pages `quest-ether-*`, `quest-sheet-ether*`.
- Tests: `internal/questboard/ledger_test.go` (two builds, fake ledger: once, repeat, crash, no backfill, hand-played, skip, difficulty, no wallet, two heroes one account, desk), `internal/watch/questsapi_test.go` `TestQuestsRouteEtherGoldens`, `quest_ui.mjs`, `cmd/d2bot` `TestQuestLedgerOverWallet` (real wallet).
- NOT verified live (U): the live drill (one real hand-in awards once; a restart does not repeat it) is for the next live pass.

## Journey retired (slice S9, card retirej, TASK-629)

> Built and tested offline, NOT verified live. User decision 2026-11-17. Tags: tests V, live behaviour U.

| what | now |
|---|---|
| knob `quests.board` | default 1. With 1 only an accepted chunk runs a quest; between chunks the hero acts on its mode (train, idle, farm). 0 stays as the rollback and for measured runs: the old floor planner plays the quests, with NO Journey (no chapters, no questions, no Go now). |
| removed UI | panels `journey`, `journey-strip`, `journey-chip` (files, cockpit slots, phone tab, status band); the free-grid presets Standard, Simple, Stream and Debug lost the Journey cell and the cells below moved up; frozen Classic keeps its `journey` id, which `shell.json` lists as reserved (drawn as nothing). Glance has no `journey` block. The onboarding step id `journey` stays (old ticks stay valid) but reads "Open the Quests tile". |
| removed code | `internal/campaign` controller, view and the decision function; `internal/task` guide; `internal/bot` guided.go (floor question, ask gap); `internal/watch` campaign.go (`GET /api/campaign`, `POST /api/campaign/intent`, the chapter-done Chronicle line and the `chapter:<id>` unlock event; no unlock data used it); the "journey" command intent; the Go now and Hold actions of the blocked row. |
| kept | `internal/campaign` spine, bands and state reader (the level plan reads the spine; the import reads the state), `data/campaign/normal.json`, the decision point `t1.journey.floor` in the catalogue (nothing asks it any more), the knobs `campaign.journey*` (RETIRED, no effect; the keys stay so saved overrides load). |
| charter | the "Pay and go" charter lived only on the Journey sheet and is gone with it. The zone-order charter (an order above the hero's range, `internal/currency/tallyrows.go`, `watch/tallystrip.go`, `proposals.go`) is untouched, as decided 2026-11-16 (design question 5). |
| quests the bot cannot do | a chunk with `unsupported` shows "not supported yet: <reason>", cannot be started, and the hero trains (unchanged by S9, V `TestEveryChunkHasAHandlerOrIsUnsupported`). |

### Old heroes (state migration)

On the first load of a hero's plan (`cmd/d2bot/questwire.go` `importJourney`, `questboard.Store.ImportLegacy`):
1. `state/campaign.json` is never written or deleted. A copy `campaign.json.pre-board` is made next to it once (kept if present). A build from before S9 finds its file as it was.
2. Chapters the player set to Skip (Normal) become whole-quest skips in `state/quests/<hero>.quests.json` (task ids such as `sisters_burial` map to the board's ids by `taskId`; a chapter with no board quest, like the Rogue claim, is ignored).
3. Mode `orders` becomes the board mode `farm` when the hero has no mode yet. Free and Guided need nothing (train is the default).
4. Not imported: done chapters (the board derives done from the game's own quest bits and the save, so a hero mid-Journey loses nothing and is offered its next quest), Hold, Go now, asked questions, the farm clock.
5. The plan carries `legacy: "campaign/1"` so it runs once. An unreadable or foreign-schema file is logged ("journey import skipped") and the hero starts clean. Fixtures: `cmd/d2bot/testdata/journey/` (a Paladin mid-Journey, a Sorceress in orders mode); tests `TestImportJourney*`, `TestImportLegacyOnce`.

### Live check list (NOT run; hidden instance, a hero with an old `state/campaign.json`)
1. Start the build: the log has "journey import: decisions taken into the quest plan" once; `campaign.json` and `campaign.json.pre-board` are byte-identical.
2. The page shows the quest tile and no Journey tile or chip at phone, tablet and laptop widths; Standard and Simple have no empty cell.
3. A hero that skipped the Countess shows her as skipped; Blood Raven's offer matches the hero's real quest state.
4. `GET /api/campaign` answers 404.


## One button per row (card handinbtn, 2026-11-21)

> User bug: the Sisters' Burial Grounds row read NOW / "waiting for you" with the hint "Start goes straight to the hand-in" and had no button.

- Cause (V, `docs/probes/mercbr-2026-11-21.md` plus the code): a part that reads `awaiting_handin` with no accepted record (case C5, the game counts the field work done) has no run, so the view has no `awaiting` object and the Now tile has nothing to draw; the phone's Quests tab has no Now tile at all, and the part line only drew Start for `ready`.
- Rule: the server puts the one button of a part in `quests[].chunks[].action` (`{verb, label, disabled?, confirm?, why?}`, `internal/questboard/desk_action.go`): Start (ready and startable; awaiting without a record), Hand in (awaiting with a run; disabled "Handing in..." while the dialogue runs), Stop (accepted, in progress, asks once), Try again (blocked). Locked, available, done, skipped and ready-but-not-startable have none and say why in `reason` or `why`.
- The web part line renders that action (the drawer skips it when the Now tile already carries the head's button); the Proving Ground tapper (`cmd/d2scenario/tap.go`) presses the same action, so a part without a button is never tapped. Tests: `TestChunkActionEveryState`, the table in `testdata/questui/quest_ui.mjs`, `TestTapperPressesTheRowsButton`.
- The "Sail east to Act 2: DONE" row of the same screenshot: FIXED (card equipswap, 2026-11-22). Root cause (V by test, `TestActHopIsNotDoneByTheDialogueAlone`): the hand-in step of the hop is `StepTalk(..., done, ready)` and `Step.Done` read `!Need`; `Need` says "not needed" while the ready test is false (Andariel alive), so the part read done on proxy evidence (T3, which is why Radament stayed locked: T3 never unlocks) for every hero in Act 1 that had not killed her. The same class as CAIN-L3. The four act-hop hand-in steps (`internal/task` act1 to act4) now set `Step.Evidence` to the world's act, and a hand-in dialogue no longer latches an `unlock` quest done (`chunkFact`), so the row is done only when the world is in the next act. Not read from any save.

## Stale accept (auto-finish)

A live run whose chunk reads done or skipped in the view (the quest word says done for the whole quest while the chunk's own check does not) is closed on the next load: the chunk is latched with source `auto-finish` and the log line `quest board: auto-finish <quest>/<chunk>` says why. Before this the run stayed live, a done row has no Stop control, and D5 (one quest at a time) refused every other Start with 409 `busy`. Fixed in `Exec.smAutoFinish` (internal/questboard/exec_sm.go); tests `TestStaleAcceptDoesNotBlock`, scenario `board.stale-accept-unblocks`. Finished runs staying in `accepted` is by design (the dual write keeps them as the proof the board ran them); only live runs block.

## Try anyway (owner override, 2026-11-24, ADR 0030 amended)
A part that reads `available` because of a readiness refusal (the board word `too hard for now`, a level or life floor, the Andariel unattended hold) offers a second button, `Try anyway (risk)` (`POST /api/quests/<id>/try`, `{"confirm": true}` on the second tap; `Desk.StartAnyway`, `Exec.AcceptTry`). It starts that one part for one attempt: `Exec.TryKey` holds the chunk key in memory, `task.SetTry` turns the readiness checks of the gates off while the board runs it (the board word in `CanStart`, `task.routeFloor`, `BarracksWhy`, `BossGate.floor`, the tower floor, `andarielHeld`, `gate()`, the boss script floor `boss.Opts.FloorOff`), and the executor ends it when the head decision is no longer a live run of that chunk, when a death is reported, or at a restart. The kit gates, the death-spiral guards and the combat reactions are not touched. The reply carries `override` (the refusal it skipped); the audit line `override: try anyway (refused: ...)`. Refused with `confirm` without the flag, `measured_run` under `D2BOT_RUN`, `not_overridable` for a part that is locked, done, parked or running. Operator: `d2bot ops quest start <quest> --try-anyway --why ...` (docs/operator.md). Tests: `TestTryAnywayStartsOneNotReadyChunk`, `TestTryAnywayIsAuditedAndRateLimited`, `TestQuestsTryAnywayRoute`, `TestTryAnywaySkipsReadinessNotTheKit`, `TestTryAnywayLeavesTheSurvivalTripsAlone`.

Hand-in tap (akarafix): the log has `quest board: hand-in queued|served|dropped <quest>/<chunk>`. The mailbox holds one request with the tapper's ctx; a request whose ctx ended (5 minute timeout) is removed and never talked out, so a later tap is not refused with "already waiting". Idle goals serve the tap in town (a Runs goal calls the same function before its farm step; U, not driven in a test). Live drill: not run.

## boardSM legacy removal (card bsmdef prep, 2026-11-27; plan only, nothing removed)

Precondition (all three, in this order): the restart drill is green (`board.sm.restart-mid-chain`, `board.park.persist-restart`, `board.sm.stop-cascades`, Paladin 11 and Sorceress 3; docs/probes/lab-night-plan-2026-11-27.md Run 1), the 2 h flip window with `quests.boardSM` 1 on the quest sandbox is clean, and the accelerated soak is green. Then flip the default (`internal/env/tuning_quests.go`, `BoardSM: 1`), keep it for one release, and do the removal below. Every read of the knob goes through one seam per package; removal deletes the branches that read it. V: inventory from `grep -rn "BoardLegacy\|boardLegacy()" internal cmd` (no read of the knob in cmd/d2bot or cmd/d2drill; the scenario runner only sets it as a knob).

| File | Symbol | Replacement |
|---|---|---|
| internal/env/tuning_quests.go | `QuestsTuning.BoardSM`, `BoardLegacy()`, the knob row | delete knob, field and method; `data/darklaunch.json` row to `retired`; regenerate docs/knobs.md |
| internal/bot/questboard.go | `Runner.boardLegacy()`, `Options{SM: ...}` | delete helper; drop `SM` from the options literal |
| internal/bot/task.go | `w.BoardSM = !r.boardLegacy()` | delete line (World field goes) |
| internal/bot/routeguard.go | `kitPlanOn()`: `... \|\| !r.boardLegacy()` | `return true` (then fold into `Town.Plan` logic or inline) |
| internal/task/world.go, gate.go | `World.BoardSM`, `World.boardLegacy()`; `refusal`, `gatedReady`, `needs_potions` branches (4 sites) | keep the coded branch, delete the `else` (uncoded `Fault`); delete the field |
| internal/questboard/exec.go | `Exec.boardLegacy()` (in exec_sm.go) read at `derive` (smLoad), `open` (the legacy early return), `AcceptRuns`/`acceptLocked`, `CancelRuns`, `Reset`, `UnblockRuns`, `next` | keep the `sm*` arm, delete the legacy arm |
| internal/questboard/exec.go | `repairLocked`, the old `runState` counters used only without the machine, plan-file dual write of `Accepted` (store.go) | delete after the arms above; `Accepted` stays readable for old plan files (load step) |
| internal/questboard/exec_sm.go | the `if e.boardLegacy() { return }` guards in `smPark`, `smReleaseRetry` | delete the guard |
| internal/questboard/desk.go, desk_reason.go | `x.boardLegacy()` in `Desk` (changed refs), `SMRuns`, `SMOn()` | keep SM arm; `SMOn()` returns true then delete callers' checks |
| internal/questboard/exec_test.go | `testSM`, `skipSM`, `QUESTBOARD_SM` | delete; tests that state a changed old behaviour go with it |
| internal/questboard/legacy_test.go | `TestBoardLegacySwitchIdle` | delete with the seam |
| internal/bot/kitplan_test.go | stage 10 case | rename or delete |

The seam test `TestBoardLegacySwitchIdle` (internal/questboard) proves for two builds that offers, state, the next decision and the plan file are byte-identical at knob 0 and 1 on an idle board; the paths the machine changes on purpose (accept, park, Retry, holds) are the SM suite (`exec_sm_test.go`, `QUESTBOARD_SM=1 go test ./internal/questboard`).

Drill scenario state (V `go run ./cmd/d2scenario validate`: 258 scenarios ok): `board.sm.restart-mid-chain` active, has `restart_at_s` 300, asserts the scroll step ran at most once and the chunk is done after the restart (proves a latch survives, not a park). `board.sm.stop-cascades` active. `board.park.persist-restart` was pending with no `restart_at_s` and a setup that cannot park; rewritten (not run) to take its park from the `board.hold.timeout` setup, restart at 1500 wall seconds, assert `blocked/hold_timeout` and no new start of the chunk after the restart. Gaps: (1) the park drill has never run and its timing is U; (2) `board.hold.timeout` itself was RED (kit minima, fixed for restock only by bsm7), so the park may not form on the drill hero; (3) Sorceress never ran restart-mid-chain; (4) no scenario restarts while a run is `held` or `awaiting`.
## Parks after deaths and the pause policy (card questresume, 2026-11-27)

Sandbox logs 2026-11-27 (`docs/probes/sandbox-log-analysis-2026-11-27.md`, problem 1): a Questy hero did no quest for about 42 h after "2 deaths in one chunk".

- **What pauses the board.** Two deaths on one chunk (`Options.MaxDeaths`, default 2) park the chunk: with `quests.boardSM` the run is stored as `parked` with code `died_too_often` and the park time (`Run.ParkedAt`); a parked head holds every chunk behind it. V by code (exec.go `Report`, sm.go `Promote`). The "pause: 2 deaths in one chunk" audit line of that sandbox came from the operator's quest loop outside this repo (U), which cancelled the run: after a cancel the board has no run at all and only a player's Start, or an opted-in auto-start (S8), begins the next chunk.
- **What clears it.** Before this card only the player (Try again, Reset); no timer, no recovery. Now, with `quests.resumeAfterDeaths` 1 and `quests.boardSM` 1, a row of `data/questboard/resume.json` lifts a park by itself: `afterMin` minutes without a death (counted from the park or from the hero's last death), life at `minLifePct`, calm, no purse recovery, at most `dayCap` resumes of the board in 24 h, and at most `maxAutoResumes` in a row per chunk (the park that follows an automatic resume waits for the player's Try again, which resets the count). Default row: 45 min, 95 %, 1 in a row, 3 a day. Each resume is an `autoResume` event in the plan's auto log (the "while you were away" card shows it) and a log line. Knob and ledger row: `data/darklaunch.json` (`quests.resumeAfterDeaths`, default 0, built offline only, U live).
- **What the player sees, always (knob on or off).** `/api/quests`: the chunk row, the run and the Needs-you row carry `reason.code` `paused_after_deaths` (the stored code stays `died_too_often`), `reason.since` (the park time) and `reason.clears` (in words: resumes in about N min / only you can clear it with Try again / already resumed once). Another chunk's offer says "X is paused and holds the board" instead of "another chunk is running", and its Hand In and Start stay held behind the park by design (one head per hero). The Return screen has a "Quests paused" section with the same lines (`Desk.ParkLines`).
- **Refusal park (card refusalpark, `quests.refusalPark`, default 0, dark).** A guard that refuses one chunk with the same code `parkAfter` times in a row (`data/questboard/refusal.json`, 3) parks it as `refused_repeatedly` instead of counting failures that end in `gave_up`; the run keeps the refusal's code and text (`refusal_code`, `refusal_why`), shown in `reason.code`/`reason.text` and the Return screen "Quests paused" line. Mixed codes and refusals without a code keep today's counting. The failsafe also skips rung 4 (the game restart) when at least 3 refused steps happened since the last kill, xp or area change, and goes to the visible pause. Ledger row in `data/darklaunch.json`; U live.
- **Is a human click meant to be the only starter of chunks?** No. The design is: the player taps Start, or opts a quest into auto-start (S8; a safe offer starts by itself after a countdown, rate limited), and with the pause policy on a death park lifts by itself. With no opt-in and the policy off a hero left alone starts nothing, which is what the 42 h was. Auto-start is still per quest and default off.
- **Campaign counter.** With the board on the task executor never ticks, so `Runner.Progress` (the only source of the campaign goal's Current) was never refreshed and read 0 of 6 for ever; `reportBoardProgress` (internal/bot/boardprogress.go) now reports the task layer's count over the world plus every quest the board shows done. V by the must-fail test `TestCampaignProgressCountsQuestsUnderTheBoard`.
- **`arbiter-starved`.** The incident text now names the intent that won, the area, and the parked board runs (internal/bot/starvedcause.go).
