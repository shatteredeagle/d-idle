## MODIFIED Requirements

### Requirement: Twenty percent tolerance
The d2d mean of XP per hour and kills per hour SHALL be within `max(20 % of the baseline mean, abs)` of the original-game baseline. Deaths per hour and deaths per 1000 kills are no longer judged by a relative tolerance: they follow the requirements "Death budget and Poisson bound" and "Proxy metrics" below. A gap above 20 percent that is not explained by a known gap is a fidelity bug. Source: user decision 2026-10-30; plan-v2 section 11 (V, the XP and kill rule is tested on real logs).

#### Scenario: XP gap
- **WHEN** the baseline mean is 10.9k XP/h and the d2d mean is 13.5k XP/h
- **THEN** the gap is +24 percent and the metric is out of tolerance

## ADDED Requirements

### Requirement: Death budget
Under gate v2 the pooled d2d runs SHALL show at most one death per full 6 hours of d2d exposure, and at least one 6 hour block is needed for PASS; less exposure is INCONCLUSIVE unless a rule already fails. Source: user decision 2026-10-30.

#### Scenario: Two deaths in 6 hours
- **WHEN** the d2d runs total 6 virtual hours and 2 deaths
- **THEN** the budget of 1 is exceeded and the death verdict is FAIL

#### Scenario: Short exposure
- **WHEN** the d2d runs total 4 virtual hours and no death
- **THEN** the death verdict is INCONCLUSIVE with the hours still needed

### Requirement: Death rate cap and Poisson bound
Under gate v2 the exact one-sided 95 percent lower bound of the pooled d2d death rate MUST NOT exceed 0.15 deaths/h, and the exact Poisson test of the pooled count against the baseline death rate MUST NOT reject at the 5 percent level; a zero-death baseline counts as 0.5 deaths for the null rate. Deaths per 1000 kills uses the same test with kills as exposure. Source: user decision 2026-10-30 (cap), Poisson form is the tool (V, internal/d2dsoakstat/gatev2_test.go).

#### Scenario: Point rate above the cap
- **WHEN** there is 1 death in 6 virtual hours (0.167/h, lower bound far below 0.15)
- **THEN** the verdict is PASS with a note that more hours would settle the cap

### Requirement: Proxy metrics
Under gate v2 potions used per 100 kills, minimum life fraction per fight and damage taken per kill SHALL each be within 25 percent of the baseline mean, with the same PASS, FAIL and INCONCLUSIVE classes as the 20 percent metrics. Source: user decision 2026-10-30.

#### Scenario: Potion use
- **WHEN** the baseline uses 4 potions per 100 kills and d2d uses 5.5
- **THEN** the gap is +37 percent and the proxy fails

### Requirement: Proxy definitions
Potions per 100 kills MUST be the drunk-counter increments over the kill increments times 100 (V: `potionsUsed` and `kills` are in the minute line). Minimum life fraction is the mean over minutes with a kill of the lowest life percent in that minute divided by 100, and damage per kill is the `dmgTaken` counter increments over the kill increments (U: the bot does not emit `dmgTaken` and `hpMinPct` yet). Source: user decision 2026-10-30.

#### Scenario: Definition check
- **WHEN** a run has 20 kills, 3 potions drunk and 300 life lost
- **THEN** potions per 100 kills is 15 and damage per kill is 15

### Requirement: Missing proxy data
A proxy missing from the d2d runs or the baseline MUST give INCONCLUSIVE with a note, so the overall verdict cannot be PASS on missing data.

#### Scenario: Proxy not emitted
- **WHEN** the minute lines carry no `dmgTaken` and `hpMinPct`
- **THEN** damage per kill and minimum life fraction are INCONCLUSIVE ("not emitted") and the overall verdict is at best INCONCLUSIVE
