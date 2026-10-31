# Troubleshooting D.idle (agent USERDOC, 2026-10-29)

> status: STALE for the Wine path: same caveat as quickstart.md. Wine container work is PAUSED (docs/decisions.md, ADR 0002); d2d is the likely primary backend, see docs/d2d-container/ and docs/d2d-backend.md.

Symptom, cause, fix, distilled from docs/stallfatal.md, restloop.md, leakhunt.md, oog.md, soak.md, bot.md, sentinel.md, review-*.md and review/gotchas.md. Target: classic 1.14d,
offline single player. Tags: V = recorded in a doc as a dated live observation or a unit test (named), S = secondary, U = unverified / inferred from code or reading. Setup: [quickstart.md](quickstart.md).
Where the fix is a command, it is written for a user; the dev route (`tools/game.sh`, `tools/helper.sh`, `tools/live.sh`) exists only on the project's the worker host box and is marked DEV.

First, three commands that answer most questions:
```
d2doctor --quick --json                          # environment: runner, Yama, Xvfb, disk, ports, game version (exit 0 ok, 1 warn, 2 fail)
curl -s localhost:8420/api/v1/bot/health         # state, stuckReason, ladderStep, paused, selfPaused, fatal   (add the token header if you set one)
docker logs --tail 100 d2idle   |   tail -100 <data>/instances/iN/logs/d2bot.log      # container | fleet slot; native: the d2bot terminal
```

## A. Start and setup
| symptom | cause | fix | tag |
|---|---|---|---|
| Game dies within 2 s of launch on a desktop (ssh start) | the session variables are missing: needs `DISPLAY`, `WAYLAND_DISPLAY`, `XAUTHORITY` of the desktop | start from a terminal inside the desktop session, or use Xvfb (managed `--instance iN`, container). DEV: `tools/game.sh start` sets them | V docs/oog.md |
| "Diablo II Critical Error: Only one copy of Diablo II may run at a time" | two Game.exe in one Wine prefix (per-wineserver guard) | one prefix per running game; `d2fleet`/`--instance iN` give each its own. Stop the stray game | V docs/multi.md |
| d2doctor `fs.prefix` fails | prefix or data dir on NTFS/exFAT/FAT or `noexec` | move `--data` to ext4/xfs/btrfs on a local disk | V docs/setup.md |
| d2doctor `kernel.yama` fails (scope 2/3) | the kernel forbids even the parent reading its child | `sysctl kernel.yama.ptrace_scope=1` (or 0); scope 1 is verified. Capabilities are not needed at 0/1 | V docs/setup.md, container.md |
| d2doctor `game.version` warns/fails | not 1.14.3.71 or unknown sha256 (another build, a mod) | patch with your own `LODPatch_114d.exe`; the bot only supports 1.14d memory offsets | V docs/setup.md, review/gotchas.md 1 |
| `host.runner` warn | Proton other than GE-Proton11-11, or plain Wine | warn only. The container uses plain Wine (V); native: put GE-Proton11-11 in `compatibilitytools.d` or set `PROTON=` | V docs/setup.md |
| Wizard stops at `character`, nothing starts | by design: no `charName` yet | enter name and class in the form, Save, Start | V docs/container.md |
| Wizard step `patch` waits (`waiting-user`) | the install is not 1.14d and the patch is running in a window you cannot see | supply an already patched folder in `game-dir` mode (no window needed) | U docs/container.md |
| Wizard returns 400 on `cdKey` | intended: keys are never accepted or stored | none | V docs/setup.md |

