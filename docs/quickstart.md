# Quickstart: run D.idle on your own machine (agent USERDOC, 2026-10-29)

> status: STALE for the Wine path: user docs written before the container/d2d decision; steps unproven on a stranger machine. Wine container work is PAUSED (docs/decisions.md); d2d is the primary backend (ADR 0009, 2026-10-30), see docs/d2d-container/ and docs/d2d-backend.md.

For a user, not a developer. Target: classic Diablo II Lord of Destruction **1.14d** (file version 1.14.3.71), offline single player, softcore,
Linux. Not D2R. Tags: V = verified (a doc records a dated live run, named below), S = secondary, U = unverified (read in code or
written from design, never run end to end). Every flag below was checked against the `flag` definitions in `cmd/*/main.go` (V, read 2026-10-29).
Problems: [troubleshooting.md](troubleshooting.md). Reference for every knob: [knobs.md](knobs.md).

## 0. What this is and is not
- It is a bot that plays one character of a game YOU run on your machine: it reads the game's memory and sends mouse/keyboard messages through a
  small helper (`d2input.exe`) running inside the same Wine prefix. Nothing is injected into the game files and nothing talks to the network (V
  docs/container.md ran with `--network none`).
- **Offline single player only.** It never uses Battle.net or chat, and the repo rule is that nothing that talks to Battle.net is added
  (AGENTS.md). Only softcore, expansion characters are exercised; Hardcore exists as a switch but is U ([knobs.md](knobs.md) `hardcore`).
- **You must own Diablo II and Lord of Destruction.** Nothing Blizzard owns is shipped, downloaded or mirrored by this project. You supply your own
  install folder (and, if it is not yet 1.14d, your own `LODPatch_114d.exe`). The setup code never stores a CD key (no input field exists; unknown
  JSON fields are rejected; V docs/setup.md offline tests).
- **Honest state of the bot** (docs/INDEX.md): Normal Act 1 levelling and quests work live for a Sorceress (about clvl 1 to 13, 90+ min unattended
  runs, V docs/launch.md s9); other classes ran only short smoke runs; Acts 2 to 5 and Nightmare/Hell exist partly or only as offline code. Set
  expectations accordingly: this is a work in progress, not a finished product.

