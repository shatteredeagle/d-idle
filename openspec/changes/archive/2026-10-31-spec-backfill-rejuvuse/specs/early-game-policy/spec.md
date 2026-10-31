## ADDED Requirements

### Requirement: Full Rejuvenation is preferred at the cap
Below knob loot.rejuvSpare (default 3, range 0 to 6) rejuvenation potions held in belt plus pack any one SHALL be picked up; at the cap only a Full Rejuvenation is still wanted while a regular one sits in the pack, and the town sale drops regular ones first. With 0 none is wanted. The cap itself is in Rejuvenation potion cap. Source: card rejuvuse; internal/loot/loot.go (wantRejuv); tests TestRejuvIsCapped, TestFullPackOfRejuvsIsSoldDown.

#### Scenario: At the cap
- **WHEN** the belt holds 1 Full and the pack 2 regular rejuvenation potions
- **THEN** another regular one is not wanted and a Full is

#### Scenario: Pack full of both
- **WHEN** the pack holds regular and Full rejuvenation potions alternately and a sell visit runs
- **THEN** 3 remain, all Full

### Requirement: A pack rejuvenation is belted without a rejuvenation column
FillBelt SHALL move a rejuvenation potion from the pack to a free belt slot even when the plan has no rejuvenation column (below clvl 31), so the belt hotkey can drink it. Source: card rejuvuse; internal/town/beltfill.go; test TestPackRejuvIsBelted.

#### Scenario: Clvl 14 with one in the pack
- **WHEN** a clvl 14 character carries one rejuvenation potion in the pack and the belt has room
- **THEN** after FillBelt the pack holds none and the belt holds it
