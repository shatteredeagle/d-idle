# d2d replicated soak: harness, metrics, pass rule (agent D2DSOAK, 2026-10-30)

The decisive d2d acceptance test (plan-v2 11, docs/d2d-backend.md 7): the same character and the same bot, N seeds x H virtual
hours on the private lockstep d2d, compared with our original-game soaks. A gap above 20 % on a gated metric is a fidelity bug.
Tags: **V** = verified (how stated), **S** = secondary, **U** = unverified.

Code: `cmd/d2dsoak` (runner, CLI), `internal/d2dsoakstat` (pure stats, fixtures in its `testdata/`), `tools/d2dsoak/`
(`baselines.json`, `baselines-early.json`, `run.sh`). Nothing of d2d's source is in the repo; the tool only starts the private
fork binary and talks to it through `internal/d2dctl`.

## 1. Usage

```
go build -o d2dsoak ./cmd/d2dsoak                      # or tools/d2dsoak/run.sh (builds here, ships to the worker host)

d2dsoak run -seeds 1-8 -hours 2 -out DIR -baselines tools/d2dsoak/baselines.json \
    -clock bot -cmd 'bot --backend d2d --sock $D2DSOAK_SOCK ...'      # the real test (needs D2DBACKEND, U flags)
d2dsoak run -seeds 1 -hours 0.05 -out DIR                              # dry run: steps the clock only
d2dsoak run -fake -seeds 1-8 -hours 2 -out DIR -baselines ...         # pipeline test, synthetic lines, no d2d
d2dsoak extract [-hours H] LOG...                  # metrics of minute-line logs (also original-game soak logs)
d2dsoak compare [-baselines F] [-json] [-holdout 16-23] RUN...   # RUN = run.json, seed dir or minute log; exit codes: section 3c; invariants: 3d
d2dsoak baseline 'NAME|LOG|SOURCE|NOTE' ...        # print baseline runs as JSON (how baselines.json was built)
```

Per seed `run` does: take `flock /tmp/d2d-lab.lock` (non-blocking, fails if another agent holds it), refuse if `pgrep -x d2d`
finds any d2d, refuse above load 45 or under 15 GB free on `/` (the fidelity-run safety rule), start the private fork
(`nice -n 10 ... --headless --no-video --lockstep --no-save --toggle autoloot=off,autoparty=off --seed S --start-screen ingame
--start-class C --start-name N`, own XDG dirs and socket under `~/d2d-lab/soak/home`, MPQ copy read-only), wait for `ping`,
optionally run a `-setup` devctl script (character spec), run `-cmd` (`sh -c`, env `D2DSOAK_SEED`, `D2DSOAK_SOCK`, `D2D_SOCK`,
`D2DSOAK_DIR`, `D2DSOAK_HOURS`; stdout+stderr = `bot.log`), read the final `clock`, `quit` d2d, SIGTERM/SIGKILL its process group,
check `pgrep -x d2d` is empty, release the lock. Output per seed in `DIR/seed-N/`: `bot.log`, `d2d.log`, `clock.jsonl`
(`{vmin, gameMs, tick, wallS}`), `run.json` (metrics, game_ms, wall time, speedup, how it ended). `DIR/aggregate.json` and
`verdict.txt` for the set.

Clock modes (the d2d control socket takes ONE client, so exactly one party steps):
- `-clock runner` (default): the tool calls `step 1500` (one virtual minute) until `-hours` are reached. The command must not use the
  socket. Meant for the dry run and for commands that only read files.
- `-clock bot`: the command (the bot with `--backend d2d`) owns the socket and steps the game itself; the tool closes its client,
  waits for the command to exit (or `-max-wall`, default 3 h) and then reads the clock. The command must stop itself after `D2DSOAK_HOURS` virtual hours. **U**: how D2DBACKEND paces the bot (its minute ticker must follow the d2d clock) is not
  merged yet; when the pacing differs, only the run length changes (see below), not the metric definitions.
