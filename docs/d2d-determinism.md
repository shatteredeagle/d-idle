# Determinism of the full bot loop on d2d (agent DETLOOP, 2026-10-30)

Tags: V = verified live on the lockstep fork `d2d.lab5` (how stated) or by an offline test named here, S = read in the fork's
source, U = unverified. Card: `openspec/changes/detloop`. Background: docs/d2d-loop.md "Determinism of lab5" (scripted walks and
fights are deterministic: det.py; two full-bot runs with one seed were not).

Question: why do two full-bot runs with the same seed, the same fresh character and the same binaries send different commands
or see different worlds, and which of the causes are ours.

## 1. The divergence finder

Three parts, all stdlib Go and shell, owned by this card:

- **Journal** (`internal/d2dctl/journal.go`, on when the bot runs with `D2DCTL_JOURNAL=<file>`): every command the client sends,
  in order, with the lockstep tick it was sent at and a digest of the reply (`>  tick  command  digest`), plus a **checkpoint**
  every `D2DCTL_JOURNAL_EVERY` ticks (default 250): the sha256 of the replies to `dump player,stats,items,quests,ui,merc`,
  `monsters` and `ground`, taken right after the `step` that crossed the interval, on the same connection (`C  tick  hash`).
  Both runs therefore take the checkpoints at the same ticks, between the same two commands, so the probe itself does not make
  the runs differ (V `TestJournalRecordsAndCheckpoints`: two identical fake runs give byte-identical journals). Left out: `ping`,
  `info`, `clock`, `screenshot` (the health monitor sends them on a wall timer; they read nothing the world depends on); the
  `frame` field of a dump's `time` section (the process frame counter differs by the start-up and idle frames, V first pair);
  `step` and `clock` replies are recorded as `tick=<n>` instead of a hash for the same reason.
- **Comparer** `cmd/d2ddet`: `d2ddet compare A.jnl B.jnl [-context N] [-ignore verb,...]` prints the first differing record,
  the identical prefix, the last equal checkpoint and a classification: a different COMMAND on equal state so far is the bot
  (it decided differently on the same observation), a different REPLY or STATE HASH under identical commands is the game (or an
  earlier bot effect that only shows in a read). `d2ddet stats A.jnl` counts verbs. Exit 0 identical (a shorter run that is a
  prefix of the longer one counts as identical), 1 divergent.
- **Runner** `tools/d2ddet/pair.sh TAG BOTBIN SEED TICKS [RUNFOR] [N]` on the worker host in `~/d2d-lab/det` (put there by
  `tools/d2ddet/build.sh`, which builds `d2bot.det` and `d2ddet` on the worker host through labrun): N runs of the full bot (`--backend d2d
  --managed --d2d-class 4 --char Dety --seed SEED --goals campaign`, lockstep lab5, fresh character in a run-private data dir,
  `D2DSOAK_HOURS` = TICKS/90000 so the run stops after that many virtual ticks) one after the other under `flock -w 3000
  /tmp/d2d-lab.lock`, then every run compared with the first. Journals `TAG-A.jnl`, `TAG-B.jnl`, ... bot logs `bot-TAG-X.log`.

Exact commands:

```
tools/d2ddet/build.sh                                   # The head box: build d2bot.det + d2ddet on the worker host into ~/d2d-lab/det
ssh worker 'cd ~/d2d-lab/det && ./pair.sh p2 d2bot.det 21 5000 6m 3'   # 3 runs, seed 21, 5000 ticks, compare B and C with A
ssh worker 'cd ~/d2d-lab/det && ./d2ddet compare p2-A.jnl p2-B.jnl -context 10'
```

## 2. Sources of divergence

Pairs p1 to p3 (seed 21, 5000 ticks, fresh Sorceress, bot of commit e9fb299 and 436b763, V): every pair diverged at **tick 2**,
the first action, with identical state hashes: the first divergence is a COMMAND, i.e. bot-side.

