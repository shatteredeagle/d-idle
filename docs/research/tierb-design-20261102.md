# Tier B design: A:B testing of the d2d backend against the original game (card tierbdesign, 2026-11-02)

> status: design, docs only (card `tierbdesign`). No build, no game start, no live run, no hook change. User approved Tier B design work on 2026-11-02; the hook release stays a separate decision. Parents: docs/research/original-game-lockstep.md (tiers A/B/C), docs/research/seed-study.md (seed pinning, divergence order), docs/research/lockstep-p1p2.md, lockstep-p3.md, lockstep-p4.md, docs/timescale.md, openspec/specs/fidelity-gate, docs/ab-design.md, docs/research/baseline-audit.md.
> Tags: **V read** = read by me in this repo or a cited source this session; **V run** = measured by a run (mine or a cited one); **D** = decompile note of the d2decomped project as cited by seed-study.md; **S** = secondary; **I** = my inference; **U** = unverified. Machines are "the worker host" and "the driver host". Private hook details are in the private hook repository's note "Tier B: private hook notes" and are not repeated here.

## 0. Recommendation first

1. **Do the statistical A:B first, not Tier B.** The goal "same bot commit, same character and seed, compare outcomes" can be met as *paired statistics* today: pin the map seed on both backends with one `.d2s` file (seed-study.md section 2: d2s 0xAB plus the active-difficulty bit; d2d honours the same field), reset the character from a kit copy per run, interleave the arms, and judge each metric by an equivalence interval. With the late-band CV of 11 to 14 percent **about 11 to 15 game.exe runs and as many d2d seeds** answer "d2d XP/hour within 15 percent of game.exe"; the fresh-kit sets measured today have a CV of 4 to 5 percent, which would bring it down to the spec floor of 5 runs per arm if a pilot confirms it (sections 3.4 and 6).
2. **Tier B cannot deliver a seed-for-seed cross-backend comparison on its own.** Even a perfectly stepped, deterministic game.exe diverges from d2d at the first monster: d2d's runtime RNG is one fixed-constant stream, game.exe steps the game seed once per unit made and rolls life, looks and drops on per-unit seeds (seed-study.md sections 1.3, 3, 7; D plus V read of the d2d source by that agent). Equal layout and equal spawn positions are reachable; equal fights, drops and XP are not, until d2d models the game-seed stream (a large fork card). So Tier B buys **game.exe self-repeatability and per-event diagnosis**, not a cheaper fidelity verdict.
3. **Move to Tier B design 1 (frozen virtual clock with a step verb, section 2.1) only when** the statistical A:B gives a FAIL or a gap above 15 percent that the known-gap list does not explain and the per-area time split cannot localise, or when the game.exe within-arm CV comes out above 20 percent (then the run count in section 3 doubles and noise reduction pays). Designs 2 and 3 are not recommended now.
4. **k=2 as a second regime: yes, tagged and separate** (section 6): kitk2 is inside the kitb noise on XP/h (-2.9 %), kills/min and deaths, with one consistent delay (clvl 6 reached 12 percent later); use it for pilots and screening, not yet for a fidelity verdict.

## 1. What true stepping must control in game.exe 1.14d

Determinism needs every input to the server's state to be a function of (seed, input sequence, frame number). The table lists every known source of wall time or entropy, who reads it, and whether today's clock hook (Tier A, docs/timescale.md) covers it. Coverage is as published in lockstep-p1p2.md "Per-API clock coverage" and lockstep-p3.md; anything finer is in the private note.

### 1.1 Time sources

