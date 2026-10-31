# Setup: doctor checks and wizard backend (WP-SETUP)

> status: Wine-era setup wizard and doctor; d2d is the primary backend (ADR 0009, 2026-10-30; docs/plan/STATUS.md), Wine container work paused. Kept for the frozen Game.exe dev path.

Code: `internal/setup` (checks, wizard, steps, HTTP), `cmd/d2doctor`. Tags: V verified, S secondary, U unverified.
Rules: nothing Blizzard owns is downloaded or bundled (the 1.14d patch and installers are paths the user supplies); CD keys are
typed in the installer window and never stored or logged (no input field, unknown JSON fields rejected, log lines scrubbed by
`setup.Redact`). V: offline tests with fakes (`go test ./internal/setup ./cmd/d2doctor`); the subreaper probe also ran for real
on the head box (Yama 1) and passed.

## d2doctor
`d2doctor [--json] [--data DIR] [--game DIR] [--slots N] [--addr H:P] [--pre-install] [--quick]`. Exit 0 all pass, 1 warnings,
2 any failure, 3 usage/internal. `--pre-install` turns missing-game results into warnings. `--quick` skips Slow checks.
A binary that wants the probe must call `setup.MaybeProbe()` first in main (the probe re-executes the binary).

Check ids (`results[].id`; status pass|warn|fail|skip; `fix` is a human hint with commands):
host.container, host.runner (GE-Proton11-11 = pass, other Proton or plain Wine = warn, none = fail), host.wine32,
kernel.yama (0/1 pass, 2 fail unless CAP_SYS_PTRACE, 3 fail), kernel.subreaper (slow: spawns a probe that becomes subreaper,
adopts an orphan and reads its /proc/pid/mem), display.xvfb, resources.disk (8 GB + 0.25 GB/slot, U), resources.cpu and
resources.ram (cgroup-aware; 1 core and about 0.3 GB per slot, V docs/multi.md), fs.prefix (NTFS/exFAT/FAT fail, noexec fail,
network fs warn), net.ports (UI addr, helper ports 47012+n, 4000 for party), helper.d2input, game.files, game.version (file version
1.14.3.71 plus sha256 against `setup.KnownGameExe`: known hash pass, right version unknown hash warn, other version fail; the one
pinned hash is the worker host Game.exe, V), game.mpq (slow), data.extract (slow, real extraction into a temp dir).
Added by DOCTOR (V: offline tests in internal/setup/doctor_test.go, real temp dirs plus fakes; `d2doctor --instance iN` picks the
game for char.difficulty): config.parse (config.json, setup.json, per-slot goals/runtime/deaths/routes must be valid JSON, absent is
fine), instance.paths (per planned slot iN: prefix, save dir, instance dir; missing warns, a file instead of a dir fails), data.tables
(cache/data non-empty; warn pre-install), helper.reach (dial 127.0.0.1:47012+N, closed = warn), tokens.perms (every file in
<data>/tokens must have no group/other bits, else fail with the chmod), resources.load (1-min load vs cores: warn when free cores <
slots, fail above 1.5x cores), clock.sanity (year < 2024 fail, data dir mtime in the future warn), char.difficulty (`d2.DoctorLine`
of the attached game, skip when no game; area mlvl is passed as 0, so no gap yet, U). Game.exe hash is the existing game.version.
New Env hooks: Dial, Now, Player (nil = not tested).
Checks with `needs` are skipped when a game.* prerequisite is not passing.

