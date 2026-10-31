## ADDED Requirements

### Requirement: A/B protocol for bot changes
A claim that a bot change moves XP per hour or kills per hour on d2d SHALL come from a paired same-seed before/after on one binary, at least 8 seeds per arm and 2 virtual hours per seed, and the effect is claimed only when the 95 percent interval of the paired mean difference excludes zero; anything less is reported as `no effect shown`. Source: docs/d2d-xp-gap.md (xpgap 2026-10-31: SE 7 to 9 percent with 4 seeds per arm); card m2-abdesign.

#### Scenario: Four seeds per arm
- **WHEN** a card reports +8 percent XP/h from 4 seeds per arm
- **THEN** the report says `no effect shown` and names the seeds per arm the SD requires

#### Scenario: Paired result
- **WHEN** 8 paired seeds give a mean difference of +600 XP/h with a 95 percent interval of +150 to +1050
- **THEN** the change is reported as an improvement of about 12 percent