| Source | Who reads it in game.exe | Evidence | Today (Tier A) | What stepping needs |
|---|---|---|---|---|
| `timeGetTime` | server frame gate: frames run when `timeGetTime() - last >= 40 ms` (1.14d `GAME_UpdateGamesProgress`), with catch-up clamp | S (D2MOO source, original-game-lockstep.md 2.1); V run: frame counter at 25 k fps to 0.015 percent for k = 1, 2, 4 (lockstep-p3.md) | scaled, V run | frozen between steps: the gate then yields 0 frames, a natural pause without a code patch (I) |
| `GetTickCount` / `GetTickCount64` | a few logic timers (waypoint refusal 10 s window, teleport landing stamp), frame statistics | V read (waypoint-trace.md, cited in original-game-lockstep.md 2.1); S for teleport | scaled from the same base, V run | same frozen clock as `timeGetTime` (one base, already so) |
| `QueryPerformanceCounter` | load measure in the frame gate; **seed of a new game** (`FUN_0052c280`) | S (D2MOO); D (seed-study.md 1.1) | scaled, V run (scaling does not make the seed repeatable: it still starts from the real counter) | frozen and started from a fixed value, or the seed pinned by the `.d2s` route (preferred: no hook needed) |
| `Sleep`, `SleepEx`, `WaitForSingleObject(Ex)`, `WaitForMultipleObjects(Ex)`, `MsgWaitForMultipleObjects(Ex)` | frame limiter sleeps, thread waits | V run: the hook's 1 ms floor fired 647 to 2491 times per run without changing the frame ratio (lockstep-p3.md, p4.md) | divided by k, V run | with a frozen clock a timed wait must return (yield) instead of sleeping forever, else the client loop deadlocks or spins; a spin costs one core per instance (I) |
| `timeSetEvent`, `SetTimer` / `WM_TIMER` | audio and video libraries | V read (lockstep-p1p2.md coverage list) | divided by k for the libraries loaded after the hook, V read | either frozen (timers fire only on steps) or left real for audio (muted, no state effect, I) |
| `GetSystemTimeAsFileTime`, `GetLocalTime`, waitable timers, delay-loaded imports | unknown whether used | U (lockstep-p3.md "not covered by the hook") | not covered | an import scan of the exe and the game's own DLLs (offline, no game) settles it; cheap first task of card TIERB-HOOK |
| DirectDraw / Direct3D present, vsync under Wine on Xvfb | client render cadence | U | not covered | not needed for server determinism if the client consumes packets per loop iteration (U); needed for "client memory shows frame N" (the read barrier) |
| Message pump (`PeekMessage` / `GetMessage`) and X11 input | bot clicks arrive through X11, Wine posts them to the queue at wall time; the client turns them into a packet for the server's next frame | V read (original-game-lockstep.md 2.3, seed-study.md 5) | not affected by any clock hook | **the main determinism gap**: inputs must be delivered only while the clock is frozen and be consumed before the next step (design 1), or be posted as packets at a frame boundary (design 2) |
| Server speed factor `game+0x1DBC` (0x400 = 1.0) computed from real time per frame | possibly unit logic | D (seed-study.md 1.4) against D2MOO "always 1024" (S) | indirectly scaled (I) | must read 0x400 every frame under stepping; U on 1.14d. One read-only probe decides it |
| Client/server threading and the loopback packet queue | single-player server runs inside the process; the client main loop calls the server frame and flush | D (seed-study.md 1.4); U whether any other thread touches state | n/a | if the server frame is called from the client loop (D), no cross-thread barrier is needed, only a "client consumed frame N" check (I) |
| Out-of-game loops (menus, character select, game creation) | wall-time UI | V run: frame stalls of 5 to 100 s around re-entry and area loads, longer at higher k (lockstep-p3.md, p4.md) | scaled | stepping starts at "first frame after the player is placed" (seed-study.md 7.7); OOG runs on real time and is excluded from the measured span |
| Wine internals (its own timer thread, the wineserver) | under the game's imports | S (original-game-lockstep.md 2.4) | not covered by design | only matters through the APIs above (I) |

### 1.2 Randomness

| Source | Rule | Evidence | Under stepping |
|---|---|---|---|
| Game seed at game creation | from `QueryPerformanceCounter`, unless forced or the save carries the map id with the active-difficulty bit | D (seed-study.md 1.1); V live: map id stable across 15 loads on one difficulty (docs/saves.md as cited there) | pin by the `.d2s` (setup action, no memory write); the conflict with docs/orig-spawns.md ("new map seed each time") is still open: card AB-SEEDPIN settles it |
| Per-unit seeds | every unit made steps the game seed; life, look, drops roll on the unit seed | D | deterministic given equal inputs and order on game.exe; **not modelled by d2d** (B2 of seed-study.md) |
| Missile seeds | from position and age | V run (game.exe code in the emulator, sk-ground-trace.md) | deterministic given equal frames |
| Vendor stock, gamble, hire lists | sunitproxy seed, rerolled on a timer | D | deterministic under a frozen clock |
| Client visual RNG (rain, ambience) | client copy | D | no server effect expected (I) |
| Bot side | nav and combat RNGs constant-seeded; bot timers on `internal/gclock`; the d2input helper's holds and sleeps are real time | V read (docs/d2d-determinism.md, docs/timescale.md) | the bot must take its clock from the step counter and the helper holds must become "hold for N steps" (both are bot-side work, card TIERB-BOT) |

