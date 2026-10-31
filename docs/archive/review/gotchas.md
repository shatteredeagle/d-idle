# Gotchas for an external D2 1.14d bot under Wine/Proton

Each entry: **symptom / cause / mitigation / status**. Status: HANDLED (in code, with evidence), PARTIAL, OPEN.
Tags: (V) read in repo/docs or run here, (S) secondary source, (U) unverified or from memory.

## 1. Game version and patch drift
1. **Wrong build attaches.** Symptom: garbage reads, clicks in wrong places. Cause: a different Game.exe (1.14b/c, a mod, D2R).
   Mitigation: `mem.VerifyBuild` checks SizeOfImage 0x5ba000 + TimeDateStamp 0x574ddfbc (V docs/mem.md); the installer verifies by
   `FileVersion` 1.14.3.71 (V docs/launch.md 4). Status: HANDLED for attach; make it a hard gate in managed mode and the wizard too.
2. **Patch download disappears.** `LODPatch_114d.exe` is fetched over plain HTTP from ftp.blizzard.com (V docs/launch.md 4). Blizzard
   can retire it at any time; the Downloader stubs' CDN already answers 403 (V). Mitigation: wizard accepts a user-supplied patch file
   verified by sha256; never mirror it ourselves. Status: PARTIAL (hash check exists; upload path does not).
3. **Data tables from the wrong archive.** Cause: priority order Patch_D2 > d2exp > d2data matters; monstats differs per archive
   (V docs/data.md). Status: HANDLED (priority loader, size stamp). Gotcha: the stamp uses file *sizes*; a same-size modded MPQ is not
   detected. Mitigation: add a sha256 of Patch_D2.mpq to the stamp.
4. **`-direct -txt` overlay left behind.** Symptom: experience curve or drop rates silently wrong after a test. Cause: overlay folder
   plus compiled `.bin` files stay in the instance game dir (V docs/cheats.md). Mitigation: the bot refuses to start when a `Data/`
   overlay exists unless `--allow-cheats`; the dashboard shows it. Status: OPEN.
5. **Telemetry trusts cheated runs.** Ledger exists and `cheat.Cheated(since)` (V docs/cheats.md) but the runner does not call it.
   Status: OPEN (sketch only).

## 2. Wine / Proton updates
6. **Proton upgrade writes through hardlinks.** Prefix clones share inodes; an in-place rewrite changes every instance and the master
   (V docs/multi.md caveat). Mitigation: pin the runner (FindRunners prefers GE-Proton11-11, V docs/launch.md 2); `mkprefix` after any
   runner change; the wizard records the runner version in the instance manifest and refuses a silent change. Status: PARTIAL.
7. **WoW64 mode change.** Recent Wine/Proton can run 32-bit apps in "new WoW64" (one 64-bit Unix process) instead of a 32-bit
   preloader (U). The image is still mapped at 0x400000 today (V docs/launch.md 1), but process discovery matches
   `/proc/<pid>/exe -> wine-preloader` (V docs/launch.md 1; tools/game.sh `gamepids`). A runner that changes the preloader name or
   layout breaks discovery, not offsets. Mitigation: discovery by cmdline + PE header at 0x400000 only; a canary test per runner version.
8. **Proton without Steam runtime in a container.** `proton run` works on the worker host because host libraries exist (V docs/launch.md 1).
   A minimal image may lack them (U). Mitigation: container spike with plain Wine (distro or WineHQ build) vs GE-Proton; pick one and pin.
9. **Wine flushes the registry lazily.** `Config.ReadUserReg` may read stale values (V docs/launch.md 5). Mitigation: settings changes
   through `reg.exe` and a wineserver flush; never edit `user.reg` while the wineserver runs. Status: PARTIAL.
10. **Session environment needed on the desktop.** A game started from ssh without DISPLAY/WAYLAND_DISPLAY/XAUTHORITY dies in 2 s
   (V docs/oog.md). In-container and Xvfb modes need only DISPLAY (V tools/game.sh). Status: HANDLED in game.sh; port to Go launch.
11. **Helper console window covers the game.** A console-subsystem `d2input.exe` opens a Wine console over the client area on Xvfb
   (V docs/multi.md note). Status: HANDLED (`-H=windowsgui`); add a build check in CI.