| # | side | source | evidence | status |
|---|---|---|---|---|
| 1 | bot | **The runner's guard is a second goroutine on the lockstep clock.** `Runner.guard` sleeps `WatchMs` on `env.Clock` and polls a snapshot while the action runs on another goroutine; under lockstep that Sleep is `Pacer.Sleep`, so the guard STEPS the game itself, and the order in which its steps and reads interleave with the action's is Go scheduling. The action's `cmd move` then lands at a different tick in each run. | p2-A/B and p3-A/B: at tick 2 run A has a full snapshot read, then a second reader's `dump items, monsters, npcs, ground, debug portals` (journal sites `d2dsrc.(*builder).world`, `Source.dump`: a concurrent `Source.Read`), run B the other order | fixed (3.1) |
| 1b | bot | **Prefetch cross-talk**: `Client.Prefetch` keeps one store per client; a concurrent `Source.Read` consumes the other reader's prefetched `state`, `debug players`, `dump` head replies (served without a round trip, so they do not even appear in the journal) and the first reader falls back to round trips. Harmless to the world (reads), but it made the interleaving of source 1 visible and would make any future concurrent reader non-reproducible. | p3 journals: the second reader's records start at `dump items` with no `state` before them | moot after 3.1 (no concurrent reader is left under lockstep); not changed |
| 2 | bot | `task.Executor` strike expiry (`strikeWindow` 90 min) used `time.Now` (no `Now` was wired): a quest strike expired by wall time, so a slow and a fast run disagree on whether a quest is blocked | read; `Executor.now()` fell back to the wall clock | fixed (3.2) |
| 3 | bot | `goals.Manager` ticks once a wall second (`Options.Clock` default `time.Now`): `ActiveSeconds`, pause/resume times and the autosave are wall-clocked. For a `campaign` goal nothing of it reaches the runner's decisions (the goal is active from `Start()` and ends by progress), so it does not change the command stream; a time-boxed `farm` goal (`farmRemaining` on `farm.spent`, wall `time.Since` in `internal/bot/farm.go`) WOULD. | read | open: wire `goals.Options.Clock = e.Clock.Now` in `cmd/d2bot/d2dbackend.go` (owned by card d2d-default-backend, not edited here) and `farm.spent` onto `r.e.Clock` (not needed for campaign runs) |
| 4 | bot/d2d | Neighbour-link wait: `d2dworld.waitLinked` polls until the wall deadline `LinkWait`, each poll 2 ticks + 10 ms wall (`LateClock.Poll`); how many polls (ticks) pass before d2d has linked the level depends on the fork's builder thread (section 4.1) | read; D2DEXIT/ECONOMY live logs (11 to 16 waits per 2 h run) | blocked on the fork (4.1): with the one-line lockstep change the link lands on the first tick after arrival and the wait ends after one poll in both runs |
| 5 | d2d | the `frame` counter in `dump`'s `time` section and in `step`/`clock` replies counts wall-driven idle frames | p1: two `dump ...time` reads at tick 0 differ only there | not a bot input (`d2dsrc` keeps it in `Extras.Clock.Frame`, nothing decides on it, S read); the journal masks it |

Not sources (checked): nav and combat RNGs are seeded with a constant (`rand.NewSource(1)`; `Fighter.SetSeed`); nav, combat, town
and the runner's timers use `env.Clock` (virtual under lockstep, docs/d2d-pace.md); the health monitor's `ping`/`clock`/`info` and the
`traceLoop` are wall-timed but read-only and never step; `d2dinput.Controller` waits go through the Pacer (`hoverFirst.wait`).
Map iteration order in decision code was not audited line by line: the finder would show it as a COMMAND divergence at equal
hashes, and after 3.1 none appeared within the runs below.

## 3. Fixed on our side

3.1 `Pacer.Watch(every, fn)` (`internal/d2dinput/pacer.go`): a watcher that runs ON THE STEPPING GOROUTINE right after the step
chunk that crossed a multiple of `every` virtual ms (once per chunk, not re-entered when fn itself sleeps; `stop` unregisters).
`LateClock.Watch` exposes it under lockstep and returns nil on a wall clock. `Runner.guard` (`internal/bot/loop.go`) uses it when
the env clock offers it: the guard's goroutine only waits for the action to end, the checks (`guardOnce`, the old loop body
unchanged) run inside the action's own `Clock.Sleep` calls at deterministic ticks. Game.exe: `env.RealClock` has no `Watch`, the
goroutine loop is as before. V `TestPacerWatchRunsOnTheSteppingGoroutine`; `go test ./internal/bot` green.
**Game.exe-affecting: `internal/bot/loop.go` (guard split into `guard` + `guardOnce`, same checks in the same order) needs a G1/G2 check.**

3.2 `internal/bot/task.go`: `ex.Now = r.e.Clock.Now` (strike expiry on the env clock; wall on Game.exe as before, one line).
**Game.exe-affecting in name only (RealClock.Now is time.Now); listed for the record.**

3.3 Journal/finder (section 1) and the `frame` masking.

## 3.4 Verification status (V: pairs p4 and p5 ran with `d2bot.det3`, commit f0cd13f)

| pair | bot | seed | ticks | runs | result |
|---|---|---|---|---|---|
| p1, p2, p3 | before 3.1 | 21 | 5000 | 2 to 3 | DIVERGE at tick 2, record 22 to 32 (source 1, every pair) |
| p4 | 3.1 + 3.2 | 21 | 5000 | 3 | B = C for **20861 records and 12 checkpoints (to tick 3002)**, first divergence tick 3068: run B refetches the level (`debug level` then `debug grid 5`, `debug layout`, `debug nearby`: `d2dworld.update` after the neighbour list changed), run C steps on. A (run 6 minutes earlier under another load) = B/C for 113 records only: the `state` reply differed at tick 8 (source 6) while every checkpoint stayed equal |
| p5 | same | 22 | 12000 | 2 | A = B for **21804 records and 12 checkpoints (to tick 3002)**, first divergence tick 3230, same shape (`debug grid 5` refetch in one run only) |