### 1.3 What is already hooked, what is missing (summary)

Hooked and V run: the frame-gate clock, tick counts, the performance counter, timed waits and the library timers, all scaled by one factor from one base, null control at k=1 clean. Missing for stepping: (a) a clock that advances only on a step command; (b) wait semantics under a frozen clock; (c) an input boundary (the message path is untouched by any clock hook); (d) a read barrier (the bot reads client memory asynchronously, so "after step N" needs one counter that the client has consumed frame N); (e) the seed at creation (solved without a hook by the `.d2s` pin, pending AB-SEEDPIN); (f) the speed factor question; (g) the not-covered APIs list, to be closed by an offline import scan.

## 2. Three designs, cheapest first

| | 2.1 Frozen virtual clock, stepped by the bot | 2.2 Frame-synchronous stepping through the d2input helper | 2.3 Emulator-level replay |
|---|---|---|---|
| Idea | Tier A hook, but the virtual clock advances only on `step N` (N x 40 ms) from a control pipe; the frame gate yields nothing between steps; timed waits yield; the bot steps, waits until a client frame counter (read-only, already found in P1) reaches the target, reads memory, sends input while frozen, steps again | 2.1 plus input delivered as messages or packets at a frame boundary by the helper (not X11 events at wall time), a client-consumed barrier inside the process, the speed factor held at 0x400 | run game.exe's server frame headless in the existing function-level emulator (`tools/emu`), Win32 stubbed, fed with a recorded client-to-server packet stream; replay any stream bit-for-bit |
| Needs game addresses? | no in the hook (generic, like Tier A); the bot reads one counter it already reads | yes (client frame state, packet queue, speed factor): 1.14d-specific | yes, all of the server frame's callees; it is reverse-engineered execution |
| Feasibility | medium: the clock and waits are a small extension of code that passes its tests; the open risk is the client loop under a frozen clock (spin, deadlock, render starvation) (I) | low to medium: unknown thread structure around the packet queue (D says single loop, U), and every patch writes code or data of the game | low for whole-game play (graphics, sound, file and OOG stubs); medium for server-only replay of short scripted spans, which `tools/emu` already does per function |
| Risk of breaking the game | medium: a frozen clock can hang the window or audio threads; a hang is recoverable by an instance stop | high: code patches, input reordering, a desync between client and server copies | none for the live game (offline); high risk of a replay that is not faithful (stub errors) |
| EULA exposure (lockstep-licences.md 6) | same kind as Tier A: an in-process DLL that changes game timing; no game bytes patched; release would stay possible in principle | higher: a game-specific hook built from reverse engineering; any release becomes much harder | highest for distribution (runs Blizzard code outside the product); private use only, like the existing emulator work |
| Agent-hours (U estimates) | 12 to 20 (hook 6 to 8, bot step mode 4 to 8, idle and scripted determinism probes 2 to 4) | 25 to 40 on top of 2.1 | 60 to 150 for a server-frame replayer; far more for whole-game |
| A:B question it answers | "Does game.exe repeat itself given the same seed and the same timed inputs?" (P3a of seed-study.md, idle town then a short scripted walk); consistent snapshots for the bot; the game.exe arm's within-arm noise from timing removed; with a d2d run of the same seed, the first divergence tick per state class (positions, life, drops) | the same with the bot's real input path, so full bot runs repeat on game.exe; cross-backend first divergence per class with real bot play | "Given exactly the inputs d2d's server saw, what would game.exe's server have done?": a server-logic differential for whole spans, the strongest fidelity test, without any live game |
| Does it give seed-for-seed d2d vs game.exe outcomes? | no (d2d RNG stream, B2) | no (same) | no for d2d's own runs; yes for "game.exe server on d2d's packet stream" (a different, stronger question) |

