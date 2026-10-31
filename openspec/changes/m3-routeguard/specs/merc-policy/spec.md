## MODIFIED Requirements

### Requirement: Route guard
The Monastery road (areas 7, 26 to 37, so Andariel too) and the Forgotten Tower (areas 20 to 25) MUST be entered only at clvl 16 or higher with no merc, or 14 or higher with a living merc (`route.monasteryMinLevel`, `route.towerMinLevel`, with-merc knobs, `route.towerNeedsMerc` 0), plus the supply kit on every guard; knobs at 0 turn a guard off; unknown merc counts as none. Interim. Source: ADR 0011, docs/knobs.md.

#### Scenario: No merc at clvl 14
- **WHEN** a clvl 14 caster without a merc picks a leveling area
- **THEN** the Monastery road and the Forgotten Tower are excluded and the planner levels in the Blood Moor to Black Marsh, caves and Crypt instead

#### Scenario: Living merc at clvl 14
- **WHEN** a clvl 14 caster with a living merc and the full supply kit picks a leveling area
- **THEN** the Monastery road and the Forgotten Tower are allowed

#### Scenario: Thin kit with a merc
- **WHEN** a clvl 15 caster with a living merc holds 2 healing potions
- **THEN** the Monastery road is excluded until the kit is refilled

#### Scenario: Road floor trigger
- **WHEN** Monastery road deaths in the act1run soak fail the death rule at the clvl 16 floor
- **THEN** the floor is raised to 18 and the decision goes back to a grill, and if the soak shows no road deaths 14 is tried

### Requirement: Andariel floor
The Normal campaign SHALL attempt Andariel from clvl `boss.andarielMinLevel` 17 with no merc and clvl 15 with a living merc, with the kit gate on top plus at least 2 antidote potions; the former +2 for a dead merc is replaced by the no-merc floor. No unattended attempt before the throwaway fight of card m3-andariel has won once. Source: ADR 0011 (grill 0003, 2026-10-31), docs/knobs.md, internal/merc `MercAliveGate`.

#### Scenario: Floor read from the knob
- **WHEN** an agent needs the Andariel level
- **THEN** it reads docs/knobs.md or the knob, not a number copied from a doc

#### Scenario: Dead merc
- **WHEN** a clvl 16 character has no living merc and enough gold to revive
- **THEN** Andariel is not attempted until clvl 17 or the merc is revived and clvl 15 is met

#### Scenario: No antidotes
- **WHEN** a clvl 17 character holds a full kit but 1 antidote potion
- **THEN** the Andariel attempt does not start and the bot restocks
