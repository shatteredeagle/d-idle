## ADDED Requirements

### Requirement: Campaign stall escalation
While a goal runs, the bot SHALL escalate when the campaign makes no progress (kill, non-town area entered, purchase, quest step) on the virtual clock: after `bot.stallWarnMin` (30) minutes one WARN `campaign stalled: no progress for X virtual min` and a fresh game; after `bot.stallExitMin` (30) more, exit status 5 with `campaign stalled, exiting for a restart`. Progress resets it; 0 turns it off. Source: card nowall, docs/nowall.md.

#### Scenario: Legitimate wait
- **WHEN** the bot waits 28 virtual minutes (for example area bans of 3 min each) without progress
- **THEN** nothing is logged and no game is started

#### Scenario: Stall through both windows
- **WHEN** no progress for 31 virtual minutes, then 31 more after the fresh game
- **THEN** the first window logs the WARN and starts a fresh game, the second ends Run with ErrCampaignStalled (exit status 5)

#### Scenario: Progress resets
- **WHEN** a kill or purchase happens after the fresh game
- **THEN** the ladder returns to stage 0 and a full window must pass before the next WARN
