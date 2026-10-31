# Town basics specification

## Purpose
What the bot does in town with gold, scrolls, potions, repairs, the stash and the pack so that a poor early character spends on survival first and never burns time or gold on trips that pay nothing. Source: docs/basics-audit.md (the d2-sme's 15 scenarios, 2026-10-31), docs/BACKLOG.md section 0 (user milestone).

## ADDED Requirements

### Requirement: Stash gold cap
When the stash refuses a deposit because it is full the bot SHALL deposit only the difference to the cap (stash step and bank-before-shop alike), log it once, and MUST NOT fail the stash step on every later visit. Source: docs/basics-audit.md E4 and G10; the cap constant (2,500,000) is UNVERIFIED until card m1-liveproof measures it; a refused deposit into a stash holding 100,000 gold or more is taken as the cap.

#### Scenario: Full stash
- **WHEN** the stash gold is at the cap and the pack holds 2000 gold above the keep line
- **THEN** the deposit is skipped with one log line and the next visit does not retry it while the stash is still full

#### Scenario: Partial deposit
- **WHEN** the stash has room for 100 gold and the pack holds gold above the keep line
- **THEN** exactly 100 gold are deposited and the stash step does not fail
