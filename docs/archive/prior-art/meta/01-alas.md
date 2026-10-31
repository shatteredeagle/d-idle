# 1. Alas (AzurLaneAutoScript): scheduler + GUI + 24/7 unattended

Repo: https://github.com/LmeSzinc/AzurLaneAutoScript. GPL-3.0, ~9.8k stars, pushed 2026-10-28 (V, `gh api`). Mobile game (Azur Lane) driven on an Android emulator.
Tags: V = fetched by me, S = snippet/secondary, U = unverified/memory. Not a D2 fact set: nothing here is game-mechanics.

## What it is
- "Bot with GUI, designed for 24/7 running, can take over almost all gameplay" (V, README). The author frames it as "leave everything to Alas".
- Control path: ADB / uiautomator2 screenshots plus template matching and a custom map-detection module (`module.map_detection`) (V, README). This is the "computer vision" quadrant, not memory read.

## Architecture worth copying (V, read `alas.py` on master)
- **Task scheduler, not a script**: each task (daily, commission, research, campaign...) has a `next_run` timestamp. `get_next_task()` picks the soonest; if it is in the future the bot waits. While waiting a config option `Optimization_WhenTaskQueueEmpty` chooses `close_game`, `goto_main` or `stay_there`. (`alas.py` lines ~496-552.)
- **Failure budget per task**: a `failure_record` dict; a task failing 3+ times raises `RequestHumanTakeover`, logs critical, sends a notification ("Task X failed 3 or more times") and stops (V, `alas.py` ~595-610).
- **Typed exceptions drive recovery**: `GameStuckError`, `GameTooManyClickError`, `GamePageUnknownError`, `GameNotRunningError`, `EmulatorNotRunningError`, `RequestHumanTakeover` (V, `module/exception.py`). Stuck/too-many-clicks => restart game and retry; unknown page => human.
- **Error bundle**: each error writes `log/error/<ms timestamp>/log.txt` plus the recent screenshots; users are told to upload it (V, README). A diagnostic habit we should copy.
- **GUI**: PyWebIO web UI served by the same process (V: `module/webui/` has `app.py`, `process_manager.py`, `remote_access.py`, `updater.py`, `discord_presence.py`).
- **Many instances**: `ProcessManager.get_manager(config_name)` and `running_instances()` => one OS process per named config, GUI manages them (V, `process_manager.py`). That is the fleet model: config-per-instance, process-per-instance, one UI.
- **Self-updater** built into the webui (V, file exists; behavior U).

## What it got right
- Time is first-class: everything is "when may this run next" (cooldowns, daily reset), which maps 1:1 to our goal schedules.
- Humans are an explicit escalation tier (take-over request + notify), not an afterthought.
- Config is data, GUI is generated from config schema (U, believed to be from `module/config/argument/*.yaml`).

## What it got wrong / limits
- No progression/meta layer: it optimizes the game's own dailies, it does not invent goals (rebirth, cheats-as-rewards). (V by absence in README.)
- Vision on an emulator: breaks when the game UI updates; maintenance is the whole cost (S, README says "late stage of life cycle", i.e. the game's updates are the risk).
- Community guidelines present; GUI-only mobile ToS risk irrelevant to us (offline).

## Reuse
- Code: GPL-3.0, so copy only ideas, not code, unless d2lod-bot goes GPL. Design reusable: scheduler loop, wait policy (park in town vs quit vs keep the game running), failure budget, error bundle, one-process-per-config fleet.
- Mapping to ours: `Optimization_WhenTaskQueueEmpty` == "what does the idle bot do between goals" (we have close-game vs park-in-town choice, relevant to the worker host CPU cost, about one core per instance).
