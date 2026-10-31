# Real Diablo II 1.14d headless in the container (agent CONTAINERGAME, VERIFIED 2026-10-28/29 on the worker host)

> status: STALE: Wine container work is PAUSED (docs/decisions.md); d2d is the primary backend (ADR 0009, 2026-10-30), see docs/d2d-container/ and docs/d2d-backend.md.

Result: the real game runs under **plain Debian Wine** (no Proton) in the `d2idle:dev` image, on Xvfb, with `d2bot --managed`
reading its memory with **CapEff=0 and Yama ptrace_scope=1** (no `SYS_PTRACE`, no setcap). A fresh Sorceress "Contai" was created
through the OOG flow inside the container and played for 20 minutes without a death. Tags: V = measured in this run.

## Build and run (all on a lab worker with Docker, here the worker host, Docker 29.9)
```
labrun worker -- tools/container-build.sh                  # d2idle:dev; cross-compiles d2container, d2bot, d2input.exe (windows/386)
labrun worker -- tools/container-game-copy.sh              # copy of YOUR game dir -> ~/d2lod-container-data/game (source untouched)
docker volume create d2cg-data
tools/container-run.sh cg-init --rm --entrypoint /usr/local/bin/d2init.sh --          # once: prefix, registry, game link (12 s)
tools/container-run.sh cg-mk --rm -- run -- d2container mkchar -name Contai -class sorceress   # once per character (25 s)
tools/container-run.sh cg-bot -d -- run -- d2bot --managed --char Contai --autostart --goals reach:8 --stop-game --run-for 20m
```
`container-run.sh` adds `--network none --cap-drop ALL --security-opt no-new-privileges --shm-size 256m --memory 3g --cpus 2` and the two mounts.
The image user is uid 1000 `d2`; nothing Blizzard owns is in the image or the repo.

## Volumes
| mount | content | notes |
|---|---|---|
| `/data` (named volume, owned by uid 1000) | `proton/pfx` Wine prefix, `logs/`, bot state, **characters** (`proton/pfx/drive_c/users/d2/Saved Games/Diablo II`) | prefix is created by `d2init.sh`; saves survived container removal (V) |
| `/game` (bind, **read-only**) | the user's Diablo II dir (1.9 GB, 24 files) | `d2init.sh` symlinks `Program Files (x86)/Diablo II` to it; the game ran fine read-only because `NewSavePath` points into the prefix (V) |

How plain Wine is used without touching Go: `internal/launch` runs `$PROTON run EXE`; the image sets `PROTON=/opt/d2/runner/proton`,
a 6-line shell shim that exports `WINEPREFIX=$STEAM_COMPAT_DATA_PATH/pfx` and execs `wine`; `runner/files/bin/wineserver` makes
`Config.WineserverKill` work. The helper is `D2INPUT_EXE=/opt/d2/d2input.exe`. `d2container mkchar` (new, cmd in
`deploy/container/entrypoint/mkchar.go`) does the OOG character creation as the subreaper, because `d2bot` itself never creates a character.
`d2init.sh` writes the D2 registry values (`NewSavePath` for the Wine user `d2`, `GamePath`, `InstallPath`, `Render=1`) and `Wine\Drivers Audio=""`.