Comparisons with `-ignore state` (p4-A differs in `state` only; the journal now masks the field, see source 6). So after the
guard fix two same-seed runs send the same commands and see the same world for about 3000 ticks (2 virtual minutes: the town
chores, the walk out, the first fights), and part ways at the first level change, where the refetch depends on when d2d
installed the neighbour level (source 4, fork). Whether the refetch itself is the only consequence of source 4 is U: after it
the two command streams are offset and the journal cannot say what else differs.

The gate "N=5 pairs of 10000+ ticks identical at every checkpoint" is therefore NOT met: 2 pairs ran (plus one with a load
artefact), both identical to tick 3002 and both diverging at the link path. Re-run after the fork change (4.1):
`tools/d2ddet/build.sh 3` then on the worker host `cd ~/d2d-lab/det && ./pair.sh p6 d2bot.det3 23 12000 8m 5` (five runs of one seed,
four comparisons). New runs go through the slot pool (`~/d2d-lab/d2dslot.sh run -- <script>`, COMPUTE); `pair.sh` takes the
legacy lock (slot 0) and keeps working.

Source 6 (d2d, found by p4): the `state` reply carries `cam=x,y`, the camera, which follows the player with a smoothing that
runs in wall time (U: inferred from a hash difference at tick 8 with equal dump/monsters/ground and `walking` unchanged; the
raw text was not kept). `d2dsrc` does not read it (`internal/d2dcombat/viewport.go` says so); the journal masks it like `frame`.

## 4. Remaining, and what they need from the fork

### 4.1 Neighbour levels are built on a wall-time thread pool and installed on the first tick that finds them ready (d2d-side, S)

`components/game/gamedata.cpp` (fork, base e36922e): `GameData::want_level(id)` queues `std::async` jobs on a builder pool;
`GameData::poll_levels()` runs once per world tick (`components/game/world.cpp`, the tick loop) and installs every job whose
future is ready at that moment (`wait_for(0s)`). `want_nearby` runs on the first tick the player's level changed. So the tick at
which a neighbour level (its monsters, its walkable edge strip in our grid) appears depends on how long the build took in WALL
time versus how fast the bot steps. The bot's link wait (`d2dworld.waitLinked`, `LateClock.Poll`: 2 ticks + 10 ms wall per poll)
cannot make this deterministic from outside: it can only wait longer, and the install tick still depends on the pool.

Fork request (lab6, one line, no new verb): under `--lockstep`, `poll_levels` waits for every pending job instead of polling
it (`job.wait()` in place of `wait_for(0s)` when the lockstep clock is on), so a wanted level is installed on the FIRST tick after
it was wanted, whatever the build took in wall time. Cost: that one stepped frame blocks for the build (100 to 300 ms, V d2d logs
of D2DEXIT); nothing else changes outside lockstep. Alternative if blocking a frame is unwanted: a verb `linkwait` that blocks
until the builder has no pending job and replies `ok pending=0`; the bot would call it after every level change before its next
step, and `poll_levels` would then install on the next tick in both runs. The first form needs no bot change and cannot be
missed by a code path; prefer it.

## 5. Replay readiness: what a replay file must contain

The REPLAY agent (agent/replay) ships `detloop-journal-v1`: its checkpoint hash is byte-identical to this journal's `C` record
(same verbs, same joining), so a journal of a run and a recording of the same run agree checkpoint for checkpoint.

For the REPLAY agent (`cmd/d2dreplay`, `internal/d2dreplay`; nothing of it is touched here). A replay is a lockstep d2d plus a
command stream; it reproduces a run exactly as far as section 2 and 4 allow (the fork request in 4.1 is the precondition for a
bit-exact replay across level changes).

Header (one JSON line, `{"v":1, ...}`):

| field | value | why |
|---|---|---|
| `seed` | d2d `--seed` (int) | the map and the world RNG draw order start from it (docs/d2d-fork.md section 2) |
| `class`, `name` | `--start-class`, `--start-name` of a NEW character, or |
| `save_sha256` | sha256 of the `.d2s` the run started from (and the file itself next to the replay) | a loaded character is start state; a fresh one is `class`+`name` |
| `d2d_build` | the fork binary's sha256 and its `git describe` (lab5: `~/d2d-lab/fork/bin/d2d.lab5.txt`) | a different build is a different world |
| `d2d_args` | the full d2d argument list (`--lockstep --toggle ... --headless`) | toggles change the simulation (autoloot, autoparty) |
| `bot_build` | `buildinfo.Version` of the recording bot, informative (playback does not run the bot) | |
| `every` | checkpoint interval in ticks | |

