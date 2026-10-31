# Soak gates specification

## Purpose
Numeric pass criteria for the G1 (2 h), G2 (8 h) and 24 h soaks and the health guard that decides when a soak may start, so a run is judged by numbers and not by opinion. Evidence: docs/review-strategy.md (G1 criteria), docs/soak24.md section 3 (profile m24), docs/g1-log.md (attempt log), docs/plan/plan-v2.md section 5.

## Requirements

### Requirement: G1 stability
A G1 run (2 h, original game) MUST show zero panics, zero manual interventions, d2bot and the game alive at 2 h, and a saved XP equal to the character XP at a graceful stop. Source: docs/review-strategy.md G1 criterion 1 (S, calibrated on attempts 6 to 18).

#### Scenario: Clean two hour run
- **WHEN** a G1 run ends at 2 h
- **THEN** the report shows 0 panics, no process exit other than planned ones, and `saveXp == charXp`

### Requirement: G1 progression
A G1 run SHALL earn true saved XP per hour at or above the baseline for the same clvl (Soaky: 6.6k XP/h over clvl 1 to 7; attempt 18 measured 9.1k XP/h) and MUST NOT idle in one state for more than 10 minutes without progress. Source: docs/review-strategy.md criterion 3, docs/g1-log.md attempt 18.

#### Scenario: XP below baseline
- **WHEN** true XP/h is under the baseline at the same clvl
- **THEN** the run is a FAIL even without a crash

### Requirement: G1 belt
A G1 run SHALL have no window longer than 5 minutes with zero healing potions while outside town, and at least one town portal bought and used. Source: docs/review-strategy.md criterion 4.

#### Scenario: Empty belt in the field
- **WHEN** the belt holds 0 healing potions outside town for 5 minutes
- **THEN** the run is a FAIL

### Requirement: G1 kill recovery
A G1 run SHALL recover from an injected Game.exe kill within 100 seconds with exactly one restart counted and no orphaned winedevice. Source: docs/g1-log.md attempt 8 (V, last exercised), docs/BACKLOG.md item 1 (injection permission blocked on the user).

#### Scenario: Injected kill
- **WHEN** the soak's own Game.exe is killed
- **THEN** the bot is back in game in at most 100 s, `restarts` rises by exactly 1 and no orphan winedevice remains

### Requirement: G2 meaning and floors
The 8 h profile m24 run SHALL be called a stability and economy soak, not a progression soak, until Andariel and Act 2 work; it SHALL reach clvl 16, attempt Andariel at clvl 14 or higher, hire the merc and keep gold from staying flat. Source: docs/plan/plan-v2.md section 5 and section 10 (approved 2026-10-30).

#### Scenario: Cheated run
- **WHEN** a run used cheated gold or cheated levels
- **THEN** it does not count toward any gate

### Requirement: Soak progress floors
The m24 profile MUST fail a run whose average XP is below 1000 per hour, whose XP does not change for more than 10 minutes (idle and paused exempt), whose no-gain window (no clvl, quest step or gold gain) exceeds 90 minutes, or whose bot health is down for more than 3 minutes at a time. Source: docs/soak24.md section 3 (U until the first 24 h run calibrates it).

#### Scenario: Idling bot
- **WHEN** no XP change is seen for 11 minutes outside pause
- **THEN** d2soak reports a stall FAIL

### Requirement: Resource slopes
A soak MUST keep helper, game and bot fd growth at or below 5 per hour after a 10 minute warm-up, RSS growth at or below 40 MB per hour, threads and goroutines at or below 5 per hour, and ladder rungs and sentinel actions at or below 2 per hour each. Source: docs/soak24.md section 3.

#### Scenario: Leaking helper
- **WHEN** the fd slope exceeds 5 per hour after warm-up
- **THEN** the run fails the fd criterion

### Requirement: Heap runaway alert
The soak harness SHALL raise a critical alert when the bot RSS exceeds 8192 MB or rises by more than 1024 MB within 5 minutes. Source: docs/soak24.md section 4 (V offline, live U); attempt 16 grew to 48.8 GB before the fix.

#### Scenario: Runaway
- **WHEN** the bot RSS jumps 1.2 GB in 4 minutes
- **THEN** a `heap_runaway` critical alert is written

