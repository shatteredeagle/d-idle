## ADDED Requirements

### Requirement: Rejuvenation is drunk at a higher line against a boss pack
With a hostile in range, pack danger 1 or more (champion, unique or boss), a rejuvenation potion in the belt and knob combat.drinkRejuvBossPct above 0 (default 50, range 0 to 70), PotionPolicy SHALL drink a rejuvenation potion below that life percent; against plain monsters only below combat.drinkRejuvPct (35) or the chicken line. 0 turns the boss line off. Source: card rejuvuse; internal/resource/potion.go (bossRejuv); test TestBossRejuvLine. The champion bonus is in its own requirement.

#### Scenario: Trash at 45 percent
- **WHEN** a clvl 20 character at 45 percent life with a rejuvenation potion fights plain monsters (danger 0)
- **THEN** no rejuvenation potion is drunk

#### Scenario: Boss pack at 45 percent
- **WHEN** the same character fights a pack of danger 2
- **THEN** a rejuvenation potion is drunk

#### Scenario: Knob 0
- **WHEN** combat.drinkRejuvBossPct is 0 and the boss pack is there
- **THEN** no rejuvenation potion is drunk at 45 percent
