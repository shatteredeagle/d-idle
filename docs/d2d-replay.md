# d2d replay: record a run, play it back, verify determinism (agent REPLAY, 2026-10-30)

Tags: V = verified (offline test named, or live log), S = secondary (read in docs/code), U = unverified.
Contract: `openspec/changes/replay/specs/replay/spec.md`. Code: `internal/d2dreplay`, `cmd/d2dreplay`. Stdlib only.

The lockstep fork is deterministic for a seed and an input sequence (docs/d2d-fork.md section 5, docs/d2d-loop.md
phase 0). A replay file is that input sequence plus state-hash checkpoints: play it back on the same build and every
hash matches; on another build, or with a bot that decides differently, the first divergent tick says where.

## File format (`*.replay.gz`, version 1)

gzip of JSON lines. Line 1 is the header, then body entries, then one trailer. A file killed mid-run (no trailer, or
a cut gzip stream) is still read: `Read` returns the readable prefix with `Truncated` set (V `TestTruncatedFileStillReads`).

```
{"replay":1,"created":"2026-10-30T23:01:00Z","hash":"detloop-journal-v1","every":250,
 "d2d":{"sha256":"<sha256 of the d2d binary>","args":["--toggle","autoloot=off,autoparty=off","--lockstep","--no-save"]},
 "start":{"seed":3,"class":1,"name":"Det","screen":"ingame"},            # fresh character recipe (or "save":{"name","sha256","seed"})
 "meta":{"tool":"d2dreplay record","bot":"...","note":"...","fresh":true}}
{"t":0,"v":"debug stat 12 8","w":2003}        # verb: tick before it, the line, wall ms since the previous entry; "e":"err ..." if d2d refused it
{"t":0,"v":"step 10","w":3001}
{"c":265,"h":"<64 hex>","lv":30,"area":2,"life":"..."}   # checkpoint right after the step that crossed a multiple of "every"
{"r":true,"t":1200}                            # restart marker: the d2d behind the tap was replaced (managed bot restart)
{"end":true,"ticks":2015,"verbs":23,"wall_ms":1234}
```

- Only **mutating** verbs are stored (`Mutating`, V `TestMutating`): step, click, rclick, key, text, move, wheel, pad,
  save, quit, lockstep, timescale, every `cmd`, and `debug` except its read sub-verbs (players, level, warps, portals,
  presets, objects, corpses, layout, nearby, grid, blocked, attack, passives, points, roster, `quest N show`). Reads
  (`state`, `monsters`, `dump`, `screenshot`, ...) pass through the tap and are not written, so no path ever lands in a
  file. Nothing in the file names a socket, a directory or a host; the binary is identified by its sha256 only.
- A tick is one lockstep frame = 40 virtual ms (docs/d2d-fork.md). `t` is the sum of the `step n` counts before the verb.
- **Checkpoint hash `detloop-journal-v1`**: sha256 over the replies to `dump player,stats,items,quests,ui,merc`,
  `monsters`, `ground` (per verb: each payload line + `\n`, then the ok rest, then a zero byte), taken right after the
  `step` that crossed a multiple of `every` ticks. This is byte for byte the journal checkpoint of agent DETLOOP
  (`internal/d2dctl/journal.go` on agent/detloop, docs/d2d-determinism.md, S: read in its worktree), so the first 16
  hex digits equal its `C <tick> <hash>` lines and a journal can confirm a replay. The reads are plain verbs and do not
  touch the simulation (docs/d2d-loop.md phase 0 reads `monsters` between steps with equal hashes, V). `lv`, `area`,
  `life` come from one extra `state` read and are for scrubbing only.
- A reader refuses another major version (`ErrVersion`, V) and the player refuses an unknown hash name (V).

## Commands (`cmd/d2dreplay`, runs on the d2d host)

```
d2dreplay record -upstream D2D.sock -listen TAP.sock -out RUN.replay.gz -d2d-bin BIN -seed N -class N -name X [-d2d-args "..."] [-every 250] [-idle 30s]
d2dreplay record -launch -run DIR -data MPQDIR -d2d-bin BIN -seed N -class N -name X -out FILE [-no-lock]   # starts its own hidden d2d
d2dreplay play FILE -d2d-bin BIN -data MPQDIR [-run DIR] [-until TICK] [-gap-min 1s] [-lock-wait 3000s] [-no-lock]
d2dreplay info FILE
d2dreplay render FILE -out DIR -d2d-bin BIN -data MPQDIR [-stride 500ms] [-fps 10] [-video=false]
```