## 1. Prerequisites
| need | detail | tag |
|---|---|---|
| A legally owned D2 + LoD, patched to 1.14d | `Game.exe` file version 1.14.3.71 (`d2doctor` checks it and its sha256) | V docs/setup.md |
| Linux x86_64, kernel Yama `ptrace_scope` 0 or 1 | scope 1 is fine: the bot is the game's parent (subreaper), so it needs no capability. Scope 2 is refused by `d2doctor`; 3 fails | V docs/launch.md s1, docs/container.md |
| Wine or Proton able to run 32-bit Windows | native path: **GE-Proton11-11** is the only runner verified for D2 (found in `compatibilitytools.d` under `~/.steam/root`, `~/.local/share/Steam`, `/usr/share/steam`; env `PROTON` overrides). Container path: plain Debian Wine inside the image (V docs/container.md) | V |
| Xvfb (headless) or a desktop session | the game is started `-w -ns` (windowed, muted). A hidden game needs Xvfb; on a desktop the window just appears | V docs/oog.md, multi.md |
| CPU, RAM, disk | about 1 core and 0.3 GB RAM per running game (Game.exe busy-loops one core even idle); 8 GB free plus 0.25 GB per slot for the data dir (the disk figure is U, `d2doctor`'s default); prefix on a real Unix filesystem, NOT NTFS/exFAT/FAT and not `noexec` | V docs/multi.md, container.md; U disk figure |
| Go toolchain (to build) | Go 1.27 or newer, standard library only; no dependency downloads | V AGENTS.md |
| Your Diablo II folder | about 1.9 GB, 24 files for a patched install (V docs/container.md) | V |
| Docker (container path only) | Docker 29.x was tested; podman, NAS and LXC hosts were not | V/U docs/container.md |

Build the binaries (any Linux box with Go; the helper is a Windows 386 exe built with `-H=windowsgui` so it opens no console window over the game, V docs/multi.md note):
```
git clone <this repo> && cd d2lod-bot
go build -o bin/d2bot    ./cmd/d2bot
go build -o bin/d2fleet  ./cmd/d2fleet      # wizard + multi-slot supervisor
go build -o bin/d2doctor ./cmd/d2doctor     # pre-flight checks
GOOS=windows GOARCH=386 go build -ldflags "-H=windowsgui" -o bin/d2input.exe ./cmd/d2input
packaging/release.sh                         # or: one tarball with the linux binaries and d2input.exe (note: it does not build d2fleet or d2doctor, U)
```
Run `bin/d2doctor --pre-install --json` first. Exit 0 = all pass, 1 = warnings, 2 = a failure (fix text is printed per check), 3 = usage error
(V cmd/d2doctor/main.go header). `--game DIR`, `--data DIR`, `--slots N` narrow it; `--quick` skips the slow checks.

## 1b. Extract the gear tables from your own game files
The bot's equip and shop logic needs a few game tables (item bases, skill prerequisites, armor). They are not shipped: run `go run ./cmd/d2tables --data <dir with d2data.mpq, d2exp.mpq, Patch_D2.mpq>` once (or set `D2_DATA_DIR` and the bot extracts them on first start). The result goes to `~/.cache/d2lod/tables/` (`D2_TABLES_DIR` overrides), stamped with the sizes and hashes of your MPQs; rerun after patching the game (it skips when current, `--force` redoes it). Without the cache the bot stops with "run d2tables first". The tables stay on your machine and are never committed (U: not yet tried by a stranger).

## 2. Way A: the container with the setup wizard (recommended; wizard UI is not yet run for real)
One container = one bot host with several character slots; no Docker socket, no capabilities (V docs/container.md decision). The container holds
plain Wine, Xvfb, `d2bot`, `d2fleet`, `d2input.exe`. Status: the game and `d2bot --managed` inside the container are **V** (20 min run, 63 kills, 0
deaths, 2026-10-28/29); the **wizard in the container has only been rehearsed offline** (23 checks pass against a fake host, V
`tools/container-checklist.sh`) and never run with real docker: treat steps 2 to 4 as U until someone does it.

1. Build the image (on a machine with Docker; the build needs network, running does not):
   ```
   tools/container-build.sh            # builds d2idle:dev, about 1.2 GB
   docker volume create d2data
   ```
2. Start it. `/src` is your own Diablo II folder (read-only), `/patch` a folder holding `LODPatch_114d.exe` (only needed if the folder is not 1.14d):
   ```
   docker run -d --name d2idle --stop-timeout 90 -p 127.0.0.1:8431:8431 \
     --cap-drop ALL --security-opt no-new-privileges:true --shm-size 256m --cpus 2 --memory 3g \
     -v d2data:/data -v "/path/to/Diablo II":/src:ro -v /path/to/patch:/patch:ro d2idle:dev
   docker exec d2idle cat /data/token
   ```
   (the same as `deploy/container/compose.example.yml`; `stop_grace_period: 90s` / `--stop-timeout 90` matters, see section 7). The default command
   is `d2start.sh`, which makes `/data/token` once and runs `d2fleet serve -setup -addr 0.0.0.0:8431 -data /data -token-file /data/token`.
3. Open `http://127.0.0.1:8431/setup.html?token=TOKEN`. Choose mode `game-dir`, game path `/src`, patch file `/patch/LODPatch_114d.exe`, Save, Start.
   Steps run in order: data-dir, sources, prefix (about 12 s), import-game (copies about 1.9 GB into `/data`; your folder is never written), patch
   (skipped when already 1.14.3.71), extract (game data tables), character.
   The other modes (`stubs`, `installer-dirs`) run Blizzard's own installers inside Wine and need a window you cannot yet see in the container
   (no noVNC; docs/container.md "Still missing"). **Use `game-dir`.**
4. The wizard stops at `character` on purpose. Enter a name (letters only) and a class, optionally the handicap profile and goals (sections 4 and 5), Save,
   Start. It creates the character in the game (about 25 s), then the slot starts playing. Bots begin only after the wizard is done.
5. Watch it (section 6); stop it with `docker stop d2idle` (it waits up to 90 s so each bot can Save and Exit; V container stop took 4 s when the bot
   was healthy, 4.1 s measured while playing).

Add more characters later without the wizard (a running `d2fleet serve`; token from `$D2FLEET_TOKEN`, URL from `-url` or `$D2FLEET_URL`, default
`http://127.0.0.1:8431/api/v1/fleet-admin`):
```
d2fleet add Mykara -class sorceress -profile casual -goals campaign:normal_act1,levelto:20
d2fleet list      d2fleet status      d2fleet restart i1 "reason"      d2fleet remove i1 [-force]
```
`add` takes `-class -role -profile -goals -build -difficulty normal|nightmare|hell -hc`. Slot ids are `i1..i9`. NOTE (V docs/container.md): d2bot only
ENTERS an existing character; the wizard (or `d2container mkchar`) creates it, so `d2fleet add` of a name that does not exist yet does not create it (U for the exact failure).

Dev shortcut that WAS run (V): `tools/container-run.sh cg-mk --rm -- run -- d2container mkchar -name Contai -class sorceress` once per character, then
`tools/container-run.sh cg-bot -d -- run -- d2bot --managed --char Contai --autostart --goals reach:8 --stop-game --run-for 20m` (needs `tools/container-game-copy.sh` first).

## 3. Way B: native `d2bot --managed` (what runs on the worker host today; V for the bot, hand-assembled for the rest)
`--managed` means d2bot itself launches Wine/Proton, the helper and Game.exe, becomes their subreaper, reads memory and sends input, with **no
capability and no setcap** (V docs/launch.md s9: CapEff=0, ptrace_scope=1, 90+ minutes). The project's own dev scripts (`tools/live.sh`,
`tools/game.sh`) are tied to the worker host (ssh, `worker-sudo`, instance table in `tools/instances.conf`): they are the **developer** route, not for you. A user
runs the binary directly. Layout it expects with no `--instance` (V cmd/d2bot/game.go `managedConfig`: root = `--data`):
```
<data>/                       default ~/.local/share/d2bot   (env D2BOT_DATA)
  proton/pfx/...              Wine prefix; Proton builds it on first run (about 25 s)
  proton/pfx/drive_c/Program Files (x86)/Diablo II/    the patched game (copy or install it here)
  helper/d2input.exe          the helper you built above
  logs/  instances/<id>/      goals.json, deaths.json, logs
```
Steps (U as a whole for a machine that is not the worker host: each part is V, the hand-assembly is not):
1. Put the GE-Proton11-11 folder in `~/.steam/root/compatibilitytools.d/` (or export `PROTON=/path/to/proton`).
2. Make the prefix and the game dir: run the Proton prefix once (any `proton run`), copy your patched Diablo II folder to the path above; registry values
   (`NewSavePath`, `InstallPath`, `GamePath`) are set by the container's `deploy/container/d2init.sh` for plain Wine; on the worker host the prefix came from a normal
   install (V docs/launch.md s5). Whether a plain copy launches without install registry keys on Proton is U (docs/setup.md "Live checks" item 2).
