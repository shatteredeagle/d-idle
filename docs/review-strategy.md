# Strategic review: plan, risk, verification debt, process (agent STRATEGY, 2026-10-29)

> status: SUPERSEDED by docs/plan/plan-v2.md (sections 5, 10, 11) and openspec/specs/soak-gates/spec.md; the freeze text and the G1 criteria wording here are historical.

> **Superseded in part (DOCSYNC, 2026-10-30).** The plan and gates below were written 2026-10-29 before the planning session. Current authority: [plan/plan-v2.md](plan/plan-v2.md) (sections 2, 5, 10, 11), [decisions.md](decisions.md) and [plan/STATUS.md](plan/STATUS.md). What changed: (1) the **feature freeze** is lifted for M0-M9 only (user, plan-v2 s10); everything not on that list stays frozen. (2) **Death**: no fixed "deaths <= 2/h" any more; the death rule is plan-v2 s10.2 (streak, recovery share, rate; implemented as d2soak codes). (3) **G2** is a stability + economy soak, not a progression soak (plan-v2 s5). (4) The **Andariel hold** is the knob `boss.andarielMinLevel` (default 15), temporarily set to 22 on running attempts (plan-v2 s11) and gated by the Monastery route guard (needs a merc or clvl 20, s11.2). (5) Gates are now split into stability gates and soft progression gates (plan-v2 s5). The G1 attempt history is in [g1-log.md](g1-log.md). The text below is kept as history; statements marked "(stale)" are out of date.

