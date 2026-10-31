## ADDED Requirements

### Requirement: The d2d link wait outlives the verb timeout
The d2d world's update bound SHALL be Options.Timeout (per-verb budget, default 10 s) plus Options.LinkWait, so an arrival wait for neighbour levels is not cut short by the timeout; before this nav reported "no exit" while the neighbours were still being linked. Source: card d2dexit; docs/d2d-exit.md (V by failing test); internal/d2dworld/world.go (ctx); test TestLinkWaitOutlivesTimeout.

#### Scenario: Neighbour appears late
- **WHEN** Timeout is 50 ms, LinkWait 5 s and the neighbour appears after 8 polls
- **THEN** pending ends at 0 and the level has at least 2 adjacent entries

### Requirement: The d2d backend waits for links by default
The d2d backend SHALL use a link wait of 90 s (d2denv.DefaultLinkWait) when Config.LinkWait is 0. Stepped lockstep games link neighbours only while stepped, so the wait advances the game. Source: card d2dexit; internal/d2denv/env.go. UNVERIFIED by test (live: Blood Moor reached in about 32 s, docs/d2d-exit.md).

#### Scenario: No link wait configured
- **WHEN** the backend is built with LinkWait 0
- **THEN** the world is created with LinkWait 90 s (UNVERIFIED, no test)

### Requirement: A missing layout verb is a warning
When the d2d binary has no `debug layout` verb (fork patch P3 missing), Backend.Start SHALL log a WARN `d2d has no 'debug layout' verb (fork patch P3 missing): outdoor exits cannot be found, nav will say 'no exit'`, and the arrival line carries the border note `no layout verb: no outdoor borders`. Source: card d2dexit; internal/d2denv/env.go (checkVerbs), internal/d2dworld/world.go (borders). UNVERIFIED by test; seen live with a binary lacking P3.

#### Scenario: Binary without P3
- **WHEN** the layout verb fails at start
- **THEN** the WARN is logged once and the run continues (UNVERIFIED, no test)

### Requirement: Arrival diagnostics are logged
On entering a level the world SHALL log `d2dworld: arrived in level N: <border note> (pending P, linkwait W)` and, when it waited for neighbours, `d2dworld: link wait over after <t>, <n> polls: <note> (pending P, linked [...])`. Source: card d2dexit; internal/d2dworld/world.go (update). UNVERIFIED by test; the text is read from code.

#### Scenario: Arrival with pending neighbours
- **WHEN** the player arrives in a level with pending neighbours and LinkWait above 0
- **THEN** both lines are logged (UNVERIFIED, no test)

### Requirement: The liveness beat grows only with game progress
Beat.Next SHALL return a counter that grows only when game_ms, tick, stepped or frame of `clock`, or the frame of `info`, differs from the last read, and an error only when both verbs fail; it feeds the monitor heartbeat (d2dHealth). Frames counting while lockstep idles is UNVERIFIED. Source: card d2dexit; internal/d2denv/beat.go; tests TestBeatCountsOnlyAdvance, TestBeatLockstepAnyCounter, TestBeatError.

#### Scenario: Wedged but answering
- **WHEN** game_ms rises once and then stays flat
- **THEN** the counter rises once and stays

#### Scenario: Lockstep
- **WHEN** only the info frame advances
- **THEN** the counter rises

#### Scenario: No answer
- **WHEN** clock and info both fail
- **THEN** Next returns the error

### Requirement: Unknown d2d fields are one log line per snapshot
When a snapshot sees fields never reported before, d2dsrc SHALL log ONE line `d2dsrc: N new unknown fields: a, b, ...` (sorted) and nothing further for the same field; reasons stay in Report.Unknown. Source: card d2dexit; internal/d2dsrc/report.go (flush); test TestLogOnce.

#### Scenario: Three reads
- **WHEN** three snapshots with the same unknown fields are read
- **THEN** exactly one summary line is logged
