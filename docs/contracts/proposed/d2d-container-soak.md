---
id: d2d-container-soak
goal: G-C
status: proposed
enforced_by: prose
---
# Proposed contract text: d2d-container-soak (soak-gates)

> status: PROPOSED, not in force. Converted at OpenSpec removal stage 3 from the delta spec of card `d2d-container-soak` (git tag `openspec-final`, path `openspec/changes/d2d-container-soak/specs/soak-gates/spec.md`). When the work lands, the branch that implements it folds this text into `docs/contracts/soak-gates.md` and deletes this file. Backlog task: title starts with `d2d-container-soak:`.

## Requirements (new)

### Requirement: d2d container one-slot 24 h soak
The d2d container SHALL pass a one-slot 24 h soak with restart and kill injection, judged by the same PASS rule as other soaks (report.txt PASS, exit 0), before Game.exe retires from the dev and product path. Source: ADR 0009 item 5; docs/d2d-container/risks.md step 10.

#### Scenario: Passing soak
- **WHEN** the one-slot 24 h container soak ends with report PASS and exit 0
- **THEN** the retirement trigger for Game.exe in the dev and product path fires

#### Scenario: No data
- **WHEN** the run produced no samples
- **THEN** the verdict is NO-DATA, not a pass
