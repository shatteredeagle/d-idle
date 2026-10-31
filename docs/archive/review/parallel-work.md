# Parallel work that can start now

Constraint: no package below may touch the directories of the running agents: `internal/nav` (NAV2 merged, may return),
`internal/town`, `internal/loot` (TOWN2), `internal/combat` (COMBAT2), `internal/bot`, `cmd/d2bot`, `internal/leveler`,
`tools/rigs/levelerlive` (LEVELER), `internal/env/tuning_*.go` (domain owners), `docs/classes` (RESEARCH-CLASSES), `docs/goals`
(RESEARCH-GOALS). Packages below are disjoint from each other too. Changes needed in those directories go into the
package's report as a request to the main agent.

Each package follows AGENTS.md (stdlib only, own worktree, gofmt/vet/test, commit to own branch, report under 300 words).
"Live" means a game instance; almost nothing below needs one. Sizes: S <= 1 day, M 1-3 days, L 3-5 days of agent time.
Models per the user's routing rules: Haiku for mechanical work, Sonnet by default, Opus only when asked.

## Gaps between the running agents' scopes (why some packages exist)
- Nobody owns: quests and act transitions, mercenary, boss scripts, difficulty switch (findings F2) -> WP-PROGDESIGN now, code later.
- Nobody owns: the product lifecycle (launch vs game.sh vs sentinel) (F1) -> WP-MEMFD, WP-LIFECYCLE.
- Nobody owns: save integrity (F3) -> WP-SAVES. CI (F4) -> WP-CI. Instance-scoped API (F5) -> WP-SERVER.
- LEVELER and the bot both need quest points and plan data; plan data is now in three copies (F15): resolve at the LEVELER merge (main agent), not here.
- RESEARCH-GOALS (docs/goals) designs goal types; WP-EVENT and WP-SERVER must not invent goal types, only transport them.

## Work packages

### WP-CI: continuous integration, release skeleton
- Goal: every push and merge runs gofmt, vet, `go test -race -short -count=3`, cross-builds `d2input.exe` (windows/386, `-H=windowsgui`) and all cmds; repo hygiene checks; a release recipe.
- Owns: `.forgejo/`, `tools/ci/`, `packaging/`, `.gitignore`, `CHANGELOG.md`, `THIRD_PARTY.md`.
- Inputs: The forge on Tower (memory `forge`), lab runner choice (memory `lab-machines`), AGENTS.md, findings F4/F17, gotchas 67-69.
- Acceptance: workflow file validated with `act`-style dry run or a real run on a test branch; `tools/ci/check.sh` runs locally and fails on: gofmt diff, vet, a file > 1 MB outside `testdata/`, any `*.mpq`, `*.d2s`, `*.key`, `data/global/excel/*.txt`; `tools/ci/flaky.txt` lists known flaky tests with owner; `packaging/release.sh` builds a versioned tarball (binary + d2input.exe + LICENSE notices) with `-ldflags -X main.version`; CHANGELOG.md seeded (user rule: every release has notes); THIRD_PARTY.md lists zlib blast port, D2DX (user-supplied, GPL), prior-art provenance.
- Live: no. Size S-M. Model: Sonnet (Haiku for THIRD_PARTY/CHANGELOG). Deps: none. Risk: low (runner availability).

### WP-MEMFD: memory handle passing (decides findings F1)
- Goal: prove and package "the subreaper opens `/proc/<pid>/mem` once and passes the fd to another process".
- Owns: `internal/memfd` (new).
- Inputs: docs/launch.md section 1, `internal/launch/subreaper.go`, `internal/mem` Source interface.
- Acceptance (offline, Linux only, no Wine): test binary becomes subreaper, starts a child that double-forks a sleeper (reparented to the test), opens its mem, sends the fd over a unix socket with `SCM_RIGHTS` to a *sibling* helper process; the helper reads a known marker from the sleeper while a direct open from the helper fails with EACCES under Yama scope 1 (skip with a clear message if scope is 0). `memfd.Source` implements `mem.Source` over a received fd; `memfd.Server/Client` with reconnect. A one-page result in `internal/memfd/README` section of docs (or package doc): works / does not work, kernel version.
- Live: no. Size S-M. Model: Sonnet. Deps: none. Risk: medium (if fd passing fails, fall back to in-process sentinel; that is still a decision).

