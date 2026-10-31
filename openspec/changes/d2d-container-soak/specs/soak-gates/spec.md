## ADDED Requirements

### Requirement: d2d container one-slot 24 h soak
The d2d container SHALL pass a one-slot 24 h soak with restart and kill injection, judged by the same PASS rule as other soaks (report.txt PASS, exit 0), before Game.exe retires from the dev and product path. Source: ADR 0009 item 5; docs/d2d-container/risks.md step 10.

#### Scenario: Passing soak
- **WHEN** the one-slot 24 h container soak ends with report PASS and exit 0
- **THEN** the retirement trigger for Game.exe in the dev and product path fires

#### Scenario: No data
- **WHEN** the run produced no samples
- **THEN** the verdict is NO-DATA, not a pass
