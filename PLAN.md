# d2lod-bot: plan

> **STATUS (2026-10-30): historical plan, partly superseded.** This file is the 2026-10-28 founding plan and describes a Wine/Proton-only architecture; keep it for the goal, layers and history, do not rewrite it. Current truth lives elsewhere:
> - what is true now: [docs/plan/STATUS.md](docs/plan/STATUS.md) (generated block at the bottom), plan: [docs/plan/plan-v2.md](docs/plan/plan-v2.md);
> - decisions (d2d is the likely primary backend, Wine container work paused): [docs/decisions.md](docs/decisions.md) and [docs/adr/](docs/adr/README.md);
> - contracts and numeric gates: [openspec/specs/](openspec/README.md); running jobs: `openspec/changes/`; ownership: [docs/OWNERS](docs/OWNERS).


Goal: a self-contained, native-Linux app that turns classic Diablo II: Lord of Destruction 1.14d into an idle game.
The user sets goals ("reach level 30", "farm Mephisto for runes"); the bot plays, offline single-player only.
The game itself runs under Wine/Proton; everything else is native Go. Private project, nothing Blizzard owns is shipped.

## Verified facts (2026-10-28, on the worker host: Arch, KDE Wayland, GE-Proton11-11)
- Game: 1.14.3.71 (1.14d), 32-bit, `Game.exe` at 0x400000, installed in `~/Games/d2lod/proton/pfx`, windowed 800x600 (`-w`).
- Reading: `/proc/<pid>/mem` of the Wine process works with CAP_SYS_PTRACE (Yama scope 1). Reference RVAs in `docs/d2-1.14d-memory.md`;
  verified live: PlayerUnit, FirstControl, GameInfo, UnitAny fields, Path position, StatList, area id chain, server-side unit hash table, MouseX/MouseY.
- Writing/input: `cmd/d2input` (Windows exe, runs INSIDE the prefix) PostMessage()s mouse/key messages to the "Diablo II" window over
  127.0.0.1:47012. Never touches the host cursor, no focus needed. Verified: moves, click-to-move, key press.
- Iso transform (verified within ~2 subtiles): with the player drawn at client (400,300): wx-wy = dx/16, wx+wy = dy/8 (dx,dy = screen offset).
- Reference probe: `cmd/spike` (deleted, see git history; its logic lives in the packages below).

## Architecture (layers, bottom up; each package depends only on layers below)
```
internal/mem      Source interface (+Fake, Proc, Snapshot)          read-only game memory
internal/data     static game tables (monstats, areas, skills...)   extracted at setup from the user's MPQs
internal/d2       typed game state: Player, Units, Stats, Items, UI, Skills, Inventory
internal/d2map    level/room/collision reader, preset units, exits   world grid
internal/input    client of the d2input helper, iso transform, primitives (MoveTo, Click, Cast, Key)
internal/nav      pathfinding (A*), movement executor, stuck/door/portal handling
internal/bot      behaviors: combat, survival, town, loot, runs, progression
internal/goals    goal manager: goal types, scheduler, persistence, stats
internal/api      HTTP+SSE API, embedded dashboard (web/)
internal/launch   prefix/install/patch/launch (game as child + subreaper), helper lifecycle, OOG menus
cmd/d2bot         the single binary; cmd/d2snap fixture recorder; cmd/d2input the in-prefix helper
```
Contracts: all state readers take `mem.Source`; all actions go through `input.Controller`; the goal manager talks to the bot
through the `bot.Runner` interface (Start(goal), Pause, Stop, Status, Events). Every layer is testable offline (mem.Fake/Snapshot, fake input).

