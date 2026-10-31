# Combat loop specification

## Purpose
What the fight loop of internal/combat must keep true so that it never spins without game time passing: the cost of a failed walk, when a retreat is offered, and the no-progress watchdog.

## ADDED Requirements

### Requirement: A failed walk costs one poll tick
When a walk through the Mover fails (no path, stuck) and the run is not cancelled, the fighter SHALL log `move failed` and sleep one poll tick (the `PollMs` knob) before returning, and the failure MUST NOT end the fight. Under a lockstep clock time only moves on sleeps and steps, and `nav: no path: empty path` returns at once. Source: card navstall; internal/combat/fight.go (Fighter.move); test TestFailedMoveAdvancesTime.

#### Scenario: Ten failed walks
- **WHEN** ten walks fail at once with `nav: no path: empty path`
- **THEN** the game clock has advanced by at least ten poll ticks and no error is returned

#### Scenario: Crowd with every walk failing
- **WHEN** a crowd fight runs with life low, no portal, no potion and a Mover whose every walk fails at once
- **THEN** fewer than 500 failed walks happen before the fight moves game time on (TestCrowdRetreatEmptyPathDoesNotSpin)

### Requirement: No retreat while pinned
Decide SHALL NOT be offered a crowd step-away (canRetreat is false) while the fighter is pinned (evading is off for the 8 s pinHold after two consecutive evasive moves ended within 2 subtiles of their start) or when it has no Mover, so the crowd is fought in place. Source: card navstall; internal/combat/crowd.go and fight.go; test TestCanRetreatOffWhilePinned.

#### Scenario: Not pinned
- **WHEN** ten monsters stand close and the fighter is not pinned
- **THEN** canRetreat is true because a step away exists

#### Scenario: Pinned
- **WHEN** the same crowd stands there and noEvadeUntil lies pinHold in the future
- **THEN** canRetreat is false

### Requirement: Fight-loop no-progress watchdog
When 40 consecutive fight-loop iterations pass without the clock moving, the loop SHALL force one poll sleep, reset its count and log `combat no progress, holding` at most once per fight. Source: card navstall; internal/combat/fight.go (noProgressMax = 40); test TestFightWatchdogLogs.

#### Scenario: Frozen clock
- **WHEN** the clock never moves, every walk fails and the crowd is not cleared
- **THEN** the loop reaches at least 400 sleeps on its own (the test's bound) and the log holds exactly one `combat no progress, holding` line

#### Scenario: Clock moves
- **WHEN** the clock moves between iterations
- **THEN** the count restarts at 0 and no watchdog sleep is forced