12. **Helper silently dies.** i4's helper died twice with no known cause (V docs/bot.md). Status: PARTIAL (sentinel `helper-gone`
   restarts it in ~12 s, V docs/sentinel.md). Add helper logs with timestamps and an exit reason; supervise from `launch.Helper.Supervise`.

## 3. 1.14d game quirks
13. **Map seed per game vs per character.** Layouts differ between characters; whether they persist per character across games is
   UNVERIFIED (S: `.d2s` map id at 0xAB). See findings F10. Status: OPEN (one live test).
14. **Mirrored town layouts.** Same act, different seed, mirrored town (Blood Moor east or west) (V docs/nav.md NAV2). Status: HANDLED
   (presets, no absolute coordinates). Remaining literals in `combatlive` (V docs/nav.md).
15. **Act transitions.** Act 1 to 2 needs Andariel's quest and Warriv's dialog (not a waypoint); Act 3 by Meshif; Act 4 by the portal in
   Durance 3 after Mephisto; Act 5 by Tyrael's portal (U, standard game knowledge). Waypoint tabs 2-5 UNVERIFIED (V docs/nav.md).
   Status: OPEN, unowned (findings F2).
16. **Loading screens and torn reads.** Area change mid-read gives broken pointer chains; d2 retries and reports `StateLoading`
   (V docs/mem.md). Tile exits can fire before the walk ends (`ErrLevelChanged`, V docs/nav.md). Status: HANDLED in nav; every
   behavior must treat `StateLoading` as "wait", not "error".
17. **Lag/desync.** Offline there is no network lag, but a loaded host (the worker host load 30-40, V docs/multi.md) delays the game loop:
   clicks dropped, animations longer, potion presses missed. Mitigation: verify every action by its effect in memory (most code does),
   timing knobs with jitter in `input.Timing`, and a host-load guard that pauses fights when the frame rate collapses. Status: PARTIAL.
18. **Clicks dropped when sent too fast.** Class selection drops a click within ~2 s of the previous one (V docs/oog.md); casts within the
   cast delay are ignored (V docs/combat.md, Blizzard 45 frames). Status: HANDLED where found; generic rule: verify, then retry.
19. **Esc semantics.** Esc with nothing open OPENS the game menu; with a panel open it closes it (V docs/mem.md Q1, docs/oog.md). Every
   blind Esc loop is a bug. Status: HANDLED in Flow/sentinel/bot; `input.Game.CloseAll` is safe only because it uses an allow-list.
20. **Always-on UI flags.** 0, 0x15, 0x22, 0x23 at rest; 6/7 while stat/skill points are unspent; 0x11 after any quest update;
   0x21 after `h` (V docs/mem.md, docs/town.md, docs/sentinel.md). Status: PARTIAL (three different lists, findings F9).
21. **Shop UI.** The backpack beside a shop or stash is drawn without the inventory flag; pressing `i` there breaks the panel state
   (V docs/town.md). Right-click does not sell (V). Vendor price is not `ItemBase.Cost` (V). Status: HANDLED in town.
22. **NPC dialog position.** The dialog box is drawn where the NPC stood when it opened; wandering NPCs move on (V docs/town.md).
   Status: HANDLED (snapshot at flag 0x08, offset retries).
23. **First-contact speech.** Akara's first contact plays a greeting instead of the menu (V docs/town.md); quest NPCs do this at every
   quest step (U). Mitigation: dialog state machine that recognises "speech" vs "menu" and dismisses speech. Status: PARTIAL.
24. **Inventory and stash edge cases.** Item on cursor blocks clicks (S prior-art 07); 2x/3x/4x items need fit checks; a full stash; gold
   on cursor; partial gold deposit impossible without backspace (V docs/town.md). Shared stash does not exist in 1.14d (single
   personal stash 6x8 in LoD, U); the classic 6x4 bank applies to non-expansion characters (V docs/data.md: "Bank Page 1" 6x4, "Big Bank"
   6x8; which one the client uses is UNVERIFIED). Status: PARTIAL. Never create classic (non-expansion) characters: the create screen
   ticks expansion by default (V docs/oog.md); assert expansion status from the save header bit 0x20 (V docs/cheats.md).
