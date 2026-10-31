## MODIFIED Requirements

### Requirement: Start guard
A soak start MUST be refused when free disk is under 10 GB, when the 1-minute load exceeds the backstop 28, or when `tools/capacity.sh --add <soak cost>` prints `NO`; there is no fixed Game.exe cap. A multi-hour soak is also a big run (lab-rules spec). Source: docs/soak24.md section 2; docs/process/worker-capacity.md (code change to soak24.sh is a follow-up, S).

#### Scenario: Busy host
- **WHEN** the predicted idle+nice after adding the soak is under 25 %
- **THEN** `soak24.sh start` refuses with the capacity.sh output

#### Scenario: Many instances, quiet host
- **WHEN** three Game.exe already run but the predicted idle+nice stays at or above 25 % with iowait under 40 %
- **THEN** the start is allowed