## B. The bot is idle, paused or stuck
| symptom | cause | fix | tag |
|---|---|---|---|
| Nothing happens after start; state `idle`/`stopped` | no goal queued, or the saved goals only start on `POST /control/start` | add a goal in the dashboard and press Start, or run with `--goals ... --autostart`; `PUT /settings {"autoResume":true}` makes restarts resume by themselves | V docs/launch.md s9, api.md |
| Character lies dead, bot looks idle ("death screen") | while dead the bot presses Esc 2 s after death, then every 3 s x5, then every 15 s, and never presses Esc while alive (it would open the game menu). A dead character that never recovers climbs the failsafe ladder. The out-of-process sentinel has a `death-overlay` rule (Esc up to 3, then restart game) but is OFF unless run with `--act` | wait ~1 min; check `GET /bot/health` `stuckReason` and the log's `dead` lines. DEV: `tools/sentinel.sh start --act-rules death-overlay`. If it never respawns: Pause, `d2fleet restart iN` (or SIGTERM and start d2bot again): the next start begins at the last saved point | V docs/bot.md, sentinel.md (rule table) |
| State `paused`, message starts "Failsafe: ..." and says "retrying in 5m" | a **self pause**: the ladder (Esc, town, re-enter, game restart) did not help, or 8+ deaths in an hour, or a leveler/task step reported "attention". Every self pause now **auto-resumes**: 5, 10, 20, 40, 60 min (backoff while pauses repeat within 2 h) | wait, or press Resume. If it keeps coming back read the message; the usual causes are rows below (no potions/gold/TP, no exit found) | V internal/bot/loop.go selfPauseDelay; docs/archive/review-day.md M8 fixed in commit 8f2f212 |
| Paused "Look at the character" after re-entering the same area | 4 re-entries into the game within 15 min (chicken loop) | Resume after checking potions and gold; see "chicken loop" below | V docs/stallfatal.md s6 |
| Paused, a death message | `bot.pauseDeathsPerHour` (default 8) deaths in an hour, or a death spiral in one area (3 deaths in 15 min: the area is avoided for an hour) | give the character a lower-risk goal (`levelto`, another area), a gentler `profile`, or raise the knob | V docs/bot.md "Death" |
| "no progress for 120 s" while `clearing`, position frozen, thousands of `nav no exit known yet, exploring` lines | a level with no exit the explorer can find (ExitTo retry loop) | fixed defensively: repeats now return `no exit to that level` at once and the planner avoids that area; clearing has its own 60 s idle limit (`clearIdleSec`) and rung 1 skips the stalled area for 15 min. Update to a build that has `agent/loopfix`/`sealfix`. If it still loops, Stop then Start picks another route | V docs/stallfatal.md s1,s3,s4 (reproduction U) |
| In town, ticking forever, log shows `quest...chores ... blocked ... still needed` | a failed chores step was re-picked every tick so the level step never ran | fixed (`TestBlockedInsertedStepDoesNotStarveLevelStep`); update | V docs/stallfatal.md s4 |
| Character rests ~40 s before every rat, mana 9-14, "rest timed out" | rest target was a 60 % floor even for one trash monster, with no gold for mana potions (regen about 0.5 mana/s) | fixed (`resource.PreEngage`: small packs need only their cast budget). Slow regen itself is real; give the character gold/potions | V docs/restloop.md |
| Low life, no potions, no town portal; chicken then re-entry every ~20 s | broke and without a TP scroll: the free heal in town was blocked by the trip cooldown, `walkToTown` only knew Blood Moor | fixed: heal-only trip in town, waypoint-to-town fallback (act 1), 3 failed attempts pause the goal visibly. Life never regenerates by itself in D2 (S): healer or potions only. Put a few gold and a TP scroll/tome on a starving character by hand: stop the bot, play it, restart | V docs/stallfatal.md s6 (waypoint path U) |
| Bot stands still in melee at low life, no kiting | it waited for the chicken threshold with nothing to drink | not fixed ("Not fixed", U): needs live look; keep potions stocked, use a safer goal | U docs/stallfatal.md s5 |
| "the area is cleared for this game" then waits | `farm` goals only do the area; with no way to make a fresh game it waits | managed mode creates a fresh game itself; if it waits, Stop/Start | S docs/bot.md |
| Campaign never moves on at the wrong difficulty | earlier builds tried the difficulty switch once per level | fixed with backoff in commit 8f2f212 (H2) | V docs/archive/review-day.md H2 |