## Status (2026-10-29, refreshed by DOCSYNC at master 6b920b0)
The ladder and wave lists below are the original plan. Current state is in `docs/INDEX.md` (where-am-I tables per act and class, prioritised open live checks, review fix status).
In short: waves 1 and 2 are merged; the bot levels a Sorceress in Act 1 unattended for 90+ minutes through `d2bot --managed`, runs the
town loop, death policy and failsafe ladder, and has boss scripts, Acts 2 to 5 campaign wiring (`normal_all`, `nightmare_all`, `hell_all`, chained)
and Nightmare/Hell fight, resource and entry rules (Acts 3 to 5 and Nightmare/Hell are offline only, nothing played live). Since the first status note
(a0b8588, 119 commits, all offline): farm `runs:NAME:N` goals, endgame build tables for 7 classes (proposed), Hell tuning and mana-burn tracker, quest events that
resume held goals, panic recovery and `d2bot --supervise`, the stop chain (70 s fleet grace, save before runner cancel), winedevice reaping in Go, the 24 h
soak runbook (`tools/soak/soak24.sh`), `d2doctor`, prebuilt probes (`cmd/probes`), simulations (campaign, difficulty chain, seals and waves) and fuzz/coverage
hardening. The Highs of the overnight, day and evening reviews are fixed in code (INDEX section 4); no fix has been re-run live. The headless container runs
the real game, the fleet supervisor and setup wizard exist.
Not yet: a 2 h or 24 h soak report, a complete Act 1 run, Act 2 past the Summoner portal, any live Act 3 to 5, Nightmare or Hell play, the idle-game layer (parked).

## Milestone ladder (each rung is demonstrable live)
1. Read state (DONE) -> 2. Control the character (DONE: move/click/key) -> 3. Read the world map (collision) ->
4. Walk anywhere in the current level + take exits/waypoints (nav) -> 5. Fight (target, cast, potions, chicken) ->
6. Town loop (heal, buy, stash, repair, identify) + loot filter -> 7. Run scripts (Blood Moor, Den of Evil, Cold Plains ...) ->
8. Out-of-game: menus, create/select character, create/leave game, crash recovery -> 9. Leveling plan executor (stat/skill allocation, difficulty progression) ->
10. Goal manager + dashboard (set goals, watch progress) -> 11. Setup wizard (prefix, installers, patch, extract data) -> 12. Soak: days unattended.

## Work chunks
Wave 1 (parallel, disjoint directories, offline-first):
| id | owner dir(s) | deliverable | needs live game |
|---|---|---|---|
| MEM | internal/mem, internal/d2, cmd/d2snap | Proc + Snapshot sources, snapshot recorder; port spike logic into typed readers (player, units, stats, area, inventory/belt, skills, UI flags, hover, difficulty); fixtures | yes (read-only) |
| MAP | internal/d2map | Act->Level->Room2->Room1->CollMap reader, merged walkability grid across rooms, preset units, level exits/warps, caching as rooms appear | yes (read-only) |
| INPUT | internal/input, cmd/d2input | helper client, reconnect, timing/jitter, calibrated WorldToScreen/ScreenToWorld, MoveTo/Click/Cast/Key/Interact primitives, hover verification, shift-click | yes |
| LAUNCH | internal/launch, tools/ | prefix creation, installer + patch automation, D2DX/windowed config, launch game as child + subreaper + helper, OOG menu navigation, crash detection | partly (do NOT kill the running game) |
| DATA | internal/data | MPQ reader (no deps) + extraction of needed .txt tables from the user's install, typed tables with tests | no |
| GOALS | internal/goals, internal/api, web/ | goal model, scheduler, persistence, HTTP+SSE API, embedded dashboard against a simulated Runner | no |
| PLAN | docs/ | leveling strategy: build, stat/skill order, area progression Normal->Hell, safety rules, gear/merc, run list, data needed | no |

Wave 2 (after wave 1 lands; depends on it): NAV, COMBAT/SURVIVAL, TOWN+LOOT, RUNS+LEVELING executor, OOG flows, integration + soak, packaging.