### WP-LIFECYCLE: game lifecycle in Go (replace game.sh in the product path)
- Goal: `internal/launch` gains everything `tools/game.sh` does (prefix clone, Xvfb display manager, start/stop/status, helper start, wineserver -k), with no ssh and no bash; the sentinel calls it.
- Owns: `internal/launch`, `internal/sentinel`, `cmd/d2sentinel`, `tools/game.sh` (keep as a thin dev wrapper calling a Go cmd), new `cmd/d2game`.
- Inputs: tools/game.sh, docs/multi.md, docs/oog.md, docs/sentinel.md, design-gaps.md D1/D8.
- Acceptance: `launch.Display` interface (Host, Xvfb with size and lifecycle); `launch.Instance{Start,Stop,Status,EnsureHelper,MkPrefix}` with fake exec/proc tests covering: start when running (idempotent), stop cleans wineserver and Xvfb, mkprefix refuses while running, marker env set; sentinel `restart-game` uses `launch`, not `GameSh` (config field kept, deprecated); `cmd/d2game start|stop|status|helper|mkprefix <id>` has the same output as game.sh. Optional final step on i5 when the main agent frees it: one start/stop/helper cycle.
- Live: optional (i5, end only). Size L. Model: Sonnet. Deps: none (uses WP-MEMFD later). Risk: medium (Xvfb/Proton env details only show live).

### WP-CONTAINER: headless container spike (no game)
- Goal: answer the container questions without Blizzard files: Wine/Proton runs in the image, Xvfb renders, a windows/386 probe can be read via subreaper and via SYS_PTRACE, PID 1 behavior, idle cost.
- Owns: `deploy/container/` (Dockerfile, entrypoint, compose example, README), `tools/container-spike.sh`.
- Inputs: PLAN.md "End goal", docs/launch.md 1 (winprobe in `tools/subreaperproof`), gotchas 45-49; homelab rule: no deploy to NAS without asking.
- Acceptance: image builds on a lab worker (`labrun auto`), no network at runtime except an explicit setup phase; inside: Xvfb up, `winprobe` started by a Go entrypoint that is subreaper, its memory read OK without caps; second run with a sibling reader shows EACCES without `--cap-add SYS_PTRACE` and OK with it; tini vs Go-as-PID-1 reaping checked; `xwd`/`import` capture of a Wine window works; numbers: image size, idle RSS/CPU. Written result table with tags.
- Live: no (no game). Size M. Model: Sonnet. Deps: none. Risk: medium (Proton outside Steam runtime; pick plain Wine if needed and say so).

### WP-STATE: data directory layout, versioned store, migrations
- Goal: one layout for `D2BOT_DATA`/`D2BOT_CONFIG` and one way to persist versioned JSON with migrations.
- Owns: `internal/paths`, `internal/store` (new).
- Inputs: PLAN.md constraint 2, docs/api.md persistence (atomic write, .bak, .corrupt), docs/bot.md deathLogPath, docs/cheats.md ledger, findings F16, design-gaps.md D4/D11.
- Acceptance: `paths.Layout{Root}` returns every path (config, instances/<id>/{state,saves-backup,logs,fixtures}, cache/data, prefixes, runtime locks) with XDG defaults and one env override; a doc table in the package doc. `store.File[T]` with `version`, atomic write + fsync dir, `.bak`, corrupt quarantine (newest 3), `ErrFutureVersion`, and a migration registry `Register(from, to, func(json.RawMessage) (json.RawMessage, error))` tested with a three-step chain; JSONL append log with size/age rotation. No callers changed (report lists who should migrate).
- Live: no. Size M. Model: Sonnet. Deps: none. Risk: low.

### WP-SAVES: save integrity and progression state from the save
- Goal: protect characters and expose quest, waypoint, difficulty and map-seed state read from the `.d2s` (findings F2, F3, F10).
- Owns: `internal/saves` (new), `internal/cheat/d2s` (read-only API additions only; CHEAT is idle).
- Inputs: docs/cheats.md (v96 layout: checksum, quests 0x14F, waypoints 0x279, difficulty 0xA8..0xAA), S: map id at 0xAB, docs/plan/data/quest-points.json.
- Acceptance: `saves.Snapshot(dir, name)` copies `.d2s/.key/.map/.ma*` into a rotation (keep N, keep one per day), refuses while the instance's game is in game unless told it is at a menu; `saves.Verify` checks signature, size field and checksum and parses; `saves.Progress(d2s)` returns per difficulty: quests completed (named, from a table), waypoints active (named by level id), difficulty unlocked, map id; all on hand-built fixtures (no real save committed) plus `D2S_REAL=` real-copy tests that skip. A doc lists which quest bits are verified (none live yet) and a 10-minute live checklist for the main agent (map id stable across Save and Exit; kill -9 loses what).
- Live: no (checklist for later). Size M. Model: Sonnet. Deps: WP-STATE for paths (can stub). Risk: low-medium (quest bit meanings need live confirmation).

