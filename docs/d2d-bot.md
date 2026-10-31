# The bot on the d2d backend (agent D2DBACKEND, 2026-10-30)

> status: current; d2d is the primary backend (ADR 0009, 2026-10-30; docs/plan/STATUS.md), but the flag default is still --backend game until card d2d-default-backend lands.

`d2bot --backend d2d` runs the whole bot (goal manager, planner, runner, dashboard/API, shutdown sequence, knobs) against a d2d
process instead of Game.exe. The default backend is unchanged (`--backend game`, byte-identical behaviour; the Game.exe path
only gained three lines in `main.go` and a one-line factory call in `task.go`). Code: `internal/d2denv` (new),
`cmd/d2bot/d2dbackend.go`. Tags: **V** verified (offline tests named below), **U** unverified (needs the live d2d), **S** secondary.

## Run it

```
# attach to a d2d that is already listening (the d2d is never stopped or killed):
d2bot --backend d2d --attach --d2d-sock /path/d2d.sock --goals 'farm:Blood Moor' --addr 127.0.0.1:8420

# managed: d2bot starts d2d (private fork binary), saves and quits it on stop (throwaway character):
D2D_BIN=~/d2d-lab/fork/src/build/apps/d2d/d2d D2D_DATA=~/d2d-lab/data \
d2bot --backend d2d --managed --no-save --d2d-class 1 --char Booty --seed 5 \
      --toggle autoloot=off,autoparty=off --instance i9 --autostart
```

Flags (every one has an env twin): `--backend game|d2d` (`D2BOT_BACKEND`), `--d2d-sock` (`D2D_SOCK`; default `<run dir>/d2d-<instance>.sock`,
the instance is `--instance`/`D2BOT_INSTANCE`), `--d2d-bin` (`D2D_BIN`), `--d2d-data` (`D2D_DATA`), `--d2d-class N` (d2d UI order,
1 = Necromancer; -1 = load the saved `--char` from the select screen), `--seed N` (deterministic map layout, passed to d2d; 0 = random),
`--no-save`, `--toggle k=v,...`, `--d2d-args '--lockstep ...'` (verbatim), `--d2d-nice` (default 10), `--d2d-linkwait`.
`--attach` = a d2d already answers on the socket, `--managed` = d2bot starts it; `--sim` and `--dry` are refused with d2d (dry reads
game memory). `--clean-start` (set by `--supervise` on a restart) quits a d2d left over by the crashed d2bot; without it, a managed
start on an answering socket is an error. The d2d is launched with its own `XDG_CONFIG_HOME/XDG_DATA_HOME` under the instance dir.

## What is built (`internal/d2denv.Build`)

| `env.Env` field | d2d source |
|---|---|
| `Read` | `d2denv.Reader`: `d2dsrc.Source` plus `Unit.Addr = unit id` and stats by unit id (below) |
| `In` | `d2dinput.Controller` (semantic and pixel input), one shared `d2dctl.Client` |
| `Clock` | `LateClock`: the lockstep fork's `Pacer` (Sleep steps the game, virtual time) when `clock` says lockstep, else wall clock; decided after d2d is up |
| `Mem` | `noMem`: every read fails with `d2.ErrUnsupported`, so the few remaining memory touches take their "unreadable" path instead of a nil panic |
| `World` | nil; `Backend.Navigator()` = `nav.New(env)` + `nav.UseWorld(d2dworld.World)` |

One `d2dctl.Client` per backend (d2d allows one client per socket) and it is STABLE: it redials lazily after a transport error, so
Reader, Controller and World survive a d2d restart. For that reason `d2denv.Proc` starts d2d itself (using `d2dinput.Process.Args`
for the flag line) instead of `d2dinput.Process`, which creates a new client per start.

### Open items of docs/d2d-backend.md closed here

- **Stats by unit id.** `Reader.UnitStats/ReadStatList/ReadBaseStatList(id)`: items give quantity (70), durability (72/73) and socket count
  (194) from the last snapshot's item facts (stat ids S: classic itemstatcost order); the player id gives the player's `Stats` when the
  dump verb exists; id 0, monsters, objects and raw lists stay `d2.ErrUnsupported` (callers already degrade; combat falls back to the static
  resist tables). `Unit.Addr` is set to the unit id so callers that key reads on `Addr` work (ids are unique across the lists, U).
  V: `TestUnitStatsByID`, `TestBuildReadsStateAndAddsAddr`.
- **Quest source factory.** `memQuestSource(env)` (cmd/d2bot) returns questmem for the memory reader and `d2denv.QuestSource` for the d2d
  reader; the latter is a `quest.Source` + `Waypoint` over `d2dsrc.Source.Progress` ("dump quests", cached 200 ms). Same word and waypoint-bit
  decoding as questmem. Without the dump verb every answer is unknown (the save file reader then decides), never "not done".
  V: `TestQuestSource`, `TestMemQuestSourceFactory`.
