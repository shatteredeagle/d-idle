## ADDED Requirements

### Requirement: Recovery without the kit stays near town
While the recovery phase runs and the character has no healing potion in the belt or no town portal scroll (Normal, act 1), the bot SHALL farm only Blood Moor (2) and Cold Plains (3), MUST NOT fall through to the quest ladder when neither is open (it waits at home), and MUST ask the navigator to avoid the Den of Evil (8), Cave Levels 1 and 2 (9, 13), Burial Grounds (17), the Crypt (18) and the Mausoleum (19) as transit levels. The knob `recoveryKitGate` (default 1) set to 0 SHALL restore the old area choice. A character with both kit parts SHALL get the same area as before.

#### Scenario: Recovery area without the kit
- **WHEN** a recovering character has an empty belt or no portal scroll at clvl 7 or 14
- **THEN** the area is Blood Moor or Cold Plains, or none (TestKitGateRecoveryArea, TestKitGateNoSafeFieldWaits)

#### Scenario: Kit present keeps the old choice
- **WHEN** a recovering character has a healing potion and a portal scroll, or the knob is 0
- **THEN** the area equals the one chosen before the change (TestKitGateKeepsOldChoices)

#### Scenario: Walks avoid the dungeons
- **WHEN** the gate turns on or off
- **THEN** the navigator gets the six dungeon areas, or none; Route prefers a direct exit over a seen path through an avoided level, and an avoided destination stays routable (TestKitGateNavAvoid, TestRouteAvoidLevels)
