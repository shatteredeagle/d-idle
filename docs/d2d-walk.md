# d2d end-to-end walk: cmd/d2dwalk (agent D2DWALK, 2026-10-30)

First integration of the separately built d2d pieces with the bot's REAL navigator: `internal/d2dctl` (client),
`internal/d2dsrc` (snapshot source, a `d2.StateReader`), `internal/d2dworld` (level maps), `internal/d2dinput`
(`input.Controller`), `internal/nav` (unchanged behaviour). Nothing in `cmd/d2bot` changed. Private fork only
(docs/d2d-fork.md). Tags: V = observed in the runs below, U = not verified.

## How to run

```
# The worker host, one d2d at a time (pgrep d2d), niced; throwaway character, nothing saved
cd ~/d2d-lab/fork && ./start.sh --lockstep --seed 3 --no-save --start-screen ingame --start-class 1 --start-name Dw &
# build and ship the tool (from the worktree)
GOOS=linux go build -o /tmp/dw/d2dwalk ./cmd/d2dwalk && scp /tmp/dw/d2dwalk worker:d2d-lab/fork/bin/
# walk: town -> Blood Moor (2) -> Cold Plains (3) -> Blood Moor (2) -> town (1)
ssh worker '~/d2d-lab/fork/bin/d2dwalk -sock ~/d2d-lab/fork/d2d.sock -route 2,3,2,1 -life 2000 -wall 15m'
python3 ~/d2d-lab/fork/ctl.py quit      # stop d2d afterwards
```

Flags: `-route` level ids for `nav.ExitTo` in order; `-life N` (PRIVATE FORK ONLY, throwaway character) sets clvl 30 and
life N so wandering Blood Moor monsters do not kill a clvl 1 character (a bare clvl 1 Rogue died in the first run: the
navigator has no combat here); `-linkwait` wall time to wait for neighbour levels on arrival; `-wall`, `-virtual` limits;
`-chunk` ticks per `step` call; `-v` debug nav logs. Exit status 0 only if every leg ended in its target level.

## Wiring (the seam, no cmd/d2bot change)

```
ctl  := d2dctl.New(sock)                         // one client, one socket
src  := d2dsrc.New(ctl, opts)                    // env.Env.Read   (d2.StateReader)
pace := &d2dinput.Pacer{C: ctl}                  // env.Env.Clock  AND input.Clock
in   := d2dinput.New(ctl); in.Clock = pace       // env.Env.In
w    := d2dworld.New(d2dworld.Client{Client: ctl}, d2dworld.Options{LinkWait: .., Sleep: step-and-wait})
e    := &env.Env{Read: src, In: in, Clock: pace, Tune: &tune, Log: log}
n    := nav.New(e); n.UseWorld(w)                // the one additive nav change
```

- **Pacer** (`internal/d2dinput/pacer.go`): `Sleep(d)` does not wait; it sends `step ceil(d/40ms)` (chunks of 50) to the
  lockstep fork and `Now()` is the game's virtual clock (`game_ms` of the reply). nav's deadlines, stall timers and poll
  sleeps therefore behave as in real time while the loop runs at lockstep speed. A failed step still advances virtual time so
  timeouts fire; the first error is kept in `Pacer.Err()`. New verbs `Client.Step` / `Client.Clock` (+ fake support).
- **`nav.UseWorld`** (additive, tested): `env.Env.World` is a `*d2map.Tracker`, so `nav.New` could not take d2dworld. The
  Game.exe path never calls it.
- `d2dworld.Options.Now` is left at the wall clock: its pending-neighbour retry is about d2d's worker thread, which runs in
  real time.

## Result (V, private fork on the worker host, seed 3, throwaway Rogue "Dw", `--no-save`, load average 26 to 31, niced)

Run `walk12` (`-route 2,3,2,1 -life 2000 -clear -linkwait 150s`): **success on all four legs**.

| Leg | Result | Virtual | Wall |
|---|---|---|---|
| 1 town -> Blood Moor (`ExitTo 2`, border gate x=279 y=76, push x=287) | area 2, pos (7,116) | 16.9 s | 29 s |
| 2 Blood Moor -> Cold Plains (`ExitTo 3`, gate x=479 y=182) | area 3 | 57 s | 1m47 |
| 3 Cold Plains -> Blood Moor (`ExitTo 2`, gate x=0 y=102) | area 2, pos (473,182) | 2.5 s | 4 s |
| 4 Blood Moor -> town (`ExitTo 1`, gate x=0 y=125, 480 subtiles west) | area 1, pos (272,85) | 56 s | 1m44 |

Totals: 188 `cmd move` commands from nav, 1076 `step` calls, 3338 ticks (2m13 virtual) in 4m06 wall. Wall time is dominated by
round trips (one nav observation is about 10 channel verbs: snapshot dump, monsters, ground, npcs, `debug level`, `players`)
on a loaded box, not by the game: 3338 ticks cost about 2 s of game time at 3000 ticks/s. Excerpt (our output):