## C. Game process, helper, menus
| symptom | cause | fix | tag |
|---|---|---|---|
| Log: `helper ... not reachable` / `move failed` repeatedly | the d2input helper process (one per instance, port 47012+N) died; seen several times in 40 min under heavy input on a loaded box | managed mode and `d2fleet` restart it (`Helper.Ensure`/Supervise, 1-30 s backoff); the sentinel rule `helper-gone` restarts it in about 12 s. DEV: `tools/game.sh helper iN` / `tools/helper.sh restart iN` (only when that instance is idle). Native: restart d2bot (`--supervise`) | V docs/bot.md, combat.md, sentinel.md |
| Helper climbs in fds/threads over hours | Go runtime threads under Wine, not a leak of handles | fixed by `GOMAXPROCS(1)` in the helper (`D2INPUT_PROCS=N` overrides); rebuild `d2input.exe` and restart it; judge after a 10 min warm-up | V docs/leakhunt.md HELPERLEAK/2 (threads settling U) |
| A big black box covers the top left of the game | a console-subsystem `d2input.exe` opened a Wine console | build the helper with `-ldflags "-H=windowsgui"` | V docs/multi.md |
| After Save and Exit the game is at the **main menu**, not character select, and the process stays | normal 1.14d behaviour: Save and Exit returns to the main menu | nothing. The bot clicks Single Player, selects the character by name and re-enters (about 2 s to menu, under 1 s from OK to in game). "Exit Diablo II" on the main menu ends the process | V docs/oog.md |
| The game process **exits by itself** after a chicken/Save and Exit; log says `the game is gone (exited)` then `fatal` | older builds: an unplanned exit was fatal. Cause why Game.exe dies after a leave is unknown (the box may have been killing games under load) | fixed: in managed mode an unplanned exit restarts the game (login included), up to `maxFailsafeRestarts` (3) without real progress in between, then fatal. Unmanaged stays fatal. Keep the host load low | V docs/stallfatal.md s5 (cause U) |
| `fatal: the game is gone` right after `recycle: failsafe restart` | the bot killed its own game for rung 4 and then counted the exit as a death of the game | fixed (`expectGameExit` bracket, `TestFailsafeRestartIsNotFatalAndGoalContinues`) | V docs/stallfatal.md s2 |
| A restart "spends the budget twice" | a stale exit event after a planned restart | fixed (`expectGameExit(false)` drops one pending event) | U docs/stallfatal.md (restartfix; by reading) |
| Out-of-game: the bot cannot find the character at the menu | more than 8 characters (needs the scrollbar, not implemented), a name that differs from `--char`, expansion/hardcore toggles, a difficulty dialog | keep at most 8 characters per prefix, use softcore expansion characters, and pass the exact `--char` | V docs/oog.md limits, U difficulty screen |
| Class not selected when creating a character (OOG) | a click within about 1-2 s of the previous one is dropped | the flow waits 2.5 s and retries 4 times; just retry | V docs/oog.md |
| Orphaned game after d2bot was killed: new run says "not our descendant" and cannot attach | memory is readable only by the parent/subreaper; the old game was re-parented to init | stop the old game (`wineserver -k` in that prefix or kill Game.exe), then start d2bot again. DEV: `NOLOCK=1 tools/game.sh stop iN`. Avoid SIGKILL (quickstart s7) | V docs/launch.md s1,s9 |
| `winedevice.exe` processes pile up after many restarts | one set per dead wineserver; `wineserver -k` does not reap them (about 35 MB and 15 threads each) | DEV: `tools/game.sh reap iN` (also runs after `game.sh stop`). User: `pkill winedevice` when no game runs in that prefix | V docs/leakhunt.md WINELEAK, soak.md WINEFIX |
| Crash dialog / BlizzardError on screen | game crash | the sentinel (`crash-dialog`) dismisses or restarts; managed d2bot restarts a crashed game. What 1.14d shows on a crash is not recorded (U) | U docs/launch.md s3, sentinel.md |
| Game hangs: window alive, CPU ticks flat 45 s, frozen frame | hang | d2bot's Monitor reports Hung and restarts via the ladder; sentinel `game-hung`/`game-frozen`. Hang detection in menus/loading is U | V/U docs/launch.md s3 |
| Clicks missed ("pointer never hovered the target", Akara dialog missed), quest claim failed once | timing under load; known on the host too | usually self-corrects (recovering re-entry). Cut load (section D) before suspecting a bug | V docs/container.md "Open problems" |