- **Launch Flow backend choice.** `launch.Flow` (menus over OOG) is not used. `d2denv.Cycler` is the `bot.Cycler` (and the combat `Leave` hook via
  `LeaveGame`): enter = start/attach + wait for `screen=ingame` + priming click; leave = `save` + `quit` (managed) or `save` only
  (attach). After a first game with saving on, later entries load the saved character from the select screen (`Class -1`, V flow in
  d2dinput). Difficulty other than normal: `input.ErrUnsupported` (no start flag). `NewGame` on an attached d2d: `ErrUnsupported`.
  V offline: `TestCyclerAttach`; the managed process path is U (no d2d in tests).
- **Instance and socket config.** `D2D_SOCK` and `D2BOT_INSTANCE` honoured, per-instance default socket (see above). V: `TestBackendFlags`.
- **Deterministic seed flag.** `--seed N` -> d2d `--seed N` (map layout). The bot's own randomness is already fixed (nav rng seed 1).
  Monster behaviour determinism needs the fork's `--lockstep` (`--d2d-args --lockstep`), U.

## Seams for combat and town

`d2denv.Seams{Fighter, Town, Loot}` are factories `func(*env.Env, *nav.Navigator)` returning `bot.Fighter`, `bot.Townsman`, `bot.Looter`.
`cmd/d2bot/d2dbackend.go` has `var d2dSeams d2denv.Seams`: the combat agent sets `d2dSeams.Fighter = ...`, the town agent `d2dSeams.Town = ...`
(one line each in that file, or from their own `init()` in a new file). Nil fields run `d2denv.NotYetFighter/NotYetTown`: every action fails with
`d2denv.ErrNotYet`, logs `not yet what=combat.Clear` (once a minute per kind), `Safety` never asks for anything, `NeedsTrip` is false.
`Loot` has no stub (nil = the runner's no-pickup path). Not wired yet because they need a town to exist: `Task` (campaign goals; refused at
Start while nil, then `newTaskDeps` with a `*town.Town` and `memQuestSource` already picks the right quest source), `Leveler`, `Gear`, `Quester`,
`SavedExp`, `Prepare` (hotkeys), `Recycle` (no Game.exe heap hygiene on d2d). `Restart` is wired to `Cycler.NewGame`.
Health: `launch.Monitor` over the channel (ping = alive; managed uses the d2d pid, attach uses 1); a stuck-but-answering d2d is not detected (U).

## Verified and not

- V offline: `go test -race -count=1 -p 1 ./internal/d2denv ./cmd/d2bot ./internal/d2dctl` (and the other d2d* packages). `TestRunD2DBootsOnFakeAndStopsCleanly`
  boots `runD2D` against `internal/d2dctl/fake`: state read, dashboard and API up, a `farm:Blood Moor` goal runs through the goal manager and the
  runner (`state ... to=traveling`, nav asks for the exit, the fake world has no exit so it fails and retries), and cancel stops cleanly without `quit`.
- Found and fixed on the way: `d2dctl.Client.Do` dereferenced a nil `c.conn` when its context was cancelled mid-call (the `AfterFunc` callback ran after
  `drop`); it now captures the conn. This would have crashed d2bot at SIGTERM while the runner was in a verb. (one-line change in `internal/d2dctl/client.go`,
  outside D2DBACKEND's directories: take it with the merge.)
- V live (private fork on the worker host, 2026-10-30, `flock /tmp/d2d-lab.lock`, nice 10, d2d stopped afterwards, nothing else touched): `d2bot --backend d2d --managed --no-save
  --d2d-class 1 --char Booty --seed 5 --toggle autoloot=off,autoparty=off --d2d-linkwait 90s --goals 'farm:Blood Moor' --run-for 150s`: d2d started by d2bot, character
  in the Rogue Encampment, dashboard/API/healthz up (`/api/v1/instances/d2dbot/status` showed Booty, Necromancer, area, position), goal manager ran the farm goal, nav
  reported "no exit to Blood Moor" (town exit via level border NOT found even with a 90 s link wait: open, d2dworld/nav), the failsafe ladder drove `Cycler.NewGame`
  (save skipped under --no-save, quit, relaunch, new throwaway character, back in game in about 2.5 s) and the game-restart rung twice, SIGTERM ended the bot cleanly
  (bot exit 0). First run exposed that `save` is refused under `--no-save` (`err no character store`); fixed (`Proc.NoSave`).
- U: everything beyond that against a real d2d, see the report's live checklist. The d2d is built with a different map origin per level (`OriginNone`);
  `Controller.Origin` is zero, matching `d2dsrc.Options.AddOrigin=false`.
- Known gaps: the d2dsrc "unknown field" notes are logged once at info per field (about 40 lines at boot on a stock d2d); `--dry` is not available; the
  dashboard's gear/inventory panels show only what d2dsrc fills.