```
== leg 1: ExitTo(2) (map: 11 border cells to it; 3 adjacent entries, 0 levels not linked yet)
level=INFO msg="nav exit" to=2 gate="{X:279 Y:76}" push="{X:287 Y:76}" border=true
level=INFO msg="input ClickWorld" target="{X:160 Y:71}" player="{X:153 Y:68}" vt=520ms err=<nil>
level=INFO msg="input ClickWorld" target="{X:168 Y:74}" player="{X:157 Y:70}" vt=1.24s err=<nil>
  after: area 2 pos (7,116) life 2000/2000 state ingame clock 17.64s
  leg 1 result: err=<nil> area=2 (want 2) wall=29.459s virtual=17.12s
== leg 4: ExitTo(1) ... level=INFO msg="nav exit" to=1 gate="{X:0 Y:125}" push="{X:-8 Y:125}" border=true
  leg 4 result: err=<nil> area=1 (want 1) wall=1m44.291s virtual=56.24s
== summary: ok=true ClickWorld calls=188 step calls=1076 ticks=3338 virtual=2m13.52s wall=4m6.196s pacer err=<nil>
```

The player position trail matches the level geometry (town (153,68) east along y 70 to 85, gate near x=279, Blood Moor from
x=7). The fields d2dsrc cannot fill (`mouse`, `hovered`, `player.mode`, item class, monster hostility, ...) were counted per
snapshot and did not stop nav; nav only used position, area, life, monsters, state.

Helpers used only for the smoke test, all private-fork cheats on a throwaway character: `-life` (clvl 30 and life via
`debug stat 12|7|6`: a clvl 1 Rogue died in Blood Moor in the first run, nav has no combat) and `-clear` (`debug kill`
before every move: Fallens and Quill Rats stand in the Blood Moor corridors and nav without a combat `Yield` stalls on
them, V twice at (211,225)). Without `-clear` one run still passed leg 2 by sidestepping (walk9).

## Integration bugs

Fixed (with tests):

1. **No step/clock verbs in d2dctl.** Added `Client.Step`, `Client.Clock` and `ClockInfo` (+ fake server support, test).
2. **nav could not take a non-Tracker world** (`env.Env.World` is `*d2map.Tracker`). Added `Navigator.UseWorld` (additive, test).
3. **Virtual time**: nav sleeps/stall timers need a clock that moves with `step`. `d2dinput.Pacer` (test).
4. **Neighbour levels link late, and nav gives up at once.** On arrival in Blood Moor `debug nearby` lists only the level
   we came from. Cold Plains appears only after the game has been stepped (V: 135 s of idle polling with no step: never; one
   `step 1`: at once, listed and collision readable), but nav's "no gate yet" path explores a fully known level, finds
   nothing and answers `ErrNoExit` in about 3 s virtual (V: runs walk1, walk2, walk5). Fix in d2dworld:
   `Options.LinkWait` plus `Options.Sleep` (the caller steps the game in it); on arrival the world polls the neighbour list
   and returns when all abutting levels are linked (0.5 s in the final run).
5. **A newly linked neighbour changes our own grid.** The grid fetched on arrival had the edge strip closed (`#`); after the
   link the same cells are open. A border computed from the stale grid had no cells (V: "linked=[1 3]" in the log, no gate for
   90 s). `waitLinked` now refetches the level when the list changes. Also: a level listed as linked whose strip has no open cell
   counts as pending (bounded retries), not as a closed border.
6. **The kick click walked the character.** `click 400 500` (below the player) is not harmless once the quirk is cleared: it
   moved the character south-east, which pushed it back over the Cold Plains border (leg 4 of walk9: "left level 2 for 1 but
   ended in 3"). The kick is now `click 400 300` (the player; V: clicks at (400,300) and (400,285) move nothing). Not
   re-verified that (400,300) still un-sticks `cmd move` on its own, but walk12 (4 level changes, kick on each) moved every time.
7. `d2dworld` retry timing used the caller's clock: with the Pacer clock the MinRefetch wait is virtual while d2d's worker is
   wall time. The tool leaves `Options.Now` at the wall clock.

Open:

- **Combat/monster blocking**: nav has no way to clear a corridor without a combat `Yield`; the smoke test cheats with
  `debug kill`. Next step.
- **Crossing leaves a move command running**: leg 3 returns the moment the level id changes while the unit still walks toward
  the push cell (-8 from the edge); d2d keeps going into the new level. Harmless in the final run, but a stop (`cmd move` to the
  current cell) after a level change would be cleaner (d2dinput could do it when `prime` sees a new level id; not done).
- `debug nearby` linking depends on stepping (S: the main loop consumes the worker's result on a stepped frame): a non-lockstep
  d2d links by itself in real time (not measured).
- Unlinked-neighbour tolerance is a guess: a real wall border to a linked neighbour makes `LinkWait` poll until the deadline
  (no unlinked level left stops it early, see `waitLinked`).
- Tile exits (caves, `Warps`) and doors were not exercised (Blood Moor -> Den of Evil is the next route: `ExitTo 8`).
- Wall cost: about 10 verbs per observation. A `dump`-only snapshot (one verb) would cut walk wall time several times.

## Next step

1. Combat on d2d: `Yield` for nav with `d2dinput.Cast` (skill ids from the class tables), monsters from `d2dsrc`; replace the
   `-clear` cheat. 2. Town layer (NPC menu, vendor, stash: gaps G3/G4 of docs/d2d-backend.md). 3. A bot loop behind a d2d
   backend flag in cmd/d2bot (not done here by instruction). 4. Den of Evil route to cover warps.