## Design principle: optimal first, handicap by configuration
The idle-game layer ("the bot gets better as you play") is built by making the bot optimal first and then degrading it on purpose,
never by teaching it from scratch. Every behavior that affects efficiency takes its parameters from one `Tuning` struct
(internal/bot/tuning.go in wave 2) instead of hard-coding them: path optimality (straight A* vs sloppy waypoints/detours), exploration strategy
(frontier-nearest vs informed), reaction time and action delays, loot pickup radius and filter strictness, potion and chicken thresholds,
town-trip frequency, target-selection quality, how many known routes it remembers. The default `Tuning` is the best we can do; a "handicap profile"
lowers each knob, and the progression system raises them as unlockable upgrades (goal manager owns profiles and unlocks).
Rules for wave 2: no magic numbers in nav/combat/town code, every knob has a documented range where the bot still survives, and a test per knob
proving that degrading it makes the bot slower or less efficient but not suicidal. Learnable things that are cheap and real: remembered routes to fixed-layout
areas (towns, boss lairs, preset rooms), per-monster danger stats, and per-area clear times.

## Decisions
- Softcore first (2026-10-28). The death rule for now: try to retrieve the corpse once, otherwise abandon it and carry on. Hardcore is a later feature that fits the incremental mold as the "rebirth" (prestige) mechanic: a hardcore death resets the run and pays out permanent bonuses. Do not build it yet, but do not make it impossible: keep character-mode out of nav/combat logic and in `bot.Policy`.
- Upgrades (the visible progression layer) will need UI iteration. Data contract so the UI can change freely: `Tuning` knobs carry metadata (key, label, unit, default, min, max, direction of "better", description), and `goals.Upgrade{ID, Track, Tier, Cost, Requires[], Effects[{Knob, Value}]}` is pure data. The dashboard renders from that data and from measured telemetry (e.g. seconds per run, deaths per hour), so it never hard-codes a knob.
- The meta game layer (upgrades UI, currency, prestige) is PARKED (user, 2026-10-28) until the bot plays well; it will be designed around what we can actually control in the game. Keep only the data contract and the `Tuning` metadata in mind.

## End goal: one app, two run modes (headless container or desktop)
Target (user, 2026-10-28; not for now, but every wave-2 decision must keep it possible): the same single binary runs either headless in a container
(Docker/the NAS/Proxmox LXC, web UI only) or on a desktop (game window visible, optional). Nothing Blizzard owns is in the image: Blizzard files and
the Wine prefix live on volumes the user fills during the web setup wizard.

Design constraints for all wave-2 code:
1. No desktop assumptions: no xdotool/KWin/host cursor/uinput (already true), no screenshots, no reliance on a compositor or GPU.
2. Everything configurable by flags/env and one data dir (`D2BOT_DATA`: prefix, saves, extracted tables, state; `D2BOT_CONFIG`: goals, settings); no hard-coded `$HOME` paths.
3. Logs to stdout (structured), a health endpoint, graceful shutdown (save and exit the game, `wineserver -k`), crash/hang recovery already in internal/launch.
4. A display abstraction in internal/launch: `Display` = host display (desktop mode) or a managed Xvfb (headless). UNVERIFIED and to be tested first: D2 1.14d under Wine/Xvfb with `-gdi` (software rendering, no GPU) or a GDI/software path; memory reads and PostMessage input do not need a display, so rendering is the only open question.
5. Setup wizard in the web UI (installer upload/mount, CD key entry through the installer UI): the headless setup phase needs a way to see and click the installer window (noVNC/Xpra/KasmVNC bound to the managed X display), which doubles as a "watch the bot" debug view.
6. Offline by construction: run the game container without outbound network (setup phase excepted: Wine runtime and patch downloads); the web UI is the only exposed port.
7. Multi-bot: DEFAULT is one container as a bot host with N instance slots (add/remove at runtime through launch.Life, party over loopback, no Docker access needed; see deploy/container/README.md Decision). One container per bot stays possible where an orchestrator exists (isolation, per-bot limits); both come from the Instance abstraction in docs/net.
Packaging steps for later: Dockerfile (Wine/Proton runtime fetched at setup or baked from upstream, Xvfb, the binary and d2input.exe), compose file, NAS template (deploy only through the NAS UI / Compose Manager, ask first), forge Actions build + registry, desktop AppImage or plain binary, docs.
Verify first (small spikes): `-gdi`/software rendering under Xvfb; /proc/pid/mem reads and the subreaper inside a container (default seccomp and Yama behavior); Proton direct invocation without Steam runtime inside the image; idle CPU/RAM per instance.

