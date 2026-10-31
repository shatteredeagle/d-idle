## ADDED Requirements

### Requirement: Verdict exit codes
`d2dsoak run` and `d2dsoak compare` MUST exit 0 only when the overall verdict is PASS, 2 on INCONCLUSIVE and 3 on FAIL or NO-DATA, so a script cannot read a non-pass as a pass. Source: docs/research/code-review-2026-10-30.md H1; openspec/specs/soak-gates (NO-DATA is exit 3); card m2-soakexit.

#### Scenario: Inconclusive verdict
- **WHEN** 3 seeds ran and the verdict is INCONCLUSIVE
- **THEN** the process exits 2 and the last line names the verdict and the code

#### Scenario: No usable seed
- **WHEN** every seed is excluded (no minute lines)
- **THEN** the verdict is NO-DATA and the process exits 3, never 0 or 2

#### Scenario: Failed metric
- **WHEN** any gated metric is FAIL
- **THEN** the process exits 3

### Requirement: Excluded seeds
A seed with zero minute lines, or whose d2d process ended before the requested virtual hours, SHALL contribute zero exposure hours and zero kills, SHALL be listed under `excluded seeds` with the reason, and SHALL NOT count toward the minimum-seed rule. Source: docs/research/code-review-2026-10-30.md H1; card m2-soakexit.

#### Scenario: Crashed seed
- **WHEN** seed 11 of 8 hangs and produces 1.8 virtual hours of a 2 hour run
- **THEN** the report lists seed 11 as excluded, the death metrics use the other seeds' hours and the seed count is 7