## Wizard (resumable)
State persisted with `internal/store` at a file the caller chooses (suggest `<data>/setup.json`). Steps, in order, ids stable:
`data-dir`, `sources`, `prefix`, `install-d2`, `install-lod` (not in game-dir mode), `import-game` (game-dir mode only), `patch`,
`extract`, `character`. Every step has a Verify: a satisfied step is marked done without running (so a pre-existing install skips
ahead). A crash mid-run returns running steps to pending. Modes: `stubs` (user's Blizzard downloader stubs, run in Wine),
`installer-dirs` (folders already downloaded; stubs optional, used to verify torrent piece hashes), `game-dir` (existing install,
copied in; symlinks refused). Installer and patch steps are interactive: status `waiting-user`, the user acts in the window
(shown through the viewport/noVNC on `display`).

## HTTP (mount `setup.API.Handler()` at `/api/setup/`)
All JSON. Errors are `{"error": "..."}`.
- `GET /state` -> `{inputs, steps:[{id,title,interactive,status,error,note,attempts,started,finished}], current, running, done, display, log[], updated}`. Status: pending|running|waiting-user|done|failed|skipped.
- `PUT /inputs` body (all optional, empty fields keep old values): `{dataDir, mode, stubD2, stubLoD, installerD2Dir, installerLoDDir, gameDirSrc, patchFile, slots, charName, charClass, profile, goals}` (`profile` = handicap profile of the first bot slot, one of bot.Profiles: optimal, relaxed, casual, sloppy, rookie, 400 otherwise; `goals` = comma separated presets `campaign:TEMPLATE`, `levelto:N` (2 to 99), `farm:AREA`, 400 if invalid; both validated by fleet.Spec.Validate). Absolute paths only. 400 on unknown fields, 409 while running. Returns the state.
- `POST /start` runs in the background until done, failed, cancelled (202; 409 if running). Poll `/state`.
- `POST /cancel` (202). `POST /reset[?step=ID]` forgets one step or all (409 while running).
- `GET /checks[?quick=1][&pre=1]` -> `{status, exit, counts:{pass,warn,fail,skip}, results[]}`; `pre=1` = game not installed yet.
`API{W, Env, Opts, BaseCtx}`: `Opts` returns doctor options from current inputs (DataDir, Slots).

## Wiring
`w, _ := setup.New(path, setup.NewSteps(setup.NewDeps(setup.NewEnv())))`; set `w.Display`. The process running `character`
must have called `launch.BecomeSubreaper()`. Extraction target: `setup.TablesDirFor` = `paths.Layout.DataTables(schema, first 12 hex of Patch_D2 sha256)`; other code should call that function.

## Live checks for the main agent (none were run)
1. `labrun worker -- d2doctor --pre-install` on the worker host and a clean container: statuses plausible; real probe with Wine present.
2. Wizard, `game-dir` mode, with the worker host's installed folder: does the game start from a copied folder without the installer's registry keys (InstallPath, U), and does `patch` stay skipped.
3. Wizard, `installer-dirs` or `stubs` mode on a scratch data dir under Xvfb + noVNC: installer window appears, `waiting-user`, completion detected from the install log (same verifiers as docs/launch.md 4).
4. `patch` step with the user's LODPatch_114d.exe: sha256 gate accepts it, window runs, version flips to 1.14.3.71.
5. `character` step: Helper+Game+Flow.CreateCharacter+LeaveGame end to end (composed from verified parts, never run together), then `Setupbot.d2s` exists in `cfg.SaveDir()`.
6. Run d2doctor on a Yama-2 host and an NTFS data dir to confirm fix texts.

## Server wiring and the character hold (WIZARD)
`internal/server.Config.Setup` (an `http.Handler`, normally `(&setup.API{...}).Handler()`) is mounted at `/api/setup/` behind the
server token; `/setup.html` and `setup.js` come from `Config.Assets` (`web.FS` embeds them). V: httptest `TestSetupMountAndPage`.
`Wizard.HoldForInput` (set it for the web UI, leave false for the CLI): `Start` stops cleanly (no failure, run ends, step stays
pending with a note) before a step whose `StepDef.Hold` returns text; only `character` holds, while `charName` is empty. The UI then
shows the name/class form, `PUT /inputs`, `POST /start` continues. V: `TestDefaultStepsGameDirMode`. Wired (WIZARDWIRE): `setup.NewWeb(setup.WebConfig{DataDir, Display})` builds the wizard on `<data>/setup.json` with
`HoldForInput=true` and the real deps; `Web.Handler()` goes to `server.Config.Setup`, `Web.Ready()`/`WaitReady(ctx)` = every step
incl. the character is done. `d2fleet serve -setup` (or `D2FLEET_SETUP=1`, display from `-setup-display`/`$DISPLAY`) calls
`launch.BecomeSubreaper()`, serves `/setup.html` (it now also serves `web.FS`) and restores the slots only after `WaitReady`, so no
bot runs before a charName exists and the character step is done. `d2dash -setup-data DIR` serves the same wizard next to the
simulated bots (UI trial, bots not gated). Both call `setup.MaybeProbe()` first in main. V: `TestWebEndToEnd` (httptest through
`server.New`, fresh data dir, game-dir mode with fake host effects: GET /setup.html, PUT /inputs, POST /start holds at the character
step, PUT charName, POST /start, done; a `cdKey` field is a 400 and `setup.json` has no key material).
Not done: `cmd/d2bot --managed` (not owned here) does not take the wizard; to gate it, build `NewWeb` in `serve.go` and call
`WaitReady` before `openGame`. The container image (`deploy/container/Dockerfile`) does not yet contain `d2fleet`, `d2doctor`
or a noVNC viewport.