## Numbers (V)
| what | value |
|---|---|
| image size | 1.18 GB (1,181,953,499 B); cold build 166 s on a loaded worker host, 8 s with the apt layer cached |
| prefix init | 12 s (wineboot + registry) |
| start to in game (existing character, `d2bot --managed --char`) | 14 s from `docker run` (game up 0.3 s, splash 10 s, menu, char select, in game); character creation run: 17.7 s to in game, 25 s container lifetime |
| whole container, `docker stats`, 37 samples over 20 min | CPU 102 % average (one core, Game.exe spins like on the host; limit was 2), memory 212 MiB at 1 min, 250 to 289 MiB later (peak 288.8 MiB incl. page cache) |
| RSS at 1 min (`/proc`) | Game.exe 68 MB, d2bot 37, d2input 28, explorer 26, winedevice 25+21, plugplay 21, services 21; 85 PIDs |
| memory read | OK: `CapEff=0000000000000000`, `CapPrm=0`, Yama 1, `--cap-drop ALL`; d2bot is the subreaper and Game.exe its descendant, snapshots, OOG screens and fighting all worked |
| 20 min run (goal reach_level 8, `--autostart`) | 63 kills (3.2/min including town chores), 1180 XP, level 2, 0 deaths, 6 potions, 4 drops, 0 errors, XP/h 2740, clean exit 0; `--stop-game` saved and exited in 2 s |
| `docker stop` while playing | 4.1 s, exit 0, state restored on the next start (kills 63 again), 0 zombies |
| host comparison | NOT controlled: no host instance was run (rules). docs/bot.md (older build, Blood Moor level 1) has 2.7 kills/min, 2700 XP/h; the container figure is the same order. Frame rate was not measured; the game window renders and animates (screenshots below), CPU use equals one core as on the host |

## Viewer (frame capture of the container's own Xvfb, never the desktop)
```
docker exec cg-bot sh -c 'xwd -root -silent -display :99 -out /tmp/f.xwd'; docker cp cg-bot:/tmp/f.xwd .; magick f.xwd frame.png
```
`xwd` is in the image (1.9 MB per frame); ImageMagick or `ffmpeg -i f.xwd` converts it outside. Frames from this run (V, 800x600 game): in Rogue
Encampment, in the Den of Evil with the sorceress fighting, and the main menu, were viewed and then deleted from the worker host (game art, not in the repo).

