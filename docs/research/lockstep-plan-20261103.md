# Lockstep plan, 2026-11-03: the cheapest path to "is d2d within 15 percent of game.exe on XP/h"

> status: plan, docs only (card `lockstepplan`). No game start, no live run, no host load, nothing built. Parents: docs/research/tierb-design-20261102.md, botab-20261102.md, seed-study.md, seeddist-trace.md, seedspawn-trace.md, lockstep-p3.md, lockstep-p4.md, docs/timescale.md, docs/process/baseline-runs.md section 10, docs/snaplib.md, docs/cheats.md. Private hook details are in the private hook repository and are not repeated; this doc uses only what docs/timescale.md and the lockstep-p* docs publish.
> Tags: **V** = read in a committed doc or private ops doc this session (named), or measured by a run cited there; **S** = secondary (another doc's number); **I** = my inference; **U** = unverified. "The worker host" = the 16-core lab machine with the hidden instances; "the Windows PC" = the owner's household PC that holds the game files and a build lane.

## 0. Recommendation first

1. **Run the fresh clvl 1-6 cell, game.exe at k=1, on three hidden instances of the worker host, right after Gate 2 ends.** Round 1 is the pilot (3 game.exe + 3 d2d lanes, about 2 h real); round 2 tops it up to 6 per arm, which settles the cell if the d2d CV is at most 8 percent (botab section 5). Total about 4 to 5 h real on the worker host, about 25 to 30 agent-hours across the staged cards (most of it AB-TOOL and the run records). Alongside Gate 2 do only offline work and the one-lane AB-SEEDPIN check (section 2.3).
2. **Native game.exe on the Windows PC: NO for now.** It would add lanes without loading the worker host, but it needs a Windows memory backend, a desktop session and an exemption from that PC's own safety gate: 25 to 40 agent-hours (U) to save hours of waiting on a host that is free after Gate 2. Revisit only if the pilot shows that the cell needs 15 or more game.exe lanes per arm. If extra lanes are wanted, the cheaper pilot is Wine inside the PC's existing WSL distro (section 3.6), which also needs the owner's go.
3. **AB-SEEDPIN needs no memory write and no byte edit:** the pin is the copy of a saved honest character (`d2restore place`), a logged SETUP copy; the d2d arm needs distinct seeds per lane because d2d repeats itself exactly on one seed (section 4).
4. **The decisions in section 5 are few:** margin 15 versus 20 percent, one lane next to Gate 2, round 1 start right after Gate 2, Gate 3 after the pilot, and whether the owner wants the Windows PC pursued at all.

## 1. Where lockstep stands (ten lines)

1. **Works (V run, lockstep-p3/p4):** the private clock hook at k=2 and k=4 holds the frame rate to the factor (about 50 and 100 frames per real second against 25), the bot clock is scaled by `internal/gclock` (`D2BOT_TIMESCALE`, docs/timescale.md), and kills per game minute are then close to the k=1 value (P4: 7.7 at k=2, 6.8 at k=4 against 10.0 at k=1, one run per row, noise band unknown).
2. **Pair verdict (V run, tierb section 6):** kit sets kitb (k=1) and kitk2 (k=2), 3 lanes each, same pin, XP/h -2.9 percent (interval about -12.6 to +6.7), kills/min +0.6 percent, deaths and restarts alike; k=2 is a tagged second regime for screening and pilots, not yet for a verdict.
3. **Known bad:** game minutes to clvl 6 are 12 percent longer at k=2 in all three lanes (cause U: real-time farm and goal timers, unscaled helper holds); all six kit lanes carry the weak flag because of 6 to 22 restarts, so no kit lane is a clean verdict lane; k=4 collapses on the unscaled bot (P3) and has 29 `nav stalled` lines scaled (P4).
4. **Seeds (V read, seed-study, seeddist/seedspawn traces):** layout, spawn lists and dice distributions of d2d and game.exe are statistically equal; the RNG stream after level generation is not modelled (per-unit seeds), so fights and drops cannot match seed for seed. Seed for seed A:B is therefore impossible; statistical A:B is the route.
5. **A:B so far (V D, botab):** no XP/h metric is EQUIVALENT at 15 percent and none is DIFFERENT; the records cannot answer (one clean game.exe lane at the current bot code, all d2d runs on one seed).
6. **Tier B (frozen clock, step verb):** designed, not built, held; its own hook go and 12 to 20 agent-hours; it buys diagnosis, not a cheaper verdict.
7. **Pin route:** one saved `.d2s` fixes the map for game.exe (map id at header offset 0xAB plus the active-difficulty bit; V live stable across 15 loads, docs/saves.md) and d2d reads the same file; whether it holds across prefixes is the open check of AB-SEEDPIN (the conflict with docs/orig-spawns.md).
8. **Staged cards:** AB-SEEDPIN, AB-TOOL, AB-PILOT, AB-FULL, TIERB-HOOK, TIERB-DET (tierb section 4); none started.
9. **Host:** Gate 1 ends about 8:29 AM EST on 2026-11-03, Gate 2 (8 h wall) follows on a pass; the A:B pilot runs after the gates (decided).
10. **The sample-size lever is the d2d CV at the pin** (4 to 23 percent depending on what is held equal, botab section 5): that is why round 1 is a pilot and why the fresh cell comes first.

## 2. Cheapest path to the 15 percent answer

### 2.1 Choice of cell, kit, k

| choice | pick | why |
|---|---|---|
| cell | fresh clvl 1 to 6 (rows cut at the first clvl 7 line) | the game.exe CV there is 4 to 5 percent (kitb/kitk2, n=3, V run); a lane is about 60 to 110 game minutes, so a round of three is about 2 h; the late-band cell needs 17 lanes per arm at 2 h each (botab section 5, 12 h real on three lanes) |
| character kit | fresh clvl-1 Sorceress characters from the kit snapshot of set `kit` (`baseline-run.sh ... --fresh-from-set kit`, Freshc, Freshd, Freshe), restored byte for byte before every run | docs/process/baseline-runs.md section 10; gold, stash and level start equal on both arms |
| bot pin | 15506e05 (the pin of kitb and kitk2; contains `internal/gclock`) on BOTH arms; no bot commit in between | botab: only same-code arms are apples to apples |
| k | **k=1 on the game.exe arm for the verdict**; k=2 only for dry runs of the harness (halves a lane to about 1 h) | tierb section 6: k=2 not admitted to a verdict, the clvl 6 delay is unexplained |
| d2d arm | the lab fork in lockstep with the bot, `d2dsoak run` per seed; minutes per lane | fidelity-gate "Fast d2d runs"; the regime difference (lockstep 6 to 12 virtual min per wall min against k=1) is gap G4 of botab and stays a caveat; 3 d2d lanes at k=1 would measure it (botab section 5 item 3), add them to round 2 only if the verdict is borderline |
| instances | hidden i2, i3, i5 (the tierb card table); i1 and i4 belong to the nav and bot agents; never i0 | tools/instances.conf, docs/multi.md (private ops docs) |
| markers | `d2cheat --reason R runstart <run>` per lane, `D2BOT_RUN=<run>`, `writes after start: 0` in every record | AGENTS.md, ADR 0014 |

### 2.2 How many lanes and how many wall hours

Sample sizes are botab section 5 (equivalence at 15 percent, alpha 0.05 per side, power 80 percent, true gap 0 / 5 percent):

| d2d CV at the pin (game.exe 4 percent) | runs per arm | game.exe rounds of 3 | wall h at k=1 |
|---|---|---|---|
| at most 5 percent | 4 / 5 (floor 5 per arm from the spec) | 2 (the second round of 2 lanes) | about 4 |
| 8 percent | 6 / 7 | 2 to 3 | 4 to 6 |
| 12 percent | 9 / 12 | 3 to 4 | 6 to 8 |
| 23 percent (the spread across builds today) | 23 / 36 | 8 to 12 | 16 to 24; then ask the question on another cell or take Tier B |

Round 1 (the pilot) gives the d2d CV and the cross-backend pair correlation; the plan after it is the row of this table. The d2d arm costs minutes per lane, so it is never the bottleneck. The existing kitb lanes (3 at k=1) are weak by the restart flag and are not reused for the verdict; they are context (botab uses them as S numbers).

Hour budget of the recommended path: round 1 about 2 h (3 lanes, game time to clvl 7 plus 10 min setup), round 2 about 2 h, plus 15 to 30 min of record writing each = **4 to 5 h real** and no more than 6 game.exe lanes in total.

### 2.3 Interleaving with Gate 2 on the same host

Numbers: nproc 16; the cost table of the capacity rule (private ops doc, worker-capacity): game.exe instance 1.0 core, d2d lockstep with a bot 1.0 + d2bot 0.2 (U estimates in that table); a soak harness lane adds the read-only `d2soak` process; `capacity2.sh --add N` is the bounded check (about 7 s; verdicts OK, BUSY (cpu), BUSY (io), NO; idle+nice at least 25 percent after the added cost, iowait at most 40 percent). Measured fact that sets the limit (V, private parity doc, 2026-11-02): **four niced build cores took Gate 1 from about 11.6x to 6.5-7x virtual speed, under the 8x stop line, even at nice 19**. The cause is not known (U; shared caches or SMT siblings are my guess, I), so core count alone is not the safe measure.

| situation | game.exe lanes | rule |
|---|---|---|
| Gate 2 running | **0 or 1** (AB-SEEDPIN on one instance, short); a second lane only if the gate speed over the last 10 minutes stayed at or above 10x with the first | `gate-run.sh --status` every 10 minutes; any 10-minute window below 9x: stop the lane with `game.sh stop`; below 8x is the stop line of the parity doc. A lane costs 1.2 cores against 4 for the build that broke the speed, but a real-time game starved of CPU also slows (frame rate), so a lane that falls under about 90 percent of the nominal frame rate is weak and discarded |
| Gate 2 finished, no gate running | 3 lanes (hidden i2, i3, i5) plus the d2d arm | `capacity2.sh --add 3.6` must say OK; announce as a big run (3 instances, over 30 minutes) to the user with the 5-minute hold window |
| Gate 3 or another long soak started right after Gate 2 | 1 to 2 lanes, rounds take twice as long | decision 4 in section 5: do not start Gate 3 before the pilot, or accept the slower schedule |
| builds (parity, d2d C++) | none at the same time as lanes | the Windows PC has a WSL lane for builds (the private build doc) so builds need not run on the worker host |

Do not run the d2d arm during Gate 2 either: a second lockstep d2d on the host adds the 1.0 core that the capacity table already counts, and the pilot wants d2d and game.exe lanes of one round in the same window (interleaving rule, tierb 3.1).

## 3. A native game.exe lane on the Windows PC

### 3.1 What is there (V, private build doc 2026-11-02)

- Game files copied with the owner's go: a 1.38 GB folder under an admin-only directory (Game.exe 1.14d, same sha256 as the lab's copy, MPQs, `Patch_D2.mpq`, the patch installer); not copied: the CD key, registry, Battle.net files, the intro video MPQs, saves, any Wine prefix. "game.exe is never started there." Owner and administrators only; the limited account has no entry.
- A safety gate script runs before every job and every 20 s as a watchdog; it refuses on a weekday inside the quiet window (changed 2026-11-03 to 4 PM to 8 PM EST), on any game process (name list plus anything started from game library folders), and when an interactive non-system session uses more than 25 percent CPU or GPU. A kill switch stops everything under the lab folder. Jobs run at idle priority on 8 of 16 threads.
- A WSL2 distro (Ubuntu) holds the game data, the emulator environment and the Linux build lane; it needs no desktop. It has been used for builds and one emulator oracle run (V).
- The hook is a Windows DLL proxy (it replaces the game's `ijl11.dll` by a forwarding proxy, original renamed, docs/process/baseline-runs.md section 10), so on Windows it needs no Wine layer at all; V that it works under Wine (P3, P4), U natively.

### 3.2 What a native lane needs

| need | native Windows | status |
|---|---|---|
| character files | the lab's `d2restore place` writes into a Wine prefix path on the worker host; natively the files go to the game's own save folder (the `Saved Games` folder of the account that runs the game, or the path in the game's registry); a small PowerShell copy plus sha256 check would do | U, ~2 h |
| `.d2s` pin | the same file set as on the worker host (copy-of-honest, sha256 compared on both sides) | design in section 4 |
| input helper | `d2input.exe` is already a Windows program (it runs inside the Wine prefix today and listens on a TCP port); natively it should run as is | U (the window discovery by hwnd should be simpler natively), ~2 h to confirm |
| memory reads | `internal/mem` reads `/proc/<pid>/mem` of the Wine process on Linux (V docs/multi.md); there is no Windows backend. A `ReadProcessMemory` backend with process discovery and the same `Fake` seam is new code (stdlib `syscall` lazy DLL calls, no new module) | not started; **the largest item, 8 to 14 agent-hours (U)**, plus building and testing the bot for Windows (does `GOOS=windows go build ./cmd/d2bot` pass? U, a 5-minute offline check) |
| OOG login | `cmd/oog` drives the character screen through the helper and memory; same dependency on the memory backend; Battle.net never used | U, rides on the backend, ~3 h |
| orchestration | `baseline-run.sh`, `soak24.sh`, `tools/live.sh`, `game.sh` are Linux shell scripts on the worker host; a Windows lane needs a driver that starts the game, the helper and the bot over ssh, collects the minute log and runs the quality script | ~8 to 12 h (U); or run the bot in WSL against a Windows game through a TCP bridge, which does not work for memory reads |
| hook install | copy the proxy as a new file, rename the original, config next to it; then restore and sha256 compare (as the Linux lane does) | ~2 h |
| display | **a desktop session is needed.** Game.exe draws with DirectDraw/Direct3D into a window; over ssh the process lives in the non-interactive service session and gets no window. Options: (a) the lane runs in the owner's interactive logon (needs that account logged in, screen locked is fine for a windowed game only if the session is not disconnected); (b) a scheduled task "run only when user is logged on"; (c) an RDP loopback session (the PC runs a remote-control tool already, so a second session may or may not be allowed; edition limits for concurrent sessions: U). In every case the owner's account is signed in and visible on the console while a lane runs | the main practical obstacle |

### 3.3 How it interacts with the account, the quiet window and the gate

- **The gate treats a lane as a person using the PC.** A game process run from the lab folder is not on the game name list (U: I did not read the list), but an interactive session using more than 25 percent CPU trips the watchdog, which calls the kill switch. So a native lane would need its own gate profile: refuse on the child's session being active or on a listed game process, ignore a session whose only load is the lab folder's Game.exe. That is a change to the safety logic of someone else's PC; the owner decides it, not an agent.
- **Quiet window:** 4 PM to 8 PM EST on weekdays (owner, 2026-11-03: "can be used all day today"); a lane must be stopped at 3:55 PM and not started before 8 PM on weekdays; weekends only when no game runs. A 2 h round has to fit: start by 2 PM at the latest.
- **Shared console:** the limited account signs in by fast user switching; a signed-in lab session keeps the lane running in the background (CPU and GPU contention for a game) and the 25 percent rule would trip when the child plays. Windowed 800x600 DirectDraw mode switching must not change the shared display mode (U: the Wine lane runs on a virtual display so it never touched this).
- **Anti-cheat on the same machine:** the family's online game uses an anti-cheat service with a kernel driver. Our proxy DLL and `ReadProcessMemory` target only our own Game.exe, so no overlap is expected, but kernel anti-cheat drivers are known to dislike unsigned injectors; this is a risk to the child's account, not only to the PC (U, I would not test it).

### 3.4 Risks to the PC and the household, EULA reminders

- Load and heat on a gaming PC the household depends on; mitigated by the gate, idle priority and affinity, but not measured with a game running (U in the build doc).
- A visible game window and audio on a family PC: launch muted (`-w -ns`) as everywhere; the lane would still flash a window on the console.
- Game files: the copy on a second machine, no CD key, offline single-player only; do not start Battle.net or any online mode on the PC, ever. EULA and the single-copy license are reminders for the owner, not a legal opinion; the hook stays private (docs/research/lockstep-licences.md).
- The memory backend and the Windows build would be new code in a standard-library-only repo; no dependency is needed (`syscall` lazy calls), but it widens the Linux-only assumptions of `internal/mem` that every package relies on.

### 3.5 Effort and benefit

| item | agent-hours (U) |
|---|---|
| offline check that the bot builds for Windows, `internal/mem` Windows backend with tests on a fake, process discovery | 10 to 16 |
| helper/OOG/hook install on Windows, character placement, driver script, record and quality script | 12 to 18 |
| gate profile and owner review, pilot lane (1 lane, 1 h) and fixes | 4 to 8 |
| total | **25 to 40** and a second pilot to prove the native lane matches the Wine lane (+3 lanes) |

The benefit is lanes beyond the worker host's three hidden ones, which only matters for the late-band cell or a CV of 12 percent and up. Against that, three lanes on a free worker host deliver 15 lanes in 15 h of waiting (tierb 3.4). Waiting is cheaper than 25 to 40 agent-hours plus the owner's gate work.

### 3.6 Recommendation: **NO for a native lane now; pilot-first for Wine in WSL if the owner wants lanes on the Windows PC**

- Native Windows lane: **no** until the pilot says n of 15 or more per arm is needed and the owner wants the Windows route.
- If extra lanes are wanted: **Wine in the existing WSL distro with a virtual X display**, the same stack as the hidden instances. It needs no desktop session, the gate keeps working unchanged (the lane is a nice-19 process in a VM, the gate counts the VM's CPU only through the 25 percent rule on interactive sessions), game data and the Linux bot tooling are already inside the distro, and the hook is the same Wine-tested DLL. Cost 8 to 14 agent-hours (U): install Wine (version differs from the worker host's Proton build, a comparability caveat, U), a prefix per lane, the helper under Wine, a driver wrapper around the existing scripts, one pilot lane. Needs the owner's go for Wine and for starting Game.exe on the PC; not approved today.

## 4. AB-SEEDPIN as an executable checklist

Goal: one honest saved character pins the map on both backends; settle the conflict with docs/orig-spawns.md; no memory write, no hook, no byte edit. Agent: a live agent on hidden i5 (exclusive `instance:i5`); the coordinator's tooling starts and stops the game (agents never start, stop or log in a game). Run next to Gate 2 only as one lane with the speed watch of section 2.3, or after it. Commands marked (confirm) use existing tools whose exact flags the card must read from `--help` first.

| # | step | command (existing tools) | pass | logged? |
|---|---|---|---|---|
| 1 | preflight | `tools/ops.sh run capacity2.sh --add 1.2` verdict OK; `tools/jobs.sh list` shows no job on i5; `ls /tmp/d2lod-live.notice` absent | OK | no |
| 2 | read the pin source (read only) | `d2cheat qdump <Freshc.d2s from the kit snapshot>` (header, map id, difficulty bytes) and `d2restore list --json` (manifest map id) | map id nonzero and active-difficulty bit 0x80 set; both tools agree | no |
| 3 | place the character into i5 (SETUP, a copy of an honest character) | `d2restore place --snapshot <kit id of Freshc> --as Seedpa --inst i5 --reason "AB-SEEDPIN setup: copy-of-honest Freshc as Seedpa on i5 for the map id pin test"` | prints `setup: copy-of-honest <id>`; ledger entry kind `setup-copy`; placed `.d2s` sha256 equals the source | **yes**, ledger `setup-copy`, reason string above (letters only, name 2 to 15 chars) |
| 4 | run marker (not a measured run, but the habit keeps the ledger honest) | `d2cheat --reason "AB-SEEDPIN read-only probe" runstart ab-seedpin-1` and `D2BOT_RUN=ab-seedpin-1` | marker written | yes (bookkeeping kind, not a write) |
| 5 | read the file again and start the game 1 | coordinator: `tools/game.sh start i5 && tools/game.sh helper i5`, then `D2_INSTANCE=i5 tools/live.sh <agent> oog "start Seedpa"`; wait in town; read `MapId` (client, RVA 0x3A0638, D `re/automap.md`) and `Act.dwMapSeed` read-only with the existing mem reader (confirm: `cmd/d2qmem` or `d2snap`) | in-game seed equals the file's 0xAB | no (reads) |
| 6 | save and exit, read the file | `D2_INSTANCE=i5 tools/live.sh <agent> oog "save"` (confirm the verb), `d2cheat qdump <placed file>` | 0xAB unchanged | no |
| 7 | start 2 and 3 on the same file (two cold starts) | coordinator `game.sh stop i5` then steps 5 to 6 twice | seed equal all three starts | no |
| 8 | second prefix | `d2restore place --snapshot <id> --as Seedpb --inst i3 --reason "AB-SEEDPIN setup: copy-of-honest of the pin in a second prefix"` (i3 only when no one else uses it), then step 5 there | seed equal to step 5; or a documented difference and the conflict explained | yes (second `setup-copy`) |
| 9 | layout check on the worker host (offline) | `d2mapdump dump` of Rogue Encampment and Blood Moor from the live run; offline `drlg-dump` with the same map id (docs/boss.md shows the `d2mapdump` use; the d2decomped `drlg-dump` tool; confirm flags) | room list and room seeds equal in the rooms the game activated | no |
| 10 | d2d side | put the same `.d2s` in the d2d slot's save directory (`$D2D_SLOT_DIR/...`, the same file set, V character-factory 4.7 for the XDG path), start the lab fork with the bot or `d2dsoak run -seeds <n>` for a short span and compare its map id and `dump` room list with step 9; also run once with `--seed <id from the file>` | the d2d room list equals the game.exe list and the file route equals the `--seed` route | the placement is a file copy on the worker host: `d2restore place --inst` does not cover d2d (snaplib "not tested"): note the copy and its sha256 in the card record, reason "AB-SEEDPIN setup" |
| 11 | read-only verification on both sides | sha256 of the `.d2s` and `.map` at the game.exe prefix and the d2d slot, before and after the run | identical to the snapshot on both sides | no |
| 12 | cleanup | `tools/game.sh stop i5` (coordinator), `tools/jobs.sh list` empty, `agentreap` lists empty | clean | no |

Decisions that follow from the result: pass (steps 5 to 11) means the pin is usable and AB-PILOT may start; a fail with the map id changing between starts means the pin falls back to `d2cheat saveedit` of 0xAB (needs a new reason string, for example "AB-SEEDPIN setup: pin map id 0xAB on Seedpa for cross-backend layout parity", and a card to add the verb to the allowlist, U whether the verb exists) or, as the last resort, an unpaired A:B with no layout pairing, which the unpaired sample sizes already cover.

Seed count per round (I, from docs/d2d-determinism.md: the d2d side repeats exactly on equal seed and equal bot): **every d2d lane needs a distinct map id or `--seed`; the game.exe arm may reuse a character, because its per-unit seeds and timing differ run to run.** Round 1 uses the three kit characters' ids on both arms (pairs), round 2 adds three d2d seeds by `--seed N` (free, no file) and re-runs the same three characters on game.exe, or asks the coordinator to create more fresh characters.

## 5. Decisions for the user, with my recommendation

| # | decision | recommendation |
|---|---|---|
| 1 | Margin for the claim: 15 percent (this question) or the spec's 20 percent | 15 percent for the report, state both; 20 percent costs about 1.8 times fewer runs, and the gate spec keeps 20 |
| 2 | One game.exe lane (AB-SEEDPIN) next to Gate 2, stopped on any 10-minute gate speed below 9x | yes, a single lane; no second lane unless the first leaves the gate at 10x or more |
| 3 | The `.d2s` copy pin as a declared SETUP action (ledger `setup-copy`), the d2d copy noted in the record | yes |
| 4 | Gate 3 (24 h) before or after the pilot | after the pilot: a 24 h gate would halve the lanes for a day (section 2.3) and nothing in the pilot waits on Gate 3 |
| 5 | Big-run notice for round 1 and 2 (3 instances, 2 h each, 3.6 cores) with the 5-minute hold | yes; send it when Gate 2 is judged, and start round 1 the same evening if the result is back before 6 PM EST |
| 6 | Whether k=2 lanes may fill rounds | no for the verdict; yes for a harness dry run (halves a lane), tagged `REGIME k=2` |
| 7 | Tier B | still held (tierb 0.3); revisit only on the table of tierb 3.5 |
| 8 | A native lane on the Windows PC | no; WSL Wine only if after the pilot more lanes are needed, and then it needs the owner's go (Wine on his PC, Game.exe started there, a gate exemption) |
| 9 | More fresh characters (Freshf and up) from the coordinator's OOG tooling | only if the pilot asks for more than 3 pairs; the d2d arm uses `--seed` otherwise |
| 10 | Late-band cell (clvl 12-13, 17 lanes per arm, 12 h real on three lanes) | only if the fresh cell is DIFFERENT and the d2d fix cards do not close it, or if the owner wants the cell where soaks spend their time |

## 6. Schedule, three days from the start of Gate 2

T0 = start of Gate 2 (about 8:30 to 9 AM EST on 2026-11-03 after Gate 1 is judged; Gate 2 is 8 h wall, so about 4:30 to 5 PM). Times are EST.

| day | window | work | host load |
|---|---|---|---|
| 1 (Nov 3) | T0 to T0+8 h (Gate 2) | AB-TOOL starts (offline, `tools/ab/xbackend.py`, tests on the collected kit sets and soaks); AB-SEEDPIN prepared (commands in section 4 read with `--help`, the reason strings checked); at about T0+1 h, only if the gate speed is at least 10x: steps 1 to 7 of section 4 on i5 (about 1 to 2 h, one lane); speed watched every 10 minutes | 1 lane at most; no d2d arm, no builds |
| 1 | Gate 2 judged (about 5 PM) | read the record, apply the pass/fail to the pilot start; big-run notice for round 1 | none |
| 1 | evening, by 6 PM (if the host is free) | steps 8 to 12 of AB-SEEDPIN if not done; AB-PILOT round 1: kit restore, run markers, 3 game.exe lanes at k=1 on i2, i3, i5, 3 d2d lanes (the same 3 pinned characters), to the clvl 7 line (about 2 h); overnight nothing new | 3 lanes + d2d, `capacity2.sh --add 3.6` OK first |
| 2 (Nov 4) | morning | collect round 1 (`baseline-run.sh collect`), run AB-TOOL on it: d2d CV, game.exe CV, pair correlation, n per arm from the table in section 2.2; write `docs/soaks/ab-pilot-20261104.md`; decision on rounds | none |
| 2 | midday to afternoon | AB-PILOT round 2 (three more game.exe lanes, three more d2d seeds), about 2 h; if the CV is below 8 percent this reaches the floor of 5 to 6 per arm | 3 lanes |
| 2 | late afternoon | AB-FULL record for the fresh cell: ratio and interval per metric, verdict EQUIVALENT / DIFFERENT / INCONCLUSIVE with n per arm still needed | none |
| 3 (Nov 5) | morning | if INCONCLUSIVE: the remaining rounds of the table, or the 3 d2d lanes at k=1 to measure gap G4; if DIFFERENT: attribute with the per-area time split and the known-gap list, open d2d fix cards; if EQUIVALENT: stop, update the fidelity-gate delta spec (AB-TOOL) for the user | 3 lanes |
| 3 | afternoon | decisions 4 and 10 from section 5 (Gate 3, late-band cell, Tier B); the Gate 3 start if the user gives the go and the fresh cell is closed | per decision |

Fallbacks: a restarted host or a `/tmp/d2lod-live.notice` pushes the lanes by a day; a gate run that fails does not change the schedule except decision 4; if the first round has two weak lanes out of three (restarts), stop and look at the harness before spending round 2.

## 7. What I did not check

- No game, no host, no run; the capacity figures are the cost table and the parity measurement, not a fresh sample (the check would load a host that is running Gate 1).
- The Windows PC facts come from the private build doc of 2026-11-02 and the owner's 2026-11-03 window change; its gate name list and the console behaviour were not read or tested.
- Agent-hour figures are estimates (U); the exact flags of `d2restore`, `oog` and `d2mapdump` were taken from docs/snaplib.md, docs/cheats.md and docs/boss.md, not from `--help`.