Body: the journal records of section 1, in order, one per line (tab separated):

- `>  tick  command  digest  site`: every verb sent, at the tick it was sent (the tick BEFORE a `step`, i.e. the state the command
  applied to). Reads (`state`, `monsters`, `dump ...`) may be dropped on playback or kept to re-check digests; writes (`click`,
  `key`, `cmd ...`, `step n`, `debug ...`) are the input log and are replayed verbatim in order. `step n` is the clock: playback
  sends the same `step n` batches, so a player at tick T has sent exactly the writes recorded before that tick.
- `C  tick  hash`: the checkpoint hash (sha256, 16 hex digits, of the replies to `dump player,stats,items,quests,ui,merc`, `monsters`,
  `ground`, each reply's lines joined with `\n`, replies separated by a NUL). Playback takes the same reads at the same ticks and
  stops at the first mismatch with the recorded hash: that is the replay's own divergence detector.

Rules the recorder must keep: one client per d2d (the channel is single-client); the health monitor's `ping`/`info`/`clock` are not
part of the stream; a restart of the d2d (chicken, Save and Exit, failsafe) is a new game: record it as a header line again
(`{"v":1,"restart":true,...}`) and the character's save hash at that point, because the next game loads it. Rendering to video is
`step n render` per batch plus `screenshot` at the chosen cadence, on playback only.

## 6. DETLOOP2: the `debug grid 5` divergence (2026-10-30)

Tags: S = read in source, V = offline test, U = not verified live (the slot pool was full, see below).

Finding (S): p6 (lab6, seed 23) was identical up to tick 2394..2619, then one run sent `debug grid 5` (journal site
`d2dworld.(*World).fetch`) and another did not, on the same observation. `d2dworld.World.due` gates every grid refetch
on `Options.Now().Sub(fetchedAt) >= MinRefetch` (2 s), once for the pending-neighbour retry (`l.pending > 0`, up to
`maxLinkTries`) and once for the moved-far rule. `Options.Now` defaulted to `time.Now`, and `d2denv.Build` did not set
it: the refetch decision was made against WALL time. Under lockstep a run's wall time per tick depends on machine load,
so whether 2 s had passed when the bot looked again differed between runs (the load in p6 was 12 to 15).

Fix (S, V offline: d2dworld tests unchanged and green): `d2denv.Build` passes `Now: clk.Now` (`LateClock.Now`: the Pacer's
virtual time, game_ms based, under lockstep; wall time on a free-running d2d), and `d2dcombat.NewRig` passes
`Now: r.Pace.Now`. The refetch rate limit is now 2 virtual seconds (50 ticks). Other `time.Now` uses in d2dworld
(`waitLinked`'s deadline, log timing) only bound or time the link wait, which ends on its first poll with the lab6
link-wait patch, so they do not change the command stream (U: not looked at again after a link-wait timeout).
`cmd/d2dwalk` and `cmd/d2dmap` keep wall time (not part of the bot loop).

Needs a G1/G2 check: none for Game.exe (the Game.exe path does not use d2dworld). On a free-running d2d `LateClock.Now`
is `time.Now`, so behaviour there is unchanged.

Tooling added: `D2DCTL_JOURNAL_RAW=state,dump` writes the full replies of those verbs to `<journal>.raw` (blocks
`@ tick line` + payload), to see WHICH field of the start-up `state` reply differs between two runs (the journal
digests only show that it does; `cam=` and `frame` are already masked, so something else in `state` is wall-driven).
The checkpoint hash (`C` records, `detloop-journal-v1`) never includes `state`, so the start-up difference does not
affect replay and the hash definition is unchanged (no v2). `tools/d2ddet/pair.sh` now uses lab6 (`D2D_BIN` to
override) and the slot pool, with `PAR=n` runs at once.

Result (V live, lab6, seed 23, `PAR=3 pair.sh q1 d2bot.det.d2 23 12000 8m 5`; the slot wait was about 80 minutes, then A, B, C ran
in parallel; D and E never started, the pair script was killed meanwhile): `d2ddet compare -ignore state` of A, B and C: all three
are identical for 81617 to 81838 records and 48 checkpoints up to tick 12002 (previously the first divergence was at tick 2394
to 2619). The only difference left is at tick 12056 to 12095: the end of the run (the `D2DSOAK_HOURS` stop; one run sends
`debug players` where another already steps 3 more ticks), BOT-SIDE and not the world. 3 of 3 pairs identical for 12000 ticks;
5 of 5 and the 50000-tick target are not measured. Open: the stop condition of the soak loop is still not tick-based (not
looked at), and the start-up `state` field is unidentified (`D2DCTL_JOURNAL_RAW`).