### Requirement: Start guard
A soak start MUST be refused when free disk is under 10 GB, when the 1-minute load exceeds the backstop 28, or when `tools/capacity.sh --add <soak cost>` prints `NO`; there is no fixed Game.exe cap. A multi-hour soak is also a big run (lab-rules spec). Source: docs/soak24.md section 2; docs/process/worker-capacity.md (code change to soak24.sh is a follow-up, S).

#### Scenario: Busy host
- **WHEN** the predicted idle+nice after adding the soak is under 25 %
- **THEN** `soak24.sh start` refuses with the capacity.sh output

#### Scenario: Many instances, quiet host
- **WHEN** three Game.exe already run but the predicted idle+nice stays at or above 25 % with iowait under 40 %
- **THEN** the start is allowed

### Requirement: Verdict
A verdict MUST be PASS only when `report.txt` says PASS with exit 0; NO-DATA (exit 3) is never a pass, and the death criteria follow the death-rule spec. Source: docs/soak24.md section 3.

#### Scenario: No data
- **WHEN** a run produced no samples
- **THEN** the verdict is NO-DATA, exit 3, not a pass

### Requirement: Per-seed invariants
`d2dsoak` SHALL report for every seed counters and a verdict for HANG (a pace interval without virtual advance, or `combat no progress, holding`), STALL (any `campaign stalled` rung), WEDGE (one `no route from level` or `no eligible reachable leveling area` storm longer than 15 virtual min with no non-town Entered), PROGRESS (no kill, xp, area, quest or spend for more than 30 min), DEATHS, RESTARTS and EXIT. Source: card gen-invariants; docs/geninv.md.

#### Scenario: Town wedge
- **WHEN** a log has 60 virtual minutes of `no eligible reachable leveling area` lines and no non-town Entered
- **THEN** the seed is VIOLATED with WEDGE and PROGRESS even though the process exited 0 at the time limit

#### Scenario: Frozen clock
- **WHEN** two pace lines are wall time apart and show the same virtual time
- **THEN** the seed is VIOLATED with HANG

### Requirement: Death and exit invariants
A seed SHALL violate DEATHS on more than 15 deaths per 1000 kills from 150 kills, on 3 deaths in 30 min without a clvl gain or 5 in 60 min, or on a death with no progress within 10 virtual min; RESTARTS on any bot restart; EXIT on an end under 95 percent of the requested hours, a non-zero exit or a recovered panic. Source: death-rule FAIL-1 and FAIL-3 (D); soakfix early end 95 percent (D); recovery window (J).

#### Scenario: Early end
- **WHEN** 3 virtual hours were requested and the command exited after 1
- **THEN** the seed is VIOLATED with EXIT; a log with no end marker and no requested length is `open`, not violated

### Requirement: Invariants verdict
The report SHALL end its invariants block with `INVARIANTS: PASS`, `VIOLATED(list)` or `NO-DATA`, separate from the fidelity verdict; the set is VIOLATED when any seed is, PASS only when every seed is PASS. The invariants verdict MUST NOT change the exit code of `run` or `compare`. Source: card gen-invariants; fidelity-gate Verdict exit codes.

#### Scenario: Violated but fidelity PASS
- **WHEN** the fidelity verdict is PASS and one seed has a WEDGE
- **THEN** the exit code is 0, the text and `-json` show `INVARIANTS: VIOLATED(seed N: WEDGE)` and OVERALL stays the last line

### Requirement: Holdout seeds
Seeds 16 to 23 SHALL be holdout seeds, never used for tuning, fitting or diagnosis; the report groups seeds as `tuning` and `holdout` (`-holdout` overrides, empty disables), gives each group its counters and invariants verdict, and prints a COVERAGE line naming seeds, class, act, difficulty, engine and clvl range. Source: docs/research/bot-generality.md sections 3 and 5 (experiment 1); docs/geninv.md.

#### Scenario: Gate run with holdout
- **WHEN** seeds 3 and 16 to 17 are compared
- **THEN** group tuning lists seed 3, group holdout lists seeds 16 to 17, each with its own verdict, and the COVERAGE line lists both seed sets

#### Scenario: No holdout ran
- **WHEN** only seeds 1 to 8 ran
- **THEN** the report says `group holdout: none ran` and the COVERAGE line shows holdout none