Reading: every design answers a *diagnosis* question. None makes the fidelity verdict cheaper, because the cross-backend difference is dominated by different random draws, and those need a sample whatever the clock does (I, from seed-study.md 7.2 to 7.4).

## 3. The statistical alternative (do this first)

### 3.1 Protocol

1. Same bot commit on both arms (the set's pin); the d2d arm on the lab fork in lockstep (fidelity-gate "Fast d2d runs use the lab fork").
2. Same character: fresh characters restored from a kit snapshot before every run (`baseline-run.sh --fresh-from-set`, docs/process/baseline-runs.md section 10), so gold, potions, level and stash start equal (lockstep-p3.md point 4 showed drift without it).
3. Same seed per pair: after AB-SEEDPIN, each pair p uses one `.d2s` with map id m_p on game.exe and the same file (or `--seed m_p`) on d2d. Layout and spawn placement then match (D plus V emulator diffs, seed-study.md 3); fights and drops do not.
4. Interleave: pair p runs both arms in the same window; the arm order alternates.
5. Metrics per run, all per GAME hour from the minute lines: XP/h, kills/h, deaths/h, deaths per 1000 kills (the gated four), plus drops per 100 kills by quality, and time per area (minutes per area visit and share of time per area, from the minute line's area field). Same extractor on both sides (baseline-audit.md section 3: the definition only cancels then).
6. Exclusions: the fidelity-gate "Excluded seeds" rule applies to both arms; a game.exe lane marked weak by the quality script is reported and kept out of the verdict.

### 3.2 Noise band per k (three runs)

A three-run band per k is a pilot, not a gate: with n = 3 the 95 percent half width of a mean is 2.48 x CV (t 4.30 / sqrt 3), so 27 to 35 percent at CV 11 to 14 percent (D, the same arithmetic as baseline-audit.md). Use it to measure CV and to see gross regime effects (a k=2 lane collapsing like P3's k=4), not to claim equality. `pair_compare.py` answers "is the B median inside the A min..max", which is the right crude check at n = 3 and nothing more.

### 3.3 Fidelity score per metric

For each metric report the ratio r = mean(d2d) / mean(game.exe) with a 95 percent interval (delta method on the log ratio, or Fieller; Welch degrees of freedom, both arms' error included: baseline-audit.md section 3 shows that ignoring the game.exe arm's error overstates certainty). Verdict:
- EQUIVALENT when the whole interval lies inside [1 - m, 1 + m] (two one-sided tests, margin m);
- DIFFERENT when the interval excludes 1 and lies outside the margin on one side;
- INCONCLUSIVE otherwise, with the n per arm that would settle it.
Deaths use a Poisson rate ratio (exact conditional binomial interval); drops per quality class the same.

### 3.4 Sample size for "d2d XP/hour within 15 percent of game.exe"

Equivalence with margin m = 15 percent, alpha 0.05 per side, power 80 percent, unpaired, n per arm = 2 (z_0.95 + z) ^ 2 CV ^ 2 / (m - |d|) ^ 2 (D, normal approximation; add about 2 for the t correction):

| per-run CV (both arms) | true gap d = 0 (z = 1.28) | true gap 5 percent (z = 0.84) |
|---|---|---|
| 4 to 5 percent (fresh kit, clvl 1 to 7, kitb and kitk2, n = 3 each, section 6) | 2 (floor 5) | 4 (floor 5) |
| 11 percent (d2d soakfull, game.exe late runs without G2) | 10 | 15 |
| 12 percent | 11 | 18 |
| 14 percent (baseline-audit.md section 3 upper bound) | 15 | 25 |
| 20 percent (trigger for Tier B) | 31 | 50 |

Kills/h has a game.exe CV of about 21 percent (baseline-audit.md), so a kills/h equivalence at 15 percent needs about 34 runs per arm; at 20 percent margin (the spec's tolerance) about 19. Deaths at about 0.4/h on game.exe (baseline-audit.md "after") give about 1 death per 3 h run: 15 runs give about 15 events, a rate ratio interval of roughly 0.5 to 2: deaths can only be judged for gross differences (D).

Pairing by map seed helps only as far as the two arms' per-pair outcomes correlate. Within d2d, a local bot change kept correlation 0.6 to 0.95 (docs/ab-design.md), but across backends the RNG differs from the first monster, so expect a low correlation (layout and route only, U); plan with the unpaired numbers and report the measured correlation from the pilot.

Cost: a game.exe fresh lane is 3 game hours (3 h real at k=1, 1.5 h at k=2 if admitted), three hidden instances in parallel: 15 runs = 5 rounds = 15 h real at k=1, 7.5 h at k=2. The d2d arm is minutes per seed on the fork (fidelity-gate "Fast d2d runs").

### 3.5 What decides moving on

| Result of AB-FULL | Next |
|---|---|
| XP/h and kills/h EQUIVALENT at 15 percent, deaths not DIFFERENT | stop; the A:B is answered; no Tier B needed for fidelity |
| DIFFERENT on a metric and the known-gap list plus the per-area time split explain it | a d2d fix card per gap (the existing route, docs/d2d-gaps.md) |
| DIFFERENT or a gap above 15 percent that neither explains | TIERB-HOOK and TIERB-DET (design 2.1) to find the first divergence by state class |
| game.exe within-arm CV above 20 percent | TIERB-HOOK for repeatability (fewer runs), or more runs if hidden-instance hours are cheaper than 12 to 20 agent-hours |

## 4. Staged cards (ready to write into openspec/changes)

All live work on hidden instances only (never i0), started by the coordinator's tooling; agents never start, stop or log in a game. Capacity through the capacity check; markers with `d2cheat runstart`; 0 writes after start.

| # | Card | Agent | owns | exclusive | gates | Test plan |
|---|---|---|---|---|---|---|
| 1 | **AB-SEEDPIN**: settle the map-id pin on game.exe and d2d (seed-study P3-0, P3-1 for Act 1 town and Blood Moor) | d2-live | `docs/research/ab-seedpin.md`, `openspec/changes/ab-seedpin/**` | `instance:i5` | in-game map seed equals the `.d2s` map id in 3 coordinator starts from one file and from a copy in a second prefix; d2d `--seed` of that id gives the same room list and room seeds for Rogue Encampment and Blood Moor (or the conflict with docs/orig-spawns.md is explained) | read-only reads of `MapId` and `Act.dwMapSeed`, file 0xAB before and after; offline `drlg-dump` against a live room dump; no memory write, no hook |
| 2 | **AB-TOOL**: cross-backend A:B analysis (ratio, equivalence interval, Poisson deaths, drops, time per area, measured pair correlation, n per arm) plus a delta spec for the fidelity gate | d2-worker | `tools/ab/xbackend.py`, `tools/ab/xbackend_test.py`, `docs/ab-xbackend.md`, `openspec/changes/ab-tool/**` (delta spec inside) | none | unit tests on synthetic sets; reproduces `pair_compare.py`'s lane numbers on kitb and kitk2; `changecheck` 0 warnings; leakcheck 0 | offline only, the collected kit sets and existing d2d soak sets as fixtures |
| 3 | **AB-PILOT**: 3 pairs, game.exe k=1 on hidden lanes from the kit snapshot, d2d the same pinned seeds, same pin; measures CV per arm and pair correlation | coordinator set run plus d2-mechanic for the d2d arm | `docs/soaks/ab-pilot-<date>.md`, `openspec/changes/ab-pilot/**` | `instance:i2`, `instance:i3`, `instance:i5`, `d2d-lab-binary` (read-only use) | 3 of 3 lanes not weak, 0 writes after start, AB-TOOL report with CV and n per arm | `baseline-run.sh` set with `--fresh-from-set`, the d2d arm through `d2dsoak run` with `--seed` per pair; needs cards 1 and 2 |
| 4 | **AB-FULL**: n per arm from the pilot (5 at the fresh-kit CV of section 6, 11 to 15 at the late-band CV, for XP/h at 15 percent) | as card 3 | `docs/soaks/ab-full-<date>.md`, `openspec/changes/ab-full/**` | as card 3 | AB-TOOL verdict per metric; the decision table of 3.5 applied | interleaved rounds of 3 lanes; k=2 lanes only if section 6 admits the regime and the user agrees |
| 5 | **TIERB-HOOK** (only if 3.5 says so): frozen virtual clock, `step N`, wait semantics, import scan of the not-covered APIs; private hook repository, generic, no game addresses | d2-worker (private repo) | private hook repository only; public `docs/research/tierb-hook.md` (results, public-safe), `openspec/changes/tierb-hook/**` | none (no game) | the hook's own test suite under Wine with a stub game: frame gate yields 0 frames while frozen, exactly N frames per `step N`, no deadlock in timed waits, null control at free-running k=1 unchanged | stub programs only; needs the user's go for the new hook set before any game use |
| 6 | **TIERB-DET**: bot step mode (clock from the step counter, helper holds in steps) and determinism probes P3a idle town (5000 frames) then a 2 min scripted walk, twice from the same pinned `.d2s` | d2-live | `cmd/lockstepscan/**` or a new `cmd/tierbprobe/**`, `internal/gclock/**`, `docs/research/tierb-det.md`, `openspec/changes/tierb-det/**` | `instance:i4`, `bot-binary:tierbprobe` | equal state vector V1 of seed-study.md section 4 at every 250 frames for the idle span; first divergence frame per state class reported for the walk; speed factor read 0x400 every frame or reported | hidden i4, hook set with the user's go, run markers, read-only memory; then optionally one d2d run of the same seed to report cross-backend first divergence per class |

### 4.1 Mapping onto openspec/specs/fidelity-gate

| Requirement | A:B mapping | Change needed (delta spec in AB-TOOL, not edited here) |
|---|---|---|
| Twenty percent tolerance | the margin m; the brief's question uses 15 percent | MODIFIED: tolerance per metric stays 20 percent for the gate; an A:B report may state a tighter margin and must name it |
| Verdict classes | PASS needs the d2d half width only and treats the baseline mean as exact | MODIFIED (proposed): PASS when the interval of the ratio, with both arms' error, lies inside the margin (equivalence); FAIL when it lies outside; else INCONCLUSIVE with n per arm |
| Minimum sample | 5 seeds x 2 virtual h on d2d; nothing on the game.exe side | ADDED: the game.exe arm needs at least 5 runs (baseline-audit.md: 5 runs bring the baseline mean to about 20 percent half width at CV 14 percent); the A:B sizes in 3.4 for a 15 percent claim |
| Same character, same setup | kit snapshot, `--fresh-from-set` | ADDED scenario: a pair uses one `.d2s` per seed on both backends (map id pinned); a setup-edited file is declared as SETUP |
| Known gaps are not bugs | unchanged; the per-area time split is the tool to attribute a gap | none |
| What d2d cannot validate | unchanged | none |
| Fast d2d runs use the lab fork | unchanged | none |
| A/B protocol for bot changes / sample size and interleaving | written for same-backend A/B with 8 seeds floor and paired SD | ADDED: cross-backend A:B uses the unpaired size unless the pilot shows a pair correlation above 0.5; interleaving per pair |
| Early baseline fields | fresh clvl 1 to 7 runs are exactly what the kit sets produce | ADDED (later): kit sets at k=1 with the current pin may feed `baselines-early.json` once the protocol is accepted; k=2 sets only if the regime is admitted (section 6) |

## 5. What the user must decide

1. **The margin for the A:B claim**: 15 percent (the brief) against the spec's 20 percent; 15 percent costs about 1.8 times the runs of 20 percent (3.4).
2. **The `.d2s` map-id pin as a declared SETUP action** for A:B characters (a copied honest file stays honest; an edited map id makes the file setup-edited, fine for research runs, ADR 0014 as described in seed-study.md section 2).
3. **Hidden-instance hours**: AB-PILOT about 3 h real, AB-FULL about 15 h real at k=1 on three lanes (7.5 h at k=2 if admitted).
4. **Whether k=2 lanes may form the game.exe arm** (section 6 gives the evidence; it is not decided here).
5. **Tier B is a new hook set**: the frozen-clock step mode is instrumentation under ADR 0014 but not the hook set that was approved for Tier A ("clock scaling"); it needs its own go before any game use. Design 2.2 would add game-specific patches (a code or data write, listed in the ledger like a patch).
6. **The hook release stays a separate decision**; nothing here assumes it. Design 2.1 keeps the hook generic, so it does not make a later release harder; design 2.2 would.
7. Whether a d2d unit-seed fidelity card (seed-study.md card 6) is wanted at all: it is the only route to cross-backend equal fights and drops, and it is a large fork change.

## 6. kitb (k=1) against kitk2 (k=2): should k=2 become a second regime?

Command (read-only, on the driver host after the set finished at about 7:16 PM EST): `python3 tools/baseline/pair_compare.py <set dir>/kitb <set dir>/kitk2 --a-name kitb --b-name kitk2` (ops repo), exit 0. Both sets: pin 15506e05, fresh characters Freshc, Freshd, Freshe from the kit snapshot of set `kit`, three hidden lanes, rows cut at the first clvl 7 line; kitb at k=1, kitk2 at k=2 with the scaled bot clock. All six lanes are marked weak by the quality script for the same reason (restart counter 6 to 22 inside the run), so the restarts are a bot or harness property at both k, not a k effect. V run (the table), D (the statistics).

| metric | kitb k=1 (3 lanes) | kitk2 k=2 (3 lanes) | k=2 against k=1 |
|---|---|---|---|
| XP per game hour, per lane | 11040, 11445, 11952 (mean 11479, CV 4.0 %) | 11734, 10764, 10929 (mean 11142, CV 4.7 %) | -2.9 % of the mean; 95 % interval about -12.6 % to +6.7 % (Welch, 4 df); the k=2 median is just below the k=1 minimum |
| kills per game minute | 6.29 / 6.74 / 8.60 | 6.23 / 7.37 / 8.17 | +0.6 % of the mean, median inside the spread |
| deaths | 0, 0, 0 | 1, 0, 0 | no separable difference |
| restarts | 6 / 11 / 21 | 10 / 13 / 22 | same |
| game minutes to clvl 6 | 64 / 69 / 73 | 76 / 76 / 79 | +12 %: every k=2 lane slower than every k=1 lane (cause U: the real-time farm and goal timers and the unscaled helper holds of docs/timescale.md are candidates) |

In five lines:
1. At k=2 with the scaled bot clock the fresh clvl 1 to 7 game is inside the k=1 noise on XP/h (-2.9 %), kills/min (+0.6 %), deaths and restarts; the pair is three runs per arm, so this excludes only gaps larger than about 13 percent on XP/h.
2. One consistent effect: the time to clvl 6 is 12 percent longer at k=2 in all three lanes, so k=2 is not a perfect copy of k=1 in the first hour.
3. Recommendation: yes, k=2 becomes a **second, tagged regime** (`REGIME k=2` rows, its own baseline file or field, never merged into the k=1 baselines) for development, pilots and screening, which halves the real time of a fresh lane.
4. Not yet for the game.exe arm of a fidelity verdict or a gate: that needs a k=1 against k=2 equivalence within 10 percent on XP/h, which at the CV seen here (about 5 percent) is about 5 more pairs (section 3.4), and an explanation of the clvl 6 delay.
5. A by-product with weight for section 3: the fresh-kit game.exe CV of XP/h is 4 to 5 percent (n = 3, so its upper 95 percent bound is about 6 times that), far below the 11 to 14 percent of the late-band runs; if AB-PILOT confirms it, the A:B needs the spec floor of 5 runs per arm, not 15.

## 7. Limits of this design

- Agent-hour figures are estimates (U); the 2.1 figure rests on the Tier A hook already passing its tests (V read, lockstep-p1p2.md).
- The CVs in 3.4 come from late-band game.exe runs and d2d soaks (baseline-audit.md); the fresh-kit CV is measured by the kit sets (section 6) and by AB-PILOT.
- The cross-backend pair correlation is unknown (U); the unpaired sizes are the safe plan.
- Nothing here was run except `pair_compare.py` on collected sets (read-only).