- **record** is a tap (docs/d2d-view.md has the same shape): it holds the only connection to d2d and serves one client
  on `-listen`; the bot (`d2bot --d2d-dial TAP.sock --d2d-sock D2D.sock --managed ...`) or a script (`d2dprobe -sock
  TAP.sock -script f.txt`) dials the tap. Every line is forwarded unchanged and answered with d2d's reply; a second
  client gets `err busy` like d2d itself. Nothing is injected or rewritten (V `TestTapRecordsMutatingVerbsAndCheckpoints`:
  the replies are the fake's own, the fake saw no rewritten line, the file holds exactly the mutating verbs in order).
  When the d2d behind the tap goes away (managed bot restart) the client gets `err replay: upstream d2d not reachable`,
  the next command redials (up to 90 s) and a restart marker is written (V `TestTapUpstreamRestartMarker`). Stops on
  `quit`, `-idle` without a client, `-max`, or SIGINT; the trailer is written in every case. With `-launch` it starts
  d2d itself from the recipe (same flags as the bot's launcher, V `TestLaunchArgs`) under the lab lock.
- **play** starts a hidden d2d from the header (recipe, args) in a private run dir (socket, `home/{config,data}` for
  XDG, `d2d.log`), takes `/tmp/d2d-lab.lock` first (`flock -w`, polled, released when d2d is gone; `-no-lock` when the
  caller holds it), waits for `screen=ingame` plus `-settle`, then sends the verbs in order. Recorded wall gaps of at
  least `-gap-min` are slept (level loads run on real time in the fork, docs/d2d-fork.md section 3; capped at 10 s).
  At every checkpoint it computes the hash and compares: `OK: N checkpoints matched ...` (exit 0) or `DIVERGED at tick T
  (checkpoint i): want .. got ..` (exit 1); a restart marker ends playback with "OK so far" (restarts are not replayed).
  A build sha256 different from the header is printed as a WARNING before playback. `quit` lines are not replayed;
  verbs whose ok/err outcome differs from the recording are counted (`reply differences`, a hint). V offline:
  `TestPlayerMatchesSameFixtures` (3/3 checkpoints, player sends exactly the recorded verbs and never `quit`),
  `TestPlayerReportsFirstDivergence` (changed monster list: divergence at the first checkpoint, `-until` stops before it).
- **info** prints header, counts, size per virtual hour and the checkpoint table (scrub index: tick, virtual time, clvl, area, life).
- **render** = play plus a `screenshot DIR/fNNNNNN.png` every `-stride` of virtual time (lab5 draws a frame on request,
  docs/d2d-view.md), then `ffmpeg -framerate FPS` into `DIR/replay.mp4` when ffmpeg is on PATH (the worker host has n9.0.2 at
  /usr/bin/ffmpeg, V; nothing is installed, without it the message says frames only). Speed factor = stride x fps
  (500 ms x 10 fps = 5x real time).

## Limits

- Restarts are not replayed: a managed bot restarts d2d after a death or a save point; the file marks it, playback
  stops there. Replaying across a restart needs a save-based recipe (`save` header, `--start-screen charselect`, key
  Return, `lockstep on`) which is not implemented (the `Save` header field exists, `Launch.Args` refuses it).
- Wall time: the fork loads levels and menus on real time. Gaps under `-gap-min` are not reproduced; a bot that stepped
  straight after a warp while d2d was still loading may diverge on playback (set `-gap-min 0` for full gap fidelity, slow).
- Full bot loops: the bot's decisions depend on wall-clock timing in places (docs/d2d-loop.md), so a recording is exact
  for the game (same verbs replayed) but a second *bot* run is not the same input sequence; that is DETLOOP's job
  (agent/detloop). A replay of a bot run therefore verifies the game's determinism under the bot's inputs, not the bot.
- `-every` must be the same unit as DETLOOP's journal (ticks) for hashes to line up; the two tools take checkpoints
  after the step that crossed the interval, so for the same every they land at the same ticks only if the step sizes
  are the same (they are: both see the same `step n` lines).
- Size: one verb entry is about 30 to 40 bytes before gzip. A bot sends about 9 commands per step of which 1 to 3 are
  mutating; see the measured number below.

## Live results (the worker host, lab5, hidden d2d, throwaway characters, under the lab lock)

### Scripted walk + fight (V, 2026-10-30 18:57, `~/d2d-lab/replay/inner-scripted.sh`, lab5 sha256 aaf1a72c..)
`d2dreplay record -launch` (fresh Necromancer "Det", seed 3, `--lockstep --no-save`) with `d2dprobe -script fight.txt`
through the tap: `debug stat 12 8`, `debug goto 2`, a `cmd move 36 36` walk leg of 100 ticks, `debug stat 12 30`,
`debug warp 36.0 36.0`, 8 x `step 250` (the det.py FIGHT scenario of docs/d2d-fork.md section 5 plus the walk).
Result: 20 verbs, 2120 ticks (virtual 1m25s), 8 checkpoints, 1115 bytes (47 KB per virtual hour), recorded in 6 s
wall. `play` three times on fresh hidden d2ds: **OK 8/8 checkpoints in all 3 runs**, 0 reply differences, 2120 ticks
in 5.8 s each (15x real time, the reproduced 2 s and 3 s script sleeps included). The file is committed as
`docs/probes/replay/scripted-fight-lab5-seed3.replay.gz` (`d2dreplay info` on it prints the checkpoint table).
Whole job under one `flock -w 3000 /tmp/d2d-lab.lock`: 36 s.

### Full bot loop (V, 2026-10-30 19:01, `~/d2d-lab/replay/inner-bot.sh`, lab5, `d2bot` built from master c3622dc)
`d2bot --backend d2d --managed --d2d-class 4 --char Replayy --seed 5 --goals campaign --run-for 2m --d2d-dial TAP --d2d-sock D2D`
(fresh Sorceress, saves in its own dir, no cheats) with `d2dreplay record` as the tap. First attempt: the bot chickened
at 25 percent life after 18 s wall and sent `save` + `quit`; the tap of that build took `quit` as the end of the
recording and removed its socket, so the relaunched d2d could not be reached by the bot (killed by hand, exit 137;
fixed in d071331: `quit` is now a d2d restart, the tap stays up and writes a restart marker). The recorded prefix is
still a valid replay: 12602 verbs (10698 step, 1556 cmd, 333 click, 8 key, 3 move, 2 rclick, 1 save, 1 quit),
26251 ticks (virtual 17m30s), 105 checkpoints, 55369 bytes = **190 KB per virtual hour** (gate: under 1 MB).
`play` on a fresh hidden d2d: **OK, 105/105 checkpoints matched**, 0 reply differences, 26251 ticks in 4.2 s wall
(247x real time: no wall gaps to reproduce, the bot never paused 1 s). So under lockstep the game is deterministic
for the bot's inputs even before DETLOOP lands: a replay re-sends the recorded verbs, it does not re-run the bot.
Second attempt with the fixed tap (19:07, V): the whole 2 wall minutes, **4 restarts** (every one a chicken or save
point: `save` + `quit`, bot relaunch, tap redial), 28081 verbs, 58259 ticks (virtual 38m50s), 233 checkpoints,
122739 bytes (190 KB per virtual hour), bot ended at clvl 3 with 118 kills, 0 deaths. `play`: **OK so far, 36/36
checkpoints matched up to the first restart marker at tick 9030**, then stopped (restarts are not replayed, see
Limits); 1.5 s wall. Committed as `docs/probes/replay/bot-campaign-lab5-seed5.replay.gz`.

What blocks a full-bot replay across restarts: not nondeterminism (every checkpoint matched in both attempts) but the
relaunch recipe: after a restart the bot loads the saved character (`--start-screen charselect`, key Return, `lockstep
on`), so the player would need the save file at that point (its sha256 is the natural header field) and the same
menu-time sequence; a replay file stays free of save bytes, so this needs the save kept next to the file.

### Render (V, 2026-10-30 19:04, `inner-render.sh`)
`d2dreplay render -stride 2s -fps 5` on the scripted file: 43 PNG frames (800x600, about 480 KB each, 21 MB) plus
`replay.mp4` (620 KB, 5 fps = 10x real time) by the worker host's ffmpeg n9.0.2; playback stayed OK (8/8) with the screenshots
interleaved, 7.2 s wall for 85 virtual s. The frames are real game pictures (one was viewed, see the report).