25. **Identification.** Unidentified items hide stats; pickit rules that need stats return `NeedID` (V docs/town.md). Cain identifies for free
   only after the Act 1 Cain quest (U). Mitigation: buy and use ID scrolls/tome until Cain is available; the identify step is UNVERIFIED live.
26. **Quest and NPC dialogs blocking.** Quest completions open speech or a reward dialog (Den of Evil skill point, Radament book,
   Lam Esen's tome) (U). An unread quest dialog leaves flags set. Mitigation: per-quest handler with expected UI states; sentinel `panel-stuck` as backstop.
27. **Cow level.** Needs Wirt's Leg + Tome of Town Portal in the cube in Rogue Encampment, only after killing Baal on that difficulty, and
   is closed after killing the Cow King on that character (U, standard knowledge). The plan lists Cows (V docs/plan/leveling.md).
   Mitigation: defer; when built, never kill the Cow King (filter by superunique id) and keep a spare leg in the stash.
28. **LoD-only content.** Act 5, runewords, Big Bank stash, merc items, charms all need an expansion character (U). Status: by default.
29. **Character file versioning.** v96 (0x60) only; D2R (0x61+) rejected by the editor (V docs/cheats.md). Corruption on crash mid-write
   is possible (U). Mitigation: WP-SAVES checksum validation and rotation.
30. **Ladder vs non-ladder.** Offline single player is non-ladder: ladder-only runewords (Enigma, Infinity, Insight...) are unavailable
   offline in 1.14d (U: I believe 1.14 kept the ladder restriction; verify against `runes.txt` column `server`/`ladder`, which DATA keeps (V docs/data.md)).
   The plan targets Insight, Infinity and Enigma (V docs/plan/leveling.md). Mitigation: check `runes.txt` before the planner counts on them;
   the loader already reads the column.
31. **Difficulty and penalties.** Resist penalty 0/-40/-100 and death XP penalty 0/5/10 % (V docs/data.md). A death in Nightmare/Hell loses XP
   (and can de-level progress) plus gold; corpse holds equipped gear (V docs/prior-art/07). The difficulty dialog is UNVERIFIED
   (V docs/oog.md). Status: OPEN for NM/Hell.
32. **Mercenary death.** A dead merc needs revive gold at the act's merc NPC; after an area change the merc unit is briefly missing (S
   prior-art 07 section 8). No merc reader exists (findings F2). Status: OPEN.
33. **Death mode values.** Dead mode 17 observed live by NAV2 (V docs/nav.md "mode 17 read as dead"); mode 0 is still an assumption; life 0
   without damage never kills (V docs/sentinel.md). COMBAT2 recorded traces (V agent/combat2 testdata). Status: PARTIAL, merging.
34. **Unit ids collide across types.** Waypoint object and Warriv both id 11 (V docs/nav.md). Always key units by (type, id). Status: HANDLED in nav.
35. **Units far away are absent.** Only nearby units are in the tables; presets appear as rooms load (V docs/nav.md, docs/map.md). Status: HANDLED.

## 4. Timing, tick rate, determinism
36. **Polling too slow for survival.** Prior art polls 20-100 ms; chicken below 20 % starts dying with slower loops (S prior-art 07).
   Combat knobs prove "all worst at once dies" (V docs/combat.md). Status: HANDLED by knobs; enforce a profile rule that caps stacking.
37. **Real sleeps inside fakes.** `input.Game` uses `time.Sleep` (V controller.go), bot tests take ~15 s and one flakes (V, run here).
   Mitigation: Clock everywhere; `-race -count=3` in CI. Status: OPEN.
38. **Non-determinism in tests.** Random seeds (combat `SetSeed`), map order iteration, wall clock. Mitigation: seed from the test name; never range over maps where order matters. Status: PARTIAL.
39. **Clock jumps.** Suspend or NTP jumps; goals ignore gaps over 5 min (V docs/api.md). Watchdogs must use monotonic time (Go `time.Now()` carries it unless serialised). Status: PARTIAL.

## 5. Window, display, Xvfb
40. **Focus and occlusion.** PostMessage needs no focus or visibility; reads never depend on the window (V docs/multi.md). Unknown: real
   minimize under a WM (V docs/multi.md). Status: HANDLED on Xvfb.