## Viewport: the game embedded in the UI (user, 2026-10-28; later)
Goal: the Diablo II window is never shown on the desktop; the dashboard shows the game as a live, view-only video panel (no input is forwarded; we do not defend against deliberate tampering).
Approaches, cheapest first:
1. Capture a window on a private X display: run the game on its own display (managed Xvfb in headless mode, or a dedicated virtual display on the desktop) so it never appears on the user's screen, capture frames from that display (xcb/MIT-SHM or ffmpeg x11grab; `import -window` already works on the game window today, see tools/shot.sh), encode MJPEG (or H.264 later) and stream over the existing HTTP server (WebSocket or multipart MJPEG) into a `<canvas>`/`<img>` panel. Works for both run modes and shares the Xvfb requirement of the container end goal.
2. In-prefix capture: add a `frame` command to cmd/d2input (PrintWindow/BitBlt of the game HWND, JPEG-encode in Go inside Wine) and stream frames over the existing helper socket. No X dependency on the host side, but needs a renderer that GDI capture can see (likely only a software/GDI mode). UNVERIFIED.
3. Framebuffer from memory: in a software renderer the 8-bit palettised back buffer plus palette could be read straight from game memory (no window needed at all). UNVERIFIED; needs the surface pointer and a renderer that keeps it.
Bonus: because the screen transform is known and measured (docs/input.md), the viewport panel can overlay what the bot sees and plans (path, target, monster life, danger zones) on top of the video.
First spike (small): run the game on an Xvfb display, confirm it renders and `import`/ffmpeg can capture it, measure CPU and latency at 5 to 15 fps, then add the dashboard panel (status.Panel, web/live/panels/viewport.js).

## Death is normal (user, 2026-10-28)
Characters will die constantly; the bot is designed around it, not to avoid it at all costs. Death is a first-class state and a data source.
1. Detect: player dead mode (docs/mem.md: modes 0/17, UNVERIFIED), life 0, the death/respawn screen; one authoritative `Died` transition from the snapshot stream, emitted as a `goals.Death` event with cause (killer monster id and name, area, monster level, last life curve, last position = corpse position).
2. Recover (state `Recovering` in the bot runner): the character respawns in town; decide with a documented policy knob set: retrieve the corpse only when its estimated value exceeds a threshold and the area is judged survivable, at most N attempts per death, otherwise abandon it and rebuild from vendors ("poverty mode": cheapest viable weapon/armor, potions first). Retrieval route: waypoint/portal to the area, approach the corpse with caution, pick it up, return; abort on a second death.
3. Do not die twice for the same mistake: per-area and per-monster death counters feed a "death spiral breaker": after N deaths in M minutes in one area the planner drops to an easier area or levels elsewhere, raises danger weights for that monster/affix, and emits a high-visibility event; after too many deaths per hour the bot pauses itself and asks the user. Heavy losses are bounded by rules: keep valuables stashed, never carry unequipped valuables outside town beyond a limit, keep a gold floor for rebuying gear.
4. Make dying cheap: potions and belt first, chicken thresholds from the danger table, gear insurance (backup basic gear kept in the stash at higher levels), resupply before dangerous runs.
5. Learn from it (telemetry and display): death log with cause, area heat map on the world panel, deaths per hour, cost per death (gold, items, time), and recommendations (this area/affix kills us at this level).
6. Test it: death drills offline (fake snapshots), and verify the real death sequence live on the throwaway character: record fixtures of the dying, respawn and corpse states (modes, UI flags, timings, where it respawns) instead of treating a death as a test failure.
Knobs (env.Tuning, bot domain): corpse retrieval value threshold, max retrieval attempts, spiral breaker deaths/minutes, pause-after deaths per hour, poverty-mode gear tier, gold floor.