- Run length = d2d `game_ms` at the end (V on the fork: `clock` returns it, docs/d2d-fork.md), never the bot's wall-clock timestamps.
  The minute lines are assumed evenly spread over it (needed for the clvl-at-1h/2h checkpoints). Without a clock the tool falls back to 1 line = 1 virtual minute (U).

## 2. Metrics (same fields d2soak prints, computed from the bot's `minute` lines)

| metric | definition |
|---|---|
| `xp_per_hour` | net character XP, last minus first line, per hour (unsaved progress lost to a restart counts against it, like d2soak's "from memory") |
| `kills_per_hour`, `deaths_per_hour`, `deaths_per_1000_kills`, `potions_per_hour` | sum of positive increments of the running counters (they reset when the bot restarts), per hour; potions = drunk counter |
| `gold_per_hour` | net gold balance change per hour (shop spending counts: a balance, not income) |
| `recovery_share`, `farming_share`, `town_share` | share of minutes by `state` (recovery = state `recovery` or runner state containing "recover"); absent in old logs |
| `repeat_sites` | deaths in the same area within 30 min of an earlier one, or within 3 min of the previous death (the d2soak death rule at minute resolution; area = the minute before the death) |
| `clvl_gain`, `clvl_1h`, `clvl_2h` | level change, and level at 1 h / 2 h (only when the run is that long) |

V: on the real attempt 16 and 18 logs (testdata) the extractor gives 11,226 and 9,240 XP/h against the d2soak report's 11,080 and
9,103 (within 1.5 %), 70 potions for a18 (= report), recovery share 11.7 % for a16 (g1-log: 11 %), farming 79 % (a18: 80 %). Kills/h
differs more (a18: 620 against the report's 688) because the report counts from 30 s samples; both sides of the comparison use the extractor, so the definition cancels.
Aggregate over seeds: mean, sample SD, min, max, 95 % half width (Student t) of the mean.

## 3. Pass rule

Baselines (`tools/d2dsoak/baselines.json`, source tag per run, all **V**: minute lines copied read-only from `~/d2lod-live/soak24/runs/` on the worker host, the
running G2 files were not touched): attempts 14 (first hour only), 16, 18; Soaky Sorceress clvl 11-14, Game.exe, m1 profile. Mean over runs:
XP/h 10.9k (9.2 to 12.2k), kills/h 500, deaths 0/h, potions 20/h (only a16/a18 have it). `baselines-early.json`: one run of the fresh clvl 1-7 character (soak-analysis, 6.6k XP/h, 0.29 deaths/h, **S** quality: old builds, no minute lines).

Per gated metric (XP/h, kills/h, deaths/h, deaths/1000 kills): `gap = (d2d mean - baseline mean) / baseline mean`;
`tol = max(20 % of baseline, abs)` with `abs` = 0.5 deaths/h, 1.0 deaths/1000 kills (baseline near 0, a relative gap is meaningless).
`hw` = 95 % half width of the d2d mean.
- **PASS**: `|gap| <= tol` and `hw <= tol`.
- **FAIL**: even the interval edge nearest the baseline is outside the tolerance (`|gap| - hw > tol`).
- **INCONCLUSIVE**: anything else, or fewer than 5 seeds; the output says how many seeds would settle it at the observed spread.
Overall: FAIL if any gated metric fails, PASS if all gated pass, else INCONCLUSIVE. Potions, gold, state shares, repeat sites and
clvl are reported with their gap but not gated (one clean baseline, vendor prices differ on d2d, level range depends on the start).

## 3b. Gate v2 (`d2dsoak compare -gate v2`, change gatedelta; v1 above stays the default)

User decision 2026-10-30. XP/h and kills/h stay at 20 %. Deaths: at most 1 per full 6 virtual hours pooled over the seeds, exact one-sided Poisson test against the
baseline rate (zero-death baseline counted as 0.5 deaths over its hours, **U**: only about 5 baseline hours, so the null rate is soft and the budget is the real
rule), exact lower 95 % bound of the d2d rate must not exceed the 0.15/h cap; under 6 h of exposure the verdict is INCONCLUSIVE. Deaths per 1000 kills: same Poisson test
with kills as exposure. Proxies gated at 25 %: `potions_per_100_kills` (**V**, from `potionsUsed` and `kills`), `min_life_fraction` and `damage_per_kill`
(need `dmgTaken` and `hpMinPct` in the minute line: emitted by the bot since change minutefields, **V** by unit test, not verified live; for older logs without them they print INCONCLUSIVE "not emitted" and the overall verdict cannot be PASS).
Definitions are in `internal/d2dsoakstat` (`gatev2.go`, `proxyHP`); the per-fight minimum is approximated per minute. Spec delta: `openspec/changes/gatedelta/specs/fidelity-gate/spec.md`.

Minimum sample for a meaningful verdict (the tool warns below these; numbers from the baseline spread, **I** = inference):
- **>= 5 seeds** and **>= 2 virtual hours per seed**, (**>= 8 seeds x 2 h = 16 h** recommended: XP/h per-run spread of the original is about 10 to 15 %, the t-interval needs about (1.96 s / 20 %)^2 runs, plus the baseline itself is only 3 runs).
- **>= 10 virtual hours in total for the death metrics**: at 0.1 to 0.2 deaths/h, 2 h per seed holds less than one death; deaths are Poisson, so a 0.5/h tolerance is only resolvable over many hours.
- The baseline has 3 runs (one partial) from 3 builds: its own uncertainty (min..max shown) is wider than 20 % for XP/h (9.2 to 12.2k). The rule treats the mean as exact, so a PASS here means "no evidence of a gap above 20 %", not proof. G2 (8 h, m24) should be added to `baselines.json` once it ends and its log is copied (`d2dsoak baseline`).
- Same character: the original runs are a clvl 11-14 Sorceress with her honest-play kit. A d2d run started from a fresh clvl 1 is NOT comparable with them; use `baselines-early.json` for that (weak), or give d2d the same level/kit by a `-setup` script (cheats are allowed on d2d's throwaway character; every number that depends on the setup is then only as good as the setup).

## 3c. Exit codes and invalid runs (change soakfix)

`run -baselines`, `compare` (text and `-json` alike) exit: **0 PASS**, **1 error** (I/O, bad input, or some seeds failed to run), **2 INCONCLUSIVE**, **3 FAIL or NO-DATA** (no usable seed: nothing was judged), **64 usage**; an unknown verdict is also 3. The last line of the report is `OVERALL: <verdict> (exit <code>)`. A script may only treat 0 as PASS. `-exit-zero` (run, compare) restores the old "print and exit 0" for scripts that read `verdict.txt` instead; `compare -save`/`run` always write `verdict.txt`, also with `-json`.
`tools/d2dsoak/run.sh` passes the code through (its `ssh` returns it).

A run is **invalid** (field `invalid` in run.json, reason printed as a WARNING) and excluded from hours, deaths and kills when: fewer than 2 minute lines (crashed at start: the d2d clock alone is not exposure, `hours` stays 0), the command failed (`cmdErr`), `cmd exited`/`wall limit` more than 5 % short of `-hours` (`hours` keeps the real covered virtual time), more than 2 % of the minute lines fail to parse (bad time, xp or counter; `dropped` counts them), NaN/Inf/non-positive hours or negative counters. Excluded seeds are listed under `excluded seeds` in the report and in `-json` (`excludedSeeds`, with `seeds` = the number counted), add 0 h and 0 kills, and do not count toward `minSeeds`; any excluded run makes the verdict at best INCONCLUSIVE. Duplicate seeds in `-seeds` and the same path twice in `extract`/`compare` are rejected. Fewer than `minSeeds` valid seeds, any seed under `minHoursPerSeed` or fewer than `minTotalHours` in total are INCONCLUSIVE (fidelity-gate "Minimum sample"; before this change they only warned). v2 `deaths_per_1000_kills` is INCONCLUSIVE below 200 kills (**U**, a floor of about 0.4 h) or with a zero-exposure baseline. `repeatSites` expands at most 1000 deaths per minute line.

## 3d. Invariants and holdout seeds (change gen-invariants)

Every report (`run`, `extract`, `compare`; text, `-json`, `verdict.txt`) carries a block before `OVERALL`: per seed HANG, STALL, WEDGE, PROGRESS, DEATHS, RESTARTS, EXIT counters and a verdict `PASS` / `VIOLATED(list)` / `NO-DATA`, a group line each for `tuning` and `holdout` seeds (`-holdout`, default 16-23), a `COVERAGE (proven on)` line and `INVARIANTS: ...`. The invariants verdict never changes the exit code (section 3c stays the only exit rule). `run.json` stores the counters (`invariants`) and the setup (`cover`); an older run.json is rescanned from the `bot.log` next to it, and a `supervisor.log` next to a log adds launches and the bot's exit. Definitions, thresholds and provenance: docs/geninv.md; contract: soak-gates delta of the card.

## 4. What d2d cannot validate

- Wine, input injection (SendInput/mouse), `/proc/pid/mem` reads, Game.exe crash/hang, the failsafe ladder and recycle paths, window focus, the 20-minute save point (d2d runs with `--no-save`), OOM/heap guards of the real process mix: all stay original-game soak topics (G1/G2).
- Known d2d vs original gaps (docs/d2d-fidelity.md): Tome of Town Portal 250 gold on d2d vs 450 live, sell price ratio, cast period 14 ticks for every spell (Lightning/Chain Lightning -26 %, bolts +8 %), potion double-fill absent, Den XP +60 %, Blood Moor +20 % at clvl 4-6, wake radius only checked against d2d's own source. They bias XP/h, gold/h and potions/h; a gap that matches one of them is explained, not a bug. `docs/d2d-gaps.md` (D2DTOGGLE) will hold the maintained list once merged (not in this tree yet).
- Statistics from d2d are only as good as d2d's fidelity: its drop engine was diffed against game.exe by its authors (**D2D-src**, not re-checked), nothing here was measured on the original except the baselines.

## 5. Dead ends and limits of this tool

- Wall-clock timestamps of the bot are meaningless under lockstep; do not pass `-hours` from them. The tool uses `game_ms`.
- The `clock` read happens after the command exits; if the bot crashes mid-run the clock still reports how far d2d got, and `End` says `cmd exited`/`wall limit`: such a run is shorter than requested (warned by the 2 h minimum).
- Sample of the fake mode (`-fake`): toy model scaled from attempt 18, deterministic per seed. It exists to test the pipeline; never quote its numbers.
- Live use so far (V, 2026-10-30 09:11, `tools/d2dsoak/run.sh dry 0.05`): lock taken, d2d started niced with `--no-save` and the toggles, 3 virtual minutes stepped (4500 ticks, game_ms 180000, clock.jsonl as expected, 4 s wall including start-up), `quit`, `pgrep -x d2d` empty afterwards, lock released. The tool correctly refused earlier while another agent held `/tmp/d2d-lab.lock`. A real bot run waits for D2DBACKEND/COMBAT/TOWN.
- Pacer note (review 2026-10-30, `internal/d2dinput/pacer.go`, Game.exe-adjacent: **needs a G1/G2 check**): the detloop guard (`Pacer.Watch`) runs at most once per Sleep chunk (50 ticks = 2 virtual s), not every `WatchMs` = 50 ms, so death/safety/stuck reactions under lockstep can come up to 2 virtual s later than on Game.exe. Not changed: lowering `Chunk` to the watch interval means 1 tick per step call (about 50 times more socket calls, **U** cost, it would also change the soak pace). Suggested fix if wanted: chunk = min(Chunk, ticks of `every`) only while a watcher is registered, then re-baseline deaths/h. Until then the d2d soak deaths/h is measured with a coarser guard than the original-game baseline.
