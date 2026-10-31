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

### Requirement: Champion group is avoided at low level
A Target SHALL carry `Champion` for a flag byte with the champion bit and not the unique bit (0x0c; Elite true, Boss false); three or more champions in view below character level 8 MUST all be marked Avoid. Source: card champ; internal/combat/target.go (championGroupMin 3, championGroupLevel 8); tests TestChampionIsEliteNotBoss, TestChampionGroupAvoidedAtLowLevel. SME rule of thumb, not verified live.

#### Scenario: Three champions at clvl 5
- **WHEN** three champions are in view and the character is level 5
- **THEN** all three are marked Avoid

#### Scenario: Two champions or clvl 8
- **WHEN** only two champions are in view at clvl 5, or three at clvl 8
- **THEN** none is marked Avoid

### Requirement: Champion caution raises the potion lines
With a non-boss champion among the hostiles in range, in Normal and below character level 15 (level known), the healing line (55 percent) and the boss-rejuvenation line (50 percent) SHALL rise by knob `combat.championEarlyPct` (default 10 pp, range 0 to 25, 0 = off); otherwise they are unchanged. Source: card champ; docs/knobs.md; internal/resource/potion.go (championBonus); test TestChampionEarlyTightensDrinkLines. EXP, not verified live.

#### Scenario: Healing line against a champion
- **WHEN** a clvl 10 Normal character at 62 percent life has a champion in range
- **THEN** a life potion is drunk (line 65), and with the knob at 0 none is

#### Scenario: Rejuvenation line against a champion
- **WHEN** a clvl 10 Normal character at 58 percent life holds a rejuvenation potion with a champion in range
- **THEN** the rejuvenation potion is drunk (line 60)

#### Scenario: Out of scope
- **WHEN** the character is clvl 15, or the difficulty is Nightmare, or no champion is in range, at 62 percent life
- **THEN** no potion is drunk

### Requirement: Champion death is labelled as a champion
When a champion kills the character the death record SHALL set `killerChampion` true, MUST NOT set `killerBoss` with it (a unique or superunique killer sets killerBoss and not killerChampion), and a champion killer adds no weight to the danger table even when the death starts an area ban. Source: card champ; internal/bot/observe.go and death.go; test TestDeathLabelsChampionNotBoss.

#### Scenario: Champion kill
- **WHEN** the killer's flag byte is the champion bit
- **THEN** the death record has killerChampion true and killerBoss false

#### Scenario: Unique kill
- **WHEN** the killer's flag byte is the boss bit
- **THEN** the death record has killerBoss true and killerChampion false

#### Scenario: Death spiral by a champion
- **WHEN** deaths in one area trigger the spiral ban and the killer was a champion
- **THEN** the area is banned and the danger table gains no entry for the killer's plain kind
