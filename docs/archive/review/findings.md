# Critical review: findings (agent REVIEW, 2026-10-28)

Scope: master at `bd8c13f` (NAV2 merged) plus a read of the running branches `agent/leveler` (`3c81bd1`) and
`agent/combat2` (`8d25c2f`). Docs only, no game access. Tags: **(V)** verified in repo/docs/source I read or a command
I ran here, **(S)** secondary (search result, prior-art doc), **(U)** unverified or from memory.

Severity: **blocker** = stops a usable v1 or will cause silent damage; **high** = will cost weeks or a rewrite if left;
**medium** = real cost, can be scheduled; **low** = hygiene.

## Summary table

| # | sev | finding | owner today |
|---|---|---|---|
| F1 | blocker | No product process topology: memory access, game lifecycle and recovery live in three places (setcap dev tools, internal/launch, bash game.sh over ssh), and the product path (managed mode + subreaper) has never run live | nobody |
| F2 | blocker | Progression past Act 1 has no owner: quest state, act transitions, boss scripts, mercenary, difficulty switch | nobody (gap between LEVELER, TOWN2, COMBAT2) |
| F3 | high | Character save integrity is unprotected: kill-restart loses progress, no rotation/backup outside the cheat path, no checksum check | nobody |
| F4 | high | No CI; master is red intermittently (a bot failsafe test failed 2 of 4 runs here) | main agent |
| F5 | high | Single-instance product surfaces: one goals.Manager, no `/instances/{id}` API, three HTTP servers with three auth setups | nobody |
| F6 | high | Input has no arbitration and no cancellation: bot, sentinel and tools post input to the same window; `Controller` methods take no ctx | nobody |
| F7 | high | Per-tick state is read several times by several owners (nav, bot loop, bot guard, telemetry, sentinel) with no shared tick | nobody |
| F8 | medium | Layering inversion: `env` (below behaviors) imports `goals` (top) for the Event type; dual emission patched with `disableObserve` | GOALS/BOT |
| F9 | medium | UI-flag semantics defined three ways (deny-list in d2, allow-list in input, patches in town/sentinel) | MEM/INPUT |
| F10 | medium | Map seed may be stable per character in single player; big lever for route caching and deterministic tests | NAV/MAP |
| F11 | medium | Milestone ladder mis-ordered: setup/container and soak last though they shape everything; no rung for quests/merc/saves | main |
| F12 | medium | Fixture formats proliferate with no version/build header (D2SN, d2mi, jsonl death traces) and no recorder standard | MEM/MAP/COMBAT |
| F13 | medium | Configuration has six sources and no precedence model | BOT/GOALS |
| F14 | medium | Error taxonomy is per-package sentinel errors matched by hand in the runner | BOT |
| F15 | medium | Plan data copied three times (docs/plan/data, internal/bot/plandata, internal/leveler/plandata) | BOT/LEVELER |
| F16 | medium | Dev-only paths and hosts baked into code (`/mnt/cold`, `~/.local/state`, `/tmp` locks, `game.sh` over ssh) | CHEAT/SENTINEL |
| F17 | low | A 6 MB binary `levelerlive` is committed at the repo root on `agent/leveler` | LEVELER |
| F18 | low | ~1 core per idle instance (Game.exe busy-loops) caps fleet and container density | MULTI |
| F19 | low | Memory file `~/.claude/memory/d2-idle-bot.md` is 33 KB; its first sections describe the abandoned D2R plan | main |

## Blockers

### F1. No product process topology (blocker)
Evidence:
- Every live run so far uses `--attach` with `setcap cap_sys_ptrace=ep` granted by `worker-sudo` (V `tools/live.sh`, AGENTS.md).
  The product path (`d2bot --managed`: the app launches the game as a child, `PR_SET_CHILD_SUBREAPER`, no capability) is proven
  only on a scratch prefix with a harmless probe exe (V docs/launch.md section 1), never with the game and never end to end
  (V docs/bot.md: restart hook "not wired in cmd/d2bot yet").
- docs/launch.md section 1 states the rule: the reader must be the subreaper or a descendant of it. The sentinel is a
  *separate* process that reads game memory (V docs/sentinel.md "Observation"). Under the subreaper model it cannot read
  the game unless it is the ancestor or holds CAP_SYS_PTRACE. PLAN wants both "no root" and "an out-of-process sentinel
  that does not trust the bot". These collide and no doc says so.