## D. Load, CPU, memory, disk
| symptom | cause | fix | tag |
|---|---|---|---|
| About one core busy per game even idle in town | Game.exe busy-loops; this is the game, not the bot | budget 1 core and about 0.3 GB per slot (container: 102 % CPU, 212-289 MiB measured). `d2doctor` `resources.cpu/ram/load` warn when free cores < slots, fail above 1.5x cores | V docs/multi.md, container.md |
| Bot slow, clicks dropped, animations long | host load above the core count (the dev box ran at load 30-40 with 5 games) | run fewer slots or move other jobs away; the sentinel and soak report `loadWarning` in `/status` | V docs/multi.md, review/gotchas.md 17 |
| Game.exe fds rise to ~120-230 in the first minutes, then flat | first visit of each area loads about 70 fds (ntsync objects); a plateau, not a leak. RLIMIT is 524288 | ignore; a recycle (Save and Exit, stop, start, login; ~22 s) happens by itself between goals at 500 cycles / 4000 fds / 30 min spacing (`D2BOT_RECYCLE_CYCLES`, `_FDS`, `_MIN_INTERVAL`; 0 disables) | V docs/leakhunt.md, soak.md |
| wineserver fds grow a few per hour | Wine's per-thread objects plus area loads, step-wise | judge only after 2 h of data (soak does); needs no action | V docs/leakhunt.md WINELEAK |
| `proc:...:fds` slope FAIL on a 20-45 min soak | one step in a short window looks like a slope | extend the run past 2 h or look at the plateau line (`d2soak report`); not a leak | V docs/soak.md |
| Disk fills | game copy 1.9 GB, image 1.2 GB, per extra prefix about 125 MB unique (hardlink clones), logs, `Data/` overlays; the dev box was at 97 % | keep 8 GB + 0.25 GB per slot free (`resources.disk`, figure U); put cold logs/backups elsewhere; `docker image prune`; remove unused volumes (backup saves first) | V docs/container.md, multi.md; U figure |
| `/dev/shm` pressure with several games | container shm default is 64 MB | run with `--shm-size 256m` as the commands do; size for many games is not measured | V/U docs/container.md |

## E. Container specifics
| symptom | cause | fix | tag |
|---|---|---|---|
| Wizard URL says unauthorized | the UI needs the token because it listens on a non-loopback address inside the container | `docker exec d2idle cat /data/token`, open `/setup.html?token=...` | V docs/container.md |
| `docker stop` takes long then the bots lose progress | the grace period is shorter than the bot's Save and Exit (up to ~50 s; the fleet waits 70 s before SIGKILL) | `--stop-timeout 90` / compose `stop_grace_period: 90s` | V docs/container.md |
| Wine "prefix owned by someone else" | `/data` is not owned by uid 1000 | use a named volume (owned by uid 1000 automatically) or chown the bind mount | V deploy/container/README.md |
| Frame capture cuts off the bottom 20 px | Wine draws a title bar into an 800x600 Xvfb | the image defaults to Xvfb 832x640x24; harmless for the bot (client coordinates) | V docs/container.md |
| ALSA log noise at each start | plain Wine tries card 0 | fixed in `d2init.sh` (`Drivers Audio=""`); the game also runs `-ns` | V docs/container.md |
| Memory read refused inside the container | the app must be the subreaper; `SYS_PTRACE` does not help a non-root user | do not add caps; run `d2bot --managed` (or the wizard) as the container's app, not `docker exec` a second reader | V deploy/container/README.md results 4-6 |

## F. Soak and long unattended runs
| symptom | cause | fix | tag |
|---|---|---|---|
| Soak report says FAIL on an empty run | fewer than 2 samples | now `NO-DATA` (exit 3) | V docs/soak.md |
| Dashboard soak strip empty | off by default | `d2bot --soak-dir DIR` or `D2BOT_SOAK_DIR=DIR` (also `d2fleet serve`, `d2dash`) | V docs/soak.md |
| `stall-xp` alert | no XP change for more than 10 min with a goal active and not paused | read the activity in `/status`; usually a section B row above | V docs/soak.md |
| `restarts` alert | more than 8 process pid changes per hour (crash loop) | check `lastError` in the slot's `bot` status; `maxRestarts` is 10 then `bot.state=failed` | V docs/soak.md, container.md |

## G. When it is none of the above
1. `GET /api/v1/bot/health` and the last 100 log lines; note the area, the activity, `ladderStep`.
2. If a game is up and nothing is happening, take a picture of the game window only (quickstart section 6) to see a stuck panel or menu. Esc is never a safe blind key: with nothing open it opens the game menu (V docs/oog.md, review/gotchas.md 19).
3. Stop cleanly (quickstart section 7), start again: the next start re-enters at the last save, in town.
4. Report with the log tail, `d2doctor --json`, the bot version (`d2bot --version`) and what you did to the character.
