## ADDED Requirements

### Requirement: Every equip slot click is logged
Each slot click of a town equip SHALL log `equip: slot click at (x,y)` with the slot name, the cursor item id, the try number and the slot rectangle, and the equip SHALL click at most 3 times (equipClicks) at different points of the slot box, repeating only while the item is still on the cursor. Source: card equiplive; internal/town/rebuy.go; test TestEquipSurvivesIgnoredSlotClicks (log text read from code, not asserted).

#### Scenario: First click ignored
- **WHEN** the game ignores the first slot click and takes the second
- **THEN** the item is worn after two logged clicks and the cursor is empty

### Requirement: A refused item is skipped until the character changes
An item still on the cursor after all slot clicks MUST be remembered with the character's level, Strength and Dexterity and SHALL NOT be offered or tried again (log `equip: skip <item> reason=refused`) until one of the three changes. Source: card equiplive; internal/town/rebuy.go (isRefused); tests TestRefusedGearUntilLevelUp, TestRefusedLiftedByStat. The code also lifts on a stat point, not only a level-up.

#### Scenario: Same level, then level-up
- **WHEN** an item is refused at clvl 7 and the check repeats at clvl 7 and at clvl 8
- **THEN** it is refused at clvl 7 and offered again at clvl 8

#### Scenario: Strength point
- **WHEN** the item was refused at Strength 10 and the character now has 11
- **THEN** it is offered again

### Requirement: A failed equip returns the item and says why
After a failed equip the item SHALL go back to a free pack cell and never stay held, the error MUST name the cause (`item still on the cursor (the game refused the slot click)` when it never left the cursor), and no right-click retry is made (equipRightClickFallback is false: 1.14d equips by click). Source: card equiplive; internal/town/rebuy.go; test TestEquipSurvivesIgnoredSlotClicks.

#### Scenario: Every click ignored
- **WHEN** the game ignores all slot clicks
- **THEN** nothing is worn, the cursor is empty and the error contains `still on the cursor`
