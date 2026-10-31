# d2d private fork: lockstep clock and `dump` verb (agent D2DFORK, 2026-10-30)

Result in one paragraph: a small additive patch to d2d (the headless D2 re-implementation, GPL-3.0) gives it a virtual
clock. With `--lockstep` the game advances only when told (`step <n>`), headless, at about **120x real time** with the
player idle in town, **90x to 370x** in outdoor levels with monsters, deterministic (same `--seed` + same inputs gives
byte-identical `monsters` output). `--time-scale <x>` runs the real-time mode faster (measured 10x for 10 minutes without a
problem). A read-only `dump` verb returns the player/stats/items/quests state as JSON. The patch is 287 new lines in 3 new files
(P2 145, P1 142) plus 37 added and 8 changed lines in shared files (git diff --stat vs base: 9 files, +324 -8). It lives ONLY on the worker host in a private clone; nothing of it is in this repo
(docs/decisions.md 2026-10-30, docs/d2d-backend.md section 4). Tags: V = measured on the worker host this session, S = read in
source, U = not verified.

## 1. Where it lives on the worker host

| What | Where |
|---|---|
| Fork clone (private, local only, no remote) | `~/d2d-lab/fork/src` (cloned from the local `~/d2d-lab/src`, which is upstream `psi29a/d2decomped` at `e36922e`) |
| Base commit | `e36922e` (upstream tip when the spike was built) |
| Patch commits on top | 2: "P2: virtual clock ..." and "P1: dump verb ..." (hashes: `git -C ~/d2d-lab/fork/src log --oneline -3`) |
| Patch files for re-applying | `~/d2d-lab/fork/patches/000{1,2}-*.patch` (`git format-patch`; **kept only there, never copied into this repo**) |
| Binaries | `~/d2d-lab/fork/src/build/apps/d2d/d2d` (patched), `~/d2d-lab/fork/d2d.base` (unpatched build of the same base, for comparison) |
| Run wrapper | `~/d2d-lab/fork/start.sh [d2d args]` (own `XDG_*` dirs under `~/d2d-lab/fork/home`, own socket `~/d2d-lab/fork/d2d.sock`, shares only the read-only MPQ copy `~/d2d-lab/data`) |
| Test and bench scripts (ours) | `~/d2d-lab/fork/{ctl.py,q.py,bench.py,b2.py,det.py,ts.py}` |
| The original `~/d2d-lab` build | untouched; other probes keep using `~/d2d-lab/start.sh` |

Rebuild: `cd ~/d2d-lab/fork/src && nice -n 10 ninja -C build -j4 d2d` (configured with
`cmake -S . -B build -G Ninja -DCMAKE_BUILD_TYPE=Release -DD2D_WERROR=OFF`; a clean build is about 8 minutes wall at `-j4` and
The worker host load 30; an incremental one after touching main.cpp about 1 minute) V.

Re-apply on a newer upstream:

```
cd ~/d2d-lab/fork && git clone <local upstream checkout> src2 && cd src2
git am ../patches/*.patch        # or: git cherry-pick <our two commits> from src
```

Conflicts to expect: `apps/d2d/main.cpp` (7 small hunks around the loop), `components/devctl/devctl.cpp`, `town.cpp` (one line).
If a rebase takes more than a day, stop: the policy in docs/d2d-backend.md section 4.2 applies.

## 2. How d2d keeps time (read-only study, source at `e36922e`) S

- `run_windowed` in `apps/d2d/main.cpp` holds the whole game loop (about 900 lines in one `while (!quit)`). Per frame it polls SDL
  events, pumps the control socket (`channel.pump()`), reads the clock once (`now_ms = SDL_GetTicks() - start_ticks`), runs the screen
  (`town.update(...)` for the game), uploads and presents the framebuffer, and ends with `pace_frame()`, an `SDL_Delay` up to a
  16 ms budget.
- The world does **not** use the frame delta directly. `Town::walk` runs fixed 40 ms ticks (`kTickMs`, 25 Hz) from `world_ms`:
  `for (...; frame_ms - world_ms >= kTickMs && ticks < 5; ...) world.tick(...)`. Two consequences: at most 5 ticks (200 ms game time)
  per frame, and a jump of more than 1000 ms snaps the clock instead of catching up. So "run faster" must either feed small
  deltas at a high frame rate (time scale) or exactly one tick's worth per frame (lockstep).