### WP-REPLAY: fixture envelope, stream recorder, replayer, canaries
- Goal: reproducible offline tests from recorded play; a startup canary that validates offsets.
- Owns: `internal/replay`, `internal/canary` (new), `cmd/d2rec` (new), `internal/mem` (snapshot envelope only; MEM is idle).
- Inputs: docs/mem.md (D2SN), docs/map.md (d2mi Recorder/Image), agent/combat2 `testdata/death/*.jsonl` format, docs/d2-1.14d-memory.md section 6, findings F12, design-gaps.md D10.
- Acceptance: envelope `{magic, schema, kind, build{timestamp,size}, runner, recorder, instance, char, difficulty, seed, area, createdAt, notes, verified[]}` readable by all three existing formats via adapters (existing fixtures load unchanged); `replay.Stream` = time-indexed sequence of page deltas + input calls; `replay.Source` implements `mem.Source` at a cursor; `replay.Input` implements `input.Controller` and records calls; golden test helper `replay.Run(t, file, fn)`; `canary.Check(src) []Result` runs the section-6 invariants and the VerifyBuild gate, tested on the committed fixtures; `cmd/d2rec record --for 30s` (for later live use through live.sh).
- Live: no (recording later). Size L. Model: Sonnet. Deps: none. Risk: medium (scope creep; keep the stream format simple: snapshot every tick + deltas later).

### WP-EVENT: typed, versioned event bus
- Goal: decouple behaviors from internal/goals (findings F8) and give one event model to API, telemetry, sentinel and logs.
- Owns: `internal/event` (new).
- Inputs: internal/goals/runner.go (Event), internal/telemetry kinds, docs/display.md, docs/bot.md events, design-gaps.md D2.
- Acceptance: `event.Event{V, Seq, Time, Instance, Run, Game, Kind, Source, Data}` with typed payload structs per kind (Kill, Drop, Death, LevelUp, AreaChange, Potion, Error, Failsafe, Lifecycle, Cheat, Goal*); JSON round-trip and schema version tests; `event.Bus` with per-subscriber bounded buffers, drop-oldest with a dropped counter, never blocks the publisher (race test); `event.FromGoals/ToGoals` adapters so nothing else must change now; a written migration plan for env/goals/telemetry (main agent executes after wave 3).
- Live: no. Size S-M. Model: Sonnet. Deps: none. Risk: low.

### WP-SERVER: one HTTP server, instance-scoped API, one auth
- Goal: merge the d2bot API, `/live/` hub, d2watch and sentinel endpoints behind one server and one token model (findings F5).
- Owns: `internal/server` (new), `internal/api`, `internal/status` (Go side).
- Inputs: docs/api.md, docs/display.md, docs/watch.md, docs/sentinel.md, docs/net/design-now.md (api section), design-gaps.md D6/D8.
- Acceptance: `server.New(Config{Addr, Token, AllowedHosts, Instances Registry})` serves `/api/v1/instances`, `/api/v1/instances/{id}/{status,goals,events,events/stream,health,deaths,settings}`, `/api/v1/fleet`, `/live/` per instance, and mount points for viewport and sentinel handlers (interfaces, not imports of cmd packages); old `/api/v1/*` routes alias the first instance (existing api tests pass unchanged); security tests (token, Host check, Origin, CSP) cover the new routes; `cmd/d2dash` serves two simulated instances.
- Live: no. Size M. Model: Sonnet. Deps: WP-EVENT (soft: can start with goals.Event). Risk: low-medium (keep backward compatibility).

