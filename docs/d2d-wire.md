# Whole bot on d2d (agent D2DWIRE, 2026-10-30)

`d2bot --backend d2d` now builds the same behaviour stack as the Game.exe path (`cmd/d2bot/behaviors.go`), over d2d seams.
Code: `cmd/d2bot/d2dwire.go` (`buildD2DStack`, `d2dHealth`), `cmd/d2bot/d2dbackend.go` (`runD2D`), `d2dcombat.Attach`.
Tags: V = offline test named, U = needs the live d2d.

## What runs (needs `--gamedir`/game tables; without them the "not yet" stubs stay and a warning says so)

| Piece | Source | Notes |
|---|---|---|
| Fighter | `d2dcombat.Attach` over the backend env: `combat.Fighter` with viewport reader, `Input` (cast by skill id, belt verb), `Leave = Cycler.LeaveGame`, `nav.Yield` = PackYield. Skills are bound with `Rig.BindSkills` in `Prepare` and after every leveler spend (replaces `EnsureHotkeys`). | V `TestBuildD2DStackHookups`, `TestRunD2DWholeStackOnFake` |
| Town | `town.New` with `d2dtown.HoverFirst(be.Ctl)` input and `UseSource(d2dtown.Adapter.Source())` (dialog points, fixed vendor tabs, vendor dump); `nv.Escape = ToTown`, `ClearCursor`, loot judge | V construction; vendor/dialog steps U beyond docs/d2d-town.md |
| Task deps | `newTaskDeps` (Dialog, Objects, Cube, HireMerc, Waypoint, Quests with the d2d quest source) plus `Boss`, `Farm` (bossGame) and `ClaimMerc = claimDep(d2dmerc.Claimer, d2dmerc.Reader, Normal)` | V wiring; campaign steps U live |
| Leveler, Gear, Quester | as on Game.exe (`leveler.New` with `Rebind = BindSkills`, `townvendor`, Akara claim) | leveler reads base stats through `Mem`, which is `noMem` on d2d: its "unreadable" path (U) |
| Class start | `Prepare` runs `applyClass` (combat profile, leveler build) then `BindSkills` | |
| Merc | `"merc": mercMount()` mount; `installMercSeam` wraps the Town seam after the stack's own assignment (the `init()` in mercwire.go was removed: it ran before any seam existed). `merc.enabled` stays 0 | V `TestBuildD2DStackHookups` (wrapper when on), existing mercwire tests |
| Loot | NOT wired (`Seams.Loot` nil: no pickup). The looter exists only as the town's item judge and for bossGame | pixel pickup on d2d unverified; autoloot is toggled off in the runs |
| Liveness | `Probes.Heartbeat = Beat.Next` (clock/frame counter), `HangAfter` 90 s (monitor time = env clock, virtual under lockstep). A stalled counter with a still-answering ping gives `EventHung`, which the runner's observe loop feeds into the same failsafe ladder as a frozen Game.exe | V `TestD2DHealthStalledBeatIsHung`; a really wedged d2d never reproduced (U). Lockstep risk (U): if the frame counter stops while the world is not being stepped, a long idle could read as hung |

Left off on purpose: `Recycle` (Game.exe heap hygiene), `SavedExp` (no save dir on throwaway runs), `--dry`.

## Socket safety

The d2d control socket takes ONE client and has no authentication. Rules now enforced in `cmd/d2bot/d2dbackend.go`:
- `--backend d2d --attach` is refused unless `--d2d-attach` (env `D2D_ATTACH`) is also given: attach means "drive a d2d I did not start", say so. V `TestBackendFlags`.
- `--managed` writes `<sock>.owner` after it starts d2d and removes it at exit. A managed start on an answering socket is an error; `--clean-start` (supervise restarts) only quits a d2d when the marker exists. A stranger's d2d is never driven or quit. V `TestManagedRefusesStrangerSocket`, `TestOwnerMarker`.
- The default socket is per instance: `<run dir>/d2d-<D2BOT_INSTANCE>.sock`, never `~/d2d-lab/fork/d2d.sock`. Do not pass the shared path to a bot; run your own d2d on your own socket.

## Race found on the way

`d2dsrc.Source.warnOnce` wrote an unlocked map; with the real stack several goroutines read the same Source (runner, quest source, merc, town) and `-race` caught it. Fixed with a mutex (`internal/d2dsrc/source.go`).

