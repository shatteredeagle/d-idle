# Nightmare and Hell campaigns (nightmare_all, hell_all)

> status: OFFLINE ONLY: Nightmare and Hell never ran live; 400+ code commits since this was last touched. Treat as U.

Tags: V verified in code or tests, U unverified (no live game used).

- Templates `nightmare_all` / `hell_all` (`task.NightmareAll/HellAll`, selected by `task.Template`) reuse the Normal act quest nodes (acts 1-2 today, same set as `normal_all`; they grow with it). V: quest state is read per difficulty slot (`bot.QuestReader.Read(difficulty)`), so a Nightmare game starts with every quest unsatisfied and the same nodes replay. U: the game really re-offers each quest per difficulty (Den, Tools, Radament chain...).
- Floors: `combat.EntryGate` act floor (V code) raised by the boss table in `internal/task/nightmare.go` (docs/difficulty.md 2.2, R). Hell act 1 entries use the dungeon floor 75 (U guess for Burial Grounds). Kit-less floors are dropped (the floors are far above them).
- Wrong difficulty: `Campaign.Plan` returns Stuck while the world's difficulty differs; the bot switches before planning (`ensureDifficulty`, goal difficulty).
- goals: the templates default `Params.Difficulty` to nightmare/hell (conflicting values are rejected), so `bot.ensureDifficulty` starts the new game via `Cycler.NewGame(diff)`. Only hook in `internal/bot`: labels in `campaignActs`.
- Switch after Baal: `task.NextDifficulty(cur, baalDone)` and `task.ReadyForNext(next, clvl)` are pure helpers; nothing calls them yet (Acts 4/5 and Baal nodes do not exist in master, so no Eve of Destruction signal). A goal chain "normal_all -> nightmare_all" is for the manager/UI to create when `NextDifficulty` is non-empty. U: the new-game dialog offers the next difficulty only once the character beat Baal (docs/progression/difficulty.md 3).
- Between quests the level step is answered by `progression-<difficulty>.json` (test: every floor is at or below the top `leave_clvl` of that file).

## Auto-chaining the next difficulty (NEXTDIFF)

When a `normal_all` or `nightmare_all` campaign goal with `Params.Chain` completes (all quests done, Baal killed), the
goal manager (`internal/goals/chain.go`) queues the next difficulty:

- `task.NextDifficulty(cur, true)` names it, `task.ReadyForNext(next, clvl)` gates it (clvl read from the runner status).
- Ready: one `nightmare_all` / `hell_all` campaign goal (Chain stays on, so Hell follows Nightmare; `hell_all` ends the chain).
- Not ready: a `reach_level` goal to `task.EntryLevel(next)` (in the finished difficulty) first, then the campaign goal.
- Idempotence: the created ids are stored in the finished goal's `nextIds` (persisted in the goals file). A goal with
  `nextIds` is never chained again, even if the user deletes the follow-ups; on start the manager also queues the
  follow-up of a chained goal that finished but has none (crash between finish and save).
- Dashboard: the finished goal's note says what was queued ("chained: g12 nightmare_all queued", or the level gate and why).
- Config: `Params.Chain` per goal (API field `params.chain`). `--goals campaign[:TEMPLATE]` sets it ON; `campaign:TEMPLATE:nochain`
  turns it off; goals added through the API/UI default OFF. The manager gets the rules through `goals.Options.Chain`
  (goals cannot import task); `cmd/d2bot` wires `taskChain()`. I did not add an `internal/env` knob (not my directory; per-goal
  Params is persisted and visible, which a process-wide knob is not). Tests: `internal/goals/chain_test.go` (stubbed rules).
- U: not exercised live; the reach_level goal relies on the bot leveling in the finished difficulty.