### WP-DASH: dashboard UI for fleet, failsafes and knobs
- Goal: the user can watch and steer N instances from one page on phone and desktop.
- Owns: `web/`, `web/live/` (JS/CSS/HTML only).
- Inputs: docs/api.md dashboard, docs/display.md panels, PLAN.md Decisions (Knob metadata, Upgrade contract), WP-SERVER route list (agree on it first, mock until it lands), browser is Waterfox (user).
- Acceptance: instance switcher and fleet strip; red failsafe banner from health/sentinel state; death log panel (records, heat map cells); knobs page rendered only from `GET /knobs` metadata (no knob hard-coded) with profile picker; viewport slot (img/MJPEG) with an overlay layer API; checked in headless Chromium/Firefox at 390 px and desktop, light and dark; no external requests; works against `cmd/d2dash` simulated instances.
- Live: no. Size M. Model: Sonnet. Deps: WP-SERVER contract. Risk: low.

### WP-VIEWPORT: capture and stream library
- Goal: one capture/stream implementation for the dashboard and d2watch (PLAN "Viewport"), with overlay metadata.
- Owns: `internal/viewport` (new), `cmd/d2watch`.
- Inputs: docs/watch.md (capture scheduler, costs), PLAN.md Viewport, docs/input.md transform (for overlays).
- Acceptance: `viewport.Capturer` interface with `ImportCapturer` (current `import -window`) and `FFmpegCapturer` (x11grab of a window region, persistent process) behind it; lazy capture, one in flight, shared frames, fps cap, global concurrency cap (tests with a fake capturer); `viewport.Handler` serves `/frame.jpg`, `/stream.mjpg`, and `/overlay` JSON (screen-space shapes from world points via the Transform); d2watch rebuilt on it with identical behavior; a bench on any Xvfb with a dummy X client (xclock/xeyes) on a lab worker: CPU per frame at 4 and 10 fps for both capturers.
- Live: no (dummy X app). Size M. Model: Sonnet. Deps: none. Risk: low.

### WP-SETUP: setup wizard backend and `d2bot doctor`
- Goal: a supportable setup on arbitrary hosts: preflight checks with exact fixes, a resumable wizard over the existing install steps.
- Owns: `internal/setup` (new), `cmd/d2doctor` (new).
- Inputs: docs/launch.md 4 and 7 (Installer steps, verifiers, stubmeta, patch fetch), gotchas 45-49 and 72-74, PLAN.md End goal 5.
- Acceptance: `doctor` checks with pass/warn/fail + fix text: kernel Yama scope, CAP/subreaper ability (spawns a probe), Wine/Proton runner found and version, Xvfb present, 32-bit support if needed, disk free at data dir, filesystem not NTFS/noexec for the prefix, ports free, game dir verified by stub piece hashes, Game.exe version 1.14.3.71, data cache current; JSON output for the UI. `setup.Wizard` state machine (persisted, resumable): choose mode, provide installers or an existing game dir, run interactive steps (installer window shown via the viewport/noVNC), patch (user file or download with sha256), extract data, create character via OOG flow; tested with fake steps. API contract documented for WP-DASH (wizard pages later).
- Live: no. Size M. Model: Sonnet. Deps: WP-STATE (paths). Risk: medium (distro variety; keep checks data-driven).

### WP-OBS: metrics, logs and the soak harness
- Goal: measure long runs: leaks, rates, interventions; make soak continuous from M1.
- Owns: `internal/metrics` (new), `cmd/d2soak` (new), `tools/soak/`.
- Inputs: docs/bot.md health endpoint, docs/sentinel.md JSONL, docs/multi.md resource numbers, gotchas 54-58, 62, 65-66.
- Acceptance: `metrics.Registry` (counters, gauges, histograms) with Prometheus text exposition (stdlib), `metrics.ProcStats(pid)` (RSS, PSS, fds, threads, CPU ticks) for game, wineserver, helper, Xvfb, bot; Go runtime stats; `d2soak run --target URL --for 24h` samples health/metrics/sentinel every 30 s into JSONL and writes a report (XP/h, deaths/h, ladder steps/h, sentinel actions/h, RSS/fd/goroutine slopes with thresholds, longest no-progress window); offline test against `cmd/d2dash`/`--sim`; a soak checklist doc with exit criteria from roadmap.md.
- Live: no to build; the main agent runs it live later. Size M. Model: Sonnet. Deps: none (WP-SERVER makes it nicer). Risk: low.