## Test

`go test -race -count=1 -p 1 ./cmd/d2bot ./internal/d2dcombat ./internal/d2denv ./internal/d2dtown ./internal/d2dmerc ./internal/d2dsrc`.
The whole-stack test uses header-only synthetic game tables (`synthTables`), so no Blizzard data is needed; combat and town run against the fake, which has no real world, so only wiring is proven.

## Live results (private fork d2d.lab 07453d1, throwaway Sorceress "Wirey", seed 5, `--lockstep`, `--no-save`, nice 10, load average about 22)

Command (own socket `~/d2d-lab/fork/wire-run/d2d-wire.sock`, own data dir, `flock -n /tmp/d2d-lab.lock`):
`d2bot --backend d2d --managed --no-save --d2d-class 4 --char Wirey --seed 5 --toggle autoloot=off,autoparty=off --d2d-args "--lockstep" --d2d-linkwait 60s --gamedir ~/d2d-lab/data --goals campaign --run-for 11m`

What worked (V live): the real stack boots on d2d; campaign goal runs; class-aware start (`sorceress/lightning`, build `fallback-lightning-v2`) and `BindSkills`;
town exit by level border (`nav exit ... border:true`), Rogue Encampment -> Blood Moor -> Den of Evil (planner picked the Den quest at clvl 1);
combat kills (Fallen, Fallen Shaman, Zombie; 13 to 16 kills per game), potions drunk from the belt (`combat potion kind=healing`), death recovery
(`character died` -> `death overlay` -> town chores -> shop visit, `poverty mode` with gold 0), nav stuck/yield ("stalled by monsters, yielding"), the
failsafe ladder, `Cycler.NewGame` restarts. Free Rogue claim, merc, waypoint, Akara reward: not reached (the character never left clvl 1).

Found and fixed during the runs (each has a commit):
1. `d2dinput.CloseAll` could not close a vendor window (no cancel line): post-death chores looped on "panels still open" (shop flag 12) until the ladder restarted the game.
   Fix: one Escape when `dump ui` says vendor (V: chores completed afterwards).
2. Combat's town portal needs the inventory panel flag, which d2d never reports ("inventory panel did not open" x3, then chicken exit = the game was left). New `d2dPortal` hook
   (open with `i`, fixed wait, pixel from `debug itemat`, right click). U whether it casts: in the last runs `debug itemat` ended in a client i/o timeout (see 4).
3. `d2dsrc.Source.warnOnce` data race (above).
4. Health probe: a missed 3 s ping under lockstep load was read as "exited" and restarted a healthy game three times (halt, exit 3). Pings now get a 45 s grace unless
   `Proc.Gone`. Then the heartbeat flagged "hung" instead: counters (game_ms, tick, stepped, clock frame, info frame) were all flat while the socket file had vanished
   (`dial ... no such file`), i.e. the d2d was dead or wedged (control reads timed out for 20 s first). Beat now accepts any moving counter and logs the flat set.

5. Final diagnosis of the "hung" restarts (V live, d2d.out kept): the d2d was healthy (`alive: 470 frames/5s`, a town portal had just been cast: `town portal: Den of Evil <-> Rogue
   Encampment`, so `d2dPortal` works at least once, V) while the bot restarted it. The monitor's time is the bot's virtual clock and the game counters only move when the bot steps,
   so under lockstep a flat counter says nothing. Heartbeat is now a wall-clock beat while lockstep is on (a dead d2d is still caught by `Alive`: `Proc.Gone` or 45 s of silent pings);
   the counter-based Beat stays for free-running d2d (U live). V `TestD2DHealthLockstepNeverHangs`.

Open (not fixed): the last run still hit the runner's own "no progress in state resupplying for 240 s" ladder after the emergency portal ("no town portal and no waypoint: walking back
via the neighbour"): the runner's progress clock is virtual too and the portal walk-in on d2d (`debug portals`, objects class 59) was not confirmed. Repair is "not confirmed (durability 101%
unchanged)" (the repair button geometry is a Game.exe model), stash NPC "not found or not reachable", mana 0 at clvl 1 with Fire Bolt only (out-of-mana waiting dominates), gold stays 0 (no
pickup: loot is off), so no potions are bought, a clvl 1 Sorceress dies or chickens in Den of Evil (the planner chooses it at clvl 1 with "floor waived").