41. **Client area clipped on Xvfb.** 800x600 screen gives 800x587 client because the frame is clipped (V docs/multi.md). Transform still worked
   but the bottom 13 px (belt row, y up to 600) are missing. Mitigation: run Xvfb at 1024x768 (UNVERIFIED that the client is then 800x600) and
   assert `ScreenSize` = 800x600 at start. Status: OPEN (risk for belt and bottom-bar clicks).
42. **No window manager.** Windows open at (0,0), dialogs overlap the game, `windowminimize` is a no-op (V docs/multi.md). Crash dialogs must be
   found by window list (V sentinel). Status: HANDLED.
43. **Capture cost.** ~22-31 ms CPU per `import` capture; 5 viewers ~46 % of a core (V docs/watch.md). Mitigation: lazy capture (done), one capture per
   frame shared, MJPEG; consider `ffmpeg x11grab` for >4 fps. Status: HANDLED for now.
44. **Xvfb memory and leaks.** Xvfb per instance ~10 MB (V docs/multi.md); long-running Xvfb with a leaking client grows (U). Mitigation: soak metric.

## 6. Permissions and containers
45. **Yama ptrace_scope.** Scope 1 blocks non-ancestors; the app must be subreaper and ancestor, or hold CAP_SYS_PTRACE (V docs/launch.md 1).
   Scope 2/3 on hardened distros blocks even that (U: scope 2 needs CAP_SYS_PTRACE, scope 3 forbids attach entirely). Wizard preflight must read
   `/proc/sys/kernel/yama/ptrace_scope` and explain. Status: OPEN.
46. **setcap is wiped by every rebuild** (V memory file) and is not allowed on `noexec`/`nosuid` mounts or in many containers (U). Status: dev-only; product must not need it.
47. **Docker default seccomp/caps.** `process_vm_readv` and `/proc/pid/mem` of a *child* need no capability in the same container; reading a
   sibling needs SYS_PTRACE (U). Docker's default seccomp profile allows ptrace on kernels >= 4.8 (S, Docker docs as remembered). Status: OPEN
   (container spike WP-CONTAINER).
48. **PID 1 in containers.** The subreaper role and zombie reaping (`launch.Reap`, V docs/launch.md 1) must still work when `d2bot` is PID 1 or
   under tini. Signals to PID 1 have no default handlers. Mitigation: run under `tini -s` or handle SIGTERM explicitly (d2bot does, V docs/bot.md).
49. **Unprivileged LXC.** Wine needs no privileges, but Xvfb and `/dev/shm` sizes, and `kernel.yama` sysctls are host-wide (U). Note the head box itself is an LXC.

## 7. Save files and locking
50. **Same character twice.** Two clients on one character corrupt or block (U; NET R). One prefix per instance with its own saves makes it unlikely
   (V docs/multi.md); enforce in Fleet: unique (character, save dir).
51. **Editing a save while in game.** The game overwrites it on exit (V docs/cheats.md: edits only at the menu or with no Game.exe). Status: HANDLED for cheats; WP-SAVES must use the same guard.
52. **Progress lost on kill.** See findings F3. Status: OPEN.
53. **Backups on slow NTFS.** `/mnt/cold` is ntfs3 spinning disk (V memory file); fine for cold backups, not for anything in the game loop.

## 8. Long runs and resource leaks
54. **Zombie processes.** Subreaper keeps orphans as zombies until reaped (V docs/launch.md 1). Status: HANDLED by `launch.Reap`, needs a soak metric.
55. **File descriptor and goroutine leaks.** Reconnecting input client, SSE clients, `/proc` fds per poll (U). Mitigation: soak harness records fds,
   goroutines, RSS per hour; fail on slope.
56. **Unbounded logs and ledgers.** JSONL logs (sentinel, cheat ledger) grow forever (V docs/sentinel.md, docs/cheats.md). Mitigation: rotation by size and age.
57. **Game memory growth.** Long sessions in one game accumulate (U, prior art restarts every ~2 h, V docs/plan/leveling.md 7). Mitigation: bounded game length (Cycler).
58. **Disk fills the NVMe.** User allows the worker host's NVMe down to ~10 GB free (V memory file); prefixes, logs, fixtures and `.bin` overlays add up. Mitigation: a disk guard in the health endpoint that pauses before writes fail.