### WP-DATA2: data tables the next milestones need
- Goal: fill the data gaps for merc, item evaluation, quests and runeword planning.
- Owns: `internal/data`.
- Inputs: docs/data.md "Not extracted", docs/plan/bot-requirements.md D1, gotcha 30 (ladder runewords), gotcha 3 (stamp).
- Acceptance: loaders for hireling (aura, act, difficulty, cost), itemstatcost (stat names, save bits, valshift for gold/exp scaling), properties, magicprefix/magicsuffix, states, missiles (ranges), lvlwarp; `runes.txt` exposes ladder/server flags and `DB.RunewordAvailableOffline(name)`; stamp includes sha256 of Patch_D2.mpq (schema bump); tests on hand-written excerpts; real-file tests skip without `/tmp/d2mpq`.
- Live: no. Size S-M. Model: Haiku for loaders, Sonnet review. Deps: none. Risk: low.

### WP-PROGDESIGN: progression design (quests, acts, merc, bosses, difficulty)
- Goal: the missing design for findings F2, so code can be dispatched when wave 3 lands.
- Owns: `docs/progression/` (new).
- Inputs: docs/prior-art/01, 02, 05, 08; docs/plan/*; docs/bot.md; RESEARCH-GOALS and RESEARCH-CLASSES outputs when they land (read their branches, do not edit); WP-SAVES progress API.
- Acceptance: per act and difficulty: quest list with completion detection (save bits, memory candidates, UI evidence), act transition procedure (NPC, dialog rows, portals), boss arena procedures (preset anchors, standing points, hazards), merc hire/revive/equip flow with UI states, difficulty selection screen handling; a task-layer design (step graph with preconditions, design-gaps.md D12) that keeps `bot.Runner` states small; a ranked live-verification list (exact checks, which instance, expected values, minutes each); proposed package split and owners. Tagged V/S/U throughout.
- Live: no. Size M. Model: Sonnet (Opus only if the user asks). Deps: none. Risk: low (docs), high value.

## Dependency graph

```
WP-CI ----------------------------------------------> (everything, as a gate)
WP-MEMFD ---> WP-LIFECYCLE ---> (sentinel via launch) ---> M1 managed-mode exit
WP-CONTAINER (independent spike; informs WP-LIFECYCLE Display and WP-SETUP)
WP-STATE ---> WP-SAVES
         \--> WP-SETUP
WP-EVENT ---> WP-SERVER ---> WP-DASH
                        \--> WP-OBS (nicer targets)
WP-VIEWPORT ---> WP-SERVER mount ---> WP-DASH viewport slot
WP-REPLAY (independent; canary used by WP-LIFECYCLE start gate later)
WP-DATA2 ---> WP-PROGDESIGN code phase (merc, items)
WP-SAVES ---> WP-PROGDESIGN code phase (quest/waypoint state)
```
Soft dependencies can start in parallel against stubs; only WP-DASH should wait for the WP-SERVER route list (one hour of agreement, not the code).

## Recommended order

Batch 1 (start now, 5 agents, all offline): WP-CI, WP-MEMFD, WP-LIFECYCLE, WP-SAVES, WP-PROGDESIGN.
Reason: they remove the blocker-level unknowns (F1, F2, F3, F4) and nothing running depends on them.

Batch 2 (as soon as a slot frees): WP-STATE, WP-REPLAY, WP-EVENT, WP-CONTAINER.

Batch 3 (after wave 3 merges, so APIs are stable): WP-SERVER, WP-OBS, WP-VIEWPORT, WP-DATA2, then WP-DASH and WP-SETUP.

Main-agent tasks that are not packages (they touch running agents' directories, do them at merge time):
1. Drop `levelerlive` from the LEVELER branch, fix `.gitignore` (F17).
2. Deduplicate plan data into one package (F15).
3. Unify UI flag semantics in internal/d2 and remove copies in input/town/sentinel (F9).
4. Add ctx and Clock to `input.Controller` and fix `TestFailsafeDeathOverlayNeverDismissed` (F4, F6).
5. Run the two 10-minute live checks: managed mode start with `--managed` on a free instance; map id stability across Save and Exit (F1, F10).

## Capacity note
The worker host load is 30-40 on 16 cores from Tdarr and ComfyUI (V memory file) and each instance costs ~1 core (V docs/multi.md). Every
package above runs on the head box or a lab worker and needs no instance, so batch 1 adds no load to the worker host.