- Restart logic lives in three places: `internal/launch` Keeper/Managed (Go, offline-tested only), `tools/game.sh` (bash,
  the one that works, runs `ssh <user>@worker.example...`), and the sentinel shells out to `game.sh` (V internal/sentinel/act.go:301-315;
  docs/sentinel.md "Restarts call the shipped game.sh (it ssh's to the worker host itself)"). A product cannot ship a sentinel that ssh's to the worker host.
- bot.Runner's `Deps.Restart` (ladder rung 4) is unwired (V docs/bot.md "Failsafes").

Why it matters: the container end goal, the sentinel, multi-instance and crash recovery all hang off one decision: which
process is the game's ancestor and who holds the memory handle. Every week, more code assumes `--attach` + setcap.

Recommendation (design in design-gaps.md D1):
1. One supervisor process per host/container (`d2bot --managed`) is the subreaper and the only process that launches Wine.
   It opens `/proc/<pid>/mem` once per game and hands the fd to anything else that must read: the sentinel runs in-process
   as a separate goroutine tree (plus a tiny external liveness watchdog that reads no memory), or receives the fd over a
   unix socket (`SCM_RIGHTS`). As far as I remember, `/proc/pid/mem` checks ptrace access at `open()` and later reads use the
   mm captured at open (U: verify with WP-MEMFD's offline test before relying on it).
2. In containers allow `--cap-add SYS_PTRACE` as the simple fallback (private tool), never require it.
3. Port `game.sh` start/stop/helper/mkprefix into `internal/launch`; the sentinel calls `launch`, not bash.
4. Run the managed path live once, early (M0 exit criterion), before more `--attach`-only code lands.

### F2. Progression past Act 1 has no owner (blocker for v1)
Evidence:
- `config.json` `acts` defaults to `[1]`, "act changes need quests" (V docs/bot.md "Config reference"); "Not done: ...
  difficulty switching, act changes, kill_boss scripting" (V docs/bot.md "Integration status").
- Quest bits and the waypoint-active mask in memory are unknown (V docs/prior-art/lessons.md D1; no quest RVA in docs/d2-1.14d-memory.md).
- No mercenary reader, hire or revive anywhere (V no merc code under internal/; docs/plan/bot-requirements.md C11). The plan relies
  on an Act 2 Might merc against cold immunes (V docs/plan/leveling.md 3).
- Waypoint tabs for acts 2-5 UNVERIFIED (V docs/nav.md NAV2).
- Running agents: LEVELER = stats/skills/gear/shop (V branch diff), TOWN2 = town live verification, COMBAT2 = death facts/reach
  (V branch diff), RESEARCH-CLASSES/GOALS = docs. None owns quests, act travel (Warriv, Meshif, Act 4/5 portals), boss arenas,
  merc, or the difficulty screen.
Recommendation: a PROGRESSION track now. Interim source of truth for quests and waypoints: the save file after each Save and
Exit; internal/cheat/d2s already decodes quests at 0x14F and waypoints at 0x279 (V docs/cheats.md). WP-SAVES exposes them
read-only; a later live QUESTMEM package finds the in-memory quest array. Make "Normal Act 1 to Act 2" an explicit milestone (roadmap M2).

## High

### F3. Character save integrity (high)
The sentinel's remedy for hangs/crashes is kill and restart (V docs/sentinel.md). In single player the character is written on
Save and Exit (U: possibly also on some transitions; never measured), so a kill loses everything since the last save. Backups exist
only on the cheat path (`/mnt/cold/d2lod/backups`, V docs/cheats.md). Nothing validates the `.d2s` checksum at start or rotates
snapshots; a corrupted save (crash during write, NVMe allowed down to ~10 GB free) loses the character.
Recommendation: WP-SAVES (snapshot `.d2s/.key/.map/.ma*` before every game start, rotation, checksum + parse on load, refuse a corrupt
file and offer restore). Add a bounded save cycle to the bot (leave + re-enter every 20-30 min, a knob) so a kill loses at most one cycle.
Live test to add: kill -9 after gaining XP and compare the `.d2s`.

### F4. No CI; master intermittently red (high)
Evidence (V, run here on the head box): `go test -short -count=1 ./...` failed on `TestFailsafeDeathOverlayNeverDismissed` (internal/bot:
"want a visible non-fatal failsafe error, got []"); three more internal/bot runs: ok / FAIL / ok. Known flaky already:
`TestFailsafeClearingWithNoProgress`, `mem.TestProcInstance` (V memory file). No `.forgejo/` workflows (V). Merges are tested by hand on a
4-core box. Bot tests take ~15 s despite a virtual clock: `input.Game` sleeps with `time.Sleep` directly (V internal/input/controller.go:101,227,287,315).
Recommendation: WP-CI (forge Actions: `gofmt -l`, `go vet`, `go test -race -short -count=3`, cross-build `d2input.exe` with
`-H=windowsgui`), a quarantine list with owners, and give `input.Game` a Clock.

### F5. Single-instance product surfaces (high)
`goals.New(Options{Runner})` takes one runner; API routes have no instance (V docs/api.md). NET decisions #1 and #6 (instance in API
routes, Fleet with fleet-of-one default) are not implemented (V docs/net/design-now.md section 3; no instance routes in internal/api).
Three servers: d2bot API + `/live/` (8420), d2watch (8890), d2sentinel (8895), each with its own token handling (V docs/api.md,
docs/watch.md, docs/sentinel.md). `internal/status` and `internal/api` are two SSE stacks with different event shapes (V docs/display.md, docs/api.md).
Recommendation: one HTTP server per host (WP-SERVER) with `/api/v1/instances/{id}/...`, current routes as aliases for the first
instance, one auth model, d2watch and sentinel mounted as handlers. Before the dashboard grows further.