## Plain Wine vs GE-Proton11-11: differences found
1. **Window frame vs Xvfb size.** Wine draws a title bar; with the 800x600 Xvfb the "Diablo II" window is 800x580 at +3+29 and the bottom 20 px (belt, orbs' base) are cut off in a frame capture. Fix: Xvfb 832x640x24 gives a full 800x600 client at +16+33 (V, xwininfo). `d2container run` now defaults to 832x640x24. Input is by PostMessage in client coordinates, so the bot was not affected at 800x600. The host Xvfb for i1..i5 (`launch.XvfbDisplay`, 800x600x24) has the same clipping; bump it for viewer frames.
2. **ALSA noise.** Plain Wine's winealsa tries to open card 0 and logs about 20 lines per start into `game.out`. Fix: `HKCU\Software\Wine\Drivers Audio=""` (in `d2init.sh`); after it the log had no new lines (V). Sound is off anyway: `-ns`, Music Volume 0, no sound card in the container.
3. **No DXVK, no D3D.** Not needed: the stock DirectDraw path (`Render=1`) rendered correctly under Xvfb; there is no GPU in the container.
4. **Wine mono/gecko** skipped by `WINEDLLOVERRIDES=mscoree,mshtml=` (the spike, V); first prefix start is 12 s instead of Proton's ~25 s.
5. **Wine user is `d2`**, not `steamuser`: `NewSavePath` is `C:\users\d2\...`; a prefix copied from the host needs that path changed.
6. **No Steam runtime / pressure-vessel**: the Proton python launcher is gone, so the direct child of d2bot is `wine` itself; `Game.exe` still ends up a descendant (subreaper) and `/proc/pid/mem` works (V).
7. **Same behaviour as Proton** (V): the d2input helper (windows/386, `info` answers `ok hwnd=... client=800x600 visible=1`), PostMessage clicks and keys on every OOG screen (splash, main menu, create, char select, Save and Exit), `-w -ns` windowed and muted, hotkey binding, shopping, potions, town chores, portals.
No fundamental failure: the Proton-in-image fallback was not needed (it would add the Proton tree, several hundred MB, plus the Steam runtime U).

## Open problems
- Click hover failures: 2 in 20 min (`shop`: "pointer never hovered the target" at 02:46; Akara dialog missed at 02:59), the Den of Evil quest claim failed and two "recovering" re-entries happened (02:50, 02:57) with 0 deaths, cause not investigated. The same error is known on the host (docs/bot.md "Akara click failed once"), so there is no evidence it is Wine-specific, but a controlled A/B against Proton at the same level was not done (U).
- Hang and crash signals (`CPUTicks`, BlizzardError.exe) were not exercised in the container.
- Party play, several instances per container (own display and prefix per slot through `launch.Life`), `/dev/shm` pressure with N games, Yama scope 2/3 hosts, podman, the NAS/LXC: not tested.
- Character creation needs `d2container mkchar` (this package); `d2bot` has no create-on-missing. The wizard (milestone F) should call it or `Flow` with `Create` directly. A request for the Go owners: let `launch.Config` choose plain Wine (`Runner: wine`) so the shim is not needed.
- The prefix volume must be seeded with `d2init.sh`; it hard-codes the Wine user `d2` and `/game`.
- The worker host disk is 97 % full: the 1.2 GB image stays; the 1.9 GB game copy was deleted after the run (recreate with container-game-copy.sh).

## Cleanup done
Containers `cg-*` and volume `d2cg-data` (label `d2lod.container=1`) removed; only the image `d2idle:dev` stays.

## FIRST REAL RUN OF THE WIZARD IMAGE (agent CONTAINERRUN, 2026-10-29, the worker host, Docker 29.9; V = measured in this run)
The product path works end to end in the real container: image build, wizard on 127.0.0.1:8431, game-dir import, prefix, patch check,
extract, character creation, slot start, d2bot --managed in the container, memory attach, play, `docker stop`, restart.
Exact lines that worked (the data volume is a bind dir here; a named volume is the same; `-e D2FLEET_MAX_LOAD_FACTOR=-1` only because the worker host's
host load is 25+ and the container sees the HOST's load1):
```
labrun worker -- tools/container-build.sh        # 22 s warm, 140 s cold; image 1.21 GB (V)
docker run -d --name d2idle-test --stop-timeout 90 -p 127.0.0.1:8431:8431 --cap-drop ALL --security-opt no-new-privileges:true \
  --shm-size 256m --memory 6g --cpus 3 -e D2FLEET_MAX_LOAD_FACTOR=-1 \
  -v $HOME/d2idle-container-test/data:/data \
  -v "$HOME/Games/d2lod/proton/pfx/drive_c/Program Files (x86)/Diablo II":/src:ro -v $HOME/d2idle-container-test/patch:/patch:ro d2idle:dev
TOK=$(docker exec d2idle-test cat /data/token)
curl -H "Authorization: Bearer $TOK" -X PUT  http://127.0.0.1:8431/api/setup/inputs -H 'Content-Type: application/json' \
  -d '{"mode":"game-dir","gameDirSrc":"/src","patchFile":"/patch/LODPatch_114d.exe","charName":"Contest","charClass":"sorceress","goals":"levelto:3"}'
curl -H "Authorization: Bearer $TOK" -X POST http://127.0.0.1:8431/api/setup/start     # then poll GET /api/setup/state
```
No capability and no `SYS_PTRACE` were needed: `CapEff=0`, `NoNewPrivs=1`, host Yama ptrace_scope 1 (V). d2bot is the subreaper and the
game's ancestor (`d2bot --managed`, started by d2fleet per slot). `--cap-drop ALL` is enough. The /data volume must be writable by uid 1000
(the image user is now `steamuser`, uid 1000).
| step | result (V) |
|---|---|
| doctor `checks?pre=1` | pass: container, wine32, yama, subreaper (25 ms), xvfb, disk, cpu; warn: host.runner (plain Wine), sandbox (no bwrap), helper/instance paths pre-install; fail only `resources.load` (the worker host load 26 on 16 cores, host-wide) |
| prefix / import-game / patch / extract | 10 s / 10 s (1.9 GB copy into /data) / skipped, already 1.14.3.71 / 3 s |
| character step | stops with "choose a character name and class" until inputs are saved, then creates Contest (Sorceress) in about 35 s; wizard done on a fresh data dir in one pass |
| slot start | slot i1 appears after done, prefix clone by hardlink (rsync --link-dest), Xvfb :101, helper 47013, d2bot attached to the game in 1 s, in game 15 s after the slot start |
| play | 4 minutes: 19+ kills, level 1 to 2 (goal levelto:3 not reached in the window), CPU 104 % (one core), 244 MiB, 111 PIDs, a frame of the slot's Xvfb shows the sorceress in the Blood Moor |
| `docker stop` (stop-timeout 90) | 16.1 s, exit 0; log: SIGTERM to d2bot, "click save-and-exit", Contest.d2s written 1 s after; d2savecheck on a copy: level 2, exp 513 |
| `docker run` again on the same data | wizard done at once, slot i1 restored and playing in 40 s, same character, goals not re-seeded |

### Failures found and fixed (each a commit on agent/containerrun)
1. Wizard character step: `d2input.exe missing at /data/helper/d2input.exe`. The image ships it at /opt/d2/d2input.exe (`D2INPUT_EXE`); `internal/setup` ignored the env. Fixed (check.go).
2. `cannot read game memory ... game process not found`: on Debian trixie wine 10 the game's `/proc/pid/exe` is `/usr/lib/wine/wine`, not `wine-preloader`; `mem.FindGamesFor`, `Life.GamePIDs` and the input procmem scan only matched "preloader". New `mem.IsWineExe`. (The earlier CONTAINERGAME run predates the apt package version that dropped the preloader link or did not hit this path: U which.)
3. `verify: character save ... users/steamuser/... no such file`: Go code assumes the Proton user `steamuser`; the image user was `d2`. The image user (hence the Wine user) is now `steamuser`; `d2init.sh` detects the user and is unaffected.
4. Slot start refused: `admission refused (load): load average 21.6 ... on 3.0 cores`. A container sees the host's load1. New flag `d2fleet -max-load-factor` / env `D2FLEET_MAX_LOAD_FACTOR` (negative = off).
5. `rsync: executable file not found`: per-slot prefix clone needs rsync; added to the image.
6. `rsync: change_dir /data/prefixes/i0/proton failed`: the fleet cloned from the (nonexistent) main prefix; the wizard builds `/data/prefixes/base`. `Life.Base.PrefixDir` now points at the wizard base when there is no main prefix (d2fleet); MkPrefix uses it. Also the base `logs/` dir is created (Xvfb log).
7. d2bot `live play needs the game tables (data.Open failed)`: the image ENV `D2BOT_GAMEDIR` pointed at the legacy `/data/proton/pfx` game dir. Removed from the image (legacy d2init flow passes it with `-e`).
8. d2bot `character not in the list`: slot prefixes exclude saves, the wizard made the character in the base prefix. d2fleet now seeds `<Name>.*` from the base saves into the slot's save dir in Prepare (once, never overwrites).
9. `docker stop` emptied `fleet.json` (Fleet.Close removed every slot and persisted the empty list), so only the wizard's character came back after a restart. `Close` now keeps the specs (TestCloseKeepsSlotsForNextStart).

### What a stranger would still trip on (U unless noted)
Fixed after the run (agent CONTAINERFIX, offline tests, not re-run in a container: U):
- `resources.load` fail on a busy host: the doctor check now warns (never fails) inside a container (`/.dockerenv`, `/run/.containerenv`) or when `D2FLEET_MAX_LOAD_FACTOR` is negative, because a container reads the HOST's load1. The slot admission gate is separate: still set `D2FLEET_MAX_LOAD_FACTOR` if starts are refused.
- Zombies: d2fleet (and `d2dash -setup`) now run `launch.ZombieReaper`: every 5 s it wait4()s zombie children that were zombies on the previous scan too (never a blanket wait4(-1), which would steal exit statuses from its own exec.Cmd children). Unit and real-orphan tests.
- bubblewrap: `d2start.sh` sets `D2_SANDBOX=0` explicitly (override with `-e D2_SANDBOX=auto` if you add bwrap). The container is the sandbox (explicit mounts only); bwrap needs a user namespace, which `--cap-drop ALL` plus docker's default seccomp is expected to refuse (U, argv not run in a container), so it is not added to the image.
- `tools/container-run.sh` now uses the default bridge network and publishes `127.0.0.1:${UI_PORT:-8431}:8431` (`DOCKER_NETWORK=none` or `UI_PORT=` for no publish).
- The patch file: it was never required for an already patched install (the patch step's Verify passes before its Ready is checked; the UI marks the field optional in game-dir mode). A test (`TestDefaultStepsGameDirMode`, no patch file) now pins it, so `/patch` need not be mounted when Game.exe is 1.14.3.71. Unpatched installs still need it.
Still open:
- The wizard copies the whole game (1.9 GB) into /data and each slot clones it by hardlink: the bind dir and /data must be one filesystem for the links; a 6 GB `--memory` is generous (peak 245 MiB for one slot).
- The token is only in `/data/token`; the page needs `?token=`; the sandbox/doctor warnings for host.runner (plain Wine) are expected.
- Not exercised: goal completion (reach:3 not reached in 4 minutes), a second slot, Xvfb viewer in the UI, hang/crash signals, installer modes.

## User test checklist: first run through the setup wizard (item 7; nothing here has been run)
Prerequisites: the image now carries `d2fleet`, `d2doctor` and (embedded) the web assets (IMAGE; static checks only, no docker build
was run: U), `tools/container-build.sh` builds them, and the default command (`d2start.sh`) serves the wizard on 8431 and creates
`/data/token`. Exact lines (also in `deploy/container/README.md` and `compose.example.yml`, `stop_grace_period: 90s`):
```
labrun worker -- tools/container-build.sh && docker volume create d2data
docker run -d --name d2idle --stop-timeout 90 -p 127.0.0.1:8431:8431 --cap-drop ALL --security-opt no-new-privileges:true \
  --shm-size 256m --cpus 2 --memory 3g -v d2data:/data -v "/path/to/Diablo II":/src:ro -v /path/to/patch:/patch:ro d2idle:dev
docker exec d2idle cat /data/token      # step 1's token file is made by the image; step 2 is the container's default command
```
Still missing: a way to see the Xvfb display for installer windows (noVNC/Xpra, README "Not verified"); until then use game-dir
mode, which needs no window. Needs the user's own Diablo II dir and `LODPatch_114d.exe`; no CD key is needed in game-dir mode.
1. Volumes. `/data`: named volume owned by uid 1000, EMPTY (fresh; holds `setup.json`, prefix, saves, state). `/src`: read-only bind of
   your install folder (any path; the wizard copies it into `/data`). `/patch`: read-only bind with `LODPatch_114d.exe`.
   Publish only the UI port (8431), e.g. `-p 127.0.0.1:8431:8431`; other flags as `tools/container-run.sh` (cap-drop ALL, no-new-privileges, shm 256m).
   The UI needs a token if it is bound to a non-loopback address inside the container: `-token-file /data/token` (create it first, 20+ random chars).
2. Start: `d2fleet serve -setup -addr 0.0.0.0:8431 -data /data -token-file /data/token` (env `D2FLEET_SETUP=1` is the same).
   Expect on stdout `d2fleet: admin API on ...`; no slots start; `docker top` shows no Game.exe.
3. Open `http://127.0.0.1:8431/setup.html?token=...`. Expected: step list in order data-dir, sources, prefix, install-d2, install-lod,
   import-game, patch, extract, character; all pending; the doctor panel (`/api/setup/checks?pre=1`) shows host.runner as warn (plain
   Wine) or pass, kernel.yama pass, kernel.subreaper pass, game.* warn (pre-install), no fail.
4. Sources: choose mode `game-dir`, path `/src`, patch file `/patch/LODPatch_114d.exe`, save. Press Start.
   Expected: data-dir and sources done, prefix done (about 12 s), install-d2/install-lod show skipped, import-game done (copy of
   about 1.9 GB), patch skipped if the version is already 1.14.3.71 (else the sha256 gate, then `waiting-user` in the window), extract done.
5. Character hold. Expected: the run STOPS (not failed) at `character` with the note "choose a character name and class"; the UI shows the
   name and class form; `docker top` still shows no Game.exe and no bot. Enter a name (letters only) and class, save, press Start.
   Expected: the game starts under Xvfb, a character is created (about 25 s), the game exits, step done, wizard shows done,
   `/data/.../Saved Games/Diablo II/<Name>.d2s` exists.
6. Bots resume. Expected within about 1 s of done: `d2fleet list` shows the restored slots (none on a truly fresh volume; add one with
   `d2fleet add <Name> -class <class>`), the slot reaches in-game, the dashboard shows it.
7. Restart test: `docker restart`. Expected: wizard done at once (every Verify passes), slots restored without the wizard.
8. Negative checks: PUT `/api/setup/inputs` with a `cdKey` field returns 400; `grep -ri key /data/setup.json` finds no key; kill the
   container during `extract`, restart: the step resumes (note "interrupted; will resume").
9. Cleanup: `docker rm` the container, `docker volume rm` the data volume, delete the copies of game files you made on the host.

### Offline rehearsal and what needs you (WIZARD)
`tools/container-checklist.sh` runs steps 1 to 9 against the real `d2fleet serve -setup -fake -token-file` binary (wizard host
effects faked by `internal/setup/setuptest`, runtime starts no game): token required, step order, doctor endpoint, game-dir flow,
STOP at `character` with no slot, `cdKey` is a 400, name/class/handicap/goals saved, slot Wizbot appears only after done with that
class, handicap and goals, no key in `setup.json`, restart gives done at once and exactly one restored slot, simulated crash resumes. V: all
23 checks pass on the head box. Same flow as a Go test: `go test ./cmd/d2fleet -run TestWizardToManagedBot`.
Hand-off: after the wizard is done `d2fleet` restores saved slots, then adds a slot for the wizard's `charName` (class `charClass`,
handicap `profile`, goals `goals`, both wizard inputs of PUT `/api/setup/inputs`, validated by fleet.Spec.Validate) if none exists. `-fake` is test only.
The wizard UI (web/setup.js, character step) sends two things. `goals` from the "What should it work on?" picker: nothing (d2bot's own default), `campaign:normal_act1|normal_act2|normal_all`, `levelto:N`, `farm:AREA`. `profile` from the "How well should it play?" select: the bot.Profiles handicap names (optimal, relaxed, casual, sloppy, rookie). `fleet.Spec.GoalsArg()` translates goals into the d2bot `--goals` syntax (`campaign:T` stays, `levelto:N` becomes `reach:N`, `farm:AREA` becomes `farm:AREA` with `_` as space, joined with `;`), `Spec.BotArgs()` returns `--goals G --autostart`. V: unit tests (internal/fleet/spec_test.go) check the translation against the grammar read from cmd/d2bot enqueueGoals. The bot launcher (below) now starts d2bot per slot and applies the handicap profile. The farm area name is not checked against the area table (U).

### Bot launcher (internal/fleet/bot.go, offline-tested only)
`d2fleet serve` sets `Config.Bots = &fleet.ProcBots{...}` (off with `-no-bots`; `-fake` has none). Model (bot-owned game, replaces the earlier "fleet owns the game" attempt that Yama would refuse):
- For a slot with a bot the fleet starts only the prefix, the slot's Xvfb and the d2input helper (`fleet.Preparer`: `LifeRuntime.Prepare` = `Life.EnsureHelper`). It does NOT run `Runtime.Start`/`Login`. It then runs `d2bot --managed --stop-game --instance iN --char NAME --addr 127.0.0.1:0 --data ROOT --config ROOT/instances/iN/config.json [--goals G --autostart]`. d2bot becomes the subreaper, launches Game.exe as its child, attaches and walks the menus itself, which is the proven container layout (d2bot is the game's ancestor, so reads work with CapEff=0 and Yama ptrace_scope=1). The fleet waits (up to `Config.GameWait`, 3 min) for the game process, then the slot is `running`.
- A slot's class/create flag is not used in this mode: d2bot only enters an existing character (the wizard creates it). Without goals there is no `--autostart`.
- Stop paths all go through d2bot: Remove and Restart SIGTERM it (SIGKILL only after `--stop-grace`, default 70 s; d2bot needs at most 50 s; the container needs `stop_grace_period: 90s`); `--stop-game` makes d2bot save/leave and stop its game; the fleet then runs `Runtime.Stop` as cleanup (wineserver, Xvfb). Remove does no fleet-side OOG leave (the fleet holds no memory handle). If d2bot dies by itself, `ProcBots.CleanGame` (d2fleet: `LifeRuntime.StopGame`, Game.exe and wineserver only, Xvfb and helper stay) removes the orphaned game before the backoff restart, because a new d2bot that is not its ancestor could not read it.
- Old path: slots without bots (nil `Bots`, `-no-bots`) keep `Runtime.Start` + `Login` + `LeaveGame`. `ProcBots{Attach: true}` keeps the earlier attach model (fleet starts the game, no `--stop-game`; needs CAP_SYS_PTRACE or ptrace_scope 0).
- Port: d2bot's API listens on an ephemeral loopback port (`:0`), so there is no new fixed port; the address is read from d2bot's JSON "listening" log line and shown as `bot.addr` in the slot JSON. The game pid is read from the "managed game attached" log line (`bot.gamePid`), not polled from the API.
- Profile: `bot.Config.Profile` is a config key, so the launcher writes `instances/iN/config.json` before every start: the host `config.json` (or its `bot` section) with `profile` set, or removed for optimal. The host file is never modified.
- Goals: d2bot appends `--goals` on every start, so they are passed only on the first launch of a character and goal string (marker `instances/iN/goals-seeded`); restarts pass only `--autostart` and rely on the manager's saved `goals.json`. U: removing a slot and adding a different goal string for the same character appends to the old saved goals.
- Supervision: restart with backoff 2 s to 2 min (stable after 5 min), `MaxRestarts` 10 then `bot.state=failed`. Log: `instances/iN/logs/d2bot.log` (append, rotated once at 5 MB per start).
- Status: `View.Bot` = `{state starting|running|restarting|failed|stopped, pid, gamePid, restarts, lastError, since, addr, log}` in `/api/v1/fleet-admin/slots`; the dashboard fleet strip shows a d2bot problem as the slot error.
- V: unit tests with fake exec and fake runtime (internal/fleet `TestManaged*`, bot_test.go: args, config merge, goals once, backoff, give up, game pid, Restart/Remove SIGTERM the bot and no fleet start/login/leave), `-race`. U (needs the container): (1) a real `d2bot --managed --stop-game` under the fleet with a prepared prefix/Xvfb/helper launches and attaches (read from cmd/d2bot/game.go, not run; the Xvfb and helper that d2bot also ensures are already up, which is the "attach" branch). (2) d2bot reads the same data dir and prefix as the fleet. (3) SIGTERM to d2bot makes `--stop-game` save and exit within the 70 s grace in the container. Goals are marked seeded only after d2bot logs `goals enqueued`; the fleet only treats a missing game as dead after `GameWait` (3 min), so d2bot's own recycles are not slot restarts; d2bot gets `D2BOT_INSTANCE=<slot>`. Stop chain (V by unit test TestStopGraceChain): compose `stop_grace_period` 90 s > d2container waits 80 s for d2fleet after SIGTERM (`D2CONTAINER_GRACE` or `run -grace`) > d2fleet `-stop-grace` 70 s > d2bot SIGTERM worst case 50 s (`bot.ShutdownWorstCase`: settle 10, runner quiesce 2, Save and Exit 30, waits 2+2, game stop 4; SIGTERM saves from the field, no long walk home). Fleet.Close drains slots in parallel (max 8), so the grace is sized for one slot. Save and Exit waits up to 2 s for the cancelled runner to stop before opening the Esc menu.

### Dashboard, per-slot proxy and game view (FLEETVIEW)

Each d2bot listens on its own ephemeral loopback port, which nothing outside the container could reach, so the dashboard on 8431 showed "No Instances" while a slot played. `d2fleet serve` now passes `server.Config.Remotes` (cmd/d2fleet/remotes.go): every slot is a server instance (`id` = slot id, `title` = character).
- `GET /api/v1/instances` lists the slots; `/api/v1/instances/{id}/...` is a reverse proxy to the slot bot's API at its CURRENT `bot.addr` (looked up per request, so a restarted d2bot on a new port is followed at once; `health|deaths|state` map to the bot's `/bot/...`; SSE `events/stream` is flushed through; Authorization, Cookie and X-Token are not forwarded). While the bot has no address or does not answer: 503 `{"code":"bot_starting","error":"the bot of slot i1 is not reachable yet (slot running): ..."}` with `Retry-After: 3`.
- `GET /api/v1/fleet` and its stream take the real status from each bot's `/api/v1/status` (character, totals, `lastSeq`, xp/h, `ready`, failsafe from the runner state), cached 1 s per bot address with a 2 s timeout; a slot whose bot is down shows only the slot state (`slot` set, `ready` false).
- Game view: mount `viewport` per slot on the same port and behind the same token/cookie: `GET /api/v1/instances/{id}/viewport/stream` (MJPEG, alias `stream.mjpg`) and `/viewport/frame.jpg`, plus `/status` and `/overlay`. One shared capture per slot (`internal/viewport.Stream`: lazy, 4 fps, 800 px wide, latest frame wins, stops 3 s after the last viewer), 3 s write deadline per frame (a stalled tab is dropped), at most 8 viewers per slot (503 beyond), at most 4 captures in flight over all slots. View only: the handler answers GET and HEAD only, no input path exists.
- Capture: `viewport.XWDCapturer` runs `xwd -root -silent -display :10N` (the slot's Xvfb from `Alloc.Display`; the display holds only the game) and encodes the JPEG in Go (`DecodeXWD`: 24/32 bpp TrueColor). `xwd` is in `x11-apps`, which the image already installs, so the Dockerfile is unchanged (no ImageMagick, no ffmpeg).
- Dashboard (web/panels.js): the "Game view" card plays `/viewport/stream` of the selected slot while the tab is visible and the overview is shown, reopens a stream that ended or failed with a 1 to 30 s backoff, and stops when the tab is hidden (as cmd/d2watch/web/app.js).
- V (offline, `-race`): internal/server `TestRemote*` (fake bots: list, fleet, restart on a new port, bot down, POST and credential stripping, SSE flush), cmd/d2fleet `TestSlotsBecomeInstances` (real fleet + fake BotRunner + fake capturer: instance appears, 503 while starting, follows the bot to a new addr, viewport frame from display :101, removal), internal/viewport (XWD decode of a synthesized dump, capturer with a fake `xwd`, slow-client drop, viewer cap, 503 without capture), `tools/container-checklist.sh`. U (needs the container): real `xwd` output of the slot Xvfb decodes (header layout is from xwd.h, not yet seen live); the captured picture is the game (Xvfb 800x600, no window manager); MJPEG in the browser through the NAS port mapping; the first frame arrives within the 4 s xwd timeout.

Needs the user (cannot be done offline):
1. Done (IMAGE, static checks only): `d2fleet`/`d2doctor` in `tools/container-build.sh` and the Dockerfile, `EXPOSE 8431`; still to run: the build.
2. Real `docker run` flags (cap-drop ALL, no-new-privileges, volumes with uid 1000), token file creation.
3. Real prefix creation (12 s), the 1.9 GB copy, the real patch sha256 and `waiting-user` window if the dir is unpatched.
4. Real `character` step under Xvfb (Helper + Game + CreateCharacter), then the bot reaching in-game from the slot.
5. noVNC/Xpra viewer for installer windows (only for `stubs`/`installer-dirs` modes).
6. Real doctor result in the container (host.runner warn on plain Wine is expected, kernel.subreaper pass).
