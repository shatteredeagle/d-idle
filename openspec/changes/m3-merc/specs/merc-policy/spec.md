## ADDED Requirements

### Requirement: Merc uptime in Act 1
After the free Rogue is claimed the bot SHALL keep her alive for at least 70 percent of the field time of the rest of Act 1 Normal (revives within the revive rule, potions below the merc potion line) and SHALL stop reviving after `merc.stopDeathsPerHour` deaths in an hour or once Andariel is dead (the Act 2 hire replaces her). Source: plan-v2 section 5 (soft gate 70 percent, U); d2-sme 2026-10-31 (EXP: revive only while the cost is at most half the gold on hand plus the kit); card m3-merc.

#### Scenario: Four virtual hours with the Rogue
- **WHEN** a clvl 9 Sorceress claims the Rogue and plays 4 virtual hours in Act 1
- **THEN** the minute lines show the merc alive in at least 70 percent of the field minutes and every revive line carries price and gold delta

#### Scenario: Revive too expensive
- **WHEN** the revive price is 600 and the character holds 900 gold with a 500 gold kit
- **THEN** the revive is skipped with reason `gold`