## Cheats (user, 2026-10-28): for testing now, a possible meta reward later
We control the game's memory and files, so for testing we may cheat on throwaway characters in offline instances: set level/XP/stats/gold, refill life and mana, invincibility, change XP/drop rates (e.g. via -direct -txt data overrides), edit the save header. Rules: only through internal/cheat (one logged entry point, reason required, character allowlist, offline only, save backups first), everything recorded in an append-only cheat ledger (time, actor, instance, character, kind, params, before/after, reason). Never in production modes unless explicitly enabled. Later the same mechanism and ledger can back "cheats" as earned meta rewards (parked with the meta layer): the display can show active cheats, and the bot's death/XP statistics must be able to exclude cheated sessions.

## Failsafes: a bot that sits in a bad state (user, 2026-10-28)
Long unattended runs WILL hit bad states (death overlay with nobody pressing Esc, a menu or panel left open, a crash dialog, a hung game, a lost helper, a plan that no longer progresses). Defense in depth, each layer independent of the one above:
1. In-bot (internal/bot): every state has a maximum duration; a monotonic progress counter (XP, position, area, gold, kills, state change) feeds a no-progress ladder per state: close panels / Esc x3, reset to town, leave the game and re-enter through the OOG flow, request a game restart, then pause with a visible error. Nothing may wait forever (death recovery included). A health endpoint reports state, time in state, last progress time and stuck reason.
2. Out-of-process sentinel (internal/sentinel, cmd/d2sentinel): a separate process per fleet that does NOT trust the bot. It watches memory state, the game process (alive, CPU, hung window), the input helper, a frozen-frame detector and the bot's heartbeat, and applies a rule table with durations: death overlay -> Esc; Esc menu or any panel open too long -> Esc; out-of-game when it should be in game -> OOG Start; crash/critical-error dialog -> dismiss or kill and restart; game or helper gone -> restart; frozen game -> kill and restart; bot hung or silent -> restart the bot process; no progress for N minutes -> escalate. Observe-only by default, every action logged with cool-downs and attempt limits so it never fights the bot.
3. Run-level limits: max consecutive failures, per-goal timeouts, max deaths per hour (already planned), and a panic mode: get the character safe in town, pause, raise the alert.
4. Visibility: failsafe events appear on the status page (red banner, sentinel panel) and in the event log; later a phone notification.
5. Fault drills: each rule is tested by deliberately injecting its fault (kill the game, SIGSTOP it, leave the Esc menu open, kill the helper, make the character die) on a throwaway instance and measuring detection and recovery time.
Process rule for agents and tools: long live runs go in the background and are polled, so an agent is never blocked and deaf for 9 minutes (this is what made i4 look hung while the BOT agent could not read messages).



## Decision: how much freedom the player gets in the idle-game layer (2026-10-28)
Roll-out order: (1) curated routines first (for example one whole difficulty as a single routine), (2) then player-customised sets of routines, (3) then a full sandbox/automation mode. The bot is built optimal-first and the game layer constrains it by configuration (handicap profiles, goal vocabulary); if the bot turns out more flexible than planned we can loosen the constraints later. The goal vocabulary (reach level, clear area, kill boss, collect item, idle, plus the campaign templates in docs/goals/planner.md) is what routines are made of.

## Decision: guide-based play first, farming optimizer later (2026-10-28)
Build the bot to play the way a player following a good guide would (class builds, quest order, normal progression, sensible gear and areas), because that reads as more natural play and matches the idle-game feel. A dedicated maximum-efficiency farming mode (optimised run lists, MF runs, rune farming, speed-focused routes) comes in a later phase and is a separate profile: both sit behind the same goal vocabulary and the handicap profiles, so the player-facing layer can offer "play it like a guide" and "farm hard" as choices. Prior-art bots (Kolbot etc.) are a cross-check for farming-run details, not the template for the default play style.