### F6. Input arbitration and cancellation (high)
`input.Controller` methods take no `context.Context` (V internal/input/controller.go:55-74); `MoveTo`, `Interact`, `CloseAll` loop
with real sleeps. The sentinel acts on the same window (Esc, dismiss, OOG start) while the bot may be acting; its protection is a 5 s
settle and the instance lock for some actions, and Esc/dismiss/SIGCONT do not take the lock (V docs/sentinel.md "Engine"). An Esc from
the sentinel just after the bot closed a panel opens the game menu (V docs/mem.md Q1 caution).
Recommendation: an input lease (design-gaps.md D9): every call carries an owner and priority; an arbiter serialises and rejects lower
priorities while a higher one holds the lease; the sentinel takes the lease before acting. Add ctx to Controller in one sweep after the wave-3 merges.

### F7. No single per-tick observation (high)
nav calls `Tracker.Update` and reads `d2.Reader.Snapshot` itself (V docs/nav.md "nav is the caller of Tracker.Update"); the bot loop
polls a snapshot and its guard goroutine polls again (V docs/bot.md "Run loop", "Guard"); telemetry gets the runner's snapshots via
`Deps.OnSnapshot`; the sentinel reads every 2 s. `d2.Reader` reports torn reads as `StateLoading` (V docs/mem.md). Two readers in one
tick can disagree (dead vs alive), the class of bug behind the 7-minute death-overlay stall (V docs/bot.md run 9). Each extra reader
also costs ~150 small `/proc` reads per Tracker update (V docs/map.md).
Recommendation: one Observer goroutine per instance produces `Tick{Seq, Time, Snapshot, World}`; consumers read the latest tick or
subscribe; behaviors never call `Snapshot()` directly. Design in design-gaps.md D5.

## Medium

### F8. Layering inversion through events
`internal/env` imports `internal/goals` for `goals.Event` (V `go list` import graph), so nav/combat/town/loot compile against the goal
manager. `goals.Event` is a flat union struct without a version (V internal/goals/runner.go). Combat and loot emit some events, the bot
observer derives the same ones, and `disableObserve` turns duplicates off (V docs/bot.md "Events").
Recommendation: `internal/event` (WP-EVENT): typed, versioned events and a bounded fan-out bus; the "who emits what" rule in code.

### F9. UI flag semantics defined three ways
`d2.UIState.AnyPanelOpen` is a deny-list that needed live fixes for 6/7 and 0x11 (V internal/d2/ui.go:41-56); `input.Reader.UIOpen` is an
allow-list of closable panels (V internal/input/state.go:113-129); `town/hand.PanelOpen` clears 0x11 again (V internal/town/hand/hand.go:305);
the sentinel has its own list. Any newly found always-on flag (0x21 after `h`, V docs/mem.md) makes deny-list users see a panel forever.
Recommendation: one table in internal/d2 with per-index semantics (blocks clicks / Esc closes / HUD only / unknown) and an allow-list
`BlockingPanelOpen()`; delete the copies after the wave-3 merges.