Scope: the plan and the risk, not the code line by line. Master b16de10. Docs only.
Tags: **V** = read in the repo or measured by me (command or file named); **S** = my reasoning from V facts; **U** = unverified opinion.
Read: PLAN.md, AGENTS.md, docs/INDEX.md, CHANGELOG.md (Unreleased), docs/soak-analysis.md, docs/soak24.md, docs/BACKLOG.md,
docs/liveruns*.md, docs/review-{overnight,day,evening,night}.md, docs/act3.md (head), docs/container.md, docs/knobs.md, `git log --first-parent -200`.
Plus read-only ssh to the worker host at 11:42 (uptime, ps, the running gate's soak dir). Nothing was started, stopped or written on the worker host.

## 0. The numbers (V)

| what | value | how |
|---|---|---|
| Go, non-test | 97.5k lines, 780 files, 46 internal packages, 33 cmd binaries | `git ls-files '*.go' \| grep -v _test \| xargs cat \| wc -l` |
| Go tests | 43.0k lines | same, `*_test.go` |
| docs | 155 md files, 14.8k lines | `git ls-files '*.md'` |
| commits since 2026-10-28 22:00 | 649 (178 first-parent merges) | `git log --since` |
| live evidence for the unattended bot | one Sorceress (Soaky) 1 to 7 over ~3.5 h in 4 runs, 1 panic, 1 pause flap (docs/soak-analysis.md); short scripted live checks per area (INDEX s1) | docs |
| the "2 h gate" now | RUNNING on i4 since 10:48: at 52 min clvl 7, 12443 XP/h, **2 deaths**, 0 rungs, 0 restarts, alerts ok | `~/d2lod-live/soak/i4-20261029-104844/status.txt`, `alerts.json` on the worker host |
| that run's limits | `MinXPPerHour 15000`, `MaxDeathsPerHour 1` (not the `--gate` profile, which sets min XP 0) | `limits.json` there vs `tools/soak/soak24.sh:81` |
| that run's binary | `livebot/d2bot` mtime 10:48, `--version` prints `dev` (no commit) | ssh |
| The worker host | load 26.6 / 31.4 / 32.9 on 16 cores; `tdarr-ffmpeg` at **nice 0, 753 % CPU**; ComfyUI python nice 0 97 %; Game.exe (i4) 97 % | `uptime`, `ps -eo ni,pcpu,comm` |

Two consequences right away (V/S):
1. The running gate tests a binary built before at least BOSSGATE, SHOPKIT, EQUIPFIX, DENQUEST, BELTPLAN, SOAKTOOLS2, QUESTSAVE, XPLOSS, ENTRYFIX, TICKWATCH, FARMFIX and FIELDGATE were merged (V: those merges are dated 11:20 to 11:38 in `git log --first-parent --date`). It is a baseline of the old build, not a verdict on master. With the limits it was started with it will most likely FAIL on deaths/h and XP/h (S: 2 deaths in 52 min, 12.4k < 15k). Neither failure says anything about the new fixes.
2. Nobody can say which commit a live result belongs to: the live binary is unstamped (`buildinfo.Version` is only set by `packaging/release.sh`, V `internal/buildinfo/buildinfo.go:2`). This silently corrupts every "fixed (S), live untested" line.

## 1. Honest assessment

### What is solid (V, live evidence exists)
- The foundations: memory reads through /proc with the ptrace cap, the d2input PostMessage helper, the iso transform, OOG menus, the managed launch with subreaper, multi-instance on private Xvfb, the container running the real game under plain Wine with CapEff=0 (docs/container.md numbers). This is the hard, novel part and it is done well.
- Act 1 Normal combat and town for a Sorceress at low level: hours of real play, deaths and corpse pickup, chicken exits, the town loop (docs/soak-analysis.md, docs/bot.md, docs/town.md).
- The honesty discipline: V/S/U tags are used almost everywhere and the docs mostly do not overclaim. The CHANGELOG "Unreleased / known issues" is accurate (it even says the gate never completed). Keep this.
- Tooling for evidence: d2soak samples, alerts, minute summary, the soak-analysis method (counts from logs). SOAKANALYSIS was the single most useful document of the day because it is the only one built from a long real run.
- Reviews singly with scratch tests found real Highs (night H1 to H3 were reproduced by tests, V in docs/archive/review-night.md).

### What is speculative (code + fakes only)
- Everything after the Act 1 chain: Act 2 past the Summoner portal, Acts 3 to 5 (every NPC row, object class id, portal class and quest bit is U, docs/act3.md s1, docs/actbot.md), all Nightmare and Hell rules (immunity, affix flag layout, resist penalty, missiles: U, INDEX P4), farm runs, endgame builds, cube jobs, mercenary revive/equip, the fleet beyond one slot, the stop chain under `docker stop`, the wizard with a stranger's files.
- Every knob range ("No knob has a recorded live A/B test of its range", docs/knobs.md s1, V).
- Every fix to a review High (INDEX s4: all "fixed (S)", none re-run live).
- Much of the failsafe stack: `--supervise`, panic recovery, the TICKWATCH watchdog, CRASHDRILL, the recycler drill, the sentinel death drill: merged, never run live (CHANGELOG Unreleased, V).

### What live evidence says is actually broken (V, docs/soak-analysis.md)
The live data contradicts the plan's sequencing: the bot cannot yet keep itself supplied in Act 1 Normal. Empty belt most of the time, 3 TP casts in 3.3 h, 11 empty-belt trips, Blood Raven 0 kills in 189 attempts, the Den quest never closes, a 25 min pause flap (12 % of the soak), 24 % of counted XP not kept. The offline layers built on top (Hell resource tuning, mana-burn tracker, antidote policy, farm loops) presuppose a resource loop that does not work at clvl 6.

### Top 10 unverified subsystems by blast radius x probability of being wrong (S, ranked)

| # | subsystem | blast radius | P(wrong) | why (evidence) |
|---|---|---|---|---|
| 1 | **The failsafe/supervision stack as a whole**: in-bot ladder + self-pause/auto-resume + recycler + panic recovery + `--supervise` + TICKWATCH watchdog + sentinel + fleet restart + d2soak alerts | every unattended hour; a wrong actor kills a healthy game or loops a dead one | high | at least 6 independent actors can restart or exit something; four reviews in a row found Highs inside this layer itself (overnight H1/H2, day H2, night H2/H3, M1 to M6); the only long live run hit a pause flap and a failed leave; supervise/watchdog/crash drill never ran live (V docs). No document says which actor owns which fault. |
| 2 | **Save and XP persistence** | all progress; a corrupt or stale .d2s loses the character's day | high | 7.3k of 30.6k XP counted but not kept (soak-analysis s6); `internal/launch/recycle.go:59` (saved on area change) vs `cmd/d2bot/live.go:289` (only on Save and Exit) disagree; `saves.Snapshot/Verify` not wired (BACKLOG 9); `Deps.SavedExp` not wired in cmd/d2bot (V grep: only declared in `internal/bot/runner.go:46` and read in observe.go) |
| 3 | **Resource economy (potions, TP, gold, shop budget)** | deaths, chicken loops, poverty spirals: the main cause of lost time | high (proven broken before the fixes) | soak-analysis R2; SHOPKIT and BELTPLAN fixes merged offline; FieldGate and the thin-belt retreat are **default off** (merge b16de10, 3cde7ad), so the soak will not exercise them unless someone turns them on |
| 4 | **Goal manager, campaign chain and persistence** (holds, locks, seeded goals, prune, quest events, chain to NM/Hell) | the bot runs the wrong goal, or the user's pause is overridden, after any restart | medium-high | Highs here in three reviews (day H1, H5; night H1, H2); each fix added persisted state (Locks, HeldAt, seeded marker, prune list); restart paths only in fakes |
| 5 | **Quest-state semantics** (clear vs turn-in bits) and task-graph completion | the campaign cannot advance past a quest whose bit it misreads: Act 1 never completes | high | Den never closed in 3.5 h (soak-analysis R6); "clear and turn-in bits still U" (INDEX P0 #6); Acts 3 to 5 bits U |
| 6 | **Boss scripts** (Blood Raven, Andariel, Smith, Duriel, Summoner, Council, Diablo, Baal) | stalls the chain; deaths | high | Blood Raven 0/189 live; Smith never found; Andariel never fought; only the Countess killed, under god mode (INDEX s1) |
| 7 | **Wine/process hygiene over 24 h** (wineserver fds, winedevice orphans, helper deaths) | the 24 h soak dies at hour 10 for a non-bot reason | medium | wineserver at **1770 fds** at 52 min of the current run (V samples.jsonl tail); judged only by plateau after 2 h (soak24 s3); ~35 MB per Wine restart (leakhunt); helper deaths under load (combat.md:106) |
| 8 | **Nightmare/Hell reads and rules** (immunity, affixes, resist penalty, missiles) | every NM/Hell fight, but gated off in Normal (review-evening V) | near certain somewhere | offsets U (`d2/monflags.go`), M10 resist penalty double count open; zero NM/Hell play |
| 9 | **Acts 3 to 5 task graph, NPC menus, portals** | campaign stalls in a new act | near certain somewhere | all rows and class ids U (docs/act3.md s1, actbot.md); bounded by the failsafe only if #1 works |
| 10 | **Fleet/container stop chain and reaping** | a container user's save cut mid-write on `docker stop` | medium | evening H1/H2 fixed offline; ENTRYFIX exit code fix offline; never run live (INDEX P5 #29) |

Ranked lower on purpose: merc (small blast while it is optional), stream/publish (not on the critical path), knob ranges (only matter once handicap profiles ship, which is parked).

## 2. Was "build acts 3-5 / NM / Hell / farm / endgame / container offline before the first long soak" sound?

No, not in this proportion (S). The parts that were sound:
- Research docs and data from the real MPQ tables (act3/4/5 plans, difficulty research, builds): cheap, durable, and they will be needed.
- The container spike: it retired a real architectural risk (headless rendering, ptrace without caps) early and was verified live.
- Simulations and fuzzing that found real bugs (grid cap, NaN knob, nil snapshot; docs/coverage.md).

What was not sound:
- **Inventory of unverified code.** About 100k lines exist against a few hours of long-run evidence. Each offline feature encodes guesses (U offsets, U quest bits, U dialog rows) that will each need a live pass and probably rework. Writing it before the foundations below it are proven means the rework lands on top of more code, and the review load grows with it (four merge-risk reviews in one day, each finding Highs).
- **Wrong dependency order.** Hell resource tuning before the Normal resource loop works; farm loops before one boss kill without god mode; NM chaining before the Den quest closes; fleet of N before one slot runs 24 h.
- **The verification capacity was the bottleneck and the plan ignored it.** With the worker host at load ~30 and a guard of 24, live slots were mostly closed (docs/liveruns.md s1: "written while the worker host was too loaded"). Five offline agents kept producing; nothing could drain the queue. Debt grows monotonically under that regime.
- **Complexity in the failsafe layer instead of fewer failure causes.** Many of the failsafe additions (supervise, watchdog, panic budget, crash drill) respond to failures that the Act 1 resource loop and pause flap produce. Fixing the causes would have removed load from a layer that is itself unverified.

What to change: a **WIP limit on unverified behaviour**. No new feature work until master passes the gates in section 3. After that, a feature merges only (a) with a live run line in docs/progression/results.md, or (b) default-off behind a knob with a dated owner who will live-test it within 48 h, otherwise it is reverted. Keep at most ~15 "fixed (S), live untested" items open at any time; the dispatcher stops launching feature work when the count is over.

## 3. Re-prioritised plan with gates

### Freeze now (no new code, no new agents) (S) (stale: freeze lifted for M0-M9, plan-v2 s10; the list below is what stays frozen)
Acts 3 to 5, Nightmare/Hell rules, farm runs (`runs:NAME:N`), endgame builds, cube jobs, mercenary farming, fleet multi-slot, party/multiplayer, setup wizard UI, stream/Kick/publish, viewport, idle-game layer (already parked). Bug fixes found by live runs in these areas are fine; features are not.

### Delete or archive to cut weight (S; the user decides deletions)
- `cmd/spike` (763 lines, PLAN says "do not extend"): delete once nothing imports it.
- One-off live rigs whose results are recorded: `cmd/act2live` (1301), `cmd/classsmoke` (1496), `cmd/druidlive`, `cmd/necrolive`, `cmd/friendlive`, `cmd/levelerlive`, `cmd/combatclass`, `cmd/d2helpersoak`: fold the still-useful bits into `cmd/probes` subcommands or move to `tools/rigs/` outside CI. Each costs build and review time on every refactor.
- Default-off knobs with no live owner after one week (FieldGate, thin-belt retreat, `Bot.CubeJobs`, `combat.missileDodge`): turn on with a live A/B, or delete the code path.
- Dead state flagged by reviews (night M7 `Goal.HeldAt`, if GOALHOLD did not already remove it: check).
- Docs: move to `docs/archive/` the stale planning layers: `docs/milestones/*` (rescored 2026-10-28, "stale" per INDEX), `docs/review/*` (design review 10-28), `docs/wave2.md`, `docs/goals/catalog.md` (127 objectives, research), `docs/archive/prior-art/meta/*`. Merge the four `review-*.md` open findings into ONE table of open findings (id, severity, state, fixing commit, live check) and archive the bodies. INDEX, CHANGELOG, BACKLOG, liveruns and milestones restate the same state five times and disagree within hours (e.g. CHANGELOG says review-night.md "is not in this tree" while it is, V): keep INDEX s1/s2 as the one status source and generate BACKLOG from it.

### Next 24 h
| step | what | gate / pass criteria |
|---|---|---|
| T0 (now) | Let the running gate end (~12:50). Re-judge it, do not rerun it: `d2soak report --in <run>/samples.jsonl --profile m1 --min-xp-per-hour 0`. Treat it as the **pre-fix baseline** (deaths/h, chicken/h, paused %, true XP/h, belt-empty minutes). | none; it is a baseline |
| T1 (1 h, offline) | Stamp every live build with the commit (`buildinfo` falls back to `debug.ReadBuildInfo` `vcs.revision`; `tools/live.sh` logs it; d2bot prints it in the first log line and `/status`). | `d2bot --version` on the worker host prints a hash |
| T2 (30 min, offline) | Write the failsafe ownership table (one owner per fault: death overlay, panel open, OOG, game gone, helper gone, frozen, no progress, panic, deadlock) and set the others to observe-only for soaks (sentinel `observe`, supervise off or on, not both with the watchdog). | a table in docs/stallfatal.md; soak24.sh matches it |
| T3 (5 min live, i4 or i5) | Save semantics: does an area change write the .d2s? Compare `saveXp` and `charXp` across an area change and across a hard kill (soak-analysis s6). | answer recorded in results.md; recycle.go:59 or live.go:289 comment corrected |
| G1 (2 h live, i4, master stamped) | **The real 2 h gate**: `soak24.sh start i4 --gate` on Soaky (continuing, no cheat) with FieldGate and the thin-belt retreat ON (the soak showed they are needed; default-off fixes are not tested by a default soak). One injected fault at ~60 min: kill Game.exe (the managed bot must restart it). | see G1 criteria below |
| G2 (8 h overnight, i4) (stale: now a stability + economy soak, clvl floor 16, Andariel attempt at the knob floor, merc hired; plan-v2 s5) | Only after G1 passes. Same build, `soak24.sh start i4 --for 8h --profile m24`. Fewer instances running overnight (section 4). | m24 limits; no manual action; wineserver fds plateau judged at 4 h and 8 h |

**G1 pass criteria (S; numbers calibrated from the Soaky baseline; item 2 already follows the new death rule; item 5 and 7 unchanged; plan-v2 s5 adds soft progression gates)**
1. No process exit except planned ones; 0 panics; 0 manual interventions; d2bot and the game alive at 2 h; graceful stop saves (`saveXp == charXp` at the end, xpLost 0 on planned recycles).
2. Deaths follow the death rule (docs/plan/plan-v2.md 10.2, `d2soak report`, docs/soak-analysis.md section 12), not a fixed count: no `death_streak` (3 deaths in 30 min with no clvl gain, or 5+ in 60 min), no `death_recovery_stuck` (recovery above 10 % of a rolling 60 min, or one death to back-in-field above 5 min), no `death_rate` (> 15 deaths per 1000 kills with >= 150 kills in the window); WARNs are read, not ignored. Chicken exits <= 4/h; no single state > 10 min without progress; paused time < 2 % (baseline 12 %).
3. True (saved) XP/h >= the baseline's true XP/h at the same clvl (Soaky 6.6k/h wall over clvl 1 to 7; current run 12.4k/h counter at clvl 7). A drop is a fail even with no crash.
4. Belt: no window > 5 min with 0 healing potions while outside town; at least one TP bought and used.
5. No Blood Raven attempt while the BOSSGATE refuses (grep `blocked: not ready`); no boss-run thrash (< 1 start per 30 s).
6. The injected Game.exe kill is detected and the bot is back in game within 3 min, with one restart counted, not more.
7. wineserver/winedevice: no orphaned winedevice after the injected restart.
If any fails: fix only that, rerun G1. No feature work in parallel.

### Next week (gated, S)
| day | goal | gate to the next |
|---|---|---|
| 1 | G1, G2 (above) | G2 passed |
| 2 | **24 h soak** on master (i4), profile m24, one character; a second character on another instance only if load allows (section 4) | 24 h pass = the alpha soak half |
| 2-3 | **Act 1 complete, unattended, no god mode**: Den claim (DENQUEST), Blood Raven when ready, Tristram/Cain, Smith (find out why he is never found), Andariel with real potions (stale: Andariel is not attempted unattended before the free Rogue and a potion stock, plan-v2 s11). Booti or Towni, real gear by save edit allowed, fights not cheated. | Warriv hop to Act 2 by the bot = alpha |
| 3-4 | Fix list from the 24 h log (same method as soak-analysis, ideally `d2soak` built-in report, soak-analysis WP8) | second 24 h soak passes with the fixes |
| 4-5 | Act 2 live chain on Navi (ACT2LIVE7: wp 46, tomb 73, Duriel, Summoner), then Act 2 unattended | Act 2 completes unattended |
| 6 | First Nightmare contact, read-only: the `monflags` and `missiledump` probes, resist penalty readout (settle evening M10 first, BACKLOG 19 RESISTCHECK), on an NM character by save edit | offsets V or corrected |
| 7 | Container user test by a non-agent (docs/container.md checklist) on the build that passed the 24 h soak; one slot only | the stranger reaches in-game |
Acts 3 to 5 start only after Act 2 runs unattended. Fleet N>1 only after one slot passes 24 h in the container.

## 4. Live-verification schedule under the worker host's load

Facts (V, 11:42): load 26.6 to 32.9 on 16 cores; `tdarr-ffmpeg` runs at **nice 0** and takes about 7.5 cores; ComfyUI 1 core; each Game.exe 1 core even idle.
Reasoning (S):
- The load average is the wrong guard. It counts runnable threads, and ffmpeg runs many; it says nothing about whether Game.exe and d2bot get CPU when they need it. The failure mode that matters is a starved game (tick lag, frozen frames, helper timeouts), and the bot already measures it: runner tick age (TICKWATCH), the frozen-frame detector, helper reachability.
- Linux CFS weights by nice: with ffmpeg at nice 19 and the game at nice 0, the game gets roughly 70x the share of an ffmpeg thread under contention (U exact factor; the order of magnitude is the standard CFS weight table). Off-peak scheduling does not help much because the Tdarr queue is continuous.
- "Per-instance nice" for the game the other way (negative nice) needs root, and worker-sudo is allowed only for setcap (AGENTS.md): not an option.

Recommendation to put to the user (in this order):
1. **Renice the Tdarr worker on the worker host to 19** (or set the Tdarr node's worker niceness / lower its CPU worker count, or pause the worker host node during soak windows; Tdarr has a node schedule). One action by the user, outside the project; it removes the root cause. If ffmpeg runs as the worker user, `renice -n 19 -p $(pgrep tdarr-ffmpeg)` needs no root (U who owns it).
2. **Replace the load guard with a health guard**: launch live work when (a) the soak instance's tick age and frame times are healthy, (b) free memory and disk are fine, and (c) `tools/capacity.sh --add <cost>` is not NO (floating cap, docs/process/worker-capacity.md; replaces the fixed Game.exe count). Keep a load ceiling only as a backstop, raised to 28 once Tdarr is niced.
3. **Run fewer instances**: during G1/G2/24 h at most two games (the soak + one d2-live slot); i0 stays as is; all others stopped. Each idle Game.exe spins a core (docs/container.md, V 102 %), so stopped instances are free capacity.
4. **Night slot for the long soaks** (G2, 24 h start in the evening), day slots for short (< 15 min) live checks one at a time.
If the user declines 1: raise the guard to 28, cap at two games, and accept that soak timing numbers (stall, RSS) are noisier; judge tick age instead of load.

## 5. Process review

### What the fan-out got right (V/S)
- Disjoint directory ownership plus worktrees kept merge conflicts rare: 6 of the last 300 first-parent merges mention reconciling ("keeps ...", "reconciled with"), the rest merged clean (V `git log --format=%s`). Flaky tests were fixed rather than quarantined (`tools/ci/flaky.txt` empty, V).
- Commit-often and the status board made dropped agents cheap.
- Tagging discipline (V/S/U) and the "never present a guess as a fact" rule held up across ~100 agents. That is unusual and worth more than any feature.
- Opus reviewers singly, with scratch reproductions, found real defects; that is the right use of the strong model.

### What it got wrong (S, with evidence)
- **Throughput far beyond verification capacity.** ~178 merges in ~14 h against ~0 to 1 usable live slots; the queue of "fixed (S), live untested" only grew (INDEX s4: every High fixed offline, none re-run). Tokens spent on code that cannot be tested this week are mostly spent on future rework.
- **Fix-on-fix churn in the same layer.** The failsafe/supervise/goal-persistence layer was reviewed four times and had Highs each time; each fix added state or an actor. A review cycle every few hours on unverified code converges slowly; one live gate would have found the important ones faster.
- **Duplicated or overlapping work.** cmd/d2soak was touched by SOAKTOOLS, SOAKTOOLS2, SOAK24, SOAKPID, LOGPARSE, SOAKFIX, XPLOSS, DASHWIRE; equip by EQUIPFIX and SHOPKIT (reconciled at merge, V subject 722e773); status docs by DOCINDEX, DOCSYNC, CHANGELOG, BACKLOGGEN, LIVERUNBOOK, MILESTONES2 (six agents whose output is a restatement of the log).
- **Over-reporting.** Every agent writes code, a domain doc, a board card, a report, and often a backlog/INDEX edit. 14.8k doc lines for a project with one long live run; several docs are stale on the day they were written (INDEX "no live run since the first version" while the Soaky soak ran; CHANGELOG on review-night).
- **Process failures that cost evidence:** the live binary is unstamped; the current gate was started with the wrong limits (min XP 15000 instead of the gate's 0), so its verdict is not the gate's; the gate runs a build that predates the fixes it was meant to judge.

### Leaner operating model (S)
1. **Three lanes, not five-plus agents**: (a) one live lane (d2-live, Sonnet) that runs gates and soaks and writes the evidence; (b) one or two fix lanes (d2-worker, Sonnet) that only take items whose evidence is a live log line or a reproduced test; (c) an Opus reviewer per ~10 merges or per gate failure, singly. Research only on demand (Haiku/Sonnet), docs regenerated by script, not by agents.
2. **Live-first definition of done**: a behaviour change is done when a live line exists; until then it is "merged, unverified" and counts against the WIP limit (section 2).
3. **One status source**: INDEX s1/s2 plus a single open-findings table; BACKLOG generated from them; the CHANGELOG from merge subjects by script. No agent whose only output is restating state.
4. **Haiku for log mining**: the soak-analysis style counting is mechanical once the method exists (or built into d2soak, WP8).
5. **Stamp and pin every live run**: commit hash, profile, knob overrides, character, instance in the first log line and the soak dir; refuse to start a gate from an unstamped or dirty build.
Expected effect (U): roughly a third of the tokens, and the verified surface grows faster than now because the bottleneck (live evidence) gets the attention.

## 6. The 10 highest-value next work items (ranked, S)

1. **Re-judge the running gate as a baseline** after 12:50 with `--min-xp-per-hour 0` and record deaths/h, chicken/h, paused %, true XP/h, belt-empty minutes in docs/soak-analysis.md (Haiku or the live lane; no new run).
2. **Build stamping** (`internal/buildinfo` fallback to `vcs.revision`, live.sh and soak24.sh log it, gate refuses `dev`). Small, d2-worker.
3. **Failsafe ownership table + soak config to match** (one owner per fault, the rest observe-only), docs/stallfatal.md and tools/soak/soak24.sh. d2-worker plus an Opus read.
4. **Save semantics live check** (5 min, soak-analysis s6) and, depending on the answer, wire `Deps.SavedExp` from the .d2s and periodic Save and Exit; wire `saves.Snapshot/Verify` (BACKLOG 9 SAVESNAP).
5. **G1: the real 2 h gate on stamped master**, FieldGate and thin-belt retreat on, one injected Game.exe kill, criteria in section 3.
6. **User decision on Tdarr niceness / instance cap** (section 4); until then cap concurrent games at two.
7. **G2 8 h overnight, then the 24 h soak**, each judged on stamped builds; fix only what the logs show.
8. **Act 1 chain unattended without god mode**: Den claim, Blood Raven gate, Tristram/Cain, Smith root cause, Andariel with real potions (INDEX P0 #2, #3, #5).
9. **Settle the resist penalty question** (BACKLOG 19 RESISTCHECK, research) and fix evening M10 before any Nightmare fight; it is cheap and will otherwise poison every NM/Hell danger number.
10. **Consolidate and prune**: one open-findings table from the four reviews, archive stale docs, retire `cmd/spike` and the one-off live rigs, and a deadline (one week) on default-off knobs: live A/B or delete.

## Dead ends and limits of this review
- I did not read code beyond greps; "fixed" claims are taken from INDEX s4 and commit subjects (S).
- Merge times are commit dates (V, 11:20 to 11:38); the live binary mtime is 10:48 (V).
- The CFS weight factor and Tdarr ownership on the worker host are U.
