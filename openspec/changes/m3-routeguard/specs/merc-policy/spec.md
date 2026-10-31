## MODIFIED Requirements

### Requirement: Route guard
The Monastery road (areas 7, 26 to 37, so Andariel too) and the Forgotten Tower (areas 20 to 25) MUST be entered only at clvl 16 or higher with no merc, or 14 or higher with a living merc (`route.monasteryMinLevel*`, `route.towerMinLevel*`, `route.towerNeedsMerc` 0), plus the supply kit on every guard. Each effective floor is logged as `route guard: <gate> floor=<n> (merc=<alive|dead|none>)`. A no-merc knob at 0 turns a guard off; unknown merc is none. Interim. ADR 0011.

#### Scenario: No merc at clvl 14
- **WHEN** a clvl 14 caster without a merc picks a leveling area
- **THEN** the Monastery road and the Forgotten Tower are excluded and the planner levels in the Blood Moor to Black Marsh, caves and Crypt instead

#### Scenario: Living merc at clvl 14
- **WHEN** a clvl 14 caster with a living merc and the full supply kit picks a leveling area
- **THEN** the Monastery road and the Forgotten Tower are allowed

#### Scenario: Thin kit with a merc
- **WHEN** a clvl 15 caster with a living merc holds 2 healing potions
- **THEN** the Monastery road is excluded until the kit is refilled

#### Scenario: Effective floor logged
- **WHEN** the merc state of a clvl 12 caster changes from none to alive
- **THEN** the log shows `route guard: monastery floor=16 (merc=none)` once and then `route guard: monastery floor=14 (merc=alive)` once

#### Scenario: Road floor trigger
- **WHEN** Monastery road deaths in the act1run soak fail the death rule at the clvl 16 floor
- **THEN** the floor is raised to 18 and the decision goes back to a grill, and if the soak shows no road deaths 14 is tried

### Requirement: Andariel floor
The Normal campaign SHALL attempt Andariel from clvl `boss.andarielMinLevel` 17 with no merc and `boss.andarielMinLevelMerc` 15 with a living merc, with the kit gate on top plus at least 2 antidote potions; the former +2 for a dead merc is replaced by the no-merc floor. No unattended attempt before the throwaway fight of card m3-andariel has won once: `boss.andarielHold` 1 makes the campaign level instead, 0 lifts it. Source: ADR 0011, docs/knobs.md.

#### Scenario: Floor read from the knob
- **WHEN** an agent needs the Andariel level
- **THEN** it reads docs/knobs.md or the knob, not a number copied from a doc

#### Scenario: Dead merc
- **WHEN** a clvl 16 character has no living merc and enough gold to revive
- **THEN** Andariel is not attempted until clvl 17 or the merc is revived and clvl 15 is met

#### Scenario: No antidotes
- **WHEN** a clvl 17 character holds a full kit but 1 antidote potion
- **THEN** the Andariel attempt does not start and the bot restocks

#### Scenario: Unattended hold
- **WHEN** a clvl 20 character with a living merc and a full kit reaches Andariel in the campaign while `boss.andarielHold` is 1
- **THEN** the campaign levels instead and logs `boss: held for andariel (unattended hold ...)`, and with the knob 0 the attempt starts

#### Scenario: Old numbers by knobs
- **WHEN** the road and tower floors are set to 20 with the with-merc road floor 0 and `route.towerNeedsMerc` 1, and the Andariel floor to 15
- **THEN** the former decisions return (a living merc opens the road and tower at any level); the dead-merc +2 is not restored, the no-merc floor 17 equals it
