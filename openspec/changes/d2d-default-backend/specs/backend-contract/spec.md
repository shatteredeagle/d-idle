## MODIFIED Requirements

### Requirement: Original game remains the acceptance target
d2d SHALL be the primary backend for development, soak and the container; results verified only on d2d MUST be tagged V-d2d, and original-game confirmation is required only for the replicated-soak gate metrics and for probes whose answer changes bot decision code (merc rule, death rule, a price the planner uses). Source: ADR 0009 (accepted 2026-10-30) items 1 to 4; docs/decisions.md 2026-10-30.

#### Scenario: Probe result
- **WHEN** a d2d probe says the free Rogue is granted without a window and the answer changes the merc rule
- **THEN** the doc tags it V-d2d and the original-game confirmation stays an open item until original-game work resumes

#### Scenario: Cosmetic probe
- **WHEN** a probe answer changes no decision code
- **THEN** it is answered on d2d only

## ADDED Requirements

### Requirement: Game.exe is frozen
The bot SHALL keep the Game.exe backend working but MUST NOT gain new Game.exe-only features or new Game.exe memory-offset work until the d2d container passes its one-slot 24 h soak, and the code SHALL stay in the repo for a possible v2. Source: ADR 0009 item 5 and addendum (user, 2026-10-30: original-game work is much further down the line).

#### Scenario: New feature request
- **WHEN** a change needs a new Game.exe memory offset
- **THEN** it is refused or deferred and the feature is built on d2d

### Requirement: Leaving d2d is break glass
Leaving d2d SHALL be a user decision, never automatic, and MUST be considered only when a d2d engine gap needs a rewrite (more than about 16 agent-hours for one gap) and that gap moves a gated fidelity metric past 20 percent; container build failures are tasks, never an exit trigger. Source: ADR 0009 exit trigger.

#### Scenario: Toolchain failure
- **WHEN** the container build fails on the C++26 toolchain
- **THEN** it is handled as a task (other compiler or base image), not as a reason to leave d2d
