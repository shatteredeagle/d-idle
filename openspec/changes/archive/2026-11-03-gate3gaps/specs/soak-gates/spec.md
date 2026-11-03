## MODIFIED Requirements

### Requirement: Resource slopes
A soak MUST keep helper, game and bot fd growth at or below 5 per hour after a 10 minute warm-up, RSS growth at or below 40 MB per hour, threads and goroutines at or below 5 per hour, rungs and sentinel actions at or below 2 per hour. On d2d the runner SHALL sample fd, thread and RSS of d2d and bot each wall minute into `mem.csv`; the judge FAILS a series whose fitted slope exceeds its limit and reads under 30 samples as NO-DATA. Source: docs/soak24.md section 3; gate3gaps.md section 1.

#### Scenario: Leaking helper
- **WHEN** the fd slope exceeds 5 per hour after warm-up
- **THEN** the run fails the fd criterion

#### Scenario: d2d fd leak on the gate runner
- **WHEN** `mem.csv` shows the d2d fd count rising from 400 to 470 over 8 h after warm-up
- **THEN** the slope is about 9 per hour and the run FAILS the d2d fd criterion

#### Scenario: Bot RSS slope within the limit
- **WHEN** the bot RSS rises 120 MB over 8 h after warm-up
- **THEN** the bot RSS row passes at 15 MB per hour

#### Scenario: No samples for a series
- **WHEN** `mem.csv` lacks the bot columns
- **THEN** the bot rows read NO-DATA and a Gate 3 run is not a pass

#### Scenario: Restart inside the series
- **WHEN** the d2d pid changes
- **THEN** each segment is fitted after its own 10 minute warm-up and the worst slope counts

### Requirement: Gate 2 and Gate 3 on d2d
Gate 2 (8 h) and Gate 3 (24 h) SHALL be soaks of profile m24 on a hidden d2d slot, judged by the Gate 1 criteria over their length, the m24 limits and floors, the resource slopes and `d2dsoak report` ending with `INVARIANTS: PASS`. Gate 3 also records the memory curve, restart interval and log size. A d2d exit is planned only as an injected fault or after a `planned recycle` supervisor line; others fail stability. Source: docs/review-strategy.md; gate3gaps.md section 2.

#### Scenario: Gate 2 with a violated invariant
- **WHEN** the Gate 2 report ends with `INVARIANTS: VIOLATED(...)`
- **THEN** the run is not a pass even if the m24 limits hold

#### Scenario: Planned recycle
- **WHEN** supervisor.log shows `planned recycle` before a d2d exit and the bot is in game within 3 wall minutes
- **THEN** the exit is not unplanned and the interval is recorded

#### Scenario: Unplanned d2d exit
- **WHEN** d2d exits with no injected fault and no planned recycle
- **THEN** the stability row FAILS

## ADDED Requirements

### Requirement: Full bag and disk guard
A d2d soak SHALL FAIL when `invPct` stays above 95 outside town for more than 60 consecutive minute lines (-1 is unknown and never counts, U threshold). The runner MUST stop with an alert and the verdict NO-DATA when free disk falls under 5 GB, and total logs of a 24 h run SHOULD stay under 20 GB (U). Source: docs/research/gate3gaps.md sections 2.3 and 2.4.

#### Scenario: Full bag
- **WHEN** 61 consecutive minute lines outside town show `invPct` above 95
- **THEN** the inventory criterion FAILS

#### Scenario: Disk falling
- **WHEN** free disk drops under 5 GB at hour 15
- **THEN** the runner stops with an alert and the verdict is NO-DATA

### Requirement: Gate 0.5 smoke
Before Gate 2 or Gate 3 a smoke run SHALL pass on the same build: profile m24, 45 wall minutes on a hidden d2d slot (about 9 virtual hours), one injected fault at 15 wall minutes. It is judged by BELT, NOKILL, PROGRESS, WEDGE, HANG, the death rule, fault recovery and the presence of resource samples, not by XP floors or slopes, and never counts as a ladder gate. Source: docs/research/gate3gaps.md section 3 (U thresholds).

#### Scenario: Belt failure caught early
- **WHEN** the smoke run shows 6 minute lines outside town without a healing potion
- **THEN** Gate 0.5 FAILS and no Gate 2 starts on that build

#### Scenario: Slow clock
- **WHEN** the run reached only 4 virtual hours in 45 wall minutes
- **THEN** the verdict is NO-DATA, not a pass

#### Scenario: Smoke passes
- **WHEN** all criteria of the subset hold
- **THEN** the build may start Gate 2, which must still pass on its own
