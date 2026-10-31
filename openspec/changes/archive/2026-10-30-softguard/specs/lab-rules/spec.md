## MODIFIED Requirements

### Requirement: Safety limits for lab runs
Lab runs SHALL be niced (10), refuse above load 45 or under 15 GB free disk, and leave no d2d process behind. New live work MUST also pass the floating capacity rule: `tools/capacity.sh --add <cores>` must not print `NO` (predicted idle+nice >= 25 %, iowait < 40 %, 5 minute average); there is no fixed Game.exe or d2d count, and offline and bug-fix work is not capped. Source: docs/d2d-soak.md section 1 (V); docs/process/worker-capacity.md.

#### Scenario: Run ends
- **WHEN** a seed finishes or its wall limit is hit
- **THEN** d2d is quit, the process group is killed and `pgrep -x d2d` is empty

#### Scenario: Quiet machine, third instance
- **WHEN** two Game.exe are running, the 5 minute average idle+nice is 60 % and a d2d run costing 0.9 cores is requested on 16 cores
- **THEN** `tools/capacity.sh --add 0.9` prints OK and the run starts without asking anyone

#### Scenario: Busy machine
- **WHEN** the predicted idle+nice after the added cost is under 25 %
- **THEN** the verdict is NO and the live work waits or is made smaller

## ADDED Requirements

### Requirement: Big run warning
A BIG RUN (more than one new game or d2d instance at once, over 2 cores for over 30 minutes, or any multi-hour soak) SHALL be announced by the coordinator in chat and MUST start after 5 minutes unless the user replied hold or no; silence means go. The dispatch agent re-checks `tools/capacity.sh` before starting and aborts only on NO. Other work SHALL NOT wait for the user. Source: user decision 2026-10-30.

#### Scenario: Silence
- **WHEN** the warning was sent and 5 minutes pass without a reply
- **THEN** the run starts, after a final capacity check that is not NO

#### Scenario: Hold
- **WHEN** the user replies "hold" within 5 minutes
- **THEN** the run does not start until the user says go

#### Scenario: Small run
- **WHEN** a single d2d run of 0.9 cores for 20 minutes is requested
- **THEN** it is not a big run, no warning is sent and only the capacity check applies
