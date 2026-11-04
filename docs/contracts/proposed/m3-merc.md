# Proposed contract text: m3-merc (merc-policy)

> status: PROPOSED, not in force. Converted at OpenSpec removal stage 3 from the delta spec of card `m3-merc` (git tag `openspec-final`, path `openspec/changes/m3-merc/specs/merc-policy/spec.md`). When the work lands, the branch that implements it folds this text into `docs/contracts/merc-policy.md` and deletes this file. Backlog task: title starts with `m3-merc:`.

The MODIFIED block `Revive and stop rule` of the card was dropped: stale, see `docs/process/spec-ledger-resolutions-20261104.md` row 8.

## Requirements (new)

### Requirement: Merc uptime in Act 1
After the free Rogue is claimed the bot SHALL keep her alive for at least 70 percent of the field time of the rest of Act 1 Normal (revives within the revive rule, potions below the merc potion line) and SHALL stop reviving after `merc.stopDeathsPerHour` deaths in an hour or once Andariel is dead (the Act 2 hire replaces her). Source: plan-v2 section 5 (soft gate 70 percent, U); d2-sme 2026-10-31 (EXP: revive only while the cost is at most half the gold on hand plus the kit); card m3-merc.

#### Scenario: Four virtual hours with the Rogue
- **WHEN** a clvl 9 Sorceress claims the Rogue and plays 4 virtual hours in Act 1
- **THEN** the minute lines show the merc alive in at least 70 percent of the field minutes and every revive line carries price and gold delta

#### Scenario: Revive too expensive
- **WHEN** the revive price is 600 and the character holds 900 gold with a 500 gold kit
- **THEN** the revive is skipped with reason `gold`


## Requirements (new)

### Requirement: Blood Raven bailout
Blood Raven SHALL count as skipped for good (`BloodRavenSkipped`) after 3 failed attempts or at clvl 12 without the kill, and the paid Rogue fallback then applies. Source: ADR 0011 (grill 0003, 2026-10-31).

#### Scenario: Third failure
- **WHEN** the third Blood Raven attempt fails
- **THEN** `BloodRavenSkipped` is set and a paid Rogue hire is allowed under the price rule

#### Scenario: Clvl 12 without the kill
- **WHEN** the character reaches clvl 12 and Blood Raven is alive after 1 failed attempt
- **THEN** she counts as skipped and no further attempt starts

### Requirement: Keeper and throwaway mercs
A merc that is the build's target merc or wears its target gear SHALL be a keeper, always revived and never replaced to save gold; any other merc is a throwaway and SHALL be re-hired when the hire price is below the revive price (same gold-minus-kit rule). Until builds define a target merc the bot MUST always revive. Source: ADR 0011 (grill 0003, 2026-10-31).

#### Scenario: Throwaway cheaper to hire
- **WHEN** a throwaway merc dies, hire costs 200 and revive costs 600 and gold covers both rules
- **THEN** the bot hires a new merc instead of reviving

#### Scenario: No build target yet
- **WHEN** no build names a target merc and the merc dies
- **THEN** the bot revives it under the revive rule

