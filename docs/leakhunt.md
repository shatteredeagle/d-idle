# fd growth hunt (agent LEAKHUNT, VERIFIED LIVE on the worker host i3, 2026-10-28)

The 45 min soak reported Game.exe fds +143/h (54 to 224) and wineserver +30/h. Cause: not cycles, not a leak on our side.

## Measurements (cmd/oog `fdcycle N NAME`, `fdidle N SECS`; /proc/PID/fd, GE-Proton11-11, ntsync)
- 30 leave (Save and Exit) + re-enter cycles, 6 s each, character in town: Game.exe 57 -> 57 (0 per cycle, 36 cycles in all). A 3 min idle control: flat.
- First minute of a fresh helper only: d2input.exe 39 -> ~100 and wineserver +100 (Go runtime/threads warm-up), then flat (101-104 over 40+ cycles and 400 ping/command round trips on one or many connections; no CLOSE_WAIT, the server closes properly).
- What grows Game.exe: first visit of an AREA in the process lifetime. Town -> Blood Moor -> town: 57 -> 126 (+69, all `anon_inode:ntsync`). Five more round trips to Blood Moor: 126 -> 126. Leave/re-enter in the same process: stays 126 (no drop, no growth). The soak's jumps (54 -> 186 in the first 4 min, then +14, +24 at later area changes) line up with `area_change` events and then sit flat for 25+ min: a plateau, not a slope.
- Recycle (Save and Exit, stop, start, login): 22 s on i3, Game.exe 126 -> 57, wineserver 1823 -> 1741, d2input 102 -> 50.
- Limit: RLIMIT_NOFILE of Game.exe and wineserver is 524288. About 70 fds per new area, at most ~140 areas: <10k even if every area is visited once per process life. No exhaustion risk; harmless growth.

## Our side
No leak found: input client keeps one connection, d2input closes every connection, the reader opens /proc/PID/mem once per attach (and closes it), Xvfb grows 21 -> 33 fds in the first minutes (X clients), then flat.

## Mitigation (a safety net, off by the zero value)
`internal/launch/recycle.go`: `Recycler{Policy, FDs, Leave, Stop, Start, Login}`; `Maybe(ctx)` between goals (in town) recycles when `MaxCycles` cycles (`NoteCycle`) or `MaxFDs` game fds are reached, `MinInterval` guards loops; `Recycle(ctx, reason)` forces one. Defaults 500 cycles / 4000 fds / 30 min, env `D2BOT_RECYCLE_CYCLES`, `D2BOT_RECYCLE_FDS`, `D2BOT_RECYCLE_MIN_INTERVAL` (0 disables). Live check: `oog 'recycle NAME'`. Not wired into cmd/d2bot or the fleet (not our directories): call `Recycler.Maybe` where a goal ends.

## For d2soak (not ours)
The `--max-fds-per-hour` slope must be judged after a warm-up (first ~10 min) or on the late half of the run; fds step up at area loads and plateau.

# d2input fd slope in the 2 h soak (agent HELPERLEAK, 2026-10-29)

Soak i4-20261029-054013: helper fds 74 -> 134 over 1h42m (15 fds/h, limit 5).

- V (samples.jsonl of the run + `tools/helper.sh log i4`): growth is slow and stepwise (+1..+5 per 30 s sample, long flat stretches, not tied to area_change events), and the helper's threads grow with it: 9 -> 17 threads for +60 fds, about 7.5 fds per thread, consistently across the run. The helper's own heartbeat shows heap flat (1.7-3.7 MB), goroutines flat (6), conns 2, errs 0, panics 0 after 15k commands. So no Win32 handle, socket or goroutine leak: the fds are Wine's per-thread objects.
- V (code audit): no OpenProcess/CreateEvent/timer anywhere; window handles are plain HWNDs (no close needed); the one NewCallback is created once; the client keeps one connection and closes it on error. Nothing per command or per reconnect is acquired and left open.
- U (root cause, not reproducible offline): the Go runtime starts an extra OS thread whenever a goroutine sits in a Win32 call (each one a wineserver round trip under Wine) long enough for its P to be handed off while other goroutines are runnable (3 to 4 goroutines doing list/post concurrently: bot, sentinel, short-lived clients, heartbeat), and it never ends idle threads. Concurrency spikes keep adding threads.
- Fix: `cmd/d2input/thread.go`. All Win32 calls (list, valid, post, scan, info, hung) run on one dedicated goroutine locked to one OS thread, so the helper's Win32 thread count is fixed at one and calls are serialized (the enumMu lock is gone). Heartbeat now logs `threads=` (Go runtime threadcreate count) next to `handles=` (always 0 under Wine, GetProcessHandleCount is unimplemented there).
- Unit tests: `go test ./cmd/d2input` (200 concurrent jobs run on exactly one thread; a panic is re-raised to the caller and the thread survives). The Windows layer itself is only vet-checked (`GOOS=windows GOARCH=386 go vet ./cmd/d2input`).
- How to verify live (needs a rebuilt helper, `tools/helper.sh restart` / `tools/game.sh helper iN`): run `d2helpersoak -d 30m` (already exercises move, click, key, seq, list, info, hung, ping, short-lived and dropped clients, screen changes) and sample the helper fds from /proc plus `tools/helper.sh log iN | grep hb`: expect `threads=` to settle (about 10) after the first minutes and fds to be flat after warm-up. If threads still climb, the cause is outside the Win32 calls (net poller or timers): then try GOMAXPROCS=1 in the helper.

