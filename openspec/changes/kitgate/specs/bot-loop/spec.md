## ADDED Requirements

### Requirement: Recovery without the kit stays near town
While recovery runs and the character has no healing potion in the belt or no town portal scroll (Normal act 1), the bot SHALL farm only Blood Moor (2) and Cold Plains (3), wait at home when neither is open, and MUST keep its walks out of the dungeon areas (8, 9, 13, 17, 18, 19). The knob `recoveryKitGate` (default 1) at 0 SHALL restore the old choice.

#### Scenario: Recovery area without the kit
- **WHEN** a recovering character has an empty belt or no portal scroll at clvl 7 or 14
- **THEN** the area is Blood Moor or Cold Plains, or none (TestKitGateRecoveryArea, TestKitGateNoSafeFieldWaits)

#### Scenario: Kit present keeps the old choice
- **WHEN** a recovering character has a healing potion and a portal scroll, or the knob is 0
- **THEN** the area equals the one chosen before the change (TestKitGateKeepsOldChoices)

#### Scenario: Walks avoid the dungeons
- **WHEN** the gate turns on or off
- **THEN** the navigator gets the six dungeon areas, or none; Route prefers a direct exit over a seen path through an avoided level, and an avoided destination stays routable (TestKitGateNavAvoid, TestRouteAvoidLevels)