## 9. Anti-stuck and state machines
59. **Ladder loops.** Movement or area changes caused by the ladder itself reset progress; fixed by counting only XP/kills/items/gold (V docs/bot.md). Status: HANDLED.
60. **Two failsafe layers fight.** Sentinel and bot ladder act on the same symptoms (Esc, re-enter, restart). Status: PARTIAL (sentinel cool-downs, settle, locks for some actions). Add input lease (design-gaps.md D9) and make the bot report "ladder step in progress" so the sentinel waits.
61. **State-machine explosion.** Each new feature (quests, merc, act travel, boss) adds states and transitions to `bot.Runner`. Mitigation: a hierarchical plan: Runner states stay few; a "task" layer (run script = list of steps with preconditions, PLAN/prior-art "skip/need predicates", V lessons A4) holds the detail. Status: OPEN; decide before quests land.
62. **"Healthy but useless".** Bot alive, sentinel green, no progress (V docs/sentinel.md "Cannot catch"). Mitigation: XP/hour and kills/hour floors per goal as a watchdog input, not only position change.

## 10. Test fixtures and observability
63. **Fixture rot.** Fixtures encode old assumptions (wrong offsets, old seeds); see findings F12. Mitigation: header with build/recorder version and a "re-verify" date; tests that assert *invariants* (docs/d2-1.14d-memory.md section 6 canaries) on every fixture.
64. **Fixtures with personal data.** Snapshots contain character names and possibly the CD-key hash in memory pages (U). Mitigation: recorder records only pages the reader touched (Recorder does, V docs/map.md); never commit `.key` or `.d2s` (V docs/cheats.md says none are committed).
65. **No correlation id.** Logs from bot, sentinel, d2watch, helper are separate files with separate clocks. Mitigation: `inst`, `run`, `game` ids on every line and event; one log stream per instance.
66. **No metrics endpoint.** Health JSON exists (V docs/bot.md); nothing scrapeable. Mitigation: Prometheus text format (stdlib) for the soak harness.

## 11. Distribution, legal, supportability
67. **Shipping Blizzard bytes by accident.** Fixtures with memory pages, extracted tables in testdata, screenshots, `.d2s` files. Mitigation: CI check that
   rejects MPQ/`.d2s`/`.txt` table names and files > 1 MB outside allowed testdata; tests use hand-written excerpts (V docs/data.md).
68. **Prior-art licenses.** Kolbot has no license file; paraphrase only and keep derived work private (V docs/prior-art/index.md, memory file).
   Tyniann is source-available (V). zlib-licensed `explode.go` port needs its notice kept (V memory file). Mitigation: `THIRD_PARTY.md` before any release, even private.
69. **D2DX is GPL-3.0** (V docs/launch.md 6): the user supplies the dll; never bundle it.
70. **Legal exposure.** Offline single-player automation of a game the user owns, nothing redistributed, no Battle.net: low practical risk (S/U, not legal advice).
   The EULA forbids reverse engineering and unauthorized third-party programs in general terms (U). Keep: no Battle.net code path (AGENTS.md, V), no
   distribution of game files or derived tables, private repo, no public release without a license review. PvPGN/D2GS (NET) is the riskiest idea (V docs/net/multiplayer.md).
71. **CD keys.** Typed by the user into the installer UI; never stored (V docs/launch.md 4). The `.key`/registry may hold key-derived data (U). Never log registry dumps.
72. **Arbitrary distros.** Missing 32-bit libs, no Xvfb package, SELinux/AppArmor denying ptrace, hardened Yama, Flatpak Steam paths, NTFS game dirs
   (V memory file: never put prefixes on NTFS). Mitigation: container is the supported path; desktop mode gets a `d2bot doctor` preflight with exact fixes per check (WP-SETUP).
73. **Installer GUI needs a visible desktop.** The BAM installer and the patcher have no verified silent mode (V docs/launch.md 4). Headless setup needs noVNC/Xpra
   bound to the Xvfb display (V PLAN.md 5). Status: OPEN.
74. **Blizzard downloader stubs break.** CDN retirement (V 403 on direct download). Mitigation: accept already-installed game dirs (copy-in) verified by the torrent piece hashes (`VerifyDir`, V docs/launch.md 4) as an alternative to running stubs.