3. Create a character once **by hand**: start the game yourself in that prefix, pick Single Player, Create, an expansion softcore character; Save and Exit; quit the
   game. (The bot never creates characters. A `cmd/oog` driver exists, `oog create NAME CLASS`, but it needs the dev ptrace setup; the container wizard is the supported creation path.)
4. Run:
   ```
   bin/d2bot --managed --char Mykara --goals 'campaign:normal_act1' --autostart --stop-game
   ```
   Expected log line `listening addr=127.0.0.1:8420 dashboard=http://127.0.0.1:8420/`. Without `--goals`/`--autostart` the bot idles until you press Start
   in the dashboard (V docs/launch.md s9: saved goals start only after `POST /control/start` or `--autostart`).
   Useful flags (all V, cmd/d2bot/main.go): `--data DIR`, `--config FILE`, `--addr HOST:PORT` (non-loopback needs `--token`), `--instance ID`,
   `--gamedir DIR`, `--log-level debug|info|warn|error`, `--run-for 2h` (stop gracefully after), `--stop-game` (also close the game on exit),
   `--supervise` / `--supervise-max N` (restart d2bot itself with backoff, cleaning the old game first), `--clean-start`, `--soak-dir`. Dev only: `--sim` (no game at all,
   handy to look at the dashboard), `--dry` (read-only: print what the bot sees, sends no input), `--attach` (existing game; needs ptrace rights).
5. For several characters at once give each its own prefix, Xvfb display and helper port; that is what `tools/game.sh` and `d2fleet serve` do (instances `i1..i9`,
   helper port 47012+N, display `:10N`; V docs/multi.md). A second Game.exe in the SAME prefix refuses to run ("Only one copy of Diablo II may run at a time").