### F10. The map seed may be stable per character
NAV2 concluded "every game has its own seed" by comparing two *characters* (V docs/nav.md NAV2). The `.d2s` header carries a 4-byte map id at
offset 171 (0xAB) (S: d2s format docs; the [d2mapseed](https://gh.r-mj.com/feored/d2mapseed) tool rewrites it in single-player saves), and saves
come with per-difficulty `.ma0..` automap files (V docs/launch.md 5), which suggests the layout persists per character across games (U).
If true: routes and exit positions can be cached per (character, difficulty); the save editor can *set* the seed for reproducible tests and
soak comparisons; the tracker cache can be persisted. Recommendation: one cheap live test (read Act+0x0C, Save and Exit, re-enter, read again;
then set 0xAB with the editor) in the next MAP/NAV session; WP-SAVES exposes the seed.

### F11. Milestone ladder ordering
PLAN ladder: 7 runs, 8 OOG, 9 leveling, 10 goal manager, 11 setup wizard, 12 soak (V PLAN.md). OOG and the goal manager were needed and built
early; the ladder no longer reflects reality. Missing rungs: quests/act transitions, merc, save integrity, difficulty switch, managed lifecycle.
Soak is last though every failure so far came from running longer (V docs/bot.md failures). Setup/container decide process topology (F1) and
file layout (F16) and must be designed now even if built later. Recommendation: roadmap.md.

### F12. Fixture formats without headers
D2SN page snapshots (V docs/mem.md), d2mi images (V docs/map.md), JSONL death traces (V agent/combat2), town `sim` worlds with made-up coordinates
(V docs/nav.md). None records build id, runner version, recorder version, character, seed or what was verified. When an offset is corrected,
nobody knows which fixtures encode the old assumption. Recommendation: WP-REPLAY: one envelope for every fixture, a stream recorder that also
records the input calls, a replayer for golden tests, and invariant checks (memory doc section 6 canaries) over every fixture.

### F13. Configuration sources
Flags + `D2BOT_*` env, strict `config.json` with `tuning` overrides and profiles, `PUT /settings` (chicken/potion % mapped onto combat knobs
"if COMBAT defines those knobs"), sentinel JSON thresholds, `tools/instances.conf`, and `D2_*` script env (V docs/bot.md, docs/api.md,
docs/sentinel.md, tools/). Precedence between `/settings.chickenLifePct` and `config.json tuning` is undocumented.
Recommendation: design-gaps.md D4 (layered config, one schema, per-instance overlays, API writes only the runtime layer).

### F14. Error taxonomy
nav has ~12 sentinel errors, combat 6+, bot its own (`ErrPortalOpened`, `ErrLeftGame`), launch `ErrStuck/ErrTimeout/ErrCrashLoop` (V docs).
The runner picks recovery by matching them; a new behavior error falls into "generic failure, ladder" unnoticed.
Recommendation: design-gaps.md D3: errors carry a `Kind`; the runner switches on Kind; a test asserts every exported error has one.

### F15. Plan data copied three times
`docs/plan/data/*.json`, `internal/bot/plandata` (sync test) and now `internal/leveler/plandata` (V agent/leveler diff: build-*.json and
quest-points.json again). Recommendation: one `internal/plandata` package (`embed.FS`, data only), one sync test; assign at the LEVELER merge.

### F16. Host-specific paths in code
`cheat.DefaultBackupDir = "/mnt/cold/d2lod/backups"` (V internal/cheat/cheat.go:88); `~/.local/state/d2lod/*.jsonl` (ledger, sentinel log);
`/tmp/d2lod-game-<id>.lock` (V internal/sentinel/observe_linux.go:59-61); live tools with hard-coded install paths (V combatlive, townlive).
PLAN constraint 2 forbids hard-coded `$HOME` paths (V PLAN.md). Recommendation: WP-STATE defines the data-dir layout and a `paths` package;
each owner migrates on next touch.

## Low
- **F17** `levelerlive` binary committed on `agent/leveler` (V `git diff --stat`). Ignore build outputs (`/levelerlive`, `/d2*`), drop before
  merge; WP-CI rejects non-testdata files over 1 MB.
- **F18** ~1.05 cores per idle instance (V docs/multi.md). Options to test later with the soak harness: cgroup `cpu.max` per instance, a frame
  limiter (U: may slow the game loop and break timing knobs).
- **F19** Memory hub file 33 KB vs the 12 KB flag (V); compression is already a pending promise in its HANDOFF.

## Things that are good (keep them)
- Consumer-side interfaces with fakes; behaviors offline-testable (V docs/wave2.md); 59 test files, most suites under a second (V, run here).
- Honest VERIFIED/UNVERIFIED tags, dead-end sections, drill tables with measured detection times (V docs/sentinel.md).
- Seed independence fixed early with two fixtures (V docs/nav.md NAV2).
- Tuning knobs with metadata and the "all knobs at worst die" proof (V docs/combat.md) keep the future meta layer honest.
- One prefix per instance via hardlink clones: cheap, isolated, verified (V docs/multi.md).
