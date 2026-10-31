## ADDED Requirements

### Requirement: A/B protocol for bot changes
A claim that a bot change moves XP per hour or kills per hour on d2d SHALL come from a paired same-seed before/after on one binary, at least 8 seeds per arm and 2 virtual hours per seed, and the effect is claimed only when the 95 percent interval of the paired mean difference excludes zero; anything less is reported as `no effect shown`. Source: docs/ab-design.md. Source: docs/d2d-xp-gap.md (xpgap 2026-10-31: SE 7 to 9 percent with 4 seeds per arm); card m2-abdesign.

#### Scenario: Four seeds per arm
- **WHEN** a card reports +8 percent XP/h from 4 seeds per arm
- **THEN** the report says `no effect shown` and names the seeds per arm the SD requires

#### Scenario: Paired result
- **WHEN** 8 paired seeds give a mean difference of +600 XP/h with a 95 percent interval of +150 to +1050
- **THEN** the change is reported as an improvement of about 12 percent

### Requirement: A/B sample size and interleaving
The arms of an A/B SHALL be interleaved per seed, and the seeds per arm for a planned relative effect e SHALL be at least ((1.96 + 0.84) * SD_diff / (e * mean_before))^2 from the measured SD of the per-seed differences, never below 8. Source: docs/ab-design.md; tools/ab/paired.py.

#### Scenario: Sample size from the measured SD
- **WHEN** a pilot gives an SD of the per-seed XP/h differences of 1052 on a mean of 5273 and the planned effect is 10 percent
- **THEN** the plan names about 34 seeds per arm, not 8