6. Hardware cap: each game is about one core. On a busy machine the bot is slower and click-timing mistakes rise (V docs/archive/review/gotchas.md 17).

Unattended on a server: wrap step 4 in a systemd user unit with `--supervise` and `TimeoutStopSec=90`.

## 4. Characters, handicap profiles and game difficulty
- Class: any of the seven can be created; **Sorceress is the reference class**. Paladin, Necromancer, Amazon, Assassin and Druid ran short live
  smoke tests; Barbarian has research and a build file but no live run (docs/INDEX.md "Per class").
- Handicap profile ("how well should it play?"): `optimal` (default, every knob at its best), `relaxed`, `casual`, `sloppy`, `rookie` (the worst setting where
  the bot still survives; V internal/bot/profile.go). A profile only changes reaction, detours, town-trip frequency and similar knobs (V docs/knobs.md
  s3). Set it by `d2fleet add -profile NAME`, the wizard's "How well should it play?" select, `"profile"` in `config.json`, or at run time
  `PUT /api/v1/instances/{id}/knobs` with `{"profile":"casual"}`. Own knob overrides: `"tuning": {"bot.clearRadius": 20}`. Unknown names are errors. Does a profile make the
  bot dumber without ever dying more? Not measured (U).
- Difficulty: campaigns select it themselves (`normal_*`, `nightmare_all`, `hell_all`); `PUT /settings` has a `difficulty` field. Nightmare/Hell paths are offline-tested only (U).

## 5. Choosing goals
Goals are a queue the bot works through in order. Give them on the command line (`--goals 'A;B;C'`, `;`-separated, parsed in `cmd/d2bot/util.go`
`enqueueGoals`) or in the dashboard/API (`POST /api/v1/goals`). Via the wizard or `d2fleet add -goals`, the form is comma separated and `levelto` is
translated (V `internal/fleet/spec.go GoalsArg`).

| you want | d2bot `--goals` | wizard / d2fleet `-goals` | notes |
|---|---|---|---|
| play the story | `campaign:normal_act1` (also `normal_act2..5`, `normal_all`, `nightmare_all`, `hell_all`) | `campaign:normal_act1` ... | Act 1 is the best-tested; Acts 2+ are partial (docs/INDEX.md) |
| reach a level | `reach:30` | `levelto:30` (2 to 99) | picks areas from the progression plan by character level; re-plans on level-up (V docs/bot.md) |
| farm one area | `farm:Blood Moor` | `farm:Blood_Moor` (`_` = space) | endless unless the API goal has limits (kills, minutes, drops) |
| boss runs | `runs:Mephisto:10`, `runs:Eldritch_Shenk:5`, `runs:auto:20` | not offered by the wizard | `_` = space; `runs:auto` lets the farm scheduler pick; **U: offline tests only, never run live** (docs/farm.md) |
| one boss | `boss:Andariel` | not offered | counts kills; scripted boss fights are partial |
| rest | `idle` | | rests in town |

**Chain on/off.** A `campaign:...` started from the command line chains the next difficulty when it finishes (the `*_all` templates: Normal then
Nightmare then Hell). Switch that off with `campaign:normal_all:nochain` (V `enqueueGoals`; chain mechanics in docs/nightmare-campaign.md). The
wizard's campaign presets are passed as `campaign:T` with no suffix, so they chain (V by reading `GoalsArg`). The dashboard goal form: chain
control is not visible in the UI code I searched (U); use the CLI suffix.
Goals persist across restarts in `<data>/instances/<id>/goals.json`. `--goals` is APPENDED every start; the fleet therefore passes it only on the first
launch of a character and goal string (V docs/container.md "Bot launcher"). Do not put the same `--goals` in a restart script (duplicates), or clear the queue in the dashboard first.
Autostart: `--autostart` or config `"autostart": true`; or `PUT /settings` `autoResume: true`.