- `Town::update` draws inside the same call (`draw(framebuffer, ...)`), so skipping drawing means a flag, not a reordering.
  Drawing plus present costs about 17 ms per frame in the software renderer V (stepping 5000 frames: 1.6 s without drawing, 87.7 s with a
  draw on every frame).
- Other clocks: `SDL_GetTicks` is also read for the frame pacer, the stall watchdog (a real-time thread), the gamepad, and audio fades
  (headless uses openal's null device); `netgame.cpp` uses `steady_clock` (multiplayer only). None of these feed the world in
  single player, so they stay on real time.
- All game RNGs are fixed-seed constants (`Rng rng{0x2545f491u}` in World, `0x7f4a7c15u` in Town, sound RNG `0x5eed5u`) S. The map seed
  is `--seed`. That explains the spike's finding that rolls differ between runs: the number of RNG draws depends on how many ticks
  land in which frame, which depends on wall-clock jitter. Lockstep removes the jitter (section 5).

## 3. The patch

New files (287 lines): `apps/d2d/lockstep.hpp`, `apps/d2d/lockstep.cpp` (P2, 145); `apps/d2d/devctl_dump.hpp` (P1, 142).
Shared files (37 lines added, 8 changed; policy was under 100): `main.cpp` 7 hooks, `town.cpp`/`town.hpp` 1 line each, `components/devctl/devctl.{hpp,cpp}` the
deferred reply, `apps/d2d/CMakeLists.txt` the source list.

Behaviour:

- `--lockstep`: the game clock moves only while a `step` is running. Each stepped frame adds exactly 40 ms to the virtual clock, so
  each frame is exactly one world tick. Idle (nothing pending) the loop still runs at the normal pace with a frozen clock, so
  screenshots, `state`, `click` etc. keep working; clicks are queued and take effect on the next stepped frame.
- `step <n> [render]`: reply (after the last frame): `frame=<f> game_ms=<ms> tick=<t>` then `ok`. The reply is deferred, so a
  client blocks until the n frames are done. Intermediate frames skip drawing and presenting; the last one is drawn so a screenshot
  after a step matches. `render` draws every frame (for checking that skipping the draw changes nothing). No pacing sleep while stepping.
- `clock`: `lockstep=<0|1> scale=<x> frame=<f> stepped=<n> game_ms=<ms> tick=<t> pending=<n> render_steps=<0|1>`.
- `lockstep on|off`, `timescale <x>`: runtime switches. Menus, character select, level loading and the main-menu screens still
  rely on real time; use `lockstep off` for them or start with `--start-screen ingame`.
- `--time-scale <x>` (also `timescale`): outside lockstep, virtual time = x times real time, no pacing sleep for x > 1, drawing at 10 fps real.
  The 5-tick-per-frame cap means the practical ceiling is about x11 at 53 fps; above that frames must be cheaper (the draw cadence
  of 10 fps is what keeps them cheap: see measurements).
- Deferred reply: `Channel::defer()` / `Channel::reply()`. A handler that calls `defer()` and returns an empty string holds the
  connection: no further line is dispatched until `reply()`. Disconnecting clears it.
- `dump [section,...]` (P1): one line of JSON then `ok`. Sections `player, stats, skills, items, quests, ui, merc, time` (default all).
  Fields follow docs/d2d-backend.md section 4.3 where d2d has the data: `player` (name, class name, level, exp, gold, stash gold,
  difficulty, hardcore, area id and name, x, y, mode, life/mana/stamina pairs, points, left/right skill), `stats` (base attributes,
  panel bonuses, defense, resists and caps, raw stat vector), `skills.base_levels` (30 entries in Skills.txt order for the class),
  `items` (id, code, location word, slot, x, y, w, h, quality, ilvl, sockets, qty, durability pair, unique/set id, flags, display name),
  `quests` (3 difficulties x 48 little-endian words as in the .d2s block, plus 3 waypoint masks), `ui` panel flags, `merc`,
  `time`. Tested: a fresh Necromancer, all sections parse as JSON (python json.loads), `dump player,time` works, an unknown section returns `err unknown section foo` V.
  Not in this first cut: monsters, ground, objects, missiles (the existing verbs cover them), unit stat lists, quests as
  named words, belt/cube split beyond `loc`, ids of skills (needs the class table).

## 4. Measured speed (the worker host, load average 28 to 35 during the runs, niced) V

Real time is 25 world ticks per second. "Frames" below are stepped frames = world ticks.

| Scenario | Frames/s | x real time | CPU |
|---|---|---|---|
| town (Rogue Encampment, idle), lockstep, no draw | 3000 (3058, 3101, 2908 in three runs) | 116 to 124 | 0.98 core |
| same, `render` on every frame | 57 | 2.3 | 0.99 |
| Blood Moor, clvl 30, 2 to 6 monsters awake | 9400 | 377 | 0.95 |
| Burial Grounds, 17 monsters (passive) | 2200 | 87 | 1.00 |
| `--time-scale 10`, Blood Moor with warping, 10 minutes | 53 (draw every frame, first build) | 10.0 exactly | 0.98 |
| `--time-scale 20`, 60 s, after the draw cadence change to 10 fps | 2170 | 20.0 exactly | about 1 |

The loop uses one core flat out; the speed-up is therefore bounded by the single-thread cost of one tick. The town is slower
than the Blood Moor because its NPC and critter simulation runs every tick. A parallel soak means N processes, not threads.

## 5. Determinism V

Method (`~/d2d-lab/fork/det.py`): fresh process each run with `--lockstep --seed 3 --start-screen ingame --start-class 1`, `debug stat 12 8` (or 30),
`debug goto 2` (Blood Moor), steps in blocks of 500 (or 250), `monsters` after every block; compare md5 of the concatenated outputs.

| Run set | Result |
|---|---|
| 3 runs, passive zombies, 8 x 500 steps | identical hash (`d1ca6c46...`) |
| 3 runs, player warped next to the monsters, zombies walking (`WL`) and quill rats, 8 x 250 steps | identical hash (`5f906303...`) |
| same with `step ... render` (drawing every frame) | identical hash: skipping the draw does not change the logic in this scenario |

Frame numbers in the full logs differ only because the start-up frames before the first step vary; use `stepped` and `game_ms`,
which are identical. Without lockstep the spike measured different monster HP between runs with the same seed (docs/d2d-spike.md),
so **lockstep makes combat reproducible** for a given seed and input sequence. Not covered: inputs that arrive at different
frames (a real client's clicks are applied at the next stepped frame, so the same `click` then `step n` sequence is reproducible, but
click-while-stepping is not possible by construction), multi-monster fights with player damage (the player stayed at full life; the
zombies walked but did not reach attack range in 2000 ticks) U.

## 6. Checks that nothing else broke

- Control verbs (`ping`, `info`, `state`, `click`, `cmd move`, `monsters`, `debug stat|goto|warp`, `save`, `screenshot` not retried) answer as before, also
  while `--lockstep` is on V. `step` reply is held until the last frame, a second connection still gets `err busy` (unchanged).
- `cmd move` after `click 400 500` moves the character (30.70,13.70 -> 33.10,16.10 in 100 steps) V. Without the click it also worked in this fresh
  start (no level change was involved), so the spike's "ignored after load or goto until a click" quirk (P3) is not fixed or changed by this patch; not re-tested after a goto U.
- Save and load: `save`, quit, restart with `--start-screen charselect --start-name Det`, `key Return`: character loads (about 1 s with warm cache), `lockstep on`
  then `click`, `step`, `cmd move` works (31.10,14.10 after 80 steps), `save` ok V. Menu screens ran in real time before `lockstep on`.
- 10 minutes at `--time-scale 10` (first build, draw every 16 ms): no crash, no "MAIN STUCK", 1 slow-frame warning; game time 6042 s
  (100.7 min) in 600 s; RSS 525 MB at start, 545 MB at the end (10 monsters, level built once) V. The later draw-cadence change was only run for 60 s at x20 V.
- Lockstep long run: 5 x 25000 plus the determinism runs (about 150000 ticks in total), no crash V. A 10-minute lockstep soak was not run U.
- The fork's `dump` returns 3 KB for a fresh character (7 items); a loaded late-game character may exceed the 64 KiB read cap of the devctl
  line parser only for requests, not replies, so replies are not limited U.

## 7. Risks

- Rendering is fused into `Town::update`; the skip flag leaves other per-frame work that touches the framebuffer (cursor blit, overlays in
  main.cpp's `InGame` case) running. If upstream moves work between `update` and `draw`, the flag may skip logic. The determinism
  run with and without drawing guards this; rerun `det.py` after every rebase.
- The `tick` cap (5 per frame) and the "more than 1000 ms snaps" rule are upstream behaviour; `--time-scale` above about 11 loses
  ticks and is not a faithful speed-up. Use lockstep for fast runs.
- Level changes that load asynchronously (`std::future`) were exercised only through `debug goto` and the save/load path.
- Frame counters include start-up frames; clients must use `game_ms` or `stepped`.
- Memory grows by about 100 MB per first-visited level (docs/d2d-probes.md); a long soak must restart the process.
- Audio and music run on real time (null device in headless), so audio state is not reproducible.

## 8. Licence notes (not legal advice)

d2d is GPL-3.0-or-later. A private clone on the worker host with local patches imposes no obligation. If a binary or container image containing
the patched d2d is given to anyone, the corresponding source (including these patches) must be offered under GPL-3.0. Our Go
bot talks to it over a socket and is a separate program. These notes and this repo contain none of d2d's code: only descriptions in our own words and short
identifiers. The patch files stay under `~/d2d-lab/fork/patches/` on the worker host.

## 9. Draft upstream text (for the user to review; NOTHING has been posted)

Issue: "devctl: deterministic stepping (`--lockstep`, `step <n>`, `clock`) and `--time-scale`".

> We drive d2d headless as a test rig for a bot. Game time follows `SDL_GetTicks`, so runs with the same `--seed` still diverge (monster HP and
> AI rolls differ) and long soak runs cost real hours. We tried a small opt-in patch: one function that supplies the loop's game
> time, `--time-scale <x>`, and `--lockstep` where each `step <n>` advances n fixed 40 ms world ticks, skips pacing and (except the last)
> drawing, and replies when done (devctl gets a deferred reply). Default behaviour is unchanged. On a loaded 16-core box the town runs at about 120x real time
> and the Blood Moor at 300x+ per core, and three runs with the same seed and inputs produced identical `monsters` output. Happy to turn it into a
> PR with a smoke test (`step 250` then `clock`) if the shape is acceptable.

Issue 2: "devctl: read-only `dump [sections]` verb returning JSON". Body: one versioned JSON document per section for the player, stats,
items (code, location, size, quality, sockets), quests and waypoints (save layout) and panel flags, so tools need no per-verb text parsers; read-only; we
would add tests to `tests/smoke_d2d.py`.

Both are posted only if the user decides to, under the user's account.

## Where the fork lives (2026-10-30)
Private forge repo `<owner>/d2d-fork` (ssh://nas.example:2222/<owner>/d2d-fork.git), pushed from `~/d2d-lab/fork/src` on the worker host (remote `forge`, branch main; clones were shallow and were unshallowed first). Never mirrored into d2lod-bot.

## 10. Standard launch toggles (2026-10-30, agent D2DTOGGLE)

Upstream d2d ships QoL deviations that are on by default and switchable with `--toggle name=off[,name=off]` (`apps/d2d/main.cpp` `kToggles`: `light_grid`,
`trans_roof`, `autoloot`, `autoparty`, `pad_controls`, `coop_lobby`; SRC `~/d2d-lab/src/apps/d2d/main.cpp:1688`, parse at 1806 to 1823). `autoloot` makes the player pick up
gold by walking within half a cell; the original needs a click, so our early probes (death gold, drops, d2dwalk) ran with a behaviour the bot never sees live.

Our the worker host launch path now passes `--toggle "${D2D_TOGGLES:-autoloot=off,autoparty=off}"` by default (V: `bash -n`, and a live run showed gold stays on the ground):
- `~/d2d-lab/fork/start.sh` and `~/d2d-lab/start.sh` (so `d2dstart.sh` and `restart.sh` inherit it);
- `~/d2d-lab/fidelity/fid.py` `start()` (it launches d2d itself, not through start.sh).
Override: `D2D_TOGGLES=autoloot=on ~/d2d-lab/fork/start.sh ...` (any `--toggle` argument given on the command line comes after and is parsed after, so it also wins for the same name, U).
No fork code or patch is involved; the scripts live outside this repo. `light_grid`, `trans_roof` etc. stay at their defaults (they do not change game state).

Bot side: picking up gold on d2d goes through the existing verb `cmd pickup <unit>` (`d2dctl.Client.CmdPickup`, `d2dinput.Controller.Pickup`, also what `Interact` maps a type-4
ground unit to, docs/d2d-backend.md section "Phase 2b"); V-d2d that it walks to a pile and takes it. Nothing in `internal/d2dinput`, `d2dwalk` or `d2dctl` relied on auto pickup
(grep for gold/pickup: no code path assumes it); the unit test `TestController` pickup row and `client_test.go` cover the verb mapping.

## 11. lab3 verb reference (2026-10-30, agent D2DVERBS)

Fork branch `lab3` (= `lab2` + the commits below, each its own commit). Requests came from docs/d2d-town.md "Fork verb requests". Replies follow devctl style: `ok` / `ok <detail>` / `err <why>`. Tags: V = verified in a private lockstep run of `d2d.lab3`, S = read in source only.

| Request | Result |
|---|---|
| (1) click with modifier / pack to belt | **Not done as a modifier.** d2d has no shift-click-to-belt behaviour at all (S: the only `SDL_KMOD_SHIFT` readers are the belt-use hotkey and the stat-point button, `town.cpp`), so a modifier on `click` would change nothing for a pack potion. Done instead: `cmd belt put <item id> [box]` (picks the item up, then puts it in belt box `box`, default 0) and `cmd tobelt <box>` (the held item into box 0..15). The existing `cmd belt <slot> [merc]` (use a belt item) is unchanged. |
| (2) `cmd use <id>` for portals, chests, waypoints, shrines | **Already exists as `cmd interact <id>`** (upstream, S: `World` handles `cmd::Interact`): an NPC/preset index (stash, waypoint, chests and shrines are presets: list them with `debug objects` / `debug presets`), or `-2000 - k` for the player's town portal `k` from `debug portals` (it walks there and takes it), `-3000 - k` for a corpse. `cmd use <item id>` is taken (uses an inventory item, e.g. a scroll), so no new verb was added. |
| (3) panel rects in `dump ui` | `dump ui` gains `ui.rects` (see below). |
| (4) `debug wp` | **Already exists** (`debug wp <index>` sets that waypoint bit, upstream). lab3 adds `debug wp all` (all 39) and `debug wp level <Levels.txt id>`. |

`ui.rects`: only open things appear. Every rect is `[x, y, w, h]` in 800x600 game pixels, derived from the numbers the hit tests use (so a click at the rect's middle hits). Keys: `inv_grid`, `inv_coin` (pack gold coin, 483,454 20x18), `stash_grid`, `stash_coin` (153,85 20x18), `cube_grid`, `store_buy`, `store_sell`, `store_repair` (the repair toggle) and `store_repair_all` (at repair vendors; else `store_spare` and `store_close`), `store_stock`, `store_tabs` (list of 4), `belt` (list of boxes, row 1 and the popup rows), `wp_tabs` (open acts), `wp_tab` (active), `wp_rows` (list of `{"level": id, "rect": [...]}`, activated waypoints only), `wp_cancel`, and `gold_box_up|down|ok|cancel` while the gold box is up.
Note: our old constant "repair (368,461)" is the **repair all** button (store_repair_all); the repair toggle is at x 300..331.

Verification status of lab3 (private lockstep run, `verify-lab3.py` / `verify2-lab3.py`, fork commit 8b649b5): V: `cmd belt put 8 0` moved a pack potion into belt box 0; `dump ui` rects `inv_grid`, `inv_coin` (493,463 centre, matches our constant), `stash_grid`, `stash_coin` (163,94 centre vs our 165,92), `belt`; `debug wp all` (mask 2^39-1) and `debug wp level 3`. Note `belt put` onto an occupied box swaps (the displaced item ends on the cursor), so choose a box. **Not verified (U)**: the store, waypoint and gold-box rects (the run did not get the vendor or waypoint panel open: `cmd interact` on Akara/waypoint within 150..200 steps showed no panel, and `cmd trade 4` needs the menu first), and `cmd interact -2000` / chests: `cmd use <tsc id>` in the Blood Moor opened no portal in `debug portals`, so the portal path stays right-click-the-scroll (docs/d2d-town.md). The Blood Moor has chests (`debug objects` rows 0 and 3) that were not operated. `dump` quests waypoint masks update only after a `step`.

## lab4 (lockstep speed, agent D2DSPEED, 2026-10-30)

Branch `lab4` on the forge (`<owner>/d2d-fork`) = `lab3` + 3 small commits to `apps/d2d/main.cpp` and `apps/d2d/lockstep.cpp`; worktree `~/d2d-lab/fork/wt-lab4`, build dir `build-lab4`; binaries on the worker host `bin/d2d.lab4` (floor only, no effect), `bin/d2d.lab4.1`, `bin/d2d.lab4.2` (current; commit, sha256 in `bin/d2d.lab4.2.txt`). `start.sh` still defaults to `d2d.lab2`.

Why (V, measured with `D2DCTL_TRACE`, docs/d2d-pace.md): under `--lockstep` every devctl round trip took one 16.6 ms frame, because `devctl::Channel::pump` runs once per frame, idle lockstep frames were paced at the 60 fps budget, and every idle frame was drawn (software render avg 9 ms + present avg 4 ms in the `alive:` line). Stepped frames are not paced and not drawn, which is why the bare game steps at 120 to 377x but a bot that reads between steps did not.

Patch (verbs unchanged, no new verb):
1. idle lockstep frame: `SDL_Delay(1)` instead of the 16 ms pace (the floor keeps an idle d2d off 100 % CPU);
2. idle frames and the last frame of a `step` are drawn at the time-scaled cadence (every 100 ms of real time) instead of every frame; `step n render` still draws every step frame. A `screenshot` right after a `step` can therefore show a frame up to 100 ms old: sleep 150 ms or use `step n render` first.

Not changed: the world logic does not depend on drawing (stepped frames already skip the draw, V by reading `town.cpp` `skip_draw`). Another d2d client behaviour that helps without any patch: `pump` dispatches EVERY complete line in its receive buffer in one frame (V reading `devctl.cpp`, live: the pipelined snapshot of D2DSPEED costs one round trip), so a batch of read verbs is one frame; lines behind a `step` wait for its reply.

### lab5 (D2DSPEED round 2)
Branch `lab5` (the forge `<owner>/d2d-fork`) = lab4 + one commit, worktree `~/d2d-lab/fork/wt-lab5`, binary `bin/d2d.lab5` (sha256 in `bin/d2d.lab5.txt`). In lockstep: frames never render except during `step n render`, or when a `screenshot` is requested (the reply is deferred until the next frame has been drawn); the idle loop blocks in `poll()` on the control socket (`Channel::wait`) instead of a 1 ms timer, so a buffered command is dispatched immediately. Verbs unchanged. Result: verb cost 10 to 60 us, step about 0.5 ms (docs/d2d-pace.md Round 2). Determinism re-check: not re-run (U): world time still advances only by step frames and drawing was already skipped on stepped frames, so the simulation path is unchanged, but run `det.py` against lab5 before relying on it for replicated soaks.

## lab6 (2026-10-30, agent FORKREBASE)

Branch `lab6` on the forge `<owner>/d2d-fork`, commit `e417d2b8` (on that remote) = upstream main `32768ddb` (has the `kill_exp` fix 01337ba3 and our merged PRs #127, #128) + the lab patches still needed (lockstep clock, `dump`, `dump vendor`, `debug layout/nearby`, belt put, `dump ui` rects, `debug wp all|level`, the four lockstep speed commits) + one new commit, "lockstep: make level linking deterministic" (DETLOOP's request, docs/d2d-determinism.md 4.1). The price and cast patches were dropped (upstream). Binary `~/d2d-lab/fork/bin/d2d.lab6` on the worker host (lab5 untouched). Patch inventory, verification and the divergence finding: docs/d2d-lab6.md.

Verification in one line: upstream tests OK; det.py passive scenario 3/3 identical and equal to lab5; fight scenario 3/3 identical but different from lab5 (rules changed); lab3 verb script identical to lab5; lockstep speed 6 % faster; the first world-side divergence in five bot runs is gone, bot-side refetch divergence remains (tick 2394 to 2619, seed 23).

Migration: use lab6 for runs past clvl 7 (`D2D_BIN=$HOME/d2d-lab/fork/bin/d2d.lab6` or `D2D_LAB=lab6` with `~/d2d-lab/fork/start.sh`; the default stays lab2); lab5 stays for comparison; never mix builds inside one run.