# d2input fd/thread slope, second pass (agent HELPERLEAK2, 2026-10-29)

Soak i4-20261029-074315: the pinned-thread build still went threads 15 -> 19 in 33 min (hb threads=), fds +22.9/h.

- V (ssh worker, /proc/<helper pid>, read-only, helper of i4 on port 47016): 18 OS threads, all asleep in ntsync, fds = 39 anon_inode:ntsync + 72 pipes + 5 sockets + a few files. Pipes and ntsync objects scale with threads (about 4 pipes + 2 ntsync per thread, Wine's per-thread objects); sockets are flat, so no connection or Win32 handle leak. hb: goroutines=7, conns=2, heap flat, threads= (Go threadcreate) 19 vs 18 live OS threads: the threads are Go's.
- V (thread ids): new threads appear one at a time every 10-15 min, not tied to load (commands/min flat): rare scheduler events (a P handed off while another goroutine was in a syscall), each adding a permanent thread.
- Cause (S, Linux stand-in: osThread + 6 clients + blocking syscalls, threadcreate after the same load): GOMAXPROCS 8 -> 12, 4 -> 9, 1 -> 7. The helper sees all host cores (16 on the worker host), so the pool of Ms the runtime may spin up and keep is large. Pinning the Win32 calls to one thread did not bound it because other goroutines (conn handlers, heartbeat, timers) still hand off Ps.
- Fix: `limitProcs()` in cmd/d2input/main.go sets GOMAXPROCS(1) (override `D2INPUT_PROCS=N`); the helper start log line now shows `gomaxprocs=`.
- Deploy and check (not done here): `tools/helper.sh restart i4` (kills and restarts i4's helper and runs deploy first; only when i4 is idle, i.e. after the soak) or `tools/helper.sh deploy` and restart per instance later. Expect `hb threads=` to settle at about 7-9 and stay flat, fds flat after warm-up (limit 5/h). If threads still creep, next suspect is Wine-side (IOCP completion threads of the net poller); then try fewer connections or `D2INPUT_PROCS=1` vs 2 comparison.

# wineserver fd slope, per-instance (agent WINELEAK, 2026-10-29, read-only on the worker host i4)

Question: the soak shows the i4 wineserver at +4..+18 fds/h (limit 5/h), about 1750-1870 fds. Tags: V = measured or fetched by me, S = secondary, U = unverified.

## What the fds are
- V (`/proc/<wineserver>/fd`, GE-Proton11-11 = Wine 11.x, ntsync, kernel 7.2.9; `wine --version` on the host says wine-11.20 Staging but the prefix runs the Proton one): 1772 fds = 1254 `anon_inode:ntsync` + 262 pipes + 203 regular files + 29 sockets + 24 anon/misc (+1 eventpoll). 71% are ntsync objects.
- V (Wine source, `server/inproc_sync.c`, tag wine-11.0, https://raw.githubusercontent.com/wine-mirror/wine/wine-11.0/server/inproc_sync.c): with `/dev/ntsync` present, the SERVER creates one ntsync object, and keeps one fd for it, for every event, mutex and semaphore (`server/event.c`, `mutex.c`, `semaphore.c` call `create_inproc_*_sync`) plus one `alert_sync` per thread (`thread.c:567`); `inproc_sync_destroy` closes it, so there is no leak in the object lifecycle itself: the server count is the number of live sync objects plus live threads.
- V (counts across the prefix): 126 live Wine threads in the i4 prefix (explorer 39, winedevice x2 26, xalia 13, services 11, d2input 9, plugplay 7, rpcss 6, Game 4, others) against 262 pipes: about 2 pipes per thread (reply and wait pipes). The server fd total is therefore dominated by fixed system processes (services, explorer, winedevice, xalia, rpcss, tabtip: all of them idle) and scales with thread count. Game.exe itself holds only 162 of the 1254 ntsync objects (client side caches them lazily, the server holds all), d2input 15.

## Growth, with event correlation
- V (soak i4-20261029-091750 samples every 30 s, 94 samples, 09:17 to 10:04; helper log `hb` lines): wineserver 1748 -> 1772 in 47 min, in exactly two steps, flat otherwise: +11 at 09:26 (helper fds +5, helper threads 8 -> 9, the helper's `hb threads=` 9 -> 10 at up=10m) and +13 at 09:32 (no helper change; Game.exe had stepped +8 fds at the area_change at 09:29, wineserver follows it a few minutes later, so this one is the game's first visit of an area, as in the LEAKHUNT section above). During the same window: 8 area_change events, 133 kills, 4 potions, 1 level-up in the first step; none of potions, kills, drops or revisited areas moves the counter. Nothing correlates with the helper's TCP reconnects (conns constant at 2, no step between).
- V (my own sampler on the worker host, 150 s apart from 10:03, `/proc/<pid>/fd` of the wineserver and every Wine process in the prefix, thread and ntsync counts): flat for 10 min (5 samples, 10:03-10:14, the whole fd breakdown identical; no step happened in that window, so it shows the plateau but cannot reproduce a step; i2 was not running, so no comparison instance). Wine system processes (explorer 39 threads, xalia 13, services 11, winedevice 9 and 17) did not change at all. So no continuous leak: a step function.
- So the slope the soak prints is two kinds of discrete events: (a) a new Go OS thread in d2input costs +11 server fds (V: 2 pipes + socket + alert ntsync + file handles, measured by the step), (b) first visit of an area costs Game.exe +8..+69 ntsync objects. HELPERLEAK2's GOMAXPROCS(1) helper (`gomaxprocs=1` in the start line of the running helper, V) added exactly one thread in 47 min, at 10 min of uptime, i.e. it now converges instead of creeping (previous runs: 1 thread per 10-15 min).
- Why the soak still flags it: warm-up is 10 min and only the first area_change steps are ignored (fd plateau fit in `cmd/d2soak/analyze.go`); the wineserver step at 09:32 sat in a sample with an area_change and a helper step at 09:26 landed right after the warm-up, so on short runs (20-45 min) one step gives +10..+25 fds/h on the fitted slope. On long runs the slope decays to zero.

## Known bugs and mitigations
- S/U: no upstream report of an ntsync fd leak in wineserver found (searches of Wine bugzilla (Anubis-blocked from curl, headless fetch not allowed here), GitHub, forums). The closest are (S) Proton fork notes (https://github.com/jegglest/proton-je/releases/tag/proton-je-11.0-20261014-4) describing a `keep_inproc_sync` request change in the close/dup_handle path (pending upstream, bugs 60397/60417, not checked), which is about handle lifetime, not a confirmed leak; (S) a Fedora guide that counts 576 ntsync fds for an idle Notepad (https://computingforgeeks.com/how-to-install-wine-on-fedora/): normal usage. No fix to wait for.
- V: limit `RLIMIT_NOFILE` 524288 for the wineserver. Worst observed slope +18/h x 24 h = +430 fds (to ~2200, 0.4% of the limit); even +70 per new area x 140 areas is under 10k.

## Side finding (not the fd slope): orphaned winedevice.exe
- V: five `winedevice.exe` of the i4 prefix with ppid 1 and ages 1h to 7h49 (35 MB RSS, 15 threads, 112 fds each, stuck in `ntsync_schedule`, 3-5 min CPU each), one per earlier i4 wineserver restart; the live server has its own two. They belong to dead wineservers (only one wineserver exists) and `tools/game.sh stop`'s `wineserver -k` does not reap them. Cost at the current restart rate: roughly 35 MB and 15 threads per restart; 24 h with a recycle every 30 min would leave tens of them (~1.5 GB). U (not tested): killing them as part of stop (e.g. kill processes whose environ has the prefix's `WINEPREFIX` and ppid 1 after `wineserver -k`). Not touched here (read-only brief).

## Recommendation: ignore the wineserver slope, change how it is judged
1. Do not mitigate in the helper. HELPERLEAK2's GOMAXPROCS(1) already removed the only helper-driven growth (one step, +11); the rest is first-visit-of-area and Wine's thread objects, not ours.
2. Soak limits: for `proc:wineserver:fds` (and other Wine system-process fds) judge the plateau, not the raw slope: either treat the process as exempt from `MaxFDsPerHour` and use an absolute headroom limit (fail when above ~5000 fds or +1000 over the first hour), or require >= 2 h before fitting. Evidence: stepwise, flat between steps, 3 steps/h at most, worst +18/h, 0.4% of the fd limit per day.
3. The Recycler needs no change. It keys on Game.exe fds (4000), cycles (500) and 30 min spacing (`internal/launch/recycle.go`); at +18 wineserver fds/h the wineserver would need >120 h to reach even 4000, and every Recycler restart resets the count to ~1750 (LEAKHUNT measured 1823 -> 1741). A restart is not needed for this at all; 24 h unattended needs none.
4. Do fix (separate task): reap orphaned winedevice processes at game stop, they are the real resource growth of the i4 restart loop (memory and threads), not the fds.