## 6. Watching it
- **Dashboard** (the bot's own page): `http://127.0.0.1:8420/` for `d2bot` (change with `--addr`; a non-loopback address needs `--token`, which you pass once as
  `?token=`); `http://127.0.0.1:8431/` for `d2fleet serve` (the fleet strip shows each slot's `bot.state`, restarts, last error). It shows status, activity, goals, XP/h, deaths and a
  stats API (`GET /api/v1/status`, `/events`, `/stats`, docs/api.md). Controls: Start / Pause / Stop (`POST /control/start|pause|stop`).
  `d2dash` runs the same page against simulated bots without a game: `bin/d2dash -addr 127.0.0.1:8420`.
- **Logs**: container `docker logs d2idle`; fleet `<data>/instances/iN/logs/d2bot.log`; native stdout/`--log-level debug`.
- **Pictures of the game** (the game runs hidden): `d2watch` (view-only viewer on port 8890, `-addr :8890`, token-protected, V docs/watch.md): it captures only the game
  window with ImageMagick `import` + `xdotool`, so it needs those and it is written for the dev layout (`tools/instances.conf`, instances `i1..iN`, displays `:100+N`);
  on a plain native or desktop setup it may show nothing (U). `tools/watch.sh url` prints its URL on the worker host. Container: V one frame by
  `docker exec d2idle sh -c 'xwd -root -silent -display :99 -out /tmp/f.xwd'; docker cp d2idle:/tmp/f.xwd .; magick f.xwd frame.png` (display `:99` is the d2container direct run of docs/container.md; a fleet slot uses its own `:10N`, so use `-display :101` for slot i1, U). A real window you can click in a browser (noVNC/Xpra) does not exist yet. Do not take screenshots of a desktop that holds unrelated windows.
- Health in one call: `GET /api/v1/bot/health` (state, stuck reason, ladder step, paused, selfPaused, fatal).

## 7. Stopping safely (Save and Exit timing)
The character is only saved with its latest quests and waypoints when the game does Save and Exit; killing the process mid-game loses everything since the last save.
- **Native**: send **SIGTERM** (Ctrl-C) to d2bot started with `--stop-game`. It lets the bot finish its current step and walk home, then clicks Save and Exit, then
  stops the game; healthy case about 2 s for the exit itself (V docs/launch.md s9). The bot waits up to `stopTimeoutSec` (60 by default) for the walk to town; if it cannot get to town
  it saves where it stands and the next start begins in town anyway. Give it up to ~50 to 70 s before any SIGKILL (the fleet's own grace is 70 s and the
  container needs 90, V docs/container.md).
- **Container**: `docker stop d2idle` (it was set to wait 90 s); never `docker kill` and never `docker rm -f` a running one.
- Do NOT SIGKILL d2bot while a game runs: the game then outlives it, is unreadable ("not our descendant") and must be stopped by hand before the next run
  (V docs/launch.md s9). With `--supervise`, a crashed d2bot cleans up the old game first.
- Do not close the game window yourself while the bot runs; the bot treats an unplanned exit as a crash and restarts the game (up to `maxFailsafeRestarts`, 3 without progress; V docs/stallfatal.md).
- A character left in a game that did not Save and Exit is simply loaded from its last save next time; you lose progress since then, not the character (S, standard D2 behaviour).
- A pause is not a stop: Pause lets the current action reach a safe moment; Stop returns the active goal to the queue (docs/api.md).

## 8. What the bot will and will not do
- WILL: fight, loot (by built-in rules), drink potions, go to town, repair, buy, stash, spend stat/skill points by build, take quests, flee/chicken at low life, recover from
  deaths (corpse retrieval only when worth it, never in hardcore), re-enter the game through the menus, restart a hung game, pause itself and try again later (section troubleshooting P).
- WILL NOT: touch Battle.net, chat, realm characters, other windows on your desktop, or anything outside the game's memory, its helper and the data dir. It never creates or deletes a character.
- Offline only. `internal/cheat` may edit memory or save files of a small allow-list of **throwaway test characters only** (Testy, Botty, Navi, Combi, Towni, Booti, Multi and the
  smoke-test names); any other name, and any game with a server address or account in memory, is refused, and every attempt is logged in `<data>/ledger/cheats.jsonl`
  (V docs/cheats.md). So your own characters are never cheated by the tooling. There is no cheat option in the wizard or the dashboard.
- Known weaknesses you may see (all in the docs, still true as of the last merge): low-life loops with no potions and no town portal; Acts 2+ unproven; mercenaries unhandled; first visit of
  an area can stall a while (troubleshooting S1).

## 9. Updating, cleanup
`git pull`, rebuild, restart. State files (goals, deaths, saves) are in the data dir and survive; the game files you supplied are never modified in container mode (read-only `/src`, copied to
`/data`). To remove everything: `docker rm d2idle; docker volume rm d2data` (deletes the character saves too; copy `Saved Games/Diablo II` out of the volume first) and delete the copied game files.
